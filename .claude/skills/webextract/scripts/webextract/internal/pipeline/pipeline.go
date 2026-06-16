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

	// 3. 提取正文 + 元数据
	pageURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL 非法: %w", err)
	}
	art, err := extractor.Extract(protectedHTML, pageURL)
	if err != nil {
		return fmt.Errorf("正文提取失败: %w", err)
	}

	// 4. 补回代码块语言：readability 会剥掉 <code> 的 class，按序号标记把语言补回。
	art.HTML = codeguard.StampLanguage(art.HTML, blocks)

	// 5. 正文 HTML → Markdown
	art.Markdown, err = converter.ToMarkdown(art.HTML)
	if err != nil {
		return fmt.Errorf("Markdown 转换失败: %w", err)
	}

	// 4. 按格式输出
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
