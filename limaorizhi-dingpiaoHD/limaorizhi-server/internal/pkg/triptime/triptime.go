package triptime

import (
	"time"
)

// Parse 解析班次日期+时间字符串为 time.Time（本地时区）
// 兼容 "HH:MM" 与 "HH:MM:SS" 两种时间格式：
//   - 班次 departure_time/arrival_time 由管理端 trips.vue 的 padTime 补全为 HH:MM:SS
//   - 线路站点 route_stations.arrival_time 由 routes.vue 直接保存 el-time-select 的 HH:MM
// 历史数据两种格式都可能存在，故按"具体格式优先"双次尝试，避免任一格式解析失败触发 fail-closed。
//
// dateStr 可能是数据库 date 列被 driver 补全的 RFC3339 格式，需先剥离为纯日期
func Parse(dateStr, timeStr string) (time.Time, error) {
	// dateStr 长度 > 10 说明不是纯日期（"2006-01-02" 长度=10），尝试当作 RFC3339 解析取日期部分
	if len(dateStr) > 10 {
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			dateStr = t.Format("2006-01-02")
		}
	}
	combined := dateStr + " " + timeStr
	// 优先 HH:MM:SS（更具体，匹配则直接返回）
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", combined, time.Local); err == nil {
		return t, nil
	}
	// 退回 HH:MM
	return time.ParseInLocation("2006-01-02 15:04", combined, time.Local)
}

// ParseWithOffset 解析班次日期+时间字符串为 time.Time，并按天数偏移调整日期
// dayOffset=0 表示发车当天，1=次日...用于跨省长途跨天班次（如下午发车、次日到达）
// 跨天班次的 arrival_time 仍是 HH:MM 时刻，但实际到达日期 = trip_date + dayOffset
func ParseWithOffset(dateStr, timeStr string, dayOffset int) (time.Time, error) {
	t, err := Parse(dateStr, timeStr)
	if err != nil {
		return t, err
	}
	return t.AddDate(0, 0, dayOffset), nil
}

// MustParse 解析班次日期+时间，解析失败返回零值（用于不需要错误处理的场景，如冲突检测）
func MustParse(dateStr, timeStr string) time.Time {
	t, _ := Parse(dateStr, timeStr)
	return t
}

// FormatDateTime 拼接班次日期+时间为 "2006-01-02 15:04" 格式字符串
// tripDate 可能是数据库 date 列被 driver 补全的 RFC3339 格式，需先剥离为纯日期
func FormatDateTime(tripDate, timeStr string) string {
	dateStr := tripDate
	if len(dateStr) > 10 {
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			dateStr = t.Format("2006-01-02")
		}
	}
	return dateStr + " " + timeStr
}
