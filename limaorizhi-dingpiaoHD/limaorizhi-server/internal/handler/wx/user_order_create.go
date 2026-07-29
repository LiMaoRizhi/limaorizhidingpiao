// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/idcard"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)
// 订单相关错误（sentinel error，避免字符串比较与切片越界panic）

var (
	errNoSeat             = errors.New("余座不足")
	errTripUnavailable   = errors.New("班次已发车或已取消")
	errTripDeparted      = errors.New("班次已过发车时间")
	errOrderNotFound     = errors.New("订单不存在")
	errTripNotFound      = errors.New("班次不存在")
	errOnlyPendingCancel = errors.New("仅待支付订单可取消，已支付订单请申请退票")
	errOnlyTravelRefund  = errors.New("只有待出行/待运输订单可退票")
	errInvalidPhone      = errors.New("手机号格式不正确")
)

// payLocks 同一订单同时只允许一个支付请求，避免重复下单
var payLocks sync.Map

// 订单管理

// isValidPhone 校验中国大陆手机号格式（1开头，11位数字）
func isValidPhone(phone string) bool {
	if len(phone) != 11 {
		return false
	}
	if phone[0] != '1' {
		return false
	}
	for i := 1; i < len(phone); i++ {
		if phone[i] < '0' || phone[i] > '9' {
			return false
		}
	}
	return true
}

// CreateOrder 创建订单（含座位锁定）
type createOrderPassenger struct {
	PassengerID uint   `json:"passenger_id"` // 非零时从常用乘客表取完整信息，前端无需传明文证件号
	Name        string `json:"name"`            // passenger_id 为 0 时必填
	IDCardType  int8   `json:"id_card_type"`
	IDCardNo    string `json:"id_card_no"`         // passenger_id 为 0 时必填
	Phone       string `json:"phone"`
}

type createOrderRequest struct {
	TripID        string                 `json:"trip_id" binding:"required"`
	FromStationID uint                   `json:"from_station_id" binding:"required"` // 上车站
	ToStationID   uint                   `json:"to_station_id" binding:"required"`   // 下车站
	Passengers    []createOrderPassenger `json:"passengers" binding:"required"`
	ContactName   string                 `json:"contact_name" binding:"required"`
	ContactPhone  string                 `json:"contact_phone" binding:"required"`
	CouponID      uint                   `json:"coupon_id"` // 可选，优惠券ID
}

func (h *UserHandler) CreateOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	passengerCount := len(req.Passengers)
	if passengerCount == 0 {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 限制单次下单乘客数，防止恶意大量下单
	if passengerCount > 10 {
		response.FailMsg(c, response.CodeParamError, "单次最多购买10张票")
		return
	}
	// 限制未支付订单数量，防刷单占座（放在事务内加锁校验，避免并发绕过）
	// 此处仅做事务外快速预检，减少无效事务开销；事务内会再次校验
	var pendingCount int64
	h.DB.Model(&model.Order{}).Where("user_id = ? AND status = ?", userID, model.OrderStatusPending).Count(&pendingCount)
	if pendingCount >= 5 {
		response.FailMsg(c, response.CodeServerError, "您有过多未支付订单，请先支付或取消后再下单")
		return
	}
	// 校验联系人手机号格式
	if !isValidPhone(req.ContactPhone) {
		response.FailMsg(c, response.CodeParamError, "联系人手机号格式不正确")
		return
	}

	// 身份验证：下单前校验每位乘客的身份证（格式校验 + 实名认证）
	// 如果传了 passenger_id，从常用乘客表取完整信息，避免前端传输明文证件号
	for i := range req.Passengers {
		p := &req.Passengers[i]
		if p.PassengerID != 0 {
			var saved model.Passenger
			if err := h.DB.Where("id = ? AND user_id = ?", p.PassengerID, userID).First(&saved).Error; err != nil {
				response.FailMsg(c, response.CodeParamError, fmt.Sprintf("第%d位乘客: 常用乘客不存在", i+1))
				return
			}
			p.Name = saved.Name
			p.IDCardType = saved.IDCardType
			p.IDCardNo = saved.IDCardNo
			p.Phone = saved.Phone
		}
		if p.IDCardNo == "" {
			response.FailMsg(c, response.CodeParamError, fmt.Sprintf("第%d位乘客: 身份证号不能为空", i+1))
			return
		}
		if p.Name == "" {
			response.FailMsg(c, response.CodeParamError, fmt.Sprintf("第%d位乘客: 姓名不能为空", i+1))
			return
		}
		// 第一层：本地格式校验
		if err := idcard.ValidateFormat(p.IDCardNo); err != nil {
			response.FailMsg(c, response.CodeIDCardInvalid, fmt.Sprintf("第%d位乘客: %s", i+1, err.Error()))
			return
		}
		// 第二层：云市场实名认证（姓名+身份证号比对）
		result, err := h.Verifier.Verify(p.Name, p.IDCardNo)
		if err != nil {
			if h.Verifier.IsStrictMode() {
				// 严格模式：认证API失败时拒绝下单，防止降级绕过实名认证
				response.FailMsg(c, response.CodeVerifyServiceErr, fmt.Sprintf("第%d位乘客实名认证服务异常，请稍后重试", i+1))
				return
			}
			// 非严格模式：降级策略，仅记录日志
			log.Printf("[WARN] 实名认证服务异常: %v\n", err)
		} else if !result.Matched {
			response.FailMsg(c, response.CodeIDCardNotMatch, fmt.Sprintf("第%d位乘客姓名与身份证号不匹配", i+1))
			return
		}
	}

	// 事务处理：锁座 + 创建订单 + 创建乘客
	var order model.Order
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 查询班次（加行锁防止并发超卖）
		// 必须是事务内第一条语句：MySQL REPEATABLE READ 下，第一条普通 SELECT
		// 会建立一致性快照。若 Count 在 FOR UPDATE 之前执行，快照会在获取
		// 行锁之前建立，导致后续 AvailableSeatsForSegment/AssignSeats 看不到
		// 并发事务已提交的 order_passengers 插入，造成座位重复分配（超卖）。
		var trip model.Trip
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&trip, req.TripID).Error; err != nil {
			return errTripNotFound
		}

		// 0. 事务内再次校验未支付订单数量（在行锁之后执行，确保快照已更新）
		var pendingCountTx int64
		if err := tx.Model(&model.Order{}).Where("user_id = ? AND status = ?", userID, model.OrderStatusPending).Count(&pendingCountTx).Error; err != nil {
			return fmt.Errorf("查询未支付订单数量失败")
		}
		if pendingCountTx >= 5 {
			return fmt.Errorf("您有过多未支付订单，请先支付或取消后再下单")
		}
		// 允许已发车(status=2)的班次下单：途经站乘客在车到站前仍可购票
		// isStationPassed 会判断车是否已驶过用户上车站
		if trip.Status != model.TripStatusSale && trip.Status != model.TripStatusDepart {
			return errTripUnavailable
		}
		// 检查车是否已驶过用户上车站（多源融合判断，发车后未到站可下单）
		if isStationPassed(tx, trip, req.FromStationID, nil) {
			return errTripDeparted
		}

		// 2. 查询线路站点序列，校验上下车站合法性 + 区间座位容量（区间复用模型）
		var routeStations []model.RouteStation
		if err := tx.Preload("Station").Where("route_id = ?", trip.RouteID).Order("stop_order ASC").Find(&routeStations).Error; err != nil {
			return fmt.Errorf("线路站点序列不存在")
		}
		if len(routeStations) < 2 {
			return fmt.Errorf("线路站点配置不完整")
		}
		var fromOrder, toOrder int = -1, -1
		var fromPrice, toPrice float64
		var fromStationName, toStationName string
		for _, rs := range routeStations {
			if rs.StationID == req.FromStationID {
				fromOrder = rs.StopOrder
				fromPrice = rs.Price
				if rs.Station != nil {
					fromStationName = rs.Station.Name
				}
			}
			if rs.StationID == req.ToStationID {
				toOrder = rs.StopOrder
				toPrice = rs.Price
				if rs.Station != nil {
					toStationName = rs.Station.Name
				}
			}
		}
		if fromOrder < 0 || toOrder < 0 {
			return fmt.Errorf("上下车站不在该线路中")
		}
		if fromOrder >= toOrder {
			return fmt.Errorf("上车站必须在下车站之前")
		}
		if toPrice < fromPrice {
			return fmt.Errorf("线路票价配置异常，请联系管理员")
		}
		// 查询线路起步价（最低票价）
		var route model.Route
		if err := tx.Select("min_fare").First(&route, trip.RouteID).Error; err != nil {
			return fmt.Errorf("线路信息不存在")
		}
		// 实际票价 = max(起步价, 下车站price-上车站price)，起步价为0时不启用
		// 用整数分运算避免浮点误差
		fare := money.Sub(toPrice, fromPrice)
		if route.MinFare > 0 && fare < route.MinFare {
			fare = route.MinFare
		}
		// 同一订单内同一乘客（身份证号）不可重复添加（防构造请求绕过）
		seenIDCards := make(map[string]bool)
		for _, p := range req.Passengers {
			if seenIDCards[p.IDCardNo] {
				return fmt.Errorf("乘客%s的身份证号重复添加", p.Name)
			}
			seenIDCards[p.IDCardNo] = true
		}
		// 同一班次同一乘客（身份证号）不可重复购票
		// AES-GCM随机nonce导致无法用WHERE等值查询，改为加载到内存比对明文
		var existingPassengers []model.OrderPassenger
		tx.Joins("JOIN orders ON orders.id = order_passengers.order_id").
			Where("orders.trip_id = ? AND orders.status IN (?, ?)", trip.ID, model.OrderStatusPending, model.OrderStatusPaid).
			Find(&existingPassengers)
		for _, p := range req.Passengers {
			for _, ep := range existingPassengers {
				if p.IDCardNo == ep.IDCardNo {
					return fmt.Errorf("乘客%s的身份证号已在本班次购票", p.Name)
				}
			}
		}
		// 区间座位容量（同一座位在不同区段可复用）
		avail, err := service.AvailableSeatsForSegment(tx, trip.ID, trip.TotalSeats, fromOrder, toOrder)
		if err != nil {
			return err
		}
		if avail < passengerCount {
			return errNoSeat
		}

		// 分配具体座位号（区间复用模型：同一座位在不同不重叠区间可复用）
		seatNos, err := service.AssignSeats(tx, trip.ID, trip.TotalSeats, fromOrder, toOrder, passengerCount)
		if err != nil {
			return errNoSeat
		}

		// 3. 生成订单号（加密随机后缀，防止枚举攻击）
		randomBytes := make([]byte, 4)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("生成订单号失败")
		}
		orderNo := fmt.Sprintf("DP%s%s", time.Now().Format("20060102"), hex.EncodeToString(randomBytes))

		// 4. 创建订单
		// JSONDate.Scan 已确保格式为 "2006-01-02"，无需再手动剥离 RFC3339
		order = model.Order{
			OrderNo:        orderNo,
			OrderType:      1, // 车票
			UserID:         userID,
			TripID:         trip.ID,
			RouteID:        trip.RouteID,
			FromStationID:   req.FromStationID,
			FromStationName: fromStationName,
			ToStationID:     req.ToStationID,
			ToStationName:   toStationName,
			TripDate:        trip.TripDate,
			DepartureTime:  trip.DepartureTime,
			PassengerCount: passengerCount,
			TotalPrice:     money.Mul(fare, passengerCount),
			Status:         model.OrderStatusPending, // 待支付
			ContactName:    req.ContactName,
			ContactPhone:   req.ContactPhone,
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		// 5. 创建乘客记录（含座位号分配）
		for i, p := range req.Passengers {
			idCardType := p.IDCardType
			if idCardType == 0 {
				idCardType = 1
			}
			passenger := model.OrderPassenger{
				OrderID:    order.ID,
				Name:       p.Name,
				IDCardType: idCardType,
				IDCardNo:   p.IDCardNo,
				Phone:      p.Phone,
				SeatNo:     seatNos[i], // 分配的座位号
			}
			if err := tx.Create(&passenger).Error; err != nil {
				return err
			}
		}

		// 6. 处理优惠券（可选）
		if req.CouponID != 0 {
			var userCoupon model.UserCoupon
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ? AND status = ?", req.CouponID, userID, model.UserCouponStatusUnused).First(&userCoupon).Error; err != nil {
				return fmt.Errorf("优惠券不存在或已使用")
			}
			if time.Now().After(time.Time(userCoupon.ExpiredAt)) {
				return fmt.Errorf("优惠券已过期")
			}
			var coupon model.Coupon
			if err := tx.First(&coupon, userCoupon.CouponID).Error; err != nil {
				return fmt.Errorf("优惠券信息不存在")
			}
			if coupon.MinSpend > 0 && order.TotalPrice < coupon.MinSpend {
				return fmt.Errorf("订单金额不满足优惠券使用条件")
			}
			var discount float64
			switch coupon.Type {
			case 1: // 满减券
				discount = coupon.DiscountValue
			case 2: // 折扣券
				discount = money.Discount(order.TotalPrice, coupon.DiscountValue)
			case 3: // 固定金额券
				discount = coupon.DiscountValue
			}
			if discount > order.TotalPrice {
				discount = order.TotalPrice
			}
			order.TotalPrice = money.Sub(order.TotalPrice, discount)
			if err := tx.Model(&order).Update("total_price", order.TotalPrice).Error; err != nil {
				return err
			}
			if err := tx.Model(&userCoupon).Updates(map[string]interface{}{
				"status":   model.UserCouponStatusUsed,
				"used_at":  time.Now(),
				"order_id": order.ID,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		switch {
		case errors.Is(err, errNoSeat):
			response.Fail(c, response.CodeNoSeat)
		case errors.Is(err, errTripUnavailable), errors.Is(err, errTripDeparted):
			response.Fail(c, response.CodeTripUnavailable)
		default:
			log.Printf("[ERROR] 下单失败 err:%v\n", err)
			response.FailMsg(c, response.CodeServerError, "下单失败，请稍后重试")
		}
		return
	}

	// 重新加载完整订单信息
	if err := h.DB.Preload("Trip.Route.FromStation").Preload("Trip.Route.ToStation").
		Preload("FromStation").Preload("ToStation").First(&order, order.ID).Error; err != nil {
		log.Printf("[ERROR] 下单后重新加载订单失败 order_id=%d err:%v\n", order.ID, err)
		response.FailMsg(c, response.CodeServerError, "下单成功但加载订单详情失败")
		return
	}

	var passengers []model.OrderPassenger
	if err := h.DB.Where("order_id = ?", order.ID).Find(&passengers).Error; err != nil {
		log.Printf("[ERROR] 下单后加载乘客列表失败 order_id=%d err:%v\n", order.ID, err)
	}
	model.MaskPassengers(passengers)
	order.Mask()

	response.OKMsg(c, "下单成功", gin.H{
		"order":      order,
		"passengers": passengers,
	})
}

