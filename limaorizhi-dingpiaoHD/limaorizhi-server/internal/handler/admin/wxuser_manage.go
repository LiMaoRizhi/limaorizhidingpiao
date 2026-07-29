// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"strconv"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/sanitize"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 用户管理

type WxUserHandler struct{ DB *gorm.DB }

func NewWxUserHandler(db *gorm.DB) *WxUserHandler { return &WxUserHandler{DB: db} }

func (h *WxUserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.DB.Model(&model.User{})
	if phone := c.Query("phone"); phone != "" {
		query = query.Where("phone LIKE ?", "%"+sanitize.EscapeLikePattern(phone)+"%")
	}
	if nickname := c.Query("nickname"); nickname != "" {
		query = query.Where("nickname LIKE ?", "%"+sanitize.EscapeLikePattern(nickname)+"%")
	}
	query.Count(&total)

	var list []model.User
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	model.MaskUsers(list)

	// 批量查询当前页用户的订单统计（订单数 + 有效消费金额），避免 N+1 查询
	type orderStat struct {
		UserID      uint    `json:"user_id"`
		OrderCount  int64   `json:"order_count"`
		TotalAmount float64 `json:"total_amount"`
	}
	var stats []orderStat
	if len(list) > 0 {
		userIDs := make([]uint, len(list))
		for i, u := range list {
			userIDs[i] = u.ID
		}
		h.DB.Model(&model.Order{}).
			Select("user_id, COUNT(*) as order_count, COALESCE(SUM(CASE WHEN status IN (1, 2) THEN total_price ELSE 0 END), 0) as total_amount").
			Where("user_id IN ?", userIDs).
			Group("user_id").
			Scan(&stats)
	}
	statsMap := make(map[uint]orderStat, len(stats))
	for _, s := range stats {
		statsMap[s.UserID] = s
	}

	// 构造含统计字段的响应列表
	type userItem struct {
		model.User
		OrderCount  int64   `json:"order_count"`
		TotalAmount float64 `json:"total_amount"`
	}
	items := make([]userItem, len(list))
	for i, u := range list {
		items[i] = userItem{User: u}
		if s, ok := statsMap[u.ID]; ok {
			items[i].OrderCount = s.OrderCount
			items[i].TotalAmount = s.TotalAmount
		}
	}
	response.Page(c, items, total, page, pageSize)
}

func (h *WxUserHandler) Detail(c *gin.Context) {
	id := c.Param("id")
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		response.Fail(c, response.CodeUserNotFound)
		return
	}

	// 订单统计：总订单数 + 有效消费金额（已支付+已完成）
	var orderCount int64
	h.DB.Model(&model.Order{}).Where("user_id = ?", id).Count(&orderCount)
	var totalAmount float64
	h.DB.Model(&model.Order{}).Where("user_id = ? AND status IN (1, 2)", id).
		Select("COALESCE(SUM(total_price), 0)").Scan(&totalAmount)

	// 退票数
	var refundCount int64
	h.DB.Model(&model.Order{}).Where("user_id = ? AND status = 3", id).Count(&refundCount)

	// 托运单数
	var cargoCount int64
	h.DB.Model(&model.Order{}).Where("user_id = ? AND order_type = 2", id).Count(&cargoCount)

	// 最近20笔订单（Preload站点+班次信息，供前端展示路线）
	var orders []model.Order
	h.DB.Preload("FromStation").Preload("ToStation").Preload("Trip").
		Where("user_id = ?", id).Order("created_at DESC").Limit(20).Find(&orders)

	// 常用乘客
	var passengers []model.Passenger
	h.DB.Where("user_id = ?", id).Order("created_at DESC").Find(&passengers)

	user.Mask()
	model.MaskPassengerList(passengers)
	model.MaskOrders(orders)

	response.OK(c, gin.H{
		"user":          user,
		"order_count":   orderCount,
		"total_amount":   totalAmount,
		"refund_count":   refundCount,
		"cargo_count":    cargoCount,
		"orders":         orders,
		"passengers":     passengers,
	})
}

type updateUserStatusRequest struct {
	Status int8 `json:"status"`
}

func (h *WxUserHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req updateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 校验状态值合法性（1=正常 0=封禁）
	// 注：status=2(已注销)由用户注销时自动设置，管理员可通过设置1来恢复
	if req.Status != 0 && req.Status != 1 {
		response.FailMsg(c, response.CodeParamError, "非法的状态值")
		return
	}
	if err := h.DB.Model(&model.User{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "状态更新失败")
		return
	}
	WriteLog(c, h.DB, "用户", "状态更新", id, fmt.Sprintf("用户ID:%s 状态:%d", id, req.Status))
	response.OKMsg(c, "状态更新成功", nil)
}

// 常用乘客列表

func (h *WxUserHandler) PassengerList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.DB.Model(&model.Passenger{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+sanitize.EscapeLikePattern(name)+"%")
	}
	query.Count(&total)

	var list []model.Passenger
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	model.MaskPassengerList(list)
	response.Page(c, list, total, page, pageSize)
}
