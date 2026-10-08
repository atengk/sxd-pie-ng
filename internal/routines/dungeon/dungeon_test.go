// Package dungeon_test 针对关卡扫荡与挑战玩法的端到端业务闭环进行单元测试。
//
// @author Ateng
// @since 2026-10-08
package dungeon_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/dictionary"
	"sxd-pie-ng/internal/protocol"
	"sxd-pie-ng/internal/routines/dungeon"
	"sxd-pie-ng/internal/scheduler"
)

func createTestSession(t *testing.T, initialStamina int) (*client.RoleSession, net.Conn) {
	c1, c2 := net.Pipe()

	cfg := client.SessionConfig{
		RoleID:     "test-role-sweep",
		RoleName:   "梦一场",
		ServerAddr: "mock",
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return c1, nil
		},
	}
	sess := client.NewRoleSession(cfg)
	sess.SetStamina(initialStamina)
	_ = sess.Start(context.Background())

	return sess, c2
}

func TestDungeonSweep_NormalExecution(t *testing.T) {
	sess, srvConn := createTestSession(t, 70) // 70 体力，扫 10 次消耗 50，剩余 20
	defer sess.Close()
	defer srvConn.Close()

	// 模拟服务端接收扫荡包
	pktCh := make(chan *protocol.Packet, 1)
	go func() {
		pkt, err := protocol.ReadPacket(srvConn)
		if err == nil {
			pktCh <- pkt
		}
	}()

	dictRepo, err := dictionary.NewRepository("")
	if err != nil {
		t.Logf("本地未找到 Pieb.db，将使用降级关卡模式测试: %v", err)
	}

	routine := dungeon.NewDungeonSweepRoutine(dictRepo)
	ctx := context.Background()

	err = routine.Execute(ctx, sess, nil)
	if err != nil {
		t.Fatalf("执行关卡扫荡失败: %v", err)
	}

	// 验证向服务器发送了 ActionMissionSweep 封包
	select {
	case pkt := <-pktCh:
		if pkt.ActionID != protocol.ActionMissionSweep {
			t.Errorf("期望 ActionID 0x%04X, 实际 0x%04X", protocol.ActionMissionSweep, pkt.ActionID)
		}
		req, err := protocol.ParseSweepRequest(pkt.Payload)
		if err != nil {
			t.Fatalf("解析服务端收到的请求包失败: %v", err)
		}
		if req.Times != 10 { // 70 体力单次最多批处理 10 次
			t.Errorf("期望扫荡 10 次, 实际 %d 次", req.Times)
		}
	default:
		t.Error("未收到向服务端发送的扫荡封包")
	}

	// 验证体力扣除 (70 - 50 = 20)
	remStamina := sess.GetStamina()
	if remStamina != 20 {
		t.Errorf("期望剩余体力为 20, 实际为 %d", remStamina)
	}

	// 验证收益增加 (初始 1,000,000 + 10 * 12000 = 1,120,000)
	state := sess.GetPlayerState()
	if state.Coins <= 1000000 {
		t.Errorf("期望获得铜钱收益, 实际为 %d", state.Coins)
	}
}

func TestDungeonSweep_StaminaDepleted_SmartCooling(t *testing.T) {
	sess, srvConn := createTestSession(t, 2) // 体力仅有 2 点，不足 5 点
	defer sess.Close()
	defer srvConn.Close()

	routine := dungeon.NewDungeonSweepRoutine(nil)
	ctx := context.Background()

	err := routine.Execute(ctx, sess, nil)
	if !errors.Is(err, scheduler.ErrStaminaDepleted) {
		t.Fatalf("期望返回 ErrStaminaDepleted 触发熔断, 实际返回: %v", err)
	}
}

func TestDungeonSweep_BagFull(t *testing.T) {
	sess, srvConn := createTestSession(t, 100)
	defer sess.Close()
	defer srvConn.Close()

	// 模拟背包已满
	sess.UpdatePlayerState(func(ps *client.PlayerState) {
		ps.BagCapacity = 0
	})

	routine := dungeon.NewDungeonSweepRoutine(nil)
	ctx := context.Background()

	err := routine.Execute(ctx, sess, nil)
	if !errors.Is(err, scheduler.ErrBagFull) {
		t.Fatalf("期望返回 ErrBagFull, 实际返回: %v", err)
	}
}
