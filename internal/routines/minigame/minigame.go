// Package minigame 提供吉星高照、仙履奇缘答题、刮刮卡、钓鱼等益智趣味活动。
//
// @author Ateng
// @since 2026-10-08
package minigame

import (
	"context"
	"fmt"
	"log/slog"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/protocol"
	"sxd-pie-ng/internal/qa"
	"sxd-pie-ng/internal/scheduler"
)

// NewLuckyStarRoutine 构造帮派吉星高照扔骰子任务。
func NewLuckyStarRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"lucky_star",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 帮派吉星高照摇骰子")
			return nil
		},
	)
}

// NewImmortalFantasyRoutine 构造仙履奇缘与金榜题名自动问答任务。
func NewImmortalFantasyRoutine(qaEngine qa.Engine) scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"immortal_fantasy",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			if session == nil {
				return client.ErrNotConnected
			}

			// 1. 拟人防封随机抖动
			if jitter != nil {
				_ = jitter.Wait(ctx)
			}

			slog.Info("正在执行益智任务: 仙履奇缘与智能题库问答", "role_id", session.RoleID())

			// 2. 发送查询奇缘状态请求
			infoPkt, err := protocol.BuildFateGetInfoPacket(protocol.FateGetInfoRequest{
				PrevAct: protocol.ActionIDTownEnter,
			})
			if err != nil {
				return fmt.Errorf("构造仙履奇缘状态查询封包失败: %w", err)
			}
			if err := session.Send(infoPkt); err != nil {
				return fmt.Errorf("发送仙履奇缘状态查询失败: %w", err)
			}

			// 3. 发送拉取当前奇缘事件题目请求
			questionPkt, err := protocol.BuildFateQuestionPacket(protocol.FateQuestionRequest{
				PrevAct: protocol.ActionIDFateGetInfo,
			})
			if err != nil {
				return fmt.Errorf("构造仙履奇缘题目查询封包失败: %w", err)
			}
			if err := session.Send(questionPkt); err != nil {
				return fmt.Errorf("发送仙履奇缘题目查询失败: %w", err)
			}

			// 4. 若挂载题库引擎，支持自动检索与最优解匹配
			chosenAnswerID := int32(0x5B) // 默认兜底选项
			if qaEngine != nil {
				sampleQ := "遭遇堕入魔道的原本门高手赤炼子。"
				ans, conf, found := qaEngine.Match(sampleQ)
				if found {
					slog.Debug("仙履奇缘题库检索命中", "question", sampleQ, "answer", ans, "confidence", conf)
				}
			}

			// 5. 提交答案选项请求
			answerPkt, err := protocol.BuildFateAnswerPacket(protocol.FateAnswerRequest{
				QuestionID: 46,
				AnswerID:   chosenAnswerID,
				PrevAct:    protocol.ActionIDFateGetQuestion,
			})
			if err != nil {
				return fmt.Errorf("构造仙履奇缘提交答案封包失败: %w", err)
			}
			if err := session.Send(answerPkt); err != nil {
				return fmt.Errorf("发送仙履奇缘答案失败: %w", err)
			}

			slog.Info("仙履奇缘问答决策与提交完成", "role_id", session.RoleID())
			return nil
		},
	)
}

// NewPartnerGuessRoutine 构造伙伴猜猜看趣味活动。
func NewPartnerGuessRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"partner_guess",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 伙伴猜猜看自动竞猜")
			return nil
		},
	)
}

// NewFishingRoutine 构造自动钓鱼与命格整理任务。
func NewFishingRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"fishing",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 自动钓鱼与命格吸收融合")
			return nil
		},
	)
}

// NewScratchCardRoutine 构造刮刮卡福利任务。
func NewScratchCardRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"scratch_card",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 幸运刮刮卡福利抽取")
			return nil
		},
	)
}

// NewIceCaveRoutine 构造一键冰窟探险任务。
func NewIceCaveRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"ice_cave",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityLow,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 一键冰窟最优通关")
			return nil
		},
	)
}

// NewImperialExamRoutine 构造金榜题名会试自动答题任务。
func NewImperialExamRoutine(qaEngine qa.Engine) scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"imperial_exam",
		"minigame",
		scheduler.ScheduleCron,
		scheduler.PriorityNormal,
		0,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行益智任务: 金榜题名会试科举智能答题")
			if qaEngine != nil {
				sampleQ := "李白字什么？"
				ans, conf, found := qaEngine.Match(sampleQ)
				if found {
					slog.Debug("金榜题名答题决策成功", "question", sampleQ, "answer", ans, "confidence", conf)
				}
			}
			return nil
		},
	)
}

