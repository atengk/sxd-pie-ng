// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-08
package protocol

import (
	"fmt"
)

// StLoginRequest 代表跨服/网页端登录请求参数 (Mod_StLogin_Base.login, ActionID=0x005E)。
type StLoginRequest struct {
	// ServerID 区服标识符 (如 "fengwanyx_s813")，按 4 字节前缀存储
	ServerID string
	// ClientType 客户端类型 (微端/Web 对应数值 4)
	ClientType int32
	// RoleName 游戏角色名称 (如 "梦一场")，按 2 字节前缀存储
	RoleName string
	// Time1 动态登录时间戳 (来源于网关 Cookie)
	Time1 int32
	// Hash1 动态验签 MD5 散列 (来源于网关 Cookie)
	Hash1 string
}

// StLoginResult 代表跨服/网页端登录服务器回包结果。
type StLoginResult struct {
	// Result 登录结果状态码 (0=SUCCESS, 1=FAILED)
	Result uint8
	// PlayerID 角色在当前区服的数字标识 (如 65536)
	PlayerID int32
	// ServerTime 服务器当前 Unix 时间戳
	ServerTime int32
}

// BuildStLoginPacket 构造跨服登录请求二进制协议封包。
func BuildStLoginPacket(req StLoginRequest) (*Packet, error) {
	if req.ClientType == 0 {
		req.ClientType = 4
	}

	w := NewWriter()
	// 1. 写入 ServerID (4 字节长度前缀)
	w.WriteString32(req.ServerID)
	// 2. 写入 ClientType (4 字节大端序整型)
	w.WriteInt32(req.ClientType)
	// 3. 写入 RoleName (2 字节长度前缀)
	w.WriteString(req.RoleName)
	// 4. 写入 Time1 (4 字节大端序整型)
	w.WriteInt32(req.Time1)
	// 5. 写入 Hash1 (2 字节长度前缀)
	w.WriteString(req.Hash1)

	return NewPacket(ActionIDStLogin, w.Bytes()), nil
}

// ParseStLoginRequest 从封包载荷中反序列化跨服登录请求参数。
func ParseStLoginRequest(payload []byte) (*StLoginRequest, error) {
	r := NewReader(payload)

	serverID, err := r.ReadString32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read server_id: %v", ErrInvalidPayload, err)
	}

	clientType, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read client_type: %v", ErrInvalidPayload, err)
	}

	roleName, err := r.ReadString()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read role_name: %v", ErrInvalidPayload, err)
	}

	time1, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read time1: %v", ErrInvalidPayload, err)
	}

	hash1, err := r.ReadString()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read hash1: %v", ErrInvalidPayload, err)
	}

	return &StLoginRequest{
		ServerID:   serverID,
		ClientType: clientType,
		RoleName:   roleName,
		Time1:      time1,
		Hash1:      hash1,
	}, nil
}

// BuildStLoginResultPacket 构造跨服登录响应二进制封包。
func BuildStLoginResultPacket(res StLoginResult) (*Packet, error) {
	w := NewWriter()
	w.WriteUint8(res.Result)
	w.WriteInt32(res.PlayerID)
	w.WriteInt32(res.ServerTime)
	return NewPacket(ActionIDStLogin, w.Bytes()), nil
}

// ParseStLoginResult 从响应载荷中解析登录结果。
func ParseStLoginResult(payload []byte) (*StLoginResult, error) {
	r := NewReader(payload)

	resCode, err := r.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read result code: %v", ErrInvalidPayload, err)
	}

	playerID, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read player_id: %v", ErrInvalidPayload, err)
	}

	serverTime, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read server_time: %v", ErrInvalidPayload, err)
	}

	return &StLoginResult{
		Result:     resCode,
		PlayerID:   playerID,
		ServerTime: serverTime,
	}, nil
}
