// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-08
package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// Reader 提供基于大端序的应用层载荷流式读取工具。
type Reader struct {
	buf []byte
	pos int
}

// NewReader 基于给定的载荷字节切片创建 Reader。
func NewReader(buf []byte) *Reader {
	if buf == nil {
		buf = []byte{}
	}
	return &Reader{buf: buf, pos: 0}
}

// Remaining 返回尚未读取的剩余字节数量。
func (r *Reader) Remaining() int {
	return len(r.buf) - r.pos
}

func (r *Reader) require(n int) error {
	if r.Remaining() < n {
		return io.ErrUnexpectedEOF
	}
	return nil
}

// ReadUint8 读取 1 字节无符号整数。
func (r *Reader) ReadUint8() (uint8, error) {
	if err := r.require(1); err != nil {
		return 0, err
	}
	v := r.buf[r.pos]
	r.pos++
	return v, nil
}

// ReadUint16 读取 2 字节大端序无符号整数。
func (r *Reader) ReadUint16() (uint16, error) {
	if err := r.require(2); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint16(r.buf[r.pos : r.pos+2])
	r.pos += 2
	return v, nil
}

// ReadUint32 读取 4 字节大端序无符号整数。
func (r *Reader) ReadUint32() (uint32, error) {
	if err := r.require(4); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint32(r.buf[r.pos : r.pos+4])
	r.pos += 4
	return v, nil
}

// ReadInt32 读取 4 字节大端序有符号整数。
func (r *Reader) ReadInt32() (int32, error) {
	u, err := r.ReadUint32()
	if err != nil {
		return 0, err
	}
	return int32(u), nil
}

// ReadInt64 读取 8 字节大端序有符号整数。
func (r *Reader) ReadInt64() (int64, error) {
	if err := r.require(8); err != nil {
		return 0, err
	}
	v := int64(binary.BigEndian.Uint64(r.buf[r.pos : r.pos+8]))
	r.pos += 8
	return v, nil
}

// ReadString 读取 2 字节大端序长度前缀的 UTF-8 字符串。
func (r *Reader) ReadString() (string, error) {
	length, err := r.ReadUint16()
	if err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}
	if err := r.require(int(length)); err != nil {
		return "", err
	}
	str := string(r.buf[r.pos : r.pos+int(length)])
	r.pos += int(length)
	return str, nil
}

// ReadBytes 读取指定长度的裸字节切片。
func (r *Reader) ReadBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, errors.New("protocol: negative byte length")
	}
	if n == 0 {
		return []byte{}, nil
	}
	if err := r.require(n); err != nil {
		return nil, err
	}
	dst := make([]byte, n)
	copy(dst, r.buf[r.pos:r.pos+n])
	r.pos += n
	return dst, nil
}

// Writer 提供基于大端序的应用层载荷流式组装工具。
type Writer struct {
	buf bytes.Buffer
}

// NewWriter 创建一个新的 Writer 实例。
func NewWriter() *Writer {
	return &Writer{}
}

// WriteUint8 写入 1 字节无符号整数。
func (w *Writer) WriteUint8(v uint8) *Writer {
	w.buf.WriteByte(v)
	return w
}

// WriteUint16 写入 2 字节大端序无符号整数。
func (w *Writer) WriteUint16(v uint16) *Writer {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], v)
	w.buf.Write(b[:])
	return w
}

// WriteUint32 写入 4 字节大端序无符号整数。
func (w *Writer) WriteUint32(v uint32) *Writer {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	w.buf.Write(b[:])
	return w
}

// WriteInt32 写入 4 字节大端序有符号整数。
func (w *Writer) WriteInt32(v int32) *Writer {
	return w.WriteUint32(uint32(v))
}

// WriteInt64 写入 8 字节大端序有符号整数。
func (w *Writer) WriteInt64(v int64) *Writer {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(v))
	w.buf.Write(b[:])
	return w
}

// WriteString 写入带有 2 字节大端序长度前缀的 UTF-8 字符串。
func (w *Writer) WriteString(s string) *Writer {
	data := []byte(s)
	w.WriteUint16(uint16(len(data)))
	w.buf.Write(data)
	return w
}

// WriteBytes 写入裸字节切片。
func (w *Writer) WriteBytes(b []byte) *Writer {
	w.buf.Write(b)
	return w
}

// Bytes 返回当前已组装的完整字节切片。
func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}
