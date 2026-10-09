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
