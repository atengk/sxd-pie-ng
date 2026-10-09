// Package protocol 提供神仙道客户端与服务器之间的二进制协议序列化、反序列化、大端序编解码与透明 zlib 流解压缩引擎。
//
// @author Ateng
// @since 2026-10-09
package protocol

import (
	"encoding/hex"
	"testing"
)

func TestFateCodec_BuildAndParse(t *testing.T) {
	// 1. 测试 FateGetInfoRequest 序列化与反序列化
	getInfoReq := FateGetInfoRequest{PrevAct: 0x00010000}
	pkt, err := BuildFateGetInfoPacket(getInfoReq)
	if err != nil {
		t.Fatalf("BuildFateGetInfoPacket 失败: %v", err)
	}
	if pkt.ActionID != ActionIDFateGetInfo {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDFateGetInfo, pkt.ActionID)
	}

	parsedGetInfo, err := ParseFateGetInfoRequest(pkt.Payload)
	if err != nil {
		t.Fatalf("ParseFateGetInfoRequest 失败: %v", err)
	}
	if parsedGetInfo.PrevAct != 0x00010000 {
		t.Errorf("PrevAct 不匹配: 期望 0x00010000, 实际 0x%08X", parsedGetInfo.PrevAct)
	}

	// 2. 测试 FateQuestionRequest 序列化与反序列化
	questionReq := FateQuestionRequest{PrevAct: ActionIDFateGetInfo}
	pktQuestion, err := BuildFateQuestionPacket(questionReq)
	if err != nil {
		t.Fatalf("BuildFateQuestionPacket 失败: %v", err)
	}
	if pktQuestion.ActionID != ActionIDFateGetQuestion {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDFateGetQuestion, pktQuestion.ActionID)
	}

	parsedQuestion, err := ParseFateQuestionRequest(pktQuestion.Payload)
	if err != nil {
		t.Fatalf("ParseFateQuestionRequest 失败: %v", err)
	}
	if parsedQuestion.PrevAct != ActionIDFateGetInfo {
		t.Errorf("PrevAct 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDFateGetInfo, parsedQuestion.PrevAct)
	}

	// 3. 测试 FateAnswerRequest 对齐真实抓包 Hex
	// 真实抓包: 0000002e0000005c00150001 (QuestionID=46, AnswerID=92, PrevAct=0x00150001)
	answerReq := FateAnswerRequest{
		QuestionID: 46,
		AnswerID:   92,
		PrevAct:    0x00150001,
	}
	pktAnswer, err := BuildFateAnswerPacket(answerReq)
	if err != nil {
		t.Fatalf("BuildFateAnswerPacket 失败: %v", err)
	}
	if pktAnswer.ActionID != ActionIDFateAnswer {
		t.Fatalf("ActionID 不匹配: 期望 0x%08X, 实际 0x%08X", ActionIDFateAnswer, pktAnswer.ActionID)
	}

	expectedHex := "0000002e0000005c00150001"
	actualHex := hex.EncodeToString(pktAnswer.Payload)
	if actualHex != expectedHex {
		t.Errorf("载荷 Hex 与抓包不一致: 期望 %s, 实际 %s", expectedHex, actualHex)
	}

	parsedAnswer, err := ParseFateAnswerRequest(pktAnswer.Payload)
	if err != nil {
		t.Fatalf("ParseFateAnswerRequest 失败: %v", err)
	}
	if parsedAnswer.QuestionID != 46 || parsedAnswer.AnswerID != 92 || parsedAnswer.PrevAct != 0x00150001 {
		t.Errorf("解析结果与原值不符: %+v", parsedAnswer)
	}
}

func TestFateCodec_ParseAnswerResult(t *testing.T) {
	// 构造带有 2 字节长度前缀的 UTF-8 字符串载荷
	w := NewWriter()
	w.WriteString("你获得50阅历,5体力！")
	payload := w.Bytes()

	res, err := ParseFateAnswerResult(payload)
	if err != nil {
		t.Fatalf("ParseFateAnswerResult 失败: %v", err)
	}
	if res.Message != "你获得50阅历,5体力！" {
		t.Errorf("解析文本不符: %s", res.Message)
	}
}
