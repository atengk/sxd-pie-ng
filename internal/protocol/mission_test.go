// Package protocol_test 提供神仙道关卡协议封包的单元测试。
//
// @author Ateng
// @since 2026-10-08
package protocol_test

import (
	"testing"

	"sxd-pie-ng/internal/protocol"
)

func TestMission_SweepPacket(t *testing.T) {
	req := protocol.SweepRequest{
		MissionID: 105,
		Times:     10,
	}

	pkt, err := protocol.BuildSweepPacket(req)
	if err != nil {
		t.Fatalf("BuildSweepPacket 失败: %v", err)
	}
	if pkt.ActionID != protocol.ActionMissionSweep {
		t.Errorf("期望 ActionID 为 0x%04X, 实际为 0x%04X", protocol.ActionMissionSweep, pkt.ActionID)
	}

	parsedReq, err := protocol.ParseSweepRequest(pkt.Payload)
	if err != nil {
		t.Fatalf("ParseSweepRequest 失败: %v", err)
	}
	if parsedReq.MissionID != req.MissionID || parsedReq.Times != req.Times {
		t.Errorf("解析请求不一致: 期望 %+v, 实际 %+v", req, *parsedReq)
	}

	res := protocol.SweepResult{
		Success:   true,
		MissionID: 105,
		Times:     10,
		CostPower: 50,
		GainExp:   25000,
		GainCoins: 120000,
		Message:   "扫荡完毕",
	}

	resPkt, err := protocol.BuildSweepResultPacket(res)
	if err != nil {
		t.Fatalf("BuildSweepResultPacket 失败: %v", err)
	}

	parsedRes, err := protocol.ParseSweepResult(resPkt.Payload)
	if err != nil {
		t.Fatalf("ParseSweepResult 失败: %v", err)
	}
	if !parsedRes.Success || parsedRes.CostPower != 50 || parsedRes.GainCoins != 120000 || parsedRes.Message != "扫荡完毕" {
		t.Errorf("解析结算不一致: %+v", *parsedRes)
	}
}

func TestMission_LoginPacket(t *testing.T) {
	pkt, err := protocol.BuildLoginPacket("梦一场", "mock-auth-token-123")
	if err != nil {
		t.Fatalf("BuildLoginPacket 失败: %v", err)
	}
	if pkt.ActionID != protocol.ActionPlayerLogin {
		t.Errorf("期望 ActionID 为 0x%04X, 实际为 0x%04X", protocol.ActionPlayerLogin, pkt.ActionID)
	}

	name, tok, err := protocol.ParseLoginPacket(pkt.Payload)
	if err != nil {
		t.Fatalf("ParseLoginPacket 失败: %v", err)
	}
	if name != "梦一场" || tok != "mock-auth-token-123" {
		t.Errorf("解析登录封包不匹配: name=%s, token=%s", name, tok)
	}
}
