// Package scheduler_test 验证拟人随机抖动计算与上下文取消能力。
//
// @author Ateng
// @since 2026-10-08
package scheduler_test

import (
	"context"
	"testing"
	"time"

	"sxd-pie-ng/internal/scheduler"
)

func TestJitter_DurationBounds(t *testing.T) {
	min := 1000 * time.Millisecond
	max := 3000 * time.Millisecond
	j := scheduler.NewJitter(scheduler.JitterConfig{
		MinDuration: min,
		MaxDuration: max,
	})

	// 采样 1000 次，验证每次计算结果均严密落在 [1s, 3s] 闭区间内
	for i := 0; i < 1000; i++ {
		d := j.NextDuration()
		if d < min || d > max {
			t.Fatalf("iteration %d: jitter duration %v out of bounds [%v, %v]", i, d, min, max)
		}
	}
}

func TestJitter_WaitContextCancel(t *testing.T) {
	j := scheduler.NewJitter(scheduler.JitterConfig{
		MinDuration: 1 * time.Second,
		MaxDuration: 2 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	// 50ms 后取消
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := j.Wait(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("Wait did not respect cancellation promptly, elapsed: %v", elapsed)
	}
}
