---
name: webextract
description: 把任意网页提取为 Markdown/JSON。默认单页提取；crawl 子命令从种子 URL 批量抓取整站相关页面落盘。完整保留正文的表格/代码/列表等结构化内容并剥离导航/广告/侧边栏，支持静态页与 React/Vue 等 SPA 动态页。当用户给出 URL 想了解、总结、翻译、引用网页内容，或要批量下载整站/文档站时使用。
---

webextract 将网页正文提取为 Markdown（或 JSON），优先保证「人在页面上能看到的内容」完整不丢失——表格、代码块、列表等结构化内容均按原顺序保留，同时剥离导航/广告/侧边栏等噪声。

```bash
WX=.claude/skills/webextract/scripts/webextract.sh   # 定义一次，下文统一用 "$WX"
```

## 单页还是批量
- 给一个 URL，想了解/总结/翻译/引用**这一页**的内容 → 单页（默认）
- 读在线文档、API 文档、技术文章作为上下文 → 单页（默认）
- 明确要「整个站点 / 所有页面 / 批量 / 整站 / 文档站 / 全部下载」 → `crawl` 批量
- 意图模糊、未明确批量 → 一律走单页


## 单页用法（默认）

```bash
"$WX" <url>                          # 提取正文为 Markdown（最常用）
"$WX" --render <url>                 # SPA 动态页（React/Vue 等 JS 渲染）
"$WX" --format json <url>            # 输出 JSON：title/description/url/content
"$WX" <url> -o /tmp/page.md          # 写入文件，供后续多次引用
"$WX" <url> | head -100              # 内容很长时截断省 token
```

| 参数 | 说明 | 默认 |
|------|------|------|
| `<url>` | 目标网页地址（必填） | - |
| `-r, --render` | 启用无头浏览器渲染（JS 动态页） | false |
| `-f, --format` | 输出格式：`markdown` \| `json` | markdown |
| `--wait <时长>` | 动态渲染后额外等待，如 `3s` | 2s |
| `-o <文件>` | 输出到文件（默认 stdout） | - |

## 批量用法（crawl）

`"$WX" crawl <url>` 从种子 URL 广度优先抓取同域相关页面，每页正文写入指定目录。种子页**内容必达**，固定写入 `<output>/<host>/index.md`，绝不因 `--max` 截断而丢失；其余页按 `<host>/<path-slug>.md` 组织，每页首行带标题与源 URL 便于溯源。

```bash
"$WX" crawl <url> --max 20 -o docs        # 最多 20 页写进 docs/
"$WX" crawl <url> --max 1                 # 仅主链接这一页
"$WX" crawl --render <url> --max 15       # SPA 站点批量
```

| 参数 | 说明 | 默认 |
|------|------|------|
| `<url>` | 种子 URL（主链接，必填） | - |
| `--max <n>` | 最大抓取页数（含种子），`--max 1`=仅主链接 | 10 |
| `--depth <n>` | 抓取深度（0=仅种子页） | 1 |
| `--concurrency <n>` | 并发 worker 数 | 5 |
| `--same-domain` | 仅抓同域链接，`=false` 跨域 | true |
| `-r, --render` | SPA 动态渲染（透传） | false |
| `-o, --output <目录>` | 输出目录 | docs |

批量时 默认输出到当前项目下的docs目录中

## 核心原则

1. **默认单页**：只给 URL 走单页；仅在用户明确表达批量/整站意图时才用 `crawl`，意图模糊一律单页。
2. **完整优先**：保证页面可见正文（含表格/代码/列表）完整且保序，不是干净但残缺。
3. **静态优先，动态兜底**：先不加 `--render`（快，~1 秒）；仅当输出明显残缺/过短、或已知是 SPA 时才加 `--render`。
4. **礼貌爬取**：批量必须用 `--max` 收口、默认仅同域，避免压垮目标站。

## 决策流程

1. 判模式：单页还是整站？→ 单页 / `crawl`
2. 静态优先：先不加 `--render` 抓取
3. 查输出：完整即用；明显残缺/过短/已知 SPA → 加 `--render` 重抓
4. 要结构化字段（标题、摘要）→ 加 `--format json`

## 失败排查

- `HTTP 403/429`：站点反爬 → 加 `--render`
- `HTTP 404`：URL 不存在 → 检查链接
- 动态模式超时/无输出 → 增大 `--wait`（如 `--wait 5s`）
- crawl 报「主链接抓取失败」→ 先用单页 `"$WX" <url>` 确认种子可抓通

## 工具构建（仅当平台无预编译二进制时）

`scripts/` 下已附带 `webextract-{darwin,linux}-{arm64,amd64}`，包装脚本自动选用。仅当平台不在其中（如 Windows）或需重新构建时才手动编译（需 Go 1.26+）：

```bash
cd .claude/skills/webextract/scripts/webextract
OS="$(uname -s | tr A-Z a-z)"; ARCH="$(uname -m)"
case "$ARCH" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; esac
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "../webextract-$OS-$ARCH" ./cmd/webextract
```

动态模式（`--render`）额外需系统已安装 Google Chrome。

## 绝对不能做的事

1. 不要在用户只给单个 URL、未表达批量意图时擅自用 `crawl`
2. 不要用本工具获取登录或付费墙后的内容（不支持鉴权）
3. 批量抓取不要忘记用 `--max` 收口
4. 需要原始 HTML 源码时不要用本工具——改用 `curl`
5. 不要写成裸 `webextract`——必须通过 `scripts/webextract.sh` 调用，否则克隆项目后无法直接运行
