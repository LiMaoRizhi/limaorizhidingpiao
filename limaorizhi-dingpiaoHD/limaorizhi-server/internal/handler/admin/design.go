// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"encoding/json"
	"fmt"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 小程序首页装修

// LayoutComponent 首页装修组件
type LayoutComponent struct {
	Type    string `json:"type"`    // banner=轮播图 coupon=优惠券 notice=公告 search=搜索筛选 trips=车次列表
	Title   string `json:"title"`  // 组件显示名称
	Visible bool   `json:"visible"` // 是否显示
}

// DesignHandler 装修管理
type DesignHandler struct{ DB *gorm.DB }

func NewDesignHandler(db *gorm.DB) *DesignHandler { return &DesignHandler{DB: db} }

// defaultLayout 默认首页布局
func defaultLayout() []LayoutComponent {
	return []LayoutComponent{
		{Type: "banner", Title: "轮播图", Visible: true},
		{Type: "coupon", Title: "优惠券展示", Visible: true},
		{Type: "notice", Title: "公告通知", Visible: true},
		{Type: "search", Title: "搜索筛选", Visible: true},
		{Type: "trips", Title: "车次列表", Visible: true},
	}
}

// validComponentTypes 合法的组件类型
var validComponentTypes = map[string]bool{
	"banner": true,
	"coupon": true,
	"notice": true,
	"search": true,
	"trips":  true,
}

// GetLayout 管理端获取首页布局配置
func (h *DesignHandler) GetLayout(c *gin.Context) {
	var cfg model.SystemConfig
	if err := h.DB.Where("config_key = ?", "homepage_layout").First(&cfg).Error; err != nil {
		response.OK(c, defaultLayout())
		return
	}
	var layout []LayoutComponent
	if err := json.Unmarshal([]byte(cfg.ConfigValue), &layout); err != nil {
		response.OK(c, defaultLayout())
		return
	}
	response.OK(c, layout)
}

// UpdateLayout 管理端更新首页布局配置（仅超级管理员）
func (h *DesignHandler) UpdateLayout(c *gin.Context) {
	roleVal, _ := c.Get("admin_role")
	role, _ := roleVal.(int8)
	if role != 1 {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以修改装修布局")
		return
	}
	var layout []LayoutComponent
	if err := c.ShouldBindJSON(&layout); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	for _, comp := range layout {
		if !validComponentTypes[comp.Type] {
			response.FailMsg(c, response.CodeParamError, fmt.Sprintf("无效的组件类型: %s", comp.Type))
			return
		}
		if comp.Title == "" {
			comp.Title = comp.Type
		}
	}
	jsonBytes, err := json.Marshal(layout)
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}
	var cfg model.SystemConfig
	result := h.DB.Where("config_key = ?", "homepage_layout").First(&cfg)
	if result.Error != nil {
		h.DB.Create(&model.SystemConfig{
			ConfigKey:   "homepage_layout",
			ConfigValue: string(jsonBytes),
			Description: "首页装修布局配置",
		})
	} else {
		h.DB.Model(&cfg).Update("config_value", string(jsonBytes))
	}
	WriteLog(c, h.DB, "装修", "更新", "", "更新首页装修布局")
	response.OKMsg(c, "保存成功", nil)
}

// PublicLayout 公开接口：返回首页布局（仅 visible=true 的组件，按排序顺序）
func (h *DesignHandler) PublicLayout(c *gin.Context) {
	var cfg model.SystemConfig
	if err := h.DB.Where("config_key = ?", "homepage_layout").First(&cfg).Error; err != nil {
		response.OK(c, defaultLayout())
		return
	}
	var layout []LayoutComponent
	if err := json.Unmarshal([]byte(cfg.ConfigValue), &layout); err != nil {
		response.OK(c, defaultLayout())
		return
	}
	var visible []LayoutComponent
	for _, comp := range layout {
		if comp.Visible {
			visible = append(visible, comp)
		}
	}
	if visible == nil {
		visible = []LayoutComponent{}
	}
	response.OK(c, visible)
}

// 多页面装修（订单页/我的页）

// pageLayoutConfig 单个页面的装修配置
type pageLayoutConfig struct {
	ConfigKey  string            // SystemConfig 存储键
	Desc       string            // 描述
	Defaults   []LayoutComponent // 默认布局
	ValidTypes map[string]bool   // 合法组件类型
}

// pageLayoutConfigs 各页面装修配置
// page 参数: order_tabs=订单页标签 mine_order_grid=我的页订单分类 mine_menu=我的页功能菜单
var pageLayoutConfigs = map[string]pageLayoutConfig{
	"order_tabs": {
		ConfigKey: "page_layout_order_tabs",
		Desc:      "订单页标签装修布局",
		Defaults: []LayoutComponent{
			{Type: "all", Title: "全部", Visible: true},
			{Type: "0", Title: "待支付", Visible: true},
			{Type: "1", Title: "待出行", Visible: true},
			{Type: "2", Title: "已完成", Visible: true},
			{Type: "3", Title: "已退款", Visible: true},
			{Type: "4", Title: "已取消", Visible: true},
		},
		ValidTypes: map[string]bool{"all": true, "0": true, "1": true, "2": true, "3": true, "4": true},
	},
	"mine_order_grid": {
		ConfigKey: "page_layout_mine_order_grid",
		Desc:      "我的页订单分类装修布局",
		Defaults: []LayoutComponent{
			{Type: "pending_pay", Title: "待付款", Visible: true},
			{Type: "pending_travel", Title: "待出行", Visible: true},
			{Type: "completed", Title: "已完成", Visible: true},
			{Type: "refund", Title: "退款", Visible: true},
		},
		ValidTypes: map[string]bool{"pending_pay": true, "pending_travel": true, "completed": true, "refund": true},
	},
	"mine_menu": {
		ConfigKey: "page_layout_mine_menu",
		Desc:      "我的页功能菜单装修布局",
		Defaults: []LayoutComponent{
			{Type: "passenger", Title: "常用乘客", Visible: true},
			{Type: "gift", Title: "惊喜礼包", Visible: true},
			{Type: "orders", Title: "行程管理", Visible: true},
			{Type: "cargo", Title: "货物托运", Visible: true},
			{Type: "wechat_service", Title: "微信客服", Visible: true},
			{Type: "phone", Title: "电话热线", Visible: true},
			{Type: "verify", Title: "司机核销", Visible: true},
		},
		ValidTypes: map[string]bool{"passenger": true, "gift": true, "orders": true, "cargo": true, "wechat_service": true, "phone": true, "verify": true},
	},
}

// GetPageLayout 管理端获取指定页面布局配置
func (h *DesignHandler) GetPageLayout(c *gin.Context) {
	page := c.Query("page")
	cfg, ok := pageLayoutConfigs[page]
	if !ok {
		response.FailMsg(c, response.CodeParamError, "无效的页面: "+page)
		return
	}
	var sc model.SystemConfig
	if err := h.DB.Where("config_key = ?", cfg.ConfigKey).First(&sc).Error; err != nil {
		response.OK(c, cfg.Defaults)
		return
	}
	var layout []LayoutComponent
	if err := json.Unmarshal([]byte(sc.ConfigValue), &layout); err != nil {
		response.OK(c, cfg.Defaults)
		return
	}
	response.OK(c, layout)
}

// UpdatePageLayout 管理端更新指定页面布局配置（仅超级管理员）
func (h *DesignHandler) UpdatePageLayout(c *gin.Context) {
	roleVal, _ := c.Get("admin_role")
	role, _ := roleVal.(int8)
	if role != 1 {
		response.FailMsg(c, response.CodeForbidden, "只有超级管理员可以修改装修布局")
		return
	}
	page := c.Query("page")
	cfg, ok := pageLayoutConfigs[page]
	if !ok {
		response.FailMsg(c, response.CodeParamError, "无效的页面: "+page)
		return
	}
	var layout []LayoutComponent
	if err := c.ShouldBindJSON(&layout); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	for _, comp := range layout {
		if !cfg.ValidTypes[comp.Type] {
			response.FailMsg(c, response.CodeParamError, fmt.Sprintf("无效的组件类型: %s", comp.Type))
			return
		}
		if comp.Title == "" {
			comp.Title = comp.Type
		}
	}
	jsonBytes, err := json.Marshal(layout)
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}
	var sc model.SystemConfig
	result := h.DB.Where("config_key = ?", cfg.ConfigKey).First(&sc)
	if result.Error != nil {
		h.DB.Create(&model.SystemConfig{
			ConfigKey:   cfg.ConfigKey,
			ConfigValue: string(jsonBytes),
			Description: cfg.Desc,
		})
	} else {
		h.DB.Model(&sc).Update("config_value", string(jsonBytes))
	}
	WriteLog(c, h.DB, "装修", "更新", "", "更新"+cfg.Desc)
	response.OKMsg(c, "保存成功", nil)
}

// PublicPageLayout 公开接口：返回指定页面布局（仅 visible=true，按排序顺序）
func (h *DesignHandler) PublicPageLayout(c *gin.Context) {
	page := c.Query("page")
	cfg, ok := pageLayoutConfigs[page]
	if !ok {
		response.FailMsg(c, response.CodeParamError, "无效的页面: "+page)
		return
	}
	var sc model.SystemConfig
	if err := h.DB.Where("config_key = ?", cfg.ConfigKey).First(&sc).Error; err != nil {
		// 未配置时返回默认布局（默认全 visible）
		response.OK(c, cfg.Defaults)
		return
	}
	var layout []LayoutComponent
	if err := json.Unmarshal([]byte(sc.ConfigValue), &layout); err != nil {
		response.OK(c, cfg.Defaults)
		return
	}
	var visible []LayoutComponent
	for _, comp := range layout {
		if comp.Visible {
			visible = append(visible, comp)
		}
	}
	if visible == nil {
		visible = []LayoutComponent{}
	}
	response.OK(c, visible)
}
