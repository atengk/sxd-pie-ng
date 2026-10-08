// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-08
package protocol

const (
	// 核心业务模块编号 (Module)
	ModulePlayer          uint8 = 0  // 角色主模块 (Mod_Player_Base)
	ModuleTown            uint8 = 1  // 城镇场景模块 (Mod_Town_Base)
	ModuleMissionPractice uint8 = 25 // 副本挂机扫荡模块 (Mod_MissionPractice_Base)
	ModuleStLogin         uint8 = 94 // 跨服/网页登录模块 (Mod_StLogin_Base)
)

const (
	// ActionIDStLogin 跨服登录握手协议号 (Module 94, Action 0 -> 0x005E)
	ActionIDStLogin uint16 = 0x005E

	// ActionIDPlayerLogin 角色主服登录协议号 (Module 0, Action 0 -> 0x0000)
	ActionIDPlayerLogin uint16 = 0x0000
	// ActionIDPlayerGetInfo 获取角色主属性协议号 (Module 0, Action 2 -> 0x0200)
	ActionIDPlayerGetInfo uint16 = 0x0200
	// ActionIDPlayerUpdateData 服务器主动下发属性变更协议号 (Module 0, Action 3 -> 0x0300)
	ActionIDPlayerUpdateData uint16 = 0x0300

	// ActionIDTownEnter 进入城镇协议号 (Module 1, Action 0 -> 0x0001)
	ActionIDTownEnter uint16 = 0x0001
	// ActionIDTownLeave 离开城镇协议号 (Module 1, Action 1 -> 0x0101)
	ActionIDTownLeave uint16 = 0x0101

	// ActionIDPracticeGetInfo 查询副本扫荡信息 (Module 25, Action 0 -> 0x0019)
	ActionIDPracticeGetInfo uint16 = 0x0019
	// ActionIDPracticeStart 开始副本扫荡 (Module 25, Action 1 -> 0x0119)
	ActionIDPracticeStart uint16 = 0x0119
	// ActionIDPracticeCancel 取消副本扫荡 (Module 25, Action 2 -> 0x0219)
	ActionIDPracticeCancel uint16 = 0x0219
	// ActionIDPracticeQuickly 立即加速秒完成扫荡 (Module 25, Action 3 -> 0x0319)
	ActionIDPracticeQuickly uint16 = 0x0319
	// ActionIDPracticeNotify 扫荡轮次战斗结算服务端主动通知 (Module 25, Action 4 -> 0x0419)
	ActionIDPracticeNotify uint16 = 0x0419
)

// 副本扫荡状态结果码
const (
	PracticeResultSuccess        uint8 = 0 // 扫荡成功
	PracticeResultNotEnoughPower uint8 = 1 // 体力不足
	PracticeResultBagFull        uint8 = 2 // 背包已满
	PracticeResultInPractice     uint8 = 3 // 正在扫荡中
	PracticeResultNotEnoughIngot uint8 = 4 // 元宝不足
)

// 角色属性更新键 (Mod_Player_Base 属性枚举)
const (
	PlayerPropLevel         uint8 = 10 // 等级
	PlayerPropIngot         uint8 = 11 // 元宝
	PlayerPropCoins         uint8 = 12 // 铜钱
	PlayerPropHealth        uint8 = 13 // 当前生命
	PlayerPropMaxHealth     uint8 = 14 // 最大生命
	PlayerPropPower         uint8 = 15 // 当前体力
	PlayerPropMaxPower      uint8 = 16 // 最大体力
	PlayerPropExperience    uint8 = 17 // 当前经验
	PlayerPropMaxExperience uint8 = 18 // 最大经验
	PlayerPropPackEmptyNum  uint8 = 25 // 背包剩余空闲格子数
	PlayerPropVIPLevel      uint8 = 31 // VIP等级
)
