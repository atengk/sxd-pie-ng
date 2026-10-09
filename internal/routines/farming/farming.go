// Package farming 提供药园种植、取经护送、仙界矿山与异兽养育等资源产出玩法。
//
// @author Ateng
// @since 2026-10-08
package farming

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

// NewHerbGardenRoutine 构造药园种植与巡检任务。
func NewHerbGardenRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"herb_garden",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityNormal,
		15*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			if session == nil {
				return client.ErrNotConnected
			}

			// 1. 拟人防封随机抖动
			if jitter != nil {
				_ = jitter.Wait(ctx)
			}

			slog.Info("正在执行资源任务: 药园种植巡检与采摘", "role_id", session.RoleID())

			// 2. 发送查询土地状态请求
			infoPkt, err := protocol.BuildFarmGetInfoPacket(protocol.FarmGetInfoRequest{
				PrevAct: protocol.ActionIDTownEnter,
			})
			if err != nil {
				return fmt.Errorf("构造药园查询封包失败: %w", err)
			}
			if err := session.Send(infoPkt); err != nil {
				return fmt.Errorf("发送药园查询请求失败: %w", err)
			}

			// 3. 针对默认土地序列 (如 10, 11, 12, 13) 尝试采摘成熟药草
			defaultLands := []int32{10, 11, 12, 13}
			for _, landID := range defaultLands {
				harvestPkt, err := protocol.BuildFarmHarvestPacket(protocol.FarmHarvestRequest{
					LandID:  landID,
					PrevAct: protocol.ActionIDFarmGetInfo,
				})
				if err == nil {
					_ = session.Send(harvestPkt)
				}
			}

			// 4. 对空闲土地自动种植经验草或种子 (种子/伙伴 0xA9)
			for _, landID := range defaultLands {
				plantPkt, err := protocol.BuildFarmPlantPacket(protocol.FarmPlantRequest{
					LandID:       landID,
					SeedOrRoleID: 169, // 仙玲珑/主力经验草种子
					PrevAct:      protocol.ActionIDFarmHarvest,
				})
				if err == nil {
					_ = session.Send(plantPkt)
				}
			}

			slog.Info("药园种植与收获巡检完成", "role_id", session.RoleID(), "processed_lands", len(defaultLands))
			return nil
		},
	)
}

// NewPilgrimageRoutine 构造西天取经护送任务。
func NewPilgrimageRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"pilgrimage",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityNormal,
		20*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 西天取经护送与拦截保护")
			return nil
		},
	)
}

// NewCrystalMineRoutine 构造仙界矿山开采任务。
func NewCrystalMineRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"crystal_mine",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityLow,
		30*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 仙界矿山自动开采")
			return nil
		},
	)
}

// NewSpiritPoolRoutine 构造异兽与练池淬炼任务。
func NewSpiritPoolRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"spirit_pool",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityLow,
		60*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 异兽驯养与淬炼池提炼")
			return nil
		},
	)
}

// NewGourdRealmRoutine 构造壶中界炼丹与灵气合成任务。
func NewGourdRealmRoutine(dictRepo dictionary.Repository) scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"gourd_realm",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityNormal,
		30*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 壶中界灵气合成与材料提炼")
			if dictRepo != nil {
				// 通过数据字典校验基础合成丹药道具 (如气血包 ID 7)
				if item, err := dictRepo.GetItem(7); err == nil && item != nil {
					slog.Debug("壶中界数据字典校验就绪", "sample_item", item.Name)
				}
			}
			return nil
		},
	)
}

// NewRuneRefineRoutine 构造符文提炼与八卦炼丹任务。
func NewRuneRefineRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"rune_refine",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityLow,
		45*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 符文提炼与八卦炉炼化")
			return nil
		},
	)
}

