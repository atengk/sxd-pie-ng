# 神仙道助手重构工程 (sxd-pie-ng) 领域上下文

本项目是对《神仙道》传统桌面挂机辅助（Pie 助手）的现代化重构，采用 Go 语言实现脱机协议通信、自动化日常调度与内嵌 Web 管理。

## 统一术语表 (Language)

**Platform Account (平台账号)**:
游戏运营方平台的通行证凭据（如心动、疯玩等），用于向平台认证网关换取短期有时效性的登录凭据（Token / Session Key）。
_避免使用_: User, Member, LoginID

**Role Session (角色会话)**:
针对特定区服下某个具体游戏角色的在线运行时连接，维护独立的 TCP Socket 长连接、登录态、心跳保活与状态机。
_避免使用_: Game Connection, Player Instance

**Packet (数据封包)**:
客户端与游戏网关服务器之间通过 TCP 传输的二进制协议数据包，由固定包头（长度、消息号）与载荷（Payload）组成。
_避免使用_: Message Frame, Socket Data

**Protocol Codec (协议编解码器)**:
负责封包二进制序列化、反序列化、大端序解析及透明 zlib 流解压缩的底层协议引擎。
_避免使用_: Parser, Converter

**Activity Routine (日常活动任务)**:
游戏中具体某类玩法的自动化执行单元（如药园种植、帮派吉星高照、取经护送、竞技场等）。
_避免使用_: Job, Action, Automation

**Jitter Scheduler (拟人抖动调度器)**:
负责协调各活动任务执行时机、注入 1~3 秒动态随机延迟与操作抖动的核心防封调度引擎。
_避免使用_: Timer, Cron, Task Manager

**Web Console (内嵌 Web 控制台)**:
通过 Go 内嵌文件系统（`go:embed`）与二进制打包在一起的本地 Web 仪表盘，供用户在浏览器中配置账号与监视日志。
_避免使用_: GUI, Client UI, Dashboard

**Pieb SQLite Engine (游戏字典引擎)**:
基于纯 Go SQLite 驱动挂载原版 `Pieb.db`，提供道具、伙伴、怪物、NPC 等游戏静态元数据的统一只读数据字典仓储。
_避免使用_: Game Database, Local Cache

**QA Bank (题库问答引擎)**:
以内存哈希与模糊容错相似度索引加载并检索 `answers.txt` / `scra.ini` 的答题系统，为仙履奇缘、金榜题名等提供毫秒级自动答题决策。
_避免使用_: Quiz Solver, Auto Answer

**Activity Matrix (全量活动玩法矩阵)**:
对齐原版 `01.ini` 与 `pieb.ini` 的 30+ 细分玩法模块化注册表与状态驱动生命周期骨架。
_避免使用_: Plugins, Feature List

**Dual-Mode Config (双模配置体系)**:
同时支持“一键推荐预设”与“专家精细微调”的分层配置模型，兼顾开箱即用与深度定制。
_避免使用_: Simple/Advanced Settings

**Dual-Track Scheduler (双轨复合调度器)**:
结合定时批处理（Cron，对齐 00:00/06:00 等整点重置）与短周期自驱巡检（Loop），并由统一拟人抖动管道驱动的复合任务执行引擎。
_避免使用_: Cron Job, Simple Loop

**Domain Routine Subpackages (六大玩法领域包)**:
将 30+ 玩法按业务内聚性切分的六大子域：日常签到 (`daily`)、农场取经 (`farming`)、帮派社交 (`social`)、副本推图 (`dungeon`)、竞技对抗 (`pvp`) 与益智小游戏 (`minigame`)。
_避免使用_: Feature Modules, Task Folders

**Three-Stage QA Funnel (三级题库漏斗检索)**:
由文本标点归一化查表、实体关键词倒排交集、Levenshtein 编辑距离相似度兜底构成的三级答题检索算法。
_避免使用_: Text Matcher, Regex Solver

**Semantic Cooling-off (语义智能熔断)**:
当玩法执行捕获到服务器确定性的业务状态（如次数耗尽、体力不足）时，自动将该任务挂起至下一个恢复时间窗口（如次日 00:00 或体力恢复点）的非破坏性降级保护机制。
_避免使用_: Circuit Breaker, Error Retry, Task Disable


