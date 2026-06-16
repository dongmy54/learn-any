package extractor

import (
	"net/url"
	"strings"
	"testing"

	"webextract/internal/model"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("URL 解析失败: %v", err)
	}
	return u
}

// TestPrefersContainerWhenReadabilityDropsTable：
// 当 readability 产物缺少容器里的表格时，应改用语义主容器、并就地保留表格与顺序。
func TestPrefersContainerWhenReadabilityDropsTable(t *testing.T) {
	raw := `<html><body>
<nav>导航 应被剔除</nav>
<main>
  <h1>标题</h1>
  <p>第一段引导文字介绍下面的表格。</p>
  <table><thead><tr><th>键</th><th>说明</th></tr></thead>
  <tbody><tr><td>text</td><td>文本输出</td></tr></tbody></table>
  <p>表格后面还有一段补充说明文字。</p>
</main>
<footer>页脚 应被剔除</footer>
</body></html>`

	// 模拟 readability 把表格整片丢掉、只留下零散文字的情形。
	readArt := &model.Article{
		Title: "标题",
		HTML:  `<div><p>第一段引导文字介绍下面的表格。</p></div>`,
	}

	got, ok := CompleteArticle(raw, readArt, mustURL(t, "https://e.com/p"))
	if !ok {
		t.Fatal("应改用容器（readability 丢了表格），实际未切换")
	}
	if !strings.Contains(got.HTML, "<table") {
		t.Fatalf("容器产物应包含表格:\n%s", got.HTML)
	}
	if strings.Contains(got.HTML, "导航") || strings.Contains(got.HTML, "页脚") {
		t.Fatalf("导航/页脚噪声未被剔除:\n%s", got.HTML)
	}
	// 引导文字应在表格之前，补充说明在表格之后——顺序不被打乱。
	iIntro := strings.Index(got.HTML, "引导文字")
	iTable := strings.Index(got.HTML, "<table")
	iTail := strings.Index(got.HTML, "补充说明")
	if !(iIntro < iTable && iTable < iTail) {
		t.Fatalf("正文顺序被打乱: intro=%d table=%d tail=%d", iIntro, iTable, iTail)
	}
}

// TestKeepsReadabilityWhenComparable：
// readability 已完整（表格数相当、文字量相当）时，不应切换到容器，保留更干净的 readability 结果。
func TestKeepsReadabilityWhenComparable(t *testing.T) {
	raw := `<html><body><main>
  <p>这是一篇普通文章，正文内容和 readability 提取出来的基本一致，没有被丢弃的结构化内容。</p>
</main></body></html>`
	readArt := &model.Article{
		HTML: `<div><p>这是一篇普通文章，正文内容和 readability 提取出来的基本一致，没有被丢弃的结构化内容。</p></div>`,
	}
	if _, ok := CompleteArticle(raw, readArt, mustURL(t, "https://e.com/p")); ok {
		t.Fatal("readability 已足够完整时不应切换到容器")
	}
}

// TestNoContainerReturnsFalse：页面没有语义主容器时应放弃，沿用 readability。
func TestNoContainerReturnsFalse(t *testing.T) {
	raw := `<html><body><div><p>没有 main/article/[role=main] 容器的页面。</p></div></body></html>`
	readArt := &model.Article{HTML: `<p>没有 main/article 容器的页面。</p>`}
	if _, ok := CompleteArticle(raw, readArt, mustURL(t, "https://e.com/p")); ok {
		t.Fatal("无语义容器时不应切换")
	}
}

// TestAbsolutizesRelativeLinks：容器路径下相对链接/图片应被补成绝对地址。
func TestAbsolutizesRelativeLinks(t *testing.T) {
	raw := `<html><body><main>
  <h1>标题</h1>
  <p>正文里有一个<a href="/docs/next">相对链接</a>和一张<img src="../img/a.png">图。</p>
  <table><tr><th>K</th></tr><tr><td>v</td></tr></table>
  <p>补充正文确保容器比 readability 大很多很多很多，触发切换条件。再加一些字数。再加一些字数。</p>
</main></body></html>`
	readArt := &model.Article{HTML: `<p>少量文字</p>`}
	got, ok := CompleteArticle(raw, readArt, mustURL(t, "https://e.com/docs/guide/"))
	if !ok {
		t.Fatal("应切换到容器")
	}
	if !strings.Contains(got.HTML, "https://e.com/docs/next") {
		t.Fatalf("相对链接未绝对化:\n%s", got.HTML)
	}
	if !strings.Contains(got.HTML, "https://e.com/docs/img/a.png") {
		t.Fatalf("相对图片地址未绝对化:\n%s", got.HTML)
	}
}
