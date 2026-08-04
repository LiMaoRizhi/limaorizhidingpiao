package service

import (
	"fmt"
	"log"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/sanitize"

	"gorm.io/gorm"
)

// PrepareRefundRecord 创建或复用退款记录（防双重退款）
// refundType：0=整单退款，1=改签差价退款。防双重检查仅匹配同类型记录，
// 避免改签差价退款记录阻塞后续整单退票（或反之）。
// 若订单已存在同类型处理中/成功的退款记录 → 返回错误（调用方应跳过）；
// 若存在同类型已失败(status=2)的退款记录 → 复用其 refund_no 重置为处理中并更新金额/原因
//   （微信退款 API 按 refund_no 幂等，复用单号不会造成重复退款）
// 否则创建新退款记录。
// 返回 (退款记录, 是否新建, error)。事务内调用。
func PrepareRefundRecord(tx *gorm.DB, orderID uint, amount float64, reason string, preStatus int8, refundType int8) (model.Refund, bool, error) {
	// 已有同类型处理中/成功 → 跳过（防止并发创建重复记录）
	var existing model.Refund
	if err := tx.Where("order_id = ? AND status IN (?, ?) AND refund_type = ?", orderID, model.RefundStatusProcessing, model.RefundStatusSuccess, refundType).First(&existing).Error; err == nil {
		return model.Refund{}, false, fmt.Errorf("订单已有处理中或成功的退款记录，跳过")
	}
	// 已失败 → 复用 refund_no 重试
	var failed model.Refund
	if err := tx.Where("order_id = ? AND status = ? AND refund_type = ?", orderID, model.RefundStatusFailed, refundType).First(&failed).Error; err == nil {
		if err := tx.Model(&failed).Where("id = ? AND status = ?", failed.ID, model.RefundStatusFailed).
			Updates(map[string]interface{}{"status": model.RefundStatusProcessing, "amount": amount, "reason": reason}).Error; err != nil {
			return model.Refund{}, false, err
		}
		failed.Status = model.RefundStatusProcessing
		failed.Amount = amount
		failed.Reason = reason
		log.Printf("[INFO] 复用失败退款单号重试 订单ID:%d 退款单号:%s\n", orderID, failed.RefundNo)
		return failed, false, nil
	}
	// 无已有记录 → 创建新退款记录
	refund := model.Refund{
		OrderID:    orderID,
		RefundNo:   sanitize.GenerateRefundNo(orderID),
		Amount:     amount,
		Reason:     reason,
		Status:     model.RefundStatusProcessing,
		PreStatus:  preStatus,
		RefundType: refundType,
	}
	if err := tx.Create(&refund).Error; err != nil {
		return model.Refund{}, false, err
	}
	return refund, true, nil
}

// PrepareChangeRefundRecord 创建或复用改签差价退款记录（refund_type=1）
// 与整单退款不同：多次改签可产生多笔差价退款，因此不拦截已有成功记录；
// 仅复用已失败(status=2)的差价退款单号重试（微信按 refund_no 幂等，复用单号不会重复退款）。
// 并发安全由改签事务中的订单行锁保证（同一订单改签串行执行，不会并发创建重复记录）。
// 返回 (退款记录, 是否新建, error)。事务内调用。
func PrepareChangeRefundRecord(tx *gorm.DB, orderID uint, amount float64) (model.Refund, bool, error) {
	// 已失败 → 复用 refund_no 重试
	var failed model.Refund
	if err := tx.Where("order_id = ? AND status = ? AND refund_type = ?", orderID, model.RefundStatusFailed, 1).First(&failed).Error; err == nil {
		if err := tx.Model(&failed).Where("id = ? AND status = ?", failed.ID, model.RefundStatusFailed).
			Updates(map[string]interface{}{"status": model.RefundStatusProcessing, "amount": amount, "reason": "改签差价退款"}).Error; err != nil {
			return model.Refund{}, false, err
		}
		failed.Status = model.RefundStatusProcessing
		failed.Amount = amount
		failed.Reason = "改签差价退款"
		log.Printf("[INFO] 复用改签差价失败退款单号重试 订单ID:%d 退款单号:%s\n", orderID, failed.RefundNo)
		return failed, false, nil
	}
	// 无失败记录 → 创建新差价退款记录
	refund := model.Refund{
		OrderID:    orderID,
		RefundNo:   sanitize.GenerateRefundNo(orderID),
		Amount:     amount,
		Reason:     "改签差价退款",
		Status:     model.RefundStatusProcessing,
		PreStatus:  model.OrderStatusPaid,
		RefundType: 1,
	}
	if err := tx.Create(&refund).Error; err != nil {
		return model.Refund{}, false, err
	}
	return refund, true, nil
}

// MarkCouponUsed 下单时标记优惠券已使用（事务内调用，与下单同事务保证原子性）
// 返回受影响行数；0=券状态已变化（并发使用/过期），调用方应视为使用失败
func MarkCouponUsed(tx *gorm.DB, userCouponID uint, orderID uint, userID uint) (int64, error) {
	now := time.Now()
	result := tx.Model(&model.UserCoupon{}).
		Where("id = ? AND user_id = ? AND status = ?", userCouponID, userID, model.UserCouponStatusUnused).
		Updates(map[string]interface{}{
			"status":   model.UserCouponStatusUsed,
			"used_at":  now,
			"order_id": orderID,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected > 0 {
		// 模板使用次数+1（营销统计）
		tx.Model(&model.Coupon{}).Where("id = (SELECT coupon_id FROM user_coupons WHERE id = ?)", userCouponID).
			UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	}
	return result.RowsAffected, nil
}

// ReturnOrderCoupon 归还订单绑定的优惠券（订单取消/超时/退款时调用，事务内）
// 条件更新：仅当券仍标记为已使用(order_id匹配且status=1)时才归还为未使用，
// 防止与并发操作（如券已过期任务改为已过期）冲突。
// 归还同时递减优惠券模板 used_count（营销统计），并记录日志便于排查。
func ReturnOrderCoupon(tx *gorm.DB, orderID uint) error {
	var userCoupon model.UserCoupon
	if err := tx.Where("order_id = ? AND status = ?", orderID, model.UserCouponStatusUsed).First(&userCoupon).Error; err != nil {
		return nil // 没有使用中的券绑定该订单，无需归还
	}
	result := tx.Model(&model.UserCoupon{}).
		Where("id = ? AND status = ? AND order_id = ?", userCoupon.ID, model.UserCouponStatusUsed, orderID).
		Updates(map[string]interface{}{
			"status":   model.UserCouponStatusUnused,
			"used_at":  nil,
			"order_id": 0,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("[INFO] 订单取消/退款归还优惠券 userCouponID=%d orderID=%d\n", userCoupon.ID, orderID)
		if err := tx.Model(&model.Coupon{}).Where("id = ?", userCoupon.CouponID).
			UpdateColumn("used_count", gorm.Expr("GREATEST(used_count - 1, 0)")).Error; err != nil {
			log.Printf("[WARN] 归还优惠券递减used_count失败 couponID=%d err:%v\n", userCoupon.CouponID, err)
		}
	}
	return nil
}
