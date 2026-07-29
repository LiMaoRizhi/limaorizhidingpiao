// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"fmt"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/triptime"

	"gorm.io/gorm"
)

// ConflictInfo 冲突详情（供管理员确认后决定是否强制分配）
type ConflictInfo struct {
	TripID        uint   `json:"trip_id"`
	TripNo        string `json:"trip_no"`
	RouteName     string `json:"route_name"`
	FromStation   string `json:"from_station"`
	ToStation     string `json:"to_station"`
	DepartureTime string `json:"departure_time"`
	ArrivalTime   string `json:"arrival_time"`
	Status        int8   `json:"status"`
	StatusText    string `json:"status_text"`
	ConflictType  string `json:"conflict_type"` // time_overlap / location_gap / vehicle_overlap
	ConflictDesc  string `json:"conflict_desc"`
}

// TripConflictCheck 班次冲突检测参数
type TripConflictCheck struct {
	ExcludeTripID    uint   // 排除的班次ID（编辑时排除自身）
	DriverID         uint   // 司机ID（0表示未分配司机，跳过司机检测）
	VehicleID        uint   // 车辆ID
	TripDate         string // 班次日期（YYYY-MM-DD）
	DepartureTime    string // 发车时间（HH:MM）
	ArrivalTime      string // 到达时间（HH:MM）
	ArrivalDayOffset int    // 到达相对发车日的天数偏移(0=当天,1=次日...)，跨天班次用
	FromStationID    uint   // 新班次起点站ID
	ToStationID      uint   // 新班次终点站ID
}

// tripsTimeOverlap 判断两个同日发车班次的时间段是否重叠，支持跨天偏移
// baseDate 为发车日期；dep*/arr* 为 HH:MM[:SS] 时刻；off* 为到达天数偏移
// 解析失败时降级为字符串比较（向后兼容旧数据）
func tripsTimeOverlap(baseDate, dep1, arr1 string, off1 int, dep2, arr2 string, off2 int) bool {
	d1 := triptime.MustParse(baseDate, dep1)
	a1 := triptime.MustParse(baseDate, arr1)
	if off1 > 0 {
		a1 = a1.AddDate(0, 0, off1)
	}
	d2 := triptime.MustParse(baseDate, dep2)
	a2 := triptime.MustParse(baseDate, arr2)
	if off2 > 0 {
		a2 = a2.AddDate(0, 0, off2)
	}
	if d1.IsZero() || a1.IsZero() || d2.IsZero() || a2.IsZero() {
		// 解析失败降级为字符串比较（不感知跨天）
		return dep1 < arr2 && arr1 > dep2
	}
	return d1.Before(a2) && a1.After(d2)
}

// CheckTripConflict 检查班次冲突（司机时间重叠、车辆时间重叠、位置断层）
// 使用站点ID进行比较，避免站点名称比较的误判
// 返回所有冲突列表（硬冲突+软警告），调用方根据业务决定是否阻止
func CheckTripConflict(db *gorm.DB, check TripConflictCheck) []ConflictInfo {
	var conflicts []ConflictInfo

	// 跨天感知：计算新班次的发车/到达 time.Time（到达日期 = 发车日 + arrival_day_offset）
	newDep := triptime.MustParse(check.TripDate, check.DepartureTime)
	newArr := triptime.MustParse(check.TripDate, check.ArrivalTime)
	if check.ArrivalDayOffset > 0 {
		newArr = newArr.AddDate(0, 0, check.ArrivalDayOffset)
	}

	// 司机冲突检测
	if check.DriverID > 0 {
		var allTrips []model.Trip
		if err := db.Preload("Route.FromStation").Preload("Route.ToStation").
			Where("driver_id = ? AND trip_date = ? AND status IN (?, ?) AND id != ?",
				check.DriverID, check.TripDate, model.TripStatusSale, model.TripStatusDepart, check.ExcludeTripID).
			Order("departure_time ASC").Find(&allTrips).Error; err != nil {
			// DB查询失败时 fail-safe：返回冲突信息阻止操作，防止漏检导致冲突分配
			conflicts = append(conflicts, ConflictInfo{
				ConflictType: "time_overlap",
				ConflictDesc:  fmt.Sprintf("司机冲突检测查询失败: %v，请重试", err),
			})
			return conflicts
		}

		for _, ct := range allTrips {
			ctFromStationID, ctToStationID, ctFromStation, ctToStation, routeName := extractRouteInfo(ct.Route)

			var statusText string
			if ct.Status == model.TripStatusDepart {
				statusText = "已发车(未结束)"
			} else {
				statusText = "可售(待发车)"
			}

			// 1. 时间重叠检测（硬冲突）—— 跨天感知
			ctDep := triptime.MustParse(check.TripDate, ct.DepartureTime)
			ctArr := triptime.MustParse(check.TripDate, ct.ArrivalTime)
			if ct.ArrivalDayOffset > 0 {
				ctArr = ctArr.AddDate(0, 0, ct.ArrivalDayOffset)
			}
			overlap := false
			if !newDep.IsZero() && !newArr.IsZero() && !ctDep.IsZero() && !ctArr.IsZero() {
				overlap = newDep.Before(ctArr) && newArr.After(ctDep)
			} else {
				overlap = check.DepartureTime < ct.ArrivalTime && check.ArrivalTime > ct.DepartureTime
			}
			if overlap {
				conflicts = append(conflicts, ConflictInfo{
					TripID:        ct.ID,
					TripNo:        ct.TripNo,
					RouteName:     routeName,
					FromStation:   ctFromStation,
					ToStation:     ctToStation,
					DepartureTime: ct.DepartureTime,
					ArrivalTime:   ct.ArrivalTime,
					Status:        ct.Status,
					StatusText:    statusText,
					ConflictType:  "time_overlap",
					ConflictDesc:  fmt.Sprintf("时间重叠：该班次(%s-%s)与已有班次(%s-%s)时间冲突", check.DepartureTime, check.ArrivalTime, ct.DepartureTime, ct.ArrivalTime),
				})
				continue
			}

			// 2. 位置断层检测（软警告）—— 使用站点ID比较，跨天感知
			// 场景A：已有班次在新班次之前结束 → 司机在已有班次终点站，新班次从另一站出发
			ctBeforeNew := false
			if !ctArr.IsZero() && !newDep.IsZero() {
				ctBeforeNew = !ctArr.After(newDep) // ctArr <= newDep
			} else {
				ctBeforeNew = ct.ArrivalTime <= check.DepartureTime
			}
			if ctBeforeNew {
				if ctToStationID != 0 && check.FromStationID != 0 && ctToStationID != check.FromStationID {
					newFromName := stationNameByID(db, check.FromStationID)
					conflicts = append(conflicts, ConflictInfo{
						TripID:        ct.ID,
						TripNo:        ct.TripNo,
						RouteName:     routeName,
						FromStation:   ctFromStation,
						ToStation:     ctToStation,
						DepartureTime: ct.DepartureTime,
						ArrivalTime:   ct.ArrivalTime,
						Status:        ct.Status,
						StatusText:    statusText,
						ConflictType:  "location_gap",
						ConflictDesc:  fmt.Sprintf("位置断层：前序班次终点[%s] ≠ 新班次起点[%s]，司机需从%s移动到%s", ctToStation, newFromName, ctToStation, newFromName),
					})
				}
			}

			// 场景B：新班次在已有班次之前结束 → 司机在新班次终点站，已有班次从另一站出发
			newBeforeCt := false
			if !newArr.IsZero() && !ctDep.IsZero() {
				newBeforeCt = !newArr.After(ctDep) // newArr <= ctDep
			} else {
				newBeforeCt = check.ArrivalTime <= ct.DepartureTime
			}
			if newBeforeCt {
				if check.ToStationID != 0 && ctFromStationID != 0 && check.ToStationID != ctFromStationID {
					newToName := stationNameByID(db, check.ToStationID)
					conflicts = append(conflicts, ConflictInfo{
						TripID:        ct.ID,
						TripNo:        ct.TripNo,
						RouteName:     routeName,
						FromStation:   ctFromStation,
						ToStation:     ctToStation,
						DepartureTime: ct.DepartureTime,
						ArrivalTime:   ct.ArrivalTime,
						Status:        ct.Status,
						StatusText:    statusText,
						ConflictType:  "location_gap",
						ConflictDesc:  fmt.Sprintf("位置断层：新班次终点[%s] ≠ 后续班次起点[%s]，司机需从%s移动到%s", newToName, ctFromStation, newToName, ctFromStation),
					})
				}
			}
		}
	}

	// 车辆冲突检测
	if check.VehicleID > 0 {
		var vehicleTrips []model.Trip
		if err := db.Preload("Route.FromStation").Preload("Route.ToStation").
			Where("vehicle_id = ? AND trip_date = ? AND status IN (?, ?) AND id != ?",
				check.VehicleID, check.TripDate, model.TripStatusSale, model.TripStatusDepart, check.ExcludeTripID).
			Find(&vehicleTrips).Error; err != nil {
			// DB查询失败时 fail-safe：返回冲突信息阻止操作
			conflicts = append(conflicts, ConflictInfo{
				ConflictType: "vehicle_overlap",
				ConflictDesc:  fmt.Sprintf("车辆冲突检测查询失败: %v，请重试", err),
			})
			return conflicts
		}

		for _, vt := range vehicleTrips {
			// 跨天感知：车辆时间重叠检测
			vtDep := triptime.MustParse(check.TripDate, vt.DepartureTime)
			vtArr := triptime.MustParse(check.TripDate, vt.ArrivalTime)
			if vt.ArrivalDayOffset > 0 {
				vtArr = vtArr.AddDate(0, 0, vt.ArrivalDayOffset)
			}
			vtOverlap := false
			if !newDep.IsZero() && !newArr.IsZero() && !vtDep.IsZero() && !vtArr.IsZero() {
				vtOverlap = newDep.Before(vtArr) && newArr.After(vtDep)
			} else {
				vtOverlap = check.DepartureTime < vt.ArrivalTime && check.ArrivalTime > vt.DepartureTime
			}
			if vtOverlap {
				_, _, vtFromStation, vtToStation, routeName := extractRouteInfo(vt.Route)
				conflicts = append(conflicts, ConflictInfo{
					TripID:        vt.ID,
					TripNo:        vt.TripNo,
					RouteName:     routeName,
					FromStation:   vtFromStation,
					ToStation:     vtToStation,
					DepartureTime: vt.DepartureTime,
					ArrivalTime:   vt.ArrivalTime,
					Status:        vt.Status,
					StatusText:    "车辆占用",
					ConflictType:  "vehicle_overlap",
					ConflictDesc:  fmt.Sprintf("车辆冲突：同一车辆在此时间段已被班次%s占用(%s-%s)", vt.TripNo, vt.DepartureTime, vt.ArrivalTime),
				})
			}
		}
	}

	return conflicts
}

// extractRouteInfo 从 Route 提取站名、站点ID和路线名（统一 nil 检查和路线名拼接）
func extractRouteInfo(route *model.Route) (fromStationID, toStationID uint, fromStation, toStation, routeNameStr string) {
	if route == nil {
		return
	}
	fromStationID = route.FromStationID
	toStationID = route.ToStationID
	if route.FromStation != nil {
		fromStation = route.FromStation.Name
	}
	if route.ToStation != nil {
		toStation = route.ToStation.Name
	}
	if fromStation != "" && toStation != "" {
		routeNameStr = fromStation + " → " + toStation
	}
	return
}

// stationNameByID 根据站点ID查询站点名称
func stationNameByID(db *gorm.DB, stationID uint) string {
	var s model.Station
	if err := db.First(&s, stationID).Error; err != nil {
		return fmt.Sprintf("站点#%d", stationID)
	}
	return s.Name
}

// HasHardConflict 判断冲突列表中是否包含硬冲突（时间重叠或车辆重叠）
func HasHardConflict(conflicts []ConflictInfo) bool {
	for _, c := range conflicts {
		if c.ConflictType == "time_overlap" || c.ConflictType == "vehicle_overlap" {
			return true
		}
	}
	return false
}

// ConflictSummary 生成冲突摘要消息
func ConflictSummary(conflicts []ConflictInfo) string {
	hardCount, softCount, vehicleCount := 0, 0, 0
	for _, cf := range conflicts {
		switch cf.ConflictType {
		case "time_overlap":
			hardCount++
		case "vehicle_overlap":
			vehicleCount++
		case "location_gap":
			softCount++
		}
	}
	msg := fmt.Sprintf("该司机/车辆在此日期有冲突：%d个时间重叠(硬冲突)，%d个位置断层(软警告)", hardCount, softCount)
	if vehicleCount > 0 {
		msg += fmt.Sprintf("，%d个车辆占用(硬冲突)", vehicleCount)
	}
	return msg
}
