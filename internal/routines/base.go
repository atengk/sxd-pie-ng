// Package routines 提供对齐原版辅助 30+ 自动化玩法的六大业务子域实现矩阵。
//
// @author Ateng
// @since 2026-10-08
package routines

import (
	"context"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

// BaseRoutine 为所有具体玩法提供标准化基础骨架实现。
type BaseRoutine struct {
	id        string
	domain    string
	schedType scheduler.ScheduleType
	priority  scheduler.RoutinePriority
	interval  time.Duration
	fn        func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error
}

// NewBaseRoutine 构造新的标准化 Routine 实例。
func NewBaseRoutine(
	id string,
	domain string,
	schedType scheduler.ScheduleType,
	priority scheduler.RoutinePriority,
	interval time.Duration,
	fn func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error,
) *BaseRoutine {
	return &BaseRoutine{
		id:        id,
		domain:    domain,
		schedType: schedType,
		priority:  priority,
		interval:  interval,
		fn:        fn,
	}
}

// Name 返回玩法唯一标识符。
func (b *BaseRoutine) Name() string {
	return b.id
}

// Domain 返回玩法所属领域。
func (b *BaseRoutine) Domain() string {
	return b.domain
}

// ScheduleType 返回触发类型（定点批处理或周期巡检）。
func (b *BaseRoutine) ScheduleType() scheduler.ScheduleType {
	return b.schedType
}

// Priority 返回调度优先级。
func (b *BaseRoutine) Priority() scheduler.RoutinePriority {
	return b.priority
}

// Interval 返回执行周期。
func (b *BaseRoutine) Interval() time.Duration {
	return b.interval
}

// Execute 执行业务逻辑。
func (b *BaseRoutine) Execute(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
	if b.fn == nil {
		return nil
	}
	return b.fn(ctx, session, jitter)
}
