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
