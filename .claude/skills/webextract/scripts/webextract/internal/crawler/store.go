package crawler

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"webextract/internal/model"
)

// urlToPath 把 URL 映射为安全的本地文件路径：{outputDir}/{host}/{name}.md
// isSeed=true 时主链接固定用 index.md，凸显其优先地位。
func urlToPath(rawURL, outputDir string, isSeed bool) string {
	u, _ := url.Parse(rawURL)
	name := "index"
	if !isSeed {
		slug := strings.Trim(u.Path, "/")
		if slug == "" {
			// 站点根（如 https://host/）：不能叫 index，否则会覆盖种子专属的 index.md
			name = "home"
		} else {
			name = strings.ReplaceAll(slug, "/", "-")
		}
		// 同 path 不同 query：追加短哈希，避免互相覆盖
		if u.RawQuery != "" {
			sum := sha1.Sum([]byte(u.RawQuery))
			name += "_" + hex.EncodeToString(sum[:4])
		}
		// 兜底：index.md 是种子专属文件名，任何非种子页都不得占用，
		// 否则爬到的根页/路径恰为 "index" 的页面会覆盖种子页（曾导致种子内容丢失）。
		if name == "index" {
			sum := sha1.Sum([]byte(rawURL))
			name = "index_" + hex.EncodeToString(sum[:4])
		}
	}
	return filepath.Join(outputDir, u.Hostname(), name+".md")
}

// WriteArticle 把正文写入文件，首行带标题与源 URL 便于溯源。
// 不同 URL 经 Normalize 后唯一 → 文件路径唯一 → 并发写不同文件天然安全。
// os.MkdirAll 幂等，并发调用同样安全。
func WriteArticle(outputDir string, art *model.Article, isSeed bool) string {
	path := urlToPath(art.URL, outputDir, isSeed)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ""
	}
	tag := ""
	if isSeed {
		tag = "（主链接）"
	}
	body := "# " + art.Title + tag + "\n\n> 源: " + art.URL + "\n\n" + art.Markdown
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return ""
	}
	return path
}
