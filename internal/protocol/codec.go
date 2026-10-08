// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-08
package protocol

import (
	"bytes"
	"compress/zlib"
	"io"
)

// ZlibMagicByte zlib 标准头部压缩方法与标志字节 (CMF，通常为 0x78)
const ZlibMagicByte byte = 0x78

// Compress 将载荷数据通过 zlib 算法压缩。
func Compress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecompressIfNeeded 动态嗅探载荷首字节是否为 0x78 魔数。
// 若命中魔数，则执行流式 zlib 解压缩；若解压缩因损坏或误判失败，则具备自愈能力（Fallback 返回原切片）。
func DecompressIfNeeded(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	// 嗅探首字节是否为 0x78
	if src[0] != ZlibMagicByte {
		// 未压缩，直接原样透传
		dst := make([]byte, len(src))
		copy(dst, src)
		return dst, nil
	}

	// 尝试 zlib 解压
	r, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		// 首字节为 0x78 但非合法 zlib 头，自愈降级为未压缩原始数据
		dst := make([]byte, len(src))
		copy(dst, src)
		return dst, nil
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		// 流式解压中途校验失败或中断，自愈降级为未压缩原始数据
		dst := make([]byte, len(src))
		copy(dst, src)
		return dst, nil
	}

	return decompressed, nil
}
