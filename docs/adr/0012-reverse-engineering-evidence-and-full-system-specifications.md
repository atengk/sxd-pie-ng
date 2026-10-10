# 12. 基于原版易语言逆向成果与 pcapng 抓包构建真实协议规范与数据底座

日期: 2026-10-10

## 状态

已通过 (Accepted)

## 上下文 (Context)

在重构工程 `sxd-pie-ng` 的早期阶段，代码库虽然搭建了较为完备的分层架构（`internal/protocol`, `internal/routines`, `internal/scheduler`），但存在一个根本性断点：**协议和业务逻辑并非真实实现**，大多数 Routine 仅为包含 `slog.Info; return nil` 的半成品伪桩，导致系统无法向真实游戏网关正常通信。

为了彻底解决“功能不真实”的研发痛点，我们使用 REA 工具套件与静态字节级探针对原版客户端 `C:\software\疯玩神仙道\pie.exe`（易语言 5.2+ 独立编译、ASPack + 免杀壳双重嵌套保护）进行了全量架构侦察与协议解密推演，同时对本地已捕获的 2.5 MB 真实网络抓包文件 `sxd.pcapng`（5,328 条 TCP 封包）进行了逐字段解构。

## 架构决策 (Decision)

我们决定将逆向工程取得的技术凭证全面工程化，建立正式的开发规范体系，并重塑数据底座：

1. **三维白皮书规范确立**：
   - 编写 `docs/protocol-spec.md`：详细定义大端序 `[4B Length] + [2B Module] + [2B Action] + Payload` 帧格式、类型编解码规范、透明 zlib 解压契约，并直接引入 `sxd.pcapng` 中的真实握手与登录 Hex 报文作为 Golden Cases。
   - 编写 `docs/database-dictionary.md`：全面解析 `Pieb.db`（SQLite 3 数据库）37 张表全量结构，建立合成推导、自动寻路与商店限购的数据关系网。
   - 编写 `docs/routine-spec.md`：依据原版 `pieb.ini` 与 `01.ini` 的 23 个子系统，明确六大领域包的任务状态机、卫语句检查与发包时序。

2. **核心静态数据资产收编**：
   - 将原版客户端的 `Pieb.db`（1.52 MB）、`answers.txt`（仙履奇缘题库）、`monkeyanswer.ini`（1.2 万行灵猴题库）、`scra.ini` 完整同步至仓库 `data/` 目录，由纯 Go SQLite 驱动以只读模式（`mode=ro`）挂载。

3. **逆向资产与 Agent 技能入库治理**：
   - 将 `.agents/` 技能目录直接迁移至项目根目录，供后续 AI 持续维护；
   - 将 4 份核心逆向分析报告归档至 `docs/research/`；
   - 将 8 个专用分析脚本归类沉淀至 `scripts/probes/`。

4. **单点垂直击穿迭代路线**：
   - 确立“优先打通真实登录与角色同步（Module 0, Action 0 & 72），再逐步替换 30 个玩法 Routine”的研发推进原则。

## 结果与影响 (Consequences)

### 正向收益
- 终结了协议“脑补与伪造”的风险，Go 协议实现有了 100% 真实的抓包十六进制数据作为对比基准；
- 数据字典、问答知识库完全就绪，玩法开发无需频繁向服务端请求冗余元数据；
- 建立了完备的文档与规范索引，后续 AI Agent 或研发人员可直接根据文档编写高质量生产代码。

### 潜在代价
- `data/` 目录下增加了约 2MB 的二进制静态字典文件，需确保 Git 配置对其进行大文件正常纳管；
- 存量 Routine 需要在后续里程碑中逐步替换真实发包逻辑。
