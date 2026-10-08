// Package main 提供神仙道助手服务端主入口与运行时生命周期管理。
//
// @author Ateng
// @since 2026-10-08
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"sxd-pie-ng/internal/client"
	"sxd-pie-ng/internal/config"
	"sxd-pie-ng/internal/scheduler"
	"sxd-pie-ng/internal/web"
)

var (
	// Version 编译期注入的版本号
	Version = "dev"
	// Commit 编译期注入的 Git Commit 散列
	Commit = "none"
	// Date 编译期注入的构建日期
	Date = "unknown"
)

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		_ = h.Handle(ctx, r)
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var next []slog.Handler
	for _, h := range m.handlers {
		next = append(next, h.WithAttrs(attrs))
	}
	return &multiHandler{handlers: next}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	var next []slog.Handler
	for _, h := range m.handlers {
		next = append(next, h.WithGroup(name))
	}
	return &multiHandler{handlers: next}
}

type sessionManager struct {
	sessions []*client.RoleSession
	roles    []web.RoleInfo
	mu       sync.RWMutex
}

func (m *sessionManager) ListRoles() []web.RoleInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]web.RoleInfo, len(m.roles))
	for i, r := range m.roles {
		res[i] = r
		if i < len(m.sessions) && m.sessions[i] != nil {
			res[i].State = m.sessions[i].State().String()
		}
	}
	return res
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("服务异常退出", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("sxd-pie-ng", flag.ContinueOnError)
	versionFlag := fs.Bool("v", false, "打印版本信息并退出")
	configPath := fs.String("config", "configs/config.yaml", "配置文件路径")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *versionFlag {
		fmt.Printf("sxd-pie-ng version %s (commit: %s, built: %s)\n", Version, Commit, Date)
		return nil
	}

	// 1. 初始化环形日志收集器与受管日志
	logBuf := web.NewLogBuffer(200)
	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	multiH := &multiHandler{
		handlers: []slog.Handler{stdoutHandler, logBuf},
	}
	slog.SetDefault(slog.New(multiH))

	targetConfig := *configPath
	if _, err := os.Stat(targetConfig); os.IsNotExist(err) && targetConfig == "configs/config.yaml" {
		targetConfig = "configs/config.example.yaml"
	}

	slog.Info("正在加载系统配置...", "file", targetConfig)
	cfg, err := config.Load(targetConfig)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 初始化角色会话与多角色调度器
	mgr := &sessionManager{}
	dispatcher := scheduler.NewDispatcher()

	for _, acc := range cfg.Accounts {
		for _, role := range acc.Roles {
			roleID := fmt.Sprintf("%s-%s", role.ServerID, role.RoleName)
			roleCfg := client.SessionConfig{
				RoleID:     roleID,
				RoleName:   role.RoleName,
				ServerAddr: "127.0.0.1:843", // 默认网关占位
			}
			sess := client.NewRoleSession(roleCfg)
			mgr.sessions = append(mgr.sessions, sess)
			mgr.roles = append(mgr.roles, web.RoleInfo{
				RoleID:   roleID,
				RoleName: role.RoleName,
				ServerID: role.ServerID,
				State:    sess.State().String(),
			})

			jitterCfg := scheduler.JitterConfig{
				MinDuration: time.Duration(cfg.Scheduler.Jitter.MinSeconds * float64(time.Second)),
				MaxDuration: time.Duration(cfg.Scheduler.Jitter.MaxSeconds * float64(time.Second)),
			}
			roleSched, _ := dispatcher.AddRole(roleID, sess, jitterCfg)

			if roleSched != nil {
				if cfg.Scheduler.Routines["herb_garden"] {
					_ = roleSched.Register(scheduler.NewFuncRoutine(
						"herb_garden",
						scheduler.PriorityNormal,
						15*time.Second,
						func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
							slog.Info("正在执行日常任务: 药园种植巡检", "role_id", roleID)
							return nil
						},
					))
				}
				if cfg.Scheduler.Routines["lucky_star"] {
					_ = roleSched.Register(scheduler.NewFuncRoutine(
						"lucky_star",
						scheduler.PriorityNormal,
						30*time.Second,
						func(ctx context.Context, s *client.RoleSession, j *scheduler.Jitter) error {
							slog.Info("正在执行日常任务: 帮派吉星高照", "role_id", roleID)
							return nil
						},
					))
				}
			}

			if role.AutoLogin {
				_ = sess.Start(ctx)
			}
		}
	}

	dispatcher.StartAll(ctx)

	// 3. 启动 Web 控制台服务
	webServer := web.NewServer(cfg.Server.Host, cfg.Server.Port, mgr, logBuf, Version)
	if err := webServer.Start(ctx); err != nil {
		return fmt.Errorf("启动 Web 控制台失败: %w", err)
	}

	slog.Info("服务就绪，等待运行信号 (按 Ctrl+C 退出)")
	<-ctx.Done()

	slog.Info("收到停机信号，正在优雅关闭服务...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	_ = webServer.Shutdown(shutdownCtx)
	dispatcher.StopAll()
	for _, sess := range mgr.sessions {
		sess.Close()
	}

	slog.Info("神仙道助手已安全退出")
	return nil
}
