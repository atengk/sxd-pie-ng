// Package web_test 验证 Web 控制台嵌入式前端托管、REST API 与日志缓冲区。
//
// @author Ateng
// @since 2026-10-08
package web_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"sxd-pie-ng/internal/web"
)

type mockRoleProvider struct {
	roles []web.RoleInfo
}

func (m *mockRoleProvider) ListRoles() []web.RoleInfo {
	return m.roles
}

func TestWebServer_LifecycleAndAPIs(t *testing.T) {
	logBuf := web.NewLogBuffer(10)
	logger := slog.New(logBuf)
	logger.Info("系统初始化完成", "module", "main")
	logger.Warn("测试告警条目", "code", 101)

	provider := &mockRoleProvider{
		roles: []web.RoleInfo{
			{
				RoleID:   "role-101",
				RoleName: "测试剑灵角色",
				ServerID: "s1",
				State:    "Active",
			},
		},
	}

	srv := web.NewServer("127.0.0.1", 0, provider, logBuf, "v1.0.0-test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Server.Start failed: %v", err)
	}
	defer func() {
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer sCancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	addr := srv.Addr()
	if addr == "" {
		t.Fatal("expected non-empty bound address")
	}

	baseURL := fmt.Sprintf("http://%s", addr)

	// 1. 验证静态 HTML 资源内嵌托管
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "sxd-pie-ng") {
		t.Errorf("expected HTML body to contain 'sxd-pie-ng'")
	}

	// 2. 验证 /api/health 健康检查接口
	healthResp, err := http.Get(baseURL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health failed: %v", err)
	}
	defer healthResp.Body.Close()
	var healthData map[string]any
	_ = json.NewDecoder(healthResp.Body).Decode(&healthData)
	if healthData["status"] != "ok" || healthData["version"] != "v1.0.0-test" {
		t.Errorf("unexpected health response: %+v", healthData)
	}

	// 3. 验证 /api/roles 角色列表接口
	rolesResp, err := http.Get(baseURL + "/api/roles")
	if err != nil {
		t.Fatalf("GET /api/roles failed: %v", err)
	}
	defer rolesResp.Body.Close()
	var rolesData []web.RoleInfo
	_ = json.NewDecoder(rolesResp.Body).Decode(&rolesData)
	if len(rolesData) != 1 || rolesData[0].RoleID != "role-101" {
		t.Errorf("unexpected roles response: %+v", rolesData)
	}

	// 4. 验证 /api/logs 实时日志接口
	logsResp, err := http.Get(baseURL + "/api/logs")
	if err != nil {
		t.Fatalf("GET /api/logs failed: %v", err)
	}
	defer logsResp.Body.Close()
	var logsData []web.LogEntry
	_ = json.NewDecoder(logsResp.Body).Decode(&logsData)
	if len(logsData) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d", len(logsData))
	}
	if logsData[0].Level != "INFO" || logsData[1].Level != "WARN" {
		t.Errorf("unexpected logs levels: %+v", logsData)
	}
}

func TestLogBuffer_Capacity(t *testing.T) {
	buf := web.NewLogBuffer(3)
	logger := slog.New(buf)

	logger.Info("1")
	logger.Info("2")
	logger.Info("3")
	logger.Info("4")
	logger.Info("5")

	logs := buf.GetLogs()
	if len(logs) != 3 {
		t.Fatalf("expected 3 entries after overflow, got %d", len(logs))
	}
	if logs[0].Message != "3" || logs[1].Message != "4" || logs[2].Message != "5" {
		t.Errorf("expected [3, 4, 5], got %+v", logs)
	}
}
