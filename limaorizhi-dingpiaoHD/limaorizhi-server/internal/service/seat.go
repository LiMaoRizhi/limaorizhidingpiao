// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"fmt"
	"sort"
	"strconv"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// segmentOrder 一个订单占了哪段区间
type segmentOrder struct {
	FromOrder int // 上车站序号
	ToOrder   int // 下车站序号
	Count     int // 几个人（占几个座）
}

// seatAssignment 乘客的座位占用情况
type seatAssignment struct {
	SeatNo    string
	FromOrder int
	ToOrder   int
}

// AssignSeats 给新订单的乘客分座位号
// 区间复用：比如商丘到界沟镇这条线，甲买1→3站占1号座，乙买3→8站也能坐1号座（不重叠就行）
// 5月16号刚写这版的时候区间复用还没捋顺，改了好几版才对，最早用的笨办法嵌套循环
// 必须搁事务里调，不然并发会分到同一个座
func AssignSeats(tx *gorm.DB, tripID uint, totalSeats, fromOrder, toOrder, passengerCount int) ([]string, error) {
	// 查这个班次已分了座位的乘客，连带他们的上下车站序
	var assignments []seatAssignment
	err := tx.Table("order_passengers").
			Select("order_passengers.seat_no AS seat_no, rs_from.stop_order AS from_order, rs_to.stop_order AS to_order").
			Joins("JOIN orders ON orders.id = order_passengers.order_id").
			Joins("JOIN route_stations rs_from ON rs_from.route_id = orders.route_id AND rs_from.station_id = orders.from_station_id").
			Joins("JOIN route_stations rs_to ON rs_to.route_id = orders.route_id AND rs_to.station_id = orders.to_station_id").
			Where("orders.trip_id = ? AND orders.order_type = 1 AND orders.status IN (?, ?) AND order_passengers.seat_no != ''", tripID, model.OrderStatusPending, model.OrderStatusPaid).
			Scan(&assignments).Error
	if err != nil {
		return nil, err
	}

	// 按座位号分组，看每个座都被哪些区间占了
	seatOccupancy := make(map[int][][2]int) // 座位号 -> [from, to) 区间列表
	for _, a := range assignments {
		seatNum, convErr := strconv.Atoi(a.SeatNo)
		if convErr != nil {
			continue // 座位号不合法，跳过
		}
		seatOccupancy[seatNum] = append(seatOccupancy[seatNum], [2]int{a.FromOrder, a.ToOrder})
	}

	// 从1号座开始挨个找，看哪个座搁这区间没人坐
	var assignedSeats []string
	for seat := 1; seat <= totalSeats && len(assignedSeats) < passengerCount; seat++ {
		intervals := seatOccupancy[seat]
		available := true
		for _, iv := range intervals {
			// 半开区间重叠：[from, to) 和 [iv0, iv1) 有交集就不能用
			if fromOrder < iv[1] && toOrder > iv[0] {
				available = false
				break
			}
		}
		if available {
			assignedSeats = append(assignedSeats, strconv.Itoa(seat))
		}
	}

	if len(assignedSeats) < passengerCount {
		return nil, fmt.Errorf("座位不足，无法分配座位号")
	}

	return assignedSeats, nil
}

// RealtimeAvailableSeats 算班次还能卖多少票（取最挤的那段）
// 首页列表用这个，比 trips.available_seats 那个字段准
// 6月18号又来改过几次，之前没考虑跨天班次，最挤那段算错了好几个bug
func RealtimeAvailableSeats(tx *gorm.DB, tripID uint, totalSeats int) int {
	var rows []segmentOrder
	err := tx.Table("orders").
			Select("rs_from.stop_order AS from_order, rs_to.stop_order AS to_order, orders.passenger_count AS count").
			Joins("JOIN route_stations rs_from ON rs_from.route_id = orders.route_id AND rs_from.station_id = orders.from_station_id").
			Joins("JOIN route_stations rs_to ON rs_to.route_id = orders.route_id AND rs_to.station_id = orders.to_station_id").
			Where("orders.trip_id = ? AND orders.order_type = 1 AND orders.status IN (?, ?)", tripID, model.OrderStatusPending, model.OrderStatusPaid).
			Scan(&rows).Error
	if err != nil {
		// 数据库异常时返回 0（fail-closed），避免掩盖故障导致超卖
		return 0
	}
	if len(rows) == 0 {
		return totalSeats // 无订单时返回满座
	}

	// 找最大的站序号，建差分数组
	maxOrder := 0
	for _, r := range rows {
		if r.ToOrder > maxOrder {
			maxOrder = r.ToOrder
		}
	}
	if maxOrder == 0 {
		return totalSeats
	}

	// 差分数组算每段客流，比嵌套循环快得多
	diff := make([]int, maxOrder+2)
	for _, r := range rows {
		if r.FromOrder >= 0 && r.FromOrder < len(diff) {
			diff[r.FromOrder] += r.Count
		}
		if r.ToOrder >= 0 && r.ToOrder < len(diff) {
			diff[r.ToOrder] -= r.Count
		}
	}

	maxOcc := 0
	occ := 0
	for i := 1; i < maxOrder && i < len(diff); i++ {
		occ += diff[i]
		if occ > maxOcc {
			maxOcc = occ
		}
	}

	avail := totalSeats - maxOcc
	if avail < 0 {
		avail = 0
	}
	return avail
}

// StationBoardingStat 每站上下车统计
type StationBoardingStat struct {
	StopOrder    int    `json:"stop_order"`
	StationID    uint   `json:"station_id"`
	StationName  string `json:"station_name"`
	Boarding     int    `json:"boarding"`     // 上车人数
	Alighting    int    `json:"alighting"`   // 下车人数
	Onboard      int    `json:"onboard"`     // 车上当前人数（累计上车-下车）
	Available    int    `json:"available"`   // 该站之后可用余座
}

// SegmentStats 每站上下车统计
// 返回各站上车下车人数、车上累计人数、之后还能坐多少人
func SegmentStats(tx *gorm.DB, tripID uint, totalSeats int, routeStations []model.RouteStation) []StationBoardingStat {
	var rows []segmentOrder
	err := tx.Table("orders").
			Select("rs_from.stop_order AS from_order, rs_to.stop_order AS to_order, orders.passenger_count AS count").
			Joins("JOIN route_stations rs_from ON rs_from.route_id = orders.route_id AND rs_from.station_id = orders.from_station_id").
			Joins("JOIN route_stations rs_to ON rs_to.route_id = orders.route_id AND rs_to.station_id = orders.to_station_id").
			Where("orders.trip_id = ? AND orders.order_type = 1 AND orders.status IN (?, ?)", tripID, model.OrderStatusPending, model.OrderStatusPaid).
			Scan(&rows).Error
	if err != nil {
		return nil
	}

	// 按站序排序
	sorted := make([]model.RouteStation, len(routeStations))
	copy(sorted, routeStations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StopOrder < sorted[j].StopOrder })

	// 同样用差分数组，跟RealtimeAvailableSeats一个思路
	maxOrder := 0
	for _, r := range rows {
		if r.ToOrder > maxOrder {
			maxOrder = r.ToOrder
		}
	}
	// 同时考虑线路站点的最大stop_order，防止数组越界
	for _, rs := range routeStations {
		if rs.StopOrder > maxOrder {
			maxOrder = rs.StopOrder
		}
	}
	diffBoarding := make([]int, maxOrder+2)
	diffAlighting := make([]int, maxOrder+2)
	for _, r := range rows {
		if r.FromOrder >= 0 && r.FromOrder < len(diffBoarding) {
			diffBoarding[r.FromOrder] += r.Count
		}
		if r.ToOrder >= 0 && r.ToOrder < len(diffAlighting) {
			diffAlighting[r.ToOrder] += r.Count
		}
	}

	stats := make([]StationBoardingStat, 0, len(sorted))
	cumulativeOnboard := 0

	for _, rs := range sorted {
		var boarding, alighting int
		if rs.StopOrder >= 0 && rs.StopOrder < len(diffBoarding) {
			boarding = diffBoarding[rs.StopOrder]
			alighting = diffAlighting[rs.StopOrder]
		}
		cumulativeOnboard += boarding - alighting
		if cumulativeOnboard < 0 {
			cumulativeOnboard = 0
		}
		avail := totalSeats - cumulativeOnboard
		if avail < 0 {
			avail = 0
		}
		stationName := ""
		if rs.Station != nil {
			stationName = rs.Station.Name
		}
		stats = append(stats, StationBoardingStat{
			StopOrder:   rs.StopOrder,
			StationID:    rs.StationID,
			StationName:  stationName,
			Boarding:     boarding,
			Alighting:    alighting,
			Onboard:      cumulativeOnboard,
			Available:    avail,
		})
	}

	return stats
}

// AvailableSeatsForSegment 算某个区间还能卖几张票
// 区间复用：同一座位在不同区段可以重复卖
// 比如1→3和3→8能共用一个座，3站下车3站上车不冲突
//
// 逐座重叠检查：查询已分配座位的乘客区间，统计与目标区间[fromOrder, toOrder)
// 有重叠的座位数，可用 = 总座 - 被占座数。
// 与 AssignSeats 使用完全相同的查询和重叠判断逻辑，保证预检与实际分座结果一致。
//
// 之前用差分数组算 maxOcc，对部分区间会高估：
//   座1占[1,2)、座2占[2,3)，算[1,3)区间 maxOcc=1 → 返回2座可用
//   但全程[1,3)实际只有座3可用（座1和座2部分占用 → 不可全程使用）
//   导致预检通过但 AssignSeats 失败，用户看到“余座不足”报错。
func AvailableSeatsForSegment(tx *gorm.DB, tripID uint, totalSeats, fromOrder, toOrder int) (int, error) {
	if fromOrder >= toOrder {
		return totalSeats, nil
	}

	// 查询已分配座位的乘客区间（与 AssignSeats 完全相同的查询）
	var assignments []seatAssignment
	err := tx.Table("order_passengers").
		Select("order_passengers.seat_no AS seat_no, rs_from.stop_order AS from_order, rs_to.stop_order AS to_order").
		Joins("JOIN orders ON orders.id = order_passengers.order_id").
		Joins("JOIN route_stations rs_from ON rs_from.route_id = orders.route_id AND rs_from.station_id = orders.from_station_id").
		Joins("JOIN route_stations rs_to ON rs_to.route_id = orders.route_id AND rs_to.station_id = orders.to_station_id").
		Where("orders.trip_id = ? AND orders.order_type = 1 AND orders.status IN (?, ?) AND order_passengers.seat_no != ''", tripID, model.OrderStatusPending, model.OrderStatusPaid).
		Scan(&assignments).Error
	if err != nil {
		return 0, err
	}
	if len(assignments) == 0 {
		return totalSeats, nil
	}

	// 统计被占用的座位号（区间与目标区间有重叠的座位）
	occupiedSeats := make(map[int]bool)
	for _, a := range assignments {
		seatNum, convErr := strconv.Atoi(a.SeatNo)
		if convErr != nil {
			continue // 座位号不合法，跳过
		}
		// 半开区间重叠检查：[fromOrder, toOrder) 与 [a.FromOrder, a.ToOrder) 有交集则该座被占
		// 与 AssignSeats 中的重叠判断完全一致
		if fromOrder < a.ToOrder && toOrder > a.FromOrder {
			occupiedSeats[seatNum] = true
		}
	}

	avail := totalSeats - len(occupiedSeats)
	if avail < 0 {
		avail = 0
	}
	return avail, nil
}
