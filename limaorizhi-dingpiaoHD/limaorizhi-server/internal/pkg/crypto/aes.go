// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

// 加过密的数据搁前边加个ENC:，没前缀的是老明文数据，直接透传
const encPrefix = "ENC:"

var encryptionKey []byte

// InitKey 初始化密钥，启动时整一次
// key是base64编码的32字节（AES-256）
// 不配的话Encrypt会报错，Decrypt透传原文（开发环境能跑）
func InitKey(key string) error {
	if key == "" {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return errors.New("身份证加密密钥必须为 base64 编码")
	}
	if len(decoded) != 32 {
		return errors.New("身份证加密密钥必须为 32 字节(AES-256)，请用 base64 编码后配置")
	}
	encryptionKey = decoded
	return nil
}

// Enabled 配没配密钥
func Enabled() bool {
	return len(encryptionKey) > 0
}

// Encrypt 加密
// 开发环境没配密钥就透传原文，不报错
// 生产环境main.go里强制校验密钥配了
// 返回ENC: + base64(nonce+密文)，解密时靠前缀判断
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	// 已经是加密格式就不重复加密了
	if strings.HasPrefix(plaintext, encPrefix) {
		return plaintext, nil
	}
	if !Enabled() {
		// 没配密钥，透传原文，别挡着写入
		return plaintext, nil
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密
// 没ENC:前缀的是历史明文，直接返回
// 解密失败返回空字符串，防止密文通过脱敏函数泄露为明文
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !strings.HasPrefix(ciphertext, encPrefix) {
		// 没前缀，老明文，直接返回
		return ciphertext, nil
	}
	if !Enabled() {
		// 有加密数据但没配密钥，返回空串防止密文泄露
		return "", errors.New("有加密数据但密钥没配")
	}
	raw := strings.TrimPrefix(ciphertext, encPrefix)
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文长度不足")
	}
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		// 解密失败返回空串，避免密文被当作明文输出导致脱敏失效
		return "", err
	}
	return string(plaintext), nil
}

// GenerateKey 生成新密钥，运维用的，别搁运行时调
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
