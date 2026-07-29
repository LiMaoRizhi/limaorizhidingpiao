// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port            int    `yaml:"port"`
	Mode            string `yaml:"mode"`
	TrustedProxies  string `yaml:"trusted_proxies"` // 逗号分隔的可信代理IP/CIDR，留空则不信任任何代理（防X-Forwarded-For伪造）
}

type MySQLConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
}

type JWTConfig struct {
	AdminSecret  string `yaml:"admin_secret"`
	WXSecret     string `yaml:"wx_secret"`
	DriverSecret string `yaml:"driver_secret"` // 司机端独立密钥
	AdminExpire   int    `yaml:"admin_expire"`   // seconds
	WXExpire      int    `yaml:"wx_expire"`      // seconds
	DriverExpire  int    `yaml:"driver_expire"`  // seconds 司机端Token有效期
}

type UploadConfig struct {
	Path       string `yaml:"path"`
	URLPrefix  string `yaml:"url_prefix"`
	MaxSize    int64  `yaml:"max_size"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type WechatConfig struct {
	Appid            string `yaml:"appid"`
	Secret           string `yaml:"secret"`
	MchID            string `yaml:"mch_id"`

	// === APIv3 字段 ===
	APIv3Key         string `yaml:"apiv3_key"`               // APIv3 密钥（32字节），用于回调解密（AES-256-GCM）和平台证书下载解密
	MchSerialNo      string `yaml:"mch_serial_no"`           // 商户证书序列号（16进制字符串，用于请求头 Wechatpay-Serial）
	MchPrivateKeyPath string `yaml:"mch_private_key_path"` // 商户私钥 PEM 文件路径（apiclient_key.pem），用于请求签名
	MchCertPEMPath   string `yaml:"mch_cert_pem_path"`     // 商户证书 PEM 文件路径（apiclient_cert.pem），用于 v3 敏感接口 mTLS

	// === 微信支付公钥模式（2024 新方案，推荐）===
	// 配置后跳过 /v3/certificates 平台证书自动下载，直接用此公钥验签响应和回调
	// 适用于商户后台已申请「微信支付公钥」的场景（公钥长期有效，无需轮换）
	WxPayPublicKeyPath string `yaml:"wxpay_public_key_path"` // 微信支付公钥 PEM 文件路径
	WxPayPublicKeyID   string `yaml:"wxpay_public_key_id"`   // 微信支付公钥 ID（如 PUB_KEY_ID_xxx），用于请求头 Wechatpay-Serial

	NotifyURL        string `yaml:"notify_url"`         // 支付回调通知地址
	RefundNotifyURL  string `yaml:"refund_notify_url"`   // 退款回调通知地址

	// 订阅消息模板ID（在小程序后台→功能→订阅消息中申请）
	// 留空则禁用对应通知类型，不影响其他功能
	// 前端调用 wx.requestSubscribeMessage 时传入这些模板ID，用户授权后后端获得1次发送配额
	SubscribeTemplates SubscribeTemplatesConfig `yaml:"subscribe_templates"`
}

// SubscribeTemplatesConfig 订阅消息模板配置
// 每个模板ID在微信小程序后台申请获得，留空表示该类型通知未启用
type SubscribeTemplatesConfig struct {
	PaymentSuccess string `yaml:"payment_success"` // 支付成功通知模板ID
	TripDeparture  string `yaml:"trip_departure"`  // 班次发车通知模板ID（司机出发/自动发车时触发）
	TripArrival    string `yaml:"trip_arrival"`    // 班次到达通知模板ID（班次到达终点时触发）
	RefundSuccess  string `yaml:"refund_success"`  // 退款到账通知模板ID
}

type IDCardVerifyConfig struct {
	Enabled     bool   `yaml:"enabled"`             // 是否启用实名认证
	StrictMode  bool   `yaml:"strict_mode"`          // 严格模式：API失败时拒绝操作
	AppCode     string `yaml:"app_code"`            // 阿里云云市场AppCode
	Endpoint    string `yaml:"endpoint"`            // API地址
	Path        string `yaml:"path"`                // API路径
	CacheTTL    int    `yaml:"cache_ttl"`           // 认证结果缓存有效期(秒)，0表示使用默认30天。可通过环境变量 IDCARD_VERIFY_CACHE_TTL 覆盖
}

// SecurityConfig 安全相关配置（敏感字段加解密 + 核销凭证签名）
type SecurityConfig struct {
	IDCardAESKey string `yaml:"id_card_aes_key"` // 身份证号AES-256加密密钥(base64编码的32字节)，必须通过环境变量 IDCARD_AES_KEY 注入
	VerifySecret string `yaml:"verify_secret"` // 核销凭证HMAC签名密钥，必须通过环境变量 VERIFY_SECRET 注入随机32字节密钥
}

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	MySQL       MySQLConfig       `yaml:"mysql"`
	Redis       RedisConfig       `yaml:"redis"`
	JWT         JWTConfig         `yaml:"jwt"`
	Wechat      WechatConfig      `yaml:"wechat"`
	Upload      UploadConfig      `yaml:"upload"`
	CORSOrigins string            `yaml:"cors_origins"` // 逗号分隔的CORS白名单
	IDCardVerify IDCardVerifyConfig `yaml:"idcard_verify"` // 身份实名认证配置
	Security     SecurityConfig     `yaml:"security"`      // 安全配置（敏感数据加解密）
}

var AppConfig Config

// envOverride 用环境变量覆盖敏感配置项，优先级: 环境变量 > YAML文件
func envOverride() {
	// MySQL
	if v := os.Getenv("MYSQL_HOST"); v != "" {
		AppConfig.MySQL.Host = v
	}
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		AppConfig.MySQL.Password = v
	}
	if v := os.Getenv("MYSQL_USERNAME"); v != "" {
		AppConfig.MySQL.Username = v
	}
	if v := os.Getenv("MYSQL_DATABASE"); v != "" {
		AppConfig.MySQL.Database = v
	}
	// Redis
	if v := os.Getenv("REDIS_HOST"); v != "" {
		AppConfig.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		AppConfig.Redis.Password = v
	}
	// JWT
	if v := os.Getenv("JWT_ADMIN_SECRET"); v != "" {
		AppConfig.JWT.AdminSecret = v
	}
	if v := os.Getenv("JWT_WX_SECRET"); v != "" {
		AppConfig.JWT.WXSecret = v
	}
	if v := os.Getenv("JWT_DRIVER_SECRET"); v != "" {
		AppConfig.JWT.DriverSecret = v
	}
	// WeChat
	if v := os.Getenv("WECHAT_APPID"); v != "" {
		AppConfig.Wechat.Appid = v
	}
	if v := os.Getenv("WECHAT_SECRET"); v != "" {
		AppConfig.Wechat.Secret = v
	}
	if v := os.Getenv("WECHAT_MCH_ID"); v != "" {
		AppConfig.Wechat.MchID = v
	}
	if v := os.Getenv("WECHAT_NOTIFY_URL"); v != "" {
		AppConfig.Wechat.NotifyURL = v
	}
	if v := os.Getenv("WECHAT_REFUND_NOTIFY_URL"); v != "" {
		AppConfig.Wechat.RefundNotifyURL = v
	}
	// 订阅消息模板ID
	if v := os.Getenv("WECHAT_SUBSCRIBE_PAYMENT_SUCCESS"); v != "" {
		AppConfig.Wechat.SubscribeTemplates.PaymentSuccess = v
	}
	if v := os.Getenv("WECHAT_SUBSCRIBE_TRIP_DEPARTURE"); v != "" {
		AppConfig.Wechat.SubscribeTemplates.TripDeparture = v
	}
	if v := os.Getenv("WECHAT_SUBSCRIBE_TRIP_ARRIVAL"); v != "" {
		AppConfig.Wechat.SubscribeTemplates.TripArrival = v
	}
	if v := os.Getenv("WECHAT_SUBSCRIBE_REFUND_SUCCESS"); v != "" {
		AppConfig.Wechat.SubscribeTemplates.RefundSuccess = v
	}
	// APIv3 配置
	if v := os.Getenv("WECHAT_API_V3_KEY"); v != "" {
		AppConfig.Wechat.APIv3Key = v
	}
	if v := os.Getenv("WECHAT_MCH_SERIAL_NO"); v != "" {
		AppConfig.Wechat.MchSerialNo = v
	}
	if v := os.Getenv("WECHAT_MCH_PRIVATE_KEY_PATH"); v != "" {
		AppConfig.Wechat.MchPrivateKeyPath = v
	}
	if v := os.Getenv("WECHAT_MCH_CERT_PEM_PATH"); v != "" {
		AppConfig.Wechat.MchCertPEMPath = v
	}
	// 微信支付公钥模式（2024 新方案）
	if v := os.Getenv("WECHAT_WXPAY_PUBLIC_KEY_PATH"); v != "" {
		AppConfig.Wechat.WxPayPublicKeyPath = v
	}
	if v := os.Getenv("WECHAT_WXPAY_PUBLIC_KEY_ID"); v != "" {
		AppConfig.Wechat.WxPayPublicKeyID = v
	}
	// Server
	if v := os.Getenv("SERVER_MODE"); v != "" {
		AppConfig.Server.Mode = v
	}
	// CORS
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		AppConfig.CORSOrigins = v
	}
	// 身份实名认证
	if v := os.Getenv("IDCARD_VERIFY_APPCODE"); v != "" {
		AppConfig.IDCardVerify.AppCode = v
	}
	if v := os.Getenv("IDCARD_VERIFY_ENABLED"); v != "" {
		AppConfig.IDCardVerify.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("IDCARD_VERIFY_STRICT_MODE"); v != "" {
		AppConfig.IDCardVerify.StrictMode = v == "true" || v == "1"
	}
	// 认证结果缓存有效期（秒），非负整数。不合法的值将被忽略并警告。
	if v := os.Getenv("IDCARD_VERIFY_CACHE_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			AppConfig.IDCardVerify.CacheTTL = n
		} else {
			log.Printf("[WARN] 无效的 IDCARD_VERIFY_CACHE_TTL 值: %s，已忽略（需为非负整数）", v)
		}
	}
	// 身份证加密密钥
	if v := os.Getenv("IDCARD_AES_KEY"); v != "" {
		AppConfig.Security.IDCardAESKey = v
	}
	// 核销凭证签名密钥
	if v := os.Getenv("VERIFY_SECRET"); v != "" {
		AppConfig.Security.VerifySecret = v
	}
	// 启动时校验必填密钥（release模式下密钥不能为空）
	if AppConfig.Server.Mode == "release" {
		if AppConfig.JWT.AdminSecret == "" || AppConfig.JWT.WXSecret == "" {
			log.Fatalf("生产环境必须通过环境变量配置 JWT_ADMIN_SECRET 和 JWT_WX_SECRET")
		}
		if AppConfig.JWT.DriverSecret == "" {
			log.Fatalf("生产环境必须通过环境变量配置 JWT_DRIVER_SECRET，不能与 AdminSecret 共用")
		}
		if AppConfig.MySQL.Password == "" {
			log.Fatalf("生产环境必须通过环境变量 MYSQL_PASSWORD 配置数据库密码")
		}
		// 身份证号加密密钥：生产环境必须配置，防止数据库泄露时身份证号明文暴露
		if AppConfig.Security.IDCardAESKey == "" {
			log.Fatalf("生产环境必须通过环境变量 IDCARD_AES_KEY 配置身份证加密密钥(base64编码的32字节)")
		}
		// 核销凭证签名密钥：生产环境必须配置，防止二维码伪造
		if AppConfig.Security.VerifySecret == "" {
			log.Fatalf("生产环境必须通过环境变量 VERIFY_SECRET 配置核销凭证签名密钥")
		}
		// Redis：生产环境强烈建议配置，多实例部署时限流和登录失败计数需要 Redis 共享
		if AppConfig.Redis.Host == "" {
			log.Println("[警告] 生产环境未配置Redis，限流和登录失败计数将使用进程内存，多实例部署时可能被绕过。请通过环境变量 REDIS_HOST 配置")
		}
		// 微信支付配置（release模式必须配置，防止模拟支付绕过）
		if AppConfig.Wechat.Appid == "" || AppConfig.Wechat.Secret == "" ||
			AppConfig.Wechat.MchID == "" ||
			AppConfig.Wechat.NotifyURL == "" {
			log.Fatalf("生产环境必须配置微信支付参数（WECHAT_APPID, WECHAT_SECRET, WECHAT_MCH_ID, notify_url）")
		}
		// APIv3 密钥：用于回调通知 AES-256-GCM 解密，缺失会导致回调无法验证
		if AppConfig.Wechat.APIv3Key == "" {
			log.Fatalf("生产环境必须通过环境变量 WECHAT_API_V3_KEY 配置 APIv3 密钥（32字节，商户后台→账户中心→API安全→APIv3密钥）")
		}
		// 商户证书序列号 + 商户私钥：用于请求 RSA-SHA256 签名
		if AppConfig.Wechat.MchSerialNo == "" {
			log.Fatalf("生产环境必须通过环境变量 WECHAT_MCH_SERIAL_NO 配置商户证书序列号")
		}
		if AppConfig.Wechat.MchPrivateKeyPath == "" {
			log.Fatalf("生产环境必须通过环境变量 WECHAT_MCH_PRIVATE_KEY_PATH 配置商户私钥 PEM 文件路径（apiclient_key.pem）")
		}
		// 微信支付公钥模式：若配置了公钥路径则必须配套公钥ID（两者成对使用）
		if AppConfig.Wechat.WxPayPublicKeyPath != "" && AppConfig.Wechat.WxPayPublicKeyID == "" {
			log.Fatalf("配置了 WECHAT_WXPAY_PUBLIC_KEY_PATH 时必须同时配置 WECHAT_WXPAY_PUBLIC_KEY_ID（公钥ID，如 PUB_KEY_ID_xxx）")
		}
		// 实名认证：若启用但未配置AppCode，仅警告不阻止启动（实名认证API调用时会返回错误，其他功能不受影响）
		if AppConfig.IDCardVerify.Enabled && AppConfig.IDCardVerify.AppCode == "" {
			log.Println("[警告] 实名认证已启用(IDCARD_VERIFY_ENABLED=true)但未配置IDCARD_VERIFY_APPCODE，实名认证功能将不可用，其他功能不受影响。请在.env中配置IDCARD_VERIFY_APPCODE以启用")
		}
	}
}

// loadEnvFile 从 .env 文件加载环境变量（如果存在）
// 逐行解析 KEY=VALUE 格式，跳过注释和空行
// 优先级：系统环境变量 > .env 文件 > config.yaml 默认值
// 设计目的：不依赖 start.sh 的 source .env，使直接运行二进制时也能正确加载配置
func loadEnvFile(configPath string) {
	// 1. 尝试 config.yaml 同目录下的 .env
	envPath := filepath.Join(filepath.Dir(configPath), ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		// 2. 尝试可执行文件同目录下的 .env
		exe, exeErr := os.Executable()
		if exeErr == nil {
			envPath = filepath.Join(filepath.Dir(exe), ".env")
			data, err = os.ReadFile(envPath)
		}
		// 3. 尝试工作目录下的 .env
		if err != nil {
			data, err = os.ReadFile(".env")
		}
		if err != nil {
			return // .env 不存在，跳过（依赖系统环境变量或 start.sh）
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 解析 KEY=VALUE
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		// 去除值两端的引号
		value = strings.Trim(value, `"'`)
		// 仅当系统环境变量未设置时才设置（系统环境变量优先级更高，允许 start.sh 或 docker -e 覆盖）
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
	log.Println("[INFO] 已从 .env 文件加载环境变量")
}

func Load(path string) error {
	// 先加载 .env 文件（如果存在），将变量注入系统环境变量
	// 优先级：系统环境变量 > .env 文件 > config.yaml 默认值
	loadEnvFile(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return err
	}
	// 环境变量覆盖敏感配置（优先级高于YAML文件）
	envOverride()
	return nil
}
