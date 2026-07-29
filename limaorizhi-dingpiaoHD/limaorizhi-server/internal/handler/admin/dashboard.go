// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	DB *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{DB: db}
}

// allowedSumColumns 允许在 SUM 表达式中拼接的列/表达式白名单，杜绝SQL注入风险
var allowedSumColumns = map[string]bool{
	"total_price":                   true,
	"total_seats":                   true,
	"total_seats - available_seats": true,
}

// sumFloat 辅助函数：求和float字段
func sumFloat(db *gorm.DB, model interface{}, column string, where string, args ...interface{}) float64 {
	if !allowedSumColumns[column] {
		return 0 // 非白名单列，拒绝拼接防SQL注入
	}
	var result struct{ Total float64 }
	db.Model(model).Where(where, args...).
		Select(fmt.Sprintf("COALESCE(SUM(%s), 0) as total", column)).
		Scan(&result)
	return result.Total
}

// sumInt 辅助函数：求和int字段
func sumInt(db *gorm.DB, model interface{}, column string, where string, args ...interface{}) int64 {
	if !allowedSumColumns[column] {
		return 0
	}
	var result struct{ Total int64 }
	db.Model(model).Where(where, args...).
		Select(fmt.Sprintf("COALESCE(SUM(%s), 0) as total", column)).
		Scan(&result)
	return result.Total
}

// Stats 仪表盘统计数据
func (h *DashboardHandler) Stats(c *gin.Context) {
	now := time.Now()
	today := now.Format("2006-01-02")

	// 本周起始（周一）
	weekStart := now.AddDate(0, 0, -int(now.Weekday())+1) // Sunday=0, so -int(weekday)+1 gives Monday
	if now.Weekday() == time.Sunday {
		weekStart = now.AddDate(0, 0, -6) // Sunday: go back 6 days to Monday
	}
	weekStartStr := weekStart.Format("2006-01-02")

	// 今日订单数
	var todayOrders int64
	h.DB.Model(&model.Order{}).Where("CAST(created_at AS DATE) = ?", today).Count(&todayOrders)

	// 今日营收
	todayRevenue := sumFloat(h.DB, &model.Order{}, "total_price", "CAST(created_at AS DATE) = ? AND status IN (1, 2)", today)

	// 今日新增用户
	var todayUsers int64
	h.DB.Model(&model.User{}).Where("CAST(created_at AS DATE) = ?", today).Count(&todayUsers)

	// 本周订单数
	var weekOrders int64
	h.DB.Model(&model.Order{}).Where("CAST(created_at AS DATE) >= ?", weekStartStr).Count(&weekOrders)

	// 本周营收
	weekRevenue := sumFloat(h.DB, &model.Order{}, "total_price", "CAST(created_at AS DATE) >= ? AND status IN (1, 2)", weekStartStr)

	// 总用户数
	var totalUsers int64
	h.DB.Model(&model.User{}).Count(&totalUsers)

	// 总订单数
	var totalOrders int64
	h.DB.Model(&model.Order{}).Count(&totalOrders)

	// 总营收
	totalRevenue := sumFloat(h.DB, &model.Order{}, "total_price", "status IN (1, 2)")

	// 退票率
	var refundedCount int64
	h.DB.Model(&model.Order{}).Where("status = 3").Count(&refundedCount)
	var refundRate float64
	if totalOrders > 0 {
		refundRate = float64(refundedCount) / float64(totalOrders) * 100
	}

	// 今日上座率
	var todayTotalSeats int64
	h.DB.Model(&model.Trip{}).Where("trip_date = ? AND status IN (1, 2)", today).
		Select("COALESCE(SUM(total_seats), 0)").Scan(&todayTotalSeats)
	// 从订单表查询今日实际已售座位数（已支付+已完成的订单乘客数之和）
	var todaySoldSeats int64
	h.DB.Model(&model.Order{}).
		Joins("JOIN trips ON trips.id = orders.trip_id").
		Where("trips.trip_date = ? AND orders.order_type = 1 AND orders.status IN (1, 2)", today).
		Select("COALESCE(SUM(orders.passenger_count), 0)").Scan(&todaySoldSeats)
	var seatOccupancy float64
	if todayTotalSeats > 0 {
		seatOccupancy = float64(todaySoldSeats) / float64(todayTotalSeats) * 100
	}

	// 可售班次数
	var activeTrips int64
	h.DB.Model(&model.Trip{}).Where("status = 1 AND trip_date >= ?", today).Count(&activeTrips)

	// 近30天订单趋势（单次GROUP BY查询替代30*2=60次查询）
	type trendItem struct {
		Date    string  `json:"date"`
		Orders  int64   `json:"orders"`
		Revenue float64 `json:"revenue"`
	}
	var trend []trendItem
	thirtyDaysAgo := now.AddDate(0, 0, -29).Format("2006-01-02")
	h.DB.Model(&model.Order{}).
		Select("CAST(created_at AS DATE) as date, COUNT(*) as orders, COALESCE(SUM(CASE WHEN status IN (1,2) THEN total_price ELSE 0 END), 0) as revenue").
		Where("CAST(created_at AS DATE) >= ?", thirtyDaysAgo).
		Group("CAST(created_at AS DATE)").
		Order("date ASC").
		Scan(&trend)
	// 补全无数据的日期
	trendMap := make(map[string]trendItem)
	for _, t := range trend {
		trendMap[t.Date] = t
	}
	trend = nil
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		if item, ok := trendMap[date]; ok {
			trend = append(trend, item)
		} else {
			trend = append(trend, trendItem{Date: date, Orders: 0, Revenue: 0})
		}
	}

	// 热门线路Top10
	type hotRoute struct {
		RouteID    uint    `json:"route_id"`
		RouteName  string  `json:"route_name"`
		OrderCount int64   `json:"order_count"`
		Revenue    float64 `json:"revenue"`
	}
	var routeRank []hotRoute
	h.DB.Table("orders o").
		Select("o.route_id, r.name as route_name, COUNT(*) as order_count, COALESCE(SUM(o.total_price), 0) as revenue").
		Joins("LEFT JOIN routes r ON r.id = o.route_id").
		Where("o.status IN (1, 2)").
		Group("o.route_id, r.name").
		Order("order_count DESC").
		Limit(10).
		Scan(&routeRank)

	// 今日按小时订单分布（单次GROUP BY查询替代24*2=48次查询）
	type hourlyItem struct {
		Hour    string  `json:"hour"`
		Orders  int64   `json:"orders"`
		Revenue float64 `json:"revenue"`
	}
	var hourly []hourlyItem
	h.DB.Model(&model.Order{}).
		Select("EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as orders, COALESCE(SUM(CASE WHEN status IN (1,2) THEN total_price ELSE 0 END), 0) as revenue").
		Where("CAST(created_at AS DATE) = ?", today).
		Group("EXTRACT(HOUR FROM created_at)").
		Order("hour ASC").
		Scan(&hourly)
	// 补全无数据的小时
	hourlyMap := make(map[int]hourlyItem)
	for _, h2 := range hourly {
		var hourInt int
		fmt.Sscanf(h2.Hour, "%d", &hourInt)
		h2.Hour = fmt.Sprintf("%02d:00", hourInt)
		hourlyMap[hourInt] = h2
	}
	hourly = nil
	for hour := 0; hour < 24; hour++ {
		if item, ok := hourlyMap[hour]; ok {
			hourly = append(hourly, item)
		} else {
			hourly = append(hourly, hourlyItem{Hour: fmt.Sprintf("%02d:00", hour), Orders: 0, Revenue: 0})
		}
	}

	// 最近10笔订单
	var recentOrders []model.Order
	h.DB.Preload("FromStation").Preload("ToStation").
		Order("created_at DESC").Limit(10).Find(&recentOrders)
	model.MaskOrders(recentOrders)

	response.OK(c, gin.H{
		"today_orders":    todayOrders,
		"today_revenue":   todayRevenue,
		"today_users":     todayUsers,
		"week_orders":     weekOrders,
		"week_revenue":    weekRevenue,
		"total_users":     totalUsers,
		"total_orders":    totalOrders,
		"total_revenue":   totalRevenue,
		"refund_rate":     refundRate,
		"seat_occupancy":  seatOccupancy,
		"active_trips":    activeTrips,
		"trend":           trend,
		"route_rank":      routeRank,
		"hourly":          hourly,
		"recent_orders":   recentOrders,
	})
}
