package extractor

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"

	"webextract/internal/model"
)

// 背景：readability 为「文章」设计，靠给容器打内容分选出单一最高分容器作正文。
// 对 Fern/Mintlify 等文档框架、以及 Wikipedia 比较页这类「表格/结构化内容为主」
// 的页面，它会整片丢弃低文本密度但信息密集的区域（表格、定义卡片等），导致正文
// 大面积缺失（实测 Wikipedia 比较页只剩 ~7% 文字、12 张表全丢）。tableguard 只能
// 把丢失的表格堆到文末，顺序错乱且非表格正文仍会丢。
//
// 解法：当页面存在语义主容器（<main> / <article> / [role=main]）时，把该容器整棵
// 子树（剥离导航/侧栏/页脚等噪声后）作为正文的候选；若它明显比 readability 的产物
// 更完整（表格更多，或文字量显著更大），就改用它——这样内容不丢失、顺序不错乱。
// 否则沿用 readability（它在普通文章页上更干净）。

// noiseSelectors 是从语义主容器里剔除的非正文区域。
// 只删「确定是页面骨架/导航/装饰」的元素，避免误删正文；正文优先于干净。
// 注意：不删 <header>——容器内的 <header> 往往是文章自身的标题区（正文），
// 页面级横幅 header 通常在主容器之外，无需在此处理。
var noiseSelectors = strings.Join([]string{
	// 结构性骨架
	"nav", "footer", "aside",
	// 非渲染 / 脚本样式
	"script", "style", "noscript", "template", "link", "meta",
	// 交互控件与无法转 markdown 的嵌入
	"svg", "button", "form", "iframe", "dialog",
	// ARIA 角色标注的页面骨架
	"[role=navigation]", "[role=banner]", "[role=complementary]",
	"[role=search]", "[role=contentinfo]", "[role=dialog]",
	// 不可见元素
	"[aria-hidden=true]", "[hidden]",
	// MediaWiki（Wikipedia）特有的编辑/引用/导航装饰
	".mw-editsection", ".mw-jump-link", ".mw-indicators",
	".reference", ".reflist", ".mw-references-wrap",
	".navbox", ".vertical-navbox", ".sistersitebox",
	".catlinks", ".printfooter", ".hatnote", ".shortdescription", ".noprint",
	// MediaWiki/Vector 皮肤的端口栏（语言下拉、工具菜单等），保留 <header> 里的 H1 标题。
	// 注意：不剥离 .ambox/.metadata 等维护提示框——它们对人类可见，属页面展示内容，
	// 且与 readability 路径保持一致（避免两条路径行为分叉）。
	".mw-portlet", ".interlanguage-link", "#p-lang-btn", ".vector-page-toolbar",
	// 文档框架常见的侧栏/面包屑/分页（子串匹配，大小写不敏感）
	"[class*=sidebar i]", "[class*=breadcrumb i]", "[class*=pagination i]",
	// 目录（精确匹配，避免误伤 "protocol" 等含 toc 子串的类名）
	".toc", "#toc",
}, ", ")

// containerSelectors 是语义主容器的候选选择器，按优先级排列。
// 实际选择时会在所有匹配中挑「纯文本最长」的那个，兼顾多 <article> 等情况。
var containerSelectors = []string{"[role=main]", "main", "article"}

// CompleteArticle 在 readability 结果可能不完整时，尝试用语义主容器的完整子树替代。
//
// rawHTML 应为已被 codeguard.Protect 处理过的 HTML（代码块已重写为标准 <pre><code>，
// 带 data-wx-codeblock 标记），这样容器子树里的代码块同样能在后续被 StampLanguage 补回语言。
//
// 返回 (替代后的 Article, true) 表示采用容器；返回 (nil, false) 表示沿用 readability。
// 判定为「更完整」的条件（任一成立）：
//  1. 容器里的正文表格数 > readability 产物里的正文表格数（readability 丢了表）；
//  2. readability 的正文体量 < 容器体量的 60%（readability 大面积截断了正文）。
//
// 体量以「非空白字符数」度量，对中英文混排都成立（不能用按空格分词，中文无空格）。
func CompleteArticle(rawHTML string, readabilityArt *model.Article, pageURL *url.URL) (*model.Article, bool) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil, false
	}
	container := bestContainer(doc)
	if container == nil {
		return nil, false
	}

	// 在原容器上统计「干净后的」体量前，先克隆再去噪，避免污染原文档。
	clone := container.Clone()
	clone.Find(noiseSelectors).Remove()

	containerSize := textSize(clone.Text())
	containerTables := contentTableCount(clone)

	// readability 产物的体量
	var readSize, readTables int
	if rd, err := goquery.NewDocumentFromReader(strings.NewReader(readabilityArt.HTML)); err == nil {
		readSize = textSize(rd.Text())
		readTables = contentTableCount(rd.Selection)
	}

	// 容器空壳（去噪后既没文字也没表格）则放弃，避免反而劣化。
	if containerSize < 40 && containerTables == 0 {
		return nil, false
	}

	moreTables := containerTables > readTables
	muchMoreText := readSize*100 < containerSize*60
	if !moreTables && !muchMoreText {
		return nil, false
	}

	// 把容器内的相对链接/图片地址补成绝对地址（readability 会做，这里手动对齐）。
	absolutizeLinks(clone, pageURL)

	html, err := goquery.OuterHtml(clone)
	if err != nil {
		return nil, false
	}

	return &model.Article{
		Title:       readabilityArt.Title, // 标题/摘要仍取 readability 的元数据识别结果
		Description: readabilityArt.Description,
		URL:         pageURL.String(),
		HTML:        html,
	}, true
}

// bestContainer 在候选语义容器里选出纯文本最长的那个；都不存在则返回 nil。
func bestContainer(doc *goquery.Document) *goquery.Selection {
	var best *goquery.Selection
	bestLen := -1
	for _, sel := range containerSelectors {
		doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
			if l := len(s.Text()); l > bestLen {
				bestLen = l
				best = s
			}
		})
	}
	return best
}

// contentTableCount 统计「正文表格」数量，口径与 tableguard.Collect 一致：
// 排除 <pre> 内的代码高亮表格与嵌套子表格。
func contentTableCount(s *goquery.Selection) int {
	n := 0
	s.Find("table").Each(func(_ int, t *goquery.Selection) {
		if t.Closest("pre").Length() > 0 {
			return
		}
		if t.ParentsFiltered("table").Length() > 0 {
			return
		}
		n++
	})
	return n
}

// textSize 计非空白字符数，作为语言无关的「正文体量」度量。
// 不能用按空格分词：中文/日文等无词间空格，会把整段误判为极少量内容。
func textSize(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// absolutizeLinks 把子树内 <a href> / <img src> 的相对地址按 base 解析为绝对地址。
func absolutizeLinks(sel *goquery.Selection, base *url.URL) {
	if base == nil {
		return
	}
	resolve := func(s *goquery.Selection, attr string) {
		if v, ok := s.Attr(attr); ok {
			v = strings.TrimSpace(v)
			if v == "" || strings.HasPrefix(v, "#") {
				return
			}
			if ref, err := url.Parse(v); err == nil {
				s.SetAttr(attr, base.ResolveReference(ref).String())
			}
		}
	}
	sel.Find("a[href]").Each(func(_ int, a *goquery.Selection) { resolve(a, "href") })
	sel.Find("img[src]").Each(func(_ int, im *goquery.Selection) { resolve(im, "src") })
}
