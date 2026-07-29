// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/redis"
	"limaorizhi-server/internal/pkg/sanitize"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/triptime"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)
// OrderList 我的订单列表
func (h *UserHandler) OrderList(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := h.DB.Model(&model.Order{}).Where("user_id = ? AND user_hidden = ?", userID, false)
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if orderType := c.Query("order_type"); orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	query.Count(&total)

	var list []model.Order
	query.Preload("FromStation").Preload("ToStation").Preload("Trip").Preload("Trip.Driver").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	model.MaskOrders(list)

	response.Page(c, list, total, page, pageSize)
}

// OrderDetail 订单详情
func (h *UserHandler) OrderDetail(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID := c.Param("id")

	var order model.Order
	if err := h.DB.Preload("FromStation").Preload("ToStation").
		Preload("Trip.Route.FromStation").Preload("Trip.Route.ToStation").
		Preload("Trip.Vehicle").Preload("Trip.Driver").
		Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}

	var passengers []model.OrderPassenger
	h.DB.Where("order_id = ?", order.ID).Find(&passengers)
	model.MaskPassengers(passengers)
	order.Mask()

	// 检查是否有有效的车辆位置数据（班次已发车且5分钟内有上报，司机结束行程后位置记录被清除）
	hasLocation := false
	if order.Trip != nil && order.Trip.Status == model.TripStatusDepart {
		var locCount int64
		fiveMinAgo := time.Now().Add(-5 * time.Minute)
		h.DB.Model(&model.VehicleLocation{}).Where("trip_id = ? AND reported_at > ?", order.TripID, fiveMinAgo).Count(&locCount)
		hasLocation = locCount > 0
	}

	response.OK(c, gin.H{
		"order":       order,
		"passengers":  passengers,
		"has_location": hasLocation,
	})
}

// PayOrder 支付订单（调用微信统一下单API，前端通过wx.requestPayment完成支付）
func (h *UserHandler) PayOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID := c.Param("id")

	// 防并发下单，Redis不可用时降级为进程级锁
	payLockKey := fmt.Sprintf("pay:%d:%s", userID, orderID)
	redisLockKey := "lock:" + payLockKey
	acquired, fallback := redis.TryLock(redisLockKey, 30*time.Second)
	if !fallback {
		// Redis 模式
		if !acquired {
			response.FailMsg(c, response.CodeOrderStatusErr, "该订单正在处理支付，请勿重复提交")
			return
		}
		defer redis.Unlock(redisLockKey)
	} else {
		// 降级：进程级锁（单实例有效）
		if _, loaded := payLocks.LoadOrStore(payLockKey, struct{}{}); loaded {
			response.FailMsg(c, response.CodeOrderStatusErr, "该订单正在处理支付，请勿重复提交")
			return
		}
		defer payLocks.Delete(payLockKey)
	}

	// 先查询订单基本信息（不加锁，用于微信下单）
	var order model.Order
	if err := h.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}
	if order.Status != model.OrderStatusPending {
		response.FailMsg(c, response.CodeOrderStatusErr, "订单状态不允许支付")
		return
	}

	// 支付前校验班次状态和车辆是否过站
	var trip model.Trip
	if err := h.DB.First(&trip, order.TripID).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}
	// 校验班次状态：已下架(0)/已取消(3)/已完成(4)不允许支付
	if trip.Status == 0 || trip.Status == model.TripStatusCancel || trip.Status == model.TripStatusFinish {
		response.FailMsg(c, response.CodeTripUnavailable, "班次已取消或已结束，无法支付")
		return
	}
	if isStationPassed(h.DB, trip, order.FromStationID, nil) {
		response.FailMsg(c, response.CodeTripUnavailable, "班次车辆已驶过该上车站，无法下单/支付")
		return
	}

	// 微信支付：调用统一下单API，前端通过 wx.requestPayment 完成支付
	if !isWxPayConfigured() {
		response.FailMsg(c, response.CodePayFail, "微信支付未配置，请联系管理员")
		return
	}

	var user model.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}

	payCfg := getWxPayConfig()
	prepayID, err := createUnifiedOrder(payCfg, order, user.OpenID, c.ClientIP())
	if err != nil {
		log.Printf("[ERROR] 微信下单失败 订单号:%s err:%v\n", order.OrderNo, err)
		response.FailMsg(c, response.CodePayFail, "微信下单失败，请稍后重试")
		return
	}

	// 返回小程序支付参数，前端调用 wx.requestPayment（v3 为 RSA 签名，signType=RSA）
	payParams, err := buildJSAPIParams(payCfg, prepayID)
	if err != nil {
		log.Printf("[ERROR] 构建支付参数失败 订单号:%s err:%v\n", order.OrderNo, err)
		response.FailMsg(c, response.CodePayFail, "构建支付参数失败，请稍后重试")
		return
	}
	response.OK(c, gin.H{
		"payment_params": payParams,
		"order_id":       order.ID,
		"order_no":       order.OrderNo,
	})
}

// CancelOrder 取消订单（行锁防并发双取消+双回补座位）
func (h *UserHandler) CancelOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID := c.Param("id")

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 加行锁查询订单
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
			return errOrderNotFound
		}

		// 2. 检查订单状态（行锁内检查，防止并发）
		// 仅允许待支付订单取消；已支付订单需走退票流程退款
		if order.Status != model.OrderStatusPending {
			return errOnlyPendingCancel
		}

		// 3. 更新订单状态
		if err := tx.Model(&order).Update("status", model.OrderStatusCancelled).Error; err != nil {
			return err
		}
		// 4. 区间复用模型：座位容量按区间实时计算，取消订单后该区间容量自然恢复，无需回补 available_seats
		return nil
	})

	if err != nil {
		switch {
		case errors.Is(err, errOrderNotFound):
			response.Fail(c, response.CodeOrderNotFound)
		case errors.Is(err, errOnlyPendingCancel):
			response.FailMsg(c, response.CodeOrderStatusErr, err.Error())
		default:
			log.Printf("[ERROR] 取消订单失败 订单ID:%s err:%v\n", orderID, err)
			response.FailMsg(c, response.CodeServerError, "取消失败，请稍后重试")
		}
		return
	}

	response.OKMsg(c, "取消成功", nil)
}

// RefundOrder 申请退票（行锁防并发双退款+双回补座位）
type refundOrderRequest struct {
	Reason string `json:"reason"`
}

func (h *UserHandler) RefundOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID := c.Param("id")
	var req refundOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// reason为可选字段，客户端可能不传body，降级为空reason继续
		req.Reason = ""
	}

	// 读取退票配置（事务外读取，不涉及并发竞争）
	var beforeHoursCfg, feeRateCfg model.SystemConfig
	beforeHours := 2 // 默认发车前2小时可退票
	if err := h.DB.Where("config_key = ?", "refund_before_departure_hours").First(&beforeHoursCfg).Error; err == nil {
		if v, err := strconv.Atoi(beforeHoursCfg.ConfigValue); err == nil && v >= 0 {
			beforeHours = v
		}
	}
	feeRate := 0.0 // 默认手续费率0%
	if err := h.DB.Where("config_key = ?", "refund_fee_rate").First(&feeRateCfg).Error; err == nil {
		if v, err := strconv.ParseFloat(feeRateCfg.ConfigValue, 64); err == nil && v >= 0 {
			feeRate = v
		}
	}
	if feeRate > 100 {
		feeRate = 100
	}

	var order model.Order
	var refund model.Refund

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 加行锁查询订单
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
			return errOrderNotFound
		}

		// 2. 检查订单状态（行锁内检查）
		if order.Status != model.OrderStatusPaid {
			return errOnlyTravelRefund
		}

		// 3. 检查发车前可退票时间
		var trip model.Trip
		if err := tx.First(&trip, order.TripID).Error; err != nil {
			return errTripNotFound
		}
		departureTime, parseErr := triptime.Parse(string(trip.TripDate), trip.DepartureTime)
		if parseErr != nil {
			// fail-closed：发车时间解析失败时拒绝退票，防止构造异常时间绕过退票时限
			return fmt.Errorf("班次发车时间格式异常，无法退票")
		}
		deadline := departureTime.Add(-time.Duration(beforeHours) * time.Hour)
		if time.Now().After(deadline) {
			return fmt.Errorf("发车前%d小时内不可退票", beforeHours)
		}

		// 4. 计算退票手续费（统一用整数分运算，与全项目money包风格一致，避免浮点精度丢失）
		totalFen := money.ToFen(order.TotalPrice)
		refundFeeFen := int64(float64(totalFen) * feeRate / 100)
		refundAmount := money.FromFen(totalFen - refundFeeFen)

		// 5. 更新订单状态（车票→3已退款，托运→4已取消）
		refundStatus := int8(model.OrderStatusRefunded) // 车票退款后状态
		if order.OrderType == 2 {
			refundStatus = int8(model.OrderStatusCancelled) // 托运退款后状态（已取消）
		}
		if err := tx.Model(&order).Update("status", refundStatus).Error; err != nil {
			return err
		}
		// 6. 区间复用模型：座位容量按区间实时计算，退款后该区间容量自然恢复，无需回补 available_seats
		// 7. 创建退款记录
		refundNo := sanitize.GenerateRefundNo(order.ID)
		refund = model.Refund{
			OrderID:    order.ID,
			RefundNo:   refundNo,
			Amount:     refundAmount,
			Reason:     req.Reason,
			Status:     model.RefundStatusProcessing, // 处理中
			PreStatus:  model.OrderStatusPaid,        // 退款前订单状态（待出行/待运输）
		}
		return tx.Create(&refund).Error
	})

	if err != nil {
		errMsg := err.Error()
		switch {
		case errors.Is(err, errOrderNotFound):
			response.Fail(c, response.CodeOrderNotFound)
		case errors.Is(err, errOnlyTravelRefund):
			response.FailMsg(c, response.CodeOrderStatusErr, errMsg)
		case errors.Is(err, errTripNotFound):
			response.Fail(c, response.CodeTripNotFound)
		case strings.HasPrefix(errMsg, "发车前"):
			response.FailMsg(c, response.CodeOrderStatusErr, errMsg)
		default:
			response.FailMsg(c, response.CodeRefundFail, "退票失败: "+errMsg)
		}
		return
	}

	if err := InitiateWxRefund(h.DB, order, refund); err != nil {
		log.Printf("[ERROR] 退款失败 订单号:%s 退款单号:%s err:%v\n", order.OrderNo, refund.RefundNo, err)
		// 退款失败回滚订单状态和座位（用户退票回滚为待出行/待运输(1)）
		service.RollbackRefundFailure(h.DB, order, refund, model.OrderStatusPaid)
		response.FailMsg(c, response.CodeRefundFail, "退款失败，请稍后重试")
		return
	}
	// 退款成功时退款记录将在 RefundNotify 回调中更新为成功

	response.OKMsg(c, "退款申请已提交，退款将在1-3个工作日内到账", nil)
}

// OrderStats 订单统计（各状态数量）
func (h *UserHandler) OrderStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	type stat struct {
		Status int8  `json:"status"`
		Count  int64 `json:"count"`
	}
	var stats []stat
	h.DB.Model(&model.Order{}).
		Select("status, COUNT(*) as count").
		Where("user_id = ? AND user_hidden = ?", userID, false).
		Group("status").
		Scan(&stats)

	result := map[string]int64{
		"pending_pay":    0,
		"pending_travel": 0,
		"completed":      0,
		"refunded":       0,
		"cancelled":      0,
		"picked_up":      0,
	}
	for _, s := range stats {
		switch s.Status {
		case model.OrderStatusPending:
			result["pending_pay"] = s.Count
		case model.OrderStatusPaid:
			result["pending_travel"] = s.Count
		case model.OrderStatusCompleted:
			result["completed"] = s.Count
		case model.OrderStatusRefunded:
			result["refunded"] = s.Count
		case model.OrderStatusCancelled:
			result["cancelled"] = s.Count
		case 5:
			result["picked_up"] = s.Count
		}
	}
	response.OK(c, result)
}

// HideOrder 用户隐藏/删除订单（软删除：仅对用户端隐藏，管理端数据不受影响）
func (h *UserHandler) HideOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID := c.Param("id")

	result := h.DB.Model(&model.Order{}).
		Where("id = ? AND user_id = ?", orderID, userID).
		Update("user_hidden", true)
	if result.Error != nil {
		log.Printf("[ERROR] 隐藏订单失败 订单ID:%s err:%v\n", orderID, result.Error)
		response.FailMsg(c, response.CodeServerError, "删除失败，请稍后重试")
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}

	response.OKMsg(c, "删除成功", nil)
}

