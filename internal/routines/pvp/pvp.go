// Package pvp 提供本服竞技场、仙界竞技场、神魔大战与劫镖等对抗性玩法。
//
// @author Ateng
// @since 2026-10-08
package pvp

import (
	"context"
	"log/slog"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

// NewArenaRoutine 构造本服竞技场挑战任务。
func NewArenaRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"arena",
		"pvp",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行对抗任务: 本服竞技场自动挑战与领奖")
			return nil
		},
	)
}

// NewCelestialArenaRoutine 构造仙界竞技场挑战与下注任务。
func NewCelestialArenaRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"celestial_arena",
		"pvp",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行对抗任务: 仙界竞技场挑战与竞猜下注")
			return nil
		},
	)
}

// NewGodsAndDemonsRoutine 构造神魔大战限时活动任务。
func NewGodsAndDemonsRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"gods_and_demons",
		"pvp",
		scheduler.ScheduleCron,
		scheduler.PriorityHigh,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行对抗任务: 神魔大战阵营自动参战")
			return nil
		},
	)
}

// NewCaravanHijackRoutine 构造阵营劫镖任务。
func NewCaravanHijackRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"caravan_hijack",
		"pvp",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行对抗任务: 阵营商队巡查与自动截击")
			return nil
		},
	)
}

// NewWorldBossRoutine 构造世界Boss击杀争夺任务。
func NewWorldBossRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"world_boss",
		"pvp",
		scheduler.ScheduleCron,
		scheduler.PriorityHigh,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行对抗任务: 世界Boss准时参战与鼓舞输出")
			return nil
		},
	)
}

// NewCrossServerLadderRoutine 构造跨服天梯争霸挑战任务。
func NewCrossServerLadderRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"cross_server_ladder",
		"pvp",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行对抗任务: 跨服天梯排位挑战与段位领奖")
			return nil
		},
	)
}

