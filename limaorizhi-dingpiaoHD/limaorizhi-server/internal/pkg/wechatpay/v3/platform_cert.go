// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
// 微信支付 v3 平台证书自动加载（Redis + 内存双级缓存）
package v3

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"sync"
	"time"

	"limaorizhi-server/internal/pkg/redis"
)

// PlatformCertManager 微信平台证书管理器
// 平台证书用于验证微信 v3 接口响应的签名（防中间人攻击）
// 证书有效期约 1 年，微信会提前轮换，故不能写死必须定期从 API 下载
//
// 2024 新方案：微信支付公钥模式
//   商户可以在商户后台申请一对「微信支付公钥 + 公钥ID」（长期有效，不会轮换）
//   启用后跳过 /v3/certificates 下载逻辑，直接用预置的公钥验签响应和回调
//   请求头 Wechatpay-Serial 使用公钥ID（而非平台证书序列号）
type PlatformCertManager struct {
	cfg Config

	// 反向引用 V3Client，复用其 doRequest 发起带签名的请求
	// 同包内直接引用，不构成跨包循环依赖
	client *V3Client

	mu         sync.RWMutex
	localCache map[string]*rsa.PublicKey // serial -> public key
	localTTL   time.Duration            // 进程内存缓存 TTL
	cacheTime  time.Time                // 上次全量下载时间

	redisTTL time.Duration // Redis 缓存 TTL

	// 微信支付公钥模式（可选）：以下两个字段同时非空时启用
	// 启用后 GetPublicKey/CurrentSerial 直接返回此公钥/ID，跳过下载逻辑
	wxPayPublicKey   *rsa.PublicKey
	wxPayPublicKeyID string
}

// certListResponse /v3/certificates 接口响应
type certListResponse struct {
	Data []struct {
		SerialNo       string `json:"serial_no"`
		EffectiveTime  string `json:"effective_time"`
		ExpireTime     string `json:"expire_time"`
		EncryptCertificate struct {
			Algorithm      string `json:"algorithm"`      // AEAD_AES_256_GCM
			Nonce          string `json:"nonce"`
			AssociatedData string `json:"associated_data"`
			Ciphertext     string `json:"ciphertext"`
		} `json:"encrypt_certificate"`
	} `json:"data"`
}

// NewPlatformCertManager 创建平台证书管理器
// client 参数用于反向引用以复用签名请求能力
// 若 client 已加载微信支付公钥（公钥模式），则同时拷贝到 certMgr 中以启用跳过下载逻辑
func NewPlatformCertManager(cfg Config, client *V3Client) *PlatformCertManager {
	m := &PlatformCertManager{
		cfg:        cfg,
		client:     client,
		localCache: make(map[string]*rsa.PublicKey),
		localTTL:   10 * time.Minute,
		redisTTL:   10 * time.Minute,
	}
	// 拷贝微信支付公钥（如果已加载）
	if client.wxPayPublicKey != nil {
		m.wxPayPublicKey = client.wxPayPublicKey
		m.wxPayPublicKeyID = cfg.WxPayPublicKeyID
	}
	return m
}

// CurrentSerial 返回当前在用的平台证书序列号（用于请求头 Wechatpay-Serial）
// 微信支付公钥模式：直接返回公钥ID（如 PUB_KEY_ID_xxx），跳过 Redis 查询
// 平台证书模式：优先返回 Redis 中缓存的序列号；其次返回进程内存缓存的一个序列号
func (m *PlatformCertManager) CurrentSerial() string {
	// 1. 支付公钥模式优先
	if m.wxPayPublicKeyID != "" {
		return m.wxPayPublicKeyID
	}

	// 2. 平台证书模式 - 进程内存缓存
	m.mu.RLock()
	for serial := range m.localCache {
		m.mu.RUnlock()
		return serial
	}
	m.mu.RUnlock()

	// 3. 平台证书模式 - Redis 缓存
	if redis.Enabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		serial, err := redis.Client().Get(ctx, "wxpay:platform_cert:current_serial").Result()
		if err == nil && serial != "" {
			return serial
		}
	}
	return ""
}

// GetPublicKey 获取指定序列号对应的平台证书公钥
// 微信支付公钥模式：serial 为公钥ID 时直接返回预置公钥，跳过所有下载逻辑
// 平台证书模式：优先级 进程内存 > Redis > 重新下载证书列表
// 返回的公钥用于验证微信响应的 RSA-SHA256 签名
func (m *PlatformCertManager) GetPublicKey(ctx context.Context, serial string) (*rsa.PublicKey, error) {
	// 1. 支付公钥模式：serial 匹配公钥ID 时直接返回预置公钥
	//    注意：微信回调中 Wechatpay-Serial 头会传公钥ID，不需要任何查询
	if m.wxPayPublicKey != nil && m.wxPayPublicKeyID != "" && serial == m.wxPayPublicKeyID {
		return m.wxPayPublicKey, nil
	}

	// 2. 平台证书模式 - 进程内存缓存
	m.mu.RLock()
	if key, ok := m.localCache[serial]; ok {
		m.mu.RUnlock()
		return key, nil
	}
	m.mu.RUnlock()

	// 3. 平台证书模式 - Redis 缓存（PEM 字符串）
	if redis.Enabled() {
		ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		pemStr, err := redis.Client().Get(ctx2, "wxpay:platform_cert:"+serial).Result()
		if err == nil && pemStr != "" {
			if pubKey, parseErr := parsePEMPublicKey(pemStr); parseErr == nil {
				m.mu.Lock()
				m.localCache[serial] = pubKey
				m.mu.Unlock()
				return pubKey, nil
			}
		}
	}

	// 4. 平台证书模式 - 重新下载证书列表（首次使用或缓存全过期时触发）
	//    支付公钥模式不会走到这里（上面第 1 步已返回）
	if err := m.downloadAndCache(ctx); err != nil {
		return nil, fmt.Errorf("下载平台证书失败: %w", err)
	}

	// 5. 再查进程内存（下载完成后应当已有缓存）
	m.mu.RLock()
	defer m.mu.RUnlock()
	if key, ok := m.localCache[serial]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("平台证书序列号 %s 不在微信返回的证书列表中（可能为已过期证书，建议稍后重试）", serial)
}

// downloadAndCache 调用 GET /v3/certificates 下载证书列表 + AES-256-GCM 解密 + 缓存
// 该方法会复用 V3Client 的签名请求能力（client.doRequest）
func (m *PlatformCertManager) downloadAndCache(ctx context.Context) error {
	// 防止并发重复下载：若距上次下载不足 1 分钟，跳过（极短时间内多个并发请求触发下载时复用上次结果）
	m.mu.RLock()
	if !m.cacheTime.IsZero() && time.Since(m.cacheTime) < time.Minute {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	// 复用 V3Client 的请求能力（带商户签名 + 验签响应）
	respBody, err := m.client.doRequest(ctx, "GET", pathCertList, nil, false)
	if err != nil {
		return fmt.Errorf("调用 /v3/certificates 失败: %w", err)
	}

	var resp certListResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("解析证书列表响应失败: %w", err)
	}

	if len(resp.Data) == 0 {
		return fmt.Errorf("微信返回的平台证书列表为空")
	}

	// 遍历所有证书，逐一 AES-256-GCM 解密 + 缓存
	var latestSerial string
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空进程内存缓存（避免已过期证书残留）
	m.localCache = make(map[string]*rsa.PublicKey)

	for _, item := range resp.Data {
		pemBytes, err := decryptCertificate(m.cfg.APIv3Key, item.EncryptCertificate.AssociatedData, item.EncryptCertificate.Nonce, item.EncryptCertificate.Ciphertext)
		if err != nil {
			// 单个证书解密失败不中断整体流程，记录后继续处理其他证书
			log.Printf("[WARN] 平台证书 %s 解密失败: %v\n", item.SerialNo, err)
			continue
		}
		pemStr := string(pemBytes)
		pubKey, err := parsePEMPublicKey(pemStr)
		if err != nil {
			log.Printf("[WARN] 平台证书 %s PEM 解析失败: %v\n", item.SerialNo, err)
			continue
		}
		m.localCache[item.SerialNo] = pubKey
		// 缓存到 Redis（PEM 字符串，方便下次启动读取后直接解析）
		if redis.Enabled() {
			ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			redis.Client().Set(ctx2, "wxpay:platform_cert:"+item.SerialNo, pemStr, m.redisTTL)
		}
		latestSerial = item.SerialNo
	}

	if len(m.localCache) == 0 {
		return fmt.Errorf("全部平台证书解密失败")
	}

	// 记录当前序列号到 Redis，方便请求头 Wechatpay-Serial 使用
	if redis.Enabled() && latestSerial != "" {
		ctx3, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		redis.Client().Set(ctx3, "wxpay:platform_cert:current_serial", latestSerial, m.redisTTL)
	}

	m.cacheTime = time.Now()
	return nil
}

// decryptCertificate AES-256-GCM 解密平台证书 / 回调通知 ciphertext
// 算法：AEAD_AES_256_GCM
// 参数：
//   apiV3Key: 32 字节 APIv3 密钥
//   associatedData: 附加数据（可能为空字符串）
//   nonce: 12 字节随机串
//   ciphertext: base64 编码的密文
// 返回：解密后的原始字节流（可能是 PEM 证书文本，也可能是回调明文 JSON）
func decryptCertificate(apiV3Key, associatedData, nonce, ciphertext string) ([]byte, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("ciphertext base64 解码失败: %w", err)
	}
	nonceBytes := []byte(nonce)
	// 验证 nonce 长度（GCM 规范要求 12 字节，微信固定用 12）
	if len(nonceBytes) != 12 {
		return nil, fmt.Errorf("nonce 长度必须为 12 字节，实际 %d", len(nonceBytes))
	}

	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		return nil, fmt.Errorf("AES cipher 创建失败（请检查 APIv3Key 是否为 32 字节）: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM 创建失败: %w", err)
	}

	// AES-GCM 解密：Open 会验证末尾 16 字节 GCM tag，验证失败说明密文被篓改或 APIv3Key 错误
	plaintext, err := gcm.Open(nil, nonceBytes, ciphertextBytes, []byte(associatedData))
	if err != nil {
		return nil, fmt.Errorf("AES-GCM 解密失败（请确认 APIv3Key 是否与商户后台一致）: %w", err)
	}

	return plaintext, nil
}

// parsePEMPublicKey 从 PEM 字符串解析 RSA 公钥
func parsePEMPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("PEM 解析失败（无 PEM 块）")
	}
	// 优先尝试 x509.ParsePKIXPublicKey（标准公钥格式）
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// 兼容老式 PKCS#1 公钥
		if pub1, err1 := x509.ParsePKCS1PublicKey(block.Bytes); err1 == nil {
			return pub1, nil
		}
		return nil, fmt.Errorf("公钥解析失败（PKIX err=%v）", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("解析出的公钥不是 RSA 类型")
	}
	return rsaPub, nil
}
