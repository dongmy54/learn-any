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
		if slug != "" {
			name = strings.ReplaceAll(slug, "/", "-")
		}
		// 同 path 不同 query：追加短哈希，避免互相覆盖
		if u.RawQuery != "" {
			sum := sha1.Sum([]byte(u.RawQuery))
			name += "_" + hex.EncodeToString(sum[:4])
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
