// Package platform 提供多运营平台 Web 自动化认证、凭据换取与区服标识归一化驱动层。
//
// @author Ateng
// @since 2026-10-08
package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FengwanAuthenticator 提供疯玩游戏平台的原生 HTTP 登录与双轨凭据提取驱动。
type FengwanAuthenticator struct {
	client     *http.Client
	loginURL   string
	enterURL   string
	defaultTTL time.Duration
}

// FengwanOption 配置 FengwanAuthenticator 的可选参数函数。
type FengwanOption func(*FengwanAuthenticator)

// WithHTTPClient 允许外部注入自定义 HTTP 客户端 (例如单元测试 Mock Transport)。
func WithHTTPClient(client *http.Client) FengwanOption {
	return func(a *FengwanAuthenticator) {
		a.client = client
	}
}

// WithBaseEndpoints 允许覆盖默认 Web 端点 (用于本地沙箱或 Mock 联调)。
func WithBaseEndpoints(loginURL, enterURL string) FengwanOption {
	return func(a *FengwanAuthenticator) {
		if loginURL != "" {
			a.loginURL = loginURL
		}
		if enterURL != "" {
			a.enterURL = enterURL
		}
	}
}

// NewFengwanAuthenticator 创建并初始化疯玩平台认证驱动。
func NewFengwanAuthenticator(opts ...FengwanOption) (*FengwanAuthenticator, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("fengwan: failed to create cookie jar: %w", err)
	}

	auth := &FengwanAuthenticator{
		client: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
		loginURL:   "http://member.fengwanyx.com/login.php?action=checkuser",
		enterURL:   "http://member.fengwanyx.com/entergame.php",
		defaultTTL: 90 * time.Minute,
	}

	for _, opt := range opts {
		opt(auth)
	}

	return auth, nil
}

// Platform 返回平台唯一标识 "fengwan"。
func (a *FengwanAuthenticator) Platform() string {
	return "fengwan"
}

// Login 执行两阶段平台认证并换取目标区服的双轨凭据。
func (a *FengwanAuthenticator) Login(ctx context.Context, username, password, serverID string) (*Ticket, error) {
	if username == "" || password == "" {
		return nil, errors.New("fengwan: username and password cannot be empty")
	}

	// 1. 发起第一阶段平台登录验证 (支持线上 GET checkuser 与测试环境 POST form)
	var (
		req *http.Request
		err error
	)
	if strings.Contains(a.loginURL, "action=checkuser") {
		checkURL := fmt.Sprintf("%s&username=%s&password=%s", a.loginURL, url.QueryEscape(username), url.QueryEscape(password))
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	} else {
		form := url.Values{}
		form.Set("username", username)
		form.Set("password", password)
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, a.loginURL, strings.NewReader(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("fengwan: failed to create login request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	loginResp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fengwan: login request failed: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode >= 400 {
		return nil, fmt.Errorf("fengwan: login returned http status %d", loginResp.StatusCode)
	}

	// 2. 发起第二阶段请求 entergame.php，解析核心 iframe 或重定向 Location
	serverSlug := extractServerSlug(serverID)
	enterQuery := url.Values{}
	enterQuery.Set("game", "sxd")
	enterQuery.Set("server", serverSlug)

	enterFullURL := a.enterURL
	if !strings.Contains(enterFullURL, "?") {
		enterFullURL = fmt.Sprintf("%s?%s", a.enterURL, enterQuery.Encode())
	}

	enterReq, err := http.NewRequestWithContext(ctx, http.MethodGet, enterFullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fengwan: failed to create enter game request: %w", err)
	}
	enterReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	enterReq.Header.Set("Referer", fmt.Sprintf("http://member.fengwanyx.com/entergame.php?game=sxd&server=%s", serverSlug))

	var redirectLocation string
	redirectClient := *a.client
	redirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		redirectLocation = req.URL.String()
		return http.ErrUseLastResponse
	}

	enterResp, err := redirectClient.Do(enterReq)
	if err != nil && !errors.Is(err, http.ErrUseLastResponse) {
		return nil, fmt.Errorf("fengwan: enter game request failed: %w", err)
	}
	var enterBodyStr string
	if enterResp != nil {
		if redirectLocation == "" {
			redirectLocation = enterResp.Header.Get("Location")
		}
		enterBodyBytes, _ := io.ReadAll(enterResp.Body)
		_ = enterResp.Body.Close()
		enterBodyStr = string(enterBodyBytes)
	}

	targetGatewayURL := ""
	authCode := ""
	if redirectLocation != "" {
		authCode = extractQueryParam(redirectLocation, "code")
	}

	var cookies []*http.Cookie
	if enterResp != nil {
		cookies = append(cookies, enterResp.Cookies()...)
	}

	nowUnix := int32(time.Now().Unix())
	mainTicket := MainServerTicket{
		Code: authCode,
		Time: nowUnix,
	}
	crossTicket := CrossServerTicket{
		Time1: nowUnix,
	}

	var roleName string
	iframeTarget := ""
	if mIframe := iframeRegex.FindStringSubmatch(enterBodyStr); len(mIframe) >= 2 {
		iframeTarget = mIframe[1]
	}

	if iframeTarget != "" {
		if !strings.HasPrefix(iframeTarget, "http://") && !strings.HasPrefix(iframeTarget, "https://") {
			iframeTarget = fmt.Sprintf("http://member.fengwanyx.com/%s", strings.TrimPrefix(iframeTarget, "/"))
		}
		if ifReq, err := http.NewRequestWithContext(ctx, http.MethodGet, iframeTarget, nil); err == nil {
			ifReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			if ifResp, err := a.client.Do(ifReq); err == nil {
				defer ifResp.Body.Close()
				ifBytes, _ := io.ReadAll(ifResp.Body)
				ip, port, h, t, p := parseFlashVars(string(ifBytes))
				if ip != "" && port != "" {
					targetGatewayURL = fmt.Sprintf("%s:%s", ip, port)
				}
				if h != "" {
					mainTicket.Hash = h
				}
				if t != "" {
					if tv, err := strconv.ParseInt(t, 10, 32); err == nil {
						mainTicket.Time = int32(tv)
					}
				}
				if p != "" {
					roleName = p
					if mainTicket.Code == "" {
						mainTicket.Code = p
					}
				}
			}
		}
	}

	if a.client.Jar != nil {
		if u, err := url.Parse(a.enterURL); err == nil {
			cookies = append(cookies, a.client.Jar.Cookies(u)...)
		}
		for _, c := range a.extractAllCookies() {
			cookies = append(cookies, c)
		}
	}

	for _, c := range cookies {
		val := strings.TrimSpace(c.Value)
		name := strings.ToLower(c.Name)
		switch {
		case strings.HasPrefix(name, "login_time_sxd"):
			if parsedTime, err := strconv.ParseInt(val, 10, 32); err == nil {
				crossTicket.Time1 = int32(parsedTime)
			}
		case strings.HasPrefix(name, "login_hash_sxd"):
			crossTicket.Hash1 = val
		case name == "_time" || name == "time":
			if mainTicket.Time == nowUnix {
				if parsedTime, err := strconv.ParseInt(val, 10, 32); err == nil {
					mainTicket.Time = int32(parsedTime)
				}
			}
		case name == "_hash" || name == "hash":
			if mainTicket.Hash == "" {
				mainTicket.Hash = val
			}
		case name == "user" || name == "sxd_user":
			if mainTicket.Code == "" {
				mainTicket.Code = val
			}
		}
	}

	normalizedServerID := NormalizeServerID("fengwan", serverID)
	ticket := &Ticket{
		Platform:    "fengwan",
		ServerID:    normalizedServerID,
		RawServerID: serverID,
		RoleName:    roleName,
		GatewayURL:  targetGatewayURL,
		MainServer:  mainTicket,
		CrossServer: crossTicket,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(a.defaultTTL),
	}

	return ticket, nil
}

func (a *FengwanAuthenticator) extractAllCookies() []*http.Cookie {
	var res []*http.Cookie
	domains := []string{
		"http://member.fengwanyx.3fangyuan.com",
		"http://member.fengwanyx.com",
		"http://sxd.fengwanyx.com",
		"http://fengwanyx.3fangyuan.com",
	}
	for _, d := range domains {
		u, err := url.Parse(d)
		if err == nil && a.client.Jar != nil {
			res = append(res, a.client.Jar.Cookies(u)...)
		}
	}
	return res
}

func extractServerSlug(serverID string) string {
	s := strings.TrimSpace(serverID)
	// 去除 fengwanyx_ 前缀
	s = strings.TrimPrefix(s, "fengwanyx_")
	// 确保是形如 s813
	if !strings.HasPrefix(s, "s") && !strings.HasPrefix(s, "S") {
		return "s" + s
	}
	return strings.ToLower(s)
}

var (
	codeParamRegex  = regexp.MustCompile(`[?&]code=([^&#]+)`)
	iframeRegex     = regexp.MustCompile(`(?i)<iframe[^>]*\ssrc=["']([^"']+)["']`)
	hashCodeRegex   = regexp.MustCompile(`[&?]hash_code=([a-f0-9]{32})`)
	timeRegex       = regexp.MustCompile(`[&?]time=([0-9]+)`)
	ipRegex         = regexp.MustCompile(`[&?]ip=([^&"'\s]+)`)
	portRegex       = regexp.MustCompile(`[&?]port=([0-9]+)`)
	playerNameRegex = regexp.MustCompile(`[&?]player_name=([^&"'\s]+)`)
)

func parseFlashVars(html string) (ip, port, hashCode, timeVal, playerName string) {
	if m := hashCodeRegex.FindStringSubmatch(html); len(m) >= 2 {
		hashCode = m[1]
	}
	if m := timeRegex.FindStringSubmatch(html); len(m) >= 2 {
		timeVal = m[1]
	}
	if m := ipRegex.FindStringSubmatch(html); len(m) >= 2 {
		ip = m[1]
	}
	if m := portRegex.FindStringSubmatch(html); len(m) >= 2 {
		port = m[1]
	}
	if m := playerNameRegex.FindStringSubmatch(html); len(m) >= 2 {
		playerName = m[1]
	}
	return
}

func extractQueryParam(rawURL, param string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err == nil {
		q := u.Query().Get(param)
		if q != "" {
			return q
		}
	}
	matches := codeParamRegex.FindStringSubmatch(rawURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
