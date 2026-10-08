// Package scheduler 提供拟人随机抖动任务调度引擎，协调各类日常活动任务的执行时机，防止风控异常。
//
// @author Ateng
// @since 2026-10-08
package scheduler

import (
	"context"
	"fmt"
	"sync"

	"sxd-pie-ng/internal/client"
)

// Dispatcher 负责统筹管理系统中所有角色的独立任务调度器。
type Dispatcher struct {
	schedulers map[string]*RoleScheduler
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
	stateMu    sync.Mutex
}

// NewDispatcher 创建一个多角色调度管理器。
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		schedulers: make(map[string]*RoleScheduler),
	}
}

// AddRole 为指定角色注册并分配独立的调度器。
func (d *Dispatcher) AddRole(roleID string, session *client.RoleSession, jitterCfg JitterConfig) (*RoleScheduler, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.schedulers[roleID]; exists {
		return nil, fmt.Errorf("scheduler: role %s already registered", roleID)
	}

	jitter := NewJitter(jitterCfg)
	sched := NewRoleScheduler(roleID, session, jitter)
	d.schedulers[roleID] = sched

	// 若 Dispatcher 当前已处于全局运行态，则自动启动新加入角色的调度协程
	d.stateMu.Lock()
	if d.running {
		_ = sched.Start(d.ctx)
	}
	d.stateMu.Unlock()

	return sched, nil
}

// RemoveRole 停止并移除指定角色的调度器。
func (d *Dispatcher) RemoveRole(roleID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if sched, exists := d.schedulers[roleID]; exists {
		sched.Stop()
		delete(d.schedulers, roleID)
	}
}

// GetRole 获取指定角色的任务调度器。
func (d *Dispatcher) GetRole(roleID string) *RoleScheduler {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.schedulers[roleID]
}

// StartAll 启动所有已注册角色的调度协程。
func (d *Dispatcher) StartAll(parentCtx context.Context) {
	d.stateMu.Lock()
	if d.running {
		d.stateMu.Unlock()
		return
	}
	d.running = true
	d.ctx, d.cancel = context.WithCancel(parentCtx)
	d.stateMu.Unlock()

	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, sched := range d.schedulers {
		_ = sched.Start(d.ctx)
	}
}

// StopAll 安全停止所有角色的调度协程。
func (d *Dispatcher) StopAll() {
	d.stateMu.Lock()
	if !d.running {
		d.stateMu.Unlock()
		return
	}
	d.running = false
	if d.cancel != nil {
		d.cancel()
	}
	d.stateMu.Unlock()

	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, sched := range d.schedulers {
		sched.Stop()
	}
}

// TriggerBatch 为所有受管角色的调度器统一触发指定类型的批量任务唤醒。
func (d *Dispatcher) TriggerBatch(schedType ScheduleType) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	total := 0
	for _, sched := range d.schedulers {
		total += sched.TriggerBatch(schedType)
	}
	return total
}

