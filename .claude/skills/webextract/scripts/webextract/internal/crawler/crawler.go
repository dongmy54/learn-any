// Package crawler 在 webextract 既有能力之上增加「批量爬取」调度层：
//
//   - 复用 fetcher.Fetcher 拿整页 HTML
//   - 复用 pipeline.BuildArticle 提取正文并转 Markdown（与单页完全同一套完整性管线）
//   - 新增：goquery 抽链接、BFS 调度、去重、并发 worker pool、批量落盘
//
// 设计要点：主链接（种子）由主 goroutine 同步优先抓取，保证其内容优先且必达；
// 随后 worker pool 并发抓取 depth≥1 的相关链接。
package crawler

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"sync"
	"sync/atomic"

	"webextract/internal/fetcher"
	"webextract/internal/pipeline"
)

// Task 是一个待抓取任务。
type Task struct {
	URL   string
	Depth int // 相对种子的深度，种子本身为 0
}

// Options 控制爬取行为。
type Options struct {
	Max         int  // 最大抓取页数（含种子）
	MaxDepth    int  // 最大深度（0 = 仅种子页）
	Concurrency int  // 并发 worker 数
	SameDomain  bool // 仅抓同域链接
	OutputDir   string
}

// Report 汇总爬取结果。
type Report struct {
	Total    int               // 成功写盘页数（含种子）
	Failed   int               // 失败页数（不含种子，种子失败直接返回 error）
	URL2File map[string]string // URL → 落盘文件路径
}

// state 收敛并发共享的可变状态，避免参数列表膨胀。
type state struct {
	written, failed, claimed, inflight int64 // 全部用 atomic 操作
	visited                            map[string]bool
	url2file                           map[string]string
	mu                                 sync.Mutex // 保护 visited / url2file 两个 map
}

// Crawl 从 seed 批量抓取。
//
//	Phase 1：主 goroutine 同步抓种子（主链接），保证其内容优先且必达
//	Phase 2：worker pool 并发抓 depth ≥ 1 的链接
func Crawl(ctx context.Context, seed string, f fetcher.Fetcher, opt Options) (*Report, error) {
	if opt.Concurrency < 1 {
		opt.Concurrency = 5
	}
	seedNorm, err := Normalize(seed)
	if err != nil {
		return nil, fmt.Errorf("种子 URL 非法: %w", err)
	}
	seedURL, err := url.Parse(seed)
	if err != nil {
		return nil, fmt.Errorf("种子 URL 非法: %w", err)
	}
	allowHost := ""
	if opt.SameDomain {
		allowHost = seedURL.Hostname()
	}

	st := &state{
		visited:  map[string]bool{seedNorm: true},
		url2file: map[string]string{},
	}

	// ===== Phase 1：主链接优先（同步）=====
	html, err := f.Fetch(ctx, seed)
	if err != nil {
		return st.report(), fmt.Errorf("主链接抓取失败: %w", err)
	}
	// 与单页共用 pipeline.BuildArticle：代码块保护 + 完整性兜底 + 表格回填，确保种子页完整。
	art, err := pipeline.BuildArticle(html, seedURL)
	if err != nil {
		return st.report(), fmt.Errorf("主链接正文提取失败: %w", err)
	}
	st.url2file[seed] = WriteArticle(opt.OutputDir, art, true) // true → index.md
	atomic.StoreInt64(&st.written, 1)
	atomic.StoreInt64(&st.claimed, 1) // 种子已占一个抓取名额（计入 --max）

	// --max=1 或 --depth=0：只抓种子，主链接已满足
	if opt.Max <= 1 || opt.MaxDepth < 1 {
		return st.report(), nil
	}

	// 从种子页抽 depth=1 链接，作为并发起点
	pending := st.takePending(html, seedURL, allowHost)
	if len(pending) == 0 {
		return st.report(), nil // 种子页无有效链接
	}

	// ===== Phase 2：worker pool 并发抓 depth ≥ 1 =====
	tasks := make(chan Task, opt.Concurrency*2)
	var wg sync.WaitGroup

	// 关闭监控：inflight 归零 → 关 channel → worker 的 range 自然退出。
	// 正确性依赖不变式：worker 先把子任务的 inflight Add 完（在 takePending 内），
	// 再 Done 自身；因此监控看到 inflight==0 时，绝无 worker 还握着未投递的子任务。
	// 此时 inflight 已被 takePending 预置为 len(pending) > 0，监控不会误关。
	go func() {
		for atomic.LoadInt64(&st.inflight) > 0 {
			runtime.Gosched() // 规模小，轮询开销可忽略
		}
		close(tasks)
	}()

	for i := 0; i < opt.Concurrency; i++ {
		wg.Add(1)
		go st.worker(ctx, f, opt, allowHost, tasks, &wg)
	}

	for _, link := range pending {
		tasks <- Task{URL: link, Depth: 1}
	}
	wg.Wait()
	return st.report(), nil
}

// worker 消费任务通道，处理或被 --max 截断后跳过。
// tasks 用双向 channel：worker 既要 range 消费，又要借 processOne 投递子任务。
func (s *state) worker(ctx context.Context, f fetcher.Fetcher, opt Options,
	allowHost string, tasks chan Task, wg *sync.WaitGroup) {
	defer wg.Done()
	for t := range tasks {
		// --max 截断：用 CAS 抢名额，保证并发下全局认领数严格 ≤ Max。
		// 不能用「读 written < max 再抓」——并发 worker 会同时读到未达标而全部放行，导致超抓。
		if !s.claimSlot(opt.Max) {
			atomic.AddInt64(&s.inflight, -1)
			continue // 名额已满，放弃本任务
		}
		s.processOne(ctx, f, opt, allowHost, t, tasks)
		atomic.AddInt64(&s.inflight, -1) // Done 自身（子任务的 Add 已在 takePending 完成）
	}
}

// claimSlot 用 CAS 原子地抢占一个抓取名额，保证全局认领数 claimed ≤ max。
// 抢到的名额即使后续 processOne 失败也不归还（失败页消耗一个名额），
// 以保持礼貌爬取语义：--max 是「最多发起多少次抓取尝试」的上限。
func (s *state) claimSlot(max int) bool {
	for {
		cur := atomic.LoadInt64(&s.claimed)
		if cur >= int64(max) {
			return false
		}
		if atomic.CompareAndSwapInt64(&s.claimed, cur, cur+1) {
			return true
		}
		// CAS 失败说明有并发竞争，重读重试
	}
}

// processOne 抓单页（depth ≥ 1）：Fetch → 提正文 → 落盘 →（未到深度）抽链接入队。
func (s *state) processOne(ctx context.Context, f fetcher.Fetcher, opt Options,
	allowHost string, t Task, tasks chan<- Task) {
	html, err := f.Fetch(ctx, t.URL)
	if err != nil {
		atomic.AddInt64(&s.failed, 1)
		return
	}
	pageURL, _ := url.Parse(t.URL)
	// 与单页/种子页共用同一套完整性管线，保证批量抓取的每页内容同样完整。
	art, err := pipeline.BuildArticle(html, pageURL)
	if err != nil {
		atomic.AddInt64(&s.failed, 1)
		return
	}
	path := WriteArticle(opt.OutputDir, art, false)
	s.mu.Lock()
	s.url2file[t.URL] = path
	s.mu.Unlock()
	atomic.AddInt64(&s.written, 1)

	// 深度截断：已达最大深度则不再抽取下层链接
	if t.Depth >= opt.MaxDepth {
		return
	}
	// 抽链接入队：takePending 内会先批量 Add inflight，再由调用方投递
	pending := s.takePending(html, pageURL, allowHost)
	for _, link := range pending {
		tasks <- Task{URL: link, Depth: t.Depth + 1}
	}
}

// takePending 从页面 HTML 抽链接，按 visited 去重后取出，并预增 inflight。
// 调用方必须在返回后立即把这些链接投递到 tasks，以维持「先 Add 后 Done」不变式。
func (s *state) takePending(html string, pageURL *url.URL, allowHost string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pending []string
	for _, link := range ExtractLinks(html, pageURL, allowHost) {
		if !s.visited[link] {
			s.visited[link] = true
			pending = append(pending, link)
		}
	}
	atomic.AddInt64(&s.inflight, int64(len(pending)))
	return pending
}

func (s *state) report() *Report {
	return &Report{
		Total:    int(atomic.LoadInt64(&s.written)),
		Failed:   int(atomic.LoadInt64(&s.failed)),
		URL2File: s.url2file,
	}
}
