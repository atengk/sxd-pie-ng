// Package main 提供真实游戏服务器快速登录与角色信息嗅探探针工具。
//
// @author Ateng
// @since 2026-10-10
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var opts ProbeOptions
	flag.StringVar(&opts.ConfigPath, "config", "configs/config.yaml", "配置文件路径")
	flag.StringVar(&opts.ConfigPath, "c", "configs/config.yaml", "配置文件路径 (简写)")
	flag.StringVar(&opts.ServerAddr, "server", "", "游戏区服网关 TCP 地址 (例如 49.232.196.100:8381)")
	flag.StringVar(&opts.ServerAddr, "s", "", "游戏区服网关 TCP 地址 (简写)")
	flag.StringVar(&opts.Platform, "platform", "", "所属运营平台标识 (例如 fengwan)")
	flag.StringVar(&opts.Platform, "p", "", "所属运营平台标识 (简写)")
	flag.StringVar(&opts.Username, "username", "", "平台登录通行证账号")
	flag.StringVar(&opts.Username, "u", "", "平台登录通行证账号 (简写)")
	flag.StringVar(&opts.Password, "password", "", "平台登录密码 (用于自治 Web 换票)")
	flag.StringVar(&opts.Password, "pwd", "", "平台登录密码 (简写)")
	flag.StringVar(&opts.ServerID, "server-id", "", "游戏区服标识 (例如 s813)")
	flag.StringVar(&opts.ServerID, "sid", "", "游戏区服标识 (简写)")
	flag.StringVar(&opts.RoleName, "role", "", "角色显示名称 (例如 梦一场)")
	flag.StringVar(&opts.Token, "token", "", "平台授权票据 (Token)")
	flag.StringVar(&opts.Hash, "hash", "", "本服主网关验签哈希散列")
	flag.DurationVar(&opts.Timeout, "timeout", 10*time.Second, "探针全局执行超时时间")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "启用虚拟沙箱探测模式 (离线与CI保障)")
	flag.BoolVar(&opts.OneShot, "oneshot", true, "探测输出报告后立即退出 (设为 false 持续保持长连接)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	report, err := RunProbe(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[PROBE ERROR] 诊断探测未通过: %v\n", err)
		os.Exit(1)
	}

	if !report.Success {
		os.Exit(1)
	}
}
