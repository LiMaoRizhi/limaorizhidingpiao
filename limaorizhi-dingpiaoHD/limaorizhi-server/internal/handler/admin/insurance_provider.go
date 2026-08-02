// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"strconv"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 保险公司配置管理（通用保险对接框架）
// 仅超级管理员可操作。同一时刻仅允许一家 is_active=true。
// AppSecret 在响应中脱敏（仅返回 app_secret_masked），明文只在创建/更新时接收。

type InsuranceProviderHandler struct{ DB *gorm.DB }

func NewInsuranceProviderHandler(db *gorm.DB) *InsuranceProviderHandler {
	return &InsuranceProviderHandler{DB: db}
}

// maskSecret 将密钥脱敏为 ab****yz 形式（少于8位时全部用 *）
func maskSecret(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// fillMasked 给返回列表中的 AppSecret 填充脱敏值
func fillMasked(list []model.InsuranceProvider) {
	for i := range list {
		list[i].AppSecretMasked = maskSecret(list[i].AppSecret)
	}
}

// List 分页查询保险公司列表
func (h *InsuranceProviderHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := h.DB.Model(&model.InsuranceProvider{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	query.Count(&total)

	var list []model.InsuranceProvider
	query.Order("is_active DESC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	fillMasked(list)
	response.Page(c, list, total, page, pageSize)
}

// insuranceProviderCreateRequest 创建请求体
type insuranceProviderCreateRequest struct {
	Name        string  `json:"name" binding:"required"`
	APIURL      string  `json:"api_url" binding:"required"`
	AppID       string  `json:"app_id" binding:"required"`
	AppSecret   string  `json:"app_secret"`
	ProductCode string  `json:"product_code"`
	Fee         float64 `json:"fee"`
	Required    bool    `json:"required"`
	Remark      string  `json:"remark"`
	IsActive    bool    `json:"is_active"`
}

// validateInsuranceProvider 校验创建/更新请求参数
func validateInsuranceProvider(req *insuranceProviderCreateRequest, requireSecret bool) error {
	if req.Name == "" {
		return fmt.Errorf("保险公司名称不能为空")
	}
	if req.APIURL == "" {
		return fmt.Errorf("出单API地址不能为空")
	}
	if req.AppID == "" {
		return fmt.Errorf("商户号/AppID不能为空")
	}
	if requireSecret && req.AppSecret == "" {
		return fmt.Errorf("商户密钥不能为空")
	}
	if req.Fee < 0 || req.Fee > 1000 {
		return fmt.Errorf("保险费必须为0-1000之间的数字")
	}
	return nil
}

// Create 新增保险公司配置
func (h *InsuranceProviderHandler) Create(c *gin.Context) {
	var req insuranceProviderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if err := validateInsuranceProvider(&req, true); err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}

	provider := model.InsuranceProvider{
		Name:        req.Name,
		APIURL:      req.APIURL,
		AppID:       req.AppID,
		AppSecret:   req.AppSecret,
		ProductCode: req.ProductCode,
		Fee:         req.Fee,
		Required:    req.Required,
		Remark:      req.Remark,
		IsActive:    req.IsActive,
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 若新增为启用，先将其他全部置为未启用
		if req.IsActive {
			if err := tx.Model(&model.InsuranceProvider{}).
				Where("is_active = ?", true).
				Update("is_active", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&provider).Error
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	provider.AppSecretMasked = maskSecret(provider.AppSecret)
	WriteLog(c, h.DB, "保险配置", "新增", provider.Name,
		fmt.Sprintf("保险公司ID:%d 名称:%s AppID:%s", provider.ID, provider.Name, provider.AppID))
	response.OK(c, provider)
}

// Update 编辑保险公司配置
func (h *InsuranceProviderHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var provider model.InsuranceProvider
	if err := h.DB.First(&provider, id).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "保险公司配置不存在")
		return
	}
	var req insuranceProviderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if err := validateInsuranceProvider(&req, false); err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}

	updates := map[string]interface{}{
		"name":         req.Name,
		"api_url":      req.APIURL,
		"app_id":       req.AppID,
		"product_code": req.ProductCode,
		"fee":          req.Fee,
		"required":     req.Required,
		"remark":       req.Remark,
		"is_active":    req.IsActive,
	}
	// app_secret 为空时不更新（保留原值）
	if req.AppSecret != "" {
		updates["app_secret"] = req.AppSecret
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 若设为启用，先将其他全部置为未启用（排除当前ID）
		if req.IsActive {
			if err := tx.Model(&model.InsuranceProvider{}).
				Where("is_active = ? AND id != ?", true, provider.ID).
				Update("is_active", false).Error; err != nil {
				return err
			}
		}
		return tx.Model(&provider).Where("id = ?", provider.ID).Updates(updates).Error
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}

	// 重新加载（含脱敏）
	h.DB.First(&provider, provider.ID)
	provider.AppSecretMasked = maskSecret(provider.AppSecret)
	WriteLog(c, h.DB, "保险配置", "编辑", provider.Name,
		fmt.Sprintf("保险公司ID:%s 名称:%s", id, provider.Name))
	response.OK(c, provider)
}

// Delete 删除保险公司配置（允许删已启用的，删后无启用家）
func (h *InsuranceProviderHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var provider model.InsuranceProvider
	if err := h.DB.First(&provider, id).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "保险公司配置不存在")
		return
	}
	if err := h.DB.Delete(&model.InsuranceProvider{}, id).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}
	WriteLog(c, h.DB, "保险配置", "删除", provider.Name,
		fmt.Sprintf("删除保险公司ID:%s 名称:%s", id, provider.Name))
	response.OKMsg(c, "删除成功", nil)
}

// Activate 切换某家保险公司为启用状态（其余自动置为未启用）
func (h *InsuranceProviderHandler) Activate(c *gin.Context) {
	id := c.Param("id")
	var provider model.InsuranceProvider
	if err := h.DB.First(&provider, id).Error; err != nil {
		response.FailMsg(c, response.CodeParamError, "保险公司配置不存在")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 先将其他全部置为未启用
		if err := tx.Model(&model.InsuranceProvider{}).
			Where("is_active = ? AND id != ?", true, provider.ID).
			Update("is_active", false).Error; err != nil {
			return err
		}
		// 将目标设为启用
		return tx.Model(&provider).Where("id = ?", provider.ID).
			Update("is_active", true).Error
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "启用失败")
		return
	}
	WriteLog(c, h.DB, "保险配置", "启用", provider.Name,
		fmt.Sprintf("启用保险公司ID:%s 名称:%s", id, provider.Name))
	response.OKMsg(c, "启用成功", nil)
}
