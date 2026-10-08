// Package protocol_test 验证应用层载荷二进制大端序读写器。
//
// @author Ateng
// @since 2026-10-08
package protocol_test

import (
	"bytes"
	"testing"

	"sxd-pie-ng/internal/protocol"
)

func TestPayload_WriterAndReader_Roundtrip(t *testing.T) {
	w := protocol.NewWriter()
	w.WriteUint8(42)
	w.WriteUint16(10086)
	w.WriteUint32(12345678)
	w.WriteInt32(-987654)
	w.WriteString("神仙道助手-sxd-pie-ng")
	rawBytes := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	w.WriteBytes(rawBytes)

	data := w.Bytes()

	r := protocol.NewReader(data)

	u8, err := r.ReadUint8()
	if err != nil || u8 != 42 {
		t.Fatalf("ReadUint8 failed: got %d, err %v", u8, err)
	}

	u16, err := r.ReadUint16()
	if err != nil || u16 != 10086 {
		t.Fatalf("ReadUint16 failed: got %d, err %v", u16, err)
	}

	u32, err := r.ReadUint32()
	if err != nil || u32 != 12345678 {
		t.Fatalf("ReadUint32 failed: got %d, err %v", u32, err)
	}

	i32, err := r.ReadInt32()
	if err != nil || i32 != -987654 {
		t.Fatalf("ReadInt32 failed: got %d, err %v", i32, err)
	}

	str, err := r.ReadString()
	if err != nil || str != "神仙道助手-sxd-pie-ng" {
		t.Fatalf("ReadString failed: got %s, err %v", str, err)
	}

	readBytes, err := r.ReadBytes(len(rawBytes))
	if err != nil || !bytes.Equal(readBytes, rawBytes) {
		t.Fatalf("ReadBytes failed: got %x, err %v", readBytes, err)
	}

	if r.Remaining() != 0 {
		t.Errorf("expected 0 remaining bytes, got %d", r.Remaining())
	}
}

func TestPayloadReader_ShortBuffer(t *testing.T) {
	r := protocol.NewReader([]byte{0x01})
	_, err := r.ReadUint16()
	if err == nil {
		t.Fatal("expected error reading uint16 from 1-byte buffer")
	}

	_, err = r.ReadUint32()
	if err == nil {
		t.Fatal("expected error reading uint32 from 1-byte buffer")
	}

	_, err = r.ReadInt64()
	if err == nil {
		t.Fatal("expected error reading int64 from 1-byte buffer")
	}

	_, err = r.ReadBytes(5)
	if err == nil {
		t.Fatal("expected error reading 5 bytes from 1-byte buffer")
	}

	_, err = r.ReadBytes(-1)
	if err == nil {
		t.Fatal("expected error reading negative bytes")
	}
}

func TestPayload_Int64AndEmpty(t *testing.T) {
	w := protocol.NewWriter()
	w.WriteInt64(-9223372036854775807)
	w.WriteString("")
	w.WriteBytes(nil)

	r := protocol.NewReader(w.Bytes())
	i64, err := r.ReadInt64()
	if err != nil || i64 != -9223372036854775807 {
		t.Fatalf("ReadInt64 failed: got %d, err %v", i64, err)
	}

	s, err := r.ReadString()
	if err != nil || s != "" {
		t.Fatalf("ReadString empty failed: got %s, err %v", s, err)
	}

	b, err := r.ReadBytes(0)
	if err != nil || len(b) != 0 {
		t.Fatalf("ReadBytes(0) failed: got %v, err %v", b, err)
	}
}
