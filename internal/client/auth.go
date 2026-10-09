// Package client 提供特定区服角色的独立 TCP Socket 长连接会话、登录态维护与心跳保活状态机。
//
// @author Ateng
// @since 2026-10-08
package client

import (
	"log/slog"
	"strconv"
	"sync"

	"sxd-pie-ng/internal/platform"
	"sxd-pie-ng/internal/protocol"
)

// ApplyTicket 将双轨平台凭据一键装载至角色会话配置中。
func ApplyTicket(cfg *SessionConfig, ticket *platform.Ticket) {
	if cfg == nil || ticket == nil {
		return
	}
	if ticket.Platform != "" {
		cfg.Platform = ticket.Platform
	}
	if ticket.ServerID != "" {
		cfg.ServerID = ticket.ServerID
	}
	cfg.Code = ticket.MainServer.Code
	cfg.Time = ticket.MainServer.Time
	cfg.Hash = ticket.MainServer.Hash
	cfg.Time1 = ticket.CrossServer.Time1
	cfg.Hash1 = ticket.CrossServer.Hash1
}

// DefaultAuthenticator 为角色会话提供标准登录认证握手与回包解析调度。
func DefaultAuthenticator(s *RoleSession) error {
	var initOnce sync.Once
	sendInitPackets := func() {
		initOnce.Do(func() {
			initPackets := [][]byte{
				{0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x00},
				{0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x63, 0x00, 0x00, 0x00, 0x48},
				{0x00, 0x00, 0x00, 0x08, 0x00, 0xa5, 0x00, 0x00, 0x00, 0x00, 0x00, 0x63},
				{0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x02, 0x00, 0xa5, 0x00, 0x00},
			}
			go func() {
				for _, ipkt := range initPackets {
					_ = s.SendRaw(ipkt)
				}
			}()
		})
	}

	// 1. 注册主服 0x0000 核心角色资产包与初始化触发序列
	s.RegisterHandler(protocol.ActionIDPlayerLogin, func(p *protocol.Packet) {
		res, err := protocol.ParsePlayerLoginResult(p.Payload)
		if err == nil && res.RoleName != "" {
			s.UpdatePlayerState(func(ps *PlayerState) {
				ps.Stamina = int(res.Stamina)
				ps.MaxStamina = int(res.MaxStamina)
				ps.Coins = res.Coins
				ps.Ingots = int64(res.Ingots)
				ps.Level = int(res.Level)
			})
			slog.Info("主服全量角色资产同步成功",
				"role_id", s.RoleID(),
				"role_name", res.RoleName,
				"level", res.Level,
				"stamina", res.Stamina,
				"max_stamina", res.MaxStamina,
				"ingots", res.Ingots,
				"coins", res.Coins,
			)

			sendInitPackets()
		}
	})

	// 注册跨服 0x005E 登录握手与初始化序列
	s.RegisterHandler(protocol.ActionIDStLogin, func(p *protocol.Packet) {
		res, err := protocol.ParseStLoginResult(p.Payload)
		if err == nil {
			slog.Info("跨服登录认证成功", "player_id", res.PlayerID, "server_time", res.ServerTime)
			sendInitPackets()
		}
	})

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
				case protocol.PlayerPropMaxPower:
					ps.MaxStamina = int(val)
				case protocol.PlayerPropCoins:
					ps.Coins = int64(val)
				case protocol.PlayerPropIngot:
					ps.Ingots = int64(val)
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

	// 4. 组装并发送登录握手请求包
	// 优先支持 Mod_Player_Base 0x0000 (主服原生登录与全量资产获取)
	var pkt *protocol.Packet
	var err error

	if s.cfg.Username != "" && s.cfg.Hash != "" && s.cfg.Time != 0 {
		pkt, err = protocol.BuildPlayerLoginPacket(protocol.PlayerLoginRequest{
			Username:   s.cfg.Username,
			Hash:       s.cfg.Hash,
			Time:       strconv.FormatInt(int64(s.cfg.Time), 10),
			Source:     "sxd_baidu_pinpai_bt",
			Platform:   "疯玩",
			ClientType: "web",
		})
	} else if s.cfg.Time1 != 0 && s.cfg.Hash1 != "" {
		normalizedServerID := platform.NormalizeServerID(s.cfg.Platform, s.cfg.ServerID)
		pkt, err = protocol.BuildStLoginPacket(protocol.StLoginRequest{
			ServerID:   normalizedServerID,
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

	state := s.GetPlayerState()
	slog.Info("角色登录认证握手成功，进入激活态",
		"role_id", s.RoleID(),
		"role_name", s.RoleName(),
		"stamina", state.Stamina,
		"coins", state.Coins,
	)

	return nil
}
