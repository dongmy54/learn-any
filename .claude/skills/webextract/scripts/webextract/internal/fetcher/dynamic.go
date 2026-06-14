package fetcher

import (
	"context"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// DynamicFetcher 基于 chromedp 驱动无头 Chrome 渲染页面。
// 适用于 React/Vue 等 SPA 单页应用，能拿到 JS 执行后的最终 DOM。
type DynamicFetcher struct {
	allocCtx    context.Context // 浏览器分配上下文，复用浏览器实例省去重复启动开销
	allocCancel context.CancelFunc
	wait        time.Duration // 渲染后的额外等待时间，给异步数据加载留时间
}

func NewDynamicFetcher(wait time.Duration) *DynamicFetcher {
	// DefaultExecAllocatorOptions 默认已是 headless 模式 + 禁用 GPU，开箱即用
	opts := chromedp.DefaultExecAllocatorOptions[:]
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	return &DynamicFetcher{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		wait:        wait,
	}
}

// Close 释放浏览器分配上下文，避免 headless Chrome 进程残留
func (f *DynamicFetcher) Close() error {
	f.allocCancel()
	return nil
}

func (f *DynamicFetcher) Fetch(ctx context.Context, targetURL string) (string, error) {
	// 每次抓取创建独立 tab 上下文，defer cancel 后自动销毁，
	// 避免不同 URL 间的 cookie / localStorage / DOM 互相污染
	taskCtx, cancel := chromedp.NewContext(f.allocCtx)
	defer cancel()

	var html string
	err := chromedp.Run(taskCtx,
		network.Enable(),
		chromedp.Navigate(targetURL),
		// 等待 body 就绪：确保基础 DOM 已注入，是后续取值的前提
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		// 额外等待：SPA 首屏 HTML 通常只有空壳，真正的内容由组件挂载后的
		// 异步请求填充。固定 sleep 是 MVP 的简单做法；
		// 生产环境建议改为监听 network 事件实现 networkidle
		chromedp.Sleep(f.wait),
		// 取渲染后的完整 DOM——这是动态模式与静态模式的「归一化输出点」：
		// 从这一步往后，提取与转换逻辑两者完全一致
		chromedp.OuterHTML(`html`, &html, chromedp.ByQuery),
	)
	return html, err
}
