package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"webextract/internal/codeguard"
	"webextract/internal/converter"
	"webextract/internal/extractor"
	"webextract/internal/fetcher"
	"webextract/internal/model"
	"webextract/internal/tableguard"
)

// Run 是流水线的编排核心：Fetch → Extract → Convert → Render。
// 它只依赖 Fetcher 接口，对静态/动态实现完全无感知——这是可扩展性的来源。
func Run(ctx context.Context, f fetcher.Fetcher, rawURL, format string, w io.Writer) error {
	// 1. 抓取（策略可替换）
	html, err := f.Fetch(ctx, rawURL)
	if err != nil {
		return fmt.Errorf("抓取失败: %w", err)
	}

	// 2. 保护代码块：把 <pre> 的 table+shiki 高亮结构抽成纯文本，就地重写成
	// 标准 <pre><code>，否则 readability 会把这种复杂结构的代码块当噪声丢弃。
	protectedHTML, blocks := codeguard.Protect(html)

	// 2.5 抢救正文表格：Fern/Mintlify 等框架把表格单独裹在低密度容器里，readability
	// 会把整个容器连同表格一起排除。先在 readability 之前收集所有正文表格的 markdown，
	// 提取后再把丢失的回填到正文末尾。详见 tableguard 包注释。
	savedTables := tableguard.Collect(protectedHTML)

	// 3. 提取正文 + 元数据
	pageURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL 非法: %w", err)
	}
	art, err := extractor.Extract(protectedHTML, pageURL)
	if err != nil {
		return fmt.Errorf("正文提取失败: %w", err)
	}

	// 3.5 完整性兜底：readability 对文档框架/表格密集页会整片丢弃结构化内容。
	// 若存在更完整的语义主容器（main/article/[role=main]），改用其完整子树作正文，
	// 保证内容不丢失、顺序不错乱。普通文章页判定为不更完整，沿用 readability。
	usedContainer := false
	if cart, ok := extractor.CompleteArticle(protectedHTML, art, pageURL); ok {
		art = cart
		usedContainer = true
	}

	// 4. 补回代码块语言：readability 会剥掉 <code> 的 class，按序号标记把语言补回。
	art.HTML = codeguard.StampLanguage(art.HTML, blocks)

	// 4.5 探测 readability 实际保留了哪些表格，供后续去重回填。
	presentTables := tableguard.PresentSignatures(art.HTML)

	// 5. 正文 HTML → Markdown（含 Table 插件，幸存的表格在此转换为 GFM）
	art.Markdown, err = converter.ToMarkdown(art.HTML)
	if err != nil {
		return fmt.Errorf("Markdown 转换失败: %w", err)
	}

	// 5.5 回填被 readability 丢弃的表格（已保留的不会重复）。
	// 容器路径下表格已就地完整，跳过回填，避免把主容器之外的噪声表格误并进来。
	if !usedContainer {
		art.Markdown = tableguard.Restore(art.Markdown, savedTables, presentTables)
	}

	// 6. 按格式输出
	return render(w, art, format)
}

func render(w io.Writer, art *model.Article, format string) error {
	switch format {
	case "json":
		// JSON 结构对齐 webReader：title/description/url/content，便于程序解析与 AI 消费
		return json.NewEncoder(w).Encode(map[string]string{
			"title":       art.Title,
			"description": art.Description,
			"url":         art.URL,
			"content":     art.Markdown,
		})
	default: // markdown
		_, err := fmt.Fprint(w, art.Markdown)
		return err
	}
}
