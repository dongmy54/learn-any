package converter

import (
	"strings"
	"testing"
)

// TestTableConversion 是表格提取的回归测试：
// readability 会保留正文中的 <table>，但 html-to-markdown 默认只启用 commonmark 规则，
// 必须显式注册 Table() 插件，否则表格内容会被整段丢弃。
func TestTableConversion(t *testing.T) {
	html := `<table>
<thead><tr><th>模型</th><th>上下文</th></tr></thead>
<tbody>
<tr><td>GPT</td><td>128K</td></tr>
<tr><td>Claude</td><td>200K</td></tr>
</tbody>
</table>`

	out, err := ToMarkdown(html)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	// 表格至少要出现表头分隔行（|---|）与一个数据行，否则说明 table 未被转换。
	if !strings.Contains(out, "|---|") && !strings.Contains(out, "| ---") {
		t.Fatalf("表格未被转换为 Markdown 表格:\n%s", out)
	}
	for _, want := range []string{"GPT", "Claude", "128K", "200K"} {
		if !strings.Contains(out, want) {
			t.Fatalf("表格内容缺失 %q:\n%s", want, out)
		}
	}
}
