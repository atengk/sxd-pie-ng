// Package scheduler 提供拟人随机抖动任务调度引擎，协调各类日常活动任务的执行时机，防止风控异常。
//
// @author Ateng
// @since 2026-10-08
package scheduler

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// DefaultMinJitter 默认最小防封抖动延迟 (1 秒)
const DefaultMinJitter = 1000 * time.Millisecond

// DefaultMaxJitter 默认最大防封抖动延迟 (3 秒)
const DefaultMaxJitter = 3000 * time.Millisecond

// JitterConfig 拟人抖动引擎配置项
type JitterConfig struct {
	MinDuration time.Duration
	MaxDuration time.Duration
}

// Jitter 负责计算基于正态分布的动态随机延迟，并提供可中断等待。
type Jitter struct {
	minDuration time.Duration
	maxDuration time.Duration
	rng         *rand.Rand
	mu          sync.Mutex
}

// NewJitter 创建一个新的拟人抖动实例。
func NewJitter(cfg JitterConfig) *Jitter {
	minD := cfg.MinDuration
	maxD := cfg.MaxDuration

	if minD <= 0 {
		minD = DefaultMinJitter
	}
	if maxD <= 0 {
		maxD = DefaultMaxJitter
	}
	if minD > maxD {
		minD, maxD = maxD, minD
	}

	return &Jitter{
		minDuration: minD,
		maxDuration: maxD,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NextDuration 计算下一个基于正态分布并截断于 [MinDuration, MaxDuration] 的随机时间跨度。
func (j *Jitter) NextDuration() time.Duration {
	j.mu.Lock()
	defer j.mu.Unlock()

	minF := float64(j.minDuration)
	maxF := float64(j.maxDuration)

	// 正态分布均值 μ 为中位数，3σ 原则使得 99.7% 的样本落在 [min, max]
	mean := (minF + maxF) / 2.0
	stdDev := (maxF - minF) / 6.0

	sample := mean + j.rng.NormFloat64()*stdDev

	// 强制双向截断在合法闭区间内
	if sample < minF {
		sample = minF
	} else if sample > maxF {
		sample = maxF
	}

	return time.Duration(sample)
}

// Wait 计算一次随机抖动时间并阻塞等待，支持通过 context 及时取消。
func (j *Jitter) Wait(ctx context.Context) error {
	duration := j.NextDuration()
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
