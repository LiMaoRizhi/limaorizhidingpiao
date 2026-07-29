// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/pkg/jwt"
	"limaorizhi-server/internal/pkg/response"
	redis "limaorizhi-server/internal/pkg/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Token登出失效（持久化到数据库）

// checkTokenRevoked 校验Token是否在用户最近登出时间之前签发（已被踢出）
// 各用户表均含 token_invalid_before 字段，登出时写入当前时间，重启不丢失、多实例共享。
// 查询失败时放行（fail-open），避免数据库抖动导致全量用户被锁定。
// 同时返回用户状态是否为启用（status=1），被封禁用户的Token即使未过期也拒绝访问。
func checkTokenRevoked(db *gorm.DB, tableName string, userID uint, issuedAt time.Time) bool {
	var result struct {
		TokenInvalidBefore *time.Time
		Status             int8
	}
	if err := db.Table(tableName).Select("token_invalid_before, status").Where("id = ?", userID).Take(&result).Error; err != nil {
		log.Printf("[WARN] Token吊销检查失败 table:%s userID:%d err:%v\n", tableName, userID, err)
		return config.AppConfig.Server.Mode == "release"
	}
	// Token在登出时间之前签发 → 已被踢出
	if result.TokenInvalidBefore != nil && issuedAt.Before(*result.TokenInvalidBefore) {
		return true
	}
	// 用户状态不为启用(1) → 封禁
	if result.Status != 1 {
		return true
	}
	return false
}

// JWTAuth JWT鉴权中间件
func JWTAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := jwt.ParseToken(config.AppConfig.JWT.AdminSecret, token)
		if err != nil {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}
		if claims.Type != "admin" {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}

		// 校验Token是否在登出时间之前签发（已被踢出）
		if claims.IssuedAt != nil && checkTokenRevoked(db, "admin_users", claims.UserID, claims.IssuedAt.Time) {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}

		c.Set("admin_id", claims.UserID)
		c.Set("admin_name", claims.Username)
		c.Set("admin_role", claims.Role)
		c.Next()
	}
}

// RequireSuperAdmin 超级管理员权限校验中间件（需在JWTAuth之后使用）
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("admin_role")
		if !exists {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}
		role, ok := roleVal.(int8)
		if !ok || role != 1 {
			response.FailMsg(c, response.CodeForbidden, "需要超级管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

// DriverAuth 司机端JWT鉴权中间件
func DriverAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// 未配置 driver_secret 时直接拒绝，防止密钥混用越权
		driverSecret := config.AppConfig.JWT.DriverSecret
		if driverSecret == "" {
			response.FailMsg(c, response.CodeUnauthorized, "司机端JWT密钥未配置(driver_secret)，请联系管理员")
			c.Abort()
			return
		}
		claims, err := jwt.ParseToken(driverSecret, token)
		if err != nil {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}
		if claims.Type != "driver" {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}

		// 校验Token是否在登出时间之前签发（已被踢出）
		if claims.IssuedAt != nil && checkTokenRevoked(db, "drivers", claims.UserID, claims.IssuedAt.Time) {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}

		c.Set("driver_id", claims.UserID)
		c.Set("driver_name", claims.Username)
		c.Next()
	}
}

// WxUserAuth 小程序用户端JWT鉴权中间件
func WxUserAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := jwt.ParseToken(config.AppConfig.JWT.WXSecret, token)
		if err != nil {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}
		if claims.Type != "wx" {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}

		// 校验Token是否在登出时间之前签发（已被踢出）
		if claims.IssuedAt != nil && checkTokenRevoked(db, "users", claims.UserID, claims.IssuedAt.Time) {
			response.Fail(c, response.CodeUnauthorized)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_name", claims.Username)
		c.Next()
	}
}

// defaultAllowedOrigins 默认CORS白名单（开发环境）
var defaultAllowedOrigins = map[string]bool{
	"http://localhost:3000": true,
	"http://localhost:8080": true,
	"http://127.0.0.1:3000": true,
	"http://127.0.0.1:8080": true,
}

// isOriginAllowed 检查Origin是否在白名单中（支持配置文件动态添加）
func isOriginAllowed(origin string) bool {
	// 生产环境不使用默认白名单（不含localhost），仅使用配置的CORS白名单
	if config.AppConfig.Server.Mode != "release" {
		if defaultAllowedOrigins[origin] {
			return true
		}
	}
	// 从配置文件读取额外的CORS白名单（逗号分隔）
	if config.AppConfig.CORSOrigins != "" {
		for _, o := range strings.Split(config.AppConfig.CORSOrigins, ",") {
			if strings.TrimSpace(o) == origin {
				return true
			}
		}
	}
	return false
}

// CORS 跨域中间件（白名单模式）
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && isOriginAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			if origin != "" && isOriginAllowed(origin) {
				c.AbortWithStatus(204)
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

// RateLimit 请求频率限制中间件（基于IP的令牌桶限流）
// 每秒最多5个请求，突发10个
// 优先使用 Redis 分布式限流（多实例共享计数），Redis 不可用时降级为进程内存限流
type ipLimiter struct {
	tokens   float64
	lastTime time.Time
	mu       sync.Mutex
}

var limiters sync.Map
var limitersCleanupOnce sync.Once

func RateLimit() gin.HandlerFunc {
	const (
		rate       = 5.0  // 每秒补充5个令牌
		burst      = 10.0 // 桶容量
		cleanupMs = 60000 // 60秒清理一次过期记录
	)
	// 使用 sync.Once 确保清理协程只启动一次，避免每次调用 RateLimit() 都创建新协程
	limitersCleanupOnce.Do(func() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[PANIC] 限流器清理协程panic: %v\n", r)
				}
			}()
			for {
				time.Sleep(time.Duration(cleanupMs) * time.Millisecond)
				now := time.Now()
				limiters.Range(func(key, value interface{}) bool {
					l := value.(*ipLimiter)
					if now.Sub(l.lastTime) > 5*time.Minute {
						limiters.Delete(key)
					}
					return true
				})
			}
		}()
	})

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// 优先使用 Redis 分布式限流（多实例共享计数）
		allowed, fallback := redis.AllowRequest(ip, rate, burst)
		if !fallback {
			if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"code":    429,
					"message": "请求过于频繁，请稍后再试",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Redis 不可用时降级为进程内存限流
		degradedBurst := burst
		if config.AppConfig.Server.Mode == "release" {
			degradedBurst = burst / 2
			if degradedBurst < 2 {
				degradedBurst = 2
			}
			log.Printf("[WARN] Redis 不可用，限流降级为进程内存模式（burst降为%.0f），多实例部署时限流可能不精确\n", degradedBurst)
		}
		val, _ := limiters.LoadOrStore(ip, &ipLimiter{
			tokens:   degradedBurst,
			lastTime: time.Now(),
		})
		l := val.(*ipLimiter)
		l.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(l.lastTime).Seconds()
		l.tokens += elapsed * rate
		if l.tokens > degradedBurst {
			l.tokens = degradedBurst
		}
		l.lastTime = now
		if l.tokens < 1 {
			l.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		l.tokens--
		l.mu.Unlock()
		c.Next()
	}
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		fmt.Fprintf(gin.DefaultWriter, "[%s] %s %s %d %v\n",
			start.Format("2006-01-02 15:04:05"),
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			latency,
		)
	}
}

// SecurityHeaders HTTP安全响应头中间件
// 防止点击劫持、MIME嗅探攻击、XSS等常见Web安全风险
// HSTS仅在release模式下启用（开发环境需HTTP访问，强制HTTPS会导致无法访问）
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// 生产环境启用HSTS，强制浏览器后续请求走HTTPS
		if config.AppConfig.Server.Mode == "release" {
			c.Header("Strict-Transport-Security", "max-age=86400; includeSubDomains")
		}
		c.Next()
	}
}
