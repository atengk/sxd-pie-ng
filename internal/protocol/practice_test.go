// Package protocol_test 验证副本挂机扫荡与加速秒完成协议封包编解码。
//
// @author Ateng
// @since 2026-10-08
package protocol_test

import (
	"testing"

	"sxd-pie-ng/internal/protocol"
)

func TestPractice_StartPracticeRoundtrip(t *testing.T) {
	req := protocol.StartPracticeRequest{
		MissionID: 105,
		Count:     10,
		AutoSale:  1,
	}

	pkt, err := protocol.BuildStartPracticePacket(req)
	if err != nil {
		t.Fatalf("BuildStartPracticePacket failed: %v", err)
	}

	if pkt.ActionID != protocol.ActionIDPracticeStart {
		t.Errorf("expected ActionID 0x%04X, got 0x%04X", protocol.ActionIDPracticeStart, pkt.ActionID)
	}

	parsed, err := protocol.ParseStartPracticeRequest(pkt.Payload)
	if err != nil {
		t.Fatalf("ParseStartPracticeRequest failed: %v", err)
	}

	if parsed.MissionID != req.MissionID || parsed.Count != req.Count || parsed.AutoSale != req.AutoSale {
		t.Errorf("expected %+v, got %+v", req, *parsed)
	}

	// 验证结果回包编解码
	res := protocol.StartPracticeResult{
		Result:    protocol.PracticeResultSuccess,
		MissionID: 105,
		Count:     10,
	}

	resPkt, err := protocol.BuildStartPracticeResultPacket(res)
	if err != nil {
		t.Fatalf("BuildStartPracticeResultPacket failed: %v", err)
	}

	parsedRes, err := protocol.ParseStartPracticeResult(resPkt.Payload)
	if err != nil {
		t.Fatalf("ParseStartPracticeResult failed: %v", err)
	}

	if *parsedRes != res {
		t.Errorf("expected %+v, got %+v", res, *parsedRes)
	}
}

func TestPractice_QuicklyRoundtrip(t *testing.T) {
	req := protocol.QuicklyRequest{
		Count: 5,
	}

	pkt, err := protocol.BuildQuicklyPacket(req)
	if err != nil {
		t.Fatalf("BuildQuicklyPacket failed: %v", err)
	}

	if pkt.ActionID != protocol.ActionIDPracticeQuickly {
		t.Errorf("expected ActionID 0x%04X, got 0x%04X", protocol.ActionIDPracticeQuickly, pkt.ActionID)
	}

	parsed, err := protocol.ParseQuicklyRequest(pkt.Payload)
	if err != nil {
		t.Fatalf("ParseQuicklyRequest failed: %v", err)
	}

	if parsed.Count != 5 {
		t.Errorf("expected Count 5, got %d", parsed.Count)
	}

	res := protocol.QuicklyResult{
		Result: protocol.PracticeResultSuccess,
		Count:  5,
	}

	resPkt, err := protocol.BuildQuicklyResultPacket(res)
	if err != nil {
		t.Fatalf("BuildQuicklyResultPacket failed: %v", err)
	}

	parsedRes, err := protocol.ParseQuicklyResult(resPkt.Payload)
	if err != nil {
		t.Fatalf("ParseQuicklyResult failed: %v", err)
	}

	if *parsedRes != res {
		t.Errorf("expected %+v, got %+v", res, *parsedRes)
	}
}
