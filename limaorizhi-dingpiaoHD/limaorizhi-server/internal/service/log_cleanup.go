package service

import (
	"log"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// StartLogCleaner 启动操作日志清理定时任务
// 每天凌晨3点执行一次，删除超过当月天数的操作日志
// 保留天数按月份动态计算：31天的月份保留31天，30天的月份保留30天，2月保留28/29天
func StartLogCleaner(db *gorm.DB) {
	startDailyTask("操作日志清理", 3, cleanOldLogs, db)
}

// cleanOldLogs 清理超过当月天数的操作日志
func cleanOldLogs(db *gorm.DB) {
	now := time.Now()

	// 根据当前月份计算保留天数
	// 取上个月同一天作为阈值，即保留"上个月今天"到"今天"之间的日志
	// 效果：31天的月份保留31天，30天的月份保留30天，2月保留28/29天
	threshold := now.AddDate(0, -1, 0) // 上个月今天

	result := db.Where("created_at < ?", threshold).Delete(&model.OperationLog{})
	if result.Error != nil {
		log.Printf("清理操作日志失败: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("已清理 %d 条过期操作日志（保留至 %s）", result.RowsAffected, threshold.Format("2006-01-02"))
	}
}
