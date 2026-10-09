// Package daily 提供每日签到、日常任务与邮件福利自动化逻辑。
//
// @author Ateng
// @since 2026-10-09
package daily

import (
	"context"
	"net"
	"testing"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/scheduler"
)

func TestDailyRoutines_Execute(t *testing.T) {
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
		RoleID:   "role_test_daily",
		RoleName: "测试日常角色",
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return c1, nil
		},
	})
	_ = session.Start(context.Background())
	defer session.Close()

	time.Sleep(50 * time.Millisecond)

	r1 := NewSignInRoutine()
	if dr, ok := r1.(scheduler.DetailedRoutine); ok {
		if dr.Domain() != "daily" {
			t.Fatalf("领域不匹配: %s", dr.Domain())
		}
	}
	if err := r1.Execute(context.Background(), session, nil); err != nil {
		t.Fatalf("签到任务执行失败: %v", err)
	}

	r2 := NewDailyQuestRoutine()
	if err := r2.Execute(context.Background(), session, nil); err != nil {
		t.Fatalf("日常任务执行失败: %v", err)
	}

	r3 := NewMailCollectRoutine()
	if err := r3.Execute(context.Background(), session, nil); err != nil {
		t.Fatalf("邮件收取执行失败: %v", err)
	}
}
