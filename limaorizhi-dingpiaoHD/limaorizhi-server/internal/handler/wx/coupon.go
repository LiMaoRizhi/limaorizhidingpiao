// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AvailableCoupons 获取可领取的优惠券列表（首页展示的优惠券）
func (h *UserHandler) AvailableCoupons(c *gin.Context) {
	// 从系统配置读取首页展示的优惠券ID
	var cfg model.SystemConfig
	if err := h.DB.Where("config_key = ?", "homepage_coupon_ids").First(&cfg).Error; err != nil {
		response.OK(c, []interface{}{})
		return
	}
	if cfg.ConfigValue == "" {
		response.OK(c, []interface{}{})
		return
	}

	// 解析配置中的优惠券ID列表（逗号分隔）
	var couponIDs []uint
	for _, idStr := range strings.Split(cfg.ConfigValue, ",") {
		idStr = strings.TrimSpace(idStr)
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil && id > 0 {
			couponIDs = append(couponIDs, uint(id))
		}
	}
	if len(couponIDs) == 0 {
		response.OK(c, []interface{}{})
		return
	}

	// 查询用户已领取的优惠券ID集合
	userID := c.GetUint("user_id")
	var claimedIDs []uint
	h.DB.Model(&model.UserCoupon{}).Where("user_id = ?", userID).Pluck("coupon_id", &claimedIDs)
	claimedSet := make(map[uint]bool)
	for _, id := range claimedIDs {
		claimedSet[id] = true
	}

	// 查询优惠券（仅查询配置中指定的优惠券ID）
	var coupons []model.Coupon
	h.DB.Where("id IN ? AND status = ?", couponIDs, model.CouponStatusEnable).Order("id DESC").Find(&coupons)

	type availableCoupon struct {
		ID            uint    `json:"id"`
		Name          string  `json:"name"`
		Type          int8    `json:"type"`
		DiscountValue float64 `json:"discount_value"`
		MinSpend      float64 `json:"min_spend"`
		ValidDays     int     `json:"valid_days"`
		IssuedCount   int     `json:"issued_count"`
		TotalCount    int     `json:"total_count"`
		Claimed       bool    `json:"claimed"` // 当前用户是否已领取
	}
	var list []availableCoupon
	for _, cp := range coupons {
		// 检查是否有余量
		if cp.TotalCount > 0 && cp.IssuedCount >= cp.TotalCount {
			continue
		}
		list = append(list, availableCoupon{
			ID:            cp.ID,
			Name:          cp.Name,
			Type:          cp.Type,
			DiscountValue: cp.DiscountValue,
			MinSpend:      cp.MinSpend,
			ValidDays:     cp.ValidDays,
			IssuedCount:   cp.IssuedCount,
			TotalCount:    cp.TotalCount,
			Claimed:       claimedSet[cp.ID],
		})
	}
	response.OK(c, list)
}

// claimCouponRequest 领取优惠券请求
type claimCouponRequest struct {
	CouponID uint `json:"coupon_id" binding:"required"`
}

// ClaimCoupon 用户领取优惠券
func (h *UserHandler) ClaimCoupon(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req claimCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 使用事务+行锁防止并发重复领取
	var userCoupon model.UserCoupon
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 加行锁查询优惠券
		var cp model.Coupon
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cp, req.CouponID).Error; err != nil {
			return fmt.Errorf("优惠券不存在")
		}
		if cp.Status != model.CouponStatusEnable {
			return fmt.Errorf("优惠券已停用")
		}
		// 检查库存
		if cp.TotalCount > 0 && cp.IssuedCount >= cp.TotalCount {
			return fmt.Errorf("优惠券已领完")
		}

		// 2. 检查是否已领取过（加锁防止并发绕过，唯一索引兜底）
		var count int64
		tx.Model(&model.UserCoupon{}).Where("user_id = ? AND coupon_id = ?", userID, req.CouponID).Count(&count)
		if count > 0 {
			return fmt.Errorf("您已领取过该优惠券")
		}

		// 3. 创建用户优惠券记录（依赖唯一索引防并发重复领取）
		now := time.Now()
		userCoupon = model.UserCoupon{
			UserID:    userID,
			CouponID:  req.CouponID,
			Status:    model.UserCouponStatusUnused, // 未使用
			IssuedAt:  model.JSONTime(now),
			ExpiredAt: model.JSONTime(now.AddDate(0, 0, cp.ValidDays)),
		}
		if err := tx.Create(&userCoupon).Error; err != nil {
			// 唯一索引冲突：并发重复领取
			if isDuplicateKeyError(err) {
				return fmt.Errorf("您已领取过该优惠券")
			}
			return fmt.Errorf("领取失败")
		}

		// 4. 更新优惠券已发放数量
		return tx.Model(&cp).UpdateColumn("issued_count", gorm.Expr("issued_count + 1")).Error
	})

	if err != nil {
		errMsg := err.Error()
		response.FailMsg(c, response.CodeParamError, errMsg)
		return
	}

	response.OKMsg(c, "领取成功", userCoupon)
}

// isDuplicateKeyError 判断是否为数据库唯一索引冲突
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return mysqlErr.Number == 1062
	}
	return strings.Contains(err.Error(), "Duplicate entry")
}
