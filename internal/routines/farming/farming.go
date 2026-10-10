// Package farming 提供药园种植、取经护送、仙界矿山与异兽养育等资源产出玩法。
//
// @author Ateng
// @since 2026-10-08
package farming

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/dictionary"
	"sxd-pie-ng/internal/protocol"
	"sxd-pie-ng/internal/scheduler"
)

// NewHerbGardenRoutine 构造药园种植与巡检任务。
// 支持无参调用或注入 dictionary.Repository 字典仓储。
func NewHerbGardenRoutine(dictRepos ...dictionary.Repository) scheduler.ActivityRoutine {
	var dictRepo dictionary.Repository
	if len(dictRepos) > 0 {
		dictRepo = dictRepos[0]
	}

	return scheduler.NewBaseRoutine(
		"herb_garden",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityNormal,
		15*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			if session == nil {
				return client.ErrNotConnected
			}
			if session.State() != client.StateActive {
				return client.ErrNotConnected
			}

			// 1. 拟人防封随机抖动
			if jitter != nil {
				_ = jitter.Wait(ctx)
			}

			slog.Info("正在执行资源任务: 药园土地巡检、成熟采摘与播种", "role_id", session.RoleID())

			// 2. 发送查询土地状态请求 (ActionIDFarmGetInfo 0x000D0000)
			infoPkt, err := protocol.BuildFarmGetInfoPacket(protocol.FarmGetInfoRequest{
				PrevAct: protocol.ActionIDTownEnter,
			})
			if err != nil {
				return fmt.Errorf("farming: 构造药园查询封包失败: %w", err)
			}

			respPkt, err := session.Call(ctx, infoPkt, protocol.ActionIDFarmGetInfo)
			if err != nil {
				return fmt.Errorf("farming: 查询药园土地状态超时或失败: %w", err)
			}

			infoRes, err := protocol.ParseFarmGetInfoResult(respPkt.Payload)
			if err != nil {
				return fmt.Errorf("farming: 解析药园土地列表失败: %w", err)
			}

			var harvestedCount int
			var plantedCount int
			var growingCount int
			var totalGainExp int64
			var totalGainCoins int64

			// 确定待种植的种子/伙伴编号 (默认 169 仙玲珑经验草)
			seedID := int32(169)
			seedName := "仙玲珑经验草"
			if dictRepo != nil {
				if item, dErr := dictRepo.GetItem(int(seedID)); dErr == nil && item != nil {
					seedName = item.Name
				}
			}

			// 3. 遍历土地列表并执行智能处理
			for _, field := range infoRes.Fields {
				// 未开垦土地 (State == 0) 跳过
				if field.State == 0 {
					continue
				}

				// 土地处于可采摘状态 (State == 3)
				if field.State == 3 {
					harvestPkt, hErr := protocol.BuildFarmHarvestPacket(protocol.FarmHarvestRequest{
						LandID:  field.LandID,
						PrevAct: protocol.ActionIDFarmGetInfo,
					})
					if hErr == nil {
						hResp, callErr := session.Call(ctx, harvestPkt, protocol.ActionIDFarmHarvest)
						if callErr == nil {
							if hResult, pErr := protocol.ParseFarmHarvestResult(hResp.Payload); pErr == nil && hResult.Success {
								harvestedCount++
								totalGainExp += hResult.GainExp
								totalGainCoins += hResult.GainCoins
								session.AddRewards(hResult.GainExp, hResult.GainCoins)
								slog.Info("药园药草采摘成功",
									"role_id", session.RoleID(),
									"land_id", field.LandID,
									"gain_exp", hResult.GainExp,
									"gain_coins", hResult.GainCoins,
								)
								// 采摘后地块即刻变为空闲状态，可继续播种
								field.State = 1
							}
						}
					}
				}

				// 土地处于空闲状态 (State == 1，含采摘后地块)
				if field.State == 1 {
					plantPkt, pErr := protocol.BuildFarmPlantPacket(protocol.FarmPlantRequest{
						LandID:       field.LandID,
						SeedOrRoleID: seedID,
						PrevAct:      protocol.ActionIDFarmHarvest,
					})
					if pErr == nil {
						pResp, callErr := session.Call(ctx, plantPkt, protocol.ActionIDFarmPlant)
						if callErr == nil {
							if pResult, parseErr := protocol.ParseFarmPlantResult(pResp.Payload); parseErr == nil && pResult.Success {
								plantedCount++
								slog.Info("药园土地播种成功",
									"role_id", session.RoleID(),
									"land_id", field.LandID,
									"seed_name", seedName,
									"seed_id", seedID,
									"cooldown_sec", pResult.Cooldown,
								)
							}
						}
					}
				} else if field.State == 2 {
					growingCount++
				}
			}

			slog.Info("药园土地巡检与作业完成",
				"role_id", session.RoleID(),
				"total_fields", len(infoRes.Fields),
				"harvested", harvestedCount,
				"planted", plantedCount,
				"growing", growingCount,
				"gain_exp", totalGainExp,
				"gain_coins", totalGainCoins,
			)

			return nil
		},
	)
}


// NewPilgrimageRoutine 构造西天取经护送任务。
func NewPilgrimageRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"pilgrimage",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityNormal,
		20*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 西天取经护送与拦截保护")
			return nil
		},
	)
}

// NewCrystalMineRoutine 构造仙界矿山开采任务。
func NewCrystalMineRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"crystal_mine",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityLow,
		30*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 仙界矿山自动开采")
			return nil
		},
	)
}

// NewSpiritPoolRoutine 构造异兽与练池淬炼任务。
func NewSpiritPoolRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"spirit_pool",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityLow,
		60*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 异兽驯养与淬炼池提炼")
			return nil
		},
	)
}

// NewGourdRealmRoutine 构造壶中界炼丹与灵气合成任务。
func NewGourdRealmRoutine(dictRepo dictionary.Repository) scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"gourd_realm",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityNormal,
		30*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 壶中界灵气合成与材料提炼")
			if dictRepo != nil {
				// 通过数据字典校验基础合成丹药道具 (如气血包 ID 7)
				if item, err := dictRepo.GetItem(7); err == nil && item != nil {
					slog.Debug("壶中界数据字典校验就绪", "sample_item", item.Name)
				}
			}
			return nil
		},
	)
}

// NewRuneRefineRoutine 构造符文提炼与八卦炼丹任务。
func NewRuneRefineRoutine() scheduler.ActivityRoutine {
	return scheduler.NewBaseRoutine(
		"rune_refine",
		"farming",
		scheduler.ScheduleLoop,
		scheduler.PriorityLow,
		45*time.Minute,
		func(ctx context.Context, session *client.RoleSession, jitter *scheduler.Jitter) error {
			slog.Info("正在执行资源任务: 符文提炼与八卦炉炼化")
			return nil
		},
	)
}

