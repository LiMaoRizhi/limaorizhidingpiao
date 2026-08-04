package wx

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/redis"
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

	// 批量标记含无座站票乘客的订单（has_standing，供前端显示无座标识）
	standingSet := map[uint]bool{}
	var orderIDs []uint
	for _, o := range list {
		orderIDs = append(orderIDs, o.ID)
	}
	if len(orderIDs) > 0 {
		var standingOrderIDs []uint
		h.DB.Model(&model.OrderPassenger{}).
			Where("order_id IN ? AND seat_type = 1", orderIDs).
			Distinct("order_id").Pluck("order_id", &standingOrderIDs)
		for _, id := range standingOrderIDs {
			standingSet[id] = true
		}
	}
	for i := range list {
		list[i].HasStanding = standingSet[list[i].ID]
	}

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

	// 含无座站票乘客标记（供前端显示无座标识）
	for _, p := range passengers {
		if p.SeatType == 1 {
			order.HasStanding = true
			break
		}
	}

	// 检查是否有有效的车辆位置数据（班次已发车且5分钟内有上报，司机结束行程后位置记录被清除）
	hasLocation := false
	if order.Trip != nil && order.Trip.Status == model.TripStatusDepart {
		var locCount int64
		fiveMinAgo := time.Now().Add(-5 * time.Minute)
		h.DB.Model(&model.VehicleLocation{}).Where("trip_id = ? AND reported_at > ?", order.TripID, fiveMinAgo).Count(&locCount)
		hasLocation = locCount > 0
	}

	// 支付流水：让用户能看到"啥时候付的、微信交易单号、付了多少钱"，
	// 别再出"钱扣了订单里啥也看不出来"这种鬼情况
	payInfo := gin.H{"paid": false}
	var payment model.Payment
	if err := h.DB.Where("order_id = ? AND status = ?", order.ID, model.PaymentStatusSuccess).Order("id DESC").First(&payment).Error; err == nil {
		payInfo = gin.H{
			"paid":           true,
			"transaction_id": payment.TransactionID,
			"amount":         payment.Amount,
			"method":         payment.Method,
			"pay_time":       payment.PayTime,
		}
	}

	response.OK(c, gin.H{
		"order":        order,
		"passengers":   passengers,
		"has_location": hasLocation,
		"payment":      payInfo,
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
		// 降级：进程级锁（单实例有效，30秒自动释放防死锁）
		if _, loaded := payLocks.LoadOrStore(payLockKey, struct{}{}); loaded {
			response.FailMsg(c, response.CodeOrderStatusErr, "该订单正在处理支付，请勿重复提交")
			return
		}
		// 加超时保护：30秒后自动释放，防止 panic/defer 不执行导致死锁
		lockTimer := time.AfterFunc(30*time.Second, func() {
			payLocks.Delete(payLockKey)
		})
		defer func() {
			lockTimer.Stop()
			payLocks.Delete(payLockKey)
		}()
	}

	// 先查询订单基本信息（不加锁，用于微信下单）
	var order model.Order
	if err := h.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}

	// 查单兜底（必须放在状态校验之前！）：
	// 支付回调可能没送达（网络抖/配置问题），用户"钱扣了订单还待支付"，
	// 甚至订单被超时任务自动取消了——但钱在微信扣了，不能装看不见。
	// 先主动问微信这单到底付没付：
	//   已支付 + 待支付 → 直接确认订单（用户正常出行）
	//   已支付 + 已取消 → 登记退款（钱必须找回来，补偿任务自动退）
	//   已支付 + 其他终态 → 提示无需重复支付
	// 不然用户点"去支付"直接被"订单状态不允许支付"挡掉，钱永远卡死，退都没法退。
	if client, cerr := getV3Client(); cerr == nil {
		qr, qerr := client.QueryOrder(context.Background(), order.OrderNo)
		if qerr == nil && qr != nil && qr.TradeState == "SUCCESS" {
			actualPaid := money.FromFen(int64(qr.Amount.Total))
			switch order.Status {
			case model.OrderStatusPending:
				// 金额再对一遍：微信实付必须跟订单金额一致才认，防篡改
				if qr.Amount.Total != int(money.ToFen(order.TotalPrice)) {
					log.Printf("[WARN] 查单兜底金额不匹配 订单号:%s 期望%d分 实际%d分\n",
						order.OrderNo, int(money.ToFen(order.TotalPrice)), qr.Amount.Total)
					response.FailMsg(c, response.CodeServerError, "支付金额异常，请联系客服核对")
					return
				}
				txErr := h.DB.Transaction(func(tx *gorm.DB) error {
					return confirmOrderPaid(tx, &order, qr.TransactionID, "微信支付")
				})
				if txErr != nil {
					log.Printf("[ERROR] 查单兜底确认支付失败 订单号:%s err:%v\n", order.OrderNo, txErr)
					response.FailMsg(c, response.CodeServerError, "订单已支付但确认失败，请稍后刷新查看")
					return
				}
				log.Printf("[INFO] 查单兜底: 订单%s微信侧已支付，已确认订单状态\n", order.OrderNo)
				response.OKMsg(c, "订单已支付", gin.H{
					"order_id": order.ID,
					"order_no": order.OrderNo,
					"paid":     true,
				})
				return
			case model.OrderStatusCancelled:
				// 致命场景：钱扣了订单却被自动取消，用户退不了款。
				// 登记退款让补偿任务把钱退回去，绝不让人白白损失。
				if _, rerr := registerPaidCancelledRefund(h.DB, order, qr.TransactionID, actualPaid); rerr != nil {
					log.Printf("[ERROR] 查单兜底登记退款失败 订单号:%s err:%v\n", order.OrderNo, rerr)
					response.FailMsg(c, response.CodeServerError, "系统处理中，请稍后刷新查看")
					return
				}
				log.Printf("[INFO] 查单兜底: 订单%s已取消但微信侧已支付，已登记自动退款(%.2f元)\n", order.OrderNo, actualPaid)
				response.OKMsg(c, "订单已支付但已被取消，系统将自动退款", gin.H{
					"order_id":  order.ID,
					"order_no":  order.OrderNo,
					"refunding": true,
				})
				return
			case model.OrderStatusRefunded, model.OrderStatusCompleted:
				response.FailMsg(c, response.CodeOrderStatusErr, "订单已支付，无需重复支付")
				return
			}
		}
		if qerr != nil {
			// 查单失败（网络抖/配置缺）不阻断正常下单，记个日志
			log.Printf("[WARN] 查单兜底查询失败 订单号:%s err:%v\n", order.OrderNo, qerr)
		}
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
		// 4. 归还该订单绑定的优惠券（否则用户未乘车却永久损失券）
		if err := service.ReturnOrderCoupon(tx, order.ID); err != nil {
			return err
		}
		// 5. 区间复用模型：座位容量按区间实时计算，取消订单后该区间容量自然恢复，无需回补 available_seats
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

	// 退款前先查单对账：支付回调没送达时订单可能显示"待支付/已取消"，
	// 但微信钱已扣，直接退会被状态校验挡掉（用户钱就卡死了）。
	// 这里先把真实状态救回来：已支付就确认、已取消就登记自动退款。
	if err := h.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}
	if handled, rerr := ReconcileOrderPaidState(h.DB, order); rerr != nil {
		response.FailMsg(c, response.CodeRefundFail, "退票失败: "+rerr.Error())
		return
	} else if handled {
		response.OKMsg(c, "订单已取消，系统将自动退款", nil)
		return
	}

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

		// 4. 计算退票手续费（纯整数分运算，避免 float64 乘除导致 ±1 分误差）
		// 与管理端退款共用 money.CalcRefundAmount（统一四舍五入），保证对账一致
		refundAmount := money.CalcRefundAmount(order.TotalPrice, feeRate)

		// 5. 更新订单状态（车票→3已退款，托运→4已取消）
		refundStatus := int8(model.OrderStatusRefunded) // 车票退款后状态
		if order.OrderType == 2 {
			refundStatus = int8(model.OrderStatusCancelled) // 托运退款后状态（已取消）
		}
		if err := tx.Model(&order).Update("status", refundStatus).Error; err != nil {
			return err
		}
		// 6. 区间复用模型：座位容量按区间实时计算，退款后该区间容量自然恢复，无需回补 available_seats
		// 7. 创建/复用退款记录（复用已失败退款单号，防双重退款）
		// 注意：err 由同语句 `err := h.DB.Transaction(...)` 声明，作用域不覆盖闭包内，故用局部变量接收
		newRefund, _, prepareErr := service.PrepareRefundRecord(tx, order.ID, refundAmount, req.Reason, model.OrderStatusPaid, 0)
		if prepareErr != nil {
			return prepareErr
		}
		refund = newRefund
		// 全退后归还该订单绑定的优惠券（用户未乘车，券不应被消耗）
		return service.ReturnOrderCoupon(tx, order.ID)
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

