// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"strconv"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/sanitize"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)
// 线路管理

type RouteHandler struct {
	DB *gorm.DB
}

func NewRouteHandler(db *gorm.DB) *RouteHandler {
	return &RouteHandler{DB: db}
}

// All 返回全部线路（不分页，仅供管理后台下拉选择/线路总览用，预加载首末站）
func (h *RouteHandler) All(c *gin.Context) {
	var list []model.Route
	h.DB.Preload("FromStation").Preload("ToStation").
		Order("id DESC").Find(&list)
	response.OK(c, list)
}

func (h *RouteHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.DB.Model(&model.Route{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+sanitize.EscapeLikePattern(name)+"%")
	}
	query.Count(&total)

	var list []model.Route
	query.Preload("FromStation").Preload("ToStation").
		Preload("RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("RouteStations.Station").
		Order("id DESC"). // id DESC：新建路线（含反向路线）显示在第一页
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

// routeStationInput 线路站点序列输入（按数组顺序即站点顺序）
type routeStationInput struct {
	StationID        uint    `json:"station_id" binding:"required"`
	DistanceKM       float64 `json:"distance_km"` // 从起点到该站累计里程
	Price            float64 `json:"price"`       // 从起点到该站票价
	ArrivalTime      string  `json:"arrival_time"` // 该站到达时刻(如08:35)，发车后按站下单判断；空则走里程推算
	ArrivalDayOffset int     `json:"arrival_day_offset"` // 该站到达相对发车日的天数偏移(0=当天,1=次日...)
}

// createRouteRequest 线路创建请求（DTO白名单）
type createRouteRequest struct {
	Name            string              `json:"name" binding:"required"`
	RouteType       int8                `json:"route_type"` // 1=城乡公交 2=城际客运 3=旅游专线
	FromStationID   uint                `json:"from_station_id"` // 可选，兼容旧前端；优先用 stations 派生
	ToStationID     uint                `json:"to_station_id"`
	DistanceKM      float64             `json:"distance_km"`
	DurationMinutes int                 `json:"duration_minutes"`
	MinFare         float64             `json:"min_fare"` // 起步价（最低票价），0=不启用
	Status          int8                `json:"status"`
	Stations        []routeStationInput `json:"stations"` // 有序站点序列（至少2站）
}

func (h *RouteHandler) Create(c *gin.Context) {
	var req createRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.Name == "" {
		response.FailMsg(c, response.CodeParamError, "线路名称不能为空")
		return
	}
	// 站点序列校验：若提供 stations，至少2站且站点不重复
	if len(req.Stations) > 0 {
		if len(req.Stations) < 2 {
			response.FailMsg(c, response.CodeParamError, "线路至少需要2个站点")
			return
		}
		seen := make(map[uint]bool)
		for _, s := range req.Stations {
			if s.StationID == 0 {
				response.FailMsg(c, response.CodeParamError, "站点不能为空")
				return
			}
			if seen[s.StationID] {
				response.FailMsg(c, response.CodeParamError, "线路站点不能重复")
				return
			}
			seen[s.StationID] = true
		}
		// 票价单调递增校验（累计票价制：每站Price = 从起点到该站的累计票价，必须单调不减）
		for i := 1; i < len(req.Stations); i++ {
			if req.Stations[i].Price < req.Stations[i-1].Price {
				response.FailMsg(c, response.CodeParamError,
					fmt.Sprintf("站点票价必须单调递增（第%d站票价%.2f低于第%d站%.2f）",
						i+1, req.Stations[i].Price, i, req.Stations[i-1].Price))
				return
			}
		}
	} else if req.FromStationID == 0 || req.ToStationID == 0 {
		// 兼容旧前端：未传 stations 时必须传 from/to
		response.FailMsg(c, response.CodeParamError, "请填写站点序列")
		return
	}

	// 派生首末站与全程里程
	var fromID, toID uint
	var totalDist float64
	if len(req.Stations) > 0 {
		fromID = req.Stations[0].StationID
		toID = req.Stations[len(req.Stations)-1].StationID
		totalDist = req.Stations[len(req.Stations)-1].DistanceKM
	} else {
		fromID = req.FromStationID
		toID = req.ToStationID
		totalDist = req.DistanceKM
	}
	if fromID == toID {
		response.FailMsg(c, response.CodeParamError, "出发站和到达站不能相同")
		return
	}

	// 线路类型默认城乡公交
	routeType := req.RouteType
	if routeType == 0 {
		routeType = 1
	}

	// 事务：建 route → 建 route_stations
	r := model.Route{
		Name:            req.Name,
		RouteType:       routeType,
		FromStationID:   fromID,
		ToStationID:     toID,
		DistanceKM:      totalDist,
		DurationMinutes: req.DurationMinutes,
		MinFare:         req.MinFare,
		Status:          1, // 默认运营
	}
	if req.Status == 0 || req.Status == 1 {
		r.Status = req.Status
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		if len(req.Stations) > 0 {
			rss := make([]model.RouteStation, 0, len(req.Stations))
			for i, s := range req.Stations {
				rss = append(rss, model.RouteStation{
					RouteID:          r.ID,
					StationID:        s.StationID,
					StopOrder:        i + 1,
					DistanceKM:       s.DistanceKM,
					Price:            s.Price,
					ArrivalTime:      s.ArrivalTime,
					ArrivalDayOffset: s.ArrivalDayOffset,
				})
			}
			if err := tx.Create(&rss).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	// 重新加载带站点序列
	h.DB.Preload("FromStation").Preload("ToStation").
		Preload("RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("RouteStations.Station").First(&r, r.ID)
	WriteLog(c, h.DB, "线路", "新增", r.Name, fmt.Sprintf("线路ID:%d 名称:%s 站点数:%d", r.ID, r.Name, len(req.Stations)))
	response.OK(c, r)
}

func (h *RouteHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var r model.Route
	if err := h.DB.First(&r, id).Error; err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	var req struct {
		Name            string              `json:"name"`
		RouteType       int8                `json:"route_type"` // 1=城乡公交 2=城际客运 3=旅游专线
		FromStationID   uint                `json:"from_station_id"`
		ToStationID     uint                `json:"to_station_id"`
		DistanceKM      float64             `json:"distance_km"`
		DurationMinutes int                 `json:"duration_minutes"`
		MinFare         float64             `json:"min_fare"` // 起步价（最低票价），0=不启用
		Status          int8                `json:"status"`
		Stations        []routeStationInput `json:"stations"` // 若提供则整体重写站点序列
		Force           bool                `json:"force"` // 强制重写站点序列（已知有活跃订单仍继续）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 站点序列校验
	if len(req.Stations) > 0 {
		if len(req.Stations) < 2 {
			response.FailMsg(c, response.CodeParamError, "线路至少需要2个站点")
			return
		}
		seen := make(map[uint]bool)
		for _, s := range req.Stations {
			if s.StationID == 0 {
				response.FailMsg(c, response.CodeParamError, "站点不能为空")
				return
			}
			if seen[s.StationID] {
				response.FailMsg(c, response.CodeParamError, "线路站点不能重复")
				return
			}
			seen[s.StationID] = true
		}
		// 票价单调递增校验（累计票价制：每站Price = 从起点到该站的累计票价，必须单调不减）
		for i := 1; i < len(req.Stations); i++ {
			if req.Stations[i].Price < req.Stations[i-1].Price {
				response.FailMsg(c, response.CodeParamError,
					fmt.Sprintf("站点票价必须单调递增（第%d站票价%.2f低于第%d站%.2f）",
						i+1, req.Stations[i].Price, i, req.Stations[i-1].Price))
				return
			}
		}
	}

	updates := map[string]interface{}{}
	// name 为空时不写入，保留原值
	if req.Name != "" {
		updates["name"] = req.Name
	}
	// route_type=0 不在有效枚举内，不写入保留原值更安全
	if req.RouteType != 0 {
		updates["route_type"] = req.RouteType
	}
	// duration_minutes 是有效整数值，始终写入
	updates["duration_minutes"] = req.DurationMinutes
	// 起步价始终写入（0=不启用）
	updates["min_fare"] = req.MinFare
	// status 是有效值，始终写入
	updates["status"] = req.Status
	if len(req.Stations) > 0 {
		updates["from_station_id"] = req.Stations[0].StationID
		updates["to_station_id"] = req.Stations[len(req.Stations)-1].StationID
		updates["distance_km"] = req.Stations[len(req.Stations)-1].DistanceKM
	} else {
		if req.FromStationID != 0 {
			updates["from_station_id"] = req.FromStationID
		}
		if req.ToStationID != 0 {
			updates["to_station_id"] = req.ToStationID
		}
		updates["distance_km"] = req.DistanceKM
	}
	if fromI, ok := updates["from_station_id"]; ok {
		if toI, ok2 := updates["to_station_id"]; ok2 && fromI == toI {
			response.FailMsg(c, response.CodeParamError, "出发站和到达站不能相同")
			return
		}
	}

	// 重写站点序列前检查活跃订单
	if len(req.Stations) > 0 && !req.Force {
		var activeOrderCount int64
		h.DB.Model(&model.Order{}).
			Joins("JOIN trips ON trips.id = orders.trip_id").
			Where("trips.route_id = ? AND orders.status IN (0, 1)", id).
			Count(&activeOrderCount)
		if activeOrderCount > 0 {
			response.FailWithData(c, response.CodeServerError,
				fmt.Sprintf("该线路有%d个活跃订单（待支付/待出行），重写站点序列会影响已售订单的区间票价计算。如确认需要修改，请使用force=true强制更新", activeOrderCount),
				gin.H{"active_order_count": activeOrderCount})
			return
		}
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&r).Updates(updates).Error; err != nil {
			return err
		}
		// 若提供 stations，整体重写序列：先删旧后建新
		if len(req.Stations) > 0 {
			if err := tx.Where("route_id = ?", r.ID).Delete(&model.RouteStation{}).Error; err != nil {
				return err
			}
			rss := make([]model.RouteStation, 0, len(req.Stations))
			for i, s := range req.Stations {
				rss = append(rss, model.RouteStation{
					RouteID:          r.ID,
					StationID:        s.StationID,
					StopOrder:        i + 1,
					DistanceKM:       s.DistanceKM,
					Price:            s.Price,
					ArrivalTime:      s.ArrivalTime,
					ArrivalDayOffset: s.ArrivalDayOffset,
				})
			}
			if err := tx.Create(&rss).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	h.DB.Preload("FromStation").Preload("ToStation").
		Preload("RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("RouteStations.Station").First(&r, id)
	WriteLog(c, h.DB, "线路", "编辑", r.Name, fmt.Sprintf("线路ID:%s 名称:%s", id, r.Name))
	response.OK(c, r)
}

func (h *RouteHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	force := c.Query("force") == "1"

	// 统计被引用的情况
	var tripCount int64
	h.DB.Model(&model.Trip{}).Where("route_id = ?", id).Count(&tripCount)
	var orderCount int64
	h.DB.Model(&model.Order{}).Where("route_id = ?", id).Count(&orderCount)

	// 无引用：直接删除（含站点序列）
	if tripCount == 0 && orderCount == 0 {
		h.DB.Where("route_id = ?", id).Delete(&model.RouteStation{})
		if err := h.DB.Delete(&model.Route{}, id).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "删除失败")
			return
		}
		WriteLog(c, h.DB, "线路", "删除", id, "删除线路ID:"+id)
		response.OKMsg(c, "删除成功", nil)
		return
	}

	if !force {
		response.FailWithData(c, response.CodeServerError, "该线路被引用，需确认后强制删除", map[string]int64{
			"trip_count":  tripCount,
			"order_count": orderCount,
			"total_count": tripCount + orderCount,
		})
		return
	}

	// 强制删除：事务内断开外键引用并删除线路
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
		defer func() {
			tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
			if r := recover(); r != nil {
				panic(r) // FK_CHECKS 已恢复，重抛 panic 供上层事务回滚
			}
		}()
		// 断开班次和订单的路线引用
		if err := tx.Model(&model.Trip{}).Where("route_id = ?", id).Update("route_id", 0).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Order{}).Where("route_id = ?", id).Update("route_id", 0).Error; err != nil {
			return err
		}
		// 删除站点序列
		if err := tx.Where("route_id = ?", id).Delete(&model.RouteStation{}).Error; err != nil {
			return err
		}
		// 删除线路
		if err := tx.Delete(&model.Route{}, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}

	WriteLog(c, h.DB, "线路", "删除", id, fmt.Sprintf("删除线路ID:%s 强制删除 引用:班次%d/订单%d",
		id, tripCount, orderCount))
	response.OKMsg(c, "删除成功", nil)
}

// Stations 获取线路站点序列（带站点详情）
func (h *RouteHandler) Stations(c *gin.Context) {
	id := c.Param("id")
	var rss []model.RouteStation
	if err := h.DB.Preload("Station").
		Where("route_id = ?", id).
		Order("stop_order ASC").Find(&rss).Error; err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	response.OK(c, rss)
}
