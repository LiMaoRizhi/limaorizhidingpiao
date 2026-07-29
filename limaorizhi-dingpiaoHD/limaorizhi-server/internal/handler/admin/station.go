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
// 站点管理

type StationHandler struct {
	DB *gorm.DB
}

func NewStationHandler(db *gorm.DB) *StationHandler {
	return &StationHandler{DB: db}
}

func (h *StationHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.DB.Model(&model.Station{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+sanitize.EscapeLikePattern(name)+"%")
	}
	query.Count(&total)

	var list []model.Station
	query.Order("sort_order ASC, id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

func (h *StationHandler) All(c *gin.Context) {
	var list []model.Station
	h.DB.Where("status = 1").Order("sort_order ASC, id ASC").Find(&list)
	response.OK(c, list)
}

// createStationRequest 站点创建请求（DTO白名单）
type createStationRequest struct {
	Name      string  `json:"name" binding:"required"`
	Pinyin    string  `json:"pinyin"`
	SortOrder int     `json:"sort_order"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Status    int8    `json:"status"`
}

func (h *StationHandler) Create(c *gin.Context) {
	var req createStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.Name == "" {
		response.FailMsg(c, response.CodeParamError, "站点名称不能为空")
		return
	}
	s := model.Station{
		Name:      req.Name,
		Pinyin:    req.Pinyin,
		SortOrder: req.SortOrder,
		Longitude: req.Longitude,
		Latitude:  req.Latitude,
		Status:    1, // 默认启用
	}
	if req.Status == 0 || req.Status == 1 {
		s.Status = req.Status
	}
	if err := h.DB.Create(&s).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	WriteLog(c, h.DB, "站点", "新增", s.Name, fmt.Sprintf("站点ID:%d 名称:%s", s.ID, s.Name))
	response.OK(c, s)
}

func (h *StationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var s model.Station
	if err := h.DB.First(&s, id).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "站点不存在")
		return
	}
	var req struct {
		Name      string  `json:"name"`
		Pinyin    string  `json:"pinyin"`
		SortOrder int     `json:"sort_order"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
		Status    int8    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.Name == "" {
		response.FailMsg(c, response.CodeParamError, "站点名称不能为空")
		return
	}
	// 使用Updates而非Save，防止Mass Assignment
	if err := h.DB.Model(&s).Updates(map[string]interface{}{
		"name": req.Name, "pinyin": req.Pinyin, "sort_order": req.SortOrder,
		"longitude": req.Longitude, "latitude": req.Latitude, "status": req.Status,
	}).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	h.DB.First(&s, id)
	WriteLog(c, h.DB, "站点", "编辑", s.Name, fmt.Sprintf("站点ID:%s 名称:%s", id, s.Name))
	response.OK(c, s)
}

func (h *StationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	force := c.Query("force") == "1"

	// 统计被引用的情况
	var routeCount int64
	h.DB.Model(&model.Route{}).Where("from_station_id = ? OR to_station_id = ?", id, id).Count(&routeCount)
	var rsCount int64
	h.DB.Model(&model.RouteStation{}).Where("station_id = ?", id).Count(&rsCount)
	var orderCount int64
	h.DB.Model(&model.Order{}).Where("from_station_id = ? OR to_station_id = ?", id, id).Count(&orderCount)

	// 无引用：直接删除
	if routeCount == 0 && rsCount == 0 && orderCount == 0 {
		if err := h.DB.Delete(&model.Station{}, id).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "删除失败")
			return
		}
		WriteLog(c, h.DB, "站点", "删除", id, "删除站点ID:"+id)
		response.OKMsg(c, "删除成功", nil)
		return
	}

	if !force {
		response.FailWithData(c, response.CodeServerError, "该站点被引用，需确认后强制删除", map[string]int64{
			"route_count":         routeCount,
			"route_station_count": rsCount,
			"order_count":         orderCount,
			"total_count":         routeCount + rsCount + orderCount,
		})
		return
	}

	// 强制删除：事务内断开所有外键引用并删除站点
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
		defer func() {
			tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
			if r := recover(); r != nil {
				panic(r) // FK_CHECKS 已恢复，重抛 panic 供上层事务回滚
			}
		}()
		// 删除线路站点序列中的引用
		if err := tx.Where("station_id = ?", id).Delete(&model.RouteStation{}).Error; err != nil {
			return err
		}
		// 断开线路的起终点引用
		if err := tx.Model(&model.Route{}).Where("from_station_id = ?", id).Update("from_station_id", 0).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Route{}).Where("to_station_id = ?", id).Update("to_station_id", 0).Error; err != nil {
			return err
		}
		// 断开订单的站点引用（订单已冗余存储站点名，不影响展示）
		if err := tx.Model(&model.Order{}).Where("from_station_id = ?", id).Update("from_station_id", 0).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Order{}).Where("to_station_id = ?", id).Update("to_station_id", 0).Error; err != nil {
			return err
		}
		// 删除站点
		if err := tx.Delete(&model.Station{}, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}

	WriteLog(c, h.DB, "站点", "删除", id, fmt.Sprintf("删除站点ID:%s 强制删除 引用:线路%d/站点序列%d/订单%d",
		id, routeCount, rsCount, orderCount))
	response.OKMsg(c, "删除成功", nil)
}

// StationRoutes 查询某站点关联的所有线路（出发点/到达站/途经站）
// 用于站点枢纽视图：回答“从界沟客运站可以到哪些地方？”
func (h *StationHandler) StationRoutes(c *gin.Context) {
	stationID := c.Param("id")
	if stationID == "" {
		response.FailMsg(c, response.CodeParamError, "缺少站点ID")
		return
	}

	// 查询以该站为起点的线路
	var fromRoutes []model.Route
	h.DB.Preload("FromStation").Preload("ToStation").
		Where("from_station_id = ?", stationID).Find(&fromRoutes)

	type routeInfo struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		RouteType  int8   `json:"route_type"`
		FromStation string `json:"from_station"`
		ToStation   string `json:"to_station"`
		Relation    string `json:"relation"` // from / to / via
	}

	var result []routeInfo
	seen := make(map[uint]bool)

	// 出发线路
	for _, r := range fromRoutes {
		if seen[r.ID] { continue }
		seen[r.ID] = true
		var fs, ts string
		if r.FromStation != nil { fs = r.FromStation.Name }
		if r.ToStation != nil { ts = r.ToStation.Name }
		result = append(result, routeInfo{
			ID: r.ID, Name: r.Name, RouteType: r.RouteType,
			FromStation: fs, ToStation: ts, Relation: "from",
		})
	}

	// 到达线路
	var toRoutes []model.Route
	h.DB.Preload("FromStation").Preload("ToStation").
		Where("to_station_id = ?", stationID).Find(&toRoutes)
	for _, r := range toRoutes {
		if seen[r.ID] { continue }
		seen[r.ID] = true
		var fs, ts string
		if r.FromStation != nil { fs = r.FromStation.Name }
		if r.ToStation != nil { ts = r.ToStation.Name }
		result = append(result, routeInfo{
			ID: r.ID, Name: r.Name, RouteType: r.RouteType,
			FromStation: fs, ToStation: ts, Relation: "to",
		})
	}

	// 途经线路（该站是线路的中途站点，既非起点也非终点）
	var viaRoutes []model.Route
	h.DB.Distinct("routes.*").
		Joins("JOIN route_stations ON route_stations.route_id = routes.id").
		Where("route_stations.station_id = ? AND routes.from_station_id != ? AND routes.to_station_id != ?",
			stationID, stationID, stationID).
		Find(&viaRoutes)
	for _, r := range viaRoutes {
		if seen[r.ID] { continue }
		seen[r.ID] = true
		var fs, ts string
		h.DB.Model(&model.Station{}).Where("id = ?", r.FromStationID).Limit(1).Pluck("name", &fs)
		h.DB.Model(&model.Station{}).Where("id = ?", r.ToStationID).Limit(1).Pluck("name", &ts)
		result = append(result, routeInfo{
			ID: r.ID, Name: r.Name, RouteType: r.RouteType,
			FromStation: fs, ToStation: ts, Relation: "via",
		})
	}

	// 查询站点名称
	var station model.Station
	h.DB.First(&station, stationID)

	response.OK(c, gin.H{
		"station_id":   stationID,
		"station_name": station.Name,
		"routes":       result,
		"total":        len(result),
	})
}
