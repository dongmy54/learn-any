package converter

import (
	md "github.com/JohannesKaufmann/html-to-markdown"
)

// 复用全局 converter 实例，避免每次调用重复初始化规则引擎。
// 默认规则已能处理 标题/段落/列表/链接/代码块/图片 等常见标签，
// 覆盖绝大多数正文场景。
var conv = md.NewConverter("", true, nil)

// ToMarkdown 将 readability 输出的正文 HTML 转为 Markdown。
func ToMarkdown(html string) (string, error) {
	return conv.ConvertString(html)
}
