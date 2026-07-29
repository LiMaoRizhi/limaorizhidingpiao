// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"log"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// StartCouponExpireChecker 启动优惠券过期自动标记定时任务
// 每小时扫描一次，将已过期但状态仍为"未使用"(status=0)的优惠券标记为"已过期"(status=2)
func StartCouponExpireChecker(db *gorm.DB) {
	startTickerTask("优惠券过期", 60*time.Second, 1*time.Hour, expireCoupons, db)
}

// expireCoupons 将已过期但状态仍为未使用的优惠券标记为已过期
func expireCoupons(db *gorm.DB) {
	now := time.Now()
	// 条件更新：只有 status=0 且已过期才更新为 status=2
	// 防止与用户使用优惠券的并发操作冲突
	result := db.Model(&model.UserCoupon{}).
		Where("status = ? AND expired_at <= ?", model.UserCouponStatusUnused, now).
		Update("status", model.UserCouponStatusExpired)
	if result.Error != nil {
		log.Printf("优惠券过期标记失败: %v", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		log.Printf("本次自动标记 %d 个已过期优惠券", result.RowsAffected)
	}
}
