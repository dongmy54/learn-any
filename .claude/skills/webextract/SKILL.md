---
name: webextract
description: 提取任意网页正文转 Markdown/JSON（单页，默认），或从种子 URL 出发批量抓取整个站点相关页面并落盘到目录（crawl 子命令）。当用户给出 URL 想了解其内容、抓取网页正文作为上下文、或要批量下载整个站点/文档站时使用。readability 算法剥离导航/广告/侧边栏噪声只留主体正文；支持静态页面与 JS 动态渲染（React/Vue 等 SPA）页面。
---

webextract 是一个网页正文提取工具，有两种模式：**单页模式**（默认）提取一个 URL 的正文并输出 Markdown/JSON；**批量模式**（`crawl` 子命令）从种子 URL 出发广度优先抓取整个站点相关页面，一次性批量落盘。底层用 readability 算法剥离导航/广告/侧边栏噪声，只保留主体正文；支持静态页面与 React/Vue 等 SPA 动态渲染页面。

随 skill 分发预编译二进制（覆盖 macOS/Linux × arm64/amd64），通过同目录包装脚本调用，**克隆项目即可用，无需安装**。所有示例统一通过该脚本调用：

```bash
WX=.claude/skills/webextract/scripts/webextract.sh   # 定义一次，下文统一用 "$WX" 调用
```

## 核心原则

1. **默认单页，批量按需**：用户只给 URL 想看这一页 → 单页（默认）；只有明确表达「整个站点 / 所有页面 / 批量 / 整站 / 文档站 / 全部下载」时才用 `crawl`；意图模糊一律走单页。
2. **主链接优先**：批量模式下，用户输入的种子 URL 页面同步优先抓取、内容必达，固定写入 `index.md`，绝不因 `--max` 截断而丢失。
3. **正文优先**：用 readability 剥离噪声只留主体正文，而非原始 HTML 源码（需源码用 `curl`）。
4. **静态优先，动态兜底**：默认静态抓取（快，~1 秒）；仅当输出明显残缺、过短，或站点是已知 SPA 时才加 `--render`。
5. **礼貌爬取**：批量抓取必须用 `--max` 收口、默认仅同域、固定并发，避免对目标站造成压力。

## 何时使用

- 给一个 URL，想了解「这页讲了什么」→ 单页
- 把网页正文作为上下文来回答问题、做摘要/翻译/分析/引用 → 单页
- 读取在线文档（官方文档、API 文档）的内容 → 单页
- 一次性把一个站点/文档站的多个相关页面批量抓取下来 → `crawl` 批量

**不要用于**：需要原始 HTML 源码（含全部标签/脚本）的场景——用 `curl`；需要登录或付费墙后的内容——不支持鉴权。

## 模式选择（单页 / 批量）

调用本 skill 时，根据用户自然语言意图选择模式：

| 用户意图 | 模式 | 调用 |
|---------|------|------|
| 只给 URL，想看/总结/提取**这一页** | 单页（默认） | `"$WX" <url>` |
| 明确要「整个站点 / 所有页面 / 批量 / 整站 / 全部下载」 | 批量 | `"$WX" crawl <url>` |
| 意图模糊、未明确 | 单页（默认） | `"$WX" <url>` |

- 单页是高频、低成本、确定性操作；批量耗时长、产生多文件、可能触发反爬，应作为用户主动选择的「重操作」，避免误触发。
- 批量模式默认参数 `--max 10 --depth 1 --same-domain`；用户未给规模参数时用默认值即可，不必逐项确认。

## 单页用法

```bash
# 提取正文为 Markdown（最常用）
"$WX" https://example.com

# SPA 动态页面（React/Vue 等 JS 渲染）加 --render
"$WX" --render https://some-spa.com/article

# 输出 JSON（title/description/url/content 四字段，适合程序解析）
"$WX" --format json https://example.com

# 写到文件供后续多次引用，避免重复抓取
"$WX" https://example.com -o /tmp/page.md
```

### 单页参数

| 参数 | 说明 | 默认 |
|------|------|------|
| `<url>` | 目标网页地址（必填） | - |
| `-r, --render` | 启用无头浏览器渲染，用于 JS 动态页面 | false |
| `-f, --format` | 输出格式：`markdown` \| `json` | markdown |
| `--wait <时长>` | 动态渲染后额外等待，如 `3s` | 2s |
| `-o <文件>` | 输出到文件（默认 stdout） | - |

### 输出处理技巧

```bash
"$WX" <url> | head -100                       # 内容很长时截断，省 token
"$WX" --format json <url> | jq -r '.title'    # JSON 模式用 jq 取字段
```

## 批量用法（crawl）

`"$WX" crawl <url>` 从种子 URL 出发，**广度优先**抓取其同域相关页面，每页正文（Markdown）写入指定目录。主链接页面同步优先抓取、内容必达，固定命名为 `index.md`；其余页面按站点分子目录组织。

```bash
# 抓最多 20 页写进 docs/（主链接 → docs/<host>/index.md）
"$WX" crawl https://react.dev/learn --max 20 --output docs

# 只抓主链接这一页（--max=1 → 仅 docs/<host>/index.md）
"$WX" crawl https://example.com --max 1

# SPA 站点批量（透传 --render）
"$WX" crawl --render https://some-spa.com/docs --max 15
```

### crawl 参数

| 参数 | 说明 | 默认 |
|------|------|------|
| `<url>` | 种子 URL（主链接，必填） | - |
| `--max <n>` | 最大抓取页数（含种子）；`--max 1` = 仅主链接 | 10 |
| `--depth <n>` | 抓取深度（0 = 仅种子页） | 1 |
| `--concurrency <n>` | 并发 worker 数 | 5 |
| `--same-domain` | 仅抓同域链接（防爬虫外溢），`=false` 跨域 | true |
| `-o, --output <目录>` | 输出目录 | docs |
| `-r, --render` | SPA 动态渲染（透传，与单页一致） | false |

**批量输出结构**：`<output>/<host>/index.md`（主链接）+ `<host>/<path-slug>.md`（其余页）。每页首行带标题与源 URL 便于溯源。`--max` 是抓取尝试的上限（礼貌爬取），个别页失败会少抓但不会超抓。

## 决策流程

1. **判模式**：用户要单页还是整站？→ 单页 / `crawl`（见「模式选择」）
2. **静态优先**：先不加 `--render` 抓取（快）
3. **查输出**：内容完整且符合预期 → 直接用；明显残缺/过短/站点是已知 SPA → 加 `--render` 重抓
4. **要结构化字段**（标题、摘要）→ 单页加 `--format json`

## 失败排查

- `HTTP 403/429`：站点反爬，加 `--render`（带浏览器 UA 与行为更不易被拦）
- `HTTP 404`：URL 不存在，检查链接
- 动态模式超时/无输出：增大 `--wait`（如 `--wait 5s`）等待异步加载完成
- crawl 报「主链接抓取失败」：种子 URL 不可达会立即终止（主链接必达是硬约束），先用单页 `"$WX" <url>` 确认种子可抓通，再跑 `crawl`

## 工具构建（仅当平台无预编译二进制时）

`scripts/` 下已附带 `webextract-{darwin,linux}-{arm64,amd64}` 四个预编译二进制，包装脚本会自动选用。**仅当**平台不在其中（如 Windows、Linux armv7），或想自行重新构建时才需手动编译（需 Go 1.26+）：

```bash
cd .claude/skills/webextract/scripts/webextract
OS="$(uname -s | tr A-Z a-z)"; ARCH="$(uname -m)"
case "$ARCH" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; esac
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o "../webextract-$OS-$ARCH" ./cmd/webextract
```

动态模式（`--render`）额外需系统已安装 Google Chrome。

## 绝对不能做的事

1. 不要在用户只给单个 URL、未表达批量意图时擅自用 `crawl`——默认走单页
2. 不要用本工具获取登录或付费墙后的内容（不支持鉴权）
3. 批量抓取不要忘记用 `--max` 收口，避免无限制抓取压垮目标站
4. 不要用本工具获取网页原始 HTML 源码——那种用 `curl`
5. 不要把命令示例写成裸 `webextract`——必须通过 `scripts/webextract.sh` 调用，否则克隆项目后无法直接运行
