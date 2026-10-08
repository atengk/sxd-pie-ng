// Package protocol_test 验证跨服/网页登录协议封包编解码与回包解析。
//
// @author Ateng
// @since 2026-10-08
package protocol_test

import (
	"testing"

	"sxd-pie-ng/internal/protocol"
)

func TestStLogin_Roundtrip(t *testing.T) {
	req := protocol.StLoginRequest{
		ServerID:   "fengwanyx_s813",
		ClientType: 4,
		RoleName:   "梦一场",
		Time1:      1791469000,
		Hash1:      "5f4dcc3b5aa765d61d8327deb882cf99",
	}

	pkt, err := protocol.BuildStLoginPacket(req)
	if err != nil {
		t.Fatalf("BuildStLoginPacket failed: %v", err)
	}

	if pkt.ActionID != protocol.ActionIDStLogin {
		t.Errorf("expected ActionID 0x%04X, got 0x%04X", protocol.ActionIDStLogin, pkt.ActionID)
	}

	parsed, err := protocol.ParseStLoginRequest(pkt.Payload)
	if err != nil {
		t.Fatalf("ParseStLoginRequest failed: %v", err)
	}

	if *parsed != req {
		t.Errorf("expected %+v, got %+v", req, *parsed)
	}

	// 验证登录响应封包 Roundtrip
	res := protocol.StLoginResult{
		Result:     0,
		PlayerID:   65536,
		ServerTime: 1791469358,
	}

	resPkt, err := protocol.BuildStLoginResultPacket(res)
	if err != nil {
		t.Fatalf("BuildStLoginResultPacket failed: %v", err)
	}

	parsedRes, err := protocol.ParseStLoginResult(resPkt.Payload)
	if err != nil {
		t.Fatalf("ParseStLoginResult failed: %v", err)
	}

	if *parsedRes != res {
		t.Errorf("expected %+v, got %+v", res, *parsedRes)
	}
}
