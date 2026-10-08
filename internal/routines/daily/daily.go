// Package daily 提供每日签到、日常任务与邮件福利自动化逻辑。
//
// @author Ateng
// @since 2026-10-08
package daily

import (
	"context"
	"log/slog"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

// NewSignInRoutine 构造每日签到与更新福利任务。
func NewSignInRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"daily_sign_in",
		"daily",
		scheduler.ScheduleCron,
		scheduler.PriorityHigh,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行日常任务: 每日签到与在线福利领取")
			return nil
		},
	)
}

// NewDailyQuestRoutine 构造每日日常活跃度与小助手任务。
func NewDailyQuestRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"daily_quest",
		"daily",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行日常任务: 每日任务与小助手日常")
			return nil
		},
	)
}

// NewMailCollectRoutine 构造自动领取邮件附件任务。
func NewMailCollectRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"mail_collect",
		"daily",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行日常任务: 邮件附件一键收取")
			return nil
		},
	)
}

// NewOnlineGiftRoutine 构造在线福利与挂机礼包领取任务。
func NewOnlineGiftRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"online_gift",
		"daily",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行日常任务: 在线福利与成长礼包领取")
			return nil
		},
	)
}

// NewVIPWelfareRoutine 构造VIP福利与每日俸禄领用任务。
func NewVIPWelfareRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"vip_welfare",
		"daily",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行日常任务: VIP等级特权礼包与俸禄领取")
			return nil
		},
	)
}

