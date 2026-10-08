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

// RoleSession 代表与游戏服务器建立的长连接角色会话。
type RoleSession struct {
	cfg SessionConfig

	state   SessionState
	stateMu sync.RWMutex

	conn    net.Conn
	writeMu sync.Mutex

	handlers  map[uint16][]func(*protocol.Packet)
	handlerMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	reconnectAttempts int
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
	if cfg.Dialer == nil {
		if cfg.ServerAddr == "" || cfg.ServerAddr == "mock" || cfg.ServerAddr == "dry-run" {
			cfg.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
				c1, c2 := net.Pipe()
				go func() {
					buf := make([]byte, 1024)
					for {
						_, err := c2.Read(buf)
						if err != nil {
							_ = c2.Close()
							return
						}
					}
				}()
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
		cfg:      cfg,
		state:    StateDisconnected,
		handlers: make(map[uint16][]func(*protocol.Packet)),
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
