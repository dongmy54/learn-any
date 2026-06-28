---
name: webextract
description: 根据 URL 把网页提取为 Markdown：单页提取；整站抓取默认按导航菜单逐页抓取（sitemap-crawl，结构清晰不污染），仅在明确要 BFS 全站抓取时用 crawl。当用户需要提取或了解单个网页内容，或批量抓取整站内容时使用。
---


webextract（github.com/dongmy54/webextract）把网页提取为标准 Markdown。**单页**直接提取；**整站**默认走导航驱动抓取 `sitemap-crawl`（按主导航菜单逐页抓取，只抓菜单内页面、不蔓延），仅当明确要"抓全部链接 / BFS 全站"时才用 `crawl`。底层为全局 `webextract` 命令（需 v0.2.0+，含 `sitemap-crawl`，自行安装见【环境准备】）。

## 环境准备（首次使用必做）

每次调用前先确认 `webextract` 已安装且为新版（v0.2.0+，旧版无 `sitemap-crawl`）；未装/过旧则先装好再用。

```bash
# 1. 检查命令就绪 + sitemap-crawl 子命令存在（旧版无此子命令）
#    ⚠️ 不能靠退出码判断：webextract 的 -h 走错误分支、退出码恒为 1，
#       必须用 grep 看子命令名是否出现在 help 输出里。
command -v webextract >/dev/null 2>&1 \
  && webextract sitemap-crawl -h 2>&1 | grep -q 'sitemap-crawl' \
  && echo "✓ webextract v0.2.0+ 就绪（含 sitemap-crawl）" \
  || echo "✗ 未安装或版本过旧，需先安装"

# 2. 安装（任选其一）：
#    macOS / Linux 一键脚本（推荐，装到 ~/.local/bin）：
curl -fsSL https://raw.githubusercontent.com/dongmy54/webextract/main/install.sh | bash
#    或已安装 Go 1.26+：
go install github.com/dongmy54/webextract@latest
#    Windows：到 https://github.com/dongmy54/webextract/releases 下载
#    webextract_<ver>_windows_*.zip 解压后加入 PATH
```

> **运行依赖**：本机需已装 Chrome / Chromium / Edge。一键脚本装到 `~/.local/bin`，若不在 PATH 需自行加入；装完用上面的检查命令复核 `sitemap-crawl` 子命令可用。

## 核心原则

1. **默认单页**：只给 URL 走单页；仅当用户明确要批量/整站/全部下载时才进入整站抓取，意图模糊按单页。
2. **整站默认导航驱动**：整站抓取默认用 `sitemap-crawl`——按主导航菜单逐页抓取，只抓菜单列出的页面、不顺着正文链接蔓延（从源头杜绝"相关阅读/热门排行"污染），输出按菜单树结构组织。仅当用户明确要"抓全部页面 / 全部链接 / BFS / 域名下所有页"时才用 `crawl`。
3. **收口 + 限流**：`crawl` 用 `--max-pages`/`--depth` 限定；`sitemap-crawl` 用 `--nav-depth` 限定菜单层数；两者都配 `--rate-limit` 限流，避免压垮目标站。
4. **输出目录约定**：整站抓取（`sitemap-crawl` / `crawl`）一律用 `--output crawl/<入口域名>`，**不要用工具默认的 `output`**。域名取入口 URL 的 host（如 `https://docs.example.com` → `docs.example.com`、`https://go.dev` → `go.dev`）。工具会自动创建该目录，并在其下按菜单树（sitemap-crawl）或 URL 路径（crawl）展开子目录与文件，无需手动建 `crawl/`。

## 工作流程

> **结构**：先按用户意图判断，再进入下面**三条互斥分支之一**——单页 / sitemap-crawl / crawl 是并列的三选一，不是依次执行的步骤，一次只走一条。

```
用户给 URL
   │
   ├─ 只看这一页 / 意图模糊 ─────→ 分支 A：单页提取         （webextract <url>）
   │
   ├─ 要整站 / 批量（默认） ────→ 分支 B：导航驱动           （sitemap-crawl，可先用 nav 预览）
   │
   └─ 明确"抓全部链接 / BFS" ──→ 分支 C：全站 BFS            （crawl）
```

### 先判断走哪条分支

| 信号 | 走哪条分支 |
|------|------|
| 一个 URL，想了解/总结/翻译/引用**这一页** | 分支 A：单页提取 |
| 整站 / 文档站 / 所有页面 / 批量下载（**默认**） | 分支 B：`sitemap-crawl` 导航驱动 |
| 明确要「**抓全部链接 / BFS / 不限导航 / 域名下所有页**」 | 分支 C：`crawl` 全站 BFS |
| 意图模糊、未明确批量 | 分支 A：单页提取 |

---

### 分支 A：单页提取（默认）

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

> 单页仅输出 Markdown，无 JSON 选项；如需结构化索引见分支 B / C 的 `index.json`。

---

### 分支 B：整站抓取 · 导航驱动（sitemap-crawl，默认整站方式）

渲染入口页、自动识别**主导航菜单**，把每个带 URL 的菜单项当作一个独立抓取目标，**只抓该页正文、不再提取页面内链接做后续抓取**，输出严格按菜单树形结构组织目录。适合文档站及一切有清晰导航的站点。

```bash
# 方式 A：一步式（最常用）——现场提取导航再逐页抓取，输出到 crawl/<域名>/
webextract sitemap-crawl https://docs.example.com --output crawl/docs.example.com

# 方式 B：两步式——先存导航结构，可人工编辑/裁剪后再抓
webextract nav https://docs.example.com --format json -o nav.json        # 第一阶段：提取导航
webextract sitemap-crawl --nav nav.json --output crawl/docs.example.com  # 第二阶段：按 nav.json 抓取

# 导航识别不准时手动指定导航容器
webextract sitemap-crawl https://docs.example.com --nav-selector 'nav.sidebar' --output crawl/docs.example.com

# 只抓导航前 2 层（大站点收口）
webextract sitemap-crawl https://docs.example.com --nav-depth 2 --output crawl/docs.example.com
```

| 选项 | 说明 | 默认 |
|------|------|------|
| `<URL>` | 入口页，现场提取导航（与 `--nav` 二选一） | - |
| `--nav <文件>` | 读取第一阶段 `nav.json` 作为抓取目标（与 `<URL>` 二选一） | - |
| `--nav-depth <n>` | 仅使用导航前 N 层菜单（0=全部） | 0 |
| `--max-depth <n>` | 导航菜单最大层级（仅现场提取 `<URL>` 时生效，1=仅一级） | 0 |
| `--nav-selector <CSS>` | 显式指定导航容器，覆盖自动检测（仅现场提取时生效） | 自动 |
| `--all` | 保留站外导航链接（默认仅同注册域，仅现场提取时生效） | false |
| `--workers <n>` | 并发抓取数量 | 5 |
| `--rate-limit <n>` | 每秒最大请求数（0=不限；支持小数如 0.5） | 2 |
| `--output <目录>` | Markdown 输出目录（**本 skill 约定用 `crawl/<入口域名>`**） | output |
| `--crawl-timeout <秒>` | 抓取总超时秒数（0=不限） | 1800 |

单页参数（`--selector` / `--timeout` / `--user-agent` / `--include-source-url` / `--wait-for`）同样适用，作用于每个抓取页。

**落盘规则**：默认输出目录使用 `--output crawl/<域名>` 生成示例：

```
crawl/docs.example.com/         # --output crawl/<域名>；其下由工具按菜单树自动展开
├── Get Started.md              # 叶子菜单 → 直接 .md
├── Guide/
│   ├── index.md                # 既有子菜单又有 URL → index.md
│   ├── Installation.md
│   └── Configuration.md
└── API/                        # 无 URL 的分组标题 → 仅目录
    └── Reference.md
```

**辅助 · 预览/裁剪导航（nav）**：抓取前想先看站点导航长什么样、或导航识别不准要排查时，用 `nav` 单独提取导航菜单——树形输出直接看，或存成 json 供上面"方式 B"消费、人工裁剪后再抓。

```bash
webextract nav https://docs.example.com                              # 树形预览到终端
webextract nav https://docs.example.com --format json -o nav.json    # 存成 json 供 sitemap-crawl --nav 使用
```

---

### 分支 C：整站抓取 · 全站 BFS（crawl，仅当明确要求时）

仅当**导航菜单无法覆盖目标、要把整个域名下所有页面都抓下来、或用户明确要求按链接 BFS 全量抓取**时使用。会从入口出发递归抓取所有站内 `<a href>` 链接（含正文里的"相关阅读"等），按 URL 路径输出。

```bash
webextract crawl https://docs.example.com --max-pages 20 --output crawl/docs.example.com    # 输出到 crawl/<域名>/
webextract crawl https://docs.example.com --depth 1 --output crawl/docs.example.com         # 浅抓，仅入口相邻链接
webextract crawl https://docs.example.com --max-pages 500 --workers 10 --rate-limit 2 --output crawl/docs.example.com   # 大站点
```

| 选项 | 说明 | 默认 |
|------|------|------|
| `<URL>` | 种子 URL（入口页，必填，入口页必达） | - |
| `--depth <n>` | 最大爬取深度（入口页=0） | 2 |
| `--max-pages <n>` | 最大抓取页面数（含入口） | 100 |
| `--workers <n>` | 并发抓取数量（= 最大并发 tab 数） | 5 |
| `--rate-limit <n>` | 每秒最大请求数（0=不限；支持小数如 0.5） | 2 |
| `--output <目录>` | Markdown 输出目录（**本 skill 约定用 `crawl/<入口域名>`**） | output |
| `--allow-subdomains` | 允许抓取同注册域的子域（默认仅同 host） | false |
| `--crawl-timeout <秒>` | 爬取总超时秒数（0=不限） | 1800 |

**落盘规则**：默认输出目录使用 `--output crawl/<域名>`

## 失败排查

- `403 / 429`：站点反爬 → 换 `-user-agent`；批量时降 `--rate-limit`。
- `404`：URL 不存在 → 检查链接。
- 渲染超时/无输出 → 加 `-wait-for` 等关键元素，或调大 `-timeout`。
- **sitemap-crawl 抓少了 / 目录结构与预期不符**：导航自动识别不准 → 用 `--nav-selector` 手动指定导航容器；或先 `nav` 预览/存 json，确认或裁剪后用 `sitemap-crawl --nav` 抓。
- **想抓的页面不在导航菜单里**：sitemap-crawl 只抓导航内页面 → 这类页面改用分支 C（crawl）或单独走分支 A（单页）。
- crawl 报「入口页抓取失败」→ 先用单页 `webextract <url>` 确认种子可抓通。
- `webextract: command not found` 或 `sitemap-crawl` 子命令不存在 → 回到【环境准备】安装/升级到 v0.2.0+，并确认 PATH 含安装目录。

## 绝对不能做的事

1. 不要在用户只给单个 URL、未表达批量意图时擅自整站抓取（默认走分支 A 单页）。
2. **整站抓取默认走分支 B（`sitemap-crawl` 按导航）**；不要在用户未明确要求"全站 BFS / 抓所有链接"时擅自改走分支 C（`crawl`）。
3. 不要用本工具获取登录或付费墙后的内容（不支持鉴权）。
4. crawl 不要忘记用 `--max-pages` / `--depth` 收口；sitemap-crawl 抓大站点用 `--nav-depth` 限定菜单层数。
5. 整站抓取**不要用默认 `output` 目录、也不要随手命名**——统一写到 `crawl/<入口域名>/`。
6. 需要原始 HTML 源码时不要用本工具——改用 `curl`。
