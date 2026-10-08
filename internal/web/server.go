// Package web 提供基于 Go 内嵌文件系统（go:embed）的 Web 控制台服务，支持本地配置与实时日志监视。
//
// @author Ateng
// @since 2026-10-08
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

// RoleInfo 角色状态展示结构体
type RoleInfo struct {
	RoleID   string `json:"role_id"`
	RoleName string `json:"role_name"`
	ServerID string `json:"server_id"`
	State    string `json:"state"`
}

// RoleProvider 角色信息提供器接口
type RoleProvider interface {
	ListRoles() []RoleInfo
}

// Server Web 控制台与管理 API 服务
type Server struct {
	host         string
	port         int
	roleProvider RoleProvider
	logBuffer    *LogBuffer
	version      string

	server   *http.Server
	listener net.Listener
	mu       sync.Mutex
	addr     string
}

// NewServer 创建新的 Web 控制台服务实例。
func NewServer(
	host string,
	port int,
	roleProvider RoleProvider,
	logBuffer *LogBuffer,
	version string,
) *Server {
	if host == "" {
		host = "127.0.0.1"
	}
	if logBuffer == nil {
		logBuffer = NewLogBuffer(200)
	}
	if version == "" {
		version = "dev"
	}

	return &Server{
		host:         host,
		port:         port,
		roleProvider: roleProvider,
		logBuffer:    logBuffer,
		version:      version,
	}
}

// Addr 返回服务器绑定的实际网络监听地址。
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

// Start 启动 Web 控制台 HTTP 监听。
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// 1. 挂载静态前端资源
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("web: failed to load embedded static fs: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	// 2. 挂载 REST API
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/roles", s.handleRoles)
	mux.HandleFunc("/api/logs", s.handleLogs)

	listenAddr := fmt.Sprintf("%s:%d", s.host, s.port)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("web: failed to listen on %s: %w", listenAddr, err)
	}

	s.mu.Lock()
	s.listener = ln
	s.addr = ln.Addr().String()
	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.mu.Unlock()

	slog.Info("Web 控制台已启动", "url", fmt.Sprintf("http://%s", s.addr))

	go func() {
		if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Web 控制台服务异常", "error", err)
		}
	}()

	return nil
}

// Shutdown 优雅关闭 Web 控制台 HTTP 服务。
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	srv := s.server
	s.mu.Unlock()

	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"status":  "ok",
		"version": s.version,
		"time":    time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request) {
	var roles []RoleInfo
	if s.roleProvider != nil {
		roles = s.roleProvider.ListRoles()
	}
	if roles == nil {
		roles = []RoleInfo{}
	}
	writeJSON(w, http.StatusOK, roles)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs := s.logBuffer.GetLogs()
	writeJSON(w, http.StatusOK, logs)
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
