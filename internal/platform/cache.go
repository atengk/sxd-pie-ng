// Package platform 提供多运营平台 Web 自动化认证、凭据换取与区服标识归一化驱动层。
//
// @author Ateng
// @since 2026-10-08
package platform

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TicketCache 提供平台登录凭据的内存缓存、时效检测与静默刷新管理。
type TicketCache struct {
	mu             sync.RWMutex
	tickets        map[string]*Ticket
	authenticators map[string]PlatformAuthenticator
	defaultTTL     time.Duration
}

// NewTicketCache 创建并初始化票据缓存容器。
func NewTicketCache(ttl time.Duration) *TicketCache {
	if ttl <= 0 {
		ttl = 90 * time.Minute // 默认 90 分钟主动静默续约
	}
	return &TicketCache{
		tickets:        make(map[string]*Ticket),
		authenticators: make(map[string]PlatformAuthenticator),
		defaultTTL:     ttl,
	}
}

// RegisterAuthenticator 注册指定运营平台的认证驱动。
func (c *TicketCache) RegisterAuthenticator(auth PlatformAuthenticator) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authenticators[stringsLower(auth.Platform())] = auth
}

// GetOrFetch 获取有效的双轨凭据，若缓存未命中或已过期，将自动调用对应平台驱动进行换票。
func (c *TicketCache) GetOrFetch(ctx context.Context, platformName, username, password, serverID string) (*Ticket, error) {
	cacheKey := makeCacheKey(platformName, username, serverID)

	// 1. 先进行读锁快速判断
	c.mu.RLock()
	cached, ok := c.tickets[cacheKey]
	if ok && !cached.IsExpired() {
		c.mu.RUnlock()
		return cached, nil
	}
	auth, hasAuth := c.authenticators[stringsLower(platformName)]
	c.mu.RUnlock()

	if !hasAuth {
		return nil, fmt.Errorf("platform: authenticator for '%s' not registered", platformName)
	}

	// 2. 升为写锁进行换票防击穿处理
	c.mu.Lock()
	defer c.mu.Unlock()

	// 二次检查缓存
	if cached, ok := c.tickets[cacheKey]; ok && !cached.IsExpired() {
		return cached, nil
	}

	// 3. 执行平台认证换票
	ticket, err := auth.Login(ctx, username, password, serverID)
	if err != nil {
		return nil, fmt.Errorf("platform: login failed for user %s on %s: %w", username, serverID, err)
	}

	// 若未指定过期时间，填充默认 TTL
	if ticket.ExpiresAt.IsZero() {
		ticket.CreatedAt = time.Now()
		ticket.ExpiresAt = ticket.CreatedAt.Add(c.defaultTTL)
	}

	c.tickets[cacheKey] = ticket
	return ticket, nil
}

// Invalidate 使特定账号与区服的票据失效 (通常在长连接收到鉴权失败状态时触发被动重登)。
func (c *TicketCache) Invalidate(platformName, username, serverID string) {
	cacheKey := makeCacheKey(platformName, username, serverID)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tickets, cacheKey)
}

func makeCacheKey(platformName, username, serverID string) string {
	return fmt.Sprintf("%s:%s:%s", stringsLower(platformName), username, stringsLower(serverID))
}

func stringsLower(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b = append(b, c)
	}
	return string(b)
}
