// Package platform 提供多运营平台 Web 自动化认证、凭据换取与区服标识归一化驱动层。
//
// @author Ateng
// @since 2026-10-08
package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTicketFromIni(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "user.ini")

	content := `
[kongyu]
url=http://s813.sxd.fengwanyx.3fangyuan.com/
code=VUtXbqlGuR0
time=1791472200
hash=6d16f4f0063942a955debdc8ac56140b
time1=1791472200
hash1=922c09ecf6fdc148f9fbe681aa7483b3
name=梦一场
servername=fengwanyx_s813
`
	if err := os.WriteFile(iniPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	ticket, err := LoadTicketFromIni(iniPath, "梦一场")
	if err != nil {
		t.Fatalf("LoadTicketFromIni failed: %v", err)
	}

	if ticket.ServerID != "fengwanyx_s813" {
		t.Errorf("expected ServerID 'fengwanyx_s813', got %q", ticket.ServerID)
	}
	if ticket.RoleName != "梦一场" {
		t.Errorf("expected RoleName '梦一场', got %q", ticket.RoleName)
	}
	if ticket.MainServer.Code != "VUtXbqlGuR0" {
		t.Errorf("expected Code 'VUtXbqlGuR0', got %q", ticket.MainServer.Code)
	}
	if ticket.CrossServer.Time1 != 1791472200 {
		t.Errorf("expected Time1 1791472200, got %d", ticket.CrossServer.Time1)
	}
	if ticket.CrossServer.Hash1 != "922c09ecf6fdc148f9fbe681aa7483b3" {
		t.Errorf("expected Hash1 '922c09ecf6fdc148f9fbe681aa7483b3', got %q", ticket.CrossServer.Hash1)
	}
}
