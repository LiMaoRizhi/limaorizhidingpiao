package service

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// startTickerTask 启动固定间隔定时任务（goroutine + recover + 初始延迟 + ticker循环）
// 统一定时任务骨架，避免每个任务文件重复编写 recover/延迟/ticker 模板代码
// 6月18号抽出来的，之前每个定时任务都重复写 recover 那段，冗余得很
// name: 任务名称（用于日志标识）
// initialDelay: 启动后初始等待时间（避免与启动初始化冲突）
// interval: 执行间隔
// task: 实际执行函数
func startTickerTask(name string, initialDelay, interval time.Duration, task func(*gorm.DB), db *gorm.DB) {
	go func() {
		time.Sleep(initialDelay)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			// 每次 task 调用独立 recover，避免单次 panic 导致整个定时任务永久死亡
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[PANIC] %s定时任务panic: %v", name, r)
					}
				}()
				task(db)
			}()
		}
	}()
	log.Printf("%s定时任务已启动（每%s执行一次）", name, interval)
}

// startDailyTask 启动每日定时任务（goroutine + recover + 每天指定时刻执行）
// hour: 执行时刻（0-23），如 3 表示每天凌晨3点
// 5月16号写的这个，日志清理和优惠券过期用的它
func startDailyTask(name string, hour int, task func(*gorm.DB), db *gorm.DB) {
	go func() {
		time.Sleep(10 * time.Minute) // 启动后等待10分钟再开始第一次执行

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, hour, 0, 0, 0, now.Location())
			timer := time.NewTimer(next.Sub(now))
			<-timer.C
			timer.Stop()
			// 每次 task 调用独立 recover，避免单次 panic 导致整个定时任务永久死亡
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[PANIC] %s定时任务panic: %v", name, r)
					}
				}()
				task(db)
			}()
		}
	}()
	log.Printf("%s定时任务已启动（每天%d点执行）", name, hour)
}
