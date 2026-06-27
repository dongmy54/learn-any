---
name: webextract
description: 根据 URL 把网页提取为 Markdown：单页提取，或用 crawl 批量抓取整站并输出到指定目录。当用户需要提取或者了解单个网页内容，或爬取（批量抓取）整站内容时使用。
---

webextract（github.com/dongmy54/webextract）把网页提取为标准 Markdown：单条 URL 提取单页，或用 `crawl` 子命令批量抓取整站并输出到指定目录。底层为全局 `webextract` 命令（需自行安装，见【环境准备】）。

## 环境准备（首次使用必做）

每次调用前先确认 `webextract` 命令已安装；未安装则先装好再用。

```bash
# 1. 检查命令是否就绪
command -v webextract >/dev/null 2>&1 && echo "✓ webextract 已安装" || echo "✗ 未安装，需先安装"

# 2. 未安装时安装（任选其一）：
#    macOS / Linux 一键脚本（推荐，装到 ~/.local/bin）：
curl -fsSL https://raw.githubusercontent.com/dongmy54/webextract/main/install.sh | bash
#    或已安装 Go 1.26+：
go install github.com/dongmy54/webextract@latest
#    Windows：到 https://github.com/dongmy54/webextract/releases 下载
#    webextract_<ver>_windows_*.zip 解压后加入 PATH
```

> **运行依赖**：本机需已装 Chrome / Chromium / Edge。一键脚本装到 `~/.local/bin`，若不在 PATH 需自行加入；装完用 `command -v webextract` 复核。

## 核心原则

1. **默认单页**：只给 URL 走单页；仅当用户明确要批量/整站/全部下载时才用 `crawl`，意图模糊按单页。
2. **批量收口**：crawl 必须用 `--max-pages` 限定页数、默认仅同域、配 `--rate-limit` 限流，避免压垮目标站。

## 工作流程

### 第一步：判断模式

| 信号 | 模式 |
|------|------|
| 一个 URL，想了解/总结/翻译/引用**这一页** | 单页（默认） |
| 明确要「整个站点 / 所有页面 / 批量 / 整站 / 文档站 / 全部下载」 | `crawl` 批量 |
| 意图模糊、未明确批量 | 一律走单页 |

### 第二步：单页提取（默认）

```bash
webextract <url>                                # 提取正文为 Markdown（最常用）
webextract -o /tmp/page.md <url>                # 写入文件，供后续多次引用
webextract -selector 'div.article-body' <url>   # 自动检测不准时手动指定正文区
webextract -include-source-url <url>            # 输出开头标注来源 URL，便于溯源
```

| 选项 | 说明 | 默认 |
|------|------|------|
| `<URL>` | 目标网页地址（必填） | - |
| `-o <文件>` | 写入指定文件（默认输出到标准输出） | stdout |
| `-selector <CSS>` | 指定正文区域（默认自动检测 main/article） | 自动 |
| `-timeout <秒>` | 等待页面渲染的最大秒数 | 60 |
| `-wait-for <CSS>` | 等待该 CSS 选择器出现后再提取 | - |
| `-user-agent <UA>` | 自定义 User-Agent（默认模拟桌面 Chrome） | 桌面 Chrome |
| `-include-source-url` | 在输出开头以 HTML 注释标注来源 URL | false |
| `-raw` | 输出渲染后的原始 HTML（调试用，而非 Markdown） | false |

> 单页仅输出 Markdown，无 JSON 选项；如需结构化索引见 crawl 的 `index.json`。

### 第三步：批量抓取（crawl）

`webextract crawl <URL>` 从种子 URL 广度优先抓取同域页面，按 URL 路径输出 Markdown 并生成索引。**入口页（种子）必达**：入口页抓取失败则整体失败，不会因 `--max-pages` 截断丢失。

```bash
webextract crawl <url> --max-pages 20 --output docs                  # 最多 20 页写进 docs/
webextract crawl <url> --depth 1                                     # 浅抓，仅入口相邻链接
webextract crawl <url> --max-pages 500 --workers 10 --rate-limit 2   # 大站点
```

| 选项 | 说明 | 默认 |
|------|------|------|
| `<URL>` | 种子 URL（入口页，必填） | - |
| `--depth <n>` | 最大爬取深度（入口页=0） | 2 |
| `--max-pages <n>` | 最大抓取页面数（含入口） | 100 |
| `--workers <n>` | 并发抓取数量（= 最大并发 tab 数） | 5 |
| `--rate-limit <n>` | 每秒最大请求数（0=不限；支持小数如 0.5） | 2 |
| `--output <目录>` | Markdown 输出目录 | output |
| `--allow-subdomains` | 允许抓取同注册域的子域（默认仅同 host） | false |
| `--crawl-timeout <秒>` | 爬取总超时秒数（0=不限） | 1800 |

单页参数（`--selector` / `--timeout` / `--user-agent` / `--include-source-url` / `--wait-for`）同样适用于 crawl，作用于每个抓取页。

**输出结构**：

```
output/
├── index.json     # 机器可读索引（每页 url/标题/深度/文件/状态）
├── index.md       # 人类可读索引（按深度分组的标题→路径列表）
├── install.md     # 各页 Markdown（按 URL 路径映射，站内链接改为相对路径可互跳）
├── config/mysql.md
└── api/user.md
```

## 失败排查

- `403 / 429`：站点反爬 → 换 `-user-agent`；批量时降 `--rate-limit`。
- `404`：URL 不存在 → 检查链接。
- 渲染超时/无输出 → 加 `-wait-for` 等关键元素，或调大 `-timeout`。
- crawl 报「入口页抓取失败」→ 先用单页 `webextract <url>` 确认种子可抓通。
- `webextract: command not found` → 回到【环境准备】先安装，并确认 PATH 含 `~/.local/bin`。

## 绝对不能做的事

1. 不要在用户只给单个 URL、未表达批量意图时擅自用 `crawl`。
2. 不要用本工具获取登录或付费墙后的内容（不支持鉴权）。
3. 批量抓取不要忘记用 `--max-pages` 收口。
4. 需要原始 HTML 源码时不要用本工具——改用 `curl`。
