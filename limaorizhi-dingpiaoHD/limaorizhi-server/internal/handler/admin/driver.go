// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/idcard"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/sanitize"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// phonePattern 中国大陆手机号：11位数字、1开头（第二位3-9）
var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// validatePhone 校验手机号：必须完整（不含*脱敏）、11位数字、1开头
// 防止后台录入脱敏手机号导致 is_driver 匹配失败、司机无法登录
func validatePhone(phone string) error {
	if strings.Contains(phone, "*") {
		return fmt.Errorf("手机号不能为脱敏格式（含*），请输入完整手机号")
	}
	if !phonePattern.MatchString(phone) {
		return fmt.Errorf("手机号格式不正确，请输入完整的11位手机号")
	}
	return nil
}

// 司机管理

type DriverHandler struct{ DB *gorm.DB }

func NewDriverHandler(db *gorm.DB) *DriverHandler { return &DriverHandler{DB: db} }

// List 司机列表
func (h *DriverHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := h.DB.Model(&model.Driver{})
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR phone LIKE ?", "%"+sanitize.EscapeLikePattern(keyword)+"%", "%"+sanitize.EscapeLikePattern(keyword)+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	var list []model.Driver
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	// 管理端不脱敏手机号：管理员需要完整手机号联系司机、编辑司机信息
	response.Page(c, list, total, page, pageSize)
}

// All 获取所有启用的司机（用于下拉选择）
func (h *DriverHandler) All(c *gin.Context) {
	var list []model.Driver
	h.DB.Where("status = 1").Order("name ASC").Find(&list)
	// 管理端不脱敏手机号：管理员需要完整手机号分配班次
	response.OK(c, list)
}

type createDriverRequest struct {
	Name       string `json:"name" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Password   string `json:"password" binding:"required"`
	LicenseNo  string `json:"license_no"`
	EmployeeNo string `json:"employee_no"` // 工号（可选，用于区分重名司机）
}

// Create 新增司机
func (h *DriverHandler) Create(c *gin.Context) {
	var req createDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 手机号格式校验：必须完整（不含*脱敏）、11位数字、1开头
	if err := validatePhone(req.Phone); err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}

	var count int64
	h.DB.Model(&model.Driver{}).Where("phone = ?", req.Phone).Count(&count)
	if count > 0 {
		response.FailMsg(c, response.CodeServerError, "手机号已存在")
		return
	}

	if err := validatePasswordStrength(req.Password); err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "密码加密失败")
		return
	}
	driver := model.Driver{
		Name:         req.Name,
		Phone:        req.Phone,
		PasswordHash: string(hash),
		LicenseNo:    req.LicenseNo,
		EmployeeNo:   req.EmployeeNo,
		Status:       1,
	}
	if err := h.DB.Create(&driver).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	WriteLog(c, h.DB, "司机", "新增", driver.Name, fmt.Sprintf("司机ID:%d 工号:%s 姓名:%s 手机:%s", driver.ID, driver.EmployeeNo, driver.Name, driver.Phone))
	response.OK(c, driver)
}

type updateDriverRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	LicenseNo  string `json:"license_no"`
	EmployeeNo string `json:"employee_no"` // 工号
	Status     int8   `json:"status"`
	Password   string `json:"password"` // 为空则不修改密码
}

// Update 更新司机
func (h *DriverHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req updateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 手机号校验：非空时必须完整且格式合法，防止脱敏格式覆盖真实手机号
	if req.Phone != "" {
		if err := validatePhone(req.Phone); err != nil {
			response.FailMsg(c, response.CodeParamError, err.Error())
			return
		}
		var count int64
		h.DB.Model(&model.Driver{}).Where("phone = ? AND id != ?", req.Phone, id).Count(&count)
		if count > 0 {
			response.FailMsg(c, response.CodeServerError, "手机号已被其他司机使用")
			return
		}
	}

	updates := map[string]interface{}{
		"name":        req.Name,
		"license_no":  req.LicenseNo,
		"employee_no": req.EmployeeNo,
		"status":      req.Status,
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	// 如果提供了新密码则更新
	if req.Password != "" {
		if err := validatePasswordStrength(req.Password); err != nil {
			response.FailMsg(c, response.CodeParamError, err.Error())
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			response.FailMsg(c, response.CodeServerError, "密码加密失败")
			return
		}
		updates["password_hash"] = string(hash)
	}
	if err := h.DB.Model(&model.Driver{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	WriteLog(c, h.DB, "司机", "编辑", id, fmt.Sprintf("司机ID:%s 姓名:%s", id, req.Name))
	response.OKMsg(c, "更新成功", nil)
}

// Delete 删除司机
func (h *DriverHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	// 删除前检查是否有未发车的活跃班次
	today := time.Now().Format("2006-01-02")
	var activeTripCount int64
	h.DB.Model(&model.Trip{}).Where("driver_id = ? AND status = 1 AND trip_date >= ?", id, today).Count(&activeTripCount)
	if activeTripCount > 0 {
		response.FailMsg(c, response.CodeServerError, fmt.Sprintf("该司机有%d个未发车的活跃班次，请先取消班次司机分配再删除", activeTripCount))
		return
	}
	// 使用事务保证原子性
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Trip{}).Where("driver_id = ?", id).Update("driver_id", 0).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Driver{}, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}
	WriteLog(c, h.DB, "司机", "删除", id, "删除司机ID:"+id)
	response.OKMsg(c, "删除成功", nil)
}

// TripPassengers 班次乘客名单（含核销状态）
func (h *DriverHandler) TripPassengers(c *gin.Context) {
	tripID := c.Param("id")
	var passengers []model.OrderPassenger
	h.DB.Joins("JOIN orders ON orders.id = order_passengers.order_id").
		Preload("Order.FromStation").
		Preload("Order.ToStation").
		Where("orders.trip_id = ? AND orders.status = 1", tripID).
		Order("order_passengers.seat_no ASC").
		Find(&passengers)
	// 脱敏：隐藏乘客身份证号和手机号中间部分
	for i := range passengers {
		passengers[i].IDCardNo = idcard.MaskIDCard(passengers[i].IDCardNo)
		passengers[i].Phone = idcard.MaskPhone(passengers[i].Phone)
	}
	response.OK(c, passengers)
}

// AssignDriver 分配司机到班次
// 增强逻辑：检测司机时间冲突（同一日期有未结束或时间重叠的班次）
// 管理员可通过 force=true 强制分配（员工少时可灵活决定）
type assignDriverRequest struct {
	DriverID uint `json:"driver_id"`
	Force    bool `json:"force"` // 强制分配（已知冲突仍继续）
}

func (h *DriverHandler) AssignDriver(c *gin.Context) {
	tripID := c.Param("id")
	var req assignDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 校验班次状态，已取消/已下架不允许分配司机
	var trip model.Trip
	if err := h.DB.Preload("Route.FromStation").Preload("Route.ToStation").First(&trip, tripID).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "班次不存在")
		return
	}
	if trip.Status == 3 || trip.Status == 0 {
		response.FailMsg(c, response.CodeServerError, "已取消或已下架的班次不能分配司机")
		return
	}

	// 清空司机时无需检测冲突
	if req.DriverID == 0 {
		if err := h.DB.Model(&model.Trip{}).Where("id = ?", tripID).Update("driver_id", 0).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "取消司机分配失败")
			return
		}
		WriteLog(c, h.DB, "班次", "取消司机分配", tripID, fmt.Sprintf("班次ID:%s", tripID))
		response.OKMsg(c, "已取消司机分配", nil)
		return
	}

	// 使用事务+行锁保护冲突检测和分配
	var conflicts []service.ConflictInfo
	var driver model.Driver
	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		// 行锁锁定班次，防止并发修改
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Route.FromStation").Preload("Route.ToStation").First(&trip, tripID).Error; err != nil {
			return fmt.Errorf("班次不存在")
		}
		if trip.Status == 3 || trip.Status == 0 {
			return fmt.Errorf("已取消或已下架的班次不能分配司机")
		}

		// 获取起终站ID（用于位置断层检测）
		var fromStationID, toStationID uint
		if trip.Route != nil {
			fromStationID = trip.Route.FromStationID
			toStationID = trip.Route.ToStationID
		}

		// 调用共享冲突检测服务（使用事务连接，确保检测数据一致性）
		tripIDUint, _ := strconv.ParseUint(tripID, 10, 64)
		conflicts = service.CheckTripConflict(tx, service.TripConflictCheck{
			ExcludeTripID:  uint(tripIDUint),
			DriverID:       req.DriverID,
			VehicleID:      trip.VehicleID,
			TripDate:       string(trip.TripDate),
			DepartureTime:  trip.DepartureTime,
			ArrivalTime:    trip.ArrivalTime,
			FromStationID:  fromStationID,
			ToStationID:   toStationID,
		})

		// 硬冲突（时间重叠/车辆冲突）不可强制分配
		if service.HasHardConflict(conflicts) {
			return nil // 不更新，外层返回错误
		}
		// 有冲突且未强制分配时返回冲突信息
		if len(conflicts) > 0 && !req.Force {
			return nil // 不更新，外层返回冲突信息
		}

		// 校验司机状态（行锁内）
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&driver, req.DriverID).Error; err != nil {
			return fmt.Errorf("司机不存在")
		}
		if driver.Status == 0 {
			return fmt.Errorf("司机已被禁用，无法分配")
		}

		// 执行分配
		if err := tx.Model(&model.Trip{}).Where("id = ?", tripID).Update("driver_id", req.DriverID).Error; err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		response.FailMsg(c, response.CodeServerError, txErr.Error())
		return
	}

	// 硬冲突（时间重叠/车辆冲突）不可强制分配
	if service.HasHardConflict(conflicts) {
		msg := service.ConflictSummary(conflicts) + "（时间重叠/车辆冲突不可强制分配）"
		response.FailWithData(c, response.CodeServerError, msg, gin.H{"conflicts": conflicts})
		return
	}
	// 有冲突且未强制分配时返回冲突信息供管理员确认
	if len(conflicts) > 0 && !req.Force {
		msg := service.ConflictSummary(conflicts)
		response.FailWithData(c, response.CodeServerError, msg, gin.H{"conflicts": conflicts})
		return
	}

	// 查询司机信息用于日志
driverName := fmt.Sprintf("driver#%d", req.DriverID)
	if req.DriverID > 0 {
		h.DB.First(&driver, req.DriverID)
		if driver.Name != "" {
			driverName = driver.Name
			if driver.EmployeeNo != "" {
				driverName = fmt.Sprintf("%s(工号:%s)", driver.Name, driver.EmployeeNo)
			}
		}
	}
	forceNote := ""
	if len(conflicts) > 0 && req.Force {
		forceNote = fmt.Sprintf(" [强制分配，已覆盖%d个冲突]", len(conflicts))
	}
	WriteLog(c, h.DB, "班次", "分配司机", tripID, fmt.Sprintf("班次ID:%s 司机:%s%s", tripID, driverName, forceNote))
	response.OKMsg(c, "分配司机成功", nil)
}

// DriverAvailability 查询司机可用性（分配班次前调用，返回该司机同日已有班次、行程链及冲突信息）
// 行程链：将司机当日所有班次按时间排序，显示起点站→终点站的站点流转链
// 位置断层：前序班次终点站 ≠ 后续班次起点站时标记
func (h *DriverHandler) DriverAvailability(c *gin.Context) {
	driverIDStr := c.Query("driver_id")
	tripDate := c.Query("trip_date")
	departureTime := c.Query("departure_time")
	arrivalTime := c.Query("arrival_time")
	excludeTripID := c.Query("exclude_trip_id")
	fromStationID := c.Query("from_station_id")
	toStationID := c.Query("to_station_id")

	if driverIDStr == "" || tripDate == "" {
		response.FailMsg(c, response.CodeParamError, "缺少 driver_id 或 trip_date 参数")
		return
	}

	driverID, _ := strconv.ParseUint(driverIDStr, 10, 64)
	if driverID == 0 {
		response.FailMsg(c, response.CodeParamError, "driver_id 无效")
		return
	}

	// 查询该司机在此日期的所有活跃班次（按发车时间排序）
	var trips []model.Trip
	query := h.DB.Preload("Route.FromStation").Preload("Route.ToStation").
		Where("driver_id = ? AND trip_date = ? AND status IN (1, 2)", driverID, tripDate)
	if excludeTripID != "" {
		query = query.Where("id != ?", excludeTripID)
	}
	query.Order("departure_time ASC").Find(&trips)

	type tripInfo struct {
		TripID         uint   `json:"trip_id"`
		TripNo         string `json:"trip_no"`
		RouteName      string `json:"route_name"`
		FromStation    string `json:"from_station"`
		ToStation      string `json:"to_station"`
		DepartureTime  string `json:"departure_time"`
		ArrivalTime    string `json:"arrival_time"`
		Status         int8   `json:"status"`
		StatusText     string `json:"status_text"`
		ConflictType   string `json:"conflict_type"`   // time_overlap / location_gap / ""
		ConflictDesc   string `json:"conflict_desc"`
	}

	// 待分配班次的起点站和终点站
	newFromStation := ""
	newToStation := ""
	if departureTime != "" && arrivalTime != "" {
		// 优先用 from/to_station_id 查站点名
		if fromStationID != "" {
			var fs model.Station
			if err := h.DB.First(&fs, fromStationID).Error; err == nil { newFromStation = fs.Name }
		}
		if toStationID != "" {
			var ts model.Station
			if err := h.DB.First(&ts, toStationID).Error; err == nil { newToStation = ts.Name }
		}
		// 如果未传入站点ID，备选从 exclude_trip_id 反查线路站点
		if newFromStation == "" && newToStation == "" && excludeTripID != "" {
			var exTrip model.Trip
			h.DB.Preload("Route.FromStation").Preload("Route.ToStation").First(&exTrip, excludeTripID)
			if exTrip.Route != nil {
				if exTrip.Route.FromStation != nil { newFromStation = exTrip.Route.FromStation.Name }
				if exTrip.Route.ToStation != nil { newToStation = exTrip.Route.ToStation.Name }
			}
		}
	}

	var infoList []tripInfo
	for _, t := range trips {
		var statusText string
		if t.Status == 2 {
			statusText = "已发车(未结束)"
		} else {
			statusText = "可售(待发车)"
		}

		var routeName, fromStation, toStation string
		if t.Route != nil {
			if t.Route.FromStation != nil {
				fromStation = t.Route.FromStation.Name
			}
			if t.Route.ToStation != nil {
				toStation = t.Route.ToStation.Name
			}
			if fromStation != "" && toStation != "" {
				routeName = fromStation + " → " + toStation
			}
		}

		conflictType := ""
		conflictDesc := ""

		// 如果提供了待分配班次的发车/到达时间，检测冲突
		if departureTime != "" && arrivalTime != "" {
			// 1. 时间重叠检测
			if departureTime < t.ArrivalTime && arrivalTime > t.DepartureTime {
				conflictType = "time_overlap"
				conflictDesc = fmt.Sprintf("时间重叠：待分配班次(%s-%s)与此班次(%s-%s)时间冲突", departureTime, arrivalTime, t.DepartureTime, t.ArrivalTime)
			} else {
				// 2. 位置断层检测
				if t.ArrivalTime <= departureTime {
					// 已有班次在新班次之前：司机在toStation，新班次从newFromStation出发
					if toStation != "" && newFromStation != "" && toStation != newFromStation {
						conflictType = "location_gap"
						conflictDesc = fmt.Sprintf("位置断层：此班次终点[%s] ≠ 待分配班次起点[%s]", toStation, newFromStation)
					}
				} else if arrivalTime <= t.DepartureTime {
					// 新班次在已有班次之前：司机在newToStation，已有班次从fromStation出发
					if newToStation != "" && fromStation != "" && newToStation != fromStation {
						conflictType = "location_gap"
						conflictDesc = fmt.Sprintf("位置断层：待分配班次终点[%s] ≠ 此班次起点[%s]", newToStation, fromStation)
					}
				}
			}
		}

		infoList = append(infoList, tripInfo{
			TripID:         t.ID,
			TripNo:         t.TripNo,
			RouteName:      routeName,
			FromStation:    fromStation,
			ToStation:      toStation,
			DepartureTime:  t.DepartureTime,
			ArrivalTime:    t.ArrivalTime,
			Status:         t.Status,
			StatusText:     statusText,
			ConflictType:   conflictType,
			ConflictDesc:   conflictDesc,
		})
	}

	// 查询司机基本信息
	var driver model.Driver
	h.DB.First(&driver, driverID)

	// 构建行程链描述（站点流转）
	chainParts := []string{}
	for _, t := range trips {
		fs, ts := "", ""
		if t.Route != nil {
			if t.Route.FromStation != nil { fs = t.Route.FromStation.Name }
			if t.Route.ToStation != nil { ts = t.Route.ToStation.Name }
		}
		if fs != "" && ts != "" {
			chainParts = append(chainParts, fmt.Sprintf("%s(%s→%s)", t.DepartureTime, fs, ts))
		}
	}
	chainDesc := strings.Join(chainParts, " → ")

	// 司机当前所在站点（最后一个班次的终点站）
	currentLocation := ""
	if len(trips) > 0 {
		lastTrip := trips[len(trips)-1]
		if lastTrip.Route != nil && lastTrip.Route.ToStation != nil {
			currentLocation = lastTrip.Route.ToStation.Name
		}
	}

	// has_conflict 只在有真正冲突时为true
	hasConflict := false
	for _, t := range infoList {
		if t.ConflictType != "" { hasConflict = true; break }
	}

	response.OK(c, gin.H{
		"driver": gin.H{
			"id":              driver.ID,
			"name":            driver.Name,
			"employee_no":     driver.EmployeeNo,
			"phone":           driver.Phone,
			"status":          driver.Status,
			"current_location": currentLocation,
		},
		"trips":           infoList,
		"chain_desc":      chainDesc,
		"has_conflict":    hasConflict,
	})
}

// VerifyStats 核销统计
func (h *DriverHandler) VerifyStats(c *gin.Context) {
	tripID := c.Query("trip_id")
	var total, checked int64

	baseQuery := h.DB.Model(&model.OrderPassenger{}).
		Joins("JOIN orders ON orders.id = order_passengers.order_id").
		Where("orders.status = 1")
	if tripID != "" {
		baseQuery = baseQuery.Where("orders.trip_id = ?", tripID)
	}

	// 使用 gorm.Session{} 确保两次 Count 不会叠加条件
	baseQuery.Session(&gorm.Session{}).Count(&total)
	baseQuery.Session(&gorm.Session{}).
		Where("order_passengers.check_status = 1").Count(&checked)

	response.OK(c, gin.H{
		"total":     total,
		"checked":   checked,
		"unchecked": total - checked,
	})
}
