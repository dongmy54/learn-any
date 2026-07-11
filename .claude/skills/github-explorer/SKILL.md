---
name: github-explorer
description: 用 gh 命令智能获取 GitHub 开源项目信息（项目概览、star/活跃度、issues/releases，或按关键词找开源项目）。当用户提供 github.com URL 查看信息、点名要去 github 查某项目、或让帮忙找合适的开源项目时触发；非 github url 或未明确指向 github 时不触发。
---

用官方 `gh` 命令行工具从 GitHub 获取开源项目信息，按用户意图智能选择查询类型（项目概览 / 关键词搜索 / issues / releases 等），把结果清晰聚焦地呈现。**所有数据通过 `gh` 获取，绝不 curl、不抓网页。**

## 核心原则

1. **gh 优先**：一律用 `gh` 取数；`gh` 缺失或未认证先按【安装与认证】处理，不在未认证状态硬跑。
2. **判意图再选命令**：先判断用户要「单仓库概览 / 关键词搜索 / 看动态（issues/prs/releases）」中的哪一类，再选命令，不一律 dump 全部字段。
3. **聚焦呈现**：一句话用途 + 关键指标 + 必要链接；不把 README 原文或原始 JSON 整段堆给用户。
4. **省 token**：结构化字段优先 `--json <fields>` + `--jq`，只取需要的字段。
5. **严格触发边界**：仅当目标明确指向 GitHub（`github.com` URL、点名「github」「开源项目」、找开源替代）才启用；否则交给更合适的工具（普通网页用 webextract）。

## 工作流程

### 第一步：判断是否触发

仅以下情形启用：

| 信号 | 示例 |
|------|------|
| 给出 `github.com` 链接要查看/了解 | 「帮我看看 https://github.com/owner/repo」 |
| 明确要去 github 查某项目 | 「去 github 看看 xxx 项目怎么样」 |
| 让帮忙找合适的开源项目 | 「github 上有没有好用的 xxx 库」 |

不触发：URL 不是 github.com（gitlab/gitee/npm/博客站），或用户未把目标指向 github。

### 第二步：检查 gh 安装与认证

```bash
command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1 && echo OK || echo NEED_SETUP
```

- `OK` → 进入第三步。
- `NEED_SETUP` → 按【安装与认证】引导用户完成后再继续。

### 第三步：识别意图 → 选命令

把目标仓库记为 `$R`（`OWNER/REPO` 形式；用户给完整 URL 时 `gh` 也接受）。

| 用户想要 | 命令 |
|---------|------|
| 项目用途（含 README） | `gh repo view $R` |
| 结构化指标（star/fork/语言/协议/活跃度） | 见【字段速查】白名单 |
| 按关键词找开源项目 | `gh search repos "<关键词>" --sort stars --limit 10` |
| 找某语言的库 | `gh search repos "<关键词>" --language python --sort stars --limit 10` |
| 看 issues / PR / releases | `gh issue list -R $R --limit 10`（PR 换 `pr`，版本换 `release list`） |
| 看目录结构 / 贡献者 | `gh api repos/$R/contents --jq '.[].name'`（贡献者换 `contributors`，jq 取 `{login,contributions}`） |

> 🔍 **关键词策略**：找「开发框架/库/SDK」时，纯业务词（如 `ai agent`）按 star 排序易混入"**使用**该技术的应用产品"而非"**开发**它的框架"。应：①关键词带 `framework`/`sdk`/`library`；②或按已知名库（如 `eino`/`langchain`）补搜；③多组关键词并行搜再合并去重。

意图模糊时默认先给「项目概览 + 关键指标」，再按追问逐层深挖。

### 第四步：执行并清晰呈现

1. 执行选定命令。
2. 把原始输出**转译**成用户视角摘要：一句话用途 + 3-6 个关键指标（star / 主语言 / 最近更新 / 协议）+ 仓库链接。
3. 需要时附「最近活跃度」「适合场景」判断；信息不够就追问或继续深挖。

## 字段速查（`gh repo view --json`）

`gh repo view --json` 的字段集**与 GitHub REST API 不同**，绝不臆造字段名。**直接复用下面白名单**，不要自己加字段：

```bash
gh repo view $R --json nameWithOwner,description,stargazerCount,forkCount,watchers,primaryLanguage,licenseInfo,createdAt,pushedAt,updatedAt,homepageUrl,latestRelease,repositoryTopics,visibility,isArchived
```

**最关键避坑**：`gh search repos --json` 与 `gh repo view --json` 字段集不同——star/fork 差一个 s（`search` 用 `stargazersCount`/`forksCount` 带 s；`repo view` 用 `stargazerCount`/`forkCount` 不带 s）。**要 issue/PR 数量**：`repo view` 没有 count 字段，走 `gh api repos/$R --jq '.open_issues_count'`（注意该值 = open issues + open PRs 合计）。

完整字段对照、常见错误写法、拿不准时的验证技巧见 [field-reference.md](./examples/field-reference.md)。

## 安装与认证

仅当第二步检测到 `NEED_SETUP` 时执行。安装/认证多为交互式或需 sudo，**让用户在自己的终端用 `! <命令>` 执行**，完成后回来继续。安装命令（macOS / Windows / Linux）与认证步骤见 [setup.md](./examples/setup.md)。

## 绝对不能做的事

1. 不要对非 `github.com` 的 URL 或未明确指向 github 的需求启用本技能。
2. 不要用 `curl`/抓网页绕过 `gh`；`gh` 未装/未认证时不要硬跑，先引导安装认证。
3. 不要把 README 原文或原始 JSON 整段倒给用户——要转译成聚焦摘要。
4. 不要在用户只要一句话概况时把 star/issues/contributors/contents 全查一遍。
5. 不要臆造 `gh repo view --json` 字段名——只用【字段速查】白名单；要 issue/PR 数量走 `gh api`。
