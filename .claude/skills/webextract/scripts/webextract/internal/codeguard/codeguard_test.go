package codeguard

import "testing"

// TestProtect_StandardCodeBlock 验证 highlight.js/prism 风格的标准代码块：
// 语言标注在 <code> 子节点上，代码是普通文本（非 table 行结构）。
func TestProtect_StandardCodeBlock(t *testing.T) {
	html := `<html><body><article>
<h1>Doc</h1>
<p>Run this:</p>
<pre><code class="language-python">print("hi")
for i in range(3):
    print(i)</code></pre>
</article></body></html>`

	out, blocks := Protect(html)
	if len(blocks) != 1 {
		t.Fatalf("期望 1 个代码块，得到 %d", len(blocks))
	}
	if blocks[0].Lang != "python" {
		t.Errorf("语言：期望 python，得到 %q", blocks[0].Lang)
	}
	want := "print(\"hi\")\nfor i in range(3):\n    print(i)"
	if blocks[0].Code != want {
		t.Errorf("代码文本不符\n得到: %q\n期望: %q", blocks[0].Code, want)
	}
	// 重写后应是标准 <pre data-wx-codeblock="0"><code>…</code></pre>
	if got := count(out, "data-wx-codeblock"); got != 1 {
		t.Errorf("重写后应含 1 个标记，得到 %d", got)
	}
}

// TestProtect_FernTableStructure 验证 Fern/Mintlify 的 table+shiki 行结构：
// 语言在最外层容器 class 上，代码按 <tr>/<td class="code-block-line-content"> 逐行渲染。
func TestProtect_FernTableStructure(t *testing.T) {
	html := `<html><body><article>
<p>First, install:</p>
<div class="fern-code-block not-prose border language-bash">
  <div class="copy-button"><button>copy</button></div>
  <pre class="code-block-root not-prose"><div>
    <table class="code-block-line-group"><tbody>
      <tr class="code-block-line">
        <td class="code-block-line-gutter"><span>$</span></td>
        <td class="code-block-line-content"><span class="line"><span style="color:#a">npm</span><span> install </span><span style="color:#b">@openrouter/sdk</span></span></td>
      </tr>
    </tbody></table>
  </div></pre>
</div>
</article></body></html>`

	out, blocks := Protect(html)
	if len(blocks) != 1 {
		t.Fatalf("期望 1 个代码块，得到 %d", len(blocks))
	}
	if blocks[0].Lang != "bash" {
		t.Errorf("语言：期望 bash，得到 %q", blocks[0].Lang)
	}
	// gutter 的 $ 提示符应被排除，只保留内容单元格文本
	if blocks[0].Code != "npm install @openrouter/sdk" {
		t.Errorf("代码文本：得到 %q", blocks[0].Code)
	}
	if got := count(out, "data-wx-codeblock"); got != 1 {
		t.Errorf("重写后应含 1 个标记，得到 %d", got)
	}
}

// TestProtect_NoLanguage 验证无语言标注的代码块：语言为空，代码文本仍正确提取。
func TestProtect_NoLanguage(t *testing.T) {
	html := `<article><pre><code>echo hello</code></pre></article>`
	_, blocks := Protect(html)
	if len(blocks) != 1 {
		t.Fatalf("期望 1 个代码块，得到 %d", len(blocks))
	}
	if blocks[0].Lang != "" {
		t.Errorf("无语言标注应返回空，得到 %q", blocks[0].Lang)
	}
	if blocks[0].Code != "echo hello" {
		t.Errorf("代码文本：得到 %q", blocks[0].Code)
	}
}

// TestProtect_EmptyAndNone 验证空代码块被跳过、无代码块页面原样返回。
func TestProtect_EmptyAndNone(t *testing.T) {
	// 空代码块（仅空白）不占位
	html := `<article><pre><code>   </code></pre><p>text</p></article>`
	out, blocks := Protect(html)
	if len(blocks) != 0 {
		t.Errorf("空代码块应跳过，得到 %d 个", len(blocks))
	}
	if out != html {
		t.Errorf("无有效代码块时应原样返回 HTML")
	}
	// 完全无 pre
	plain := `<article><p>just text</p></article>`
	out2, blocks2 := Protect(plain)
	if len(blocks2) != 0 || out2 != plain {
		t.Errorf("无 pre 页面应原样返回")
	}
}

// TestStampLanguage 验证 readability 之后按序号标记把语言补回 <code> 的 class。
func TestStampLanguage(t *testing.T) {
	// 模拟 readability 输出：pre 保留了 data-wx-codeblock 标记，但 code 无 class
	extracted := `<article>
<pre data-wx-codeblock="0"><code>npm install x</code></pre>
<pre data-wx-codeblock="1"><code>print(1)</code></pre>
</article>`
	blocks := []Block{{Lang: "bash", Code: "npm install x"}, {Lang: "python", Code: "print(1)"}}

	out := StampLanguage(extracted, blocks)
	if got := count(out, "language-bash"); got != 1 {
		t.Errorf("bash 标注数：得到 %d", got)
	}
	if got := count(out, "language-python"); got != 1 {
		t.Errorf("python 标注数：得到 %d", got)
	}
	if count(out, "data-wx-codeblock") != 0 {
		t.Errorf("标记属性应被清理")
	}
}

// TestNormalizeLang 验证常见语言写法归一。
func TestNormalizeLang(t *testing.T) {
	cases := map[string]string{
		"typescript": "typescript",
		"TS":         "typescript",
		"sh":         "bash",
		"shell":      "bash",
		"py":         "python",
		"golang":     "go",
		"rust":       "rust",
		"":           "",
	}
	for in, want := range cases {
		if got := normalizeLang(in); got != want {
			t.Errorf("normalizeLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func count(s, sub string) int {
	n := 0
	for {
		i := indexOf(s, sub)
		if i < 0 {
			return n
		}
		n++
		s = s[i+len(sub):]
	}
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
