# 任务分诊标签规范 (Triage Labels)

各项工程技能使用 5 种标准分诊角色。下表定义了这些角色与本仓库 GitHub Issues 实际使用的标签之间的映射关系：

| 技能标准角色 | 本仓库实际标签 | 业务与调度含义 |
| :--- | :--- | :--- |
| `needs-triage` | `needs-triage` | 待维护者或 Agent 评估分诊的新 Issue |
| `needs-info` | `needs-info` | 信息不全，等待提交者补充更多上下文或复现步骤 |
| `ready-for-agent` | `ready-for-agent` | 规格与技术要求明确，可交由离线/自主 Agent 独立实现 |
| `ready-for-human` | `ready-for-human` | 复杂度较高或需人工决策，须由人类工程师亲自实现 |
| `wontfix` | `wontfix` | 经评估不予处理或关闭的任务 |

当技能提到某一分诊角色时（例如“应用已就绪标签”），自动使用表中对应的具体标签字符串。
