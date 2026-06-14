package extractor

import (
	"net/url"
	"strings"

	"github.com/go-shiori/go-readability"

	"webextract/internal/model"
)

// Extract 用 readability 算法剥离导航/广告/侧边栏，只保留主体正文，并补全元数据。
// 注意：传入 pageURL 是为了让 readability 把正文里的相对链接（如 /a/b）
// 修正为绝对链接，否则转换出的 Markdown 链接全是坏的。
// 这一步静态/动态模式共用——两种模式都归一化为「一段 HTML」后进入此处。
func Extract(rawHTML string, pageURL *url.URL) (*model.Article, error) {
	art, err := readability.FromReader(strings.NewReader(rawHTML), pageURL)
	if err != nil {
		return nil, err
	}
	return &model.Article{
		Title:       art.Title,
		Description: art.Excerpt,
		URL:         pageURL.String(),
		HTML:        art.Content,
	}, nil
}
