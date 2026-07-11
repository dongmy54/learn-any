# `gh` JSON 字段速查与避坑（github-explorer 参考资料）

本文件是 SKILL.md【字段速查】的详细参考，仅在需要确认字段名、排查 `Unknown JSON field` 报错时查阅。

## 一、`gh search repos --json` 与 `gh repo view --json` 字段对照

两者字段集**完全不同**，绝不能混用。star/fork 差一个 s，混淆直接报错：

| 字段 | `gh search repos --json` | `gh repo view --json` |
|------|--------------------------|----------------------|
| star | `stargazersCount`（带 s） | `stargazerCount`（不带 s） |
| fork | `forksCount`（带 s） | `forkCount`（不带 s） |
| issue 数 | `openIssuesCount`（有） | 无此字段，走 `gh api` |
| 时间 | `createdAt` / `updatedAt`（无 `pushedAt`） | `createdAt` / `updatedAt` / `pushedAt` |

## 二、`gh repo view --json` 常见错误写法

| 想要 | ✅ 正确 | ❌ 常见错误 |
|------|--------|------------|
| star 数 | `stargazerCount` | `starCount` / `stargazersCount`（后者属 search） |
| fork 数 | `forkCount` | `forks` / `forksCount`（后者属 search） |
| 语言 | `primaryLanguage`（对象） | `language` |
| 协议 | `licenseInfo`（对象） | `license` |
| 主题 | `repositoryTopics` | `topics` |
| 创建/更新时间 | `createdAt` / `updatedAt` / `pushedAt` | `created_at` / `pushed_at`（REST 风格） |
| issue/PR 数量 | 无此字段 | `openIssuesCount` / `closedIssuesCount` |

## 三、验证可用字段白名单（直接复用）

```bash
gh repo view $R --json nameWithOwner,description,stargazerCount,forkCount,watchers,primaryLanguage,licenseInfo,createdAt,pushedAt,updatedAt,homepageUrl,latestRelease,repositoryTopics,visibility,isArchived
```

## 四、要 issue / PR 数量时

`gh repo view --json` 没有 count 字段，改走 REST API：

```bash
gh api repos/$R --jq '.open_issues_count'   # ⚠ 该值 = open issues + open PRs 合计
```

## 五、拿不准字段名时

只从上面白名单挑字段，绝不凭印象加。或故意写一个错字段跑一次——`gh` 会报 `Unknown JSON field` 并**列出全部合法字段**，照着挑即可。
