package service

import (
	"fmt"
	"log"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RollbackRefundFailure 退款失败后的统一回滚逻辑（用户端退票与管理端退款共用）。
// 事务外调用，使用条件更新保证幂等。
// 6月18号补的，之前退款失败没回滚订单状态，有人退票失败钱扣了订单还是已退款状态，被投诉了
func RollbackRefundFailure(db *gorm.DB, order model.Order, refund model.Refund, preRefundStatus int8) {
	// 1. 标记退款记录失败（条件更新：仅当仍为处理中(0)时才标记，防重复）
	// 关键：只有成功将退款标记为失败，才允许回滚订单。
	// 如果退款已被回调标记为成功(status=1)，则说明微信已退款到账，
	// 绝不能回滚订单，否则用户已收退款又可用票，造成双重损失。
	refundRes := db.Model(&model.Refund{}).
		Where("id = ? AND status = ?", refund.ID, model.RefundStatusProcessing).
		Update("status", model.RefundStatusFailed)
	if refundRes.Error != nil {
		log.Printf("[ERROR] 退款失败回滚：标记退款记录失败 退款单号:%s err:%v\n", refund.RefundNo, refundRes.Error)
		return
	}
	if refundRes.RowsAffected == 0 {
		// 退款记录已非处理中：可能已被回调标记为成功，或已被并发回滚标记为失败。
		// 无论哪种情况，都不应回滚订单状态，需查询当前退款状态以确认。
		var latestRefund model.Refund
		if err := db.First(&latestRefund, refund.ID).Error; err == nil {
			if latestRefund.Status == model.RefundStatusSuccess {
				log.Printf("[WARN] 退款失败回滚跳过：退款已被微信回调标记为成功，不回滚订单 退款单号:%s 订单号:%s\n", refund.RefundNo, order.OrderNo)
				return
			}
			log.Printf("[INFO] 退款失败回滚跳过：退款记录已为状态%d，不回滚订单 退款单号:%s\n", latestRefund.Status, refund.RefundNo)
		}
		return
	}

	// 2. 回滚订单状态（条件更新：仅当当前为退款后状态时回滚）
	refundStatus := int8(model.OrderStatusRefunded) // 车票退款后状态
	if order.OrderType == 2 {
		refundStatus = int8(model.OrderStatusCancelled) // 托运退款后状态
	}
	res := db.Model(&model.Order{}).
		Where("id = ? AND status = ?", order.ID, refundStatus).
		Update("status", preRefundStatus)
	if res.Error != nil {
		log.Printf("[ERROR] 退款失败回滚：订单状态回滚失败 订单号:%s err:%v\n", order.OrderNo, res.Error)
	} else if res.RowsAffected == 0 {
		log.Printf("[WARN] 退款失败回滚：订单状态已非已退款，跳过回滚 订单号:%s\n", order.OrderNo)
	}

	// 3. 区间复用模型：座位容量按区间实时计算，退款失败回滚订单状态后
	//    订单重新参与区间容量计算，无需回补 available_seats。
	log.Printf("[INFO] 退款失败回滚完成 订单号:%s 退款单号:%s 订单状态已回滚为%d\n", order.OrderNo, refund.RefundNo, preRefundStatus)
}

// MarkRefundSuccess 未配置微信退款证书时直接标记退款成功（开发环境降级）。
// 事务外调用，使用条件更新保证幂等。
// 标记成功后发送退款到账订阅消息通知（生产环境由 RefundNotify 回调发送，此处补齐开发模式路径）。
func MarkRefundSuccess(db *gorm.DB, refund model.Refund) error {
	now := time.Now()
	result := db.Model(&model.Refund{}).
		Where("id = ? AND status = ?", refund.ID, model.RefundStatusProcessing).
		Updates(map[string]interface{}{"status": model.RefundStatusSuccess, "refund_time": &now})
	if result.Error != nil {
		log.Printf("[ERROR] MarkRefundSuccess 更新退款记录失败 退款单号:%s err:%v\n", refund.RefundNo, result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 退款记录已非处理中（已被并发处理），幂等跳过，不重复发送通知
		return nil
	}
	log.Printf("[INFO] 退款成功（未配置证书，跳过微信退款API） 退款单号:%s 金额:%.2f\n", refund.RefundNo, refund.Amount)

	// 同步更新支付记录状态为已退款，保证财务数据一致性
	UpdatePaymentStatusRefunded(db, refund.OrderID)

	// 发送退款到账通知（异步，不阻塞调用方）
	// 生产环境由 RefundNotify 回调触发 NotifyRefundSuccess，此处仅覆盖开发模式/未配置证书的降级路径
	var order model.Order
	if err := db.First(&order, refund.OrderID).Error; err == nil {
		NotifyRefundSuccess(db, refund, order)
	} else {
		log.Printf("[WARN] MarkRefundSuccess: 查询订单失败，跳过退款通知 退款单号:%s orderID=%d err:%v\n", refund.RefundNo, refund.OrderID, err)
	}
	return nil
}

// UpdatePaymentStatusRefunded 更新支付记录状态为已退款（条件更新：仅成功的支付记录才标记为已退款）
// 可在事务内或事务外调用，幂等安全
func UpdatePaymentStatusRefunded(db *gorm.DB, orderID uint) {
	if err := db.Model(&model.Payment{}).
		Where("order_id = ? AND status = ?", orderID, model.PaymentStatusSuccess).
		Update("status", model.PaymentStatusRefunded).Error; err != nil {
		log.Printf("[WARN] 更新支付记录为已退款失败 orderID=%d err:%v\n", orderID, err)
	}
}

// RefundRetryFunc 退款重试函数类型（由 handler/wx 注入，避免 service→wx 循环依赖）
type RefundRetryFunc func(refund model.Refund, transactionID string) error

var refundRetryFn RefundRetryFunc

// SetRefundRetryFunc 注入退款重试函数（main.go 启动时调用，避免循环依赖）
func SetRefundRetryFunc(fn RefundRetryFunc) {
	refundRetryFn = fn
}

// RefundQueryFunc 退款状态查询函数类型（由 handler/wx 注入）
// 返回微信侧退款状态：SUCCESS（已到账）/PROCESSING（处理中）/CLOSED/ABNORMAL
type RefundQueryFunc func(refundNo string) (string, error)

var refundQueryFn RefundQueryFunc

// SetRefundQueryFunc 注入退款状态查询函数（main.go 启动时调用）
func SetRefundQueryFunc(fn RefundQueryFunc) {
	refundQueryFn = fn
}

// StartRefundCompensator 退款补偿定时任务：扫描 status=0 超时的退款记录重试退款API
// 兜底场景：PayNotify 创建退款记录后、调用微信退款API前进程崩溃，微信重试回调已耗尽或未覆盖
// 5月16号就写了，后来6月18号又调了几次，微信重试回调跟定时任务打架重复退款
func StartRefundCompensator(db *gorm.DB) {
	startTickerTask("退款补偿", 2*time.Minute, 5*time.Minute, compensateRefunds, db)
}

// compensateRefunds 扫描处理中超时的退款记录，重试退款API
func compensateRefunds(db *gorm.DB) {
	if refundRetryFn == nil {
		return // 未注入退款函数（开发环境未配置微信退款），跳过
	}
	// 查询 status=0 且创建超过5分钟的退款记录（5分钟窗口避免与微信重试回调竞争）
	var refunds []model.Refund
	threshold := time.Now().Add(-5 * time.Minute)
	if err := db.Where("status = ? AND created_at < ?", model.RefundStatusProcessing, threshold).Find(&refunds).Error; err != nil {
		log.Printf("[退款补偿] 查询失败: %v", err)
		return
	}
	if len(refunds) == 0 {
		return
	}
	for _, refund := range refunds {
		// 行锁 + 状态校验，防止多实例并发重试同一笔退款
		var locked model.Refund
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", refund.ID, model.RefundStatusProcessing).First(&locked).Error; err != nil {
				return err // 记录不存在或已非处理中，跳过
			}
			return nil
		})
		if err != nil || locked.ID == 0 {
			continue // 已被其他实例处理或状态已变更
		}

		// 找对应订单
		var order model.Order
		if err := db.First(&order, locked.OrderID).Error; err != nil {
			log.Printf("[退款补偿] 查询订单失败 退款单号:%s: %v", locked.RefundNo, err)
			continue
		}
		// 查 transaction_id（微信流水号，退款API离不开它）
		// 必须过滤 status=1（成功），避免取到失败/待支付记录的空 transaction_id
		var payment model.Payment
		var txnID string
		if err := db.Where("order_id = ? AND status = ?", order.ID, model.PaymentStatusSuccess).Order("id DESC").First(&payment).Error; err == nil {
			txnID = payment.TransactionID
		}
		if txnID == "" {
			log.Printf("[退款补偿] 无效transaction_id 退款单号:%s，跳过", locked.RefundNo)
			continue
		}
		// 对账兜底：先主动问微信这笔退款到底退没退，别傻等退款回调。
		// 背景：退款回调（refund/notify）可能没送达（比如 APIv3Key 配错导致解密失败），
		// 微信侧其实已经退款到账，但本地退款记录卡"处理中"，用户看不到钱到账。
		// 已成功 → 直接标记本地退款成功（钱都退了，绝不能再发起第二次退款）；
		// 处理中 → 微信还在处理，跳过等下次；
		// 查不到/查询失败 → 走下面的重试发起退款。
		if refundQueryFn != nil {
			st, qerr := refundQueryFn(locked.RefundNo)
			if qerr == nil {
				switch st {
				case "SUCCESS":
					if err := MarkRefundSuccess(db, locked); err != nil {
						log.Printf("[退款补偿] 对账标记退款成功失败 退款单号:%s err:%v", locked.RefundNo, err)
					} else {
						log.Printf("[退款补偿] 对账: 退款单%s微信侧已到账，本地已同步为已退款", locked.RefundNo)
					}
					continue
				case "PROCESSING":
					log.Printf("[退款补偿] 对账: 退款单%s微信侧处理中，跳过本次", locked.RefundNo)
					continue
				default:
					log.Printf("[退款补偿] 对账: 退款单%s微信侧状态=%s，走重试发起退款", locked.RefundNo, st)
				}
			} else {
				log.Printf("[退款补偿] 对账查退款状态失败 退款单号:%s err:%v", locked.RefundNo, qerr)
			}
		}
		if err := refundRetryFn(locked, txnID); err != nil {
			log.Printf("[退款补偿] 重试失败 退款单号:%s err:%v", locked.RefundNo, err)
			// 写告警日志，便于运维监控和人工补偿
			db.Create(&model.OperationLog{
				AdminName: "系统自动退款",
				Module:    "退款告警",
				Action:    "退款补偿重试失败",
				Target:    order.OrderNo,
				Detail:    fmt.Sprintf("退款单号:%s 退款API重试失败:%v", locked.RefundNo, err),
			})
		} else {
			log.Printf("[退款补偿] 重试已发起 退款单号:%s", locked.RefundNo)
		}
	}
}
