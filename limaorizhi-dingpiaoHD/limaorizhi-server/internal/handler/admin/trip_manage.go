package admin

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	wxpay "limaorizhi-server/internal/handler/wx"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/sanitize"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 班次管理

// disableFKChecks 临时禁用外键检查（跨数据库兼容）
// MySQL: SET FOREIGN_KEY_CHECKS = 0
// PostgreSQL: SET session_replication_role = 'replica'
func disableFKChecks(tx *gorm.DB) {
	if tx.Dialector.Name() == "postgres" {
		tx.Exec("SET session_replication_role = 'replica'")
	} else {
		tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
	}
}

// enableFKChecks 恢复外键检查
func enableFKChecks(tx *gorm.DB) {
	if tx.Dialector.Name() == "postgres" {
		tx.Exec("SET session_replication_role = 'origin'")
	} else {
		tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}
}

// readRefundFeeRate 从系统配置读取退票手续费率（与用户端/管理端退票逻辑一致）
// 返回0~100之间的费率，配置缺失或异常时返回0（不扣手续费）
func readRefundFeeRate(db *gorm.DB) float64 {
	var feeRateCfg model.SystemConfig
	feeRate := 0.0
	if err := db.Where("config_key = ?", "refund_fee_rate").First(&feeRateCfg).Error; err == nil {
		if v, err := strconv.ParseFloat(feeRateCfg.ConfigValue, 64); err == nil && v >= 0 {
			feeRate = v
		}
	}
	if feeRate > 100 {
		feeRate = 100
	}
	return feeRate
}

type TripHandler struct{ DB *gorm.DB }

func NewTripHandler(db *gorm.DB) *TripHandler { return &TripHandler{DB: db} }

func (h *TripHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.DB.Model(&model.Trip{})
	if routeID := c.Query("route_id"); routeID != "" {
		query = query.Where("route_id = ?", routeID)
	}
	if tripDate := c.Query("trip_date"); tripDate != "" {
		query = query.Where("trip_date = ?", tripDate)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if tripNo := c.Query("trip_no"); tripNo != "" {
		query = query.Where("trip_no = ?", tripNo)
	}
	query.Count(&total)

	var list []model.Trip
	query.Preload("Route.FromStation").Preload("Route.ToStation").Preload("Vehicle").Preload("Driver").
		Order("trip_date DESC, departure_time ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

// createTripRequest 班次创建请求（DTO白名单，防止Mass Assignment）
type createTripRequest struct {
	RouteID          uint    `json:"route_id" binding:"required"`
	VehicleID        uint    `json:"vehicle_id" binding:"required"`
	DriverID         uint    `json:"driver_id"`
	TripNo           string  `json:"trip_no"`
	TripDate         string  `json:"trip_date" binding:"required"`
	DepartureTime    string  `json:"departure_time" binding:"required"`
	ArrivalTime      string  `json:"arrival_time" binding:"required"`
	ArrivalDayOffset int     `json:"arrival_day_offset"` // 到达相对发车日的天数偏移(0=当天,1=次日...)，跨天班次用
	TotalSeats       int     `json:"total_seats"`
	AvailableSeats   int     `json:"available_seats"`
	BasePrice        float64 `json:"base_price"` // 0是合法值（免费班次），不设required
	// 无座票配置（春运等客流高峰开放站票）
	AllowStanding    bool    `json:"allow_standing"`    // 是否开放无座票
	StandingQuota    int     `json:"standing_quota"`    // 无座票可售数量上限（0=不允许，>0时开放）
	StandingDiscount float64 `json:"standing_discount"` // 无座票价折扣（0~1，0/留空=不强制打折按1处理，1=与座位同价）
	Force            bool    `json:"force"` // 强制创建（已知冲突仍继续）
}

func (h *TripHandler) Create(c *gin.Context) {
	var req createTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 座位数不能是负的
	if req.TotalSeats < 0 {
		response.FailMsg(c, response.CodeParamError, "总座位数不能为负数")
		return
	}
	if req.AvailableSeats < 0 {
		response.FailMsg(c, response.CodeParamError, "可售座位数不能为负数")
		return
	}
	// 无座票配置校验：开放无座必须设置正数配额；折扣限制在0~1区间（0=留空按1处理，与座位同价不强制打折）
	if req.AllowStanding && req.StandingQuota <= 0 {
		response.FailMsg(c, response.CodeParamError, "开放无座票需设置无座票配额（>0）")
		return
	}
	if req.StandingQuota < 0 {
		response.FailMsg(c, response.CodeParamError, "无座票配额不能为负数")
		return
	}
	if req.StandingDiscount < 0 || req.StandingDiscount > 1 {
		response.FailMsg(c, response.CodeParamError, "无座票价折扣需在0~1之间")
		return
	}
	// 校验时间格式和逻辑（支持跨天班次：arrival_day_offset>0 时到达时刻可小于发车时刻）
	timeRe := regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9](:[0-5][0-9])?$`)
	if !timeRe.MatchString(req.DepartureTime) {
		response.FailMsg(c, response.CodeParamError, "发车时间不正确，请重新选择发车时间")
		return
	}
	if !timeRe.MatchString(req.ArrivalTime) {
		response.FailMsg(c, response.CodeParamError, "到达时间不正确，请重新选择到达时间")
		return
	}
	if req.ArrivalDayOffset < 0 {
		response.FailMsg(c, response.CodeParamError, "到达天数偏移不能为负数")
		return
	}
	// 当天班次(offset=0)要求到达时刻晚于发车时刻；跨天班次(offset>0)允许到达时刻小于发车时刻
	if req.ArrivalDayOffset == 0 && req.DepartureTime >= req.ArrivalTime {
		response.FailMsg(c, response.CodeParamError, "当天到达时间必须晚于发车时间，如需次日到达请选择到达天数")
		return
	}
	// 车辆和线路得存在
	var vehicle model.Vehicle
	if err := h.DB.First(&vehicle, req.VehicleID).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "车辆不存在")
		return
	}
	// 校验车辆状态，维修中不可分配
	if vehicle.Status == 0 {
		response.FailMsg(c, response.CodeParamError, "车辆处于维修状态，无法分配")
		return
	}
	var route model.Route
	if err := h.DB.First(&route, req.RouteID).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "线路不存在")
		return
	}
	// 校验司机状态（若分配了司机）
	if req.DriverID > 0 {
		var driver model.Driver
		if err := h.DB.First(&driver, req.DriverID).Error; err != nil {
			response.FailMsg(c, response.CodeParamError, "司机不存在")
			return
		}
		if driver.Status == 0 {
			response.FailMsg(c, response.CodeParamError, "司机已被禁用，无法分配")
			return
		}
	}
	// 检查TripNo是否重复
	if req.TripNo != "" {
		var existCount int64
		h.DB.Model(&model.Trip{}).Where("trip_no = ?", req.TripNo).Count(&existCount)
		if existCount > 0 {
			response.FailMsg(c, response.CodeParamError, "班次号已存在，请使用不同的班次号")
			return
		}
	}
	// 发车日期不能是过去，今天的发车时刻已经过了的也不能建（要不用户买到了坐不上，闹纠纷）
	depMoment, depErr := time.ParseInLocation("2006-01-02 15:04", req.TripDate+" "+req.DepartureTime, time.Local)
	if depErr != nil {
		response.FailMsg(c, response.CodeParamError, "班次日期格式不正确")
		return
	}
	if !depMoment.After(time.Now()) {
		response.FailMsg(c, response.CodeParamError, "发车日期不能是过去，今天已过发车时刻的班次也不能创建")
		return
	}
	// 建班次时看司机/车辆冲不冲突
	conflicts := service.CheckTripConflict(h.DB, service.TripConflictCheck{
		ExcludeTripID:  0,
		DriverID:       req.DriverID,
		VehicleID:      req.VehicleID,
		TripDate:       req.TripDate,
		DepartureTime:  req.DepartureTime,
		ArrivalTime:      req.ArrivalTime,
		ArrivalDayOffset: req.ArrivalDayOffset,
		FromStationID:    route.FromStationID,
		ToStationID:    route.ToStationID,
	})
	// 硬冲突（时间重叠/车辆冲突）不可强制创建
	if service.HasHardConflict(conflicts) {
		msg := service.ConflictSummary(conflicts) + "（时间重叠/车辆冲突不可强制创建）"
		response.FailWithData(c, response.CodeServerError, msg, gin.H{"conflicts": conflicts})
		return
	}
	if len(conflicts) > 0 && !req.Force {
		msg := service.ConflictSummary(conflicts)
		response.FailWithData(c, response.CodeServerError, msg, gin.H{"conflicts": conflicts})
		return
	}
	// 构建班次对象（仅使用白名单字段，防止Mass Assignment）
	t := model.Trip{
		RouteID:          req.RouteID,
		VehicleID:        req.VehicleID,
		DriverID:         req.DriverID,
		TripNo:           req.TripNo,
		TripDate:         model.JSONDate(req.TripDate),
		DepartureTime:    req.DepartureTime,
		ArrivalTime:      req.ArrivalTime,
		ArrivalDayOffset: req.ArrivalDayOffset,
		TotalSeats:       req.TotalSeats,
		BasePrice:        req.BasePrice,
		Status:           1, // 默认可售
		AllowStanding:    req.AllowStanding,
		StandingQuota:    req.StandingQuota,
		StandingDiscount: req.StandingDiscount,
	}
	// 无座折扣兜底：0/留空 = 与座位同价（不强制打折），>1 一律归一
	if t.StandingDiscount <= 0 {
		t.StandingDiscount = 1 // 默认与座位同价（不强制打折）
	}
	if t.StandingDiscount > 1 {
		t.StandingDiscount = 1
	}
	if t.TotalSeats == 0 {
		t.TotalSeats = vehicle.SeatCount
	}
	// 可售座位不能超总数
	if req.AvailableSeats > t.TotalSeats {
		response.FailMsg(c, response.CodeParamError, fmt.Sprintf("可售座位数不能超过总座位数（%d）", t.TotalSeats))
		return
	}
	if req.AvailableSeats == 0 {
		t.AvailableSeats = t.TotalSeats
	} else {
		t.AvailableSeats = req.AvailableSeats
	}
	if t.TripNo == "" {
		t.TripNo = fmt.Sprintf("TC%d%s%s", t.RouteID, t.TripDate, t.DepartureTime)
	}
	if err := h.DB.Create(&t).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	forceNote := ""
	if len(conflicts) > 0 && req.Force {
		forceNote = fmt.Sprintf(" [强制创建，已覆盖%d个冲突]", len(conflicts))
	}
	WriteLog(c, h.DB, "班次", "新增", t.TripNo, fmt.Sprintf("班次ID:%d 班次号:%s%s", t.ID, t.TripNo, forceNote))
	response.OK(c, t)
}

type updateTripRequest struct {
	RouteID          uint     `json:"route_id"`
	VehicleID        uint     `json:"vehicle_id"`
	DriverID         uint     `json:"driver_id"`
	TripNo           string   `json:"trip_no"`
	TripDate         string   `json:"trip_date"`
	DepartureTime    string   `json:"departure_time"`
	ArrivalTime      string   `json:"arrival_time"`
	ArrivalDayOffset int      `json:"arrival_day_offset"` // 到达相对发车日的天数偏移(0=当天,1=次日...)
	TotalSeats       int      `json:"total_seats"`
	AvailableSeats   int      `json:"available_seats"`
	BasePrice        float64  `json:"base_price"`
	Status           int8     `json:"status"`
	CurrentPassedOrder int    `json:"current_passed_order"`
	// 无座票配置（指针类型：未传时保持原值，避免零值清空）
	AllowStanding    *bool    `json:"allow_standing"`
	StandingQuota    *int     `json:"standing_quota"`
	StandingDiscount *float64 `json:"standing_discount"`
	Force            bool     `json:"force"` // 强制更新（已知冲突仍继续）
}

func (h *TripHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var trip model.Trip
	if err := h.DB.First(&trip, id).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}
	var req updateTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 使用Updates而非Save，仅更新指定字段，防止Mass Assignment
	// 座位数不能是负的
	if req.TotalSeats < 0 {
		response.FailMsg(c, response.CodeParamError, "总座位数不能为负数")
		return
	}
	if req.AvailableSeats < 0 {
		response.FailMsg(c, response.CodeParamError, "可售座位数不能为负数")
		return
	}
	// 无座票配置校验：开放无座必须设置正数配额；折扣限制在0~1区间（0=留空按1处理，与座位同价不强制打折）
	if req.AllowStanding != nil && *req.AllowStanding {
		quota := 0
		if req.StandingQuota != nil {
			quota = *req.StandingQuota
		} else {
			quota = trip.StandingQuota
		}
		if quota <= 0 {
			response.FailMsg(c, response.CodeParamError, "开放无座票需设置无座票配额（>0）")
			return
		}
	}
	if req.StandingQuota != nil && *req.StandingQuota < 0 {
		response.FailMsg(c, response.CodeParamError, "无座票配额不能为负数")
		return
	}
	if req.StandingDiscount != nil && (*req.StandingDiscount < 0 || *req.StandingDiscount > 1) {
		response.FailMsg(c, response.CodeParamError, "无座票价折扣需在0~1之间")
		return
	}
	// TotalSeats=0时用当前班次值校验
	effectiveTotalSeats := req.TotalSeats
	if effectiveTotalSeats == 0 {
		effectiveTotalSeats = trip.TotalSeats
	}
	if req.AvailableSeats > effectiveTotalSeats {
		response.FailMsg(c, response.CodeParamError, fmt.Sprintf("可售座位数不能超过总座位数（%d）", effectiveTotalSeats))
		return
	}
	// 编辑座位数时校验已售座位数，防止超卖
	if req.TotalSeats > 0 {
		var soldSeats int64
		h.DB.Model(&model.Order{}).
			Where("trip_id = ? AND order_type = 1 AND status IN (1, 2)", trip.ID).
			Select("COALESCE(SUM(passenger_count), 0)").
			Scan(&soldSeats)
		if req.TotalSeats < int(soldSeats) {
			response.FailMsg(c, response.CodeParamError, fmt.Sprintf("总座位数不能小于已售座位数（已售%d座）", soldSeats))
			return
		}
	}
	// 班次更新时检测司机/车辆冲突
	// 计算生效值（0/空表示不修改，用当前值）
	effVehicleID := req.VehicleID
	if effVehicleID == 0 {
		effVehicleID = trip.VehicleID
	}
	effDriverID := req.DriverID // 0表示取消司机分配，不用于冲突检测
	effTripDate := req.TripDate
	if effTripDate == "" {
		effTripDate = string(trip.TripDate)
	}
	effDepTime := req.DepartureTime
	if effDepTime == "" {
		effDepTime = trip.DepartureTime
	}
	effArrTime := req.ArrivalTime
	if effArrTime == "" {
		effArrTime = trip.ArrivalTime
	}
	effArrOffset := req.ArrivalDayOffset
	if req.ArrivalTime == "" {
		effArrOffset = trip.ArrivalDayOffset
	}
	effRouteID := req.RouteID
	if effRouteID == 0 {
		effRouteID = trip.RouteID
	}
	// 校验车辆状态（若更换了车辆）
	if req.VehicleID > 0 && req.VehicleID != trip.VehicleID {
		var newVehicle model.Vehicle
		if err := h.DB.First(&newVehicle, req.VehicleID).Error; err != nil {
			response.FailMsg(c, response.CodeParamError, "车辆不存在")
			return
		}
		if newVehicle.Status == 0 {
			response.FailMsg(c, response.CodeParamError, "车辆处于维修状态，无法分配")
			return
		}
	}
	// 校验司机状态（若更换了司机）
	if req.DriverID > 0 && req.DriverID != trip.DriverID {
		var newDriver model.Driver
		if err := h.DB.First(&newDriver, req.DriverID).Error; err != nil {
			response.FailMsg(c, response.CodeParamError, "司机不存在")
			return
		}
		if newDriver.Status == 0 {
			response.FailMsg(c, response.CodeParamError, "司机已被禁用，无法分配")
			return
		}
	}
	// 检查TripNo是否重复（若修改了班次号）
	if req.TripNo != "" && req.TripNo != trip.TripNo {
		var existCount int64
		h.DB.Model(&model.Trip{}).Where("trip_no = ? AND id != ?", req.TripNo, id).Count(&existCount)
		if existCount > 0 {
			response.FailMsg(c, response.CodeParamError, "班次号已存在，请使用不同的班次号")
			return
		}
	}
	// 查询线路获取起终站ID（用于冲突检测）
	var fromSID, toSID uint
	var effRoute model.Route
	if err := h.DB.First(&effRoute, effRouteID).Error; err == nil {
		fromSID = effRoute.FromStationID
		toSID = effRoute.ToStationID
	}
	// 冲突检测（无论线路是否查到都执行，防止遗漏）
	conflicts := service.CheckTripConflict(h.DB, service.TripConflictCheck{
		ExcludeTripID:   trip.ID,
		DriverID:         effDriverID,
		VehicleID:        effVehicleID,
		TripDate:         effTripDate,
		DepartureTime:    effDepTime,
		ArrivalTime:      effArrTime,
		ArrivalDayOffset: effArrOffset,
		FromStationID:    fromSID,
		ToStationID:      toSID,
	})
	// 硬冲突（时间重叠/车辆冲突）不可强制保存
	if service.HasHardConflict(conflicts) {
		msg := service.ConflictSummary(conflicts) + "（时间重叠/车辆冲突不可强制保存）"
		response.FailWithData(c, response.CodeServerError, msg, gin.H{"conflicts": conflicts})
		return
	}
	if len(conflicts) > 0 && !req.Force {
		msg := service.ConflictSummary(conflicts)
		response.FailWithData(c, response.CodeServerError, msg, gin.H{"conflicts": conflicts})
		return
	}
	// 可售座位兜底：AvailableSeats=0 表示"用默认"，自动等于最终总座位数（与新增逻辑一致，避免误存0导致小程序显示售罄）
	availSeats := req.AvailableSeats
	if availSeats == 0 {
		availSeats = effectiveTotalSeats
	}
	updates := map[string]interface{}{}
	// driver_id=0 表示取消司机分配，是合法值，始终写入
	updates["driver_id"] = req.DriverID
	// status 是有效值（0=下架也是合法状态），始终写入
	newStatus := req.Status
	// 历史班次（已完成/已发车）把日期改为今天及以后时，若用户未手动调整状态，
	// 自动恢复为"可售"——否则状态仍是已完成，小程序端（只显示可售/已发车）永远搜不到
	// 判定"用户未手动调整"：前端表单回填原状态，未改动则 req.Status 与班次当前状态一致
	autoRestoredStatus := false
	if (trip.Status == model.TripStatusFinish || trip.Status == model.TripStatusDepart) &&
		req.Status == trip.Status && effTripDate != "" {
		// 按"天"比较：今天零点及以后的日期都视为可恢复（避免今天00:00被当成过去日期）
		now := time.Now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if d, err := time.Parse("2006-01-02", effTripDate); err == nil && !d.Before(todayStart) {
			newStatus = model.TripStatusSale
			autoRestoredStatus = true
		}
	}
	updates["status"] = newStatus
	// current_passed_order 是有效整数值，始终写入
	updates["current_passed_order"] = req.CurrentPassedOrder
	// available_seats 已有兜底逻辑，始终写入
	updates["available_seats"] = availSeats
	// base_price=0 是合法值（免费班次），始终写入
	updates["base_price"] = req.BasePrice
	// 以下字段为空/0时不写入，保留原值（防止零值清空）
	if req.RouteID > 0 {
		updates["route_id"] = req.RouteID
	}
	if req.VehicleID > 0 {
		updates["vehicle_id"] = req.VehicleID
	}
	if req.TripNo != "" {
		updates["trip_no"] = req.TripNo
	}
	if req.TripDate != "" {
		updates["trip_date"] = req.TripDate
	}
	if req.DepartureTime != "" {
		updates["departure_time"] = req.DepartureTime
	}
	if req.ArrivalTime != "" {
		updates["arrival_time"] = req.ArrivalTime
		updates["arrival_day_offset"] = req.ArrivalDayOffset
	}
	// 总座位数：0 表示不修改，保持原值（避免误存为0导致全部售罄）
	if req.TotalSeats > 0 {
		updates["total_seats"] = req.TotalSeats
	}
	// 无座票配置：指针非空才写入，未传时保持原值（避免零值清空）
	if req.AllowStanding != nil {
		updates["allow_standing"] = *req.AllowStanding
	}
	if req.StandingQuota != nil {
		updates["standing_quota"] = *req.StandingQuota
	}
	if req.StandingDiscount != nil {
		discount := *req.StandingDiscount
		if discount <= 0 {
			discount = 1 // 默认与座位同价（不强制打折）
		}
		if discount > 1 {
			discount = 1
		}
		updates["standing_discount"] = discount
	}
	if err := h.DB.Model(&trip).Updates(updates).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	// 重新加载完整数据
	h.DB.First(&trip, id)
	// 操作日志带上"保存后司机ID"，便于核对司机分配是否真的写入（0=未分配/已取消分配）
	driverDesc := "无(0)"
	if trip.DriverID > 0 {
		driverDesc = fmt.Sprintf("#%d", trip.DriverID)
	}
	WriteLog(c, h.DB, "班次", "编辑", trip.TripNo, fmt.Sprintf("班次ID:%s 班次号:%s 保存后司机:%s", id, trip.TripNo, driverDesc))
	if autoRestoredStatus {
		response.OKMsg(c, "班次日期已调整为今天及以后，状态已自动恢复为可售", trip)
		return
	}
	response.OK(c, trip)
}

func (h *TripHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	force := c.Query("force") == "1"

	// 查班次
	var trip model.Trip
	if err := h.DB.First(&trip, id).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}

	// 统计关联订单
	var totalCount int64
	h.DB.Model(&model.Order{}).Where("trip_id = ?", id).Count(&totalCount)

	// 零订单：直接物理删除
	if totalCount == 0 {
		h.DB.Where("trip_id = ?", id).Delete(&model.VehicleLocation{})
		if err := h.DB.Delete(&model.Trip{}, id).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "删除失败")
			return
		}
		WriteLog(c, h.DB, "班次", "删除", id, fmt.Sprintf("删除班次ID:%s (无关联订单) 物理删除", id))
		response.OKMsg(c, "删除成功", nil)
		return
	}

	// 有订单：统计活跃订单
	var pendingCount, paidCount int64
	h.DB.Model(&model.Order{}).Where("trip_id = ? AND status = 0", id).Count(&pendingCount)
	h.DB.Model(&model.Order{}).Where("trip_id = ? AND status = 1", id).Count(&paidCount)

	if !force {
		// 返回订单详情，让前端确认强制删除
		response.FailWithData(c, response.CodeServerError, "该班次存在关联订单，需确认后强制删除", map[string]int64{
			"pending_count": pendingCount,
			"paid_count":    paidCount,
			"total_count":   totalCount,
			"history_count": totalCount - pendingCount - paidCount,
		})
		return
	}

	// 强制删除流程
	// 事务内行锁班次 + 当前读订单：与下单的FOR UPDATE锁序一致，
	// 防止"快照后新支付的订单被漏掉"（置为已退款却无退款记录）与"并发下单产生孤儿订单"两类并发问题
	type refundItem struct {
		order  model.Order
		refund model.Refund
	}
	var refundItems []refundItem

	feeRate := readRefundFeeRate(h.DB)

	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		// 0. 行锁班次（与下单锁序一致，并发新订单将在此阻塞，班次删除后下单失败，不会产生孤儿订单）
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model.Trip{}, id).Error; err != nil {
			return fmt.Errorf("班次不存在或已被删除")
		}

		// 0.5 记录该班次已支付的托运订单：步骤1会把它们一并置为已取消(4)，
		//    但用户钱已付，必须走退款（否则资金直接打水漂，属严重缺陷）
		var paidCargoOrders []model.Order
		if err := tx.Where("trip_id = ? AND order_type = 2 AND status = ?", id, model.OrderStatusPaid).Find(&paidCargoOrders).Error; err != nil {
			return err
		}

		// 1. 取消活跃订单：车票待支付 + 托运全部活跃 → 已取消(4)
		if err := tx.Model(&model.Order{}).
			Where("trip_id = ? AND (order_type = 2 OR status = ?) AND status IN (?, ?)", id, model.OrderStatusPending, model.OrderStatusPending, model.OrderStatusPaid).
			Update("status", model.OrderStatusCancelled).Error; err != nil {
			return fmt.Errorf("取消订单失败: %v", err)
		}
		// 2. 车票已支付 → 已退款(3)（与用户端/管理端退款逻辑一致，保证 RollbackRefundFailure 能正确回滚）
		if err := tx.Model(&model.Order{}).
			Where("trip_id = ? AND order_type = 1 AND status = ?", id, model.OrderStatusPaid).
			Update("status", model.OrderStatusRefunded).Error; err != nil {
			return fmt.Errorf("退款状态更新失败: %v", err)
		}
		// 3. 归还被取消待支付订单绑定的优惠券（用户未乘车，券应退回）
		var pendingOrders []model.Order
		if err := tx.Where("trip_id = ? AND status = ?", id, model.OrderStatusCancelled).Find(&pendingOrders).Error; err != nil {
			return err
		}
		for _, po := range pendingOrders {
			if err := service.ReturnOrderCoupon(tx, po.ID); err != nil {
				return err
			}
		}
		// 4. 事务内当前读已置为已退款的订单（包含并发窗口内新支付的订单，杜绝"无退款记录"的资金丢失）
		var refundedOrders []model.Order
		if err := tx.Where("trip_id = ? AND order_type = 1 AND status = ?", id, model.OrderStatusRefunded).Find(&refundedOrders).Error; err != nil {
			return err
		}
		// 4.1 合并已支付托运订单（0.5 步记录的，此时已被置为已取消但钱要退）
		ordersToRefund := append(refundedOrders, paidCargoOrders...)
		// 5. 为这些订单创建退款记录（扣手续费，检查已有退款记录防重复）
		for _, o := range ordersToRefund {
			// 检查是否已存在退款记录（防止重复创建）
			var existingRefund model.Refund
			if err := tx.Where("order_id = ? AND status IN (?, ?)", o.ID, model.RefundStatusProcessing, model.RefundStatusSuccess).First(&existingRefund).Error; err == nil {
				continue // 已有处理中/成功的退款记录，跳过
			}
			// 归还已支付订单的优惠券（用户未乘车，券应退回）
			if err := service.ReturnOrderCoupon(tx, o.ID); err != nil {
				return err
			}
			refundAmount := money.CalcRefundAmount(o.TotalPrice, feeRate)
			refundRecord := model.Refund{
				OrderID:   o.ID,
				RefundNo:  sanitize.GenerateRefundNo(o.ID),
				Amount:    refundAmount,
				Reason:    "班次删除-自动退款",
				Status:    model.RefundStatusProcessing,
				PreStatus: model.OrderStatusPaid,
			}
			if err := tx.Create(&refundRecord).Error; err != nil {
				return fmt.Errorf("创建退款记录失败: %v", err)
			}
			refundItems = append(refundItems, refundItem{order: o, refund: refundRecord})
		}
		// 6. 物理删除：断开订单外键关联 → 删车辆位置 → 删班次
		//    订单保留所有冗余数据（路线、站点、日期、时间），只断开trip_id引用
		disableFKChecks(tx)
		defer func() {
			enableFKChecks(tx)
			if r := recover(); r != nil {
				panic(r) // FK检查已恢复，重抛 panic 供上层事务回滚
			}
		}()
		if err := tx.Model(&model.Order{}).Where("trip_id = ?", id).Update("trip_id", 0).Error; err != nil {
			return fmt.Errorf("断开订单关联失败: %v", err)
		}
		if err := tx.Where("trip_id = ?", id).Delete(&model.VehicleLocation{}).Error; err != nil {
			return fmt.Errorf("删除车辆位置失败: %v", err)
		}
		if err := tx.Delete(&model.Trip{}, id).Error; err != nil {
			return fmt.Errorf("删除班次失败: %v", err)
		}
		return nil
	})
	if txErr != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败: "+txErr.Error())
		return
	}

	// 3. 事务外：调用微信退款API（退款失败时回滚订单状态，与用户端/管理端逻辑一致）
	for _, item := range refundItems {
		if err := wxpay.InitiateWxRefund(h.DB, item.order, item.refund); err != nil {
			log.Printf("[ERROR] 删除班次退款失败 订单号:%s err:%v\n", item.order.OrderNo, err)
			// 退款失败回滚订单状态+退款记录标记失败（与用户端/管理端退款失败处理一致）
			service.RollbackRefundFailure(h.DB, item.order, item.refund, model.OrderStatusPaid)
		}
	}

	WriteLog(c, h.DB, "班次", "删除", id, fmt.Sprintf("删除班次ID:%s 强制删除 关联订单:共%d(待支付%d/已支付%d/历史%d) → 已物理删除，订单记录保留",
		id, totalCount, pendingCount, paidCount, totalCount-pendingCount-paidCount))
	response.OKMsg(c, "删除成功（关联订单已处理，订单记录保留）", nil)
}

type batchTripDateItem struct {
	Date            string `json:"date" binding:"required"`           // 日期 YYYY-MM-DD
	DepartureTime   string `json:"departure_time" binding:"required"` // 该日发车时间
	ArrivalTime     string `json:"arrival_time" binding:"required"`   // 该日到达时间
	ArrivalDayOffset int   `json:"arrival_day_offset"` // 到达相对发车日的天数偏移(0=当天,1=次日...)
}

type batchTripRequest struct {
	RouteID          uint                `json:"route_id" binding:"required"`
	VehicleID        uint                `json:"vehicle_id" binding:"required"`
	DriverID         uint                `json:"driver_id"`       // 司机ID（可选，批量创建时统一分配）
	DepartureTime    string              `json:"departure_time"`  // 日期范围模式用
	ArrivalTime      string              `json:"arrival_time"`    // 日期范围模式用
	ArrivalDayOffset int                 `json:"arrival_day_offset"` // 日期范围模式用：到达相对发车日的天数偏移
	BasePrice        float64             `json:"base_price" binding:"required"`
	StartDate        string              `json:"start_date"`      // 日期范围模式用
	EndDate          string              `json:"end_date"`        // 日期范围模式用
	ExcludeWeekdays  []int               `json:"exclude_weekdays"` // 排除的星期几(0=周日 1=周一 ... 6=周六)
	Force            bool                `json:"force"`           // 强制创建（已知冲突仍继续）
	TripDates        []batchTripDateItem `json:"trip_dates"`      // 自由日期模式（优先使用，各日期独立设置时间）
	// 无座票配置（批量创建的班次统一使用）
	AllowStanding    bool    `json:"allow_standing"`
	StandingQuota    int     `json:"standing_quota"`
	StandingDiscount float64 `json:"standing_discount"`
}

// BatchCreate 批量生成班次
func (h *TripHandler) BatchCreate(c *gin.Context) {
	var req batchTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 车辆和线路得存在
	var v model.Vehicle
	if err := h.DB.First(&v, req.VehicleID).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "车辆不存在")
		return
	}
	if v.Status == 0 {
		response.FailMsg(c, response.CodeParamError, "车辆处于维修状态，无法分配")
		return
	}
	var r model.Route
	if err := h.DB.First(&r, req.RouteID).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "线路不存在")
		return
	}
	if req.DriverID > 0 {
		var driver model.Driver
		if err := h.DB.First(&driver, req.DriverID).Error; err != nil {
			response.FailMsg(c, response.CodeParamError, "司机不存在")
			return
		}
		if driver.Status == 0 {
			response.FailMsg(c, response.CodeParamError, "司机已被禁用，无法分配")
			return
		}
	}
	// 无座票配置校验（批量创建）
	if req.AllowStanding && req.StandingQuota <= 0 {
		response.FailMsg(c, response.CodeParamError, "开放无座票需设置无座票配额（>0）")
		return
	}
	if req.StandingQuota < 0 {
		response.FailMsg(c, response.CodeParamError, "无座票配额不能为负数")
		return
	}
	if req.StandingDiscount < 0 || req.StandingDiscount > 1 {
		response.FailMsg(c, response.CodeParamError, "无座票价折扣需在0~1之间")
		return
	}
	// 批量创建统一的无座折扣兜底（0/留空=不强制打折，跟座位一个价）
	batchStandingDiscount := req.StandingDiscount
	if batchStandingDiscount <= 0 {
		batchStandingDiscount = 1 // 默认与座位同价（不强制打折）
	}
	if batchStandingDiscount > 1 {
		batchStandingDiscount = 1
	}

	// 构建待创建班次计划列表（统一两种模式：自由日期 / 日期范围）
	type tripPlan struct {
		dateStr          string
		departureTime    string
		arrivalTime      string
		arrivalDayOffset int
	}
	var plans []tripPlan

	if len(req.TripDates) > 0 {
		// 自由日期模式：每个日期独立设置发车/到达时间
		if len(req.TripDates) > 90 {
			response.FailMsg(c, response.CodeParamError, "日期数量不能超过90天")
			return
		}
		seen := make(map[string]bool)
		for _, td := range req.TripDates {
			if _, err := time.Parse("2006-01-02", td.Date); err != nil {
				response.FailMsg(c, response.CodeParamError, fmt.Sprintf("日期格式错误: %s", td.Date))
				return
			}
			if td.DepartureTime == "" || td.ArrivalTime == "" {
				response.FailMsg(c, response.CodeParamError, fmt.Sprintf("日期 %s 未设置发车/到达时间", td.Date))
				return
			}
			if td.ArrivalDayOffset < 0 {
				response.FailMsg(c, response.CodeParamError, fmt.Sprintf("日期 %s 的到达天数偏移不能为负数", td.Date))
				return
			}
			if td.ArrivalDayOffset == 0 && td.DepartureTime >= td.ArrivalTime {
				response.FailMsg(c, response.CodeParamError, fmt.Sprintf("日期 %s 的到达时间必须晚于发车时间，如需次日到达请选择到达天数", td.Date))
				return
			}
			if seen[td.Date] {
				continue // 去重
			}
			seen[td.Date] = true
			plans = append(plans, tripPlan{dateStr: td.Date, departureTime: td.DepartureTime, arrivalTime: td.ArrivalTime, arrivalDayOffset: td.ArrivalDayOffset})
		}
	} else {
		// 日期范围模式（向后兼容）
		if req.StartDate == "" || req.EndDate == "" || req.DepartureTime == "" || req.ArrivalTime == "" {
			response.FailMsg(c, response.CodeParamError, "请提供 trip_dates 或 start_date+end_date+departure_time+arrival_time")
			return
		}
		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			response.Fail(c, response.CodeParamError)
			return
		}
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			response.Fail(c, response.CodeParamError)
			return
		}
		if end.Sub(start).Hours()/24 > 90 {
			response.FailMsg(c, response.CodeParamError, "日期范围不能超过90天")
			return
		}
		if start.After(end) {
			response.FailMsg(c, response.CodeParamError, "开始日期不能晚于结束日期")
			return
		}
		if req.ArrivalDayOffset < 0 {
			response.FailMsg(c, response.CodeParamError, "到达天数偏移不能为负数")
			return
		}
		if req.ArrivalDayOffset == 0 && req.DepartureTime >= req.ArrivalTime {
			response.FailMsg(c, response.CodeParamError, "当天到达时间必须晚于发车时间，如需次日到达请选择到达天数")
			return
		}
		excludeSet := make(map[int]bool)
		for _, wd := range req.ExcludeWeekdays {
			if wd >= 0 && wd <= 6 {
				excludeSet[wd] = true
			}
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			if excludeSet[int(d.Weekday())] {
				continue
			}
			plans = append(plans, tripPlan{
				dateStr:          d.Format("2006-01-02"),
				departureTime:    req.DepartureTime,
				arrivalTime:      req.ArrivalTime,
				arrivalDayOffset: req.ArrivalDayOffset,
			})
		}
	}

	// 过滤过去日期/已过发车时刻：只允许创建"今天还没发车"及以后的班次
	// 要不建出来个今天上午的车，下午才创建，用户能搜到却坐不上，白搭
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	skippedCount := 0
	pastDates := []string{}
	filteredPlans := plans[:0]
	for _, p := range plans {
		if p.dateStr < todayStr {
			pastDates = append(pastDates, p.dateStr)
			skippedCount++
			continue
		}
		// 今天的班次：发车时刻已经过了的也不要，免得"能买不能坐"
		if p.dateStr == todayStr {
			dep, err := time.ParseInLocation("2006-01-02 15:04", todayStr+" "+p.departureTime, time.Local)
			if err == nil && !dep.After(now) {
				pastDates = append(pastDates, p.dateStr+"(已过发车时刻)")
				skippedCount++
				continue
			}
		}
		filteredPlans = append(filteredPlans, p)
	}
	plans = filteredPlans

	// 统一创建逻辑：遍历计划列表，检查重复/冲突，构建班次
	var trips []model.Trip
	conflictDates := []string{}
	existDates := []string{}
	for _, p := range plans {
		var existCount int64
		h.DB.Model(&model.Trip{}).Where("route_id = ? AND trip_date = ? AND departure_time = ?",
			req.RouteID, p.dateStr, p.departureTime).Count(&existCount)
		if existCount > 0 {
			existDates = append(existDates, p.dateStr)
			skippedCount++
			continue
		}
		conflicts := service.CheckTripConflict(h.DB, service.TripConflictCheck{
			ExcludeTripID:   0,
			DriverID:        req.DriverID,
			VehicleID:       req.VehicleID,
			TripDate:        p.dateStr,
			DepartureTime:   p.departureTime,
			ArrivalTime:     p.arrivalTime,
			ArrivalDayOffset: p.arrivalDayOffset,
			FromStationID:   r.FromStationID,
			ToStationID:     r.ToStationID,
		})
		// 硬冲突（时间重叠/车辆占用）不可强制创建，跳过
		if service.HasHardConflict(conflicts) {
			conflictDates = append(conflictDates, p.dateStr)
			skippedCount++
			continue
		}
		// 位置断层等软冲突不阻断批量生成：批量排班跨站属正常场景，
		// 若也拦截会导致"本月全部"被静默跳过大量日期而用户不知原因
		_ = conflicts
		trips = append(trips, model.Trip{
			RouteID:          req.RouteID, VehicleID: req.VehicleID, DriverID: req.DriverID,
			TripNo:           fmt.Sprintf("TC%d%s%s", req.RouteID, p.dateStr, p.departureTime),
			TripDate:         model.JSONDate(p.dateStr),
			DepartureTime:    p.departureTime, ArrivalTime: p.arrivalTime,
			ArrivalDayOffset: p.arrivalDayOffset,
			TotalSeats:       v.SeatCount, AvailableSeats: v.SeatCount,
			BasePrice:        req.BasePrice, Status: 1,
			AllowStanding:    req.AllowStanding,
			StandingQuota:    req.StandingQuota,
			StandingDiscount: batchStandingDiscount,
		})
	}
	if len(conflictDates) > 0 && len(trips) == 0 {
		response.FailMsg(c, response.CodeServerError, fmt.Sprintf("所有日期均存在冲突，未创建任何班次。冲突日期: %s（可使用force=true强制创建）", strings.Join(conflictDates, ", ")))
		return
	}
	if len(trips) == 0 && len(pastDates) > 0 {
		response.FailMsg(c, response.CodeParamError, fmt.Sprintf("所选日期均早于今天(%s)，未创建任何班次", todayStr))
		return
	}
	if len(trips) > 0 {
		if err := h.DB.Create(&trips).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "批量创建失败")
			return
		}
	}
	// 组装跳过原因明细（让管理员一眼看出哪些日期为什么没生成，避免"只生成了3个"的困惑）
	reasonParts := []string{}
	if len(pastDates) > 0 {
		reasonParts = append(reasonParts, fmt.Sprintf("过去日期%d个", len(pastDates)))
	}
	if len(existDates) > 0 {
		reasonParts = append(reasonParts, fmt.Sprintf("该线路同日同时段已存在%d个", len(existDates)))
	}
	if len(conflictDates) > 0 {
		reasonParts = append(reasonParts, fmt.Sprintf("车辆/司机时间冲突%d个", len(conflictDates)))
	}
	msg := fmt.Sprintf("成功创建%d个班次", len(trips))
	if len(reasonParts) > 0 {
		msg += fmt.Sprintf("，跳过%d个日期（%s）", skippedCount, strings.Join(reasonParts, "、"))
	}
	if len(conflictDates) > 0 {
		showDates := conflictDates
		if len(showDates) > 5 {
			showDates = showDates[:5]
		}
		msg += fmt.Sprintf("。冲突日期如：%s，请更换该时段空闲的车辆/司机或调整发车时间后重试", strings.Join(showDates, "、"))
	}
	WriteLog(c, h.DB, "班次", "批量创建", fmt.Sprintf("线路ID:%d", req.RouteID), fmt.Sprintf("批量创建%d个班次(跳过%d天: 冲突%d/过去%d/已存在%d)", len(trips), skippedCount, len(conflictDates), len(pastDates), len(existDates)))
	response.OKMsg(c, msg, trips)
}

// 历史班次清理

type cleanupHistoryRequest struct {
	BeforeDate string `json:"before_date"` // 删除此日期之前（不含当天）的班次，默认今天
	RouteID    uint   `json:"route_id"`    // 0=所有线路，>0=仅指定线路
	Force      bool   `json:"force"`       // false=预览（返回统计），true=执行删除
}

// CleanupHistory 清理历史班次
// 两阶段：force=false 返回预览统计，force=true 执行物理删除（保留订单冗余数据，活跃订单自动取消/退款）
// 仅清理 status IN (0,1,3,4) 的历史班次，status=2(已发车未结束) 不处理并提示
func (h *TripHandler) CleanupHistory(c *gin.Context) {
	var req cleanupHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	beforeDate := req.BeforeDate
	if beforeDate == "" {
		beforeDate = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", beforeDate); err != nil {
		response.FailMsg(c, response.CodeParamError, "日期格式错误，需 YYYY-MM-DD")
		return
	}

	// 构建查询：截止日期之前 且 非已发车状态(status!=2)
	baseQuery := h.DB.Model(&model.Trip{}).Where("trip_date < ? AND status != 2", beforeDate)
	if req.RouteID > 0 {
		baseQuery = baseQuery.Where("route_id = ?", req.RouteID)
	}

	// 找待清理的班次
	var trips []model.Trip
	baseQuery.Order("trip_date DESC").Find(&trips)

	if len(trips) == 0 {
		response.OKMsg(c, "没有需要清理的历史班次", nil)
		return
	}

	tripIDs := make([]uint, len(trips))
	for i, t := range trips {
		tripIDs[i] = t.ID
	}

	// 统计关联订单
	var totalCount, pendingCount, paidCount int64
	h.DB.Model(&model.Order{}).Where("trip_id IN ?", tripIDs).Count(&totalCount)
	h.DB.Model(&model.Order{}).Where("trip_id IN ? AND status = 0", tripIDs).Count(&pendingCount)
	h.DB.Model(&model.Order{}).Where("trip_id IN ? AND status = 1", tripIDs).Count(&paidCount)

	// 统计各状态分布（每次独立查询，避免 GORM 链式 WHERE 累积）
	countByStatus := func(status int8) int64 {
		var count int64
		q := h.DB.Model(&model.Trip{}).Where("trip_date < ? AND status = ?", beforeDate, status)
		if req.RouteID > 0 {
			q = q.Where("route_id = ?", req.RouteID)
		}
		q.Count(&count)
		return count
	}
	statusOffline := countByStatus(0)   // 下架
	statusAvailable := countByStatus(1) // 可售（过期未售）
	statusCancelled := countByStatus(3) // 已取消
	statusCompleted := countByStatus(4) // 已完成
	statusDeparted := countByStatus(2)  // 已发车未结束（不清理，仅提示）

	// 预览模式：返回统计信息，不执行删除
	if !req.Force {
		response.OK(c, gin.H{
			"trip_count":      len(trips),
			"before_date":     beforeDate,
			"total_orders":    totalCount,
			"pending_orders":  pendingCount,
			"paid_orders":      paidCount,
			"history_orders":  totalCount - pendingCount - paidCount,
			"has_active":      pendingCount > 0 || paidCount > 0,
			"status_breakdown": gin.H{
				"offline":   statusOffline,
				"available": statusAvailable,
				"cancelled": statusCancelled,
				"completed": statusCompleted,
				"departed":  statusDeparted, // 未清理的已发车班次数
			},
		})
		return
	}

	// 执行模式：事务内批量删除
	// 事务内当前读订单（与强制删除单班次逻辑一致，防止快照与事务之间的并发支付订单无退款记录）
	type refundItem struct {
		order  model.Order
		refund model.Refund
	}
	var refundItems []refundItem
	feeRate := readRefundFeeRate(h.DB)

	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		// 记录已支付托运订单：下面的"托运→已取消"会误伤它们，但钱已付必须退
		var paidCargoOrders []model.Order
		if err := tx.Where("trip_id IN ? AND order_type = 2 AND status = ?", tripIDs, model.OrderStatusPaid).Find(&paidCargoOrders).Error; err != nil {
			return err
		}
		// 取消所有活跃订单：车票待支付→已取消(4)，车票已支付→已退款(3)，托运→已取消(4)
		if err := tx.Model(&model.Order{}).
			Where("trip_id IN ? AND (order_type = 2 OR status = ?) AND status IN (?, ?)", tripIDs, model.OrderStatusPending, model.OrderStatusPending, model.OrderStatusPaid).
			Update("status", model.OrderStatusCancelled).Error; err != nil {
			return fmt.Errorf("取消活跃订单失败: %v", err)
		}
		// 车票已支付 → 已退款(3)
		if err := tx.Model(&model.Order{}).
			Where("trip_id IN ? AND order_type = 1 AND status = ?", tripIDs, model.OrderStatusPaid).
			Update("status", model.OrderStatusRefunded).Error; err != nil {
			return fmt.Errorf("退款状态更新失败: %v", err)
		}
		// 归还被取消待支付订单绑定的优惠券
		var pendingOrders []model.Order
		if err := tx.Where("trip_id IN ? AND status = ?", tripIDs, model.OrderStatusCancelled).Find(&pendingOrders).Error; err != nil {
			return err
		}
		for _, po := range pendingOrders {
			if err := service.ReturnOrderCoupon(tx, po.ID); err != nil {
				return err
			}
		}
		// 事务内当前读已置为已退款的订单，为其创建退款记录（检查已有退款记录防重复）
		var refundedOrders []model.Order
		if err := tx.Where("trip_id IN ? AND order_type = 1 AND status = ?", tripIDs, model.OrderStatusRefunded).Find(&refundedOrders).Error; err != nil {
			return err
		}
		// 合并已支付托运订单（钱已付，一并退款）
		ordersToRefund := append(refundedOrders, paidCargoOrders...)
		for _, o := range ordersToRefund {
			var existingRefund model.Refund
			if err := tx.Where("order_id = ? AND status IN (?, ?)", o.ID, model.RefundStatusProcessing, model.RefundStatusSuccess).First(&existingRefund).Error; err == nil {
				continue // 已有处理中/成功的退款记录，跳过
			}
			// 归还已支付订单的优惠券（用户未乘车，券应退回）
			if err := service.ReturnOrderCoupon(tx, o.ID); err != nil {
				return err
			}
			refundAmount := money.CalcRefundAmount(o.TotalPrice, feeRate)
			refundRecord := model.Refund{
				OrderID:   o.ID,
				RefundNo:  sanitize.GenerateRefundNo(o.ID),
				Amount:    refundAmount,
				Reason:    "历史班次清理-自动退款",
				Status:    model.RefundStatusProcessing,
				PreStatus: model.OrderStatusPaid,
			}
			if err := tx.Create(&refundRecord).Error; err != nil {
				return fmt.Errorf("创建退款记录失败: %v", err)
			}
			refundItems = append(refundItems, refundItem{order: o, refund: refundRecord})
		}
		// 物理删除：断开订单外键关联 → 删车辆位置 → 删班次
		disableFKChecks(tx)
		defer func() {
			enableFKChecks(tx)
			if r := recover(); r != nil {
				panic(r)
			}
		}()
		if err := tx.Model(&model.Order{}).Where("trip_id IN ?", tripIDs).Update("trip_id", 0).Error; err != nil {
			return fmt.Errorf("断开订单关联失败: %v", err)
		}
		if err := tx.Where("trip_id IN ?", tripIDs).Delete(&model.VehicleLocation{}).Error; err != nil {
			return fmt.Errorf("删除车辆位置失败: %v", err)
		}
		if err := tx.Where("id IN ?", tripIDs).Delete(&model.Trip{}).Error; err != nil {
			return fmt.Errorf("删除班次失败: %v", err)
		}
		return nil
	})
	if txErr != nil {
		response.FailMsg(c, response.CodeServerError, "清理失败: "+txErr.Error())
		return
	}

	// 事务外：调用微信退款API（退款失败时回滚订单状态，与用户端/管理端逻辑一致）
	for _, item := range refundItems {
		if err := wxpay.InitiateWxRefund(h.DB, item.order, item.refund); err != nil {
			log.Printf("[ERROR] 历史班次清理退款失败 订单号:%s err:%v\n", item.order.OrderNo, err)
			service.RollbackRefundFailure(h.DB, item.order, item.refund, model.OrderStatusPaid)
		}
	}

	routeNote := "全部线路"
	if req.RouteID > 0 {
		routeNote = fmt.Sprintf("线路ID:%d", req.RouteID)
	}
	WriteLog(c, h.DB, "班次", "清理历史", beforeDate, fmt.Sprintf("清理历史班次:截止%s %s 共删除%d个班次 关联订单:共%d(待支付%d/已支付%d/历史%d) 退款%d笔",
		beforeDate, routeNote, len(trips), totalCount, pendingCount, paidCount, totalCount-pendingCount-paidCount, len(refundItems)))
	response.OKMsg(c, fmt.Sprintf("成功清理%d个历史班次", len(trips)), gin.H{
		"deleted_count": len(trips),
		"refund_count":  len(refundItems),
	})
}
