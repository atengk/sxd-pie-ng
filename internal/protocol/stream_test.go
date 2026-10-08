// Package protocol_test 验证流式网络 I/O 读写器、粘包半包容错及大包防御机制。
//
// @author Ateng
// @since 2026-10-08
package protocol_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"sxd-pie-ng/internal/protocol"
)

// chunkedReader 模拟极端碎片化网络传输（每次 Read 仅返回最多 chunkSize 字节）
type chunkedReader struct {
	r         io.Reader
	chunkSize int
}

func (c *chunkedReader) Read(p []byte) (n int, err error) {
	if len(p) > c.chunkSize {
		p = p[:c.chunkSize]
	}
	return c.r.Read(p)
}

func TestStream_StickyPackets(t *testing.T) {
	var buf bytes.Buffer

	p1 := protocol.NewPacket(101, []byte("Packet-1"))
	p2 := protocol.NewPacket(102, []byte("Packet-2-Longer-Content"))
	p3 := protocol.NewPacket(103, []byte{})

	if err := protocol.WritePacket(&buf, p1); err != nil {
		t.Fatalf("WritePacket p1 failed: %v", err)
	}
	if err := protocol.WritePacket(&buf, p2); err != nil {
		t.Fatalf("WritePacket p2 failed: %v", err)
	}
	if err := protocol.WritePacket(&buf, p3); err != nil {
		t.Fatalf("WritePacket p3 failed: %v", err)
	}

	// 连续流式读取 3 个封包（粘包处理）
	r1, err := protocol.ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket 1 failed: %v", err)
	}
	if r1.ActionID != 101 || !bytes.Equal(r1.Payload, p1.Payload) {
		t.Errorf("Packet 1 mismatch: %+v", r1)
	}

	r2, err := protocol.ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket 2 failed: %v", err)
	}
	if r2.ActionID != 102 || !bytes.Equal(r2.Payload, p2.Payload) {
		t.Errorf("Packet 2 mismatch: %+v", r2)
	}

	r3, err := protocol.ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket 3 failed: %v", err)
	}
	if r3.ActionID != 103 || len(r3.Payload) != 0 {
		t.Errorf("Packet 3 mismatch: %+v", r3)
	}

	// 流结束，应返回 io.EOF
	_, err = protocol.ReadPacket(&buf)
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF on empty buffer, got %v", err)
	}
}

func TestStream_HalfPackets(t *testing.T) {
	var buf bytes.Buffer
	original := protocol.NewPacket(505, []byte("Fragmented-TCP-Stream-Data-Payload"))
	if err := protocol.WritePacket(&buf, original); err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	// 每次调用只读 2 个字节，强制触发半包分片拼接
	slowReader := &chunkedReader{
		r:         bytes.NewReader(buf.Bytes()),
		chunkSize: 2,
	}

	readPkt, err := protocol.ReadPacket(slowReader)
	if err != nil {
		t.Fatalf("ReadPacket from slow reader failed: %v", err)
	}

	if readPkt.ActionID != original.ActionID {
		t.Errorf("expected ActionID %d, got %d", original.ActionID, readPkt.ActionID)
	}
	if !bytes.Equal(readPkt.Payload, original.Payload) {
		t.Errorf("expected payload %s, got %s", original.Payload, readPkt.Payload)
	}
}

func TestStream_OOMDefense_GiantPacket(t *testing.T) {
	// 构造伪造包头，声明 32MB 载荷，超出 16MB 安全红线
	header := make([]byte, protocol.HeaderSize)
	binary.BigEndian.PutUint32(header[0:4], 32*1024*1024)
	binary.BigEndian.PutUint16(header[4:6], 9999)

	r := bytes.NewReader(header)
	_, err := protocol.ReadPacket(r)
	if !errors.Is(err, protocol.ErrPacketTooLarge) {
		t.Errorf("expected ErrPacketTooLarge, got %v", err)
	}
}

func TestStream_TruncatedPacket(t *testing.T) {
	// 声明 10 字节载荷，但实际只提供 4 字节数据
	header := make([]byte, protocol.HeaderSize+4)
	binary.BigEndian.PutUint32(header[0:4], 10)
	binary.BigEndian.PutUint16(header[4:6], 8888)
	copy(header[protocol.HeaderSize:], []byte("1234"))

	r := bytes.NewReader(header)
	_, err := protocol.ReadPacket(r)
	if !errors.Is(err, protocol.ErrPayloadTruncated) {
		t.Errorf("expected ErrPayloadTruncated, got %v", err)
	}
}

func TestStream_WriteCompressedPacket(t *testing.T) {
	var buf bytes.Buffer
	data := bytes.Repeat([]byte("SXD-STREAM-TEST-"), 200)
	p := protocol.NewPacket(606, data)

	if err := protocol.WriteCompressedPacket(&buf, p); err != nil {
		t.Fatalf("WriteCompressedPacket failed: %v", err)
	}

	// 验证流式读取时自动透明解压
	readPkt, err := protocol.ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket compressed stream failed: %v", err)
	}

	if readPkt.ActionID != 606 {
		t.Errorf("expected ActionID 606, got %d", readPkt.ActionID)
	}
	if !bytes.Equal(readPkt.Payload, data) {
		t.Errorf("payload mismatch after compressed stream read")
	}
}
