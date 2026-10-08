// Package minigame 提供吉星高照、仙履奇缘答题、刮刮卡、钓鱼等益智趣味活动。
//
// @author Ateng
// @since 2026-10-08
package minigame

import (
	"context"
	"log/slog"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/qa"
	"sxd-pie-ng/internal/scheduler"
)

// NewLuckyStarRoutine 构造帮派吉星高照扔骰子任务。
func NewLuckyStarRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"lucky_star",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 帮派吉星高照摇骰子")
			return nil
		},
	)
}

// NewImmortalFantasyRoutine 构造仙履奇缘与金榜题名自动问答任务。
func NewImmortalFantasyRoutine(qaEngine qa.Engine) scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"immortal_fantasy",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 仙履奇缘与智能题库问答")
			if qaEngine != nil {
				// 模拟检索示范
				sampleQ := "我国最大的佛像是哪一座？"
				ans, conf, found := qaEngine.Match(sampleQ)
				if found {
					slog.Debug("仙履奇缘答题决策成功", "question", sampleQ, "answer", ans, "confidence", conf)
				}
			}
			return nil
		},
	)
}

// NewPartnerGuessRoutine 构造伙伴猜猜看趣味活动。
func NewPartnerGuessRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"partner_guess",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 伙伴猜猜看自动竞猜")
			return nil
		},
	)
}

// NewFishingRoutine 构造自动钓鱼与命格整理任务。
func NewFishingRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"fishing",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 自动钓鱼与命格吸收融合")
			return nil
		},
	)
}

// NewScratchCardRoutine 构造刮刮卡福利任务。
func NewScratchCardRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"scratch_card",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 幸运刮刮卡福利抽取")
			return nil
		},
	)
}

// NewIceCaveRoutine 构造一键冰窟探险任务。
func NewIceCaveRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"ice_cave",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 一键冰窟最优通关")
			return nil
		},
	)
}

// NewImperialExamRoutine 构造金榜题名会试自动答题任务。
func NewImperialExamRoutine(qaEngine qa.Engine) scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"imperial_exam",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 金榜题名会试科举智能答题")
			if qaEngine != nil {
				sampleQ := "李白字什么？"
				ans, conf, found := qaEngine.Match(sampleQ)
				if found {
					slog.Debug("金榜题名答题决策成功", "question", sampleQ, "answer", ans, "confidence", conf)
				}
			}
			return nil
		},
	)
}

