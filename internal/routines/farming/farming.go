// Package farming 提供药园种植、取经护送、仙界矿山与异兽养育等资源产出玩法。
//
// @author Ateng
// @since 2026-10-08
package farming

import (
	"context"
	"log/slog"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/dictionary"
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
			slog.Info("正在执行资源任务: 药园种植巡检与采摘")
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

