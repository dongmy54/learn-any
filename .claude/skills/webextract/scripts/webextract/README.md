# webextract

输入任意网页 URL，提取正文并输出 Markdown / JSON 的命令行工具。给 AI Agent、脚本、开发者用的网页内容提取器。

## 安装

```bash
# 需要系统已安装 Go 1.22+ 和（动态模式可选）Google Chrome
go build -o bin/webextract ./cmd/webextract
```

## 使用

```bash
# 基础用法：静态抓取，输出 markdown
webextract https://example.com/post

# 动态页面：启用无头浏览器渲染
webextract --render https://spa-site.com/article/123

# 输出 JSON（结构对齐 webReader：title/description/url/content）
webextract --format json https://example.com/post

# 写入文件
webextract https://example.com/post -o post.md
```

### 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-r, --render` | 启用无头浏览器渲染（用于 JS 动态页面） | false |
| `-f, --format` | 输出格式：`markdown` \| `json` | markdown |
| `--wait` | 动态渲染后的额外等待时间 | 2s |
| `-o, --output` | 输出到文件（默认 stdout） | - |

## 架构

策略模式 + 流水线：只有「抓取」这一步可替换，后续提取与转换完全复用。

```
URL → Fetcher(静态/动态) → 原始HTML → readability提取正文 → html-to-markdown → 输出
```

详见 `docs/webextract技术方案.md`。
