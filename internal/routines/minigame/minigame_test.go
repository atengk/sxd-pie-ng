// Package minigame 提供吉星高照、仙履奇缘答题、刮刮卡、钓鱼等益智趣味活动。
//
// @author Ateng
// @since 2026-10-09
package minigame

import (
	"context"
	"net"
	"testing"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/qa"
	"sxd-pie-ng/internal/scheduler"
)

func TestImmortalFantasyRoutine_Execute(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	go func() {
		buf := make([]byte, 1024)
		for {
			_, err := c2.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	session := client.NewRoleSession(client.SessionConfig{
		RoleID:   "role_test_minigame",
		RoleName: "测试仙履角色",
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return c1, nil
		},
	})
	_ = session.Start(context.Background())
	defer session.Close()

	time.Sleep(50 * time.Millisecond)

	qaEngine, _ := qa.NewEngine("")
	routine := NewImmortalFantasyRoutine(qaEngine)
	if routine.Name() != "immortal_fantasy" {
		t.Fatalf("任务名称不匹配: %s", routine.Name())
	}
	if dr, ok := routine.(scheduler.DetailedRoutine); ok {
		if dr.Domain() != "minigame" {
			t.Fatalf("领域不匹配: %s", dr.Domain())
		}
	}

	err := routine.Execute(context.Background(), session, nil)
	if err != nil {
		t.Fatalf("仙履奇缘例程执行失败: %v", err)
	}
}
