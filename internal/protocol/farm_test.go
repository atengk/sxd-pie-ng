// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-09
package protocol

import (
	"encoding/hex"
	"testing"
)

func TestFarmCodec_BuildAndParse(t *testing.T) {
	// 1. 测试 FarmGetInfoRequest
	getInfoReq := FarmGetInfoRequest{PrevAct: 0x00010000}
	pktInfo, err := BuildFarmGetInfoPacket(getInfoReq)
	if err != nil {
		t.Fatalf("BuildFarmGetInfoPacket 失败: %v", err)
	}
	if pktInfo.ActionID != ActionIDFarmGetInfo {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDFarmGetInfo, pktInfo.ActionID)
	}

	parsedInfo, err := ParseFarmGetInfoRequest(pktInfo.Payload)
	if err != nil {
		t.Fatalf("ParseFarmGetInfoRequest 失败: %v", err)
	}
	if parsedInfo.PrevAct != 0x00010000 {
		t.Errorf("PrevAct 不匹配: 期望 0x00010000, 实际 0x%08X", parsedInfo.PrevAct)
	}

	// 2. 测试 FarmPlantRequest 对齐真实抓包 Hex
	// 真实抓包: 0000000a000000a9000d0016 (LandID=10, SeedID=169, PrevAct=0x000D0016)
	plantReq := FarmPlantRequest{
		LandID:       10,
		SeedOrRoleID: 169,
		PrevAct:      0x000D0016,
	}
	pktPlant, err := BuildFarmPlantPacket(plantReq)
	if err != nil {
		t.Fatalf("BuildFarmPlantPacket 失败: %v", err)
	}
	if pktPlant.ActionID != ActionIDFarmPlant {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDFarmPlant, pktPlant.ActionID)
	}

	expectedHex := "0000000a000000a9000d0016"
	actualHex := hex.EncodeToString(pktPlant.Payload)
	if actualHex != expectedHex {
		t.Errorf("播种载荷 Hex 不匹配: 期望 %s, 实际 %s", expectedHex, actualHex)
	}

	parsedPlant, err := ParseFarmPlantRequest(pktPlant.Payload)
	if err != nil {
		t.Fatalf("ParseFarmPlantRequest 失败: %v", err)
	}
	if parsedPlant.LandID != 10 || parsedPlant.SeedOrRoleID != 169 || parsedPlant.PrevAct != 0x000D0016 {
		t.Errorf("播种解析结果不符: %+v", parsedPlant)
	}

	// 3. 测试 FarmHarvestRequest 对齐真实抓包 Hex
	// 真实抓包: 0000000a000d0018 (LandID=10, PrevAct=0x000D0018)
	harvestReq := FarmHarvestRequest{
		LandID:  10,
		PrevAct: 0x000D0018,
	}
	pktHarvest, err := BuildFarmHarvestPacket(harvestReq)
	if err != nil {
		t.Fatalf("BuildFarmHarvestPacket 失败: %v", err)
	}
	if pktHarvest.ActionID != ActionIDFarmHarvest {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDFarmHarvest, pktHarvest.ActionID)
	}

	expectedHarvestHex := "0000000a000d0018"
	actualHarvestHex := hex.EncodeToString(pktHarvest.Payload)
	if actualHarvestHex != expectedHarvestHex {
		t.Errorf("收获载荷 Hex 不匹配: 期望 %s, 实际 %s", expectedHarvestHex, actualHarvestHex)
	}

	parsedHarvest, err := ParseFarmHarvestRequest(pktHarvest.Payload)
	if err != nil {
		t.Fatalf("ParseFarmHarvestRequest 失败: %v", err)
	}
	if parsedHarvest.LandID != 10 || parsedHarvest.PrevAct != 0x000D0018 {
		t.Errorf("收获解析结果不符: %+v", parsedHarvest)
	}
}

func TestFarmCodec_ResultsRoundtrip(t *testing.T) {
	// 1. 验证 FarmGetInfoResult
	infoRes := FarmGetInfoResult{
		Fields: []FarmField{
			{LandID: 10, State: 1, Cooldown: 0, SeedOrRoleID: 0},       // 空闲
			{LandID: 11, State: 3, Cooldown: 0, SeedOrRoleID: 169},     // 成熟待采摘
			{LandID: 12, State: 2, Cooldown: 3600, SeedOrRoleID: 169},  // 种植中
		},
	}
	pktInfoRes, err := BuildFarmGetInfoResultPacket(infoRes)
	if err != nil {
		t.Fatalf("BuildFarmGetInfoResultPacket 失败: %v", err)
	}
	if pktInfoRes.ActionID != ActionIDFarmGetInfo {
		t.Fatalf("ActionID 不匹配: 0x%08X", pktInfoRes.ActionID)
	}

	parsedInfoRes, err := ParseFarmGetInfoResult(pktInfoRes.Payload)
	if err != nil {
		t.Fatalf("ParseFarmGetInfoResult 失败: %v", err)
	}
	if len(parsedInfoRes.Fields) != 3 {
		t.Fatalf("土地列表长度不匹配: 期望 3, 实际 %d", len(parsedInfoRes.Fields))
	}
	if parsedInfoRes.Fields[1].State != 3 || parsedInfoRes.Fields[1].LandID != 11 {
		t.Errorf("第 2 块土地状态不符: %+v", parsedInfoRes.Fields[1])
	}

	// 2. 验证 FarmHarvestResult
	harvestRes := FarmHarvestResult{
		Success:   true,
		GainExp:   125000,
		GainCoins: 580000,
	}
	pktHarvestRes, err := BuildFarmHarvestResultPacket(harvestRes)
	if err != nil {
		t.Fatalf("BuildFarmHarvestResultPacket 失败: %v", err)
	}
	parsedHarvestRes, err := ParseFarmHarvestResult(pktHarvestRes.Payload)
	if err != nil {
		t.Fatalf("ParseFarmHarvestResult 失败: %v", err)
	}
	if !parsedHarvestRes.Success || parsedHarvestRes.GainExp != 125000 || parsedHarvestRes.GainCoins != 580000 {
		t.Errorf("收获解析结果不符: %+v", parsedHarvestRes)
	}

	// 3. 验证 FarmPlantResult
	plantRes := FarmPlantResult{
		Success:  true,
		LandID:   10,
		Cooldown: 28800,
	}
	pktPlantRes, err := BuildFarmPlantResultPacket(plantRes)
	if err != nil {
		t.Fatalf("BuildFarmPlantResultPacket 失败: %v", err)
	}
	parsedPlantRes, err := ParseFarmPlantResult(pktPlantRes.Payload)
	if err != nil {
		t.Fatalf("ParseFarmPlantResult 失败: %v", err)
	}
	if !parsedPlantRes.Success || parsedPlantRes.LandID != 10 || parsedPlantRes.Cooldown != 28800 {
		t.Errorf("播种解析结果不符: %+v", parsedPlantRes)
	}
}

