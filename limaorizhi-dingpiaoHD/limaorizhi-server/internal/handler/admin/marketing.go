// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"strconv"
	"strings"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/sanitize"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 优惠券管理

type CouponHandler struct{ DB *gorm.DB }

func NewCouponHandler(db *gorm.DB) *CouponHandler { return &CouponHandler{DB: db} }

func (h *CouponHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := h.DB.Model(&model.Coupon{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+sanitize.EscapeLikePattern(name)+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	var list []model.Coupon
	query.Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

// createCouponRequest 优惠券创建DTO（白名单，防止Mass Assignment）
type createCouponRequest struct {
	Name          string  `json:"name" binding:"required"`
	Type          int8    `json:"type" binding:"required"`
	DiscountValue float64 `json:"discount_value" binding:"required"`
	MinSpend      float64 `json:"min_spend"`
	ValidDays     int     `json:"valid_days"`
	TotalCount    int     `json:"total_count"`
	Status        int8    `json:"status"`
}

func (h *CouponHandler) Create(c *gin.Context) {
	var req createCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.Name == "" {
		response.FailMsg(c, response.CodeParamError, "优惠券名称不能为空")
		return
	}
	if req.Type < 1 || req.Type > 3 {
		response.FailMsg(c, response.CodeParamError, "优惠券类型不合法")
		return
	}
	if req.DiscountValue <= 0 {
		response.FailMsg(c, response.CodeParamError, "优惠面值必须大于0")
		return
	}
	if req.Type == 2 && (req.DiscountValue <= 0 || req.DiscountValue >= 10) {
		response.FailMsg(c, response.CodeParamError, "折扣值必须在0~10之间(如8.5表示85折)")
		return
	}
	if req.MinSpend < 0 {
		response.FailMsg(c, response.CodeParamError, "最低消费门槛不能为负数")
		return
	}
	if req.ValidDays <= 0 {
		req.ValidDays = 30
	}
	cp := model.Coupon{
		Name:          req.Name,
		Type:          req.Type,
		DiscountValue: req.DiscountValue,
		MinSpend:      req.MinSpend,
		ValidDays:     req.ValidDays,
		TotalCount:    req.TotalCount,
		Status:        req.Status,
	}
	if err := h.DB.Create(&cp).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	WriteLog(c, h.DB, "优惠券", "新增", cp.Name, fmt.Sprintf("优惠券ID:%d 名称:%s", cp.ID, cp.Name))
	response.OK(c, cp)
}

func (h *CouponHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var cp model.Coupon
	if err := h.DB.First(&cp, id).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "优惠券不存在")
		return
	}
	var req struct {
		Name          string  `json:"name"`
		Type          int8    `json:"type"`
		DiscountValue float64 `json:"discount_value"`
		MinSpend      float64 `json:"min_spend"`
		ValidDays     int     `json:"valid_days"`
		TotalCount    int     `json:"total_count"`
		Status        int8    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.Name == "" {
		response.FailMsg(c, response.CodeParamError, "优惠券名称不能为空")
		return
	}
	if req.Type < 1 || req.Type > 3 {
		response.FailMsg(c, response.CodeParamError, "优惠券类型不合法")
		return
	}
	if req.DiscountValue <= 0 {
		response.FailMsg(c, response.CodeParamError, "优惠面值必须大于0")
		return
	}
	if req.Type == 2 && (req.DiscountValue <= 0 || req.DiscountValue >= 10) {
		response.FailMsg(c, response.CodeParamError, "折扣值必须在0~10之间")
		return
	}
	if req.MinSpend < 0 {
		response.FailMsg(c, response.CodeParamError, "最低消费门槛不能为负数")
		return
	}
	if req.ValidDays <= 0 {
		req.ValidDays = 30
	}
	if err := h.DB.Model(&cp).Updates(map[string]interface{}{
		"name": req.Name, "type": req.Type, "discount_value": req.DiscountValue,
		"min_spend": req.MinSpend, "valid_days": req.ValidDays,
		"total_count": req.TotalCount, "status": req.Status,
	}).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	h.DB.First(&cp, id)
	WriteLog(c, h.DB, "优惠券", "编辑", cp.Name, fmt.Sprintf("优惠券ID:%s 名称:%s", id, cp.Name))
	response.OK(c, cp)
}

func (h *CouponHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var issuedCount int64
	h.DB.Model(&model.UserCoupon{}).Where("coupon_id = ?", id).Count(&issuedCount)
	if issuedCount > 0 {
		response.FailMsg(c, response.CodeServerError, "该优惠券已发放过，无法删除")
		return
	}
	if err := h.DB.Delete(&model.Coupon{}, id).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}
	WriteLog(c, h.DB, "优惠券", "删除", id, "删除优惠券ID:"+id)
	response.OKMsg(c, "删除成功", nil)
}

// PublicHomeCoupons 公开接口：返回首页展示的优惠券（供小程序首页调用）
// 从系统配置 homepage_coupon_ids 读取选中的优惠券ID，返回对应优惠券详情
func (h *CouponHandler) PublicHomeCoupons(c *gin.Context) {
	var cfg model.SystemConfig
	if err := h.DB.Where("config_key = ?", "homepage_coupon_ids").First(&cfg).Error; err != nil {
		response.OK(c, []interface{}{})
		return
	}
	if cfg.ConfigValue == "" {
		response.OK(c, []interface{}{})
		return
	}
	// 解析逗号分隔的ID列表
	idStrs := strings.Split(cfg.ConfigValue, ",")
	var ids []uint
	for _, s := range idStrs {
		if id, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64); err == nil && id > 0 {
			ids = append(ids, uint(id))
		}
	}
	if len(ids) == 0 {
		response.OK(c, []interface{}{})
		return
	}
	var coupons []model.Coupon
	h.DB.Where("id IN ? AND status = 1", ids).Order("id DESC").Find(&coupons)
	response.OK(c, coupons)
}

// 发放记录

type UserCouponHandler struct{ DB *gorm.DB }

func NewUserCouponHandler(db *gorm.DB) *UserCouponHandler { return &UserCouponHandler{DB: db} }

func (h *UserCouponHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := h.DB.Model(&model.UserCoupon{})
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if couponID := c.Query("coupon_id"); couponID != "" {
		query = query.Where("coupon_id = ?", couponID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	var list []model.UserCoupon
	query.Preload("User").Preload("Coupon").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

// 积分规则

type PointRuleHandler struct{ DB *gorm.DB }

func NewPointRuleHandler(db *gorm.DB) *PointRuleHandler { return &PointRuleHandler{DB: db} }

func (h *PointRuleHandler) List(c *gin.Context) {
	var list []model.PointRule
	h.DB.Order("rule_type ASC, id ASC").Find(&list)
	response.OK(c, list)
}

// createPointRuleRequest 积分规则创建DTO（白名单，防止Mass Assignment）
type createPointRuleRequest struct {
	RuleName      string  `json:"rule_name" binding:"required"`
	RuleType      int8    `json:"rule_type"`
	PointsPerYuan float64 `json:"points_per_yuan"`
	FixedPoints   int     `json:"fixed_points"`
	Description   string  `json:"description"`
	Status        int8    `json:"status"`
}

func (h *PointRuleHandler) Create(c *gin.Context) {
	var req createPointRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.RuleName == "" {
		response.FailMsg(c, response.CodeParamError, "规则名称不能为空")
		return
	}
	if req.RuleType < 1 || req.RuleType > 3 {
		req.RuleType = 1
	}
	r := model.PointRule{
		RuleName:      req.RuleName,
		RuleType:      req.RuleType,
		PointsPerYuan: req.PointsPerYuan,
		FixedPoints:   req.FixedPoints,
		Description:   req.Description,
		Status:        req.Status,
	}
	if err := h.DB.Create(&r).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	WriteLog(c, h.DB, "积分规则", "新增", r.RuleName, fmt.Sprintf("规则ID:%d 名称:%s", r.ID, r.RuleName))
	response.OK(c, r)
}

func (h *PointRuleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var r model.PointRule
	if err := h.DB.First(&r, id).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "积分规则不存在")
		return
	}
	var req struct {
		RuleName      string  `json:"rule_name"`
		RuleType      int8    `json:"rule_type"`
		PointsPerYuan float64 `json:"points_per_yuan"`
		FixedPoints   int     `json:"fixed_points"`
		Description   string `json:"description"`
		Status        int8   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.RuleName == "" {
		response.FailMsg(c, response.CodeParamError, "规则名称不能为空")
		return
	}
	if req.RuleType < 1 || req.RuleType > 3 {
		req.RuleType = 1
	}
	if err := h.DB.Model(&r).Updates(map[string]interface{}{
		"rule_name": req.RuleName, "rule_type": req.RuleType,
		"points_per_yuan": req.PointsPerYuan, "fixed_points": req.FixedPoints,
		"description": req.Description, "status": req.Status,
	}).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	h.DB.First(&r, id)
	WriteLog(c, h.DB, "积分规则", "编辑", r.RuleName, fmt.Sprintf("规则ID:%s 名称:%s", id, r.RuleName))
	response.OK(c, r)
}

func (h *PointRuleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.DB.Delete(&model.PointRule{}, id).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}
	WriteLog(c, h.DB, "积分规则", "删除", id, "删除积分规则ID:"+id)
	response.OKMsg(c, "删除成功", nil)
}

// 用户积分

type UserPointsHandler struct{ DB *gorm.DB }

func NewUserPointsHandler(db *gorm.DB) *UserPointsHandler { return &UserPointsHandler{DB: db} }

func (h *UserPointsHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := h.DB.Model(&model.UserPoints{})
	// 支持按用户手机号搜索
	if phone := c.Query("phone"); phone != "" {
		query = query.Joins("LEFT JOIN users ON users.id = user_points.user_id").Where("users.phone LIKE ?", "%"+sanitize.EscapeLikePattern(phone)+"%")
	}
	query.Count(&total)

	var list []model.UserPoints
	query.Preload("User").
		Order("user_points.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

// Records 查询某用户的积分明细记录
func (h *UserPointsHandler) Records(c *gin.Context) {
	userID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := h.DB.Model(&model.PointRecord{}).Where("user_id = ?", userID)
	query.Count(&total)

	var list []model.PointRecord
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

type adjustPointsRequest struct {
	Points int    `json:"points" binding:"required"`
	Remark string `json:"remark"`
}

// Adjust 手动调整用户积分（正数加/负数减）
func (h *UserPointsHandler) Adjust(c *gin.Context) {
	id := c.Param("id")
	userID, _ := strconv.ParseUint(id, 10, 64)

	var req adjustPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.Points == 0 {
		response.FailMsg(c, response.CodeParamError, "调整积分不能为0")
		return
	}

	adminID := c.GetUint("admin_id")
	adminName, _ := c.Get("admin_name")
	nameStr, ok := adminName.(string)
	if !ok {
		nameStr = fmt.Sprintf("%v", adminName)
	}

	changeType := int8(1) // 获得
	absPoints := req.Points
	if req.Points < 0 {
		changeType = 2 // 消耗
		absPoints = -req.Points
	}

	// 使用事务+行锁防止并发调整丢失更新
	var up model.UserPoints
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 加行锁查询积分记录
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&up).Error; err != nil {
			// 不存在则创建
			up = model.UserPoints{UserID: uint(userID)}
			if err := tx.Create(&up).Error; err != nil {
				return fmt.Errorf("创建积分记录失败")
			}
		}

		// 检查余额是否充足（消耗时）
		if req.Points < 0 && up.Balance+req.Points < 0 {
			return fmt.Errorf("用户积分余额不足")
		}

		// 使用原子更新防止丢失更新
		updates := map[string]interface{}{
			"balance": gorm.Expr("balance + ?", req.Points),
		}
		if changeType == 1 {
			updates["total_earned"] = gorm.Expr("total_earned + ?", absPoints)
		} else {
			updates["total_spent"] = gorm.Expr("total_spent + ?", absPoints)
		}
		if err := tx.Model(&up).Updates(updates).Error; err != nil {
			return err
		}
		// 写入积分明细
		record := model.PointRecord{
			UserID:     uint(userID),
			ChangeType: changeType,
			Points:     absPoints,
			Source:     "manual",
			Remark:     req.Remark,
			AdminID:    adminID,
			AdminName:  nameStr,
		}
		return tx.Create(&record).Error
	})

	if err != nil {
		errMsg := err.Error()
		if errMsg == "用户积分余额不足" {
			response.FailMsg(c, response.CodeParamError, errMsg)
			return
		}
		if errMsg == "创建积分记录失败" {
			response.FailMsg(c, response.CodeServerError, errMsg)
			return
		}
		response.FailMsg(c, response.CodeServerError, "积分调整失败")
		return
	}

	WriteLog(c, h.DB, "用户积分", "手动调整", id, fmt.Sprintf("用户ID:%s 调整积分:%d 备注:%s", id, req.Points, req.Remark))
	response.OKMsg(c, "积分调整成功", nil)
}
