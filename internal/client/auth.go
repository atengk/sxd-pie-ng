// Package client 提供特定区服角色的独立 TCP Socket 长连接会话、登录态维护与心跳保活状态机。
//
// @author Ateng
// @since 2026-10-08
package client

import (
	"log/slog"

	"sxd-pie-ng/internal/protocol"
)

// DefaultAuthenticator 为角色会话提供标准登录认证握手与回包解析调度。
func DefaultAuthenticator(s *RoleSession) error {
	// 1. 组装并发送登录握手请求包 (优先支持 Mod_StLogin_Base 0x005E 真实凭据，向下兼容基础 Token 模式)
	var pkt *protocol.Packet
	var err error

	if s.cfg.ServerID != "" && s.cfg.Time1 != 0 && s.cfg.Hash1 != "" {
		pkt, err = protocol.BuildStLoginPacket(protocol.StLoginRequest{
			ServerID:   s.cfg.ServerID,
			ClientType: 4,
			RoleName:   s.RoleName(),
			Time1:      s.cfg.Time1,
			Hash1:      s.cfg.Hash1,
		})
	} else {
		pkt, err = protocol.BuildLoginPacket(s.RoleName(), s.cfg.Token)
	}
	if err != nil {
		return err
	}

	if err := s.Send(pkt); err != nil {
		return err
	}

	// 2. 注册角色属性与体力更新处理器
	s.RegisterHandler(protocol.ActionPlayerInfo, func(p *protocol.Packet) {
		r := protocol.NewReader(p.Payload)
		stamina, err := r.ReadUint32()
		if err == nil {
			s.UpdatePlayerState(func(ps *PlayerState) {
				ps.Stamina = int(stamina)
			})
			slog.Debug("接收到角色属性同步回包", "role_id", s.RoleID(), "stamina", stamina)
		}
	})

	// 注册 Mod_Player_Base 0x0300 动态更新通知
	s.RegisterHandler(protocol.ActionIDPlayerUpdateData, func(p *protocol.Packet) {
		r := protocol.NewReader(p.Payload)
		for r.Remaining() >= 5 {
			prop, err := r.ReadUint8()
			if err != nil {
				break
			}
			val, err := r.ReadInt32()
			if err != nil {
				break
			}
			s.UpdatePlayerState(func(ps *PlayerState) {
				switch prop {
				case protocol.PlayerPropPower:
					ps.Stamina = int(val)
				case protocol.PlayerPropCoins:
					ps.Coins = int64(val)
				case protocol.PlayerPropLevel:
					ps.Level = int(val)
				case protocol.PlayerPropVIPLevel:
					ps.VIP = int(val)
				case protocol.PlayerPropPackEmptyNum:
					ps.BagCapacity = int(val)
				}
			})
		}
	})

	// 3. 注册扫荡回包全局监听更新
	s.RegisterHandler(protocol.ActionMissionSweep, func(p *protocol.Packet) {
		res, err := protocol.ParseSweepResult(p.Payload)
		if err == nil && res.Success {
			s.AddRewards(res.GainExp, res.GainCoins)
		}
	})

	state := s.GetPlayerState()
	slog.Info("角色登录认证握手成功，进入激活态",
		"role_id", s.RoleID(),
		"role_name", s.RoleName(),
		"stamina", state.Stamina,
		"coins", state.Coins,
	)

	return nil
}
