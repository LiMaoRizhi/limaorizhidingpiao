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
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/triptime"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChangeOrder 改签：同线路换班次（上下车站、乘客、联系方式保持不变，仅更换班次/日期/时间）
//
// 规则：
//   - 仅已支付(1)的车票订单可改签，且原班次在退改时限内（发车前 refund_before_departure_hours 小时）
//   - 新班次必须与原班次同线路，上下车站不变，且未驶过上车站
//   - 座位类型保持：有座订单改签后仍为有座（新班次余座充足时）；无座订单改签后仍为无座（不允许升舱为有座，避免产生未付差价）
//   - 有座订单改签时新班次余座不足，且新班次开放无座票 → 自动降级为无座（与下单兜底逻辑一致），按无座折扣价计价
//   - 价格只降不升：改签后新票价（含优惠券抵扣与保险费）高于原订单实付时拒绝改签，提示退票后重新购买
//   - 降价差价自动原路退回（复用退款记录 + 微信退款，防双重退款）
//   - 改签后原班次座位自动释放（订单 trip_id 变更后，原班次座位占用查询不再包含该订单）
type changeOrderRequest struct {
	TripID uint `json:"trip_id" binding:"required"` // 目标班次ID
}

func (h *UserHandler) ChangeOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	orderID := c.Param("id")
	var req changeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 退改时限配置（与退票共用 refund_before_departure_hours）
	var beforeHoursCfg model.SystemConfig
	beforeHours := 2
	if err := h.DB.Where("config_key = ?", "refund_before_departure_hours").First(&beforeHoursCfg).Error; err == nil {
		if v, err := strconv.Atoi(beforeHoursCfg.ConfigValue); err == nil && v >= 0 {
			beforeHours = v
		}
	}

	var order model.Order
	var refund model.Refund
	var hasRefund bool
	var diffAmount float64
	var originalTotalPrice float64 // 改签前订单实付金额（微信退款API的 amount.total 必须为原交易金额）

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 行锁订单，防止与并发支付/退款/重复改签竞争
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
			return errOrderNotFound
		}
		if order.OrderType != 1 {
			return fmt.Errorf("仅车票订单可改签")
		}
		if order.Status != model.OrderStatusPaid {
			return fmt.Errorf("仅待出行订单可改签，已支付订单可申请退票")
		}
		originalTotalPrice = order.TotalPrice
		if order.TripID == req.TripID {
			return fmt.Errorf("不能改签到当前班次")
		}

		// 2. 行锁原班次与新班次（按班次ID升序加锁，与下单事务锁序一致，避免并发死锁）
		lockIDs := []uint{order.TripID, req.TripID}
		if lockIDs[0] > lockIDs[1] {
			lockIDs[0], lockIDs[1] = lockIDs[1], lockIDs[0]
		}
		var t1, t2 model.Trip
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&t1, lockIDs[0]).Error; err != nil {
			return errTripNotFound
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&t2, lockIDs[1]).Error; err != nil {
			return errTripNotFound
		}
		var oldTrip, newTrip model.Trip
		if t1.ID == order.TripID {
			oldTrip, newTrip = t1, t2
		} else {
			oldTrip, newTrip = t2, t1
		}

		// 3. 原班次校验：可售/已发车 且未驶过上车站，且在退改时限内
		if oldTrip.Status != model.TripStatusSale && oldTrip.Status != model.TripStatusDepart {
			return fmt.Errorf("原班次已取消或已结束，无法改签")
		}
		if isStationPassed(tx, oldTrip, order.FromStationID, nil) {
			return fmt.Errorf("原班次车辆已驶过上车站，无法改签")
		}
		oldDeparture, parseErr := triptime.Parse(string(oldTrip.TripDate), oldTrip.DepartureTime)
		if parseErr != nil {
			return fmt.Errorf("原班次发车时间格式异常，无法改签")
		}
		deadline := oldDeparture.Add(-time.Duration(beforeHours) * time.Hour)
		if time.Now().After(deadline) {
			return fmt.Errorf("发车前%d小时内不可改签", beforeHours)
		}

		// 4. 新班次校验：同线路、可售/已发车、未驶过上车站
		if newTrip.RouteID != order.RouteID {
			return fmt.Errorf("仅支持同线路改签")
		}
		if newTrip.Status != model.TripStatusSale && newTrip.Status != model.TripStatusDepart {
			return fmt.Errorf("目标班次不可售或已取消")
		}
		if isStationPassed(tx, newTrip, order.FromStationID, nil) {
			return fmt.Errorf("目标班次车辆已驶过上车站")
		}

		// 5. 查询线路站点序列，校验上下车站 + 计算区间票价（与下单完全一致）
		var routeStations []model.RouteStation
		if err := tx.Where("route_id = ?", newTrip.RouteID).Order("stop_order ASC").Find(&routeStations).Error; err != nil || len(routeStations) < 2 {
			return fmt.Errorf("线路站点配置不完整")
		}
		var fromOrder, toOrder int = -1, -1
		var fromPrice, toPrice float64
		for _, rs := range routeStations {
			if rs.StationID == order.FromStationID {
				fromOrder = rs.StopOrder
				fromPrice = rs.Price
			}
			if rs.StationID == order.ToStationID {
				toOrder = rs.StopOrder
				toPrice = rs.Price
			}
		}
		if fromOrder < 0 || toOrder < 0 || fromOrder >= toOrder || toPrice < fromPrice {
			return fmt.Errorf("线路站点配置异常，无法改签")
		}
		var route model.Route
		if err := tx.Select("min_fare").First(&route, newTrip.RouteID).Error; err != nil {
			return fmt.Errorf("线路信息不存在")
		}
		// 区间座位票价（同线路同区间在原班次与新班次中一致，作为原票面价计算基准）
		seatedFare := money.Sub(toPrice, fromPrice)
		if route.MinFare > 0 && seatedFare < route.MinFare {
			seatedFare = route.MinFare
		}

		// 6. 查询订单乘客，确定座位类型与人数
		var passengers []model.OrderPassenger
		if err := tx.Where("order_id = ?", order.ID).Find(&passengers).Error; err != nil {
			return fmt.Errorf("查询乘客信息失败")
		}
		if len(passengers) == 0 {
			return fmt.Errorf("订单乘客信息缺失")
		}
		count := len(passengers)
		isStanding := passengers[0].SeatType == 1

		// 6.5 目标班次乘客身份证重复校验（与下单一致：同一班次同一身份证不可重复购票）
		// 订单改签后 trip_id 变更，若目标班次已存在同一乘客的票，会导致一人两票
		var existingPassengers []model.OrderPassenger
		tx.Joins("JOIN orders ON orders.id = order_passengers.order_id").
			Where("orders.trip_id = ? AND orders.status IN (?, ?)", newTrip.ID, model.OrderStatusPending, model.OrderStatusPaid).
			Find(&existingPassengers)
		if len(existingPassengers) > 0 {
			for _, p := range passengers {
				for _, ep := range existingPassengers {
					if p.IDCardNo == ep.IDCardNo {
						return fmt.Errorf("乘客%s的身份证号已在目标班次购票", p.Name)
					}
				}
			}
		}

		// 7. 新班次座位检查与分配
		newSeatType := int8(0)
		newSeatNos := make([]string, count)
		if isStanding {
			// 无座→无座（不允许升舱为有座，避免产生未付差价）
			if !newTrip.AllowStanding || newTrip.StandingQuota <= 0 {
				return fmt.Errorf("目标班次未开放无座票，无法改签")
			}
			_, standingAvail := service.RealtimeStandingStats(tx, newTrip.ID, newTrip.StandingQuota)
			if standingAvail < count {
				return fmt.Errorf("目标班次无座票余量不足")
			}
			newSeatType = 1
		} else {
			avail, err := service.AvailableSeatsForSegment(tx, newTrip.ID, newTrip.TotalSeats, fromOrder, toOrder)
			if err != nil {
				return err
			}
			if avail >= count {
				seatNos, _, assignErr := service.AssignSeatsWithPreferences(tx, newTrip.ID, newTrip.TotalSeats, fromOrder, toOrder, count, nil)
				if assignErr != nil {
					return errNoSeat
				}
				newSeatNos = seatNos
			} else if newTrip.AllowStanding && newTrip.StandingQuota > 0 {
				// 余座不足：新班次开放无座 → 自动降级为无座（与下单兜底逻辑一致）
				_, standingAvail := service.RealtimeStandingStats(tx, newTrip.ID, newTrip.StandingQuota)
				if standingAvail < count {
					return errNoSeat
				}
				newSeatType = 1
			} else {
				return errNoSeat
			}
		}

		// 8. 计算新票价（无座按班次折扣，整数分运算）
		newFare := seatedFare
		if newSeatType == 1 && newTrip.StandingDiscount > 0 && newTrip.StandingDiscount < 1 {
			newFare = money.FromFen(int64(float64(money.ToFen(seatedFare))*newTrip.StandingDiscount + 0.5))
		}
		// 原票面价 = 区间座位票价 × 人数（同线路同区间票价一致）
		// 无座原订单的实际票面按原班次无座折扣价计算，否则推算优惠券抵扣会虚高、差价计算失真
		originalFace := money.Mul(seatedFare, count)
		if isStanding && oldTrip.StandingDiscount > 0 && oldTrip.StandingDiscount < 1 {
			originalFace = money.FromFen(int64(float64(money.ToFen(originalFace))*oldTrip.StandingDiscount + 0.5))
		}
		// 原车票实付（不含保险）= 订单总价 - 保险费（含优惠券抵扣）
		originalTicketPay := money.Sub(order.TotalPrice, order.InsuranceFee)
		// 优惠券抵扣额（含原班次无座折扣差异，座位类型相同时即为优惠券抵扣）
		couponDiscount := money.Sub(originalFace, originalTicketPay)
		if couponDiscount < 0 {
			couponDiscount = 0
		}
		// 新车票实付 = 新票面价 - 优惠券抵扣（抵扣不超过新票面价）
		newFaceTotal := money.Mul(newFare, count)
		newTicketPay := money.Sub(newFaceTotal, couponDiscount)
		if newTicketPay < 0 {
			newTicketPay = 0
		}
		newTotal := money.Add(newTicketPay, order.InsuranceFee)
		// 价格只降不升：新价高于原价 → 拒绝改签
		if newTotal > order.TotalPrice {
			return fmt.Errorf("目标班次票价高于原票，无法改签，请退票后重新购买")
		}
		diffAmount = money.Sub(order.TotalPrice, newTotal)

		// 9. 更新订单（班次、日期、时间、总价）
		orderUpdates := map[string]interface{}{
			"trip_id":        newTrip.ID,
			"trip_date":      newTrip.TripDate,
			"departure_time": newTrip.DepartureTime,
			"total_price":    newTotal,
		}
		if err := tx.Model(&order).Updates(orderUpdates).Error; err != nil {
			return fmt.Errorf("更新订单失败")
		}
		order.TripID = newTrip.ID
		order.TripDate = newTrip.TripDate
		order.DepartureTime = newTrip.DepartureTime
		order.TotalPrice = newTotal

		// 10. 更新乘客座位信息（无座为空座位号 + seat_type=1）
		for i := range passengers {
			p := &passengers[i]
			if err := tx.Model(p).Select("seat_no", "seat_type").
				Updates(map[string]interface{}{"seat_no": newSeatNos[i], "seat_type": newSeatType}).Error; err != nil {
				return fmt.Errorf("更新乘客座位失败")
			}
			p.SeatNo = newSeatNos[i]
			p.SeatType = newSeatType
		}

		// 11. 降价差价退款：创建差价退款记录（支持多次改签各退一笔），事务外发起微信退款
		if diffAmount > 0.01 {
			newRefund, _, prepareErr := service.PrepareChangeRefundRecord(tx, order.ID, diffAmount)
			if prepareErr != nil {
				return prepareErr
			}
			refund = newRefund
			hasRefund = true
		}
		return nil
	})

	if err != nil {
		switch {
		case errors.Is(err, errOrderNotFound):
			response.Fail(c, response.CodeOrderNotFound)
		case errors.Is(err, errNoSeat):
			response.Fail(c, response.CodeNoSeat)
		case errors.Is(err, errTripNotFound):
			response.Fail(c, response.CodeTripNotFound)
		case strings.HasPrefix(err.Error(), "发车前"):
			response.FailMsg(c, response.CodeOrderStatusErr, err.Error())
		default:
			response.FailMsg(c, response.CodeOrderStatusErr, "改签失败: "+err.Error())
		}
		return
	}

	// 事务外：差价退款发起微信退款（失败时标记退款失败并记录，改签已生效需客服介入）
	refundOK := false
	if hasRefund {
		// 关键：微信退款API的 amount.total 必须等于原支付交易金额，
		// 而 order.TotalPrice 已被更新为改签后新价，必须用改签前实付金额发起退款，
		// 否则微信返回 REFUND_FEE_MISMATCH（订单金额与之前请求不一致）拒绝退款。
		refundOrder := order
		refundOrder.TotalPrice = originalTotalPrice
		if rerr := InitiateWxRefund(h.DB, refundOrder, refund); rerr != nil {
			log.Printf("[ERROR] 改签差价退款失败 订单号:%s 退款单号:%s err:%v\n", order.OrderNo, refund.RefundNo, rerr)
			h.DB.Model(&model.Refund{}).Where("id = ? AND status = ?", refund.ID, model.RefundStatusProcessing).
				Update("status", model.RefundStatusFailed)
		} else {
			refundOK = true
			log.Printf("[INFO] 改签差价退款已发起 订单号:%s 退款单号:%s 金额:%.2f\n", order.OrderNo, refund.RefundNo, refund.Amount)
		}
	}

	// 重新加载订单与乘客返回
	if err := h.DB.Preload("FromStation").Preload("ToStation").
		Preload("Trip.Route.FromStation").Preload("Trip.Route.ToStation").
		Preload("Trip.Vehicle").Preload("Trip.Driver").
		First(&order, order.ID).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "改签成功但加载订单失败")
		return
	}
	var passengers []model.OrderPassenger
	h.DB.Where("order_id = ?", order.ID).Find(&passengers)
	model.MaskPassengers(passengers)
	order.Mask()

	msg := "改签成功"
	if hasRefund && refundOK {
		msg = fmt.Sprintf("改签成功，差价¥%.2f已原路退回", diffAmount)
	} else if hasRefund {
		msg = "改签成功，但差价退款暂未完成，请稍后在订单中查看或联系客服"
	}
	response.OKMsg(c, msg, gin.H{
		"order":         order,
		"passengers":    passengers,
		"changed":       true,
		"refund_amount": diffAmount,
		"refund_no":     refund.RefundNo,
	})
}
