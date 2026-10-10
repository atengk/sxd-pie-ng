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
	// ActionHeartbeat 心跳保活协议号 (Module 0, Action 23 -> 0x00000017)
	ActionHeartbeat uint32 = 0x00000017
	// ActionPlayerLogin 角色登录握手认证协议号
	ActionPlayerLogin uint32 = 0x00000000
	// ActionPlayerInfo 角色属性与体力同步协议号
	ActionPlayerInfo uint32 = 0x00000002
	// ActionEnterTown 进入城镇协议号
	ActionEnterTown uint32 = 0x00010000
	// ActionMissionEnter 进入/查询关卡协议号 (Module 35, Action 0)
	ActionMissionEnter uint32 = 0x00230000
	// ActionMissionSweep 关卡副本扫荡请求与响应协议号 (Module 35, Action 2)
	ActionMissionSweep uint32 = 0x00230002
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

// BuildEnterMissionPacket 构造进入/激活关卡请求封包。
func BuildEnterMissionPacket(missionID uint32) *Packet {
	w := NewWriter()
	// 真实网关二进制帧格式: [2B MissionID] + [4B 00 23 00 00]
	w.WriteUint16(uint16(missionID))
	w.WriteBytes([]byte{0x00, 0x23, 0x00, 0x00})
	return NewPacket(ActionMissionEnter, w.Bytes())
}

// BuildSweepPacket 构造关卡扫荡二进制协议请求封包。
func BuildSweepPacket(req SweepRequest) (*Packet, error) {
	w := NewWriter()
	// 真实网关二进制帧格式: [2B MissionID] + [1B Times] + [5B 扩展控制标记]
	w.WriteUint16(uint16(req.MissionID))
	w.WriteUint8(uint8(req.Times))
	w.WriteBytes([]byte{0x00, 0x00, 0x23, 0x00, 0x00})
	return NewPacket(ActionMissionSweep, w.Bytes()), nil
}

// ParseSweepRequest 从封包载荷中反序列化扫荡请求参数。
func ParseSweepRequest(payload []byte) (*SweepRequest, error) {
	if len(payload) < 3 {
		return nil, fmt.Errorf("%w: payload too short", ErrInvalidPayload)
	}
	r := NewReader(payload)
	missionID, err := r.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read mission_id: %v", ErrInvalidPayload, err)
	}
	times, err := r.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read times: %v", ErrInvalidPayload, err)
	}
	return &SweepRequest{
		MissionID: uint32(missionID),
		Times:     uint16(times),
	}, nil
}

// BuildSweepResultPacket 构造关卡扫荡结算响应封包。
func BuildSweepResultPacket(res SweepResult) (*Packet, error) {
	w := NewWriter()
	var successFlag uint8
	if res.Success {
		successFlag = 0x03 // 真实网关成功标识码 0x03
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
	if len(payload) == 0 {
		return nil, fmt.Errorf("%w: empty sweep payload", ErrInvalidPayload)
	}
	r := NewReader(payload)
	flag, err := r.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read status flag: %v", ErrInvalidPayload, err)
	}

	// 真实网关短帧协议兼容: 首字节为 0x03 或 0x01 表示成功
	isSuccess := flag == 0x03 || flag == 0x01
	res := &SweepResult{
		Success: isSuccess,
		Message: "扫荡完成",
	}

	// 若载荷为完整长包 (测试桩或带元数据的回包) 则继续深度反序列化
	if r.Remaining() >= 30 {
		if mid, err := r.ReadUint32(); err == nil {
			res.MissionID = mid
		}
		if times, err := r.ReadUint16(); err == nil {
			res.Times = times
		}
		if costPower, err := r.ReadInt32(); err == nil {
			res.CostPower = int(costPower)
		}
		if gainExp, err := r.ReadInt64(); err == nil {
			res.GainExp = gainExp
		}
		if gainCoins, err := r.ReadInt64(); err == nil {
			res.GainCoins = gainCoins
		}
		if msg, err := r.ReadString(); err == nil && msg != "" {
			res.Message = msg
		}
	} else if r.Remaining() >= 4 {
		// 真实 5 字节短帧回包: 携带剩余次数或消耗
		if v, err := r.ReadInt32(); err == nil {
			res.CostPower = int(v)
		}
	}

	return res, nil
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
