package admin

import (
	"fmt"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)
// 轮播图管理

type BannerHandler struct {
	DB *gorm.DB
}

func NewBannerHandler(db *gorm.DB) *BannerHandler {
	return &BannerHandler{DB: db}
}

func (h *BannerHandler) List(c *gin.Context) {
	var list []model.Banner
	h.DB.Order("sort_order ASC, id ASC").Find(&list)
	response.OK(c, list)
}

// PublicList 公开接口：只返回显示中的轮播图（供小程序端调用）
func (h *BannerHandler) PublicList(c *gin.Context) {
	var list []model.Banner
	h.DB.Where("status = 1").Order("sort_order ASC, id ASC").Find(&list)
	response.OK(c, list)
}

// createBannerRequest 轮播图创建请求（DTO白名单）
type createBannerRequest struct {
	Title       string `json:"title"`
	TitleColor  string `json:"title_color"`
	TitleEffect int8   `json:"title_effect"`
	ImageURL    string `json:"image_url" binding:"required"`
	LinkType    int8   `json:"link_type"`
	LinkValue   string `json:"link_value"`
	SortOrder   int    `json:"sort_order"`
	Status      int8   `json:"status"`
}

func (h *BannerHandler) Create(c *gin.Context) {
	var req createBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	b := model.Banner{
		Title:       req.Title,
		TitleColor:  req.TitleColor,
		TitleEffect: req.TitleEffect,
		ImageURL:    req.ImageURL,
		LinkType:    req.LinkType,
		LinkValue:   req.LinkValue,
		SortOrder:   req.SortOrder,
		Status:      1, // 默认显示
	}
	if req.Status == 0 || req.Status == 1 {
		b.Status = req.Status
	}
	if err := h.DB.Create(&b).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	WriteLog(c, h.DB, "轮播图", "新增", b.Title, fmt.Sprintf("轮播图ID:%d 标题:%s", b.ID, b.Title))
	response.OK(c, b)
}

func (h *BannerHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var b model.Banner
	if err := h.DB.First(&b, id).Error; err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	var req struct {
		Title       string `json:"title"`
		TitleColor  string `json:"title_color"`
		TitleEffect int8   `json:"title_effect"`
		ImageURL    string `json:"image_url"`
		LinkType    int8   `json:"link_type"`
		LinkValue   string `json:"link_value"`
		SortOrder   int    `json:"sort_order"`
		Status      int8   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if err := h.DB.Model(&b).Updates(map[string]interface{}{
		"title": req.Title, "title_color": req.TitleColor, "title_effect": req.TitleEffect, "image_url": req.ImageURL, "link_type": req.LinkType,
		"link_value": req.LinkValue, "sort_order": req.SortOrder, "status": req.Status,
	}).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	h.DB.First(&b, id)
	WriteLog(c, h.DB, "轮播图", "编辑", b.Title, fmt.Sprintf("轮播图ID:%s 标题:%s", id, b.Title))
	response.OK(c, b)
}

func (h *BannerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.DB.Delete(&model.Banner{}, id).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}
	WriteLog(c, h.DB, "轮播图", "删除", id, "删除轮播图ID:"+id)
	response.OKMsg(c, "删除成功", nil)
}

