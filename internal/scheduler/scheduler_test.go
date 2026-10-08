// Package scheduler_test 验证任务调度器优先级排序、生命周期、拟人防封抖动与多角色并发隔离。
//
// @author Ateng
// @since 2026-10-08
package scheduler_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

func TestScheduler_PriorityOrder(t *testing.T) {
	fastJitter := scheduler.NewJitter(scheduler.JitterConfig{
		MinDuration: 5 * time.Millisecond,
		MaxDuration: 10 * time.Millisecond,
	})

	sched := scheduler.NewRoleScheduler("role-priority", nil, fastJitter)

	executionOrder := make([]string, 0)
	var mu sync.Mutex

	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		executionOrder = append(executionOrder, name)
	}

	done := make(chan struct{})

	// 注册低优先级任务
	_ = sched.Register(scheduler.NewFuncRoutine(
		"low_routine",
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
			record("low")
			close(done)
			return nil
		},
	))

	// 注册高优先级任务
	_ = sched.Register(scheduler.NewFuncRoutine(
		"high_routine",
		scheduler.PriorityHigh,
		0,
		func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
			record("high")
			return nil
		},
	))

	// 注册普通优先级任务
	_ = sched.Register(scheduler.NewFuncRoutine(
		"normal_routine",
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
			record("normal")
			return nil
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("sched.Start failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for routines to complete")
	}

	sched.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 executed routines, got %d", len(executionOrder))
	}
	if executionOrder[0] != "high" || executionOrder[1] != "normal" || executionOrder[2] != "low" {
		t.Errorf("unexpected execution order: %v, expected [high, normal, low]", executionOrder)
	}
}

func TestScheduler_RecurringRoutine(t *testing.T) {
	fastJitter := scheduler.NewJitter(scheduler.JitterConfig{
		MinDuration: 2 * time.Millisecond,
		MaxDuration: 5 * time.Millisecond,
	})

	sched := scheduler.NewRoleScheduler("role-recur", nil, fastJitter)

	var runCount atomic.Int32
	done := make(chan struct{})

	// 周期性任务，每 30ms 执行一次
	_ = sched.Register(scheduler.NewFuncRoutine(
		"herb_garden",
		scheduler.PriorityNormal,
		30*time.Millisecond,
		func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
			if runCount.Add(1) >= 3 {
				select {
				case <-done:
				default:
					close(done)
				}
			}
			return nil
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("sched.Start failed: %v", err)
	}

	select {
	case <-done:
		// 成功周期运行至少 3 次
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for recurring runs, actual: %d", runCount.Load())
	}

	sched.Stop()
}

func TestDispatcher_MultiRoleConcurrentIsolation(t *testing.T) {
	disp := scheduler.NewDispatcher()

	fastJitter := scheduler.JitterConfig{
		MinDuration: 2 * time.Millisecond,
		MaxDuration: 5 * time.Millisecond,
	}

	schedA, err := disp.AddRole("role-A", nil, fastJitter)
	if err != nil {
		t.Fatalf("AddRole A failed: %v", err)
	}
	schedB, err := disp.AddRole("role-B", nil, fastJitter)
	if err != nil {
		t.Fatalf("AddRole B failed: %v", err)
	}

	var roleARuns atomic.Int32
	var roleBRuns atomic.Int32
	roleBDone := make(chan struct{})

	// Role A 的任务被模拟网络延迟阻塞较长时间 (150ms)
	_ = schedA.Register(scheduler.NewFuncRoutine(
		"slow_action_A",
		scheduler.PriorityNormal,
		50*time.Millisecond,
		func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
			roleARuns.Add(1)
			time.Sleep(150 * time.Millisecond)
			return nil
		},
	))

	// Role B 的任务非常轻快，不受 Role A 阻塞影响
	_ = schedB.Register(scheduler.NewFuncRoutine(
		"fast_action_B",
		scheduler.PriorityNormal,
		20*time.Millisecond,
		func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
			if roleBRuns.Add(1) >= 4 {
				select {
				case <-roleBDone:
				default:
					close(roleBDone)
				}
			}
			return nil
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	disp.StartAll(ctx)

	// 等待 Role B 迅速完成 4 次执行
	select {
	case <-roleBDone:
		// 此时验证 Role A 并未拖垮 Role B
		bCount := roleBRuns.Load()
		aCount := roleARuns.Load()
		if bCount < 4 {
			t.Errorf("Role B should have executed >= 4 times, got %d", bCount)
		}
		if aCount >= bCount {
			t.Errorf("Role A (slow) should have executed fewer times than Role B (fast): A=%d, B=%d", aCount, bCount)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Role B to complete multiple runs")
	}

	disp.StopAll()
}

func TestDispatcher_GetAndRemoveRole(t *testing.T) {
	disp := scheduler.NewDispatcher()
	sched, err := disp.AddRole("role-c", nil, scheduler.JitterConfig{})
	if err != nil {
		t.Fatalf("AddRole failed: %v", err)
	}

	if disp.GetRole("role-c") != sched {
		t.Errorf("expected GetRole to return matching scheduler")
	}

	// 重复添加报错
	_, err = disp.AddRole("role-c", nil, scheduler.JitterConfig{})
	if err == nil {
		t.Errorf("expected error adding duplicate role")
	}

	disp.RemoveRole("role-c")
	if disp.GetRole("role-c") != nil {
		t.Errorf("expected nil after RemoveRole")
	}
}

func TestScheduler_ErrorBranches(t *testing.T) {
	sched := scheduler.NewRoleScheduler("role-err", nil, nil)
	// 1. 注册空任务
	if err := sched.Register(nil); err == nil {
		t.Errorf("expected error registering nil routine")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 2. 重复启动
	if err := sched.Start(ctx); err == nil {
		t.Errorf("expected error on duplicate Start")
	}

	sched.Stop()
}
