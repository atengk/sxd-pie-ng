// Package client_test 验证角色客户端长连接真实登录握手、场景同步与心跳保活状态机。
//
// @author Ateng
// @since 2026-10-10
package client_test

import (
	"context"
	"encoding/hex"
	"net"
	"sync"
	"testing"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/protocol"
)

// TestRoleSession_DefaultGatewayHandshake_FullFlow 验证通过 net.Pipe 模拟真实网关：
// 1. 发送 Module 0, Action 0 登录请求
// 2. 接收并校验 ResultCode=4 认证成功回包
// 3. 自动触发 Action 72 (TownID=4879) 与 Action 99 城镇场景初始化
// 4. 成功流转至 StateActive 并在激活态维持心跳保活
func TestRoleSession_DefaultGatewayHandshake_FullFlow(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	const goldenRespCase2Hex = "00000000040a000000000000000001000b0000108501"
	resp2Payload, _ := hex.DecodeString(goldenRespCase2Hex)
	const goldenRespCase3Hex = "0000130f00020000130f00001312"
	resp3Payload, _ := hex.DecodeString(goldenRespCase3Hex)

	var serverWg sync.WaitGroup
	serverWg.Add(1)

	receivedLogin := make(chan bool, 1)
	receivedStep1 := make(chan bool, 1)
	receivedStep2 := make(chan bool, 1)
	receivedHeartbeat := make(chan bool, 2)

	// 模拟远程游戏网关
	go func() {
		defer serverWg.Done()
		defer serverConn.Close()

		for {
			pkt, err := protocol.ReadPacket(serverConn)
			if err != nil {
				return
			}

			switch pkt.ActionID {
			case protocol.ActionIDPlayerLogin:
				receivedLogin <- true
				// 回复 Golden Case 2 (ResultCode=4 认证成功)
				resp := protocol.NewPacket(protocol.ActionIDPlayerLogin, resp2Payload)
				_ = protocol.WritePacket(serverConn, resp)

			case protocol.ActionIDPlayerInitStep1:
				receivedStep1 <- true
				// 回复 Golden Case 3 (TownID=4879, TownLine=2)
				resp := protocol.NewPacket(protocol.ActionIDPlayerInitStep1, resp3Payload)
				_ = protocol.WritePacket(serverConn, resp)

			case protocol.ActionIDPlayerInitStep2:
				receivedStep2 <- true
				// 回复 Step2 确认
				resp := protocol.NewPacket(protocol.ActionIDPlayerInitStep2, []byte{0x00})
				_ = protocol.WritePacket(serverConn, resp)

			case protocol.ActionHeartbeat:
				select {
				case receivedHeartbeat <- true:
				default:
				}
				// 回复心跳
				hbResp := protocol.NewPacket(protocol.ActionHeartbeat, []byte{0x06, 0x00, 0x0a, 0x28})
				_ = protocol.WritePacket(serverConn, hbResp)
			}
		}
	}()

	cfg := client.SessionConfig{
		RoleID:            "role-gateway-01",
		RoleName:          "神仙测试玩家",
		Username:          "kongyu",
		Token:             "8b3a94ab933b3657ca188f05f8405db6",
		ServerAddr:        "gateway-test-pipe",
		HeartbeatInterval: 100 * time.Millisecond,
		HeartbeatTimeout:  500 * time.Millisecond,
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return clientConn, nil
		},
		// 注意: 不配置 Authenticator，以此验证客户端默认自动触发网关握手与场景同步
	}

	session := client.NewRoleSession(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := session.Start(ctx); err != nil {
		t.Fatalf("session.Start failed: %v", err)
	}

	// 1. 验证接收到登录请求
	select {
	case <-receivedLogin:
	case <-time.After(2 * time.Second):
		t.Fatal("超时未收到客户端登录握手包 (ActionIDPlayerLogin)")
	}

	// 2. 验证接收到 Action 72 场景初始化步 1
	select {
	case <-receivedStep1:
	case <-time.After(2 * time.Second):
		t.Fatal("超时未收到客户端场景同步包 (ActionIDPlayerInitStep1)")
	}

	// 3. 验证接收到 Action 99 场景初始化步 2
	select {
	case <-receivedStep2:
	case <-time.After(2 * time.Second):
		t.Fatal("超时未收到客户端场景同步包 (ActionIDPlayerInitStep2)")
	}

	// 4. 等待会话成功激活
	activeTimeout := time.After(2 * time.Second)
	for session.State() != client.StateActive {
		select {
		case <-activeTimeout:
			t.Fatalf("超时未能激活会话，当前状态: %s", session.State())
		case <-time.After(10 * time.Millisecond):
		}
	}

	// 5. 校验 TownID 已同步更新为 4879
	state := session.GetPlayerState()
	if state.TownID != 4879 {
		t.Errorf("期望角色当前 TownID 为 4879, 实际为 %d", state.TownID)
	}

	// 6. 验证激活态心跳已定时发送
	select {
	case <-receivedHeartbeat:
	case <-time.After(2 * time.Second):
		t.Fatal("超时未收到激活态心跳包 (ActionHeartbeat)")
	}

	session.Close()
	serverWg.Wait()
}

// TestRoleSession_DefaultGatewayHandshake_Rejected 验证当服务端返回非法凭据 (ResultCode!=4) 时抛出明确异常并拒绝激活
func TestRoleSession_DefaultGatewayHandshake_Rejected(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	var serverWg sync.WaitGroup
	serverWg.Add(1)

	go func() {
		defer serverWg.Done()
		defer serverConn.Close()

		pkt, err := protocol.ReadPacket(serverConn)
		if err != nil {
			return
		}
		if pkt.ActionID == protocol.ActionIDPlayerLogin {
			// 回复 ResultCode = 1 (凭证认证失败)
			failPayload := []byte{0x00, 0x00, 0x00, 0x00, 0x01}
			_ = protocol.WritePacket(serverConn, protocol.NewPacket(protocol.ActionIDPlayerLogin, failPayload))
		}
	}()

	cfg := client.SessionConfig{
		RoleID:               "role-rejected-01",
		RoleName:             "非法凭证角色",
		Username:             "bad_user",
		Token:                "invalid_token",
		ServerAddr:           "gateway-reject-pipe",
		MaxReconnectAttempts: 1, // 失败后不重试直接退出
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return clientConn, nil
		},
	}

	session := client.NewRoleSession(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = session.Start(ctx)

	// 等待会话关闭或状态判定
	time.Sleep(300 * time.Millisecond)

	if session.State() == client.StateActive {
		t.Fatal("凭据认证失败时不应进入 StateActive")
	}

	session.Close()
	serverWg.Wait()
}
