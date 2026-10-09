// Package protocol_test 验证数据封包基础线格式序列化与反序列化逻辑。
//
// @author Ateng
// @since 2026-10-08
package protocol_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"sxd-pie-ng/internal/protocol"
)

func TestPacket_MarshalUnmarshal_Raw(t *testing.T) {
	actionID := uint32(1001)
	payload := []byte("hello sxd-pie-ng")

	pkt := protocol.NewPacket(actionID, payload)
	wire, err := pkt.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证 8 字节固定包头: [4B Length] + [4B ActionID]
	// 真实协议中 4 字节长度包含 ActionIDSize(4) + 载荷长度
	expectedLength := uint32(protocol.ActionIDSize + len(payload))
	actualLength := binary.BigEndian.Uint32(wire[0:4])
	actualActionID := binary.BigEndian.Uint32(wire[4:8])

	if actualLength != expectedLength {
		t.Errorf("expected length %d, got %d", expectedLength, actualLength)
	}
	if actualActionID != actionID {
		t.Errorf("expected action id %d, got %d", actionID, actualActionID)
	}
	if !bytes.Equal(wire[8:], payload) {
		t.Errorf("expected payload %s, got %s", payload, wire[8:])
	}

	// 验证反序列化
	decoded, err := protocol.Unmarshal(wire)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.ActionID != actionID {
		t.Errorf("expected decoded action id %d, got %d", actionID, decoded.ActionID)
	}
	if !bytes.Equal(decoded.Payload, payload) {
		t.Errorf("expected decoded payload %s, got %s", payload, decoded.Payload)
	}
}

func TestPacket_EmptyPayload(t *testing.T) {
	pkt := protocol.NewPacket(2002, nil)
	wire, err := pkt.Marshal()
	if err != nil {
		t.Fatalf("Marshal empty payload failed: %v", err)
	}

	if len(wire) != protocol.HeaderSize {
		t.Fatalf("expected wire size %d, got %d", protocol.HeaderSize, len(wire))
	}

	actualLength := binary.BigEndian.Uint32(wire[0:4])
	if actualLength != protocol.ActionIDSize {
		t.Errorf("expected empty payload length %d, got %d", protocol.ActionIDSize, actualLength)
	}

	decoded, err := protocol.Unmarshal(wire)
	if err != nil {
		t.Fatalf("Unmarshal empty payload failed: %v", err)
	}
	if decoded.ActionID != 2002 {
		t.Errorf("expected action id 2002, got %d", decoded.ActionID)
	}
	if len(decoded.Payload) != 0 {
		t.Errorf("expected empty payload, got %d bytes", len(decoded.Payload))
	}
}

func TestPacket_ActionID_Coding(t *testing.T) {
	// Module 94 (0x5E), Action 0 -> 0x005E0000
	aidStLogin := protocol.MakeActionID(94, 0)
	if aidStLogin != 0x005E0000 {
		t.Errorf("expected 0x005E0000, got 0x%08X", aidStLogin)
	}
	mod, act := protocol.SplitActionID(aidStLogin)
	if mod != 94 || act != 0 {
		t.Errorf("expected mod 94 act 0, got mod %d act %d", mod, act)
	}

	// Module 25 (0x19), Action 1 -> 0x00190001
	aidSweep := protocol.MakeActionID(25, 1)
	if aidSweep != 0x00190001 {
		t.Errorf("expected 0x00190001, got 0x%08X", aidSweep)
	}
	mod, act = protocol.SplitActionID(aidSweep)
	if mod != 25 || act != 1 {
		t.Errorf("expected mod 25 act 1, got mod %d act %d", mod, act)
	}
}

func TestPacket_LiveServerPacketHex(t *testing.T) {
	// 实测真实游戏服务器返回的 8 字节包头报文:
	// Length = 15 (4 ActionID + 11 Payload)
	// ActionID = 0x005E0000
	rawHex := []byte{
		0x00, 0x00, 0x00, 0x0F, // Length = 15 (4 ActionID + 11 Payload)
		0x00, 0x5E, 0x00, 0x00, // ActionID = 0x005E0000
		0x00,                   // result = 0 (SUCCESS)
		0x00, 0x01, 0x00, 0x00, // player_id = 65536
		0x00, 0x00, 0x6a, 0xc7, 0xa7, 0x2e, // timestamp
	}

	pkt, err := protocol.Unmarshal(rawHex)
	if err != nil {
		t.Fatalf("Unmarshal live server packet failed: %v", err)
	}
	if pkt.ActionID != 0x005E0000 {
		t.Errorf("expected ActionID 0x005E0000, got 0x%08X", pkt.ActionID)
	}
	if len(pkt.Payload) != 11 {
		t.Fatalf("expected payload length 11, got %d", len(pkt.Payload))
	}
	if pkt.Payload[0] != 0x00 {
		t.Errorf("expected result 0, got %d", pkt.Payload[0])
	}
}

func TestPacket_Unmarshal_Errors(t *testing.T) {
	// 1. 数据短于 HeaderSize
	_, err := protocol.Unmarshal([]byte{0x00, 0x01})
	if err != protocol.ErrPacketTooShort {
		t.Errorf("expected ErrPacketTooShort, got %v", err)
	}

	// 2. 声明长度小于 ActionIDSize (2)
	invalidLen := []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x01}
	_, err = protocol.Unmarshal(invalidLen)
	if err != protocol.ErrPacketTooShort {
		t.Errorf("expected ErrPacketTooShort for bodyLen < 2, got %v", err)
	}

	// 3. 声明长度超过 MaxPayloadSize
	tooLarge := make([]byte, protocol.HeaderSize)
	binary.BigEndian.PutUint32(tooLarge[0:4], 32*1024*1024)
	_, err = protocol.Unmarshal(tooLarge)
	if err == nil {
		t.Error("expected error for payload exceeding MaxPayloadSize")
	}

	// 4. 数据被截断
	truncated := make([]byte, protocol.HeaderSize+2)
	binary.BigEndian.PutUint32(truncated[0:4], 10)
	_, err = protocol.Unmarshal(truncated)
	if err != protocol.ErrPayloadTruncated {
		t.Errorf("expected ErrPayloadTruncated, got %v", err)
	}
}
