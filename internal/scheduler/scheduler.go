// Package scheduler 提供拟人随机抖动任务调度引擎，协调各类日常活动任务的执行时机，防止风控异常。
//
// @author Ateng
// @since 2026-10-08
package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"sxd-pie-ng/internal/client"
)

type scheduledItem struct {
	routine  ActivityRoutine
	nextRun  time.Time
	lastRun  time.Time
	runCount int
}

// RoleScheduler 为特定角色会话提供专属且隔离的任务调度器。
type RoleScheduler struct {
	roleID  string
	session *client.RoleSession
	jitter  *Jitter

	items   []*scheduledItem
	itemsMu sync.RWMutex

	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	stateMu sync.Mutex
}

// NewRoleScheduler 创建新的角色任务调度器。
func NewRoleScheduler(roleID string, session *client.RoleSession, jitter *Jitter) *RoleScheduler {
	if jitter == nil {
		jitter = NewJitter(JitterConfig{})
	}
	return &RoleScheduler{
		roleID:  roleID,
		session: session,
		jitter:  jitter,
		items:   make([]*scheduledItem, 0),
	}
}

// Register 注册一个日常活动任务到调度队列。
func (s *RoleScheduler) Register(routine ActivityRoutine) error {
	if routine == nil {
		return errors.New("scheduler: cannot register nil routine")
	}

	s.itemsMu.Lock()
	defer s.itemsMu.Unlock()

	item := &scheduledItem{
		routine: routine,
		nextRun: time.Now(), // 注册后立即进入就绪候选
	}
	s.items = append(s.items, item)
	return nil
}

// Start 启动调度器后台协程。
func (s *RoleScheduler) Start(parentCtx context.Context) error {
	s.stateMu.Lock()
	if s.running {
		s.stateMu.Unlock()
		return errors.New("scheduler: already running")
	}
	s.running = true
	s.ctx, s.cancel = context.WithCancel(parentCtx)
	s.stateMu.Unlock()

	s.wg.Add(1)
	go s.runLoop()

	return nil
}

// Stop 停止调度器并等待正在执行的任务结束。
func (s *RoleScheduler) Stop() {
	s.stateMu.Lock()
	if !s.running {
		s.stateMu.Unlock()
		return
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	s.stateMu.Unlock()

	s.wg.Wait()
}

func (s *RoleScheduler) runLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		item := s.pickNextReadyItem()
		if item == nil {
			// 当前无就绪任务，轻度休眠后继续探查
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}

		// 动作前强制注入拟人防封抖动延迟
		if err := s.jitter.Wait(s.ctx); err != nil {
			return
		}

		slog.Debug("开始执行日常任务",
			"role_id", s.roleID,
			"routine", item.routine.Name(),
			"priority", item.routine.Priority(),
		)

		// 执行任务
		_ = item.routine.Execute(s.ctx, s.session, s.jitter)

		item.lastRun = time.Now()
		item.runCount++

		// 处理执行周期更新或单次任务移除
		s.itemsMu.Lock()
		if item.routine.Interval() > 0 {
			item.nextRun = time.Now().Add(item.routine.Interval())
		} else {
			s.removeItemLocked(item)
		}
		s.itemsMu.Unlock()
	}
}

func (s *RoleScheduler) pickNextReadyItem() *scheduledItem {
	s.itemsMu.RLock()
	defer s.itemsMu.RUnlock()

	now := time.Now()
	var candidates []*scheduledItem
	for _, it := range s.items {
		if now.After(it.nextRun) || now.Equal(it.nextRun) {
			candidates = append(candidates, it)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// 按优先级从大到小排序 (高优先级优先执行)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].routine.Priority() > candidates[j].routine.Priority()
	})

	return candidates[0]
}

func (s *RoleScheduler) removeItemLocked(target *scheduledItem) {
	for i, it := range s.items {
		if it == target {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return
		}
	}
}
