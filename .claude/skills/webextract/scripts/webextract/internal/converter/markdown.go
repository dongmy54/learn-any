package converter

import (
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
)

// 复用全局 converter 实例，避免每次调用重复初始化规则引擎。
// 默认 commonmark 规则覆盖 标题/段落/列表/链接/代码块/图片 等标签；
// 但表格（<table>）不在 commonmark 范围内，必须显式注册 Table() 插件，
// 否则正文里的表格会被剥掉结构、压成一串粘连文本（见 markdown_test.go）。
var conv = func() *md.Converter {
	c := md.NewConverter("", true, nil)
	c.Use(plugin.Table())
	return c
}()

// ToMarkdown 将 readability 输出的正文 HTML 转为 Markdown。
func ToMarkdown(html string) (string, error) {
	return conv.ConvertString(html)
}
