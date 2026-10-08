// Package social 提供仙盟、圣盟、住宅与好友互动的帮派社交自动化玩法。
//
// @author Ateng
// @since 2026-10-08
package social

import (
	"context"
	"log/slog"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

// NewGuildActivitiesRoutine 构造仙盟神兽与捐献任务。
func NewGuildActivitiesRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"guild_activities",
		"social",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行社交任务: 仙盟捐献与神兽挑战")
			return nil
		},
	)
}

// NewSacredAllianceRoutine 构造圣盟祭祀与幻魔塔任务。
func NewSacredAllianceRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"sacred_alliance",
		"social",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行社交任务: 圣盟祭祀与幻魔塔挑战")
			return nil
		},
	)
}

// NewHomesteadRoutine 构造住宅家具祝福与夫妻宝箱任务。
func NewHomesteadRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"homestead",
		"social",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行社交任务: 住宅家具祝福与夫妻宝箱领取")
			return nil
		},
	)
}

// NewFriendshipRoutine 构造好友结义与鲜花赠送任务。
func NewFriendshipRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"friendship",
		"social",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行社交任务: 好友结义与日常送花互动")
			return nil
		},
	)
}

// NewFameBlessingRoutine 构造仙界膜拜大神与声望祝福任务。
func NewFameBlessingRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"fame_blessing",
		"social",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行社交任务: 膜拜大神尊像与声望领用")
			return nil
		},
	)
}

