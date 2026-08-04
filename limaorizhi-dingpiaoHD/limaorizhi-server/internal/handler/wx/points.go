package wx

import (
	"fmt"
	"log"
	"math"
	"strconv"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// awardPoints 消费赠送积分（在事务内调用）
// 根据"消费赠送"积分规则，按消费金额计算积分并发放
func awardPoints(tx *gorm.DB, userID uint, amount float64, orderID uint, source string) error {
	// 查找"消费赠送"规则（RuleType=1, Status=1）
	var rule model.PointRule
	if err := tx.Where("rule_type = 1 AND status = 1").First(&rule).Error; err != nil {
		log.Printf("[WARN] awardPoints: 消费赠送积分规则不存在或未启用，跳过积分发放 userID=%d orderID=%d amount=%.2f\n", userID, orderID, amount)
		return nil
	}

	// 计算应得积分（四舍五入避免浮点截断少发积分）
	points := int(math.Round(amount * rule.PointsPerYuan))
	if points <= 0 {
		log.Printf("[INFO] awardPoints: 计算积分为0，跳过 userID=%d orderID=%d amount=%.2f pointsPerYuan=%.2f\n", userID, orderID, amount, rule.PointsPerYuan)
		return nil
	}

	log.Printf("[INFO] awardPoints: 准备发放积分 userID=%d orderID=%d amount=%.2f points=%d\n", userID, orderID, amount, points)
	return addPoints(tx, userID, points, 1, source, orderID, fmt.Sprintf("消费%.2f元赠送%d积分", amount, points))
}

// awardRegisterPoints 注册赠送积分（在创建用户后调用）
func awardRegisterPoints(db *gorm.DB, userID uint) {
	var rule model.PointRule
	if err := db.Where("rule_type = 2 AND status = 1").First(&rule).Error; err != nil {
		log.Printf("[WARN] awardRegisterPoints: 注册赠送积分规则不存在或未启用，跳过 userID=%d\n", userID)
		return
	}

	if rule.FixedPoints <= 0 {
		log.Printf("[INFO] awardRegisterPoints: 注册赠送固定积分为0，跳过 userID=%d\n", userID)
		return
	}

	if err := addPoints(db, userID, rule.FixedPoints, 2, "注册赠送", 0, fmt.Sprintf("新用户注册赠送%d积分", rule.FixedPoints)); err != nil {
		log.Printf("[ERROR] awardRegisterPoints: 注册赠送积分失败 userID=%d points=%d err=%v\n", userID, rule.FixedPoints, err)
	} else {
		log.Printf("[INFO] awardRegisterPoints: 注册赠送积分成功 userID=%d points=%d\n", userID, rule.FixedPoints)
	}
}

// addPoints 积分变动核心逻辑（在事务内调用）
// changeType: 1=获得 2=消耗
func addPoints(tx *gorm.DB, userID uint, points int, changeType int8, source string, orderID uint, remark string) error {
	// 查找或创建用户积分余额
	// 并发安全：用户首次获得积分时，多请求同时 First 都查不到记录，
	// 直接 Create 会因 user_id 唯一索引冲突报错。改用 OnConflict(DoNothing)，
	// 并发下若另一方已创建则静默跳过，后续原子更新不受影响。
	var up model.UserPoints
	result := tx.Where("user_id = ?", userID).First(&up)
	if result.Error != nil {
		up = model.UserPoints{UserID: userID, Balance: 0, TotalEarned: 0, TotalSpent: 0}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&up).Error; err != nil {
			return fmt.Errorf("创建积分记录失败: %w", err)
		}
		log.Printf("[INFO] addPoints: 创建用户积分记录 userID=%d\n", userID)
	}

	// 原子更新余额（防止并发丢失更新）
	if changeType == 1 {
		// 获得积分：原子增加
		if err := tx.Model(&model.UserPoints{}).Where("user_id = ?", userID).
			Updates(map[string]interface{}{
				"balance":      gorm.Expr("balance + ?", points),
				"total_earned": gorm.Expr("total_earned + ?", points),
			}).Error; err != nil {
			return fmt.Errorf("更新积分余额失败: %w", err)
		}
	} else {
		// 消耗积分：带条件原子扣减，防止透支
		result := tx.Model(&model.UserPoints{}).
			Where("user_id = ? AND balance >= ?", userID, points).
			Updates(map[string]interface{}{
				"balance":     gorm.Expr("balance - ?", points),
				"total_spent": gorm.Expr("total_spent + ?", points),
			})
		if result.Error != nil {
			return fmt.Errorf("更新积分余额失败: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("积分不足")
		}
	}

	// 建积分明细记录
	record := model.PointRecord{
		UserID:     userID,
		ChangeType: changeType,
		Points:     points,
		Source:     source,
		OrderID:    orderID,
		Remark:     remark,
	}
	if err := tx.Create(&record).Error; err != nil {
		return fmt.Errorf("创建积分明细记录失败: %w", err)
	}
	log.Printf("[INFO] addPoints: 积分变动成功 userID=%d changeType=%d points=%d source=%s orderID=%d\n", userID, changeType, points, source, orderID)
	return nil
}

// GetPoints 获取当前用户积分余额
func (h *UserHandler) GetPoints(c *gin.Context) {
	userID := c.GetUint("user_id")
	var up model.UserPoints
	err := h.DB.Where("user_id = ?", userID).First(&up).Error
	if err != nil {
		// 未找到记录表示积分为0
		response.OK(c, gin.H{"balance": 0, "total_earned": 0, "total_spent": 0})
		return
	}
	response.OK(c, gin.H{
		"balance":      up.Balance,
		"total_earned": up.TotalEarned,
		"total_spent":  up.TotalSpent,
	})
}

// GetPointRecords 获取当前用户积分明细（分页）
func (h *UserHandler) GetPointRecords(c *gin.Context) {
	userID := c.GetUint("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	h.DB.Model(&model.PointRecord{}).Where("user_id = ?", userID).Count(&total)

	var records []model.PointRecord
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&records)

	response.OK(c, gin.H{
		"list":      records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
