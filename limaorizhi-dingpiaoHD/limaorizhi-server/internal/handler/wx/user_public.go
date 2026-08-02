// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"strconv"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 公开接口

// Stations 站点列表（公开）
func (h *UserHandler) Stations(c *gin.Context) {
	var list []model.Station
	h.DB.Where("status = 1").Order("sort_order ASC, id ASC").Find(&list)
	response.OK(c, list)
}

// Routes 线路列表（公开，含站点序列）
func (h *UserHandler) Routes(c *gin.Context) {
	var list []model.Route
	h.DB.Preload("FromStation").Preload("ToStation").
		Preload("RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("RouteStations.Station").
		Where("status = 1").Order("id ASC").Find(&list)
	response.OK(c, list)
}

// Trips 搜索可售班次
func (h *UserHandler) Trips(c *gin.Context) {
	tripDate := c.DefaultQuery("trip_date", time.Now().Format("2006-01-02"))
	fromStationID := c.Query("from_station_id")
	toStationID := c.Query("to_station_id")

	// status IN (1,2)：已发车但未到终点的班次仍对途经站乘客可见
	// 下单/支付时由 isStationPassed 判断车是否已驶过用户上车站
	query := h.DB.Model(&model.Trip{}).Where("status IN (?, ?) AND trip_date = ?", model.TripStatusSale, model.TripStatusDepart, tripDate)
	// 区间命中：优先查 route_stations 站点序列（上车站在下车站之前），兼容无序列的旧 route（按 from/to）
	if fromStationID != "" && toStationID != "" {
		query = query.Where("route_id IN ("+
			"SELECT rs1.route_id FROM route_stations rs1 JOIN route_stations rs2 ON rs1.route_id=rs2.route_id "+
			"WHERE rs1.station_id=? AND rs2.station_id=? AND rs1.stop_order < rs2.stop_order "+
			"UNION SELECT id FROM routes WHERE from_station_id=? AND to_station_id=?)",
			fromStationID, toStationID, fromStationID, toStationID)
	} else if fromStationID != "" {
		query = query.Where("route_id IN (SELECT route_id FROM route_stations WHERE station_id=? UNION SELECT id FROM routes WHERE from_station_id=?)",
			fromStationID, fromStationID)
	} else if toStationID != "" {
		query = query.Where("route_id IN (SELECT route_id FROM route_stations WHERE station_id=? UNION SELECT id FROM routes WHERE to_station_id=?)",
			toStationID, toStationID)
	}

	var trips []model.Trip
	query.Preload("Route.FromStation").Preload("Route.ToStation").Preload("Vehicle").
		Preload("Route.RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("Route.RouteStations.Station").
		Order("departure_time ASC").Find(&trips)
	// 过滤driver_id并用实时余票覆盖静态值
	for i := range trips {
		trips[i].DriverID = 0
		trips[i].Driver = nil
		trips[i].AvailableSeats = service.RealtimeAvailableSeats(h.DB, trips[i].ID, trips[i].TotalSeats)
	}
	response.OK(c, trips)
}

// TripDetail 班次详情（含站点序列与各站到站价，供前端选上下车站）
func (h *UserHandler) TripDetail(c *gin.Context) {
	id := c.Param("id")
	var trip model.Trip
	if err := h.DB.Preload("Route.FromStation").Preload("Route.ToStation").Preload("Vehicle").
		Preload("Route.RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("Route.RouteStations.Station").
		Where("id = ? AND status IN (?, ?)", id, model.TripStatusSale, model.TripStatusDepart).First(&trip).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}
	trip.DriverID = 0
	trip.Driver = nil
	// 用实时区间余票覆盖静态 available_seats，让前端看到真实余票
	trip.AvailableSeats = service.RealtimeAvailableSeats(h.DB, trip.ID, trip.TotalSeats)
	// 计算有效过站序号（GPS优先+手动标记取max），供班次详情页进度条展示
	var routeStations []model.RouteStation
	if trip.Route != nil {
		routeStations = trip.Route.RouteStations
	}
	effectiveOrder := effectivePassedOrder(h.DB, trip, routeStations)
	response.OK(c, gin.H{
		"trip":                   trip,
		"effective_passed_order": effectiveOrder,
	})
}

// TripAvailableSeats 查询班次指定区间的实时可用余票数
// GET /api/wx/trips/:id/available-seats?from_station_id=X&to_station_id=Y
func (h *UserHandler) TripAvailableSeats(c *gin.Context) {
	tripID := c.Param("id")
	fromStationID := c.Query("from_station_id")
	toStationID := c.Query("to_station_id")

	if fromStationID == "" || toStationID == "" {
		response.FailMsg(c, response.CodeParamError, "请提供上下车站ID")
		return
	}

	var trip model.Trip
	if err := h.DB.Where("id = ? AND status IN (?, ?)", tripID, model.TripStatusSale, model.TripStatusDepart).First(&trip).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}

	// 查询上下车站的 stop_order
	var fromRS, toRS model.RouteStation
	if err := h.DB.Where("route_id = ? AND station_id = ?", trip.RouteID, fromStationID).First(&fromRS).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "上车站不在该线路中")
		return
	}
	if err := h.DB.Where("route_id = ? AND station_id = ?", trip.RouteID, toStationID).First(&toRS).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "下车站不在该线路中")
		return
	}
	if fromRS.StopOrder >= toRS.StopOrder {
		response.FailMsg(c, response.CodeParamError, "上车站必须在下车站之前")
		return
	}

	avail, err := service.AvailableSeatsForSegment(h.DB, trip.ID, trip.TotalSeats, fromRS.StopOrder, toRS.StopOrder)
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "查询余票失败")
		return
	}

	response.OK(c, gin.H{
		"trip_id":         trip.ID,
		"total_seats":     trip.TotalSeats,
		"available_seats": avail,
		"from_station_id": fromStationID,
		"to_station_id":   toStationID,
	})
}

// TripLocation 查询班次车辆实时位置（需登录鉴权，校验用户是否有该班次订单）
func (h *UserHandler) TripLocation(c *gin.Context) {
	userToken := c.GetUint("user_id")
	tripID := c.Param("id")
	var trip model.Trip
	// 仅已发车(status=2)班次返回位置
	if err := h.DB.Preload("Route.FromStation").Preload("Route.ToStation").Preload("Vehicle").
		Preload("Route.RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("Route.RouteStations.Station").
		Where("status = ?", model.TripStatusDepart).First(&trip, tripID).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}

	// 校验用户是否有该班次的有效订单
	var orderCount int64
	h.DB.Model(&model.Order{}).Where("trip_id = ? AND user_id = ? AND status IN (?, ?, ?)", trip.ID, userToken, model.OrderStatusPending, model.OrderStatusPaid, model.OrderStatusCompleted).Count(&orderCount)
	if orderCount == 0 {
		response.FailMsg(c, response.CodeForbidden, "您没有该班次的订单，无法查看车辆位置")
		return
	}

	trip.DriverID = 0
	trip.Driver = nil

	// 计算有效过站序号（GPS优先+手动标记取max），供乘客端进度条展示
	// 与 isStationPassed 下单逻辑使用同一函数，保证展示与判断一致
	var routeStations []model.RouteStation
	if trip.Route != nil {
		routeStations = trip.Route.RouteStations
	}
	effectiveOrder := effectivePassedOrder(h.DB, trip, routeStations)

	// 查询5分钟内的最近位置记录
	var loc model.VehicleLocation
	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	result := h.DB.Where("trip_id = ? AND reported_at > ?", trip.ID, fiveMinAgo).Order("reported_at DESC").First(&loc)

	if result.Error != nil {
		// 无近期位置数据
		response.OK(c, gin.H{
			"trip":                   trip,
			"location":               nil,
			"effective_passed_order": effectiveOrder,
		})
		return
	}

	response.OK(c, gin.H{
		"trip":                   trip,
		"location":               loc,
		"effective_passed_order": effectiveOrder,
	})
}

// PublicConfig 公开系统配置（仅返回白名单中的配置项，防止敏感信息泄露）
func (h *UserHandler) PublicConfig(c *gin.Context) {
	// 公开配置白名单（仅返回非敏感配置项）
	publicKeys := map[string]bool{
		"site_name":                     true,
		"customer_service_phone":        true,
		"after_sales_wechat":            true,
		"notice":                        true,
		"order_expire_minutes":          true,
		"refund_before_departure_hours": true,
		"refund_fee_rate":               true,
		"insurance_fee":                 true,
		"insurance_required":            true,
		"cargo_price_per_km":            true,
		"cargo_min_fee":                 true,
		"cargo_free_weight":             true,
		"cargo_extra_weight_fee":        true,
		"cargo_max_weight":              true,
		"mine_menu_layout_type":         true,
		"logout_position":               true,
		// 协议政策（用户协议 + 隐私政策，小程序登录页展示）
		"user_agreement": true,
		"privacy_policy": true,
	}
	var configs []model.SystemConfig
	h.DB.Find(&configs)
	result := make(map[string]string)
	for _, cfg := range configs {
		if publicKeys[cfg.ConfigKey] {
			result[cfg.ConfigKey] = cfg.ConfigValue
		}
	}
	// 保险公司配置覆盖：若启用了一家保险公司，insurance_fee/insurance_required 用其值覆盖，
	// 向后兼容：未配置保险公司时仍读 system_configs 兜底。
	if provider, err := service.GetActiveProvider(h.DB); err == nil && provider != nil {
		result["insurance_fee"] = strconv.FormatFloat(provider.Fee, 'f', -1, 64)
		if provider.Required {
			result["insurance_required"] = "true"
		} else {
			result["insurance_required"] = "false"
		}
	}
	response.OK(c, result)
}
