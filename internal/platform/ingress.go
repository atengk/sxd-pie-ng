// Package platform 提供多运营平台 Web 自动化认证、凭据换取与区服标识归一化驱动层。
//
// @author Ateng
// @since 2026-10-08
package platform

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// LoadTicketFromIni 从外部 user.ini 文件解析特定角色的短期鉴权凭据 (Ticket Ingress)。
func LoadTicketFromIni(iniPath, targetRoleName string) (*Ticket, error) {
	if iniPath == "" {
		// 探测常见路径
		candidates := []string{
			`C:\software\疯玩神仙道\user.ini`,
			`user.ini`,
			`configs/user.ini`,
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				iniPath = c
				break
			}
		}
	}

	if iniPath == "" {
		return nil, fmt.Errorf("platform: user.ini not found")
	}

	rawBytes, err := os.ReadFile(iniPath)
	if err != nil {
		return nil, fmt.Errorf("platform: failed to open ini: %w", err)
	}

	text := string(rawBytes)
	if !utf8.Valid(rawBytes) {
		if utf8Decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(rawBytes); err == nil && len(utf8Decoded) > 0 {
			text = string(utf8Decoded)
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	currentSection := ""
	sectionData := make(map[string]string)
	sections := make(map[string]map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentSection != "" && len(sectionData) > 0 {
				sections[currentSection] = sectionData
				sectionData = make(map[string]string)
			}
			currentSection = strings.Trim(line, "[]")
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.ToLower(strings.TrimSpace(parts[0]))
			v := strings.TrimSpace(parts[1])
			sectionData[k] = v
		}
	}
	if currentSection != "" && len(sectionData) > 0 {
		sections[currentSection] = sectionData
	}

	// 匹配指定角色
	for secName, data := range sections {
		roleName := data["name"]
		if targetRoleName != "" && roleName != targetRoleName && secName != targetRoleName {
			continue
		}

		timeVal, _ := strconv.ParseInt(data["time"], 10, 32)
		time1Val, _ := strconv.ParseInt(data["time1"], 10, 32)
		if time1Val == 0 {
			time1Val = timeVal
		}

		serverID := data["servername"]
		if serverID == "" {
			serverID = "fengwanyx_s813"
		}

		ticket := &Ticket{
			Platform:    "fengwan",
			ServerID:    NormalizeServerID("fengwan", serverID),
			RawServerID: serverID,
			RoleName:    roleName,
			GatewayURL:  data["url"],
			MainServer: MainServerTicket{
				Code: data["code"],
				Time: int32(timeVal),
				Hash: data["hash"],
			},
			CrossServer: CrossServerTicket{
				Time1: int32(time1Val),
				Hash1: data["hash1"],
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(90 * time.Minute),
		}
		return ticket, nil
	}

	return nil, fmt.Errorf("platform: role %q not found in %s", targetRoleName, iniPath)
}
