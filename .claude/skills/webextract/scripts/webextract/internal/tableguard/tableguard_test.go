package tableguard

import (
	"strings"
	"testing"
)

// TestCollectExcludesPreTables：正文表格应被收集，而 <pre> 内的高亮表格（归 codeguard）应被排除。
func TestCollectExcludesPreTables(t *testing.T) {
	html := `
<pre><code><table><tr><td>not me</td></tr></table></code></pre>
<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>keep me</td></tr></tbody></table>`
	got := Collect(html)
	if len(got) != 1 {
		t.Fatalf("应收集 1 个正文表格，实际 %d", len(got))
	}
	if !strings.Contains(got[0].Markdown, "keep me") {
		t.Fatalf("收集到错误的表格: %s", got[0].Markdown)
	}
}

// TestRestoreMissingTables：readability 丢失的表格应被追加到末尾。
func TestRestoreMissingTables(t *testing.T) {
	saved := Collect(`<table><thead><tr><th>K</th></tr></thead><tbody><tr><td>v1</td></tr></tbody></table>`)
	md := "正文段落"
	// present 为空 → 表格被 readability 丢弃
	out := Restore(md, saved, map[string]struct{}{})
	if !strings.Contains(out, "正文段落") {
		t.Fatalf("原正文丢失:\n%s", out)
	}
	if !strings.Contains(out, "| K |") && !strings.Contains(out, "v1") {
		t.Fatalf("丢失的表格未被回填:\n%s", out)
	}
}

// TestRestoreSkipsPresent：readability 已保留的表格不应被重复回填。
func TestRestoreSkipsPresent(t *testing.T) {
	saved := Collect(`<table><thead><tr><th>K</th></tr></thead><tbody><tr><td>v1</td></tr></tbody></table>`)
	// 正文里已经存在同签名表格 → present 命中
	present := PresentSignatures(`<table><thead><tr><th>K</th></tr></thead><tbody><tr><td>v1</td></tr></tbody></table>`)
	out := Restore("正文", saved, present)
	// 只应出现一次表头 K
	if strings.Count(out, "| K |") > 1 || strings.Count(out, "K") > 2 {
		t.Fatalf("已保留的表格被重复回填:\n%s", out)
	}
}

// TestSignatureDistinguishesSameHeader：表头相同但内容不同的表，签名必须不同。
// 否则会被误判为「已保留」而漏掉该回填的表。
func TestSignatureDistinguishesSameHeader(t *testing.T) {
	t1 := Collect(`<table><thead><tr><th>Value</th><th>Description</th></tr></thead><tbody><tr><td>text</td><td>文字输出</td></tr></tbody></table>`)
	t2 := Collect(`<table><thead><tr><th>Value</th><th>Description</th></tr></thead><tbody><tr><td>image</td><td>图像输出</td></tr></tbody></table>`)
	if len(t1) != 1 || len(t2) != 1 {
		t.Fatalf("收集数量异常: %d %d", len(t1), len(t2))
	}
	if t1[0].Signature == t2[0].Signature {
		t.Fatalf("两张内容不同的表签名相同，会导致去重误判: %q", t1[0].Signature)
	}
}
