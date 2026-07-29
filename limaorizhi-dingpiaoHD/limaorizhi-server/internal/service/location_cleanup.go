// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"log"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// StartLocationCleaner 启动车辆位置记录清理定时任务
// 每6小时清理一次，删除超过7天的历史位置记录，防止表无限增长
func StartLocationCleaner(db *gorm.DB) {
	startTickerTask("车辆位置清理", 5*time.Minute, 6*time.Hour, cleanOldLocations, db)
}

// cleanOldLocations 清理超过7天的车辆位置记录
func cleanOldLocations(db *gorm.DB) {
	threshold := time.Now().AddDate(0, 0, -7) // 7天前
	result := db.Where("reported_at < ?", threshold).Delete(&model.VehicleLocation{})
	if result.Error != nil {
		log.Printf("清理车辆位置记录失败: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("已清理 %d 条过期车辆位置记录（7天前）", result.RowsAffected)
	}
}
