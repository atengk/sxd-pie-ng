// Package protocol_test 验证透明 zlib 解压缩、压缩及非压缩魔数误判自愈能力。
//
// @author Ateng
// @since 2026-10-08
package protocol_test

import (
	"bytes"
	"strings"
	"testing"

	"sxd-pie-ng/internal/protocol"
)

func TestZlib_CompressDecompress_Transparent(t *testing.T) {
	actionID := uint32(2001)
	// 构造具有良好压缩比的文本数据
	largeText := strings.Repeat("ShenXianDao-Auto-Helper-Next-Generation-2026;", 100)
	payload := []byte(largeText)

	pkt := protocol.NewPacket(actionID, payload)
	wire, err := pkt.MarshalCompressed()
	if err != nil {
		t.Fatalf("MarshalCompressed failed: %v", err)
	}

	// 验证载荷压缩后以 0x78 魔数开头
	if wire[protocol.HeaderSize] != protocol.ZlibMagicByte {
		t.Fatalf("expected compressed payload to start with 0x%x, got 0x%x",
			protocol.ZlibMagicByte, wire[protocol.HeaderSize])
	}

	// 验证透明反序列化：调用方无感知底层是否经过 zlib 压缩
	decoded, err := protocol.Unmarshal(wire)
	if err != nil {
		t.Fatalf("Unmarshal compressed packet failed: %v", err)
	}

	if decoded.ActionID != actionID {
		t.Errorf("expected action id %d, got %d", actionID, decoded.ActionID)
	}
	if !bytes.Equal(decoded.Payload, payload) {
		t.Errorf("decompressed payload mismatch! len expected %d, got %d", len(payload), len(decoded.Payload))
	}
}

func TestZlib_SelfHealing_FalsePositiveMagic(t *testing.T) {
	actionID := uint32(3001)
	// 首字节恰好为 0x78，但其余字节为任意非 zlib 业务二进制数据
	falsePositivePayload := []byte{0x78, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}

	pkt := protocol.NewPacket(actionID, falsePositivePayload)
	wire, err := pkt.Marshal() // 未压缩
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 反序列化时应当能够嗅探到 0x78，尝试解压失败后自动自愈降级为原始载荷
	decoded, err := protocol.Unmarshal(wire)
	if err != nil {
		t.Fatalf("Unmarshal should not fail on false positive 0x78: %v", err)
	}

	if decoded.ActionID != actionID {
		t.Errorf("expected action id %d, got %d", actionID, decoded.ActionID)
	}
	if !bytes.Equal(decoded.Payload, falsePositivePayload) {
		t.Errorf("expected self-healed payload %v, got %v", falsePositivePayload, decoded.Payload)
	}
}

func TestZlib_SelfHealing_TruncatedStream(t *testing.T) {
	actionID := uint32(4001)
	// 合法的 zlib 头（0x78 0x9c），但后置流被截断
	truncatedZlibPayload := []byte{0x78, 0x9C, 0x00}

	pkt := protocol.NewPacket(actionID, truncatedZlibPayload)
	wire, err := pkt.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	decoded, err := protocol.Unmarshal(wire)
	if err != nil {
		t.Fatalf("Unmarshal should self-heal on truncated zlib stream: %v", err)
	}

	if !bytes.Equal(decoded.Payload, truncatedZlibPayload) {
		t.Errorf("expected self-healed truncated payload %v, got %v", truncatedZlibPayload, decoded.Payload)
	}
}

func TestZlib_EmptyAndUncompressed(t *testing.T) {
	// 1. 空载荷压缩
	comp, err := protocol.Compress(nil)
	if err != nil || len(comp) != 0 {
		t.Fatalf("expected empty compressed bytes, got %v", comp)
	}

	// 2. 空载荷解压
	decomp, err := protocol.DecompressIfNeeded(nil)
	if err != nil || len(decomp) != 0 {
		t.Fatalf("expected empty decompressed bytes, got %v", decomp)
	}

	// 3. 非 0x78 开头的未压缩数据
	plain := []byte("plain text not compressed")
	decompPlain, err := protocol.DecompressIfNeeded(plain)
	if err != nil || !bytes.Equal(decompPlain, plain) {
		t.Fatalf("expected plain bytes untouched, got %v", decompPlain)
	}
}
