# Issue tracker：GitHub

Issues 和 specs 记录在本仓库的 GitHub Issues 中，所有操作使用 `gh` CLI。

## 约定

- **创建 issue**：`gh issue create --title "..." --body "..."`。多行 body 用 heredoc。
- **读取 issue**：`gh issue view <number> --json number,title,body,labels,comments`。
- **列出 issues**：`gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'`，按需加 `--label` 和 `--state` 过滤。
- **设为父 issue 的子 issue**：`gh issue create --parent <parent> ...`，或事后 `gh issue edit <parent> --add-sub-issue <child>`（`gh` 2.94+）。旧版 `gh`：`gh api --method POST repos/<owner>/<repo>/issues/<parent>/sub_issues -F sub_issue_id=<child-db-id>`（数据库 id，见下方 **Blocking**）。不支持 sub-issues 时，在子 issue body 顶部写 `Part of #<parent>`。
- **Blocking**：GitHub **原生 issue 依赖**，规范且 UI 可见。加边：`gh issue create --blocked-by <n>,<n> ...` 或 `gh issue edit <child> --add-blocked-by <blocker>`（`gh` 2.94+）。旧版 `gh`：`gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`，其中 `<blocker-db-id>` 是 blocker 的数字**数据库 id**（`gh api repos/<owner>/<repo>/issues/<n> --jq .id`，不是 `#number` 或 `node_id`）。依赖不可用时，退回到子 issue body 顶部的 `Blocked by: #<n>, #<n>` 行。所有 blocker 关闭后 ticket 即解除阻塞。
- **统计未关闭 blocker**：`gh api repos/<owner>/<repo>/issues/<n> --jq .issue_dependencies_summary.blocked_by`（仅未关闭；0 表示可开工）。`gh issue view <n> --json blockedBy` 也含已关闭 blocker，需过滤：`--jq '[.blockedBy.nodes[] | select(.state == "OPEN")] | length'`。
- **评论**：`gh issue comment <number> --body "..."`
- **加 / 去标签**：`gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **关闭**：先用 `gh issue comment <number> --body "..."` 发说明，再 `gh issue close <number>`。已关闭的 issue 上 `gh issue close --comment` 会丢掉评论但仍返回退出码 0。

仓库从 `git remote -v` 推断；在 clone 里运行时 `gh` 自动处理。本仓库 `origin` 指向 `wujunwei928/parse-video`。

只向用户自己的仓库提 issue。绝不向 `mattpocock/skills`（技能来源仓库）提 issue，除非用户拥有它。fork 中 `gh` 可能默认指向上游父仓库，所以每个会话第一次 `gh issue create` 前用 `gh repo view --json nameWithOwner` 确认目标，不是用户自己的仓库时先向用户确认。

## Pull requests 作为 triage 面

**PRs as a request surface: no.** _（若本仓库把外部 PR 当功能请求，改为 `yes`；`/triage` 读此开关。）_

设为 `yes` 时，PR 走与 issue 相同的标签与状态，用 `gh pr` 等价命令：

- **读 PR**：`gh pr view <number>`，再 `gh pr view <number> --comments`（非 TTY 下 `--comments` 只打印评论），以及 `gh pr diff <number>`。
- **列出待 triage 的外部 PR**：`gh api --paginate 'repos/{owner}/{repo}/pulls?state=open' --jq '.[] | select(.author_association | IN("OWNER","MEMBER","COLLABORATOR") | not) | {number, title, author: .user.login, author_association, labels: [.labels[].name]}'`。
- **评论 / 标签 / 关闭**：`gh pr comment`、`gh pr edit --add-label`/`--remove-label`、`gh pr close`。

GitHub 的 issue 与 PR 共用号码空间，裸 `#42` 可能是任意一种：先 `gh pr view 42`，失败再 `gh issue view 42`。

## 当技能说 "publish to the issue tracker"

创建一个 GitHub issue。

## 当技能说 "fetch the relevant ticket"

按上方**读取 issue**的方式读。

## Wayfinding 操作

供 `/wayfinder` 使用。**map** 是一个 issue，**child** issues 是 tickets。

- **Map**：带 `wayfinder:map` 标签的 issue，body 为 Notes / Decisions-so-far / Fog。`gh issue create --label wayfinder:map`。
- **Child ticket**：作为 GitHub sub-issue 挂到 map 上（见**设为父 issue 的子 issue**）。sub-issues 未启用时，把 child 加进 map body 的 task list，并在 child body 顶部写 `Part of #<map>`。标签：`wayfinder:<type>`（`research`/`prototype`/`grilling`/`task`）。认领后 assign 给驱动的开发。
- **Blocking**：见约定中的 **Blocking**。
- **Frontier 查询**：列出 map 的未关闭 children（`gh issue list --state open`，限定 map 的 sub-issues / task list），去掉有未关闭 blocker（见**统计未关闭 blocker**或 `Blocked by` 行）或有 assignee 的；按 map 顺序第一个胜出。
- **认领**：`gh issue edit <n> --add-assignee @me`，是本会话第一个写操作。
- **解决**：`gh issue comment <n> --body "<answer>"`，然后 `gh issue close <n>`，再往 map 的 Decisions-so-far 追加上下文指针（gist + 链接）。
