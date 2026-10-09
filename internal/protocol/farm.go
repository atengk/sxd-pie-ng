// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-09
package protocol

import (
	"fmt"
)

// FarmField 代表药园中的单块种植土地状态。
type FarmField struct {
	// LandID 土地编号 (如 10, 11, 12)
	LandID int32
	// State 土地状态 (0: 未开垦, 1: 空闲, 2: 种植中, 3: 可采摘)
	State int32
	// Cooldown 成熟倒计时 (秒)
	Cooldown int32
	// SeedOrRoleID 种植的种子或被培养的角色伙伴 ID
	SeedOrRoleID int32
}

// FarmGetInfoRequest 代表查询药园所有土地状态请求 (ActionID=0x000D0000)。
type FarmGetInfoRequest struct {
	// PrevAct 前序协议编号
	PrevAct uint32
}

// FarmGetInfoResult 代表药园全量土地状态列表。
type FarmGetInfoResult struct {
	// Fields 土地切片
	Fields []FarmField
}

// FarmPlantRequest 代表药园播种药草请求 (ActionID=0x000D0018)。
type FarmPlantRequest struct {
	// LandID 目标土地编号
	LandID int32
	// SeedOrRoleID 选中的种子或伙伴 ID (如经验种子 0xA9)
	SeedOrRoleID int32
	// PrevAct 前序协议编号 (如 0x000D0016)
	PrevAct uint32
}

// FarmPlantResult 代表药园播种响应结果。
type FarmPlantResult struct {
	// Success 播种是否成功
	Success bool
	// LandID 目标土地编号
	LandID int32
	// Cooldown 种植成熟冷却时长 (秒，通常 28800 秒 / 8 小时)
	Cooldown int32
}

// FarmHarvestRequest 代表收获成熟药草请求 (ActionID=0x000D0019)。
type FarmHarvestRequest struct {
	// LandID 目标土地编号
	LandID int32
	// PrevAct 前序协议编号 (如 0x000D0018)
	PrevAct uint32
}

// FarmHarvestResult 代表收获药草结算响应。
type FarmHarvestResult struct {
	// Success 是否收获成功
	Success bool
	// GainExp 获得的经验数值
	GainExp int64
	// GainCoins 获得的铜钱数值
	GainCoins int64
}

// BuildFarmGetInfoPacket 构造查询药园土地信息请求封包。
func BuildFarmGetInfoPacket(req FarmGetInfoRequest) (*Packet, error) {
	w := NewWriter()
	prevAct := req.PrevAct
	if prevAct == 0 {
		prevAct = ActionIDTownEnter
	}
	w.WriteUint32(prevAct)
	return NewPacket(ActionIDFarmGetInfo, w.Bytes()), nil
}

// ParseFarmGetInfoRequest 从载荷反序列化查询土地信息请求。
func ParseFarmGetInfoRequest(payload []byte) (*FarmGetInfoRequest, error) {
	r := NewReader(payload)
	prevAct, err := r.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read prev_act: %v", ErrInvalidPayload, err)
	}
	return &FarmGetInfoRequest{PrevAct: prevAct}, nil
}

// BuildFarmPlantPacket 构造播种药草二进制封包。
func BuildFarmPlantPacket(req FarmPlantRequest) (*Packet, error) {
	w := NewWriter()
	// 1. 写入 LandID (4B 大端序整型)
	w.WriteInt32(req.LandID)
	// 2. 写入 SeedOrRoleID (4B 大端序整型)
	w.WriteInt32(req.SeedOrRoleID)
	// 3. 写入 PrevAct (4B 大端序整型)
	prevAct := req.PrevAct
	if prevAct == 0 {
		prevAct = ActionIDFarmGetInfo
	}
	w.WriteUint32(prevAct)

	return NewPacket(ActionIDFarmPlant, w.Bytes()), nil
}

// ParseFarmPlantRequest 从载荷反序列化播种药草请求。
func ParseFarmPlantRequest(payload []byte) (*FarmPlantRequest, error) {
	if len(payload) < 12 {
		return nil, fmt.Errorf("%w: payload too short for farm plant", ErrInvalidPayload)
	}
	r := NewReader(payload)
	landID, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read land_id: %v", ErrInvalidPayload, err)
	}
	seedID, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read seed_id: %v", ErrInvalidPayload, err)
	}
	prevAct, err := r.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read prev_act: %v", ErrInvalidPayload, err)
	}

	return &FarmPlantRequest{
		LandID:       landID,
		SeedOrRoleID: seedID,
		PrevAct:      prevAct,
	}, nil
}

// BuildFarmHarvestPacket 构造收取成熟药草二进制封包。
func BuildFarmHarvestPacket(req FarmHarvestRequest) (*Packet, error) {
	w := NewWriter()
	// 1. 写入 LandID (4B 大端序整型)
	w.WriteInt32(req.LandID)
	// 2. 写入 PrevAct (4B 大端序整型)
	prevAct := req.PrevAct
	if prevAct == 0 {
		prevAct = ActionIDFarmPlant
	}
	w.WriteUint32(prevAct)

	return NewPacket(ActionIDFarmHarvest, w.Bytes()), nil
}

// ParseFarmHarvestRequest 从载荷反序列化收取成熟药草请求。
func ParseFarmHarvestRequest(payload []byte) (*FarmHarvestRequest, error) {
	if len(payload) < 8 {
		return nil, fmt.Errorf("%w: payload too short for farm harvest", ErrInvalidPayload)
	}
	r := NewReader(payload)
	landID, err := r.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read land_id: %v", ErrInvalidPayload, err)
	}
	prevAct, err := r.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read prev_act: %v", ErrInvalidPayload, err)
	}

	return &FarmHarvestRequest{
		LandID:  landID,
		PrevAct: prevAct,
	}, nil
}
