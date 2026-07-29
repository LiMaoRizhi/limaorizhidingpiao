// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package verifytoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"limaorizhi-server/internal/config"
)

// 凭证格式：LIMO|{orderNo}|{expireTs}|{sig}
//   - orderNo: 订单号（DP/HY开头）
//   - expireTs: 过期Unix时间戳（秒）
//   - sig: HMAC-SHA256 前16字符

const (
	prefix    = "LIMO"
	sep       = "|"
	sigHexLen = 16 // 截取HMAC hex前16字符（64位强度足够防伪，避免二维码过长）
)

// 订单号格式校验：DP(客运) 或 HY(托运) + 8位日期 + 8位hex = 18位
var orderNoPattern = regexp.MustCompile(`^(DP|HY)[0-9]{8}[0-9a-f]{8}$`)

// ErrInvalidToken 凭证格式非法
var ErrInvalidToken = errors.New("invalid verify token format")

// ErrSignatureMismatch 签名校验失败
var ErrSignatureMismatch = errors.New("verify token signature mismatch")

// ErrTokenExpired 凭证已过期
var ErrTokenExpired = errors.New("verify token expired")

// ErrInvalidOrderNo 订单号格式非法
var ErrInvalidOrderNo = errors.New("invalid order_no format")

// ValidateOrderNo 校验订单号格式（DP/HY + 8位日期 + 8位hex）
func ValidateOrderNo(orderNo string) error {
	if !orderNoPattern.MatchString(orderNo) {
		return ErrInvalidOrderNo
	}
	return nil
}

// sign 计算 HMAC-SHA256 签名（hex 前 sigHexLen 字符）
func sign(orderNo string, expireTs int64) (string, error) {
	secret := config.AppConfig.Security.VerifySecret
	if secret == "" {
		return "", errors.New("verify_secret not configured")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(orderNo))
	mac.Write([]byte(sep))
	mac.Write([]byte(strconv.FormatInt(expireTs, 10)))
	return hex.EncodeToString(mac.Sum(nil))[:sigHexLen], nil
}

// Generate 生成带签名+时效的核销凭证
//   - orderNo: 订单号
//   - expireTs: 过期Unix时间戳（秒）
//
// 返回字符串形如：LIMO|DP20260718a1b2c3d4|1784150400|9f8e7d6c5b4a3210
func Generate(orderNo string, expireTs int64) (string, error) {
	if err := ValidateOrderNo(orderNo); err != nil {
		return "", err
	}
	sig, err := sign(orderNo, expireTs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s%s%s%d%s%s", prefix, sep, orderNo, sep, expireTs, sep, sig), nil
}

// ParsedToken 解析后的凭证结构
type ParsedToken struct {
	OrderNo  string
	ExpireTs int64
}

// Parse 解析并验签核销凭证
// 返回订单号、过期时间；同时校验时效（已过期/未到生效均拒绝）
func Parse(token string) (*ParsedToken, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}
	// 必须以 LIMO| 开头，不接受纯订单号
	if !strings.HasPrefix(token, prefix+sep) {
		return nil, ErrInvalidToken
	}
	// 去掉前缀 LIMO|
	body := strings.TrimPrefix(token, prefix+sep)
	parts := strings.SplitN(body, sep, 3)
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	orderNo := parts[0]
	if err := ValidateOrderNo(orderNo); err != nil {
		return nil, ErrInvalidOrderNo
	}
	expireTs, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, ErrInvalidToken
	}
	sig := parts[2]
	// 重算签名比对
	expected, err := sign(orderNo, expireTs)
	if err != nil {
		return nil, err
	}
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, ErrSignatureMismatch
	}
	// 时效校验（fail-closed）
	now := time.Now().Unix()
	if expireTs < now {
		return nil, ErrTokenExpired
	}
	return &ParsedToken{OrderNo: orderNo, ExpireTs: expireTs}, nil
}
