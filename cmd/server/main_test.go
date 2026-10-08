// Package main 包含主程序单元测试用例。
//
// @author Ateng
// @since 2026-10-08
package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunVersion(t *testing.T) {
	err := run(context.Background(), []string{"-v"})
	if err != nil {
		t.Fatalf("run with -v failed: %v", err)
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	err := run(context.Background(), []string{"-config", "non_existent_config.yaml"})
	if err == nil {
		t.Fatal("expected error with non-existent config file")
	}
}

func TestRun_FullBootstrapAndGracefulShutdown(t *testing.T) {
	// 构造一个在端口 0 (由系统动态分配) 运行的临时测试配置
	testYaml := `
server:
  host: "127.0.0.1"
  port: 0

accounts:
  - platform: "test_plat"
    username: "user1"
    password: "pwd"
    roles:
      - server_id: "s1"
        role_name: "test_role"
        auto_login: false

scheduler:
  jitter:
    min_seconds: 0.1
    max_seconds: 0.2
`
	tmpConfig := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpConfig, []byte(testYaml), 0644); err != nil {
		t.Fatalf("write tmpConfig failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 100ms 后触发优雅停机
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := run(ctx, []string{"-config", tmpConfig})
	if err != nil {
		t.Fatalf("run bootstrap failed: %v", err)
	}
}
