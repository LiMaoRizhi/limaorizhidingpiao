// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// JSONDate 自定义日期类型（string 别名），Scan 时确保格式为 "2006-01-02"
// 解决 MySQL parseTime=True 导致 DATE 字段返回 "2026-07-21T00:00:00+08:00" 的问题
// 作为 string 别名，所有使用 trip_date 的字符串操作无需改动
type JSONDate string

// Scan 实现 sql.Scanner 接口，从数据库读取时统一格式化
func (d *JSONDate) Scan(value interface{}) error {
	switch v := value.(type) {
	case time.Time:
		*d = JSONDate(v.Format("2006-01-02"))
		return nil
	case []byte:
		s := string(v)
		// 尝试解析为 time 再格式化（处理 "2026-07-21T00:00:00+08:00" 等格式）
		for _, f := range []string{"2006-01-02 15:04:05", "2006-01-02", time.RFC3339} {
			if parsed, err := time.Parse(f, s); err == nil {
				*d = JSONDate(parsed.Format("2006-01-02"))
				return nil
			}
		}
		// 如果长度>=10，直接取前10位（纯日期）
		if len(s) >= 10 {
			*d = JSONDate(s[:10])
			return nil
		}
		*d = JSONDate(s)
		return nil
	case string:
		for _, f := range []string{"2006-01-02 15:04:05", "2006-01-02", time.RFC3339} {
			if parsed, err := time.Parse(f, v); err == nil {
				*d = JSONDate(parsed.Format("2006-01-02"))
				return nil
			}
		}
		if len(v) >= 10 {
			*d = JSONDate(v[:10])
			return nil
		}
		*d = JSONDate(v)
		return nil
	case nil:
		*d = ""
		return nil
	}
	return fmt.Errorf("JSONDate.Scan: 不支持的类型 %T", value)
}

// Value 实现 driver.Valuer 接口，写入数据库
func (d JSONDate) Value() (driver.Value, error) {
	return string(d), nil
}

// JSONTime 自定义时间类型，JSON序列化输出 "2006-01-02 15:04:05" 格式
// 用于 CreatedAt、UpdatedAt、PayTime 等需要精确到秒的字段
type JSONTime time.Time

// MarshalJSON 实现 json.Marshaler 接口
func (t JSONTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", tt.Format("2006-01-02 15:04:05"))), nil
}

// Scan 实现 sql.Scanner 接口
func (t *JSONTime) Scan(value interface{}) error {
	switch v := value.(type) {
	case time.Time:
		*t = JSONTime(v)
		return nil
	case []byte:
		for _, f := range []string{"2006-01-02 15:04:05", "2006-01-02", time.RFC3339} {
			if parsed, err := time.Parse(f, string(v)); err == nil {
				*t = JSONTime(parsed)
				return nil
			}
		}
		return fmt.Errorf("JSONTime.Scan: 无法解析 %s", string(v))
	case string:
		for _, f := range []string{"2006-01-02 15:04:05", "2006-01-02", time.RFC3339} {
			if parsed, err := time.Parse(f, v); err == nil {
				*t = JSONTime(parsed)
				return nil
			}
		}
		return fmt.Errorf("JSONTime.Scan: 无法解析 %s", v)
	case nil:
		*t = JSONTime(time.Time{})
		return nil
	}
	return fmt.Errorf("JSONTime.Scan: 不支持的类型 %T", value)
}

// Value 实现 driver.Valuer 接口
func (t JSONTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

// String 返回字符串表示
func (t JSONTime) String() string {
	return time.Time(t).Format("2006-01-02 15:04:05")
}

// Format 代理 time.Time.Format，保持兼容
func (t JSONTime) Format(layout string) string {
	return time.Time(t).Format(layout)
}

// IsZero 判断是否零值
func (t JSONTime) IsZero() bool {
	return time.Time(t).IsZero()
}

// After 代理 time.Time.After
func (t JSONTime) After(u JSONTime) bool {
	return time.Time(t).After(time.Time(u))
}

// Before 代理 time.Time.Before
func (t JSONTime) Before(u JSONTime) bool {
	return time.Time(t).Before(time.Time(u))
}
