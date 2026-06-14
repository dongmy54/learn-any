package model

// Article 是流水线的统一产物。
// 无论静态还是动态抓取，最终都归一化为这个结构，供下游提取与转换无差别处理。
type Article struct {
	Title       string // 页面标题（<title> 或 readability 识别的标题）
	Description string // 摘要（readability 生成的摘要）
	URL         string // 规范化后的最终 URL（跟随重定向后的地址）
	HTML        string // readability 提取后的正文 HTML（已剥离导航/广告/侧边栏）
	Markdown    string // 正文 HTML 转换后的 Markdown，输出阶段填充
}
