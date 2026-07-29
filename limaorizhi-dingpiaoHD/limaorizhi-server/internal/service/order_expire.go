// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"fmt"
	"log"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// StartOrderExpireChecker 启动订单超时自动取消定时任务
// 每分钟扫描一次，将超过15分钟未支付的订单自动取消
// 区间复用模型：座位容量按区间实时计算，取消订单后区间容量自然恢复，无需回补 available_seats
// 5月16号刚上线那会儿没加条件更新，用户手动取消和定时任务同时跑，座位算了两遍
// 计算了两遍TMD费劲，GO屎玩意搞心态
func StartOrderExpireChecker(db *gorm.DB) {
	startTickerTask("订单超时取消", 30*time.Second, 1*time.Minute, cancelExpiredOrders, db)
}

// cancelExpiredOrders 取消超时未支付的订单
func cancelExpiredOrders(db *gorm.DB) {
	// 从系统配置读取超时分钟数，默认15分钟
	var config model.SystemConfig
	expireMinutes := 15
	if err := db.Where("config_key = ?", "order_expire_minutes").First(&config).Error; err == nil {
		if config.ConfigValue != "" {
			var parsed int
			if _, err := fmt.Sscanf(config.ConfigValue, "%d", &parsed); err == nil && parsed > 0 {
				expireMinutes = parsed
			}
		}
	}

	// 查找超时未支付的订单（status=0 且创建时间超过阈值）
	cutoff := time.Now().Add(-time.Duration(expireMinutes) * time.Minute)
	var expiredOrders []model.Order
	if err := db.Where("status = ? AND created_at < ?", model.OrderStatusPending, cutoff).Find(&expiredOrders).Error; err != nil {
		log.Printf("查询超时订单失败: %v", err)
		return
	}

	if len(expiredOrders) == 0 {
		return
	}

	cancelledCount := 0
	for _, order := range expiredOrders {
		err := db.Transaction(func(tx *gorm.DB) error {
			// 条件更新：只有 status=0 才取消，防止与用户手动取消/支付竞态导致双回补座位
			// 6月18号加的防护，之前没加这个出了好几次座位超卖
			result := tx.Model(&model.Order{}).Where("id = ? AND status = ?", order.ID, model.OrderStatusPending).Update("status", model.OrderStatusCancelled)
			if result.Error != nil {
				return result.Error
			}
			// RowsAffected=0 表示状态已被其他操作改变，跳过座位回补
			if result.RowsAffected == 0 {
				return nil
			}
			// 区间复用模型：座位容量按区间实时计算，取消订单后区间容量自然恢复，无需回补 available_seats
			return nil
		})
		if err != nil {
			log.Printf("自动取消订单 %s 失败: %v", order.OrderNo, err)
		} else {
			cancelledCount++
			log.Printf("自动取消超时订单: %s (创建于 %s)", order.OrderNo, order.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	if cancelledCount > 0 {
		log.Printf("本次自动取消 %d 个超时未支付订单", cancelledCount)
	}
}
