package wx

import (
	"fmt"
	"log"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)
// 司机端：车辆定位上报

// ReportLocation 司机上报车辆实时位置（GPS自动获取，司机无需手动操作）
type reportLocationRequest struct {
	TripID    uint    `json:"trip_id" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Speed     float64 `json:"speed"`
	Heading   float64 `json:"heading"`
}

func (h *DriverHandler) ReportLocation(c *gin.Context) {
	var req reportLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	driverID := c.GetUint("driver_id")

	// 经纬度范围校验
	if req.Longitude < -180 || req.Longitude > 180 || req.Latitude < -90 || req.Latitude > 90 {
		response.FailMsg(c, response.CodeParamError, "经纬度数据异常")
		return
	}
	// 拒绝(0,0)坐标，GPS信号丢失时可能上报无效坐标
	if req.Longitude == 0 && req.Latitude == 0 {
		response.FailMsg(c, response.CodeParamError, "GPS坐标无效（经纬度均为0）")
		return
	}

	// 允许班次运行期间上报位置（支持跨天班次：发车日~到达日期间均可上报）
	// 班次必须已发车(status=2)且发车日不晚于今天：
	//   - status=2 保证班次正在运行（未发车/已结束的班次不会接收GPS数据）
	//   - trip_date<=today 防止未来班次提前上报
	//   - 无需上限日期限制：status=2本身已保证班次未结束，且乘客端有5分钟新鲜度校验
	today := time.Now().Format("2006-01-02")
	var trip model.Trip
	if err := h.DB.Where("id = ? AND driver_id = ? AND status = 2 AND trip_date <= ?", req.TripID, driverID, today).First(&trip).Error; err != nil {
		response.FailMsg(c, response.CodeForbidden, "无权上报此班次位置或班次未发车")
		return
	}

	loc := model.VehicleLocation{
		TripID:     req.TripID,
		VehicleID:  trip.VehicleID,
		DriverID:   driverID,
		Longitude:  req.Longitude,
		Latitude:   req.Latitude,
		Speed:      req.Speed,
		Heading:    req.Heading,
		ReportedAt: model.JSONTime(time.Now()),
	}
	if err := h.DB.Create(&loc).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "位置上报失败")
		return
	}

	response.OK(c, gin.H{"id": loc.ID})
}

// StartTrip 司机开始行程（将班次状态改为已发车，触发乘客端可查看位置）
func (h *DriverHandler) StartTrip(c *gin.Context) {
	tripID := c.Param("id")
	driverID := c.GetUint("driver_id")

	var trip model.Trip
	if err := h.DB.Where("id = ? AND driver_id = ?", tripID, driverID).First(&trip).Error; err != nil {
		response.FailMsg(c, response.CodeForbidden, "无权操作此班次")
		return
	}

	// 校验班次日期为今天，防止提前发车非今日班次
	today := time.Now().Format("2006-01-02")
	if string(trip.TripDate) != today {
		response.FailMsg(c, response.CodeParamError, "只能发车今日班次")
		return
	}

	// 班次状态改为已发车(2)
	if trip.Status == 1 {
		if err := h.DB.Model(&trip).Where("status = 1").Update("status", 2).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "状态更新失败")
			return
		}
		// 订阅消息：班次发车通知（异步发送，不阻塞司机操作）
		service.NotifyTripDeparture(h.DB, trip)
	}

	response.OKMsg(c, "行程已开始", gin.H{"trip_id": trip.ID, "status": 2})
}

// EndTrip 司机结束行程（班次状态改为已完成，同时处理关联订单，清除位置记录）
func (h *DriverHandler) EndTrip(c *gin.Context) {
	tripID := c.Param("id")
	driverID := c.GetUint("driver_id")

	var trip model.Trip
	if err := h.DB.Where("id = ? AND driver_id = ?", tripID, driverID).First(&trip).Error; err != nil {
		response.FailMsg(c, response.CodeForbidden, "无权操作此班次")
		return
	}

	// 使用事务确保班次状态更新和订单处理原子性（复用 CompleteTrip，与定时任务自动完成保持一致）
	var tripCompleted bool
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if trip.Status != 2 {
			return nil // 非已发车状态，不处理
		}
		var e error
		tripCompleted, _, _, e = service.CompleteTrip(tx, trip.ID)
		return e
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "结束行程失败")
		return
	}

	// 订阅消息：班次到达终点通知（异步发送，不阻塞司机操作）
	// 仅当本次实际完成班次时触发，避免重复通知
	if tripCompleted {
		service.NotifyTripArrival(h.DB, trip)
	}

	response.OKMsg(c, "行程已结束", gin.H{"trip_id": trip.ID, "status": 4})
}

// MarkStation 司机标记已驶过到第几站（更新 current_passed_order，控制途经站能否下单）
// 司机到达某站后点击"标记已过站"，系统将拒绝该站及之前所有站的新订单
// 传 passed_order=0 可重置标记（取消所有过站标记）
type markStationRequest struct {
	// 不用 binding:"required"，否则传 0（重置过站标记）会被零值校验拒绝
	PassedOrder int `json:"passed_order"`
}

func (h *DriverHandler) MarkStation(c *gin.Context) {
	tripID := c.Param("id")
	driverID := c.GetUint("driver_id")
	driverName, _ := c.Get("driver_name")
	driverNameStr, _ := driverName.(string)

	var req markStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.PassedOrder < 0 {
		response.FailMsg(c, response.CodeParamError, "到站标记不能为负数")
		return
	}

	var trip model.Trip
	if err := h.DB.Where("id = ? AND driver_id = ?", tripID, driverID).First(&trip).Error; err != nil {
		response.FailMsg(c, response.CodeForbidden, "无权操作此班次")
		return
	}
	// 班次必须已发车(2)才能标记过站
	if trip.Status != 2 {
		response.FailMsg(c, response.CodeForbidden, "班次尚未发车，无法标记到站")
		return
	}
	// 过站序号不能超过该线路的最大站序
	var maxStopOrder int
	h.DB.Model(&model.RouteStation{}).Where("route_id = ?", trip.RouteID).
		Select("COALESCE(MAX(stop_order), 0)").Scan(&maxStopOrder)
	if req.PassedOrder > maxStopOrder {
		response.FailMsg(c, response.CodeParamError, fmt.Sprintf("过站序号超出范围（最大%d）", maxStopOrder))
		return
	}

	if err := h.DB.Model(&trip).Update("current_passed_order", req.PassedOrder).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新到站标记失败")
		return
	}

	writeDriverLog(c, h.DB, driverID, driverNameStr, "标记过站",
		fmt.Sprintf("班次ID:%s", tripID),
		fmt.Sprintf("班次号:%s 标记驶过到第%d站", trip.TripNo, req.PassedOrder))

	response.OKMsg(c, "到站标记已更新", gin.H{
		"trip_id":              trip.ID,
		"current_passed_order": req.PassedOrder,
	})
}

// Logout 司机退出登录（写入 token_invalid_before，使该司机早于此时间签发的Token失效）
func (h *DriverHandler) Logout(c *gin.Context) {
	driverID := c.GetUint("driver_id")
	now := time.Now()
	if err := h.DB.Model(&model.Driver{}).Where("id = ?", driverID).Update("token_invalid_before", &now).Error; err != nil {
		log.Printf("写入登出时间失败: %v\n", err)
	}
	response.OKMsg(c, "退出成功", nil)
}

