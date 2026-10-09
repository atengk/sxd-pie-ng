// Package pvp 提供本服竞技场、仙界竞技场、神魔大战与劫镖等对抗性玩法。
//
// @author Ateng
// @since 2026-10-09
package pvp

import (
	"context"
	"net"
	"testing"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

func TestArenaRoutine_Execute(t *testing.T) {
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
		RoleID:   "role_test_pvp",
		RoleName: "测试竞技场角色",
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return c1, nil
		},
	})
	_ = session.Start(context.Background())
	defer session.Close()

	time.Sleep(50 * time.Millisecond)

	routine := NewArenaRoutine()
	if routine.Name() != "arena" {
		t.Fatalf("任务名称不匹配: %s", routine.Name())
	}
	if dr, ok := routine.(scheduler.DetailedRoutine); ok {
		if dr.Domain() != "pvp" {
			t.Fatalf("领域不匹配: %s", dr.Domain())
		}
	}

	err := routine.Execute(context.Background(), session, nil)
	if err != nil {
		t.Fatalf("竞技场例程执行失败: %v", err)
	}
}
