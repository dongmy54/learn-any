---
name: webextract
description: 提取任意网页的正文内容并转为 Markdown 或 JSON。当用户给出 URL 想了解其内容、需要抓取/读取网页正文、需要把网页文档作为上下文来回答问题时使用。自动用 readability 算法剥离导航/广告/侧边栏等噪声，只保留主体正文；支持静态页面和 JS 动态渲染（React/Vue 等 SPA）页面。
---

# webextract — 网页正文提取工具

输入一个 URL，输出该网页的**正文内容**（Markdown 或 JSON）。底层用 readability 算法自动剥离导航栏、广告、侧边栏、脚本样式等噪声，只保留主体正文——比 `curl` 拿原始 HTML 或 `WebFetch` 干净得多，直接可用作上下文。

## 何时使用

- 用户给出一个 URL，想了解「这个网页讲了什么」
- 需要把某个网页文档/博客/资讯的内容作为上下文来回答问题
- 需要抓取网页正文做摘要、翻译、分析、引用
- 需要读取在线文档（如官方文档、API 文档）的内容

**不要用于**：需要网页原始 HTML 源码（含全部标签/脚本）的场景——那种用 `curl`；需要登录或付费墙后的内容——本工具不支持鉴权。

## 基本用法

skill 自带预编译二进制（覆盖 macOS / Linux × arm64 / amd64），通过同目录的 `scripts/webextract.sh` 包装脚本调用。该脚本会自动识别当前平台选对应二进制、修复下载后丢失的可执行权限，**克隆项目即可用，无需任何安装**。命令格式与原 `webextract` 完全一致，参数全部透传：

```bash
# 提取网页正文为 Markdown（最常用）
.claude/skills/webextract/scripts/webextract.sh <url>

# 动态页面（React/Vue 等 JS 渲染的 SPA）加 --render
.claude/skills/webextract/scripts/webextract.sh --render <url>

# 输出 JSON（含 title/description/url/content 四字段，适合程序解析）
.claude/skills/webextract/scripts/webextract.sh --format json <url>
```

> 下文为简洁起见，所有示例写作 `webextract`，实际调用时请替换为上方脚本路径（路径偏长，可先 `WX=.claude/skills/webextract/scripts/webextract.sh` 再用 `"$WX" <url>`）。

## 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `<url>` | 目标网页地址（必填，仅一个位置参数） | - |
| `-r, --render` | 启用无头浏览器渲染，用于 JS 动态页面 | false |
| `-f, --format` | 输出格式：`markdown` \| `json` | markdown |
| `--wait <时长>` | 动态渲染后额外等待（给异步数据加载留时间），如 `3s` | 2s |
| `-o <文件>` | 输出到文件（默认 stdout） | - |

## 抓取模式选择（静态 vs 动态）

**默认用静态模式**（不加 `--render`）：速度快（~1 秒），适合绝大多数博客、文档、资讯、Wiki 等服务端渲染站点。

**何时加 `--render` 切换动态模式**：
- 目标是已知 SPA 框架站点（React/Vue/Next.js client-side 等）
- 静态模式提取出的内容明显过少或为空（很可能是 JS 渲染的空壳页面）
- 页面内容依赖异步加载（如滚动加载、点击展开）

动态模式会启动 headless Chrome，较慢（~6 秒），仅在必要时使用。

## 输出处理技巧

```bash
# 内容可能很长，截断查看前 100 行避免 token 浪费
webextract <url> | head -100

# JSON 模式用 jq 提取特定字段
webextract --format json <url> | jq -r '.title, .content'

# 保存到文件供后续多次引用，避免重复抓取
webextract <url> -o /tmp/page.md
```

## 决策流程

1. 先用静态模式 `webextract <url>` 抓取（快）
2. 检查输出：若内容完整且符合预期 → 直接使用
3. 若内容明显残缺、过短、或站点是已知 SPA → 改用 `webextract --render <url>` 重新抓取
4. 若需要结构化字段（标题、摘要）→ 加 `--format json`

## 失败排查

- `抓取失败: HTTP 403/429`：站点反爬，尝试加 `--render`（带浏览器 UA 与行为更不易被拦）
- `抓取失败: HTTP 404`：URL 不存在，检查链接
- 动态模式超时/无输出：适当增大 `--wait`（如 `--wait 5s`），等待异步加载完成

## 工具构建（仅当当前平台无预编译二进制时）

`scripts/` 下已附带 `webextract-darwin-arm64`、`webextract-darwin-amd64`、`webextract-linux-amd64`、`webextract-linux-arm64` 四个预编译二进制，`webextract.sh` 会自动选用。**仅当**你的平台不在其中（如 Windows、Linux armv7），或想自行重新构建时，才需手动编译：

```bash
# 从项目根目录运行；源码在 webextract/，需 Go 1.26+
cd webextract
OS="$(uname -s | tr A-Z a-z)"; ARCH="$(uname -m)"
case "$ARCH" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; esac
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o "../.claude/skills/webextract/scripts/webextract-$OS-$ARCH" ./cmd/webextract
```

动态模式（`--render`）额外需要系统已安装 Google Chrome。
