package service

import (
	"log"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// StartTripAutoCompleter 启动班次自动完成定时任务
// 每5分钟扫描一次，将已过发车时间的班次标记为已发车，到达终点后自动完成订单
func StartTripAutoCompleter(db *gorm.DB) {
	startTickerTask("班次自动完成", 60*time.Second, 5*time.Minute, autoCompleteTrips, db)
}

// CompleteTrip 班次到达终点完成处理：status 2→4 + 已支付订单→已完成/已到达 + 待支付→已取消
// 必须在事务内调用，使用条件更新（WHERE status=2）防止与用户退款/取消竞态
// 司机手动结束(EndTrip)与定时任务自动完成(completeArrivedTrip)共用此逻辑，避免重复实现导致行为不一致
// 返回 tripCompleted 表示班次是否实际完成（用于触发到达通知）
func CompleteTrip(tx *gorm.DB, tripID uint) (tripCompleted bool, completedOrders int, cancelledPending int, err error) {
	tripResult := tx.Model(&model.Trip{}).Where("id = ? AND status = ?", tripID, model.TripStatusDepart).Update("status", model.TripStatusFinish)
	if tripResult.Error != nil {
		return false, 0, 0, tripResult.Error
	}
	if tripResult.RowsAffected == 0 {
		return false, 0, 0, nil // 班次状态已被其他进程修改，跳过
	}
	// 已支付订单 → 完成（车票）或已到达（托运）
	// 车票订单：1=待出行 → 2=已完成；5=已核销（已上车）→ 2=已完成；托运订单：1=待运输 → 3=已到达
	ticketResult := tx.Model(&model.Order{}).
		Where("trip_id = ? AND status IN (?, ?) AND order_type = 1", tripID, model.OrderStatusPaid, model.OrderStatusPickedUp).
		Update("status", model.OrderStatusCompleted)
	if ticketResult.Error != nil {
		return false, 0, 0, ticketResult.Error
	}
	cargoResult := tx.Model(&model.Order{}).
		Where("trip_id = ? AND status = ? AND order_type = 2", tripID, model.OrderStatusPaid).
		Update("status", 3)
	if cargoResult.Error != nil {
		return false, 0, 0, cargoResult.Error
	}
	completedOrders = int(ticketResult.RowsAffected + cargoResult.RowsAffected)
	// 待支付订单 → 已取消（班次已到终点，无法再支付）
	var pendingOrders []model.Order
	if err := tx.Where("trip_id = ? AND status = ?", tripID, model.OrderStatusPending).Find(&pendingOrders).Error; err != nil {
		return false, 0, 0, err
	}
	for _, po := range pendingOrders {
		res := tx.Model(&model.Order{}).Where("id = ? AND status = ?", po.ID, model.OrderStatusPending).Update("status", model.OrderStatusCancelled)
		if res.Error != nil {
			return false, 0, 0, res.Error
		}
		if res.RowsAffected > 0 {
			cancelledPending++
			// 归还该订单绑定的优惠券
			if err := ReturnOrderCoupon(tx, po.ID); err != nil {
				return false, 0, 0, err
			}
		}
	}
	return true, completedOrders, cancelledPending, nil
}

// completeArrivedTrip 班次到达终点后完成处理（复用 CompleteTrip，确保与司机手动结束行为一致）
// 并发安全：使用条件更新（WHERE status=2）防止与用户退款/取消操作竞态
func completeArrivedTrip(db *gorm.DB, trip model.Trip) (int, int, int) {
	var tripCompleted bool
	var completedOrders, cancelledPending int
	err := db.Transaction(func(tx *gorm.DB) error {
		var e error
		tripCompleted, completedOrders, cancelledPending, e = CompleteTrip(tx, trip.ID)
		return e
	})
	if err != nil {
		log.Printf("完成班次 %s 失败: %v", trip.TripNo, err)
		return 0, 0, 0
	}
	if tripCompleted {
		// 订阅消息：班次到达终点通知（异步发送，不阻塞定时任务）
		NotifyTripArrival(db, trip)
	}
	if cancelledPending > 0 {
		log.Printf("班次 %s 到达终点，自动取消待支付订单 %d 条", trip.TripNo, cancelledPending)
	}
	return 1, completedOrders, cancelledPending
}

// departTrips 将符合条件的班次从 status=1(可售) 改为 status=2(已发车)
// 仅改状态，不处理订单（途经站乘客在车到站前仍可购票/支付）
// 并发安全：使用条件更新 WHERE status=1 防止与司机手动发车竞态
func departTrips(db *gorm.DB, query *gorm.DB, label string) int {
	var trips []model.Trip
	if err := query.Find(&trips).Error; err != nil {
		log.Printf("查询%s失败: %v", label, err)
		return 0
	}
	departed := 0
	for _, trip := range trips {
		result := db.Model(&model.Trip{}).Where("id = ? AND status = ?", trip.ID, model.TripStatusSale).Update("status", model.TripStatusDepart)
		if result.Error != nil {
			log.Printf("%s 班次 %s 失败: %v", label, trip.TripNo, result.Error)
		} else if result.RowsAffected > 0 {
			departed++
			// 订阅消息：班次发车通知（异步发送，不阻塞定时任务）
			NotifyTripDeparture(db, trip)
		}
	}
	return departed
}

// completeTrips 将符合条件的班次从 status=2(已发车) 改为 status=4(已完成)，并处理关联订单
// 对于 status=1 的过期班次，先改为 2 再走标准完成流程
func completeTrips(db *gorm.DB, query *gorm.DB, label string, forceDepart bool) (int, int, int) {
	var trips []model.Trip
	if err := query.Find(&trips).Error; err != nil {
		log.Printf("查询%s失败: %v", label, err)
		return 0, 0, 0
	}
	totalCt, totalCo, totalCp := 0, 0, 0
	for _, trip := range trips {
		if forceDepart {
			// 过期未发车班次：先改为已发车(2)，保证 completeArrivedTrip 的 WHERE status=2 能匹配
			db.Model(&model.Trip{}).Where("id = ? AND status = ?", trip.ID, model.TripStatusSale).Update("status", model.TripStatusDepart)
		}
		ct, co, cp := completeArrivedTrip(db, trip)
		totalCt += ct
		totalCo += co
		totalCp += cp
	}
	return totalCt, totalCo, totalCp
}

// autoCompleteTrips 自动完成已过发车时间的班次和订单
//
// 业务逻辑说明：
//   - 班次状态：1=可售 → 2=已发车（过发车时间触发，仅改状态不处理订单）
//               2=已发车 → 4=已完成（过到达时间触发，同时处理订单）
//   - 订单状态：0=待支付 → 4=已取消（班次到终点时处理）
//               1=已支付 → 2=已完成（班次到终点时处理）
//
// 班次发车后(status=2)，途经站乘客在车到站前仍可购票/支付
//   - 下单/支付时由 isStationPassed 判断车是否已驶过用户上车站
//   - 发车时不处理订单，避免途经站乘客的订单被误标记为已完成
//   - 订单完成/取消统一在班次到达终点(status 2→4)时处理
//
// 并发安全：条件更新（WHERE status=1/2）防止与用户退款/取消竞态
func autoCompleteTrips(db *gorm.DB) {
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	// 数据库 departure_time/arrival_time 为 HH:MM:SS，统一格式化为 15:04:05 再做字符串比较
	currentTimeStr := now.Format("15:04:05")

	totalDeparted := 0
	totalCompletedTrips := 0
	totalCompletedOrders := 0
	totalCancelledPending := 0

	// 发车判断：发车是当天行为，按 trip_date + departure_time（不涉及跨天偏移）
	// 到达判断：到达日期 = trip_date + arrival_day_offset 天，到达时刻 = arrival_time
	//   跨天班次（如下午发车、次日到达）的 trip_date 是发车日，arrival_day_offset=1
	//   必须用到达日期判断完成，否则字符串比较 arrival_time 会误判（次日08:00 < 当晚23:00）

	// 到达日期 = trip_date + arrival_day_offset 天
	// 完成条件：到达日期 < 今天（已过去）OR (到达日期 = 今天 AND arrival_time < 当前时刻)
	// 兼容 MySQL 和 PostgreSQL：MySQL 用 DATE_ADD(INTERVAL)，PostgreSQL 用 date + int
	var dateAddExpr string
	if db.Dialector.Name() == "postgres" {
		dateAddExpr = "(trip_date + arrival_day_offset)"
	} else {
		dateAddExpr = "DATE_ADD(trip_date, INTERVAL arrival_day_offset DAY)"
	}
	arrPassedToday := "(" + dateAddExpr + " = ? AND arrival_time < ?)"
	arrPassedBefore := dateAddExpr + " < ?"

	// 1. 今天已过发车时间、状态1(可售)、已分配司机 → 发车(1→2)
	//    注意：发车时不处理订单！途经站乘客在车到站前仍可购票/支付
	//    待支付订单由支付时的 isStationPassed 检查兜底（车过站则支付被拒）
	//    订单完成/取消统一在班次到达终点(status 2→4)时处理
	totalDeparted += departTrips(db,
		db.Where("status = ? AND trip_date = ? AND departure_time < ? AND driver_id > 0", model.TripStatusSale, todayStr, currentTimeStr),
		"今日待发车班次")

	// 2. 过去日期、状态1、已分配司机（定时任务停机遗留）→ 发车(1→2)
	totalDeparted += departTrips(db,
		db.Where("status = ? AND trip_date < ? AND driver_id > 0", model.TripStatusSale, todayStr),
		"历史待发车班次")

	// 3. 今天到达且到达时刻已过、状态2(已发车) → 完成(2→4) + 订单处理
	//    跨天班次：trip_date 可能是昨天，但到达日期(=trip_date+offset)=今天
	ct, co, cp := completeTrips(db,
		db.Where("status = ? AND "+arrPassedToday, model.TripStatusDepart, todayStr, currentTimeStr),
		"今日待完成班次", false)
	totalCompletedTrips += ct
	totalCompletedOrders += co
	totalCancelledPending += cp

	// 4. 到达日期已过去、状态2(已发车) → 完成(2→4) + 订单处理
	ct, co, cp = completeTrips(db,
		db.Where("status = ? AND "+arrPassedBefore, model.TripStatusDepart, todayStr),
		"历史已发车待完成班次", false)
	totalCompletedTrips += ct
	totalCompletedOrders += co
	totalCancelledPending += cp

	// 5. 今天到达且到达时刻已过、状态1(可售，未分配司机或司机未发车) → 先发车再完成 + 订单处理
	//    覆盖未分配司机或司机未发车的场景，防止过期班次仍显示为可售
	ct, co, cp = completeTrips(db,
		db.Where("status = ? AND "+arrPassedToday, model.TripStatusSale, todayStr, currentTimeStr),
		"今日过期未发车班次", true)
	totalCompletedTrips += ct
	totalCompletedOrders += co
	totalCancelledPending += cp

	// 6. 到达日期已过去、状态1(可售，无司机或定时任务遗漏) → 先发车再完成 + 订单处理
	ct, co, cp = completeTrips(db,
		db.Where("status = ? AND "+arrPassedBefore, model.TripStatusSale, todayStr),
		"历史过期未发车班次", true)
	totalCompletedTrips += ct
	totalCompletedOrders += co
	totalCancelledPending += cp

	if totalDeparted > 0 || totalCompletedTrips > 0 {
		log.Printf("本次处理：发车 %d 个班次，到达终点 %d 个班次/%d 订单完成/%d 待支付取消",
			totalDeparted, totalCompletedTrips, totalCompletedOrders, totalCancelledPending)
	}
}
