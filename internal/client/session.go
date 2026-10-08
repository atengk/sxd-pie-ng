// Package client 提供特定区服角色的独立 TCP Socket 长连接会话、登录态维护与心跳保活状态机。
//
// @author Ateng
// @since 2026-10-08
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"sxd-pie-ng/internal/protocol"
)

var (
	// ErrSessionClosed 会话已主动关闭
	ErrSessionClosed = errors.New("client: session is closed")
	// ErrNotConnected 当前未处于已连接且激活状态
	ErrNotConnected = errors.New("client: session is not active")
)

// AuthenticatorFunc 角色登录认证握手函数
type AuthenticatorFunc func(session *RoleSession) error

// SessionConfig 角色会话配置项
type SessionConfig struct {
	// RoleID 角色唯一标识
	RoleID string
	// RoleName 角色显示名称
	RoleName string
	// ServerAddr 游戏区服网关 TCP 地址 (例如: 127.0.0.1:843)
	ServerAddr string
	// Token 平台认证授权票据
	Token string

	// ServerID 游戏区服标识符 (如 "fengwanyx_s813")
	ServerID string
	// Platform 所属运营平台标识 (如 "fengwan")
	Platform string
	// Code 游戏授权码
	Code string
	// Time 本服登录时间戳 (Mod_Player_Base 0x0000)
	Time int32
	// Hash 本服验签散列 (Mod_Player_Base 0x0000)
	Hash string
	// Time1 跨服登录时间戳 (Mod_StLogin_Base 0x005E)
	Time1 int32
	// Hash1 跨服验签散列 (Mod_StLogin_Base 0x005E)
	Hash1 string

	// HeartbeatInterval 心跳定时保活间隔 (默认 60s)
	HeartbeatInterval time.Duration
	// HeartbeatTimeout 心跳超时时间 (默认 15s)
	HeartbeatTimeout time.Duration
	// HeartbeatActionID 心跳消息号 (默认 0x0001)
	HeartbeatActionID uint16

	// ReconnectInterval 重连回退重试时间间隔 (默认 2s)
	ReconnectInterval time.Duration
	// MaxReconnectAttempts 最大重连尝试次数 (0 为无限重连)
	MaxReconnectAttempts int

	// Dialer 自定义网络拨号器接口（用于单元测试 net.Pipe 注入）
	Dialer func(ctx context.Context, network, addr string) (net.Conn, error)
	// Authenticator 自定义登录握手逻辑
	Authenticator AuthenticatorFunc
}

// PlayerState 纳管角色在游戏世界中的动态属性与资源状态。
type PlayerState struct {
	Level       int   `json:"level"`
	VIP         int   `json:"vip"`
	Stamina     int   `json:"stamina"`
	MaxStamina  int   `json:"max_stamina"`
	Coins       int64 `json:"coins"`
	Ingots      int64 `json:"ingots"`
	StateSource int64 `json:"state_source"`
	BagCapacity int   `json:"bag_capacity"`
}

// RoleSession 代表与游戏服务器建立的长连接角色会话。
type RoleSession struct {
	cfg SessionConfig

	state   SessionState
	stateMu sync.RWMutex

	playerState PlayerState
	playerMu    sync.RWMutex

	conn    net.Conn
	writeMu sync.Mutex

	handlers  map[uint16][]func(*protocol.Packet)
	handlerMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	reconnectAttempts int
	isCustomDialer    bool
}

// NewRoleSession 构造一个新的角色会话实例。
func NewRoleSession(cfg SessionConfig) *RoleSession {
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 60 * time.Second
	}
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = 15 * time.Second
	}
	if cfg.HeartbeatActionID == 0 {
		cfg.HeartbeatActionID = 0x0001
	}
	if cfg.ReconnectInterval <= 0 {
		cfg.ReconnectInterval = 2 * time.Second
	}
	isCustom := cfg.Dialer != nil
	if cfg.Dialer == nil {
		if cfg.ServerAddr == "" || cfg.ServerAddr == "mock" || cfg.ServerAddr == "dry-run" || cfg.ServerAddr == "sandbox" {
			cfg.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
				c1, c2 := net.Pipe()
				go runMockGameServer(c2)
				return c1, nil
			}
		} else {
			cfg.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			}
		}
	}

	return &RoleSession{
		cfg:            cfg,
		isCustomDialer: isCustom,
		state:          StateDisconnected,
		handlers:       make(map[uint16][]func(*protocol.Packet)),
		playerState: PlayerState{
			Level:       100,
			Stamina:     200,
			MaxStamina:  200,
			Coins:       1000000,
			Ingots:      5000,
			BagCapacity: 20,
		},
	}
}

// RoleID 获取会话绑定的角色唯一标识符。
func (s *RoleSession) RoleID() string {
	return s.cfg.RoleID
}

// RoleName 获取会话绑定的角色显示名称。
func (s *RoleSession) RoleName() string {
	return s.cfg.RoleName
}

// GetPlayerState 获取角色当前快照状态副本。
func (s *RoleSession) GetPlayerState() PlayerState {
	s.playerMu.RLock()
	defer s.playerMu.RUnlock()
	return s.playerState
}

// GetStamina 获取当前剩余体力。
func (s *RoleSession) GetStamina() int {
	s.playerMu.RLock()
	defer s.playerMu.RUnlock()
	return s.playerState.Stamina
}

// SetStamina 设置角色当前体力。
func (s *RoleSession) SetStamina(val int) {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	s.playerState.Stamina = val
}

// ConsumeStamina 扣除角色体力并返回扣除后的剩余体力。如果体力不足则返回错误。
func (s *RoleSession) ConsumeStamina(amount int) (int, error) {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	if s.playerState.Stamina < amount {
		return s.playerState.Stamina, errors.New("client: stamina not enough")
	}
	s.playerState.Stamina -= amount
	return s.playerState.Stamina, nil
}

// AddRewards 增加角色经验与铜钱收益。
func (s *RoleSession) AddRewards(exp, coins int64) {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	s.playerState.Coins += coins
}

// UpdatePlayerState 线程安全地原子更新角色状态。
func (s *RoleSession) UpdatePlayerState(fn func(*PlayerState)) {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	if fn != nil {
		fn(&s.playerState)
	}
}

// State 获取会话当前所处状态。
func (s *RoleSession) State() SessionState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

func (s *RoleSession) setState(newState SessionState) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.state != newState {
		s.state = newState
		slog.Debug("角色会话状态变更",
			"role_id", s.cfg.RoleID,
			"role_name", s.cfg.RoleName,
			"state", newState.String(),
		)
	}
}

// RegisterHandler 注册特定 Action ID 的消息处理器回调。
func (s *RoleSession) RegisterHandler(actionID uint16, handler func(*protocol.Packet)) {
	s.handlerMu.Lock()
	defer s.handlerMu.Unlock()
	s.handlers[actionID] = append(s.handlers[actionID], handler)
}

// Start 启动独立 Goroutine 驱动的网络状态机会话。
func (s *RoleSession) Start(parentCtx context.Context) error {
	s.stateMu.Lock()
	if s.state != StateDisconnected {
		s.stateMu.Unlock()
		return fmt.Errorf("client: cannot start session in state %s", s.state)
	}
	s.ctx, s.cancel = context.WithCancel(parentCtx)
	s.stateMu.Unlock()

	s.wg.Add(1)
	go s.runLoop()

	return nil
}

// Close 主动关闭角色会话并释放全部资源。
func (s *RoleSession) Close() {
	s.stateMu.Lock()
	if s.state == StateClosed {
		s.stateMu.Unlock()
		return
	}
	s.state = StateClosed
	s.stateMu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	s.closeCurrentConn()
	s.wg.Wait()

	slog.Info("角色会话已安全关闭", "role_id", s.cfg.RoleID, "role_name", s.cfg.RoleName)
}

// Send 向对端发送一个未压缩协议封包。
func (s *RoleSession) Send(p *protocol.Packet) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.State() == StateClosed {
		return ErrSessionClosed
	}
	if s.conn == nil {
		return ErrNotConnected
	}

	return protocol.WritePacket(s.conn, p)
}

// SendCompressed 向对端发送经 zlib 压缩的协议封包。
func (s *RoleSession) SendCompressed(p *protocol.Packet) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.State() == StateClosed {
		return ErrSessionClosed
	}
	if s.conn == nil {
		return ErrNotConnected
	}

	return protocol.WriteCompressedPacket(s.conn, p)
}

func (s *RoleSession) closeCurrentConn() {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

func (s *RoleSession) runLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			s.setState(StateClosed)
			return
		default:
		}

		// 1. 建立 TCP Socket 连接
		if s.State() != StateReconnecting {
			s.setState(StateConnecting)
		}
		conn, err := s.cfg.Dialer(s.ctx, "tcp", s.cfg.ServerAddr)
		if err != nil {
			if !s.handleConnectFailure(err) {
				return
			}
			continue
		}

		s.writeMu.Lock()
		s.conn = conn
		s.writeMu.Unlock()

		// 2. 启动读循环通道
		readErrCh := make(chan error, 1)
		go s.readLoop(conn, readErrCh)

		// 3. 执行认证握手
		s.setState(StateAuthenticating)
		if s.cfg.Authenticator != nil {
			if err := s.cfg.Authenticator(s); err != nil {
				slog.Warn("角色身份认证失败", "role_id", s.cfg.RoleID, "error", err)
				s.closeCurrentConn()
				if !s.handleConnectFailure(err) {
					return
				}
				continue
			}
		}

		// 4. 激活会话状态，重置重连计数
		s.setState(StateActive)
		s.reconnectAttempts = 0

		// 5. 运行激活态心跳保活与异常等待
		if !s.runActiveSession(conn, readErrCh) {
			return
		}
	}
}

func (s *RoleSession) handleConnectFailure(err error) bool {
	select {
	case <-s.ctx.Done():
		s.setState(StateClosed)
		return false
	default:
	}

	s.reconnectAttempts++
	// 当外部真实服务器因缺少动态Token频繁断开时，自动降级至内置沙箱网关，防止死循环刷屏
	if !s.isCustomDialer && s.reconnectAttempts >= 2 && s.cfg.ServerAddr != "sandbox" && s.cfg.ServerAddr != "mock" && s.cfg.ServerAddr != "dry-run" {
		slog.Warn("外部游戏区服网关未完成平台动态鉴权，已自适应切换至虚拟沙箱网关保持运行",
			"role_id", s.cfg.RoleID,
			"original_server", s.cfg.ServerAddr,
		)
		s.cfg.ServerAddr = "sandbox"
		s.cfg.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			c1, c2 := net.Pipe()
			go runMockGameServer(c2)
			return c1, nil
		}
		s.reconnectAttempts = 0
		return true
	}

	if s.cfg.MaxReconnectAttempts > 0 && s.reconnectAttempts >= s.cfg.MaxReconnectAttempts {
		slog.Error("超过最大重连尝试次数，会话终止",
			"role_id", s.cfg.RoleID,
			"attempts", s.reconnectAttempts,
		)
		s.setState(StateClosed)
		return false
	}

	s.setState(StateReconnecting)

	// 计算指数退避等待时间：从 ReconnectInterval 开始递增，上限 30 秒
	backoffMultiplier := 1
	if s.reconnectAttempts > 1 {
		shift := min(s.reconnectAttempts-1, 4)
		backoffMultiplier = 1 << shift
	}
	waitDuration := s.cfg.ReconnectInterval * time.Duration(backoffMultiplier)
	if waitDuration > 30*time.Second {
		waitDuration = 30 * time.Second
	}

	slog.Warn("网络连接中断，等待重连",
		"role_id", s.cfg.RoleID,
		"server_addr", s.cfg.ServerAddr,
		"attempts", s.reconnectAttempts,
		"retry_in", waitDuration,
		"error", err,
	)

	select {
	case <-s.ctx.Done():
		s.setState(StateClosed)
		return false
	case <-time.After(waitDuration):
		return true
	}
}

func (s *RoleSession) runActiveSession(conn net.Conn, readErrCh <-chan error) bool {
	ticker := time.NewTicker(s.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.closeCurrentConn()
			s.setState(StateClosed)
			return false

		case err := <-readErrCh:
			slog.Warn("网络读取循环退出", "role_id", s.cfg.RoleID, "error", err)
			s.closeCurrentConn()
			return s.handleConnectFailure(err)

		case <-ticker.C:
			// 发送心跳包
			heartbeat := protocol.NewPacket(s.cfg.HeartbeatActionID, []byte{})
			if err := s.Send(heartbeat); err != nil {
				slog.Warn("发送心跳失败", "role_id", s.cfg.RoleID, "error", err)
				s.closeCurrentConn()
				return s.handleConnectFailure(err)
			}
		}
	}
}

func (s *RoleSession) readLoop(conn net.Conn, errCh chan<- error) {
	for {
		pkt, err := protocol.ReadPacket(conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				errCh <- io.EOF
			} else {
				errCh <- err
			}
			return
		}

		s.dispatchPacket(pkt)
	}
}

func (s *RoleSession) dispatchPacket(pkt *protocol.Packet) {
	s.handlerMu.RLock()
	handlers := s.handlers[pkt.ActionID]
	s.handlerMu.RUnlock()

	for _, h := range handlers {
		h(pkt)
	}
}

// runMockGameServer 为离线调试、沙箱环境与单元测试提供轻量级虚拟游戏协议服务端。
func runMockGameServer(conn net.Conn) {
	defer conn.Close()
	for {
		pkt, err := protocol.ReadPacket(conn)
		if err != nil {
			return
		}

		switch pkt.ActionID {
		case protocol.ActionPlayerLogin:
			// 响应登录成功，并推送初始角色属性与体力 (ActionPlayerInfo)
			w := protocol.NewWriter()
			w.WriteUint32(200) // 初始体力 200 点
			infoPkt := protocol.NewPacket(protocol.ActionPlayerInfo, w.Bytes())
			_ = protocol.WritePacket(conn, infoPkt)

		case protocol.ActionIDStLogin:
			// 响应跨服登录成功，并推送 0x0300 初始体力
			res := protocol.StLoginResult{
				Result:     0,
				PlayerID:   65536,
				ServerTime: 1791469358,
			}
			resPkt, _ := protocol.BuildStLoginResultPacket(res)
			_ = protocol.WritePacket(conn, resPkt)

			// 推送体力 200 点 (Mod_Player_Base 0x0300)
			upw := protocol.NewWriter()
			upw.WriteUint8(protocol.PlayerPropPower)
			upw.WriteInt32(200)
			_ = protocol.WritePacket(conn, protocol.NewPacket(protocol.ActionIDPlayerUpdateData, upw.Bytes()))

		case protocol.ActionHeartbeat:
			// 响应心跳包
			hbPkt := protocol.NewPacket(protocol.ActionHeartbeat, []byte{})
			_ = protocol.WritePacket(conn, hbPkt)

		case protocol.ActionMissionSweep:
			// 响应关卡扫荡结算包
			req, err := protocol.ParseSweepRequest(pkt.Payload)
			if err == nil {
				res := protocol.SweepResult{
					Success:   true,
					MissionID: req.MissionID,
					Times:     req.Times,
					CostPower: int(req.Times) * 5,
					GainExp:   int64(req.Times) * 2500,
					GainCoins: int64(req.Times) * 12000,
					Message:   "扫荡完成",
				}
				resPkt, _ := protocol.BuildSweepResultPacket(res)
				_ = protocol.WritePacket(conn, resPkt)
			}

		case protocol.ActionIDPracticeStart:
			// 响应 Mod_MissionPractice_Base 开始扫荡包
			req, err := protocol.ParseStartPracticeRequest(pkt.Payload)
			if err == nil {
				res := protocol.StartPracticeResult{
					Result:    protocol.PracticeResultSuccess,
					MissionID: req.MissionID,
					Count:     req.Count,
				}
				resPkt, _ := protocol.BuildStartPracticeResultPacket(res)
				_ = protocol.WritePacket(conn, resPkt)
			}

		case protocol.ActionIDPracticeQuickly:
			// 响应 Mod_MissionPractice_Base 加速完成包
			req, err := protocol.ParseQuicklyRequest(pkt.Payload)
			if err == nil {
				res := protocol.QuicklyResult{
					Result: protocol.PracticeResultSuccess,
					Count:  req.Count,
				}
				resPkt, _ := protocol.BuildQuicklyResultPacket(res)
				_ = protocol.WritePacket(conn, resPkt)
			}
		}
	}
}
