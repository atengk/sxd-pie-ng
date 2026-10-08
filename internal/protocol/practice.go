// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-08
package protocol

import (
	"fmt"
)

// StartPracticeRequest 代表开始关卡挂机扫荡请求 (Mod_MissionPractice_Base.start_practice, ActionID=0x0119)。
type StartPracticeRequest struct {
	// MissionID 关卡唯一数字标识
	MissionID int32
	// Count 扫荡轮次/次数
	Count int16
	// AutoSale 自动卖出白绿蓝装配置 (0: 否, 1: 是)
	AutoSale int8
}

// StartPracticeResult 代表开始扫荡服务器响应结果。
type StartPracticeResult struct {
	// Result 状态码 (0=SUCCESS, 1=NOT_ENOUGH_POWER, 2=BAG_FULL, 3=IN_PRACTICE)
	Result uint8
	// MissionID 当前开始扫荡的关卡 ID
	MissionID int32
	// Count 实际开始扫荡的总次数
	Count int16
}

// QuicklyRequest 代表加速秒完成扫荡请求 (Mod_MissionPractice_Base.quickly, ActionID=0x0319)。
type QuicklyRequest struct {
	// Count 加速完成的次数
	Count int16
}

// QuicklyResult 代表加速秒完成服务器响应结果。
type QuicklyResult struct {
	// Result 状态码 (0=SUCCESS)
	Result uint8
	// Count 成功加速完成的次数
	Count int16
}

// BuildStartPracticePacket 构造开始扫荡二进制封包。
func BuildStartPracticePacket(req StartPracticeRequest) (*Packet, error) {
	w := NewWriter()
	// 1. 写入 MissionID (4 字节大端序有符号整型)
	w.WriteInt32(req.MissionID)
	// 2. 写入 Count (2 字节大端序有符号整型)
	w.WriteInt16(req.Count)
	// 3. 写入 AutoSale (1 字节有符号整型)
	w.WriteInt8(req.AutoSale)

	return NewPacket(ActionIDPracticeStart, w.Bytes()), nil
}

// ParseStartPracticeRequest 从封包载荷中反序列化开始扫荡请求参数。
func ParseStartPracticeRequest(payload []byte) (*StartPracticeRequest, error) {
	r := NewReader(payload)

	missionID, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read mission_id: %v", ErrInvalidPayload, err)
	}

	count, err := r.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read count: %v", ErrInvalidPayload, err)
	}

	autoSale, err := r.ReadInt8()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read auto_sale: %v", ErrInvalidPayload, err)
	}

	return &StartPracticeRequest{
		MissionID: missionID,
		Count:     count,
		AutoSale:  autoSale,
	}, nil
}

// BuildStartPracticeResultPacket 构造开始扫荡响应二进制封包。
func BuildStartPracticeResultPacket(res StartPracticeResult) (*Packet, error) {
	w := NewWriter()
	w.WriteUint8(res.Result)
	w.WriteInt32(res.MissionID)
	w.WriteInt16(res.Count)
	return NewPacket(ActionIDPracticeStart, w.Bytes()), nil
}

// ParseStartPracticeResult 从响应载荷中反序列化开始扫荡结果。
func ParseStartPracticeResult(payload []byte) (*StartPracticeResult, error) {
	r := NewReader(payload)

	resCode, err := r.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read result: %v", ErrInvalidPayload, err)
	}

	missionID, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read mission_id: %v", ErrInvalidPayload, err)
	}

	count, err := r.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read count: %v", ErrInvalidPayload, err)
	}

	return &StartPracticeResult{
		Result:    resCode,
		MissionID: missionID,
		Count:     count,
	}, nil
}

// BuildQuicklyPacket 构造扫荡加速完成二进制封包。
func BuildQuicklyPacket(req QuicklyRequest) (*Packet, error) {
	w := NewWriter()
	w.WriteInt16(req.Count)
	return NewPacket(ActionIDPracticeQuickly, w.Bytes()), nil
}

// ParseQuicklyRequest 从封包载荷中反序列化加速完成请求参数。
func ParseQuicklyRequest(payload []byte) (*QuicklyRequest, error) {
	r := NewReader(payload)
	count, err := r.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read count: %v", ErrInvalidPayload, err)
	}
	return &QuicklyRequest{Count: count}, nil
}

// BuildQuicklyResultPacket 构造扫荡加速完成响应二进制封包。
func BuildQuicklyResultPacket(res QuicklyResult) (*Packet, error) {
	w := NewWriter()
	w.WriteUint8(res.Result)
	w.WriteInt16(res.Count)
	return NewPacket(ActionIDPracticeQuickly, w.Bytes()), nil
}

// ParseQuicklyResult 从响应载荷中反序列化加速完成结果。
func ParseQuicklyResult(payload []byte) (*QuicklyResult, error) {
	r := NewReader(payload)
	resCode, err := r.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read result: %v", ErrInvalidPayload, err)
	}
	count, err := r.ReadInt16()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read count: %v", ErrInvalidPayload, err)
	}
	return &QuicklyResult{
		Result: resCode,
		Count:  count,
	}, nil
}
