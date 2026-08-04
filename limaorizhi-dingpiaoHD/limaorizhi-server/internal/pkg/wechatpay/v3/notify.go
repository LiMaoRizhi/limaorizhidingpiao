// 微信支付 v3 回调通知验签 + AES-256-GCM 解密
package v3

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NotifyHeader 微信回调通知的签名头
// 4 个头必须全部存在，缺一不可
type NotifyHeader struct {
	Timestamp string // Wechatpay-Timestamp
	Nonce     string // Wechatpay-Nonce
	Serial    string // Wechatpay-Serial
	Signature string // Wechatpay-Signature
}

// ParseNotifyHeader 从 HTTP 请求头解析微信签名头
func ParseNotifyHeader(r *http.Request) NotifyHeader {
	return NotifyHeader{
		Timestamp: r.Header.Get("Wechatpay-Timestamp"),
		Nonce:     r.Header.Get("Wechatpay-Nonce"),
		Serial:    r.Header.Get("Wechatpay-Serial"),
		Signature: r.Header.Get("Wechatpay-Signature"),
	}
}

// NotifyEnvelope 回调通知外层信封
type NotifyEnvelope struct {
	ID           string `json:"id"`
	CreateTime   string `json:"create_time"`
	ResourceType string `json:"resource_type"`
	EventType   string `json:"event_type"`
	Summary      string `json:"summary"`
	Resource     struct {
		Algorithm      string `json:"algorithm"`      // AEAD_AES_256_GCM
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

// PayNotifyResource 支付回调通知解密后的明文（仅业务用到的字段）
type PayNotifyResource struct {
	TransactionID string `json:"transaction_id"`  // 微信支付订单号
	OutTradeNo    string `json:"out_trade_no"`    // 商户订单号
	TradeState    string `json:"trade_state"`     // SUCCESS/REFUND/NOTPAY/CLOSED/REVOKED/USERPAYING/PAYERROR
	TradeType     string `json:"trade_type"`      // JSAPI
	BankType      string `json:"bank_type,omitempty"`
	Payer         struct {
		OpenID string `json:"openid"`
	} `json:"payer,omitempty"`
	SuccessTime string `json:"success_time,omitempty"` // ISO 8601
	Amount      struct {
		PayerTotal    int    `json:"payer_total"`     // 用户支付金额（分）
		Total         int    `json:"total"`           // 订单总金额（分）
		Currency      string `json:"currency"`        // CNY
		PayerCurrency string `json:"payer_currency"`  // 用户支付币种
	} `json:"amount"`
}

// RefundNotifyResource 退款回调通知解密后的明文
type RefundNotifyResource struct {
	RefundID            string `json:"refund_id"`             // 微信退款单号
	OutRefundNo         string `json:"out_refund_no"`         // 商户退款单号
	TransactionID       string `json:"transaction_id"`        // 微信支付订单号
	OutTradeNo          string `json:"out_trade_no"`         // 商户订单号
	RefundStatus        string `json:"refund_status"`        // SUCCESS/CLOSED/PROCESSING/ABNORMAL
	SuccessTime         string `json:"success_time,omitempty"`
	UserReceivedAccount string `json:"user_received_account,omitempty"`
	Amount              struct {
		Refund            int    `json:"refund"`             // 退款金额（分）
		Total             int    `json:"total"`              // 订单总金额（分）
		Currency          string `json:"currency"`           // CNY
		PayerRefund       int    `json:"payer_refund"`       // 用户退款金额
		PayerTotal        int    `json:"payer_total"`        // 用户支付金额
		PayerCurrency     string `json:"payer_currency"`     // 用户支付币种
	} `json:"amount"`
}

// VerifyAndDecryptNotify 验证回调通知签名 + 解密 resource.ciphertext
// 参数：
//   rawBody:  HTTP 请求原始 body（必须未修改）
//   header:   从 ParseNotifyHeader 得到的签名头
//   apiV3Key: 商户 APIv3 密钥（32 字节）
// 返回：
//   envelope: 原始信封（含 event_type、summary 等字段，用于业务分支判断）
//   plaintext: 解密后的明文 JSON（支付/退款回调的不同结构）
//   err: 验签失败 / 解密失败
//
// 安全要点：
//   - 验签必须用平台证书公钥（不是商户私钥），公钥通过 certMgr 自动加载
//   - 失败时必须返回 HTTP 401，让微信重试
//   - 通过后必须返回 HTTP 200 + {"code": "SUCCESS"}，否则微信会重试 8 次
func (c *V3Client) VerifyAndDecryptNotify(ctx context.Context, rawBody []byte, header NotifyHeader, apiV3Key string) (envelope NotifyEnvelope, plaintext []byte, err error) {
	// 1. 校验签名头完整性
	if header.Timestamp == "" || header.Nonce == "" || header.Serial == "" || header.Signature == "" {
		err = fmt.Errorf("回调通知缺少签名头（timestamp/nonce/serial/signature 必须全部存在）")
		return
	}

	// 2. 校验时间戳防重放（5 分钟之外的请求拒绝）
	// 防止攻击者截获旧通知重放，导致订单状态被重复更新
	var ts int64
	if _, perr := fmt.Sscanf(header.Timestamp, "%d", &ts); perr != nil {
		err = fmt.Errorf("时间戳格式无效: %s", header.Timestamp)
		return
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		err = fmt.Errorf("回调通知已过期（时间戳与当前相差超过 5 分钟）")
		return
	}

	// 3. 获取平台证书公钥（自动从 Redis 或 API 下载）
	pubKey, perr := c.certMgr.GetPublicKey(ctx, header.Serial)
	if perr != nil {
		err = fmt.Errorf("获取平台证书公钥失败 serial=%s: %w", header.Serial, perr)
		return
	}

	// 4. 验签
	// 签名串格式：timestamp\nnonce\nrawBody\n
	// rawBody 必须是原始字节，不能被任何中间件修改或重新序列化
	signStr := header.Timestamp + "\n" + header.Nonce + "\n" + string(rawBody) + "\n"
	h := sha256.Sum256([]byte(signStr))
	sigBytes, serr := base64.StdEncoding.DecodeString(header.Signature)
	if serr != nil {
		err = fmt.Errorf("签名 base64 解码失败: %w", serr)
		return
	}
	if verr := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, h[:], sigBytes); verr != nil {
		err = fmt.Errorf("回调通知签名验证失败（可能遭受中间人攻击）: %w", verr)
		return
	}

	// 5. 解析信封
	if uerr := json.Unmarshal(rawBody, &envelope); uerr != nil {
		err = fmt.Errorf("解析信封 JSON 失败: %w", uerr)
		return
	}

	// 6. AES-256-GCM 解密 resource.ciphertext
	plaintext, err = decryptCertificate(apiV3Key, envelope.Resource.AssociatedData, envelope.Resource.Nonce, envelope.Resource.Ciphertext)
	if err != nil {
		err = fmt.Errorf("回调通知 ciphertext 解密失败: %w", err)
		return
	}
	return
}

// DecryptPayNotify 解密支付回调通知明文为 PayNotifyResource
// 调用前应已通过 VerifyAndDecryptNotify 验签
func DecryptPayNotify(plaintext []byte) (*PayNotifyResource, error) {
	var r PayNotifyResource
	if err := json.Unmarshal(plaintext, &r); err != nil {
		return nil, fmt.Errorf("解析支付回调明文失败: %w", err)
	}
	return &r, nil
}

// DecryptRefundNotify 解密退款回调通知明文为 RefundNotifyResource
// 调用前应已通过 VerifyAndDecryptNotify 验签
func DecryptRefundNotify(plaintext []byte) (*RefundNotifyResource, error) {
	var r RefundNotifyResource
	if err := json.Unmarshal(plaintext, &r); err != nil {
		return nil, fmt.Errorf("解析退款回调明文失败: %w", err)
	}
	return &r, nil
}

// NotifySuccessResponse 回调成功响应（返回给微信的 JSON）
// 微信收到 200 + {"code":"SUCCESS"} 后停止重试
func NotifySuccessResponse() map[string]string {
	return map[string]string{"code": "SUCCESS", "message": "OK"}
}

// NotifyFailResponse 回调失败响应（返回给微信，触发重试）
// 微信最多重试 8 次，间隔逐渐拉长
func NotifyFailResponse(msg string) map[string]string {
	return map[string]string{"code": "FAIL", "message": msg}
}
