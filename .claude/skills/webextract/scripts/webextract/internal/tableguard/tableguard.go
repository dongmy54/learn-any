// Package tableguard 抢救正文中的表格，避免被 readability 的正文清洗算法丢弃。
//
// 背景：readability（Mozilla Readability.js 移植）按「内容得分」选出最高分容器作为正文。
// 许多文档框架（Fern、Mintlify 等）把每个表格单独包裹在一个低文本密度的容器里，
// 导致该容器评分不足以进入正文——连同它里面的表格一起被整体排除。
// 实验证明：即便把表格替换为 readability 通常稳定保留的 <pre>，仍会被丢弃，
// 因为问题在「容器选择」而非「节点清理」，任何 readability 内部的就地保护都无效。
//
// 解法：在 HTML 进入 readability 之前先把所有正文表格收集起来（转成 GFM markdown），
// readability 提取后再检测哪些表格幸存、哪些丢失，把丢失的按原文档顺序回填到正文末尾。
// 位置上无法精确还原（表格与其引导文字被框架隔离在不同容器），但表格自带表头，
// 独立成段仍可理解；本机制的首要目标是「内容不丢失」。
package tableguard

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/PuerkitoBio/goquery"
)

// SavedTable 是从原文档抢救出的一个正文表格：其 GFM markdown 与用于判重的内容签名。
type SavedTable struct {
	Markdown  string
	Signature string
}

// 复用一个独立的、仅注册 Table 插件的 converter，用于把单个 <table> 的 HTML 转成 GFM。
var tableConv = func() *md.Converter {
	c := md.NewConverter("", true, nil)
	c.Use(plugin.Table())
	return c
}()

// Collect 从 HTML 中收集所有「正文表格」，为每个生成 GFM markdown 和内容签名。
// 排除：<pre> 内的表格（那是 codeguard 管的代码块语法高亮结构）、嵌套在其它表格内的子表格。
// 这些表格极易在 readability 提取时整体丢失，需提取后回填，详见 Restore。
func Collect(htmlContent string) []SavedTable {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}
	var out []SavedTable
	doc.Find("table").Each(func(_ int, t *goquery.Selection) {
		if t.Closest("pre").Length() > 0 {
			return // 代码块高亮，归 codeguard 管
		}
		if t.ParentsFiltered("table").Length() > 0 {
			return // 嵌套子表格，随父表一起处理即可
		}
		outer, err := goquery.OuterHtml(t)
		if err != nil {
			return
		}
		markdown, err := tableConv.ConvertString(outer)
		if err != nil || strings.TrimSpace(markdown) == "" {
			return
		}
		out = append(out, SavedTable{
			Markdown:  strings.TrimSpace(markdown),
			Signature: signature(t),
		})
	})
	return out
}

// PresentSignatures 从 readability 提取后的正文 HTML 中，提取所有幸存表格的内容签名集合。
// 用于在 Restore 时判断哪些表格已被保留、无需重复回填。
func PresentSignatures(articleHTML string) map[string]struct{} {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(articleHTML))
	if err != nil {
		return nil
	}
	present := make(map[string]struct{})
	doc.Find("table").Each(func(_ int, t *goquery.Selection) {
		present[signature(t)] = struct{}{}
	})
	return present
}

// Restore 把「原文档存在但 readability 正文里丢失」的表格，按原文档顺序追加到 Markdown 末尾。
// 用内容签名去重：已在正文里的表格不会重复回填。
func Restore(markdown string, saved []SavedTable, present map[string]struct{}) string {
	if len(saved) == 0 {
		return markdown
	}
	var missing []string
	for _, s := range saved {
		if _, ok := present[s.Signature]; ok {
			continue // readability 已保留这张表
		}
		missing = append(missing, s.Markdown)
	}
	if len(missing) == 0 {
		return markdown
	}
	var b strings.Builder
	b.WriteString(strings.TrimRight(markdown, "\n\r\t "))
	for _, m := range missing {
		b.WriteString("\n\n")
		b.WriteString(m)
	}
	b.WriteString("\n")
	return b.String()
}

// signature 取表格全部单元格文本（压缩空白）的前 80 个字符，作为「是否同一张表」的判据。
// 用全文而非仅表头：不同表格可能共享相同表头（如多张 "Value|Description" 表），
// 仅靠表头会把不同的表误判为同一张，导致该回填的不回填。
func signature(t *goquery.Selection) string {
	txt := strings.Join(strings.Fields(t.Text()), " ")
	if r := []rune(txt); len(r) > 80 {
		txt = string(r[:80])
	}
	return txt
}
