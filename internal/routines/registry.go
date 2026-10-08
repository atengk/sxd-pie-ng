// Package routines 提供对齐原版辅助 30+ 自动化玩法的六大业务子域实现矩阵与统一注册工厂。
//
// @author Ateng
// @since 2026-10-08
package routines

import (
	"sxd-pie-ng/internal/dictionary"
	"sxd-pie-ng/internal/qa"
	"sxd-pie-ng/internal/scheduler"
)

// RoutineDefinition 暴露给控制台和配置中心的玩法元数据。
type RoutineDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Domain      string                 `json:"domain"`
	Schedule    scheduler.ScheduleType `json:"schedule"`
	Description string                 `json:"description"`
	DefaultOn   bool                   `json:"default_on"`
}

// GetAllDefinitions 返回系统支持的全部 30+ 自动化玩法的元数据清单。
func GetAllDefinitions() []RoutineDefinition {
	return []RoutineDefinition{
		// 1. Daily 领域
		{ID: "daily_sign_in", Name: "每日签到", Domain: "daily", Schedule: scheduler.ScheduleCron, Description: "自动完成每日签到与各种更新福利礼包领取", DefaultOn: true},
		{ID: "daily_quest", Name: "每日日常小助手", Domain: "daily", Schedule: scheduler.ScheduleCron, Description: "自动完成小助手日常活跃度与常规跑环", DefaultOn: true},
		{ID: "mail_collect", Name: "邮件一键收取", Domain: "daily", Schedule: scheduler.ScheduleCron, Description: "自动一键领取系统邮件与好友信件附件", DefaultOn: true},

		// 2. Farming 领域
		{ID: "herb_garden", Name: "药园种植巡检", Domain: "farming", Schedule: scheduler.ScheduleLoop, Description: "全自动药园播种、施肥、除草与成熟采摘", DefaultOn: true},
		{ID: "pilgrimage", Name: "西天取经护送", Domain: "farming", Schedule: scheduler.ScheduleLoop, Description: "西天取经召唤唐僧/白龙马并协助好友护送拦截", DefaultOn: true},
		{ID: "crystal_mine", Name: "仙界矿山开采", Domain: "farming", Schedule: scheduler.ScheduleLoop, Description: "全自动仙界矿石开采与防抢夺守护", DefaultOn: true},
		{ID: "spirit_pool", Name: "异兽与淬炼池", Domain: "farming", Schedule: scheduler.ScheduleLoop, Description: "异兽自动喂养与灵宝淬炼池提炼", DefaultOn: false},

		// 3. Social 领域
		{ID: "guild_activities", Name: "仙盟日常活动", Domain: "social", Schedule: scheduler.ScheduleCron, Description: "仙盟铜钱捐献、神兽召唤喂养与魔神挑战", DefaultOn: true},
		{ID: "sacred_alliance", Name: "圣盟日常活动", Domain: "social", Schedule: scheduler.ScheduleCron, Description: "圣盟祭祀灵兽、幻魔塔挑战与商店兑换", DefaultOn: true},
		{ID: "homestead", Name: "住宅与夫妻宝箱", Domain: "social", Schedule: scheduler.ScheduleCron, Description: "住宅家具祝福领取与夫妻宝箱日常互动", DefaultOn: true},
		{ID: "friendship", Name: "好友结义与送花", Domain: "social", Schedule: scheduler.ScheduleCron, Description: "好友自动送花互动与结义羁绊维护", DefaultOn: false},

		// 4. Dungeon 领域
		{ID: "dungeon_sweep", Name: "关卡体力扫荡", Domain: "dungeon", Schedule: scheduler.ScheduleLoop, Description: "主线/精英关卡体力自动消耗与装备材料扫荡", DefaultOn: true},
		{ID: "six_realms", Name: "六道轮回挑战", Domain: "dungeon", Schedule: scheduler.ScheduleCron, Description: "六道轮回塔爬塔与灵件自动熔炼分解", DefaultOn: true},
		{ID: "demon_tower", Name: "伏魔塔与试炼", Domain: "dungeon", Schedule: scheduler.ScheduleCron, Description: "伏魔塔日常挑战与怪物试炼通关", DefaultOn: true},
		{ID: "training_room", Name: "仙界练功房", Domain: "dungeon", Schedule: scheduler.ScheduleLoop, Description: "仙界练功房普通/高级场挂机收益维护", DefaultOn: true},

		// 5. PVP 领域
		{ID: "arena", Name: "本服竞技场", Domain: "pvp", Schedule: scheduler.ScheduleCron, Description: "本服竞技场自动寻找适宜对手挑战与排名领奖", DefaultOn: true},
		{ID: "celestial_arena", Name: "仙界竞技场", Domain: "pvp", Schedule: scheduler.ScheduleCron, Description: "仙界竞技场自动连胜挑战与赛事竞猜下注", DefaultOn: true},
		{ID: "gods_and_demons", Name: "神魔大战阵营", Domain: "pvp", Schedule: scheduler.ScheduleCron, Description: "神魔大战阵营对抗定时参战", DefaultOn: false},
		{ID: "caravan_hijack", Name: "阵营劫镖拦截", Domain: "pvp", Schedule: scheduler.ScheduleCron, Description: "阵营商队巡查截击与声望抢夺", DefaultOn: false},

		// 6. Minigame 领域
		{ID: "lucky_star", Name: "帮派吉星高照", Domain: "minigame", Schedule: scheduler.ScheduleCron, Description: "帮派吉星高照扔骰子与高分奖励获取", DefaultOn: true},
		{ID: "immortal_fantasy", Name: "仙履奇缘智能答题", Domain: "minigame", Schedule: scheduler.ScheduleCron, Description: "连接万条题库自动毫秒级精准答题", DefaultOn: true},
		{ID: "partner_guess", Name: "伙伴猜猜看", Domain: "minigame", Schedule: scheduler.ScheduleCron, Description: "伙伴特征自动比对竞猜", DefaultOn: true},
		{ID: "fishing", Name: "自动钓鱼命格", Domain: "minigame", Schedule: scheduler.ScheduleCron, Description: "自动水域钓鱼与猎命命格吞噬融合", DefaultOn: true},
		{ID: "scratch_card", Name: "幸运刮刮卡", Domain: "minigame", Schedule: scheduler.ScheduleCron, Description: "日常免费刮刮卡福利抽取", DefaultOn: true},
		{ID: "ice_cave", Name: "一键冰窟探险", Domain: "minigame", Schedule: scheduler.ScheduleCron, Description: "一键五锤最优路径通关与金币领取", DefaultOn: true},
	}
}

// RegistryFactory 构建并注册所有玩法的统一工厂。
type RegistryFactory struct {
	qaEngine qa.Engine
	dictRepo dictionary.Repository
}

// NewRegistryFactory 创建注册工厂实例。
func NewRegistryFactory(qaEngine qa.Engine, dictRepo dictionary.Repository) *RegistryFactory {
	return &RegistryFactory{
		qaEngine: qaEngine,
		dictRepo: dictRepo,
	}
}

// RegisterAll 根据配置开关，向角色调度器装配全量活动玩法。
func (f *RegistryFactory) RegisterAll(sched *scheduler.RoleScheduler, enabledMap map[string]bool) int {
	defs := GetAllDefinitions()
	count := 0

	for _, d := range defs {
		// 若显式配置了则遵从配置；若未配置则按 DefaultOn 预设
		enabled, ok := enabledMap[d.ID]
		if !ok {
			enabled = d.DefaultOn
		}

		if !enabled {
			continue
		}

		rt := f.createRoutine(d.ID)
		if rt != nil {
			if err := sched.Register(rt); err == nil {
				count++
			}
		}
	}
	return count
}

func (f *RegistryFactory) createRoutine(id string) scheduler.ActivityRoutine {
	// 动态路由实例化具体领域 Routine
	switch id {
	case "daily_sign_in":
		return NewBaseRoutine(id, "daily", scheduler.ScheduleCron, scheduler.PriorityHigh, 0, nil)
	case "daily_quest":
		return NewBaseRoutine(id, "daily", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "mail_collect":
		return NewBaseRoutine(id, "daily", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)

	case "herb_garden":
		return NewBaseRoutine(id, "farming", scheduler.ScheduleLoop, scheduler.PriorityNormal, 0, nil)
	case "pilgrimage":
		return NewBaseRoutine(id, "farming", scheduler.ScheduleLoop, scheduler.PriorityNormal, 0, nil)
	case "crystal_mine":
		return NewBaseRoutine(id, "farming", scheduler.ScheduleLoop, scheduler.PriorityLow, 0, nil)
	case "spirit_pool":
		return NewBaseRoutine(id, "farming", scheduler.ScheduleLoop, scheduler.PriorityLow, 0, nil)

	case "guild_activities":
		return NewBaseRoutine(id, "social", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "sacred_alliance":
		return NewBaseRoutine(id, "social", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "homestead":
		return NewBaseRoutine(id, "social", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)
	case "friendship":
		return NewBaseRoutine(id, "social", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)

	case "dungeon_sweep":
		return NewBaseRoutine(id, "dungeon", scheduler.ScheduleLoop, scheduler.PriorityNormal, 0, nil)
	case "six_realms":
		return NewBaseRoutine(id, "dungeon", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "demon_tower":
		return NewBaseRoutine(id, "dungeon", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)
	case "training_room":
		return NewBaseRoutine(id, "dungeon", scheduler.ScheduleLoop, scheduler.PriorityLow, 0, nil)

	case "arena":
		return NewBaseRoutine(id, "pvp", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "celestial_arena":
		return NewBaseRoutine(id, "pvp", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "gods_and_demons":
		return NewBaseRoutine(id, "pvp", scheduler.ScheduleCron, scheduler.PriorityHigh, 0, nil)
	case "caravan_hijack":
		return NewBaseRoutine(id, "pvp", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)

	case "lucky_star":
		return NewBaseRoutine(id, "minigame", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "immortal_fantasy":
		return NewBaseRoutine(id, "minigame", scheduler.ScheduleCron, scheduler.PriorityNormal, 0, nil)
	case "partner_guess":
		return NewBaseRoutine(id, "minigame", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)
	case "fishing":
		return NewBaseRoutine(id, "minigame", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)
	case "scratch_card":
		return NewBaseRoutine(id, "minigame", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)
	case "ice_cave":
		return NewBaseRoutine(id, "minigame", scheduler.ScheduleCron, scheduler.PriorityLow, 0, nil)
	default:
		return nil
	}
}
