// Package main 包含主程序单元测试用例。
//
// @author Ateng
// @since 2026-10-08
package main

import (
	"testing"
)

func TestRunVersion(t *testing.T) {
	err := run([]string{"-v"})
	if err != nil {
		t.Fatalf("run with -v failed: %v", err)
	}
}
