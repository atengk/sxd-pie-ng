# 工程 Agent 协作规范 (Engineering Agent Guidelines)

## Agent skills

### Issue tracker

本项目的任务、需求与缺陷统一通过 GitHub Issues 进行跟踪与管理，使用 `gh` 命令行工具驱动。参见 [docs/agents/issue-tracker.md](./docs/agents/issue-tracker.md)。

### Triage labels

采用标准的五大核心分诊角色标签体系（`needs-triage`、`needs-info`、`ready-for-agent`、`ready-for-human`、`wontfix`）。参见 [docs/agents/triage-labels.md](./docs/agents/triage-labels.md)。

### Domain docs

采用单上下文（single-context）文档布局，通过根目录 `CONTEXT.md` 与 `docs/adr/` 维护领域术语与架构决策。参见 [docs/agents/domain.md](./docs/agents/domain.md)。
