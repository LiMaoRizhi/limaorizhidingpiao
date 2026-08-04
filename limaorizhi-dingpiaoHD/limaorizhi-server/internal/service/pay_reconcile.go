package service

import (
	"log"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// PayReconcileFunc 单个订单的对账处理函数（main.go 注入，避免 service 与 wx handler 循环依赖）
// 返回 handled=true 表示钱已经安排退款了（订单已取消+微信已扣款的情形）
type PayReconcileFunc func(db *gorm.DB, order model.Order) (bool, error)

var payReconcileFn PayReconcileFunc

// SetPayReconcileFunc 注入支付对账函数（main.go 启动时调用）
func SetPayReconcileFunc(fn PayReconcileFunc) {
	payReconcileFn = fn
}

// StartPayReconcileChecker 启动支付对账定时任务（每5分钟一次）
// 背景：支付回调可能没送达，订单卡"待支付/已取消"但微信钱已扣，
// 用户"付了款退不了款"。光靠用户点支付触发查单不够，存量卡死单得后台自动捞。
// 这里主动把"待支付/已取消"的订单拿去问微信：已支付就确认订单，已取消就登记退款，
// 退款补偿任务会接着把钱打回去。钱绝不凭空消失。
func StartPayReconcileChecker(db *gorm.DB) {
	startTickerTask("支付对账", 2*time.Minute, 5*time.Minute, reconcilePendingOrders, db)
}

// reconcilePendingOrders 扫描可能有"假状态"的订单拿去微信对账
// 只查待支付(0)和已取消(4)两种订单——只有这两种状态才可能是"钱扣了回调没到"的假状态，
// 已支付/已退款/已完成的订单状态是可信的，不用管。
// 限制扫描量（每次50单），避免对账任务把微信接口打爆。
func reconcilePendingOrders(db *gorm.DB) {
	if payReconcileFn == nil {
		log.Printf("支付对账任务未注入对账函数，跳过本次")
		return
	}

	var orders []model.Order
	// 已取消的订单全查；待支付的只查创建超过3分钟的（刚下单的用户可能正要支付，别打扰）
	cutoff := time.Now().Add(-3 * time.Minute)
	if err := db.Where(
		"status = ? OR (status = ? AND created_at < ?)",
		model.OrderStatusCancelled, model.OrderStatusPending, cutoff,
	).Limit(50).Find(&orders).Error; err != nil {
		log.Printf("支付对账查询订单失败: %v", err)
		return
	}
	if len(orders) == 0 {
		return
	}

	for _, order := range orders {
		if err := func(order model.Order) error {
			// 对账函数内部自己处理幂等（重复跑不会重复确认/重复退款）
			handled, err := payReconcileFn(db, order)
			if err != nil {
				return err
			}
			if handled {
				log.Printf("支付对账: 订单%s已取消但微信已扣款，已登记自动退款", order.OrderNo)
			}
			return nil
		}(order); err != nil {
			log.Printf("支付对账订单%s处理失败: %v", order.OrderNo, err)
		}
	}
	log.Printf("支付对账完成，本次检查 %d 个订单", len(orders))
}
