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
	"syscall"
)

var (
	// Version 编译期注入的版本号
	Version = "dev"
	// Commit 编译期注入的 Git Commit 散列
	Commit = "none"
	// Date 编译期注入的构建日期
	Date = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("服务异常退出", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
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

	slog.Info("正在启动神仙道助手 (sxd-pie-ng)...",
		"version", Version,
		"config", *configPath,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	slog.Info("服务就绪，等待运行信号 (按 Ctrl+C 退出)")
	<-ctx.Done()

	slog.Info("正在优雅关闭服务...")
	return nil
}
