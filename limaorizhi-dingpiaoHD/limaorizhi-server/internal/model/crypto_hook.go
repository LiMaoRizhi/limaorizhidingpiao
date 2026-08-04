package model

import (
	"log"

	"limaorizhi-server/internal/pkg/crypto"

	"gorm.io/gorm"
)

// GORM Hook：身份证号自动加解密
// BeforeCreate/BeforeUpdate 加密，AfterFind 解密，兼容历史明文（无 ENC: 前缀）。
//
// ⚠ 注意：map[string]interface{} 方式的 Updates 不触发 BeforeUpdate hook！
//   如果需要在 map Updates 中修改 id_card_no，必须先调用 EncryptIDCard 显式加密：
//   updates["id_card_no"] = model.EncryptIDCard(rawIDCard)

// EncryptIDCard 导出的身份证号加密函数，供 handler 层在 map Updates 场景显式调用
func EncryptIDCard(value string) string {
	return encryptIDCard(value)
}

func encryptIDCard(value string) string {
	if value == "" {
		return value
	}
	enc, err := crypto.Encrypt(value)
	if err != nil {
		log.Printf("[WARN] 身份证号加密失败: %v\n", err)
		return value
	}
	return enc
}

func decryptIDCard(value string) string {
	if value == "" {
		return value
	}
	dec, err := crypto.Decrypt(value)
	if err != nil {
		log.Printf("[WARN] 身份证号解密失败: %v\n", err)
		return "" // 解密失败返回空串，防止密文通过脱敏函数泄露
	}
	return dec
}

// OrderPassenger

func (p *OrderPassenger) BeforeCreate(tx *gorm.DB) error {
	p.IDCardNo = encryptIDCard(p.IDCardNo)
	return nil
}

func (p *OrderPassenger) BeforeUpdate(tx *gorm.DB) error {
	p.IDCardNo = encryptIDCard(p.IDCardNo)
	return nil
}

func (p *OrderPassenger) AfterFind(tx *gorm.DB) error {
	p.IDCardNo = decryptIDCard(p.IDCardNo)
	return nil
}

// Passenger

func (p *Passenger) BeforeCreate(tx *gorm.DB) error {
	p.IDCardNo = encryptIDCard(p.IDCardNo)
	return nil
}

func (p *Passenger) BeforeUpdate(tx *gorm.DB) error {
	p.IDCardNo = encryptIDCard(p.IDCardNo)
	return nil
}

func (p *Passenger) AfterFind(tx *gorm.DB) error {
	p.IDCardNo = decryptIDCard(p.IDCardNo)
	return nil
}
