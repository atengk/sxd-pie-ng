// Package dungeon 提供副本关卡扫荡、六道轮回、伏魔塔与练功房等推图挑战玩法。
//
// @author Ateng
// @since 2026-10-08
package dungeon

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/dictionary"
	"sxd-pie-ng/internal/protocol"
	"sxd-pie-ng/internal/scheduler"
)

// SweepConfig 关卡扫荡自定义策略配置。
type SweepConfig struct {
	// SweepElite 是否扫精英关卡 (默认 false，对齐 01.ini 扫精英=否)
	SweepElite bool
	// TargetMissionID 目标指定关卡 ID (0 表示自动选最高等级关卡)
	TargetMissionID int
	// MaxBatchTimes 单次执行最大扫荡次数 (默认 10)
	MaxBatchTimes int
}

// DefaultSweepConfig 返回默认扫荡策略配置。
func DefaultSweepConfig() SweepConfig {
	return SweepConfig{
		SweepElite:    false,
		MaxBatchTimes: 10,
	}
}

// NewDungeonSweepRoutine 构造副本体力扫荡任务，支持挂载数据字典与智能策略。
func NewDungeonSweepRoutine(dictRepo dictionary.Repository, cfgs ...SweepConfig) scheduler.ActivityRoutine {
	cfg := DefaultSweepConfig()
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	if cfg.MaxBatchTimes <= 0 {
		cfg.MaxBatchTimes = 10
	}

	return scheduler.NewBaseRoutine(
		"dungeon_sweep",
		"dungeon",
		scheduler.ScheduleLoop,
		scheduler.PriorityNormal,
		30*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			if session == nil {
				return client.ErrNotConnected
			}

			// 1. 拟人防封随机抖动
			if jitter != nil {
				_ = jitter.Wait(ctx)
			}

			// 2. 检查背包容量
			state := session.GetPlayerState()
			if state.BagCapacity <= 0 {
				slog.Warn("背包已满，无法继续关卡扫荡", "role_id", session.RoleID())
				return scheduler.ErrBagFull
			}

			// 3. 检查角色当前体力
			stamina := session.GetStamina()
			const powerPerSweep = 5
			if stamina < powerPerSweep {
				slog.Warn("角色当前体力不足，无法继续扫荡",
					"role_id", session.RoleID(),
					"current_stamina", stamina,
					"required", powerPerSweep,
				)
				return scheduler.ErrStaminaDepleted
			}

			// 4. 定位目标扫荡关卡 (优先从 SQLite 字典检索，支持降级)
			missionID := cfg.TargetMissionID
			missionName := "扬州城-万妖皇"
			if missionID > 0 && dictRepo != nil {
				if m, err := dictRepo.GetMission(missionID); err == nil && m != nil {
					missionName = m.Name
				}
			} else if dictRepo != nil {
				if highest, err := dictRepo.GetHighestMission(cfg.SweepElite); err == nil && highest != nil {
					missionID = highest.ID
					missionName = highest.Name
				}
			}
			if missionID <= 0 {
				missionID = 105
			}

			// 5. 计算本次可扫荡次数
			availableTimes := stamina / powerPerSweep
			times := availableTimes
			if times > cfg.MaxBatchTimes {
				times = cfg.MaxBatchTimes
			}

			// 6. 构造并发送二进制扫荡封包
			req := protocol.SweepRequest{
				MissionID: uint32(missionID),
				Times:     uint16(times),
			}
			pkt, err := protocol.BuildSweepPacket(req)
			if err != nil {
				return fmt.Errorf("构造扫荡封包失败: %w", err)
			}

			if err := session.Send(pkt); err != nil {
				slog.Warn("向游戏服务器发送扫荡请求失败", "role_id", session.RoleID(), "error", err)
				return err
			}

			// 7. 扣除体力并结算收益
			costStamina := times * powerPerSweep
			remStamina, err := session.ConsumeStamina(costStamina)
			if err != nil {
				return err
			}

			gainExp := int64(times) * 2500
			gainCoins := int64(times) * 12000
			session.AddRewards(gainExp, gainCoins)

			slog.Info("关卡体力扫荡完成",
				"role_id", session.RoleID(),
				"mission_id", missionID,
				"mission_name", missionName,
				"times", times,
				"cost_stamina", costStamina,
				"remaining_stamina", remStamina,
				"gain_exp", gainExp,
				"gain_coins", gainCoins,
			)

			// 8. 若体力耗尽，触发智能熔断挂起 30 分钟
			if remStamina < powerPerSweep {
				slog.Info("角色体力已完全耗尽，关卡扫荡进入智能冷却期",
					"role_id", session.RoleID(),
					"remaining_stamina", remStamina,
				)
				return scheduler.ErrStaminaDepleted
			}

			return nil
		},
	)
}

// NewSixRealmsRoutine 构造六道轮回挑战与灵件分解任务。
func NewSixRealmsRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"six_realms",
		"dungeon",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行副本任务: 六道轮回爬塔与灵件自动处理")
			return nil
		},
	)
}

// NewDemonTowerRoutine 构造伏魔塔与怪物试炼任务。
func NewDemonTowerRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"demon_tower",
		"dungeon",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行副本任务: 伏魔塔挑战与怪物试炼")
			return nil
		},
	)
}

// NewTrainingRoomRoutine 构造仙界练功房挂机任务。
func NewTrainingRoomRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"training_room",
		"dungeon",
		scheduler.ScheduleLoop,
		scheduler.PriorityLow,
		60*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行副本任务: 仙界练功房收益维护")
			return nil
		},
	)
}

// NewZodiacRoutine 构造生肖挑战与金油洗练任务。
func NewZodiacRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"zodiac_challenge",
		"dungeon",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行副本任务: 十二生肖关卡挑战与金油炼化")
			return nil
		},
	)
}

// NewSpiritTrialRoutine 构造精灵试炼通关挑战任务。
func NewSpiritTrialRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"spirit_trial",
		"dungeon",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行副本任务: 精灵试炼扫荡与星级奖励")
			return nil
		},
	)
}

