// Package farming 提供药园种植、取经护送、仙界矿山与异兽养育等资源产出玩法。
//
// @author Ateng
// @since 2026-10-09
package farming

import (
	"context"
	"net"
	"testing"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

func TestHerbGardenRoutine_Execute(t *testing.T) {
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
		RoleID:   "role_test_01",
		RoleName: "测试角色",
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return c1, nil
		},
	})
	_ = session.Start(context.Background())
	defer session.Close()

	time.Sleep(50 * time.Millisecond)

	routine := NewHerbGardenRoutine()
	if routine.Name() != "herb_garden" {
		t.Fatalf("任务名称不匹配: %s", routine.Name())
	}
	if dr, ok := routine.(scheduler.DetailedRoutine); ok {
		if dr.Domain() != "farming" {
			t.Fatalf("领域不匹配: %s", dr.Domain())
		}
	}

	err := routine.Execute(context.Background(), session, nil)
	if err != nil {
		t.Fatalf("药园例程执行失败: %v", err)
	}
}
