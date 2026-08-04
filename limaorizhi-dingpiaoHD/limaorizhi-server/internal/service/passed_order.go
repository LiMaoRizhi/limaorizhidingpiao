package service

import (
	"math"
	"sort"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/triptime"

	"gorm.io/gorm"
)

// VehiclePassedOrderByGPS 用车辆实时GPS位置判断车已驶过到第几站
// 查询最近5分钟内的最新VehicleLocation，投影到线路站点折线，判断车辆位于哪一段
// 返回 (已驶过的最后一站stop_order, true)；无GPS数据或站点坐标不足则返回 (0, false) 降级
func VehiclePassedOrderByGPS(db *gorm.DB, tripID uint, routeStations []model.RouteStation) (int, bool) {
	if db == nil || len(routeStations) == 0 {
		return 0, false
	}

	// 查询最近5分钟内的最新GPS位置（与乘客端TripLocation保持一致的5分钟过期窗口）
	var loc model.VehicleLocation
	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	if err := db.Where("trip_id = ? AND reported_at > ?", tripID, fiveMinAgo).
		Order("reported_at DESC").First(&loc).Error; err != nil {
		return 0, false // 无近期GPS数据，降级到时刻表/推算
	}

	// 确保站点坐标已加载（routeStations 可能未预加载 Station）
	for i := range routeStations {
		if routeStations[i].Station == nil {
			var st model.Station
			if err := db.First(&st, routeStations[i].StationID).Error; err != nil {
				return 0, false
			}
			routeStations[i].Station = &st
		}
	}

	// 过滤出有坐标的站点，按 stop_order 排序
	validStations := make([]model.RouteStation, 0, len(routeStations))
	for _, rs := range routeStations {
		if rs.Station != nil && rs.Station.Longitude != 0 && rs.Station.Latitude != 0 {
			validStations = append(validStations, rs)
		}
	}
	if len(validStations) < 2 {
		return 0, false // 站点坐标不足，无法投影判断
	}
	sort.Slice(validStations, func(i, j int) bool {
		return validStations[i].StopOrder < validStations[j].StopOrder
	})

	vehicleLat := loc.Latitude
	vehicleLng := loc.Longitude

	// 计算车辆到每段线路的垂直距离，找到最近段
	minSegmentDist := math.MaxFloat64
	bestSegmentIdx := 0 // 车辆在第 bestSegmentIdx 站和第 bestSegmentIdx+1 站之间

	for i := 0; i < len(validStations)-1; i++ {
		s1 := validStations[i].Station
		s2 := validStations[i+1].Station
		dist := PointToSegmentDistance(vehicleLat, vehicleLng, s1.Latitude, s1.Longitude, s2.Latitude, s2.Longitude)
		if dist < minSegmentDist {
			minSegmentDist = dist
			bestSegmentIdx = i
		}
	}

	// 到首末站的直线距离
	firstStation := validStations[0].Station
	lastStation := validStations[len(validStations)-1].Station
	distToFirst := HaversineMeters(vehicleLat, vehicleLng, firstStation.Latitude, firstStation.Longitude)
	distToLast := HaversineMeters(vehicleLat, vehicleLng, lastStation.Latitude, lastStation.Longitude)

	// 离首站最近 → 还没出发或刚出发
	if distToFirst < minSegmentDist && distToFirst < 200 { // 200米容差
		return 0, true // 已过0站（仍在首站附近）
	}

	// 离末站最近 → 已到终点
	if distToLast < minSegmentDist && distToLast < 200 {
		return validStations[len(validStations)-1].StopOrder, true
	}

	// GPS距所有线路段超过5km，可能偏离路线或GPS漂移，降级到手动标记
	if minSegmentDist > 5000 {
		return 0, false
	}

	// 车辆在 bestSegmentIdx 站和 bestSegmentIdx+1 站之间，已驶过 bestSegmentIdx 站
	return validStations[bestSegmentIdx].StopOrder, true
}

// EffectivePassedOrder 计算有效的已过站序号（供乘客展示和下单判断统一使用）
// GPS可用时优先使用GPS投影结果；GPS不可用时回退到手动标记
// 取GPS投影和手动标记(trip.CurrentPassedOrder)中的较大值：
//   - GPS显示到第5站但司机只手动标记到第3站 → 返回5（GPS更准确）
//   - GPS不可用但司机手动标记到第3站 → 返回3（回退到手动标记）
//   - GPS显示到第3站但司机手动标记到第5站 → 返回5（司机可前进覆盖）
//   - 两者都为0 → 返回0（降级到时刻表/推算）
func EffectivePassedOrder(db *gorm.DB, trip model.Trip, routeStations []model.RouteStation) int {
	manualOrder := trip.CurrentPassedOrder

	// GPS投影结果
	if gpsOrder, ok := VehiclePassedOrderByGPS(db, trip.ID, routeStations); ok {
		if gpsOrder > manualOrder {
			return gpsOrder
		}
	}

	return manualOrder
}

// LoadRouteStations 加载线路站点（按站序升序）
func LoadRouteStations(db *gorm.DB, routeID uint) []model.RouteStation {
	var routeStations []model.RouteStation
	if db != nil {
		if err := db.Where("route_id = ?", routeID).Order("stop_order ASC").Find(&routeStations).Error; err != nil {
			return nil
		}
	}
	return routeStations
}

// TripPassedOrderForTripID 便捷函数：给定班次ID，返回有效已过站序号（用于座位释放判断）
func TripPassedOrderForTripID(db *gorm.DB, tripID uint) int {
	var trip model.Trip
	if err := db.First(&trip, tripID).Error; err != nil {
		return 0
	}
	routeStations := LoadRouteStations(db, trip.RouteID)
	return EffectivePassedOrder(db, trip, routeStations)
}

// HaversineMeters 用haversine公式计算两个经纬度坐标之间的球面距离（米）
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000 // 地球平均半径（米）
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// PointToSegmentDistance 计算点P到线段AB的最近距离（经纬度坐标，小范围近似为平面）
// 参数顺序：pointLat, pointLng, aLat, aLng, bLat, bLng
func PointToSegmentDistance(pLat, pLng, aLat, aLng, bLat, bLng float64) float64 {
	// 经度按纬度修正（1度经度≈cos(lat)度纬度距离），使平面近似更准确
	cosLat := math.Cos(pLat * math.Pi / 180)
	px := pLat
	py := pLng * cosLat
	ax := aLat
	ay := aLng * cosLat
	bx := bLat
	by := bLng * cosLat

	dx := bx - ax
	dy := by - ay
	if dx == 0 && dy == 0 {
		// A和B重合
		return HaversineMeters(pLat, pLng, aLat, aLng)
	}
	// 投影比例t，限制在[0,1]区间内
	t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	projLat := aLat + t*(bLat-aLat)
	projLng := aLng + t*(bLng-aLng)
	return HaversineMeters(pLat, pLng, projLat, projLng)
}

// ParseTripDeparture 解析发车时间（返回time，解析失败返回zero并ok=false）
func ParseTripDeparture(trip model.Trip) (time.Time, bool) {
	t, err := triptime.Parse(string(trip.TripDate), trip.DepartureTime)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
