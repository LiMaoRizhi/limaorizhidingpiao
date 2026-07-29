// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"limaorizhi-server/internal/config"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client
var enabled bool

// rateLimitScript 令牌桶限流 Lua 脚本（原子操作）
// KEYS[1] = ratelimit:{ip}
// ARGV[1] = rate（令牌补充速率，每秒）
// ARGV[2] = burst（桶容量/突发上限）
// ARGV[3] = now（当前Unix时间戳，秒）
// ARGV[4] = cost（本次请求消耗的令牌数，通常为1）
// 返回: 1=允许, 0=拒绝
var rateLimitScript = redis.NewScript(`
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])

local tokens = tonumber(redis.call('HGET', KEYS[1], 'tokens')) or burst
local last_time = tonumber(redis.call('HGET', KEYS[1], 'last_time')) or now

local elapsed = math.max(0, now - last_time)
local new_tokens = math.min(burst, tokens + elapsed * rate)

if new_tokens < cost then
    redis.call('HMSET', KEYS[1], 'tokens', new_tokens, 'last_time', now)
    redis.call('EXPIRE', KEYS[1], 300)
    return 0
else
    new_tokens = new_tokens - cost
    redis.call('HMSET', KEYS[1], 'tokens', new_tokens, 'last_time', now)
    redis.call('EXPIRE', KEYS[1], 300)
    return 1
end
`)

// Init 初始化 Redis 连接
// 未配置或连接失败时不阻断启动，限流和登录失败计数将降级为进程内存模式
func Init(cfg config.RedisConfig) error {
	if cfg.Host == "" {
		log.Println("[INFO] Redis 未配置，限流和登录失败计数将使用进程内存（多实例部署时各实例独立计数）")
		return nil
	}

	client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MinIdleConns: 3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[WARN] Redis 连接失败: %v，限流和登录失败计数将降级为进程内存模式", err)
		client = nil
		return nil // 不阻断启动
	}

	enabled = true
	log.Printf("Redis 连接成功: %s:%d db=%d", cfg.Host, cfg.Port, cfg.DB)
	return nil
}

// Client 获取 Redis 客户端实例
func Client() *redis.Client {
	return client
}

// Enabled 检查 Redis 是否可用
func Enabled() bool {
	return enabled && client != nil
}

// 分布式限流

// AllowRequest 令牌桶限流检查
// 返回: allowed=是否允许请求, fallback=是否需要降级为内存限流
func AllowRequest(ip string, rate, burst float64) (allowed bool, fallback bool) {
	if !Enabled() {
		return true, true // Redis 不可用，降级
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "ratelimit:" + ip
	result, err := rateLimitScript.Run(ctx, client,
		[]string{key},
		rate, burst, float64(time.Now().Unix()), 1.0,
	).Int()

	if err != nil {
		// Redis 调用失败，降级为内存限流（fail-open）
		return true, true
	}

	return result == 1, false
}

// 分布式登录失败计数

// IsLoginLocked 检查登录是否被锁定
// 返回: locked=是否锁定, fallback=是否需要降级为内存检查
func IsLoginLocked(ip, username string) (locked bool, fallback bool) {
	if !Enabled() {
		return false, true // Redis 不可用，降级
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "loginfail:" + ip + ":" + username
	val, err := client.Get(ctx, key).Int()
	if err != nil {
		// key 不存在(redis.Nil)或 Redis 调用失败，降级为内存检查
		return false, true
	}
	return val >= 5, false
}

// RecordLoginFail 记录登录失败计数
// lockDuration: 锁定时长（从首次失败开始计时）
// 返回: fallback=是否需要降级为内存计数
func RecordLoginFail(ip, username string, lockDuration time.Duration) (fallback bool) {
	if !Enabled() {
		return true // Redis 不可用，降级
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "loginfail:" + ip + ":" + username
	val, err := client.Incr(ctx, key).Result()
	if err != nil {
		return true // Redis 调用失败，降级
	}
	// 首次失败时设置过期时间（锁定窗口期）
	if val == 1 {
		client.Expire(ctx, key, lockDuration)
	}
	return false
}

// ClearLoginFail 清除登录失败计数（登录成功时调用）
// 返回: fallback=是否需要降级为内存清除
func ClearLoginFail(ip, username string) (fallback bool) {
	if !Enabled() {
		return true // Redis 不可用，降级
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "loginfail:" + ip + ":" + username
	client.Del(ctx, key)
	return false
}

// 登录失败计数（含内存降级封装，供多包复用）

// memLoginAttempts 进程内存登录失败计数（Redis 不可用时降级）
var memLoginAttempts sync.Map // key: "ip:username"
var memLoginMutex sync.Mutex  // 保护进程内存登录计数的并发修改

type memLoginAttempt struct {
	failCount int
	lockUntil time.Time
}

// IsLoginLockedWithFallback 检查登录是否被锁定（Redis不可用时降级为进程内存检查）
func IsLoginLockedWithFallback(ip, username string) bool {
	locked, fallback := IsLoginLocked(ip, username)
	if !fallback {
		return locked
	}
	// 降级：进程内存检查
	key := ip + ":" + username
	val, ok := memLoginAttempts.Load(key)
	if !ok {
		return false
	}
	attempt := val.(*memLoginAttempt)
	return time.Now().Before(attempt.lockUntil)
}

// RecordLoginFailWithFallback 记录登录失败（Redis不可用时降级为进程内存计数）
// 5次失败后锁定 lockDuration 时长
func RecordLoginFailWithFallback(ip, username string, lockDuration time.Duration) {
	if fallback := RecordLoginFail(ip, username, lockDuration); !fallback {
		return
	}
	// 降级：进程内存计数（加互斥锁避免并发丢失计数）
	key := ip + ":" + username
	memLoginMutex.Lock()
	defer memLoginMutex.Unlock()
	var attempt *memLoginAttempt
	val, ok := memLoginAttempts.Load(key)
	if ok {
		attempt = val.(*memLoginAttempt)
	} else {
		attempt = &memLoginAttempt{}
	}
	attempt.failCount++
	if attempt.failCount >= 5 {
		attempt.lockUntil = time.Now().Add(lockDuration)
		attempt.failCount = 0
	}
	memLoginAttempts.Store(key, attempt)
}

// ClearLoginFailWithFallback 清除登录失败计数（Redis不可用时降级为进程内存清除）
func ClearLoginFailWithFallback(ip, username string) {
	if fallback := ClearLoginFail(ip, username); !fallback {
		return
	}
	// 降级：进程内存清除
	key := ip + ":" + username
	memLoginAttempts.Delete(key)
}

// 身份实名认证结果缓存

// IDCardVerifyCache 实名认证缓存结果
// 业务侧只消费 Matched 字段（Province/City/Sex 等附加字段在调用方未被使用，缓存不存储）
type IDCardVerifyCache struct {
	Matched    bool      `json:"matched"`
	VerifiedAt time.Time `json:"verified_at"`
}

// idCardVerifyTTL 认证结果缓存有效期（30天）
// 同一姓名+身份证号组合在 30 天内复用认证结果，避免重复调用云市场 API（节省 0.3 元/次）
const idCardVerifyTTL = 30 * 24 * time.Hour

// IDCardVerifyCacheKey 生成认证缓存 key
// key 同时包含姓名和身份证号的 sha256，防止同一身份证号被不同姓名冒用时错误命中"匹配"缓存
// 仅用 sha256 摘要作为 key，避免身份证号明文泄露到 Redis 中
func IDCardVerifyCacheKey(name, idCardNo string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name)) + "|" + idCardNo))
	return "idverify:" + hex.EncodeToString(h[:])
}

// GetIDCardVerify 读取实名认证缓存
// 返回: cache=缓存值（nil 表示未命中）, fallback=是否需要降级（直接调用云市场 API）
// fallback=false 且 cache=nil：表示 Redis 正常工作但未命中缓存，调用方应继续走 API 流程
// fallback=true：表示 Redis 不可用或调用失败，调用方应降级为直接调用 API（不阻断业务）
func GetIDCardVerify(name, idCardNo string) (cache *IDCardVerifyCache, fallback bool) {
	if !Enabled() {
		return nil, true // Redis 不可用，降级为实时调用 API
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := IDCardVerifyCacheKey(name, idCardNo)
	val, err := client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false // 未命中缓存，走正常 API 流程（非降级）
		}
		return nil, true // Redis 调用失败，降级
	}

	var c IDCardVerifyCache
	if err := json.Unmarshal(val, &c); err != nil {
		return nil, true // 反序列化失败，降级
	}
	return &c, false
}

// SetIDCardVerify 写入实名认证缓存
// 仅在云市场 API 明确返回 matched=true/false 时才缓存，API 调用失败（网络错误等）不缓存
// ttl: 缓存有效期，<=0 时使用默认值 idCardVerifyTTL（30 天）。允许调用方从配置注入 TTL，实现不重启调参。
// 返回: fallback=是否需要降级（写入失败不影响业务，下次仍走 API）
func SetIDCardVerify(name, idCardNo string, matched bool, ttl time.Duration) (fallback bool) {
	if !Enabled() {
		return true // Redis 不可用，不缓存
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// ttl<=0 时降级使用默认 30 天，防止配置遗漏导致缓存永不过期或过期过快
	if ttl <= 0 {
		ttl = idCardVerifyTTL
	}

	key := IDCardVerifyCacheKey(name, idCardNo)
	c := IDCardVerifyCache{
		Matched:    matched,
		VerifiedAt: time.Now(),
	}
	data, err := json.Marshal(c)
	if err != nil {
		return true
	}
	if err := client.Set(ctx, key, data, ttl).Err(); err != nil {
		return true
	}
	return false
}

// DeleteIDCardVerify 删除实名认证缓存，强制下次走云市场 API
// 返回: fallback=是否需要降级（Redis 不可用时返回 true，调用方可以认为缓存已被清除——因为下次本就会走 API）
func DeleteIDCardVerify(name, idCardNo string) (fallback bool) {
	if !Enabled() {
		return true // Redis 不可用，无需删除（原本也不会命中缓存）
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := IDCardVerifyCacheKey(name, idCardNo)
	if err := client.Del(ctx, key).Err(); err != nil {
		return true
	}
	return false
}

// 分布式锁

// TryLock 尝试获取分布式锁（SET NX EX）
// 返回: acquired=是否获取成功, fallback=是否需要降级为进程级锁
// key: 锁的 key（建议加 lock: 前缀）
// ttl: 锁的自动过期时间（防止持锁进程崩溃后死锁）
func TryLock(key string, ttl time.Duration) (acquired bool, fallback bool) {
	if !Enabled() {
		return false, true // Redis 不可用，降级
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ok, err := client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, true // Redis 调用失败，降级
	}
	return ok, false
}

// Unlock 释放分布式锁
// 仅在 TryLock 成功且业务完成后调用
// fallback=true 时为降级模式（Redis 不可用），调用方可忽略
func Unlock(key string) (fallback bool) {
	if !Enabled() {
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client.Del(ctx, key)
	return false
}

// IncrWithTTL 自增计数器（带TTL，首次自增时设置过期时间）
// 返回: count=当前计数值, fallback=是否需要降级
// 用于频率限制场景：如每小时最多N次操作
func IncrWithTTL(key string, ttl time.Duration) (count int64, fallback bool) {
	if !Enabled() {
		return 0, true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, true
	}
	// 首次自增时设置过期时间
	if val == 1 {
		client.Expire(ctx, key, ttl)
	}
	return val, false
}
