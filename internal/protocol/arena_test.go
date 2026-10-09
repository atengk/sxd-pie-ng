// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-09
package protocol

import (
	"encoding/hex"
	"testing"
)

func TestArenaCodec_BuildAndParse(t *testing.T) {
	// 1. 测试 ArenaGetTimesRequest
	getTimesReq := ArenaGetTimesRequest{PrevAct: 0x00200000}
	pktTimes, err := BuildArenaGetTimesPacket(getTimesReq)
	if err != nil {
		t.Fatalf("BuildArenaGetTimesPacket 失败: %v", err)
	}
	if pktTimes.ActionID != ActionIDArenaGetTimes {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDArenaGetTimes, pktTimes.ActionID)
	}

	parsedTimes, err := ParseArenaGetTimesRequest(pktTimes.Payload)
	if err != nil {
		t.Fatalf("ParseArenaGetTimesRequest 失败: %v", err)
	}
	if parsedTimes.PrevAct != 0x00200000 {
		t.Errorf("PrevAct 不匹配: 期望 0x00200000, 实际 0x%08X", parsedTimes.PrevAct)
	}

	// 2. 测试 ArenaGetTimesResult 对齐真实抓包 Hex
	// 真实抓包: 00000005 (5 次剩余)
	resPkt, err := BuildArenaGetTimesResultPacket(5)
	if err != nil {
		t.Fatalf("BuildArenaGetTimesResultPacket 失败: %v", err)
	}
	if hex.EncodeToString(resPkt.Payload) != "00000005" {
		t.Errorf("剩余次数 Hex 不匹配: %s", hex.EncodeToString(resPkt.Payload))
	}
	parsedRes, err := ParseArenaGetTimesResult(resPkt.Payload)
	if err != nil {
		t.Fatalf("ParseArenaGetTimesResult 失败: %v", err)
	}
	if parsedRes.RemainingTimes != 5 {
		t.Errorf("RemainingTimes 不匹配: 期望 5, 实际 %d", parsedRes.RemainingTimes)
	}

	// 3. 测试 ArenaChallengeRequest 对齐真实抓包 Hex
	// 真实抓包: 00000005001c0001 (TargetRank=5, PrevAct=0x001C0001)
	challengeReq := ArenaChallengeRequest{
		TargetRank: 5,
		PrevAct:    0x001C0001,
	}
	pktChallenge, err := BuildArenaChallengePacket(challengeReq)
	if err != nil {
		t.Fatalf("BuildArenaChallengePacket 失败: %v", err)
	}
	if pktChallenge.ActionID != ActionIDArenaChallenge {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDArenaChallenge, pktChallenge.ActionID)
	}

	expectedHex := "00000005001c0001"
	actualHex := hex.EncodeToString(pktChallenge.Payload)
	if actualHex != expectedHex {
		t.Errorf("挑战载荷 Hex 不匹配: 期望 %s, 实际 %s", expectedHex, actualHex)
	}

	parsedChallenge, err := ParseArenaChallengeRequest(pktChallenge.Payload)
	if err != nil {
		t.Fatalf("ParseArenaChallengeRequest 失败: %v", err)
	}
	if parsedChallenge.TargetRank != 5 || parsedChallenge.PrevAct != 0x001C0001 {
		t.Errorf("挑战解析结果不符: %+v", parsedChallenge)
	}
}
