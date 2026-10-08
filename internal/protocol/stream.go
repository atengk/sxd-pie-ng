// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-08
package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ReadPacket 从流式 io.Reader 中读取下一个完整封包。
// 内部使用 io.ReadFull 自动防御网络半包分片，并支持透明解压。
// 当对端正常关闭连接且未读取任何新字节时，返回 io.EOF。
func ReadPacket(r io.Reader) (*Packet, error) {
	headerBuf := make([]byte, HeaderSize)
	_, err := io.ReadFull(r, headerBuf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrPacketTooShort
		}
		return nil, err
	}

	payloadLen := binary.BigEndian.Uint32(headerBuf[0:4])
	if payloadLen > MaxPayloadSize {
		return nil, fmt.Errorf("%w: length %d > %d", ErrPacketTooLarge, payloadLen, MaxPayloadSize)
	}

	actionID := binary.BigEndian.Uint16(headerBuf[4:6])

	if payloadLen == 0 {
		return &Packet{
			ActionID: actionID,
			Payload:  []byte{},
		}, nil
	}

	rawPayload := make([]byte, payloadLen)
	_, err = io.ReadFull(r, rawPayload)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return nil, ErrPayloadTruncated
		}
		return nil, err
	}

	payload, err := DecompressIfNeeded(rawPayload)
	if err != nil {
		return nil, err
	}

	return &Packet{
		ActionID: actionID,
		Payload:  payload,
	}, nil
}

// WritePacket 将封包按未压缩线格式完整写入 io.Writer。
func WritePacket(w io.Writer, p *Packet) error {
	data, err := p.Marshal()
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// WriteCompressedPacket 将封包载荷经 zlib 压缩后完整写入 io.Writer。
func WriteCompressedPacket(w io.Writer, p *Packet) error {
	data, err := p.MarshalCompressed()
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
