// Package platform 提供多运营平台 Web 自动化认证、凭据换取与区服标识归一化驱动层。
//
// @author Ateng
// @since 2026-10-08
package platform

import (
	"context"
	"errors"
	"fmt"
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
		loginURL:   "http://member.fengwanyx.3fangyuan.com/login.php?goback=http://fengwanyx.3fangyuan.com",
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

	// 1. 发起第一阶段平台登录验证 (POST login.php)
	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("password", password)
	formData.Set("_fengwanyx_u_dl", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("fengwan: failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	loginResp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fengwan: login request failed: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode >= 400 {
		return nil, fmt.Errorf("fengwan: login returned http status %d", loginResp.StatusCode)
	}

	// 将第一阶段获取的凭据跨域注入至 member.fengwanyx.com
	if a.client.Jar != nil {
		if uLogin, err := url.Parse(a.loginURL); err == nil {
			cookies := a.client.Jar.Cookies(uLogin)
			if uMember, err := url.Parse("http://member.fengwanyx.com"); err == nil {
				a.client.Jar.SetCookies(uMember, cookies)
			}
		}
	}

	// 2. 发起第二阶段请求 game.php，捕获 302 重定向至 login_api.php
	serverSlug := extractServerSlug(serverID)
	enterQuery := url.Values{}
	enterQuery.Set("game", "sxd")
	enterQuery.Set("server", serverSlug)

	enterFullURL := a.enterURL
	if !strings.Contains(enterFullURL, "?") {
		enterFullURL = fmt.Sprintf("%s?%s", a.enterURL, enterQuery.Encode())
	}

	var redirectLocation string
	redirectClient := *a.client
	redirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		redirectLocation = req.URL.String()
		return http.ErrUseLastResponse
	}

	enterReq, err := http.NewRequestWithContext(ctx, http.MethodGet, enterFullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fengwan: failed to create enter game request: %w", err)
	}
	enterReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	enterReq.Header.Set("Referer", fmt.Sprintf("http://member.fengwanyx.com/entergame.php?game=sxd&server=%s", serverSlug))

	enterResp, err := redirectClient.Do(enterReq)
	if err != nil && !errors.Is(err, http.ErrUseLastResponse) {
		return nil, fmt.Errorf("fengwan: enter game request failed: %w", err)
	}
	defer enterResp.Body.Close()

	if redirectLocation == "" {
		redirectLocation = enterResp.Header.Get("Location")
	}

	targetGatewayURL := redirectLocation
	authCode := extractQueryParam(redirectLocation, "code")

	// 3. 发起第三阶段：若重定向目标包含 login_api.php，访问该端点提取最终的双轨 Cookie 与会话票据
	var cookies []*http.Cookie
	cookies = append(cookies, enterResp.Cookies()...)

	if redirectLocation != "" && strings.Contains(redirectLocation, "login_api.php") {
		apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectLocation, nil)
		if err == nil {
			apiReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			apiResp, err := redirectClient.Do(apiReq)
			if err == nil {
				defer apiResp.Body.Close()
				cookies = append(cookies, apiResp.Cookies()...)
				if apiLoc := apiResp.Header.Get("Location"); apiLoc != "" && authCode == "" {
					authCode = extractQueryParam(apiLoc, "code")
				}
			}
		}
	}

	// 4. 从响应头与 CookieJar 中解析提取时间戳与哈希散列
	nowUnix := int32(time.Now().Unix())
	mainTicket := MainServerTicket{
		Code: authCode,
		Time: nowUnix,
	}
	crossTicket := CrossServerTicket{
		Time1: nowUnix,
	}

	// 检查重定向 URL Query 中是否带有时效凭证
	if qTime := extractQueryParam(redirectLocation, "time"); qTime != "" {
		if tVal, err := strconv.ParseInt(qTime, 10, 32); err == nil {
			mainTicket.Time = int32(tVal)
			crossTicket.Time1 = int32(tVal)
		}
	}
	if qHash := extractQueryParam(redirectLocation, "hash"); qHash != "" {
		mainTicket.Hash = qHash
		crossTicket.Hash1 = qHash
	}

	if a.client.Jar != nil {
		if u, err := url.Parse(a.enterURL); err == nil {
			cookies = append(cookies, a.client.Jar.Cookies(u)...)
		}
		if uRedir, err := url.Parse(redirectLocation); err == nil {
			cookies = append(cookies, a.client.Jar.Cookies(uRedir)...)
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
			if parsedTime, err := strconv.ParseInt(val, 10, 32); err == nil {
				mainTicket.Time = int32(parsedTime)
				if crossTicket.Time1 == nowUnix {
					crossTicket.Time1 = int32(parsedTime)
				}
			}
		case name == "_hash" || name == "hash":
			mainTicket.Hash = val
			if crossTicket.Hash1 == "" {
				crossTicket.Hash1 = val
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

var codeParamRegex = regexp.MustCompile(`[?&]code=([^&#]+)`)

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
