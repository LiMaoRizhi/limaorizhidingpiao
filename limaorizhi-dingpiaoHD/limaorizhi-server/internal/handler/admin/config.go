// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/sanitize"
	"limaorizhi-server/internal/pkg/upload"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 系统配置

type ConfigHandler struct{ DB *gorm.DB }

func NewConfigHandler(db *gorm.DB) *ConfigHandler { return &ConfigHandler{DB: db} }

func (h *ConfigHandler) Get(c *gin.Context) {
	// 系统配置读取需要超级管理员权限，防止普通管理员获取敏感配置项
	roleVal, exists := c.Get("admin_role")
	if !exists {
		response.FailMsg(c, response.CodeForbidden, "权限信息缺失")
		return
	}
	role, ok := roleVal.(int8)
	if !ok || role != 1 {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以查看系统配置")
		return
	}
	var configs []model.SystemConfig
	h.DB.Find(&configs)
	result := make(map[string]string)
	for _, cfg := range configs {
		// 安全脱敏：所有 AI API Key（含旧版统一key和各服务商独立key）不返回密文
		// 防止通过通用配置接口泄露 ai_api_key / ai_api_key_nvidia / ai_api_key_qwen 等
		if cfg.ConfigKey == "ai_api_key" || strings.HasPrefix(cfg.ConfigKey, "ai_api_key_") {
			if cfg.ConfigValue != "" {
				result[cfg.ConfigKey] = "***"
			} else {
				result[cfg.ConfigKey] = ""
			}
		} else {
			result[cfg.ConfigKey] = cfg.ConfigValue
		}
	}
	response.OK(c, result)
}

type updateConfigRequest struct {
	Configs map[string]string `json:"configs"`
}

func (h *ConfigHandler) Update(c *gin.Context) {
	// 系统配置更新需要超级管理员权限
	roleVal, exists := c.Get("admin_role")
	if !exists {
		response.FailMsg(c, response.CodeForbidden, "权限信息缺失")
		return
	}
	role, ok := roleVal.(int8)
	if !ok || role != 1 {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以修改系统配置")
		return
	}
	var req updateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 配置项白名单校验，防止注入非法配置
	allowedConfigKeys := map[string]bool{
		"site_name": true, "customer_service_phone": true, "after_sales_wechat": true,
		"order_expire_minutes": true, "refund_before_departure_hours": true, "refund_fee_rate": true,
		"notice": true,
		"cargo_price_per_km": true, "cargo_min_fee": true, "cargo_free_weight": true,
		"cargo_extra_weight_fee": true, "cargo_max_weight": true,
		"mine_menu_layout_type": true,
		"logout_position": true,
		"homepage_coupon_ids": true,
		// 品牌设置
		"brand_name": true, "brand_logo": true,
		// 协议政策（用户协议 + 隐私政策）
		"user_agreement": true, "privacy_policy": true,
		// AI 数字员工配置项（ai_api_key 在 ai.go 中单独加密处理，此处仅做白名单兼容）
		"ai_employee_enabled": true, "ai_provider": true, "ai_base_url": true,
		"ai_model": true, "ai_system_prompt": true,
	}
	for key, value := range req.Configs {
		if !allowedConfigKeys[key] {
			response.FailMsg(c, response.CodeParamError, "不支持的配置项: "+key)
			return
		}
		// 配置值校验：对数值型配置项做范围检查，防止非法值导致业务异常
		if key == "refund_fee_rate" {
			feeVal, err := strconv.ParseFloat(value, 64)
			if err != nil || feeVal < 0 || feeVal > 100 {
				response.FailMsg(c, response.CodeParamError, "退票手续费率必须为0-100之间的数字")
				return
			}
		}
		if key == "refund_before_departure_hours" {
			hoursVal, err := strconv.Atoi(value)
			if err != nil || hoursVal < 0 || hoursVal > 720 {
				response.FailMsg(c, response.CodeParamError, "退票时限必须为0-720之间的整数")
				return
			}
		}
		if key == "order_expire_minutes" {
			minutesVal, err := strconv.Atoi(value)
			if err != nil || minutesVal < 1 || minutesVal > 1440 {
				response.FailMsg(c, response.CodeParamError, "订单过期时间必须为1-1440之间的整数")
				return
			}
		}
		// upsert: 存在则更新，不存在则创建
		var cfg model.SystemConfig
		result := h.DB.Where("config_key = ?", key).First(&cfg)
		if result.Error != nil {
			// 不存在，创建
			if err := h.DB.Create(&model.SystemConfig{ConfigKey: key, ConfigValue: value}).Error; err != nil {
				response.FailMsg(c, response.CodeServerError, "配置更新失败")
				return
			}
		} else {
			// 存在，更新
			if err := h.DB.Model(&cfg).Update("config_value", value).Error; err != nil {
				response.FailMsg(c, response.CodeServerError, "配置更新失败")
				return
			}
		}
	}
	WriteLog(c, h.DB, "系统配置", "更新", "", "更新系统配置")
	response.OKMsg(c, "配置更新成功", nil)
}

// 管理员管理

type AdminUserHandler struct{ DB *gorm.DB }

func NewAdminUserHandler(db *gorm.DB) *AdminUserHandler { return &AdminUserHandler{DB: db} }

func (h *AdminUserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	h.DB.Model(&model.AdminUser{}).Count(&total)

	var list []model.AdminUser
	h.DB.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

type createAdminRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	RealName string `json:"real_name"`
	Phone    string `json:"phone"`
	Role     int8   `json:"role"`
}

func (h *AdminUserHandler) Create(c *gin.Context) {
	var req createAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	var count int64
	h.DB.Model(&model.AdminUser{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		response.FailMsg(c, response.CodeServerError, "用户名已存在")
		return
	}
	// 弱口令校验
	if err := validatePasswordStrength(req.Password); err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "密码加密失败")
		return
	}
	admin := model.AdminUser{
		Username:     req.Username,
		PasswordHash: string(hash),
		RealName:     req.RealName,
		Phone:        req.Phone,
		Status:       1,
	}
	// 校验角色值
	if req.Role != 1 && req.Role != 2 {
		req.Role = 2 // 默认普通管理员
	}
	admin.Role = req.Role
	if err := h.DB.Create(&admin).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	WriteLog(c, h.DB, "管理员", "新增", admin.Username, fmt.Sprintf("管理员ID:%d 用户名:%s", admin.ID, admin.Username))
	response.OK(c, admin)
}

type updateAdminRequest struct {
	RealName string `json:"real_name"`
	Phone    string `json:"phone"`
	Role     int8   `json:"role"`
	Status   int8   `json:"status"`
}

func (h *AdminUserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req updateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	currentRoleVal, _ := c.Get("admin_role")
	currentRole, _ := currentRoleVal.(int8)
	isSuperAdmin := currentRole == 1

	// 查询目标管理员信息
	var target model.AdminUser
	if err := h.DB.First(&target, id).Error; err != nil {
		response.Fail(c, response.CodeUserNotFound)
		return
	}

	// 普通管理员不能修改超级管理员
	if !isSuperAdmin && target.Role == 1 {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以修改超级管理员")
		return
	}
	// 不允许将普通管理员提升为超级管理员
	if req.Role == 1 && !isSuperAdmin {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以设置管理员角色")
		return
	}
	// 普通管理员不能修改管理员角色
	if !isSuperAdmin && req.Role != target.Role {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以修改管理员角色")
		return
	}
	// 校验status值合法性（仅允许0=禁用 或 1=启用）
	if req.Status != 0 && req.Status != 1 {
		response.FailMsg(c, response.CodeParamError, "状态值只能为0(禁用)或1(启用)")
		return
	}
	updates := map[string]interface{}{
		"real_name": req.RealName,
		"phone":     req.Phone,
		"role":      req.Role,
		"status":    req.Status,
	}
	if err := h.DB.Model(&model.AdminUser{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	WriteLog(c, h.DB, "管理员", "编辑", id, fmt.Sprintf("管理员ID:%s 用户名:%s", id, req.RealName))
	response.OKMsg(c, "更新成功", nil)
}

func (h *AdminUserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	// 超级管理员才能删除其他管理员
	currentRoleVal, _ := c.Get("admin_role")
	currentRole, _ := currentRoleVal.(int8)
	if currentRole != 1 {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以删除管理员")
		return
	}
	// 不能删除自己
	currentID := c.GetUint("admin_id")
	idUint, _ := strconv.ParseUint(id, 10, 64)
	if uint(idUint) == currentID {
		response.FailMsg(c, response.CodeForbidden, "不能删除自己的账号")
		return
	}
	if err := h.DB.Delete(&model.AdminUser{}, id).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}
	WriteLog(c, h.DB, "管理员", "删除", id, "删除管理员ID:"+id)
	response.OKMsg(c, "删除成功", nil)
}

// 重置密码

type resetPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// ResetPassword 超级管理员重置其他管理员密码
func (h *AdminUserHandler) ResetPassword(c *gin.Context) {
	roleVal, _ := c.Get("admin_role")
	role, _ := roleVal.(int8)
	if role != 1 {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以重置密码")
		return
	}
	id := c.Param("id")
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 弱口令校验
	if err := validatePasswordStrength(req.Password); err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}
	var admin model.AdminUser
	if err := h.DB.First(&admin, id).Error; err != nil {
		response.Fail(c, response.CodeUserNotFound)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "密码加密失败")
		return
	}
	if err := h.DB.Model(&admin).Update("password_hash", string(hash)).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "密码重置失败")
		return
	}
	// 使旧Token失效：被重置密码的管理员的旧Token立即失效
	now := time.Now()
	if err := h.DB.Model(&admin).Update("token_invalid_before", &now).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "Token失效标记写入失败")
		return
	}
	WriteLog(c, h.DB, "管理员", "重置密码", id, fmt.Sprintf("重置管理员ID:%s 用户名:%s 的密码", id, admin.Username))
	response.OKMsg(c, "密码重置成功", nil)
}

// 操作日志

func (h *AdminUserHandler) Logs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.DB.Model(&model.OperationLog{})
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		escaped := sanitize.EscapeLikePattern(keyword)
		query = query.Where("admin_name LIKE ? OR detail LIKE ?", "%"+escaped+"%", "%"+escaped+"%")
	}
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("CAST(created_at AS DATE) >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("CAST(created_at AS DATE) <= ?", endDate)
	}
	query.Count(&total)

	var list []model.OperationLog
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

// LogsExport 导出操作日志为CSV（最多导出10000条，带筛选条件）
func (h *AdminUserHandler) LogsExport(c *gin.Context) {
	query := h.DB.Model(&model.OperationLog{})
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		escaped := sanitize.EscapeLikePattern(keyword)
		query = query.Where("admin_name LIKE ? OR detail LIKE ?", "%"+escaped+"%", "%"+escaped+"%")
	}
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("CAST(created_at AS DATE) >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("CAST(created_at AS DATE) <= ?", endDate)
	}

	var list []model.OperationLog
	query.Order("created_at DESC").Limit(10000).Find(&list)

	// 生成CSV（带BOM头防止Excel乱码）
	buf := &bytes.Buffer{}
	buf.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(buf)
	w.UseCRLF = true
	w.Write([]string{"操作人", "模块", "操作类型", "操作内容", "IP地址", "操作时间"})

	for _, log := range list {
		w.Write([]string{
			log.AdminName,
			log.Module,
			log.Action,
			log.Detail,
			log.IPAddress,
			"\t" + log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()

	c.Header("Content-Disposition", "attachment; filename=operation_logs_"+time.Now().Format("20060102_150405")+".csv")
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
}

// 退款记录

func (h *ConfigHandler) RefundList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	orderNo := c.Query("order_no")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var total int64
	query := h.DB.Model(&model.Refund{})
	if orderNo != "" {
		query = query.Where("refund_no LIKE ? OR order_id IN (SELECT id FROM orders WHERE order_no LIKE ?)",
			"%"+sanitize.EscapeLikePattern(orderNo)+"%", "%"+sanitize.EscapeLikePattern(orderNo)+"%")
	}
	if startDate != "" {
		query = query.Where("CAST(created_at AS DATE) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("CAST(created_at AS DATE) <= ?", endDate)
	}
	query.Count(&total)

	var list []model.Refund
	query.Preload("Order").Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	// 脱敏订单中的手机号
	for i := range list {
		if list[i].Order != nil {
			list[i].Order.Mask()
		}
	}
	response.Page(c, list, total, page, pageSize)
}

// 文件上传

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler { return &UploadHandler{} }

func (h *UploadHandler) Upload(c *gin.Context) {
	url, filename, ok := upload.SaveImageFile(c)
	if !ok {
		return // SaveImageFile 内部已返回错误响应
	}
	response.OK(c, gin.H{"url": url, "filename": filename})
}
