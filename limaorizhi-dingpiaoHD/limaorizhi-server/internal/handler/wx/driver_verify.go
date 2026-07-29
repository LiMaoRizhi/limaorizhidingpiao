// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/jwt"
	"limaorizhi-server/internal/pkg/redis"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/verifytoken"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)
// 核销相关错误（sentinel error）
var (
	errVerifyOrderNotFound  = errors.New("订单不存在")
	errVerifyOrderStatus    = errors.New("订单未支付或已使用")
	errVerifyTripMismatch    = errors.New("该票不属于此班次")
	errVerifyForbidden       = errors.New("无权核销此班次或班次日期不匹配")
	errVerifyAlreadyChecked  = errors.New("该票已核销")
	errVerifyTripNotDeparted = errors.New("班次尚未发车，不可核销")
	errVerifyTripNotOwned    = errors.New("此订单不属于您负责的班次")
)

// manualVerifyFallback 手动核销频率限制的进程级降级方案
// 当 Redis 不可用时，使用 sync.Map 在进程内存中计数，防止单实例完全无限制
// 多实例部署时此降级方案仅对单实例生效，建议生产环境配置 Redis
var (
	manualVerifyCounts sync.Map // key: "driverID:hourBucket", value: int64
	manualVerifyLimit   int64 = 5 // 每小时未验签核销上限
)

// writeDriverLog 写入司机核销审计日志到 operation_log 表
// 复用管理员日志表，AdminID 字段借用为 driver_id，AdminName 字段借用为司机名
func writeDriverLog(c *gin.Context, db *gorm.DB, driverID uint, driverName, action, target, detail string) {
	nameStr := driverName
	if nameStr == "" {
		nameStr = fmt.Sprintf("driver#%d", driverID)
	}
	logEntry := model.OperationLog{
		AdminID:   driverID, // 借用字段，存储 driver_id
		AdminName: nameStr,
		Module:    "司机核销",
		Action:    action,
		Target:    target,
		Detail:    detail,
		IPAddress: c.ClientIP(),
	}
	if err := db.Create(&logEntry).Error; err != nil {
		log.Printf("写入司机核销日志失败: %v", err)
	}
}

// 司机端：登录+核销

type DriverHandler struct{ DB *gorm.DB }

func NewDriverHandler(db *gorm.DB) *DriverHandler { return &DriverHandler{DB: db} }

// Login 司机登录
type driverLoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *DriverHandler) Login(c *gin.Context) {
	var req driverLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 登录失败5次锁定15分钟
	ip := c.ClientIP()
	if redis.IsLoginLockedWithFallback(ip, req.Phone) {
		response.FailMsg(c, response.CodeForbidden, "登录失败次数过多，请15分钟后再试")
		return
	}

	var driver model.Driver
	if err := h.DB.Where("phone = ?", req.Phone).First(&driver).Error; err != nil {
		redis.RecordLoginFailWithFallback(ip, req.Phone, 15*time.Minute)
		response.FailMsg(c, response.CodeDriverNotFound, "手机号或密码错误")
		return
	}

	if driver.Status != model.DriverStatusEnable {
		redis.RecordLoginFailWithFallback(ip, req.Phone, 15*time.Minute)
		response.FailMsg(c, response.CodeDriverNotFound, "手机号或密码错误")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(driver.PasswordHash), []byte(req.Password)); err != nil {
		redis.RecordLoginFailWithFallback(ip, req.Phone, 15*time.Minute)
		response.FailMsg(c, response.CodeDriverNotFound, "手机号或密码错误")
		return
	}

	// 登录成功，清除失败计数
	redis.ClearLoginFailWithFallback(ip, req.Phone)

	// 司机端使用独立JWT密钥
	driverSecret := config.AppConfig.JWT.DriverSecret
	if driverSecret == "" {
		response.FailMsg(c, response.CodeServerError, "司机端JWT密钥未配置(driver_secret)，请联系管理员")
		return
	}
	token, err := jwt.GenerateToken(
		driverSecret,
		driver.ID,
		driver.Name,
		0,
		"driver",
		config.AppConfig.JWT.DriverExpire,
	)
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}

	// 更新最后登录时间
	now := time.Now()
	h.DB.Model(&driver).Update("last_login_at", &now)

	// 同步司机手机号到当前小程序用户，建立司机-用户持久关联
	// 司机在核销页登录时若同时带了小程序用户token(X-User-Token)，把 Driver.Phone 写入 User.Phone
	// 这样"我的"页 is_driver 判断（User.Phone 匹配 Driver.Phone）能正确识别司机，稳定显示核销入口
	// 仅当司机的微信绑定手机号与后台录入的不一致时才需此兜底；一致时本就是 is_driver=true
	userToken := c.GetHeader("X-User-Token")
	if userToken != "" {
		if claims, err := jwt.ParseToken(config.AppConfig.JWT.WXSecret, userToken); err == nil && claims.Type == "wx" && claims.UserID > 0 {
			// 仅在用户手机号为空或与司机手机号不一致时更新，避免无谓写入
			var u model.User
			if err := h.DB.First(&u, claims.UserID).Error; err == nil {
				if u.Phone != driver.Phone {
					h.DB.Model(&u).Update("phone", driver.Phone)
				}
			}
		}
	}

	response.OK(c, gin.H{
		"token": token,
		"driver": gin.H{
			"id":          driver.ID,
			"name":        driver.Name,
			"phone":       driver.Phone,
			"employee_no": driver.EmployeeNo,
		},
	})
}

// Trips 司机班次列表（今日全部班次 + 所有仍在运行的跨天班次）
// 跨天班次（如昨日/前日发车今日到达）在到达日仍需司机管理（标记过站/结束行程），
// 因此查询条件为：今日所有班次 OR 任何日期且状态为已发车(2)的班次
// status=2的班次一定未结束行程，不会出现已完成的旧班次
func (h *DriverHandler) Trips(c *gin.Context) {
	driverID := c.GetUint("driver_id")
	today := time.Now().Format("2006-01-02")

	var trips []model.Trip
	h.DB.Preload("Route.FromStation").
		Preload("Route.ToStation").
		Preload("Route.RouteStations.Station").
		Preload("Vehicle").
		Where("driver_id = ? AND (trip_date = ? OR status = ?)", driverID, today, model.TripStatusDepart).
		Order("departure_time ASC").
		Find(&trips)

	response.OK(c, trips)
}

// TripPassengers 班次乘客名单（含核销状态）
func (h *DriverHandler) TripPassengers(c *gin.Context) {
	tripID := c.Param("id")
	driverID := c.GetUint("driver_id")

	// 验证班次属于该司机
	var trip model.Trip
	if err := h.DB.Preload("Route.FromStation").
		Preload("Route.ToStation").
		Preload("Route.RouteStations.Station").
		Where("id = ? AND driver_id = ?", tripID, driverID).First(&trip).Error; err != nil {
		response.FailMsg(c, response.CodeForbidden, "无权查看此班次")
		return
	}

	var passengers []model.OrderPassenger
	h.DB.Joins("JOIN orders ON orders.id = order_passengers.order_id").
		Preload("Order.FromStation").
		Preload("Order.ToStation").
		Where("orders.trip_id = ? AND orders.status = ?", tripID, model.OrderStatusPaid).
		Order("order_passengers.seat_no ASC").
		Find(&passengers)

	// 司机端保留手机号（需联系乘客），仅脱敏身份证号
	model.MaskPassengersKeepPhone(passengers)

	// 若乘客手机号为空，回退用订单联系人电话，确保司机一定能联系到乘客
	if len(passengers) > 0 {
		orderIDs := make([]uint, 0, len(passengers))
		for _, p := range passengers {
			orderIDs = append(orderIDs, p.OrderID)
		}
		var orders []model.Order
		h.DB.Select("id, contact_phone").Where("id IN ?", orderIDs).Find(&orders)
		phoneMap := make(map[uint]string, len(orders))
		for _, o := range orders {
			phoneMap[o.ID] = o.ContactPhone
		}
		for i := range passengers {
			if passengers[i].Phone == "" {
				passengers[i].Phone = phoneMap[passengers[i].OrderID]
			}
		}
	}

	// 统计核销情况
	var total, checked int64
	h.DB.Model(&model.OrderPassenger{}).
		Joins("JOIN orders ON orders.id = order_passengers.order_id").
		Where("orders.trip_id = ? AND orders.status = ?", tripID, model.OrderStatusPaid).
		Count(&total)
	h.DB.Model(&model.OrderPassenger{}).
		Joins("JOIN orders ON orders.id = order_passengers.order_id").
		Where("orders.trip_id = ? AND orders.status = ? AND order_passengers.check_status = 1", tripID, model.OrderStatusPaid).
		Count(&checked)

	response.OK(c, gin.H{
		"trip":       trip,
		"passengers": passengers,
		"stats": gin.H{
			"total":     total,
			"checked":   checked,
			"unchecked": total - checked,
		},
	})
}

// TripStationStats 每站上下车人数统计（司机端用）
// GET /api/wx/driver/trips/:id/station-stats
func (h *DriverHandler) TripStationStats(c *gin.Context) {
	tripID := c.Param("id")
	driverID := c.GetUint("driver_id")

	var trip model.Trip
	if err := h.DB.Preload("Route.RouteStations.Station").
		Where("id = ? AND driver_id = ?", tripID, driverID).First(&trip).Error; err != nil {
		response.FailMsg(c, response.CodeForbidden, "无权查看此班次")
		return
	}

	// 按stop_order排序站点
	var routeStations []model.RouteStation
	if trip.Route != nil {
		routeStations = trip.Route.RouteStations
	}

	stats := service.SegmentStats(h.DB, trip.ID, trip.TotalSeats, routeStations)

	response.OK(c, gin.H{
		"trip":  trip,
		"stats": stats,
	})
}

// verifyRequest 核销请求（扫码核销，绑定班次）
// VerifyToken: 扫码场景传入完整核销凭证，手动输入可省略
type verifyRequest struct {
	OrderNo     string `json:"order_no" binding:"required"`
	TripID      uint   `json:"trip_id" binding:"required"`
	VerifyToken string `json:"verify_token"` // 可选：带签名的核销凭证（扫码场景传入）
}

// verifyByNoRequest 按订单号核销请求（手动输入，不绑班次）
type verifyByNoRequest struct {
	OrderNo     string `json:"order_no" binding:"required"`
	VerifyToken string `json:"verify_token"` // 可选：扫码场景传入
}

// verifyCore 公共核销逻辑（Verify/VerifyByOrderNo 共用）
func (h *DriverHandler) verifyCore(c *gin.Context, orderNo string, tripID uint, verifyToken string, actionLabel string) {
	driverID := c.GetUint("driver_id")
	driverName, _ := c.Get("driver_name")
	driverNameStr, _ := driverName.(string)

	// 校验订单号格式
	if err := verifytoken.ValidateOrderNo(orderNo); err != nil {
		response.FailMsg(c, response.CodeParamError, "订单号格式非法")
		return
	}

	// 验签核销凭证（如有传入）
	verified := false
	if verifyToken != "" {
		parsed, err := verifytoken.Parse(verifyToken)
		if err != nil {
			response.FailMsg(c, response.CodeParamError, "核销凭证无效或已过期")
			return
		}
		if parsed.OrderNo != orderNo {
			response.FailMsg(c, response.CodeParamError, "核销凭证与订单号不匹配")
			return
		}
		verified = true
	} else {
		// 未验签时限制频率，每小时最多5次
		rateLimitKey := fmt.Sprintf("manual-verify:%d:%s", driverID, time.Now().Format("20060102-15"))
		count, fallback := redis.IncrWithTTL("count:"+rateLimitKey, 1*time.Hour)
		if !fallback && count > manualVerifyLimit {
			response.FailMsg(c, response.CodeForbidden, "未验签核销次数已达本小时上限（5次），请改用扫码核销")
			return
		}
		if fallback {
			// Redis 不可用时，进程级降级计数（单实例有效，多实例需配 Redis）
			bucketKey := fmt.Sprintf("%d:%s", driverID, time.Now().Format("20060102-15"))
			actual, _ := manualVerifyCounts.LoadOrStore(bucketKey, int64(1))
			if actual != nil {
				currentCount := actual.(int64) + 1
				manualVerifyCounts.Store(bucketKey, currentCount)
				if currentCount > manualVerifyLimit {
					response.FailMsg(c, response.CodeForbidden, "未验签核销次数已达本小时上限（5次），请改用扫码核销")
					return
				}
			}
			// 定期清理过期桶（每个整点自动过期，这里懒清理上一个小时的记录）
			prevBucketKey := fmt.Sprintf("%d:%s", driverID, time.Now().Add(-1*time.Hour).Format("20060102-15"))
			manualVerifyCounts.Delete(prevBucketKey)
		}
	}

	// 全流程事务+行锁，防并发竞态
	var (
		order          model.Order
		passengers     []model.OrderPassenger
		orderCompleted bool
	)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 行锁查询订单（Preload Trip，统一用 order.Trip 做班次校验）
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Trip").
			Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return errVerifyOrderNotFound
		}

		// 2. 校验订单状态（行锁内，防并发）
		if order.Status != model.OrderStatusPaid {
			var statusText string
			switch order.Status {
			case model.OrderStatusPending:
				statusText = "未支付"
			case model.OrderStatusCompleted:
				statusText = "已完成"
			case model.OrderStatusRefunded:
				statusText = "已退款"
			case model.OrderStatusCancelled:
				statusText = "已取消"
			default:
				statusText = "状态异常"
			}
			return fmt.Errorf("订单%s，不可核销: %w", statusText, errVerifyOrderStatus)
		}

		// 3. 扫码核销模式(tripID>0)：校验订单班次匹配
		if tripID > 0 && order.TripID != tripID {
			return errVerifyTripMismatch
		}

		// 4. 校验班次属于该司机
		if order.Trip == nil || order.Trip.DriverID != driverID {
			return errVerifyTripNotOwned
		}

		// 5. 校验班次日期：当天班次要求trip_date==today；跨天班次(status=2)允许在到达日核销
		today := time.Now().Format("2006-01-02")
		if string(order.Trip.TripDate) != today && order.Trip.Status != model.TripStatusDepart {
			return errVerifyForbidden
		}

		// 6. 校验班次已发车
		if order.Trip.Status != model.TripStatusDepart {
			return errVerifyTripNotDeparted
		}

		// 7. 查询未核销乘客
		if err := tx.Where("order_id = ? AND check_status = 0", order.ID).Find(&passengers).Error; err != nil {
			return err
		}
		if len(passengers) == 0 {
			return errVerifyAlreadyChecked
		}

		// 8. 核销：标记所有未核销乘客为已核销
		now := time.Now()
		if err := tx.Model(&model.OrderPassenger{}).
			Where("order_id = ? AND check_status = 0", order.ID).
			Updates(map[string]interface{}{
				"check_status": 1,
				"checked_at":   &now,
				"checked_by":   driverID,
			}).Error; err != nil {
			return err
		}

		// 9. 全部核销后更新订单状态为已完成(2)
		var uncheckedCount int64
		tx.Model(&model.OrderPassenger{}).
			Where("order_id = ? AND check_status = 0", order.ID).
			Count(&uncheckedCount)
		if uncheckedCount == 0 {
			if err := tx.Model(&order).Update("status", model.OrderStatusCompleted).Error; err != nil {
				return err
			}
			orderCompleted = true
		}

		return nil
	})

	if err != nil {
		// 映射错误到响应
		switch {
		case errors.Is(err, errVerifyOrderNotFound):
			response.Fail(c, response.CodeOrderNotFound)
		case errors.Is(err, errVerifyOrderStatus):
			response.FailMsg(c, response.CodeOrderStatusErr, err.Error())
		case errors.Is(err, errVerifyTripMismatch):
			response.Fail(c, response.CodeNotThisTrip)
		case errors.Is(err, errVerifyTripNotOwned):
			response.FailMsg(c, response.CodeForbidden, err.Error())
		case errors.Is(err, errVerifyForbidden):
			response.FailMsg(c, response.CodeForbidden, err.Error())
		case errors.Is(err, errVerifyAlreadyChecked):
			response.Fail(c, response.CodeAlreadyChecked)
		case errors.Is(err, errVerifyTripNotDeparted):
			response.FailMsg(c, response.CodeForbidden, err.Error())
		default:
			response.FailMsg(c, response.CodeServerError, "核销失败")
		}
		return
	}

	// 审计日志（事务外，失败不影响核销）
	logAction := actionLabel
	if !verified {
		logAction = actionLabel + "(未验签)"
	}
	writeDriverLog(c, h.DB, driverID, driverNameStr, logAction, order.OrderNo,
		fmt.Sprintf("订单号:%s 班次ID:%d 核销乘客数:%d 订单已完成:%v",
			order.OrderNo, order.TripID, len(passengers), orderCompleted))

	// 返回核销结果
	var allPassengers []model.OrderPassenger
	h.DB.Where("order_id = ?", order.ID).Find(&allPassengers)
	// 司机端保留手机号（需联系乘客），仅脱敏身份证号
	model.MaskPassengersKeepPhone(allPassengers)

	response.OKMsg(c, "核销成功", gin.H{
		"order_no":        order.OrderNo,
		"passenger_count": len(passengers),
		"passengers":      allPassengers,
		"trip_date":       order.TripDate,
		"departure_time":  order.DepartureTime,
		"from_station_id": order.FromStationID,
		"to_station_id":   order.ToStationID,
		"order_completed": orderCompleted,
	})
}

// Verify 核销（扫码核销，需绑定班次）
func (h *DriverHandler) Verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	h.verifyCore(c, req.OrderNo, req.TripID, req.VerifyToken, "扫码核销")
}

// VerifyByOrderNo 按订单号核销（手动输入，不绑定班次）
func (h *DriverHandler) VerifyByOrderNo(c *gin.Context) {
	var req verifyByNoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	h.verifyCore(c, req.OrderNo, 0, req.VerifyToken, "手动核销")
}

