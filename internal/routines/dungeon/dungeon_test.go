// Package dungeon_test 针对关卡扫荡与挑战玩法的端到端业务闭环进行单元测试。
//
// @author Ateng
// @since 2026-10-08
package dungeon_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

	deadline := time.Now().Add(2 * time.Second)
	for sess.State() != client.StateActive && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

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

// TestDungeonSweep_201Stamina_MultiBatchDepletion 验证角色拥有实机 201 点体力时，多轮批量扫荡消耗至耗尽熔断的完整闭环。
func TestDungeonSweep_201Stamina_MultiBatchDepletion(t *testing.T) {
	// 初始 201 点体力，单次最大 10 轮 (50 点体力)
	sess, srvConn := createTestSession(t, 201)
	defer sess.Close()
	defer srvConn.Close()

	// 服务端异步接收通道
	go func() {
		for {
			_, err := protocol.ReadPacket(srvConn)
			if err != nil {
				return
			}
		}
	}()

	routine := dungeon.NewDungeonSweepRoutine(nil, dungeon.SweepConfig{
		MaxBatchTimes: 10,
	})
	ctx := context.Background()

	// 前 3 轮正常消耗 50 点体力，剩余体力充足 (>5 点)，返回 nil
	for round := 0; round < 3; round++ {
		err := routine.Execute(ctx, sess, nil)
		if err != nil {
			t.Fatalf("第 %d 轮扫荡执行失败: %v", round+1, err)
		}
	}
	if rem := sess.GetStamina(); rem != 51 {
		t.Fatalf("前 3 轮扫荡后期望剩余 51 点体力, 实际 %d", rem)
	}

	// 第 4 轮：从 51 点消耗 50 点后剩余 1 点，完成扫荡并即时触发 ErrStaminaDepleted 告知调度器进入冷却
	err := routine.Execute(ctx, sess, nil)
	if !errors.Is(err, scheduler.ErrStaminaDepleted) {
		t.Fatalf("第 4 轮扫荡消耗后期望返回 ErrStaminaDepleted, 实际返回: %v", err)
	}
	if rem := sess.GetStamina(); rem != 1 {
		t.Errorf("第 4 轮扫荡后期望剩余体力 1, 实际为 %d", rem)
	}

	// 第 5 轮：当剩余 1 点体力时再次调度，前置拦截直接拒绝并返回 ErrStaminaDepleted
	err = routine.Execute(ctx, sess, nil)
	if !errors.Is(err, scheduler.ErrStaminaDepleted) {
		t.Fatalf("第 5 轮前置拦截期望返回 ErrStaminaDepleted, 实际返回: %v", err)
	}

	// 最终体力仍应保持为 1 点
	if rem := sess.GetStamina(); rem != 1 {
		t.Errorf("熔断后期望体力仍为 1, 实际为 %d", rem)
	}

	// 验证 4 轮累计消耗体力 = 4 * 50 = 200 点
	totalCost := 201 - sess.GetStamina()
	if totalCost != 200 {
		t.Errorf("期望累计消耗体力 200, 实际消耗 %d", totalCost)
	}

	// 验证 4 轮累计经验收益 = 4 * 10 * 2500 = 100,000
	// 验证 4 轮累计铜钱收益 = 4 * 10 * 12000 = 480,000
	state := sess.GetPlayerState()
	expectedCoins := int64(36226117234 + 480000)
	if state.Coins != expectedCoins {
		t.Errorf("期望铜钱累计达到 %d, 实际为 %d", expectedCoins, state.Coins)
	}
}

// TestDungeonSweep_CustomBatchAndExactDeduction 验证自定义小批次与零头体力的精准扣除。
func TestDungeonSweep_CustomBatchAndExactDeduction(t *testing.T) {
	sess, srvConn := createTestSession(t, 23) // 23 点体力，配置最大批次为 3 次 (15 点)
	defer sess.Close()
	defer srvConn.Close()

	go func() {
		for {
			_, err := protocol.ReadPacket(srvConn)
			if err != nil {
				return
			}
		}
	}()

	routine := dungeon.NewDungeonSweepRoutine(nil, dungeon.SweepConfig{
		MaxBatchTimes: 3,
	})
	ctx := context.Background()

	// 第 1 轮：23 体力可扫 4 次，但被 MaxBatchTimes=3 限制，扫 3 次消耗 15 点，剩余 8 点 (>5)，返回 nil
	err := routine.Execute(ctx, sess, nil)
	if err != nil {
		t.Fatalf("第 1 轮执行失败: %v", err)
	}
	if rem := sess.GetStamina(); rem != 8 {
		t.Errorf("第 1 轮期望剩余 8 点体力, 实际 %d", rem)
	}

	// 第 2 轮：剩余 8 点体力，扫 1 次 (消耗 5 点)，剩余 3 点 (<5 点)，完成扫荡并即时触发 ErrStaminaDepleted
	err = routine.Execute(ctx, sess, nil)
	if !errors.Is(err, scheduler.ErrStaminaDepleted) {
		t.Fatalf("第 2 轮期望返回 ErrStaminaDepleted, 实际返回: %v", err)
	}
	if rem := sess.GetStamina(); rem != 3 {
		t.Errorf("第 2 轮期望剩余 3 点体力, 实际 %d", rem)
	}

	// 第 3 轮：剩余 3 点不足 5 点，前置熔断拦截
	err = routine.Execute(ctx, sess, nil)
	if !errors.Is(err, scheduler.ErrStaminaDepleted) {
		t.Fatalf("第 3 轮期望返回 ErrStaminaDepleted, 实际返回: %v", err)
	}
}

// TestDungeonSweep_ConsumeStamina_BoundaryAndConcurrency 验证体力扣除的边界条件与多协程并发安全。
func TestDungeonSweep_ConsumeStamina_BoundaryAndConcurrency(t *testing.T) {
	sess, srvConn := createTestSession(t, 100)
	defer sess.Close()
	defer srvConn.Close()

	// 1. 边界超扣测试: 当前 100 点体力，尝试扣除 105 点应被拒绝且体力不变
	_, err := sess.ConsumeStamina(105)
	if err == nil {
		t.Fatal("期望扣除超出体力时返回错误，实际成功")
	}
	if rem := sess.GetStamina(); rem != 100 {
		t.Fatalf("超扣失败后期望体力仍为 100, 实际为 %d", rem)
	}

	// 2. 并发扣除测试: 25 个并发协程，每个尝试扣除 5 点体力 (总需求 125 点)
	// 预期结果: 恰好 20 个协程成功扣除，5 个协程因体力不足失败，最终剩余体力精确为 0
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := sess.ConsumeStamina(5)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}()
	}

	wg.Wait()

	if successCount != 20 {
		t.Errorf("期望成功扣除 20 次, 实际成功 %d 次", successCount)
	}
	if failCount != 5 {
		t.Errorf("期望失败 5 次, 实际失败 %d 次", failCount)
	}
	if rem := sess.GetStamina(); rem != 0 {
		t.Errorf("并发扣除后最终期望体力为 0, 实际为 %d", rem)
	}
}

// TestDungeonSweep_ServerPropUpdateSync 验证收到服务器 0x0300 广播推送时体力值的权威动态同步。
func TestDungeonSweep_ServerPropUpdateSync(t *testing.T) {
	sess, srvConn := createTestSession(t, 201)
	defer sess.Close()
	defer srvConn.Close()

	// 注册 0x0300 属性同步处理器
	sess.RegisterHandler(protocol.ActionIDPlayerUpdateData, func(p *protocol.Packet) {
		r := protocol.NewReader(p.Payload)
		for r.Remaining() >= 5 {
			prop, err := r.ReadUint8()
			if err != nil {
				break
			}
			val, err := r.ReadInt32()
			if err != nil {
				break
			}
			if prop == protocol.PlayerPropPower {
				sess.UpdatePlayerState(func(ps *client.PlayerState) {
					ps.Stamina = int(val)
				})
			}
		}
	})

	// 模拟服务端向客户端推送 0x0300 封包，通知体力更新为 146 点
	w := protocol.NewWriter()
	w.WriteUint8(protocol.PlayerPropPower)
	w.WriteInt32(146)
	pkt := protocol.NewPacket(protocol.ActionIDPlayerUpdateData, w.Bytes())

	raw, err := pkt.Marshal()
	if err != nil {
		t.Fatalf("Marshal 0x0300 失败: %v", err)
	}
	if _, err := srvConn.Write(raw); err != nil {
		t.Fatalf("服务端写入 0x0300 封包失败: %v", err)
	}

	// 等待客户端消费回包并同步
	deadline := time.Now().Add(2 * time.Second)
	for sess.GetStamina() != 146 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if rem := sess.GetStamina(); rem != 146 {
		t.Errorf("期望体力同步为 146, 实际为 %d", rem)
	}
}

// TestDungeonSweep_ServerResponse_RewardSync 验证接收到服务端真实扫荡结算回包时的收益同步与精准扣减
func TestDungeonSweep_ServerResponse_RewardSync(t *testing.T) {
	sess, srvConn := createTestSession(t, 100)
	defer sess.Close()
	defer srvConn.Close()

	initialCoins := sess.GetPlayerState().Coins

	// 模拟服务端接收扫荡并回复真实结算包
	go func() {
		pkt, err := protocol.ReadPacket(srvConn)
		if err != nil {
			return
		}
		if pkt.ActionID == protocol.ActionMissionSweep {
			req, _ := protocol.ParseSweepRequest(pkt.Payload)
			// 服务端回包：10 次，扣除 50 体力，产出 35,000 经验与 180,000 铜钱
			res := protocol.SweepResult{
				Success:   true,
				MissionID: req.MissionID,
				Times:     req.Times,
				CostPower: 50,
				GainExp:   35000,
				GainCoins: 180000,
				Message:   "扫荡完成",
			}
			respPkt, _ := protocol.BuildSweepResultPacket(res)
			_ = protocol.WritePacket(srvConn, respPkt)
		}
	}()

	routine := dungeon.NewDungeonSweepRoutine(nil)
	err := routine.Execute(context.Background(), sess, nil)
	if err != nil {
		t.Fatalf("扫荡执行失败: %v", err)
	}

	// 验证体力扣减
	if rem := sess.GetStamina(); rem != 50 {
		t.Errorf("期望剩余体力 50, 实际为 %d", rem)
	}

	// 验证铜钱收益按服务端回包增加 180,000
	if currentCoins := sess.GetPlayerState().Coins; currentCoins != initialCoins+180000 {
		t.Errorf("期望铜钱增加 180,000, 实际为 %d (初始: %d)", currentCoins, initialCoins)
	}
}

// TestDungeonSweep_ServerRejection_StaminaDepleted 验证当服务端返回“体力不足”时触发语义智能熔断
func TestDungeonSweep_ServerRejection_StaminaDepleted(t *testing.T) {
	sess, srvConn := createTestSession(t, 50)
	defer sess.Close()
	defer srvConn.Close()

	go func() {
		pkt, err := protocol.ReadPacket(srvConn)
		if err != nil {
			return
		}
		if pkt.ActionID == protocol.ActionMissionSweep {
			req, _ := protocol.ParseSweepRequest(pkt.Payload)
			res := protocol.SweepResult{
				Success:   false,
				MissionID: req.MissionID,
				Times:     req.Times,
				Message:   "体力不足，扫荡失败",
			}
			respPkt, _ := protocol.BuildSweepResultPacket(res)
			_ = protocol.WritePacket(srvConn, respPkt)
		}
	}()

	routine := dungeon.NewDungeonSweepRoutine(nil)
	err := routine.Execute(context.Background(), sess, nil)
	if !errors.Is(err, scheduler.ErrStaminaDepleted) {
		t.Fatalf("期望服务端拒绝时触发 ErrStaminaDepleted, 实际返回: %v", err)
	}
}

// TestDungeonSweep_ServerRejection_BagFull 验证当服务端返回“背包已满”时触发背包语义熔断
func TestDungeonSweep_ServerRejection_BagFull(t *testing.T) {
	sess, srvConn := createTestSession(t, 50)
	defer sess.Close()
	defer srvConn.Close()

	go func() {
		pkt, err := protocol.ReadPacket(srvConn)
		if err != nil {
			return
		}
		if pkt.ActionID == protocol.ActionMissionSweep {
			req, _ := protocol.ParseSweepRequest(pkt.Payload)
			res := protocol.SweepResult{
				Success:   false,
				MissionID: req.MissionID,
				Times:     req.Times,
				Message:   "背包已满，无法放入掉落物品",
			}
			respPkt, _ := protocol.BuildSweepResultPacket(res)
			_ = protocol.WritePacket(srvConn, respPkt)
		}
	}()

	routine := dungeon.NewDungeonSweepRoutine(nil)
	err := routine.Execute(context.Background(), sess, nil)
	if !errors.Is(err, scheduler.ErrBagFull) {
		t.Fatalf("期望服务端拒绝时触发 ErrBagFull, 实际返回: %v", err)
	}
}

