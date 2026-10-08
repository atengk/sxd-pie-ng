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
	actionID := uint16(1001)
	payload := []byte("hello sxd-pie-ng")

	pkt := protocol.NewPacket(actionID, payload)
	wire, err := pkt.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证 6 字节固定包头: [4B Length] + [2B ActionID]
	expectedLength := uint32(len(payload))
	actualLength := binary.BigEndian.Uint32(wire[0:4])
	actualActionID := binary.BigEndian.Uint16(wire[4:6])

	if actualLength != expectedLength {
		t.Errorf("expected length %d, got %d", expectedLength, actualLength)
	}
	if actualActionID != actionID {
		t.Errorf("expected action id %d, got %d", actionID, actualActionID)
	}
	if !bytes.Equal(wire[6:], payload) {
		t.Errorf("expected payload %s, got %s", payload, wire[6:])
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

func TestPacket_Unmarshal_Errors(t *testing.T) {
	// 1. 数据短于 HeaderSize
	_, err := protocol.Unmarshal([]byte{0x00, 0x01})
	if err != protocol.ErrPacketTooShort {
		t.Errorf("expected ErrPacketTooShort, got %v", err)
	}

	// 2. 声明长度超过 MaxPayloadSize
	tooLarge := make([]byte, protocol.HeaderSize)
	binary.BigEndian.PutUint32(tooLarge[0:4], 32*1024*1024)
	_, err = protocol.Unmarshal(tooLarge)
	if err == nil {
		t.Error("expected error for payload exceeding MaxPayloadSize")
	}

	// 3. 数据被截断
	truncated := make([]byte, protocol.HeaderSize+2)
	binary.BigEndian.PutUint32(truncated[0:4], 10)
	_, err = protocol.Unmarshal(truncated)
	if err != protocol.ErrPayloadTruncated {
		t.Errorf("expected ErrPayloadTruncated, got %v", err)
	}
}
