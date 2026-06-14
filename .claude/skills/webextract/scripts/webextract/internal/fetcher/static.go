package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StaticFetcher 用标准 HTTP 客户端直接请求，轻量快速。
// 适用于博客、文档、资讯等服务器端渲染的静态站点。
type StaticFetcher struct {
	client *http.Client
}

func NewStaticFetcher() *StaticFetcher {
	return &StaticFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second, // 全局超时，避免被慢站点拖死整个流水线
		},
	}
}

func (f *StaticFetcher) Fetch(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	// 伪装成主流浏览器 UA，规避「裸 HTTP 请求被反爬规则拦截」的常见问题
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "+
			"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// 不带「抓取失败」前缀，由 pipeline 统一包装，避免前缀重复
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	return string(body), err
}
