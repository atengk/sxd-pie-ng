// Package platform 提供多运营平台 Web 自动化认证、凭据换取与区服标识归一化驱动层。
//
// @author Ateng
// @since 2026-10-08
package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormalizeServerID(t *testing.T) {
	tests := []struct {
		platform string
		input    string
		expected string
	}{
		{"fengwan", "s813", "fengwanyx_s813"},
		{"fengwanyx", "813", "fengwanyx_s813"},
		{"FENGWAN", "S813", "fengwanyx_s813"},
		{"fengwan", "fengwanyx_s813", "fengwanyx_s813"},
		{"xd", "s100", "s100"},
		{"other", "", ""},
	}

	for _, tt := range tests {
		got := NormalizeServerID(tt.platform, tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeServerID(%q, %q) = %q; want %q", tt.platform, tt.input, got, tt.expected)
		}
	}
}

func TestFengwanAuthenticator_Login(t *testing.T) {
	// 启动本地 Mock Web 服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login.php":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			if r.FormValue("username") != "test_user" || r.FormValue("password") != "test_pass" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// 设置登录态 Cookie
			http.SetCookie(w, &http.Cookie{
				Name:  "fengwanyx_auth",
				Value: "mock_auth_token",
				Path:  "/",
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("login success"))

		case "/entergame.php":
			server := r.URL.Query().Get("server")
			if server != "s813" {
				http.Error(w, "unknown server", http.StatusBadRequest)
				return
			}
			// 设置动态时间戳与哈希 Cookie
			http.SetCookie(w, &http.Cookie{
				Name:  "login_time_sxd_s813",
				Value: "1791470346",
				Path:  "/",
			})
			http.SetCookie(w, &http.Cookie{
				Name:  "login_hash_sxd_s813",
				Value: "mock_hash_f70c7ac2",
				Path:  "/",
			})
			// 302 重定向到游戏入口
			targetURL := "http://s813.sxd.fengwanyx.3fangyuan.com/index.php?code=MOCK_CODE_0SYD"
			w.Header().Set("Location", targetURL)
			w.WriteHeader(http.StatusFound)

		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	auth, err := NewFengwanAuthenticator(
		WithBaseEndpoints(mockServer.URL+"/login.php", mockServer.URL+"/entergame.php"),
	)
	if err != nil {
		t.Fatalf("NewFengwanAuthenticator failed: %v", err)
	}

	ctx := context.Background()
	ticket, err := auth.Login(ctx, "test_user", "test_pass", "s813")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if ticket.ServerID != "fengwanyx_s813" {
		t.Errorf("expected ServerID 'fengwanyx_s813', got %q", ticket.ServerID)
	}
	if ticket.MainServer.Code != "MOCK_CODE_0SYD" {
		t.Errorf("expected Code 'MOCK_CODE_0SYD', got %q", ticket.MainServer.Code)
	}
	if ticket.CrossServer.Time1 != 1791470346 {
		t.Errorf("expected Time1 1791470346, got %d", ticket.CrossServer.Time1)
	}
	if ticket.CrossServer.Hash1 != "mock_hash_f70c7ac2" {
		t.Errorf("expected Hash1 'mock_hash_f70c7ac2', got %q", ticket.CrossServer.Hash1)
	}
	if ticket.IsExpired() {
		t.Errorf("new ticket should not be expired")
	}
}

func TestTicketCache_Lifecycle(t *testing.T) {
	cache := NewTicketCache(10 * time.Minute)

	// 注册一个虚拟平台驱动
	mockAuth := &mockAuthenticator{
		platform: "test_plat",
	}
	cache.RegisterAuthenticator(mockAuth)

	ctx := context.Background()

	// 1. 首次获取触发换票
	tk1, err := cache.GetOrFetch(ctx, "test_plat", "user1", "pass1", "s10")
	if err != nil {
		t.Fatalf("GetOrFetch failed: %v", err)
	}
	if mockAuth.callCount != 1 {
		t.Errorf("expected 1 login call, got %d", mockAuth.callCount)
	}

	// 2. 二次获取走缓存
	tk2, err := cache.GetOrFetch(ctx, "test_plat", "user1", "pass1", "s10")
	if err != nil {
		t.Fatalf("GetOrFetch (cached) failed: %v", err)
	}
	if mockAuth.callCount != 1 {
		t.Errorf("expected still 1 login call, got %d", mockAuth.callCount)
	}
	if tk1 != tk2 {
		t.Errorf("expected same ticket instance from cache")
	}

	// 3. 显式失效后重新换票
	cache.Invalidate("test_plat", "user1", "s10")
	_, err = cache.GetOrFetch(ctx, "test_plat", "user1", "pass1", "s10")
	if err != nil {
		t.Fatalf("GetOrFetch after invalidate failed: %v", err)
	}
	if mockAuth.callCount != 2 {
		t.Errorf("expected 2 login calls after invalidate, got %d", mockAuth.callCount)
	}
}

type mockAuthenticator struct {
	platform  string
	callCount int
}

func (m *mockAuthenticator) Platform() string {
	return m.platform
}

func (m *mockAuthenticator) Login(ctx context.Context, username, password, serverID string) (*Ticket, error) {
	m.callCount++
	return &Ticket{
		Platform: m.platform,
		ServerID: serverID,
		MainServer: MainServerTicket{
			Code: "code",
			Time: 100,
			Hash: "hash",
		},
		CrossServer: CrossServerTicket{
			Time1: 100,
			Hash1: "hash1",
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}, nil
}
