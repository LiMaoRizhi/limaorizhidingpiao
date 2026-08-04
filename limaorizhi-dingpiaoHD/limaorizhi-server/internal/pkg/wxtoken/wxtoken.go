package wxtoken

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"limaorizhi-server/internal/pkg/redis"
)

// access_token 进程内存缓存（仅作Redis不可用时的降级方案）
var (
	cacheToken    string
	cacheTokenExp time.Time
	cacheMu       sync.Mutex
	httpClient    = &http.Client{Timeout: 10 * time.Second}
	cacheKey      = "wx:access_token"
)

// GetAccessToken 获取微信access_token（多实例通过Redis共享，提前5分钟刷新）
// handler/wx/user.go 和 service/wx_notify.go 共用此函数，避免重复实现
func GetAccessToken(appid, secret string) string {
	// 1. 优先从Redis共享缓存读取
	if redis.Enabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		token, err := redis.Client().Get(ctx, cacheKey).Result()
		cancel()
		if err == nil && token != "" {
			return token
		}
	}

	// 2. 进程内存缓存（Redis不可用时降级，同时防止多goroutine并发重复获取）
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cacheToken != "" && time.Now().Before(cacheTokenExp) {
		return cacheToken
	}

	// 3. 从微信API获取新token
	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		url.QueryEscape(appid), url.QueryEscape(secret),
	)
	resp, err := httpClient.Get(tokenURL)
	if err != nil {
		log.Printf("[WARN] wxtoken: 获取access_token网络请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[WARN] wxtoken: 解析access_token响应失败: %v\n", err)
		return ""
	}
	if result.AccessToken == "" {
		log.Printf("[WARN] wxtoken: access_token为空 errcode=%d errmsg=%s\n", result.ErrCode, result.ErrMsg)
		return ""
	}

	// 提前5分钟过期，避免使用临期token
	ttl := time.Duration(result.ExpiresIn-300) * time.Second

	// 4. 写入Redis共享缓存
	if redis.Enabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		redis.Client().Set(ctx, cacheKey, result.AccessToken, ttl)
		cancel()
	}

	// 5. 写入进程内存（Redis不可用时的降级缓存）
	cacheToken = result.AccessToken
	cacheTokenExp = time.Now().Add(ttl)
	return cacheToken
}
