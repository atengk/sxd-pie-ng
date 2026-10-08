// Package scheduler 提供拟人随机抖动任务调度引擎，协调各类日常活动任务的执行时机，防止风控异常。
//
// @author Ateng
// @since 2026-10-08
package scheduler

import (
	"context"
	"time"

	"sxd-pie-ng/internal/client"
)

// RoutinePriority 定义任务执行优先级。数值越大优先级越高。
type RoutinePriority int

const (
	// PriorityLow 低优先级任务 (如后置整理、冗余上报)
	PriorityLow RoutinePriority = 10
	// PriorityNormal 普通优先级任务 (如常规日常巡检、资源领用)
	PriorityNormal RoutinePriority = 50
	// PriorityHigh 高优先级任务 (如定时准点活动、时效性抢购)
	PriorityHigh RoutinePriority = 100
)

// ActivityRoutine 抽象神仙道日常活动任务接口。
type ActivityRoutine interface {
	// Name 返回任务唯一名称 (例如 "herb_garden", "lucky_star", "arena", "pilgrimage")
	Name() string
	// Priority 返回任务优先级
	Priority() RoutinePriority
	// Interval 返回任务周期执行间隔。若 <= 0 则表示仅执行一次
	Interval() time.Duration
	// Execute 执行具体业务逻辑，传入角色网络会话与防封抖动引擎
	Execute(ctx context.Context, session *client.RoleSession, jitter *Jitter) error
}

// FuncRoutine 帮助快速构造基于闭包函数的日常活动任务实例。
type FuncRoutine struct {
	name     string
	priority RoutinePriority
	interval time.Duration
	fn       func(ctx context.Context, session *client.RoleSession, jitter *Jitter) error
}

// NewFuncRoutine 创建一个基于函数的 ActivityRoutine 实例。
func NewFuncRoutine(
	name string,
	priority RoutinePriority,
	interval time.Duration,
	fn func(ctx context.Context, session *client.RoleSession, jitter *Jitter) error,
) *FuncRoutine {
	return &FuncRoutine{
		name:     name,
		priority: priority,
		interval: interval,
		fn:       fn,
	}
}

// Name 返回任务名称。
func (r *FuncRoutine) Name() string {
	return r.name
}

// Priority 返回任务优先级。
func (r *FuncRoutine) Priority() RoutinePriority {
	return r.priority
}

// Interval 返回执行间隔。
func (r *FuncRoutine) Interval() time.Duration {
	return r.interval
}

// Execute 调用内部闭包函数。
func (r *FuncRoutine) Execute(ctx context.Context, session *client.RoleSession, jitter *Jitter) error {
	if r.fn == nil {
		return nil
	}
	return r.fn(ctx, session, jitter)
}
