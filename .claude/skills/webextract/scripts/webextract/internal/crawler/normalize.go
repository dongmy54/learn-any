package crawler

import (
	"errors"
	"net/url"
	"strings"
)

// ErrNotAbsolute 表示 URL 缺少 host，无法作为绝对地址抓取。
var ErrNotAbsolute = errors.New("URL 非绝对地址")

// Normalize 把语义等价的 URL 归一为唯一字符串，作为「已访问」集合的 key。
// 否则同一页面会因锚点、默认端口、query 顺序、尾部斜杠等差异被重复抓取。
//
// 处理项：
//   - 丢弃锚点（#section 不改变页面内容）
//   - 去默认端口（:80 / :443 是冗余信息）
//   - query 参数按键排序（?b=2&a=1 ≡ ?a=1&b=2）
//   - 尾部斜杠归一（根路径 "/" 保留）
func Normalize(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.Hostname() == "" {
		return "", ErrNotAbsolute
	}
	u.Fragment = ""
	u.RawFragment = ""
	if (u.Scheme == "http" && u.Port() == "80") ||
		(u.Scheme == "https" && u.Port() == "443") {
		u.Host = u.Hostname()
	}
	if u.RawQuery != "" {
		u.RawQuery = u.Query().Encode() // Encode 内部按 key 排序
	}
	// 空 path 与根 path "/" 视为同一页面（https://go.dev ≡ https://go.dev/）。
	// 否则二者被判为不同 URL：根链接不会被 visited 去重，入队后 urlToPath 又映射到
	// 同一个 index.md，覆盖种子页内容——主链接数据丢失。
	if u.Path == "" {
		u.Path = "/"
	}
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String(), nil
}
