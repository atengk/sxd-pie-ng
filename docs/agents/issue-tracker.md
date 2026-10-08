# 任务与缺陷跟踪器：GitHub Issues

本仓库的所有任务（Issues）、技术规范（Specs）与待办卡片均统一托管于 GitHub Issues。所有自动化技能统一使用 `gh` CLI 命令行工具进行操作。

## 操作约定与常用命令

- **创建任务**：`gh issue create --title "..." --body "..."`。多行描述建议通过 heredoc 输入。
- **查看任务**：`gh issue view <number> --comments`，可通过 `jq` 解析并提取关联标签与评论。
- **查询列表**：`gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'`，可搭配 `--label` 与 `--state` 参数进行过滤。
- **追加评论**：`gh issue comment <number> --body "..."`
- **调整标签**：`gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **关闭任务**：`gh issue close <number> --comment "..."`

仓库目标默认根据当前本地克隆的 `git remote -v` 自动解析，`gh` 在仓库工作区内执行时会自动推导。

## Pull Request 作为分诊队列

**将 PR 纳入分诊请求队列：否 (no)** _（若本项目需要将外部 PR 视同为功能诉求纳入分诊，可修改此项为 `yes`；`/triage` 技能会读取此标志位）_

当配置为 `yes` 时，PR 将沿用与 Issue 相同的标签体系与状态生命周期，并使用对应的 `gh pr` 命令进行调度：
- **查看 PR**：`gh pr view <number> --comments`，并通过 `gh pr diff <number>` 查看改动差异。
- **分诊外部 PR**：`gh pr list --state open --json number,title,body,labels,author,authorAssociation,comments`，仅保留 `authorAssociation` 为 `CONTRIBUTOR`、`FIRST_TIME_CONTRIBUTOR` 或 `NONE` 的外部提交（过滤仓库 `OWNER`/`MEMBER`/`COLLABORATOR`）。
- **评论 / 标签 / 关闭**：分别使用 `gh pr comment`、`gh pr edit --add-label`/`--remove-label`、`gh pr close`。

由于 GitHub 的 Issue 与 PR 共享同一数字编号空间，单独的 `#42` 可能代表 Issue 或 PR——优先使用 `gh pr view 42`，若非 PR 则降级回退至 `gh issue view 42`。

## 技能动作映射

- **当技能提示“发布至任务跟踪器 (publish to the issue tracker)”时**：
  创建一个新的 GitHub Issue。
- **当技能提示“获取对应任务卡片 (fetch the relevant ticket)”时**：
  执行 `gh issue view <number> --comments`。

## 寻路器调度操作 (Wayfinding Operations)

由 `/wayfinder` 技能使用。**导航图 (Map)** 为包含 `wayfinder:map` 标签的主 Issue，挂载的各**子任务 (Child tickets)** 作为具体执行卡片。

- **导航图 (Map)**：主任务卡片，标签为 `wayfinder:map`，正文包含当前笔记、既成决策记录与未知盲区。创建命令：`gh issue create --label wayfinder:map`。
- **子任务卡片 (Child ticket)**：通过 GitHub Sub-issues API 挂载至主导航图的子任务。若未开启 Sub-issues 功能，则在主任务正文的任务列表中引用，并在子任务正文顶部标注 `Part of #<map>`。标签格式为 `wayfinder:<type>`（如 `research`、`prototype`、`grilling`、`task`）。领取任务后自动指派给具体开发人员或 Agent。
- **阻塞依赖关系 (Blocking)**：使用 GitHub 原生 Issue 依赖功能进行前置依赖绑定（可通过 `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>` 添加）。若不可用，降级为在正文顶部标注 `Blocked by: #<n>, #<n>`。当所有前置阻塞任务均关闭后，当前卡片解锁。
- **前沿就绪查询 (Frontier query)**：查询导航图中所有未阻塞、未被领取的开放状态子任务。
- **任务认领 (Claim)**：执行 `gh issue edit <n> --add-assignee @me`。
- **任务结项 (Resolve)**：执行 `gh issue comment <n> --body "<answer>"`，然后 `gh issue close <n>`，并将决策链接同步追加至导航图。
