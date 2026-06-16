// Package codeguard 保护网页中的代码块，使其不被 readability 的正文清洗算法丢弃。
//
// 背景：readability（基于 Mozilla Readability.js）通过给节点打「内容得分」剥离
// 导航/广告/侧边栏等噪声。但许多文档框架（Fern、Mintlify、Docusaurus 等）把代码块
// 渲染成 <pre><table><tr><td>…<span style="color">…</span> 的「行 + 语法高亮」结构，
// 文本被拆散在 table/shiki span 里，readability 的评分机制对此失效，会把整个代码块
// 容器当低内容区域丢弃——结果是提取出的 Markdown 里所有代码块凭空消失。
//
// 解法：在 HTML 进入 readability 之前，把每个 <pre> 内部的复杂结构抽取成纯文本，
// 并「就地」重写为一个干净的标准 <pre><code>纯文本</code></pre>。对照实验证明
// readability 能稳定识别并保留这种标准结构，因此代码块不再丢失。语言信息通过
// data 属性携带，待 readability 提取后再补回 <code> 的 class（readability 会剥
// code 的 class，所以不能直接依赖它）。
package codeguard

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Block 是从页面抽取出的一个代码块：编程语言（如 "bash"、"typescript"，无则为 ""）
// 与原始代码文本。
type Block struct {
	Lang string
	Code string
}

// dataAttr 是挂到重写后 <pre> 上的序号标记，用于 readability 之后把语言对齐补回。
const dataAttr = "data-wx-codeblock"

var (
	// langRe 从 class 里识别 language-xxx / lang-xxx / highlight-xxx / brush-xxx。
	langRe = regexp.MustCompile(`\b(?:language|lang|highlight|brush)-([A-Za-z][\w+-]*)`)
)

// Protect 扫描 HTML 中所有 <pre> 代码块，把其内部的 table+shiki 高亮结构抽取为纯文本，
// 并就地重写成标准 <pre data-wx-codeblock="i"><code>纯文本</code></pre>，同时记录
// 语言。返回处理后的 HTML 与按出现顺序排列的代码块清单。解析失败或不含代码块时
// 原样返回 HTML、空清单。
func Protect(htmlContent string) (string, []Block) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent, nil
	}

	var blocks []Block
	doc.Find("pre").Each(func(_ int, pre *goquery.Selection) {
		code := extractCodeText(pre)
		if strings.TrimSpace(code) == "" {
			return // 空代码块跳过，不占用序号
		}
		idx := len(blocks)
		blocks = append(blocks, Block{Lang: detectLang(pre), Code: code})

		// 用干净的标准 pre 就地替换原 pre：清掉 fern 的噪声 class（code-block-root、
		// fern-code-content 等），仅保留纯文本 code。原父容器里的 copy 按钮、空 h5 等
		// 装饰兄弟节点保留在原处，但它们失去 pre 后文本密度极低，readability 会当噪声丢弃。
		newPre := `<pre ` + dataAttr + `="` + strconv.Itoa(idx) + `"><code>` +
			html.EscapeString(code) + "</code></pre>"
		pre.ReplaceWithHtml(newPre)
	})

	if len(blocks) == 0 {
		return htmlContent, nil
	}
	out, err := doc.Html()
	if err != nil {
		return htmlContent, blocks
	}
	return out, blocks
}

// StampLanguage 在 readability 提取后的正文 HTML 上，依据 data-wx-codeblock 标记把
// 语言补回对应 <code> 的 class。readability 会剥掉 <code> 的 class，所以语言必须在
// 它之后补。这一步在 HTML→Markdown 转换前执行，使转换器输出带语言标注的 fenced 代码块。
func StampLanguage(htmlContent string, blocks []Block) string {
	if len(blocks) == 0 {
		return htmlContent
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}
	doc.Find("pre["+dataAttr+"]").Each(func(_ int, pre *goquery.Selection) {
		idxStr, ok := pre.Attr(dataAttr)
		if !ok {
			return
		}
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(blocks) {
			return
		}
		if lang := blocks[idx].Lang; lang != "" {
			if code := pre.Find("code").First(); code.Length() > 0 {
				code.SetAttr("class", "language-"+lang)
			}
		}
		pre.RemoveAttr(dataAttr)
	})
	out, err := doc.Html()
	if err != nil {
		return htmlContent
	}
	return out
}

// extractCodeText 从一个 <pre> 节点提取干净的代码纯文本。
// 优先按行结构（每行一个 <tr>，内容单元格为 .code-block-line-content）逐行拼接，
// 这样能正确处理 Fern/Mintlify 的 table 行渲染，并排除行号/提示符（$）等 gutter 列；
// 退化情况（普通 <pre><code>）直接取其文本。提取前先剔除 <style>/<script>，
// 避免内联 CSS/JS 源码混入代码内容。
func extractCodeText(pre *goquery.Selection) string {
	pre.Find("style, script").Remove()

	var lines []string
	pre.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		// 优先只取内容单元格，跳过行号 gutter（.code-block-line-gutter 等）
		if content := tr.Find(".code-block-line-content, .code-line, .line-content").First(); content.Length() > 0 {
			lines = append(lines, strings.TrimRight(content.Text(), " \t\r"))
			return
		}
		lines = append(lines, strings.TrimRight(tr.Text(), " \t\r"))
	})
	if len(lines) > 0 {
		return strings.Join(lines, "\n")
	}
	// 兜底：普通 <pre><code>…</code></pre>，直接取 pre 文本
	return strings.TrimSpace(pre.Text())
}

// detectLang 识别代码块语言。按优先级在四处查找 language-xxx 标注：
//  1. <pre> 自身 class（部分框架标注在 pre 上）；
//  2. <pre> 内的 <code> 子节点 class（highlight.js / prism / mdx 等标准做法）；
//  3. <pre> 的祖先容器 class（Fern 等：语言标在最外层容器上）；
//  4. 祖先的 data-language 属性。
//
// 找不到返回空串，对应输出无语言标注的代码块。
func detectLang(pre *goquery.Selection) string {
	if lang := matchLangClass(pre); lang != "" {
		return lang
	}
	if code := pre.Find("code").First(); code.Length() > 0 {
		if lang := matchLangClass(code); lang != "" {
			return lang
		}
	}
	for node, depth := pre.Parent(), 0; node.Length() > 0 && depth < 8; node, depth = node.Parent(), depth+1 {
		if lang := matchLangClass(node); lang != "" {
			return lang
		}
		if node.Is("body") {
			break
		}
	}
	if c := pre.Closest("[data-language]"); c.Length() > 0 {
		if l, ok := c.Attr("data-language"); ok {
			return normalizeLang(l)
		}
	}
	return ""
}

// matchLangClass 从节点的 class 属性里提取 language-xxx 并归一化，无则返回空串。
func matchLangClass(s *goquery.Selection) string {
	if cls, ok := s.Attr("class"); ok {
		if m := langRe.FindStringSubmatch(cls); m != nil {
			return normalizeLang(m[1])
		}
	}
	return ""
}

// normalizeLang 把常见写法归一为 markdown 代码块常用的语言标签。
func normalizeLang(l string) string {
	l = strings.ToLower(strings.TrimSpace(l))
	switch l {
	case "":
		return ""
	case "sh", "shell", "console", "terminal", "zsh", "bashrc":
		return "bash"
	case "ts":
		return "typescript"
	case "js":
		return "javascript"
	case "py":
		return "python"
	case "golang":
		return "go"
	}
	return l
}
