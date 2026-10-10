# 疯玩神仙道 (pie.exe) 逆向工程与架构侦察技术资产主索引

> [!NOTE]
> 本文档为针对 `C:\software\疯玩神仙道\pie.exe` 及其配套辅助套件的全面逆向工程与架构侦察任务的**主索引归档报告**。任务历经 PE 架构侦察、数据模型剖析、网络协议还原三大阶段，已完整建立起套件运行机理的全维知识图谱。

---

## 一、 技术交付物资产清单

本次侦察输出了一套高内聚、结构化的逆向技术文档与分析探针，均已规范落盘存储在会话受管目录中：

```
📁 逆向工程成果归档
├── 📄 pie_exe_architectural_reconnaissance.md         [阶段一: PE 基础架构与平台判定]
├── 📄 pie_suite_deep_reverse_engineering_guide.md       [阶段一深化: 双重壳脱壳与符号恢复手册]
├── 📄 pie_database_and_automation_rules_analysis.md    [阶段二: 数据库与自动化规则引擎解析]
├── 📄 pie_protocol_and_network_communication_analysis.md [阶段三: 网络通信链路与协议还原报告]
└── 📁 scratch/                                          [分析探针与解析脚本套件]
    ├── pe_inspect.py                                   (纯 Python 零依赖 PE 解析器)
    ├── dump_db_schema.py                               (Pieb.db 37张表结构与抽样导出器)
    ├── disasm_outer.py / disasm_stage2.py              (基于 Capstone 的外层免杀壳反汇编器)
    ├── inspect_login_tool.py                           (MFC 登录器 WinInet/URL 提取器)
    ├── dump_pieb_ini.py                                (pieb.ini 23 个子模块策略解析器)
    └── scan_protocol_logs.py                           (多服 TCP Socket 连接日志审计器)
```

---

## 二、 系统全维知识图谱

```mermaid
graph TD
    subgraph 1. 宿主与程序架构
        PE["pie.exe (PE32, x86 32-bit GUI)"]
        NotDotNet["确定性判定: 纯原生二进制 (无 CLR/CIL)"]
        EPL["开发技术栈: 易语言 5.2+ 独立编译 (E.App)"]
        Identity["软件身份: 山寨兔 (神仙道页游全自动辅助挂机)"]
        PE --> NotDotNet
        PE --> EPL
        PE --> Identity
    end

    subgraph 2. 防护与加壳体系
        Outer["外层壳: 免杀加密壳 (cnajrruv + qvngnnjm)"]
        Inner["内层壳: ASPack 2.x (aPLib 压缩引擎)"]
        AntiDbg["反调试机制: int 3 探针 + 动态 Delta 寻址"]
        OEP["真实入口 (OEP): 易语言事件引擎核心"]
        Outer --> AntiDbg
        AntiDbg --> Inner
        Inner -->|ESP 定律脱壳| OEP
    end

    subgraph 3. 本地数据与调度引擎
        DB[("Pieb.db (SQLite 3, 37张核心表)")<br/>物品字典(8178) / 关卡(1966) / 合成(2265) / 坐标(424)]
        IniRule["pieb.ini / 01.ini<br/>8 个每日时间点轮询 + 23 个玩法子系统"]
        AI["决策题库<br/>answers.txt (收益权重比对) + monkeyanswer.ini (1.2万条问答)"]
        DB <--> IniRule
        AI <--> IniRule
    end

    subgraph 4. 通信与协议体系
        HTTPAuth["阶段一: HTTP Web 鉴权 (Sxd Assistant Login Tool.exe)"]
        MultiSocket["阶段二: 4路并发 TCP Socket (主服 / 聊天 / 仙界 / 圣域)"]
        ProtoFrame["二进制帧头: Length(4B) + Mod(2B) + Act(2B) + Payload (大端序)"]
        Compression["流压缩: 大数据段自动 zlib Deflate 编解码 (zlib.dll)"]
        HTTPAuth --> MultiSocket
        MultiSocket --> ProtoFrame
        ProtoFrame --> Compression
    end

    OEP -.-> DB
    OEP -.-> IniRule
    OEP -.-> MultiSocket
```

---

## 三、 四大阶段核心结论提炼

### 1. PE 基础架构与技术栈判定
- **指令架构**：`IMAGE_FILE_MACHINE_I386` (32-bit x86)，基址 `0x00400000`，入口点 `RVA 0x0084D000` 落在节区 `cnajrruv`。
- **运行平台**：REA `inspect_managed_artifact` 确认 `classification.status: "not-managed"`，无 CLI 数据目录，**100% 纯原生（Native x86）二进制程序**。
- **开发栈与身份**：内嵌清单包含 `<assemblyIdentity name="E.App" version="5.2.0.0"/>`，版本资源标明 `山寨兔 326.0.0.0`，为国内经典的易语言编写的神仙道挂机客户端。

### 2. 双重嵌套加壳机制
- **外层免杀壳**：随机 8 字符节区名（`qvngnnjm` 高熵 7.95、`cnajrruv`）。入口通过 `call $+6; int 3; pop eax` 获取相对偏移，检测断点后用密钥 `0x5E25B697` 与 `0x6BBDA236` 解密 aPLib 解压引擎，展开进入内层。
- **内层 ASPack**：保留 `.aspack`、`.adata`、`.idata`。静态 IAT 伪装剥离，仅留 `lstrcpy` 与 `InitCommonControls`。

### 3. 数据驱动挂机模型
- **`Pieb.db`**：本地存储 37 张游戏全量表。通过 `factureReel` 关联 `Missions` 实现装备卷轴缺料推导与定向副本扫荡；通过 `NPCposition` 驱动二维坐标精确寻路。
- **`pieb.ini`**：时间轮调度引擎在每日 `00:00, 06:02, 08:02, 12:00, 16:05, 18:01, 20:01, 22:01` 避峰轮询，任务结束触发自动销毁。
- **决策题库**：`answers.txt` 实现收益多维度加权比对（`体力 > 声望 > 阅历 > 铜钱`）；`monkeyanswer.ini` 拥有 **12,224 行** 文史常识题库支持自动答题。

### 4. 网络通信链路与脱机协议
- **鉴权阶段**：无壳 MFC 登录器通过 `WinInet` 提取 Session Cookie（`user`, `_time`, `_hash`），写入 `user.ini`。
- **Socket 阶段**：主程序向网关拉取真实配置，并发维持 **主游戏服、独立聊天室、跨服仙界、跨服圣域** 4 条 TCP 长连接。
- **协议帧格式**：大端序 `Length (4B) + Module_ID (2B) + Action_ID (2B) + Payload`，超长数据段经由 `zlib.dll` 自动压缩解压。

---

## 四、 成果直达索引

1. [阶段一报告：PE 文件基本架构侦察与平台分类判定](./pie_exe_architectural_reconnaissance.md)
2. [阶段一深化：双重壳脱壳实战手册与易语言逆向指南](./pie_suite_deep_reverse_engineering_guide.md)
3. [阶段二报告：Pieb.db 数据模型与自动化规则引擎深度解析](./pie_database_and_automation_rules_analysis.md)
4. [阶段三报告：网络通信链路与二进制协议还原报告](./pie_protocol_and_network_communication_analysis.md)
