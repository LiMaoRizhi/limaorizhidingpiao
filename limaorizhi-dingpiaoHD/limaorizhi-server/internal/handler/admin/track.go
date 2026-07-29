// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TrackHandler 轨迹可视化管理（实时监控运行中班次 + 历史轨迹回放）
type TrackHandler struct {
	DB *gorm.DB
}

func NewTrackHandler(db *gorm.DB) *TrackHandler {
	return &TrackHandler{DB: db}
}

// activeTripItem 运行中班次 + 最新位置（供地图实时监控）
type activeTripItem struct {
	TripID         uint      `json:"trip_id"`
	TripNo         string    `json:"trip_no"`
	TripDate       string    `json:"trip_date"`
	DepartureTime  string    `json:"departure_time"`
	ArrivalTime    string    `json:"arrival_time"`
	RouteName      string    `json:"route_name"`
	FromStation    string    `json:"from_station"`
	ToStation      string    `json:"to_station"`
	DriverName     string    `json:"driver_name"`
	DriverPhone    string    `json:"driver_phone"`
	VehiclePlateNo string    `json:"vehicle_plate_no"`
	VehicleType    string    `json:"vehicle_type"`
	PassedOrder    int       `json:"passed_order"`
	TotalStations  int       `json:"total_stations"`
	// 最新GPS位置（可能为null——班次已发车但司机尚未上报）
	Longitude  *float64 `json:"longitude"`
	Latitude   *float64 `json:"latitude"`
	Speed      *float64 `json:"speed"`
	Heading    *float64 `json:"heading"`
	ReportedAt *string  `json:"reported_at"`
	// 秒前上报（前端判断新鲜度，>300秒=已过期）
	SecondsAgo *int64 `json:"seconds_ago"`
}

// ActiveTrips 获取当前运行中的班次列表（status=2，已发车未完成）+ 最新GPS位置
func (h *TrackHandler) ActiveTrips(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	var trips []model.Trip
	err := h.DB.Preload("Route.FromStation").Preload("Route.ToStation").
		Preload("Driver").Preload("Vehicle").
		Preload("Route.RouteStations.Station", func(db *gorm.DB) *gorm.DB {
			return db.Order("stop_order ASC")
		}).
		Where("status = 2 AND trip_date <= ?", today).
		Order("trip_date DESC, departure_time DESC").Find(&trips).Error
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "查询运行中班次失败")
		return
	}

	items := make([]activeTripItem, 0, len(trips))
	now := time.Now()
	for _, t := range trips {
		item := activeTripItem{
			TripID:        t.ID,
			TripNo:        t.TripNo,
			TripDate:      string(t.TripDate),
			DepartureTime: t.DepartureTime,
			ArrivalTime:   t.ArrivalTime,
			PassedOrder:  t.CurrentPassedOrder,
		}

		if t.Route != nil {
			item.RouteName = t.Route.Name
			if t.Route.FromStation != nil {
				item.FromStation = t.Route.FromStation.Name
			}
			if t.Route.ToStation != nil {
				item.ToStation = t.Route.ToStation.Name
			}
			item.TotalStations = len(t.Route.RouteStations)
		}

		if t.Driver != nil {
			item.DriverName = t.Driver.Name
			item.DriverPhone = t.Driver.Phone
		}

		if t.Vehicle != nil {
			item.VehiclePlateNo = t.Vehicle.PlateNo
			item.VehicleType = t.Vehicle.VehicleType
		}

		// 查询最近一条GPS位置（不限5分钟窗口，管理端需要看到最后已知位置）
		var loc model.VehicleLocation
		if err := h.DB.Where("trip_id = ?", t.ID).Order("reported_at DESC").First(&loc).Error; err == nil {
			lng := loc.Longitude
			lat := loc.Latitude
			sp := loc.Speed
			hd := loc.Heading
			ts := time.Time(loc.ReportedAt).Format("2006-01-02 15:04:05")
			ago := now.Sub(time.Time(loc.ReportedAt)).Milliseconds() / 1000
			item.Longitude = &lng
			item.Latitude = &lat
			item.Speed = &sp
			item.Heading = &hd
			item.ReportedAt = &ts
			item.SecondsAgo = &ago
		}

		items = append(items, item)
	}

	response.OK(c, gin.H{
		"list":  items,
		"total": len(items),
	})
}

// trackPoint GPS轨迹点
type trackPoint struct {
	Longitude  float64 `json:"longitude"`
	Latitude   float64 `json:"latitude"`
	Speed      float64 `json:"speed"`
	Heading    float64 `json:"heading"`
	ReportedAt string  `json:"reported_at"`
}

// routeStationItem 线路站点（含坐标，供前端绘制路线折线）
type routeStationItem struct {
	StopOrder  int     `json:"stop_order"`
	StationID  uint    `json:"station_id"`
	Name       string  `json:"name"`
	Longitude  float64 `json:"longitude"`
	Latitude   float64 `json:"latitude"`
}

// TripTrack 获取指定班次的完整GPS轨迹 + 线路站点坐标（供地图回放）
func (h *TrackHandler) TripTrack(c *gin.Context) {
	tripID := c.Param("id")

	var trip model.Trip
	if err := h.DB.Preload("Route").
		Preload("Driver").Preload("Vehicle").
		Preload("Route.RouteStations.Station", func(db *gorm.DB) *gorm.DB {
			return db.Order("stop_order ASC")
		}).
		First(&trip, tripID).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}

	// 查询全部GPS轨迹点（按时间正序，供前端画轨迹线）
	var locs []model.VehicleLocation
	h.DB.Where("trip_id = ?", tripID).Order("reported_at ASC").Find(&locs)

	points := make([]trackPoint, 0, len(locs))
	for _, l := range locs {
		points = append(points, trackPoint{
			Longitude:  l.Longitude,
			Latitude:   l.Latitude,
			Speed:      l.Speed,
			Heading:    l.Heading,
			ReportedAt: time.Time(l.ReportedAt).Format("2006-01-02 15:04:05"),
		})
	}

	// 线路站点坐标（供前端绘制站点标记和路线折线）
	var stations []routeStationItem
	if trip.Route != nil {
		for _, rs := range trip.Route.RouteStations {
			if rs.Station != nil {
				stations = append(stations, routeStationItem{
					StopOrder: rs.StopOrder,
					StationID: rs.StationID,
					Name:       rs.Station.Name,
					Longitude:  rs.Station.Longitude,
					Latitude:   rs.Station.Latitude,
				})
			}
		}
	}

	// 班次信息
	tripInfo := gin.H{
		"trip_id":         trip.ID,
		"trip_no":         trip.TripNo,
		"trip_date":       string(trip.TripDate),
		"departure_time":  trip.DepartureTime,
		"arrival_time":    trip.ArrivalTime,
		"status":          trip.Status,
		"passed_order":    trip.CurrentPassedOrder,
		"route_name":      "",
		"from_station":    "",
		"to_station":      "",
		"driver_name":     "",
		"vehicle_plate":   "",
	}
	if trip.Route != nil {
		tripInfo["route_name"] = trip.Route.Name
	}
	if trip.Driver != nil {
		tripInfo["driver_name"] = trip.Driver.Name
	}
	if trip.Vehicle != nil {
		tripInfo["vehicle_plate"] = trip.Vehicle.PlateNo
	}
	// 查询起终站名
	if trip.Route != nil {
		var fromSt, toSt model.Station
		h.DB.First(&fromSt, trip.Route.FromStationID)
		h.DB.First(&toSt, trip.Route.ToStationID)
		tripInfo["from_station"] = fromSt.Name
		tripInfo["to_station"] = toSt.Name
	}

	response.OK(c, gin.H{
		"trip":     tripInfo,
		"points":   points,
		"stations": stations,
	})
}
