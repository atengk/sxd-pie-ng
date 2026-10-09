// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-09
package protocol

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

var (
	// ErrPlayerLoginPayloadTooShort 登录回包载荷长度不足
	ErrPlayerLoginPayloadTooShort = errors.New("protocol: player login payload too short")
)

// PlayerLoginRequest 代表角色主服登录请求协议模型 (Mod_Player_Base.login, ActionID=0x0000)。
type PlayerLoginRequest struct {
	// Username 平台登录账号
	Username string
	// Hash 平台登录动态签名散列
	Hash string
	// Time 平台登录时间戳 (字符串形式)
	Time string
	// Source 渠道来源标识 (默认 "sxd_baidu_pinpai_bt")
	Source string
	// Platform 平台显示名称 (如 "疯玩")
	Platform string
	// ClientType 客户端接入类型 (默认 "web")
	ClientType string
}

// PlayerLoginResult 代表角色主服登录成功后服务端返回的核心角色属性快照。
type PlayerLoginResult struct {
	// Result 状态结果码 (0=成功)
	Result int16
	// RoleID 角色在当前区服的主键序号
	RoleID int16
	// RoleName 角色名 (UTF-8)
	RoleName string
	// Level 角色当前等级
	Level int32
	// Ingots 当前元宝数量
	Ingots int32
	// Coins 当前铜钱数量
	Coins int64
	// Stamina 当前体力点数
	Stamina int32
	// MaxStamina 最大体力上限
	MaxStamina int32
}

// BuildPlayerLoginPacket 构造角色主服登录请求二进制协议封包 (ActionID 0x0000)。
//
// @param req 主服登录请求参数模型
// @return 编码好的封包实例或错误
func BuildPlayerLoginPacket(req PlayerLoginRequest) (*Packet, error) {
	if req.Username == "" {
		return nil, errors.New("protocol: username cannot be empty for player login")
	}
	if req.Source == "" {
		req.Source = "sxd_baidu_pinpai_bt"
	}
	if req.Platform == "" {
		req.Platform = "疯玩"
	}
	if req.ClientType == "" {
		req.ClientType = "web"
	}

	w := NewWriter()
	// 1. 保留前缀整型
	w.WriteInt16(0)
	// 2. 账号用户名
	w.WriteString(req.Username)
	// 3. 动态验签哈希
	w.WriteString(req.Hash)
	// 4. 动态验签时间戳
	w.WriteString(req.Time)
	// 5. 渠道标识
	w.WriteString(req.Source)
	// 6. 空占位字符串 * 3
	w.WriteString("")
	w.WriteString("")
	w.WriteString("")
	// 7. 客户端固定参数
	w.WriteInt16(0x62B4)
	w.WriteInt16(0x0160)
	w.WriteInt8(0)
	// 8. 平台名称
	w.WriteString(req.Platform)
	// 9. 接入协议类型
	w.WriteString(req.ClientType)
	// 10. 标志位与尾部账号
	w.WriteInt8(1)
	w.WriteString(req.Username)

	return NewPacket(ActionIDPlayerLogin, w.Bytes()), nil
}

// ParsePlayerLoginResult 解析主服登录响应载荷并提取核心角色状态数据。
// 若载荷首字节命中 zlib 魔数 (0x78)，将透明进行流式解压后再做结构反序列化。
//
// @param payload 原始封包载荷切片
// @return 解析出的角色核心数据模型或错误
func ParsePlayerLoginResult(payload []byte) (*PlayerLoginResult, error) {
	if len(payload) == 0 {
		return nil, ErrPlayerLoginPayloadTooShort
	}

	body := payload
	if body[0] == ZlibMagicByte {
		r, err := zlib.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("protocol: failed to decompress player login response: %w", err)
		}
		defer r.Close()

		decompressed, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("protocol: failed to read decompressed player login response: %w", err)
		}
		body = decompressed
	}

	if len(body) < 40 {
		return nil, ErrPlayerLoginPayloadTooShort
	}

	offset := tryDetectOffset(body)

	rd := bytes.NewReader(body)
	res := &PlayerLoginResult{
		MaxStamina: 300, // 满级角色基准体力上限
	}

	if offset == 2 {
		if err := binary.Read(rd, binary.BigEndian, &res.Result); err != nil {
			return nil, err
		}
	}

	// 1. RoleID (2B)
	if err := binary.Read(rd, binary.BigEndian, &res.RoleID); err != nil {
		return nil, err
	}

	// 2. RoleName (2B 长度 + UTF-8 字符串)
	var nameLen uint16
	if err := binary.Read(rd, binary.BigEndian, &nameLen); err != nil {
		return nil, err
	}
	if int(nameLen) > rd.Len() {
		return nil, ErrPlayerLoginPayloadTooShort
	}
	nameBytes := make([]byte, nameLen)
	if _, err := io.ReadFull(rd, nameBytes); err != nil {
		return nil, err
	}
	res.RoleName = string(nameBytes)

	// 3. Level (4B int32)
	if err := binary.Read(rd, binary.BigEndian, &res.Level); err != nil {
		return nil, err
	}
	if res.Level > 0 {
		res.MaxStamina = res.Level
	}

	// 4. Ingots (4B int32)
	if err := binary.Read(rd, binary.BigEndian, &res.Ingots); err != nil {
		return nil, err
	}

	// 5. Coins (8B int64)
	if err := binary.Read(rd, binary.BigEndian, &res.Coins); err != nil {
		return nil, err
	}

	// 6. 跳过中间 16 字节 (val1 8B + val2 8B)
	if _, err := rd.Seek(16, io.SeekCurrent); err != nil {
		return nil, err
	}

	// 7. Stamina (4B int32)
	if err := binary.Read(rd, binary.BigEndian, &res.Stamina); err != nil {
		return nil, err
	}

	return res, nil
}

// tryDetectOffset 智能嗅探载荷是否带有 2 字节 Result/ActionID 前缀。
// 返回 0 表示无前缀 (直接以 RoleID 开始)，返回 2 表示含有 2 字节前缀。
func tryDetectOffset(body []byte) int {
	check := func(start int) bool {
		if len(body) < start+8 {
			return false
		}
		nameLen := int(binary.BigEndian.Uint16(body[start+2 : start+4]))
		if nameLen <= 0 || nameLen > 30 || start+4+nameLen+36 > len(body) {
			return false
		}
		nameBytes := body[start+4 : start+4+nameLen]
		for _, b := range nameBytes {
			if b < 32 {
				return false
			}
		}
		if !utf8.Valid(nameBytes) {
			return false
		}
		lvl := int32(binary.BigEndian.Uint32(body[start+4+nameLen : start+4+nameLen+4]))
		return lvl >= 1 && lvl <= 500
	}

	if check(0) {
		return 0
	}
	if check(2) {
		return 2
	}
	return 0
}

