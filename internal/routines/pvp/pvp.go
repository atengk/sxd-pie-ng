// Package pvp 提供本服竞技场、仙界竞技场、神魔大战与劫镖等对抗性玩法。
//
// @author Ateng
// @since 2026-10-08
package pvp

import (
	"context"
	"fmt"
	"log/slog"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/protocol"
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
			if session == nil {
				return client.ErrNotConnected
			}

			// 1. 拟人防封随机抖动
			if jitter != nil {
				_ = jitter.Wait(ctx)
			}

			slog.Info("正在执行对抗任务: 本服竞技场自动挑战与领奖", "role_id", session.RoleID())

			// 2. 发送查询剩余挑战次数
			timesPkt, err := protocol.BuildArenaGetTimesPacket(protocol.ArenaGetTimesRequest{
				PrevAct: protocol.ActionIDTownEnter,
			})
			if err != nil {
				return fmt.Errorf("构造竞技场次数查询封包失败: %w", err)
			}
			if err := session.Send(timesPkt); err != nil {
				return fmt.Errorf("发送竞技场次数查询失败: %w", err)
			}

			// 3. 发送查询可挑战对手列表
			opponentsPkt, err := protocol.BuildArenaGetOpponentsPacket(protocol.ArenaGetOpponentsRequest{
				PrevAct: protocol.ActionIDArenaGetTimes,
			})
			if err != nil {
				return fmt.Errorf("构造竞技场对手查询封包失败: %w", err)
			}
			if err := session.Send(opponentsPkt); err != nil {
				return fmt.Errorf("发送竞技场对手查询失败: %w", err)
			}

			// 4. 发起挑战 (默认挑战可挑战位 TargetRank=5，符合抓包行为)
			challengePkt, err := protocol.BuildArenaChallengePacket(protocol.ArenaChallengeRequest{
				TargetRank: 5,
				PrevAct:    protocol.ActionIDArenaGetOpponents,
			})
			if err != nil {
				return fmt.Errorf("构造竞技场挑战封包失败: %w", err)
			}
			if err := session.Send(challengePkt); err != nil {
				return fmt.Errorf("发送竞技场挑战失败: %w", err)
			}

			slog.Info("本服竞技场挑战完成", "role_id", session.RoleID(), "target_rank", 5)
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

