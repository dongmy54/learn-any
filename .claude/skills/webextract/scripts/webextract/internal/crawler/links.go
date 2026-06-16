package crawler

import (
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// binaryExts：抓了也无法用 readability 提取正文的资源文件，直接跳过。
var binaryExts = map[string]bool{
	".zip": true, ".pdf": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".svg": true, ".webp": true, ".ico": true, ".mp4": true,
	".mp3": true, ".wav": true, ".css": true, ".js": true, ".json": true,
	".xml": true, ".gz": true, ".tar": true, ".rar": true, ".exe": true,
	".dmg": true, ".apk": true,
}

// ExtractLinks 从整页 HTML 抽取可抓取的链接（绝对地址，已去重）。
//   - pageURL：用于把相对链接（/a/b、../c、//host）解析为绝对地址
//   - allowHost：空表示不限域；非空则只保留同 host 的链接（防爬虫外溢）
func ExtractLinks(rawHTML string, pageURL *url.URL, allowHost string) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href == "" || strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") {
			return // 无效 / 非导航链接
		}
		// ResolveReference 自动处理 /path、../、//host、含 scheme 四种相对形态
		ref, err := url.Parse(href)
		if err != nil {
			return
		}
		abs := pageURL.ResolveReference(ref)
		if abs.Scheme != "http" && abs.Scheme != "https" {
			return // 仅保留 http/https
		}
		if allowHost != "" && abs.Hostname() != allowHost {
			return // 默认同域过滤
		}
		if binaryExts[strings.ToLower(path.Ext(abs.Path))] {
			return // 跳过二进制 / 资源文件
		}
		norm, err := Normalize(abs.String())
		if err != nil {
			return
		}
		if _, dup := seen[norm]; dup {
			return
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	})
	return out
}
