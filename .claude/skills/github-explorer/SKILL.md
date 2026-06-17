---
name: github-explorer
description: 用 gh 命令智能获取 GitHub 开源项目信息（项目概览、star/活跃度、issues/releases，或按关键词找开源项目）。当用户提供 github.com URL 查看信息、点名要去 github 查某项目、或让帮忙找合适的开源项目时触发；非 github url 或未明确指向 github 时不触发。
---

github-explorer 用官方 `gh` 命令行工具从 GitHub 获取开源项目信息，按用户意图智能选择查询类型（项目概览 / 关键词搜索 / issues / releases 等），并把结果清晰、聚焦地呈现。所有数据通过 `gh` 获取，不 curl、不抓网页。

## 核心原则

1. **gh 优先**：一律用 `gh` 命令取数，不 curl、不抓网页；`gh` 缺失先按【安装与认证】处理好再继续。
2. **智能判意图**：根据用户措辞判断要「单仓库概览 / 关键词搜索 / 看动态（issues/prs/releases）」中的哪一类，再选命令，而非一律 dump 全部字段。
3. **聚焦呈现**：只回应用户问的内容——一句话概括用途 + 关键指标 + 必要链接；不把 README 原文或 API 原始 JSON 整段堆给用户。
4. **严格触发边界**：仅当目标是 GitHub（明确 `github.com` URL、点名「github」「开源项目」，或要找开源替代）才启用；其它网站 URL 或未指向 github 时不要启用。
5. **省 token**：取结构化字段优先用 `--json <fields>` + `--jq`，只拿需要的字段，避免拉回整页大字段。

## 工作流程

### 第一步：判断是否触发

仅以下情形启用本技能：

| 信号 | 示例 |
|------|------|
| 给出 `github.com` 链接，要查看/了解信息 | 「帮我看看 https://github.com/owner/repo 这个项目」 |
| 明确要去 github 查某个项目 | 「去 github 上看看 xxx 项目怎么样」 |
| 让帮忙找合适的开源项目 | 「github 上有没有好用的 xxx 库」 |

不触发：URL 不是 github.com（如 gitlab/gitee/npm/博客站）、或用户没把目标指向 github——交给更合适的工具（普通网页用 webextract）。

### 第二步：检查 gh 是否安装并认证

```bash
command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1 && echo OK || echo NEED_SETUP
```

- 输出 `OK` → 进入第三步。
- 输出 `NEED_SETUP` → 按【安装与认证】引导用户完成后再继续，**不要**在未认证状态下硬跑命令。

### 第三步：识别意图 → 选命令

把目标仓库记为 `$R`，取 `OWNER/REPO` 形式（用户给 `github.com/OWNER/REPO` 链接时直接抽取，`gh` 也接受完整 URL）。

| 用户想要 | 命令 |
|---------|------|
| 这项目是干嘛的（含 README） | `gh repo view $R` |
| 结构化指标（star/fork/语言/协议/活跃度） | `gh repo view $R --json nameWithOwner,description,stargazerCount,forkCount,primaryLanguage,licenseInfo,pushedAt,updatedAt,homepageUrl` |
| 按关键词找开源项目 | `gh search repos "<关键词>" --sort stars --limit 10` |
| 找某语言的库 | `gh search repos "<关键词>" --language python --sort stars --limit 10` |
| 看 issues | `gh issue list -R $R --limit 10` |
| 看 PR | `gh pr list -R $R --limit 10` |
| 看发布版本 | `gh release list -R $R --limit 10` |
| 看目录结构 | `gh api repos/$R/contents --jq '.[].name'` |
| 看贡献者 | `gh api repos/$R/contributors --jq '.[] | {login,contributions}'` |

意图模糊时默认先给「项目概览 + 关键指标」（最常用），再按用户追问逐层深挖。

### ⚠️ `gh repo view --json` 字段速查与避坑（高频踩坑点）

`gh repo view --json` 的字段集**和 GitHub REST API 不一样**，绝不能凭印象臆造，也别直接套 REST 的 snake_case 字段名。

| 想要 | ✅ 正确字段 | ❌ 常见错误写法 |
|------|------------|---------------|
| star 数 | `stargazerCount` | ~~starCount / stargazersCount~~ |
| fork 数 | `forkCount` | ~~forksCount~~ |
| 语言 | `primaryLanguage`(对象) | ~~language~~ |
| 协议 | `licenseInfo`(对象) | ~~license~~ |
| 主题 | `repositoryTopics` | ~~topics~~ |
| 创建/更新时间 | `createdAt` / `updatedAt` / `pushedAt` | ~~created_at / pushed_at~~ |
| issue/PR 数量 | **无此字段** | ~~openIssuesCount / closedIssuesCount~~ |

**推荐的「验证可用字段白名单」**（直接复用，不用记）：
```bash
gh repo view $R --json nameWithOwner,description,stargazerCount,forkCount,watchers,primaryLanguage,licenseInfo,createdAt,pushedAt,updatedAt,homepageUrl,latestRelease,repositoryTopics,visibility,isArchived
```

**要 issue / PR 数量时**（`gh repo view --json` 没有 count 字段，改走 REST API）：
```bash
gh api repos/$R --jq '.open_issues_count'   # ⚠ 该值 = open issues + open PRs 合计
```

**拿不准字段名时**：只从上面白名单挑字段，绝不凭印象加；或故意写一个错字段跑一次——`gh` 会报 `Unknown JSON field` 并**列出全部合法字段**，照着挑即可。

### 第四步：执行并清晰呈现

1. 执行上一步选定的命令。
2. 把原始输出**转译**成用户视角的摘要：一句话说清项目用途 + 3-6 个关键指标（star / 主语言 / 最近更新 / 协议等）+ 仓库链接。
3. 需要时附上简短的「最近活跃度」「适合场景」判断。
4. 信息不够就追问，或继续用对应命令深挖。

## 安装与认证

仅当第二步检测到未安装/未认证时执行。安装/认证多为交互式或需 sudo，建议让用户在自己的终端用 `! <命令>` 执行，完成后回来继续。

```bash
# macOS
brew install gh
# Windows (PowerShell)
winget install --id GitHub.cli
# Linux (Debian/Ubuntu) —— 其它发行版见 https://github.com/cli/cli#installation
(type -p wget >/dev/null || (sudo apt update && sudo apt-get install wget -y)) \
  && sudo mkdir -p -m 755 /etc/apt/keyrings \
  && out=$(mktemp) && wget -nv -O$out https://cli.github.com/packages/githubcli-archive-keyring.gpg \
  && cat $out | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null \
  && sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
  && sudo mkdir -p -m 755 /etc/apt/sources.list.d \
  && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
  && sudo apt update && sudo apt install gh -y
```

装完认证（浏览器交互登录）：

```bash
gh auth login        # 选 GitHub.com → HTTPS → 浏览器登录
gh auth status       # 验证
```

## 绝对不能做的事

1. 不要对非 `github.com` 的 URL 或未明确指向 github 的需求启用本技能。
2. 不要用 `curl`/抓网页绕过 `gh`；`gh` 未安装或未认证时不要硬跑，先引导安装认证。
3. 不要把 README 原文或 API 原始 JSON 整段倒给用户——要转译成聚焦摘要。
4. 不要在用户只要一句话概况时，把 star/issues/contributors/contents 全部查一遍。
5. `gh api` 大字段查询不要忘记用 `--jq` 裁剪，避免 token 浪费。
6. 不要臆造 `gh repo view --json` 的字段名（如 `openIssuesCount`/`starCount`）——它和 REST API 字段集不同。只用【字段速查】小节里的白名单；要 issue/PR 数量走 `gh api repos/$R --jq '.open_issues_count'`（注意该值 = open issues + open PRs 合计）。
