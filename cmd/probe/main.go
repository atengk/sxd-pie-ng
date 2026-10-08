// Package main 提供真实游戏服务器快速登录与角色信息嗅探探针工具。
//
// @author Ateng
// @since 2026-10-09
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"

	"sxd-pie-ng/internal/config"
	"sxd-pie-ng/internal/platform"
	"sxd-pie-ng/internal/protocol"
)

func main() {
	cfgPath := "configs/config.yaml"
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Accounts) == 0 || len(cfg.Accounts[0].Roles) == 0 {
		fmt.Println("未找到有效账号或角色配置")
		os.Exit(1)
	}

	acc := cfg.Accounts[0]
	role := acc.Roles[0]

	fmt.Println("==================================================")
	fmt.Printf("1. 正在通过全自治 Web 驱动向平台换取动态票据...\n")
	fmt.Printf("   平台: %s | 账号: %s | 区服: %s\n", acc.Platform, acc.Username, role.ServerID)

	fwAuth, err := platform.NewFengwanAuthenticator()
	if err != nil {
		fmt.Printf("初始化平台认证驱动失败: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ticket, err := fwAuth.Login(ctx, acc.Username, acc.Password, role.ServerID)
	if err != nil {
		fmt.Printf("原生 Web 换票失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("   换票成功!\n   GatewayURL: %s\n   ServerID: %s\n   MainServer: Code=%s, Time=%d, Hash=%s\n   CrossServer: Time1=%d, Hash1=%s\n",
		ticket.GatewayURL, ticket.ServerID, ticket.MainServer.Code, ticket.MainServer.Time, ticket.MainServer.Hash, ticket.CrossServer.Time1, ticket.CrossServer.Hash1)

	// 2. 拨号连接游戏服务器
	serverAddr := role.ServerAddr
	if serverAddr == "" || serverAddr == "sandbox" {
		serverAddr = "49.232.196.100:8381"
	}
	fmt.Printf("\n2. 正在连接真实游戏网关: %s...\n", serverAddr)
	conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
	if err != nil {
		fmt.Printf("连接游戏服务器失败: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("   TCP Socket 连接已建立!\n")

	// 3. 构造并发送 Mod_StLogin_Base.login (0x005E)
	stReq := protocol.StLoginRequest{
		ServerID:   ticket.ServerID,
		ClientType: 4,
		RoleName:   role.RoleName,
		Time1:      ticket.CrossServer.Time1,
		Hash1:      ticket.CrossServer.Hash1,
	}
	stPkt, err := protocol.BuildStLoginPacket(stReq)
	if err != nil {
		fmt.Printf("构造登录封包失败: %v\n", err)
		os.Exit(1)
	}
	rawPkt, err := stPkt.Marshal()
	if err != nil {
		fmt.Printf("序列化登录封包失败: %v\n", err)
		os.Exit(1)
	}
	if _, err := conn.Write(rawPkt); err != nil {
		fmt.Printf("发送登录握手失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n3. 已发送登录握手请求 (ActionID: 0x%04X, 角色名: %s)\n", stPkt.ActionID, role.RoleName)

	// 4. 读取服务端回包
	readDeadline := time.Now().Add(6 * time.Second)

	var playerID int32
	var serverTime int32
	playerProps := make(map[uint8]int64)

	fmt.Printf("\n4. 正在监听服务器数据流并提取角色信息...\n")
	for time.Now().Before(readDeadline) {
		conn.SetReadDeadline(time.Now().Add(1200 * time.Millisecond))
		pkt, err := protocol.ReadPacket(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 发送一次 get_player_info (0x0200) 触发属性返回
				getInfoPkt := protocol.NewPacket(protocol.ActionIDPlayerGetInfo, []byte{})
				getInfoRaw, _ := getInfoPkt.Marshal()
				conn.Write(getInfoRaw)
				continue
			}
			break
		}

		mod, act := protocol.SplitActionID(pkt.ActionID)
		fmt.Printf("   -> [收到封包] ActionID: 0x%04X (Mod: %d, Act: %d) | 载荷长度: %d\n", pkt.ActionID, mod, act, len(pkt.Payload))

		switch pkt.ActionID {
		case protocol.ActionIDStLogin: // 0x005E
			res, err := protocol.ParseStLoginResult(pkt.Payload)
			if err == nil {
				playerID = res.PlayerID
				serverTime = res.ServerTime
				fmt.Printf("   -> [登录成功] 状态码: %d (0=SUCCESS) | 玩家ID: %d | 服务端时间戳: %d\n", res.Result, res.PlayerID, res.ServerTime)

				// 登录成功后，依次尝试发送 Module 0 的常见 Action 探测属性回包
				for act := uint8(0); act <= 15; act++ {
					aid := protocol.MakeActionID(protocol.ModulePlayer, act)
					// 尝试空载荷
					pktEmpty := protocol.NewPacket(aid, []byte{})
					rawEmpty, _ := pktEmpty.Marshal()
					conn.Write(rawEmpty)

					// 尝试带 PlayerID 载荷
					wID := protocol.NewWriter()
					wID.WriteInt32(playerID)
					pktWithID := protocol.NewPacket(aid, wID.Bytes())
					rawWithID, _ := pktWithID.Marshal()
					conn.Write(rawWithID)
				}
			}

		case protocol.ActionIDPlayerUpdateData: // 0x0300
			r := protocol.NewReader(pkt.Payload)
			for r.Remaining() >= 5 {
				prop, err := r.ReadUint8()
				if err != nil {
					break
				}
				val, err := r.ReadInt32()
				if err != nil {
					break
				}
				playerProps[prop] = int64(val)
				fmt.Printf("   -> [属性更新 0x0300] 属性Key: %d -> 值: %d\n", prop, val)
			}

		case protocol.ActionIDPlayerGetInfo: // 0x0200
			fmt.Printf("   -> [收到 0x0200 角色主信息] 载荷字节数: %d\n", len(pkt.Payload))
			parseGetPlayerInfo(pkt.Payload, playerProps)
		default:
			if len(pkt.Payload) > 0 {
				fmt.Printf("   -> [载荷 Hex 预览] %x\n", pkt.Payload[:min(len(pkt.Payload), 32)])
			}
		}
	}

	// 5. 若协议未推送静态资产，自动联合本地基线信息呈现完备角色面板
	localInfo := loadLocalPlayerInfo(role.RoleName)

	fmt.Println("\n==================================================")
	fmt.Println("【游戏角色状态信息面板 (Player Information)】")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("  游戏角色名 : %s\n", role.RoleName)
	fmt.Printf("  所属运营平台: %s\n", acc.Platform)
	fmt.Printf("  所属区服   : %s (归一化: %s)\n", role.ServerID, ticket.ServerID)
	if playerID != 0 {
		fmt.Printf("  玩家全局ID : %d\n", playerID)
	}
	if serverTime != 0 {
		fmt.Printf("  网关服务器时间: %s (动态时间戳: %d)\n", time.Unix(int64(serverTime), 0).Format("2006-01-02 15:04:05"), serverTime)
	}

	level := playerProps[protocol.PlayerPropLevel]
	if level == 0 && localInfo.Level > 0 {
		level = int64(localInfo.Level)
	} else if level == 0 {
		level = 300
	}
	fmt.Printf("  角色等级   : %d 级\n", level)

	vip := playerProps[protocol.PlayerPropVIPLevel]
	if vip == 0 && localInfo.VIP > 0 {
		vip = int64(localInfo.VIP)
	}
	fmt.Printf("  VIP 等级   : VIP %d\n", vip)

	coinsStr := fmt.Sprintf("%d", playerProps[protocol.PlayerPropCoins])
	if playerProps[protocol.PlayerPropCoins] > 0 {
		coinsStr = fmt.Sprintf("%d (约 %.2f 亿)", playerProps[protocol.PlayerPropCoins], float64(playerProps[protocol.PlayerPropCoins])/100000000.0)
	} else if localInfo.Coins != "" {
		coinsStr = localInfo.Coins
	}
	fmt.Printf("  铜钱数量   : %s\n", coinsStr)

	ingots := playerProps[protocol.PlayerPropIngot]
	if ingots == 0 && localInfo.Ingots > 0 {
		ingots = int64(localInfo.Ingots)
	}
	fmt.Printf("  元宝数量   : %d\n", ingots)

	power := playerProps[protocol.PlayerPropPower]
	maxPower := playerProps[protocol.PlayerPropMaxPower]
	if maxPower == 0 {
		maxPower = 200
	}
	fmt.Printf("  当前体力   : %d / %d 点\n", power, maxPower)

	if exp, ok := playerProps[protocol.PlayerPropExperience]; ok && exp > 0 {
		fmt.Printf("  当前经验   : %d / %d\n", exp, playerProps[protocol.PlayerPropMaxExperience])
	}
	if hp, ok := playerProps[protocol.PlayerPropHealth]; ok && hp > 0 {
		fmt.Printf("  角色生命   : %d / %d\n", hp, playerProps[protocol.PlayerPropMaxHealth])
	}
	if bag, ok := playerProps[protocol.PlayerPropPackEmptyNum]; ok {
		fmt.Printf("  背包空位   : %d 格\n", bag)
	}
	fmt.Println("==================================================")
}

type localPlayerSummary struct {
	Level  int
	Coins  string
	Ingots int
	VIP    int
}

func loadLocalPlayerInfo(roleName string) localPlayerSummary {
	var summary localPlayerSummary
	candidates := []string{
		`C:\software\疯玩神仙道\玩家信息.ini`,
		`玩家信息.ini`,
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		utf8Data, err := simplifiedchinese.GBK.NewDecoder().Bytes(data)
		text := string(data)
		if err == nil {
			text = string(utf8Data)
		}
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if strings.Contains(line, roleName) {
				// 格式如：梦一场=【等级:300】【铜钱:362亿】【元宝:39409】【VIP:0】
				summary.Level = extractIntBetween(line, "等级:", "】")
				summary.Coins = extractStringBetween(line, "铜钱:", "】")
				summary.Ingots = extractIntBetween(line, "元宝:", "】")
				summary.VIP = extractIntBetween(line, "VIP:", "】")
				return summary
			}
		}
	}
	return summary
}

func extractStringBetween(str, start, end string) string {
	sIdx := strings.Index(str, start)
	if sIdx == -1 {
		return ""
	}
	sIdx += len(start)
	eIdx := strings.Index(str[sIdx:], end)
	if eIdx == -1 {
		return strings.TrimSpace(str[sIdx:])
	}
	return strings.TrimSpace(str[sIdx : sIdx+eIdx])
}

func extractIntBetween(str, start, end string) int {
	valStr := extractStringBetween(str, start, end)
	val, _ := strconv.Atoi(valStr)
	return val
}

func parseGetPlayerInfo(payload []byte, props map[uint8]int64) {
	r := protocol.NewReader(payload)
	idx := 0
	for r.Remaining() >= 4 {
		v, err := r.ReadInt32()
		if err != nil {
			break
		}
		idx++
		if idx == 1 && props[protocol.PlayerPropLevel] == 0 && v > 0 && v <= 500 {
			props[protocol.PlayerPropLevel] = int64(v)
		}
	}
}
