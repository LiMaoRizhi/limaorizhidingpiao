// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
// 微信支付 APIv3 客户端
// 不用第三方SDK，搁标准库里自己怼的，省得引一堆依赖。签名验签解密全搁这一块弄。
package v3

import (
	"bytes"
	"context"
	"crypto"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// ErrMTLSNotConfigured 没配mTLS证书就调退款，那可不中
var ErrMTLSNotConfigured = errors.New("退款要mTLS证书，配一下 WECHAT_MCH_CERT_PEM_PATH 指向 apiclient_cert.pem")

// 微信v3接口地址
const (
	baseURL      = "https://api.mch.weixin.qq.com"
	pathJSAPIPay = "/v3/pay/transactions/jsapi"
	pathRefund   = "/v3/refund/domestic/refunds"
	pathCertList = "/v3/certificates"
)

// Config 配置项，从 config.WechatConfig 注入
type Config struct {
	AppID           string
	MchID           string
	APIv3Key        string  // 32字节，用于回调解密和平台证书下载解密
	MchSerialNo     string  // 商户证书序列号
	PrivateKeyPath  string  // apiclient_key.pem 路径（请求签名用）
	CertPEMPath     string  // apiclient_cert.pem 路径（mTLS 双向认证用，退款等敏感接口必需）
	NotifyURL      string
	RefundNotifyURL string

	// 微信支付公钥模式（2024 新方案，可选）
	// 配置后跳过 /v3/certificates 平台证书下载，直接用此公钥验签响应和回调
	// 需与 WxPayPublicKeyID 成对使用，两者必须同时设置或同时留空
	WxPayPublicKeyPath string  // 微信支付公钥 PEM 文件路径
	WxPayPublicKeyID   string  // 微信支付公钥 ID（如 PUB_KEY_ID_xxx），用于请求头 Wechatpay-Serial
}

// V3Client 微信支付v3客户端，单例，建好后并发安全
type V3Client struct {
	cfg        Config
	privateKey *rsa.PrivateKey
	httpClient *http.Client  // 普通客户端
	mtlsClient *http.Client  // mTLS 客户端（退款等敏感接口）
	certMgr    *PlatformCertManager

	// 微信支付公钥模式（可选）：不为 nil 时跳过平台证书下载
	wxPayPublicKey *rsa.PublicKey
}

// ErrorResponse 微信 v3 错误响应体
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  []struct {
		Issue string `json:"issue"`
		Field string `json:"field"`
	} `json:"detail"`
}

// NewV3Client 建客户端
// APIv3Key、MchSerialNo、PrivateKeyPath 这仨必填
// CertPEMPath 选填，不配的话退款调不了
func NewV3Client(cfg Config) (*V3Client, error) {
	if cfg.APIv3Key == "" {
		return nil, errors.New("APIv3 密钥不能为空")
	}
	if cfg.MchSerialNo == "" {
		return nil, errors.New("商户证书序列号不能为空")
	}
	if cfg.PrivateKeyPath == "" {
		return nil, errors.New("商户私钥路径不能为空")
	}

	// 加载商户私钥（apiclient_key.pem，PKCS#8 格式，少数情况为 PKCS#1）
	keyData, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("读取商户私钥文件失败: %w", err)
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("商户私钥 PEM 格式无效")
	}
	var rsaKey *rsa.PrivateKey
	// 先试PKCS#8，微信给的就是这个格式，个别老证书是PKCS#1
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rk, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("商户私钥不是 RSA 密钥（PKCS#8）")
		}
		rsaKey = rk
	} else if pk1, err1 := x509.ParsePKCS1PrivateKey(block.Bytes); err1 == nil {
		// 兼容老式 PKCS#1 格式
		rsaKey = pk1
	} else {
		return nil, fmt.Errorf("商户私钥解析失败（PKCS8 err=%v, PKCS1 err=%v）", err, err1)
	}

	client := &V3Client{
		cfg:        cfg,
		privateKey: rsaKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	// mTLS客户端，把apiclient_cert.pem和apiclient_key.pem拼成tls.Certificate
	if cfg.CertPEMPath != "" {
		tlsCert, err := tls.LoadX509KeyPair(cfg.CertPEMPath, cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("加载 mTLS 证书对失败（apiclient_cert.pem + apiclient_key.pem）: %w", err)
		}
		client.mtlsClient = &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{tlsCert},
					MinVersion:   tls.VersionTLS12,
				},
			},
		}
	}

	// 微信支付公钥（2024新方案），配了就不用去下载平台证书了，省事
	if cfg.WxPayPublicKeyPath != "" {
		if cfg.WxPayPublicKeyID == "" {
			return nil, errors.New("配置了 WxPayPublicKeyPath 时必须同时配置 WxPayPublicKeyID（公钥ID，如 PUB_KEY_ID_xxx）")
		}
		pubKeyData, err := os.ReadFile(cfg.WxPayPublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("读取微信支付公钥文件失败: %w", err)
		}
		pubKey, err := parsePEMPublicKey(string(pubKeyData))
		if err != nil {
			return nil, fmt.Errorf("解析微信支付公钥 PEM 失败: %w", err)
		}
		client.wxPayPublicKey = pubKey
		log.Printf("[INFO] 微信支付公钥模式已启用，公钥ID=%s（跳过平台证书自动下载）\n", cfg.WxPayPublicKeyID)
	}

	// 证书管理器，头回用的时候去微信下载+解密，之后搁Redis缓存
	// 传client进去让certMgr能复用doRequest（同包互引，不构成循环依赖）
	client.certMgr = NewPlatformCertManager(cfg, client)

	return client, nil
}

// HasMTLS 配没配mTLS，退款接口得检查这个
func (c *V3Client) HasMTLS() bool {
	return c.mtlsClient != nil
}

// generateNonceStr 随机串，16字节hex
func generateNonceStr() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		// 理论上不会挂，兜底用时间戳，别让进程崩了
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// buildAuthorization 拼请求头Authorization
// 签名串格式死板的：method\npath\ntimestamp\nnonce\nbody\n
// 最后那个\n可不敢少，少了对不上，坑过一回
func (c *V3Client) buildAuthorization(method, path string, body []byte) (string, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := generateNonceStr()
	// body 为空时签名为空串，但仍保留末尾换行符
	bodyStr := ""
	if len(body) > 0 {
		bodyStr = string(body)
	}
	signStr := method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + bodyStr + "\n"
	h := sha256.Sum256([]byte(signStr))
	sig, err := rsa.SignPKCS1v15(crand.Reader, c.privateKey, crypto.SHA256, h[:])
	if err != nil {
		return "", fmt.Errorf("RSA 签名失败: %w", err)
	}
	signature := base64.StdEncoding.EncodeToString(sig)
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`,
		c.cfg.MchID, nonce, timestamp, c.cfg.MchSerialNo, signature,
	), nil
}

// doRequest 发请求+验签
// useMTLS=true 走mTLS客户端（退款专用）
func (c *V3Client) doRequest(ctx context.Context, method, path string, body []byte, useMTLS bool) ([]byte, error) {
	var httpClient *http.Client
	if useMTLS {
		if c.mtlsClient == nil {
			return nil, ErrMTLSNotConfigured
		}
		httpClient = c.mtlsClient
	} else {
		httpClient = c.httpClient
	}

	auth, err := c.buildAuthorization(method, path, body)
	if err != nil {
		return nil, err
	}

	fullURL := baseURL + path
	var req *http.Request
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Wechatpay-Serial告诉微信用哪个证书加密响应，头回下载前可能不知道，留空让微信自己选
	if serial := c.certMgr.CurrentSerial(); serial != "" {
		req.Header.Set("Wechatpay-Serial", serial)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 业务层失败：HTTP 状态码 >= 400
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Code != "" {
			return nil, fmt.Errorf("微信 API 错误: code=%s message=%s (HTTP %d)",
				errResp.Code, errResp.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("微信 API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// 验签响应签名（仅当所有签名头都存在时才校验）
	if err := c.verifyResponse(ctx, resp, respBody); err != nil {
		return nil, fmt.Errorf("响应验签失败: %w", err)
	}

	return respBody, nil
}

// verifyResponse 验签微信的响应
// 四个签名头缺一个就跳过（比如204没body那种）
func (c *V3Client) verifyResponse(ctx context.Context, resp *http.Response, respBody []byte) error {
	timestamp := resp.Header.Get("Wechatpay-Timestamp")
	nonce := resp.Header.Get("Wechatpay-Nonce")
	serial := resp.Header.Get("Wechatpay-Serial")
	signature := resp.Header.Get("Wechatpay-Signature")

	if timestamp == "" || nonce == "" || serial == "" || signature == "" {
		// 没有签名头（如 204 No Content 等），跳过验签
		return nil
	}

	pubKey, err := c.certMgr.GetPublicKey(ctx, serial)
	if err != nil {
		return fmt.Errorf("获取平台证书公钥失败 serial=%s: %w", serial, err)
	}

	// 验签串：必须按 timestamp\nnonce\nbody\n 顺序
	signStr := timestamp + "\n" + nonce + "\n" + string(respBody) + "\n"
	h := sha256.Sum256([]byte(signStr))

	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("签名 base64 解码失败: %w", err)
	}

	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, h[:], sig); err != nil {
		return fmt.Errorf("响应签名验证失败（可能遭受中间人攻击，请立即排查）: %w", err)
	}
	return nil
}

// PostRequest 简单 POST 请求（不带 mTLS）
func (c *V3Client) PostRequest(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.doRequest(ctx, "POST", path, body, false)
}

// PostRequestMTLS POST 请求（带 mTLS，用于退款等敏感接口）
func (c *V3Client) PostRequestMTLS(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.doRequest(ctx, "POST", path, body, true)
}

// JSAPI 下单（v3）

// JSAPIPayRequest JSAPI 下单请求体
type JSAPIPayRequest struct {
	Appid       string `json:"appid"`
	Mchid       string `json:"mchid"`
	Description string `json:"description"`
	OutTradeNo  string `json:"out_trade_no"`
	NotifyURL   string `json:"notify_url"`
	Amount      struct {
		Total    int    `json:"total"`     // 金额：分
		Currency string `json:"currency"`  // CNY
	} `json:"amount"`
	Payer struct {
		OpenID string `json:"openid"`
	} `json:"payer"`
}

// JSAPIPayResponse JSAPI 下单响应体
type JSAPIPayResponse struct {
	PrepayID string `json:"prepay_id"`
}

// CreateJSAPIPay 调用 v3 JSAPI 下单接口
// 返回 prepay_id，调用方需配合 BuildJSAPIPayParams 构建小程序调起参数
func (c *V3Client) CreateJSAPIPay(ctx context.Context, orderNo string, totalFee int, openID string) (string, error) {
	req := JSAPIPayRequest{
		Appid:       c.cfg.AppID,
		Mchid:       c.cfg.MchID,
		Description: "狸猫日志-车票订单",
		OutTradeNo:  orderNo,
		NotifyURL:   c.cfg.NotifyURL,
	}
	req.Amount.Total = totalFee
	req.Amount.Currency = "CNY"
	req.Payer.OpenID = openID

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化下单请求失败: %w", err)
	}

	respBody, err := c.PostRequest(ctx, pathJSAPIPay, body)
	if err != nil {
		return "", err
	}

	var resp JSAPIPayResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("解析下单响应失败: %w", err)
	}
	if resp.PrepayID == "" {
		return "", errors.New("微信 v3 下单返回 prepay_id 为空")
	}
	return resp.PrepayID, nil
}

// BuildJSAPIPayParams 拼小程序wx.requestPayment的调起参数
// 签名串：appId\n时间戳\n随机串\nprepay_id=xxx\n
// 注意appId是小写，跟上面请求签名那套不一样
func (c *V3Client) BuildJSAPIPayParams(prepayID string) (map[string]string, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonceStr := generateNonceStr()
	packageStr := "prepay_id=" + prepayID
	// 调起支付的签名串（注意 appId 小写）：
	//   appId\n时间戳\n随机字符串\nprepay_id=xxx\n
	signStr := c.cfg.AppID + "\n" + timestamp + "\n" + nonceStr + "\n" + packageStr + "\n"
	h := sha256.Sum256([]byte(signStr))
	sig, err := rsa.SignPKCS1v15(crand.Reader, c.privateKey, crypto.SHA256, h[:])
	if err != nil {
		return nil, fmt.Errorf("构建 JSAPI 调起参数签名失败: %w", err)
	}
	return map[string]string{
		"appId":     c.cfg.AppID,
		"timeStamp": timestamp,
		"nonceStr":  nonceStr,
		"package":   packageStr,
		"signType":  "RSA",
		"paySign":   base64.StdEncoding.EncodeToString(sig),
	}, nil
}

// 退款（v3）

// RefundRequest v3 退款请求体
type RefundRequest struct {
	TransactionID  string `json:"transaction_id,omitempty"`  // 微信支付单号（二选一）
	OutTradeNo     string `json:"out_trade_no,omitempty"`    // 商户订单号（二选一）
	OutRefundNo    string `json:"out_refund_no"`
	Reason         string `json:"reason,omitempty"`
	NotifyURL      string `json:"notify_url,omitempty"`
	Amount         struct {
		Refund   int `json:"refund"`     // 退款金额：分
		Total    int `json:"total"`      // 订单总额：分
		Currency string `json:"currency"`
	} `json:"amount"`
}

// RefundResponse v3 退款响应体
type RefundResponse struct {
	RefundID            string `json:"refund_id"`            // 微信退款单号
	OutRefundNo         string `json:"out_refund_no"`        // 商户退款单号
	Status              string `json:"status"`               // SUCCESS/CLOSED/PROCESSING/ABNORMAL
	FundsAccount        string `json:"funds_account,omitempty"`
	Amount              struct {
		Refund   int `json:"refund"`
		Total    int `json:"total"`
		Currency string `json:"currency"`
	} `json:"amount"`
}

// CreateRefund v3退款，得走mTLS
// 全额退款时 refundFee = totalFee 就行
func (c *V3Client) CreateRefund(ctx context.Context, outTradeNo string, totalFee int, refundNo, transactionID string, refundFee int) (*RefundResponse, error) {
	req := RefundRequest{
		TransactionID: transactionID,
		OutTradeNo:   outTradeNo,
		OutRefundNo:  refundNo,
		Reason:       "订单退款",
		NotifyURL:    c.cfg.RefundNotifyURL,
	}
	req.Amount.Refund = refundFee
	req.Amount.Total = totalFee
	req.Amount.Currency = "CNY"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化退款请求失败: %w", err)
	}

	respBody, err := c.PostRequestMTLS(ctx, pathRefund, body)
	if err != nil {
		return nil, err
	}

	var resp RefundResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("解析退款响应失败: %w", err)
	}
	return &resp, nil
}
