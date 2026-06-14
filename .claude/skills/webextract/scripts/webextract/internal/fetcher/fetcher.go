package fetcher

import "context"

// Fetcher 抽象「拿到页面 HTML」这一步——这是整个工具唯一的可替换策略点。
// 静态与动态两种实现通过同一接口被流水线无差别调用，
// 让 --render 开关的实现只需「换一个 Fetcher 实例」，零侵入。
type Fetcher interface {
	// Fetch 抓取给定 URL，返回页面 HTML。
	// 静态实现返回服务器原始 HTML，动态实现返回 JS 渲染后的最终 DOM。
	Fetch(ctx context.Context, url string) (rawHTML string, err error)
}
