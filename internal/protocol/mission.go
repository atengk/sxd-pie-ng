// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-08
package protocol

import (
	"errors"
	"fmt"
)

const (
	// ActionHeartbeat 心跳保活协议号
	ActionHeartbeat uint16 = 0x0001
	// ActionPlayerLogin 角色登录握手认证协议号
	ActionPlayerLogin uint16 = 0x0002
	// ActionPlayerInfo 角色属性与体力同步协议号
	ActionPlayerInfo uint16 = 0x0003
	// ActionEnterTown 进入城镇协议号
	ActionEnterTown uint16 = 0x0004
	// ActionMissionSweep 关卡副本扫荡请求与响应协议号
	ActionMissionSweep uint16 = 0x000B
)

var (
	// ErrInvalidPayload 载荷格式不合法或数据解析异常
	ErrInvalidPayload = errors.New("protocol: invalid payload format")
)

// SweepRequest 代表关卡副本扫荡请求参数。
type SweepRequest struct {
	// MissionID 关卡唯一标识符
	MissionID uint32
	// Times 扫荡次数 (通常单次或体力折算多次)
	Times uint16
}

// SweepResult 代表关卡副本扫荡执行结果及收益。
type SweepResult struct {
	// Success 扫荡是否成功
	Success bool
	// MissionID 对应关卡标识符
	MissionID uint32
	// Times 实际完成扫荡次数
	Times uint16
	// CostPower 扣除总体力点数
	CostPower int
	// GainExp 获得的经验数值
	GainExp int64
	// GainCoins 获得的铜钱数值
	GainCoins int64
	// Message 结算描述信息 (如 "扫荡完成", "体力不足", "背包已满")
	Message string
}

// BuildSweepPacket 构造关卡扫荡二进制协议请求封包。
func BuildSweepPacket(req SweepRequest) (*Packet, error) {
	w := NewWriter()
	w.WriteUint32(req.MissionID)
	w.WriteUint16(req.Times)
	return NewPacket(ActionMissionSweep, w.Bytes()), nil
}

// ParseSweepRequest 从封包载荷中反序列化扫荡请求参数。
func ParseSweepRequest(payload []byte) (*SweepRequest, error) {
	r := NewReader(payload)
	missionID, err := r.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read mission_id: %v", ErrInvalidPayload, err)
	}
	times, err := r.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read times: %v", ErrInvalidPayload, err)
	}
	return &SweepRequest{
		MissionID: missionID,
		Times:     times,
	}, nil
}

// BuildSweepResultPacket 构造关卡扫荡结算响应封包。
func BuildSweepResultPacket(res SweepResult) (*Packet, error) {
	w := NewWriter()
	var successFlag uint8
	if res.Success {
		successFlag = 1
	}
	w.WriteUint8(successFlag)
	w.WriteUint32(res.MissionID)
	w.WriteUint16(res.Times)
	w.WriteInt32(int32(res.CostPower))
	w.WriteInt64(res.GainExp)
	w.WriteInt64(res.GainCoins)
	w.WriteString(res.Message)

	return NewPacket(ActionMissionSweep, w.Bytes()), nil
}

// ParseSweepResult 从响应封包载荷中反序列化扫荡结果。
func ParseSweepResult(payload []byte) (*SweepResult, error) {
	r := NewReader(payload)
	flag, err := r.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read success flag: %v", ErrInvalidPayload, err)
	}
	missionID, err := r.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read mission_id: %v", ErrInvalidPayload, err)
	}
	times, err := r.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read times: %v", ErrInvalidPayload, err)
	}
	costPower, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read cost_power: %v", ErrInvalidPayload, err)
	}
	gainExp, err := r.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read gain_exp: %v", ErrInvalidPayload, err)
	}
	gainCoins, err := r.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read gain_coins: %v", ErrInvalidPayload, err)
	}
	msg, err := r.ReadString()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read message: %v", ErrInvalidPayload, err)
	}

	return &SweepResult{
		Success:   flag == 1,
		MissionID: missionID,
		Times:     times,
		CostPower: int(costPower),
		GainExp:   gainExp,
		GainCoins: gainCoins,
		Message:   msg,
	}, nil
}

// BuildLoginPacket 构造角色登录握手协议封包。
func BuildLoginPacket(roleName, token string) (*Packet, error) {
	w := NewWriter()
	w.WriteString(roleName)
	w.WriteString(token)
	return NewPacket(ActionPlayerLogin, w.Bytes()), nil
}

// ParseLoginPacket 从登录封包载荷中解析角色名称与认证 Token。
func ParseLoginPacket(payload []byte) (roleName, token string, err error) {
	r := NewReader(payload)
	name, err := r.ReadString()
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to read role_name: %v", ErrInvalidPayload, err)
	}
	tok, err := r.ReadString()
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to read token: %v", ErrInvalidPayload, err)
	}
	return name, tok, nil
}
