package admin

import (
	"fmt"
	"log"
	"strings"

	"limaorizhi-server/internal/pkg/crypto"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// aiConfigKeys 允许API更新的配置项白名单 其他的不让改
var aiConfigKeys = map[string]bool{
	"ai_employee_enabled": true,
	"ai_provider":         true, // nvidia | qwen | deepseek | kimi | glm | doubao | minimax
	"ai_base_url":         true,
	"ai_model":            true,
	"ai_api_key":          true, // 加密存储
	"ai_system_prompt":    true,
}

// GetConfig 返回AI配置 API Key只返回配没配 不返回密文
func (h *AIHandler) GetConfig(c *gin.Context) {
	configs := h.allAIConfigs()

	provider := configs["ai_provider"]
	if provider == "" {
		provider = "nvidia"
	}
	apiKeyConfigured := h.hasConfig(fmt.Sprintf("ai_api_key_%s", provider))
	if !apiKeyConfigured && provider == "nvidia" {
		apiKeyConfigured = h.hasConfig("ai_api_key")
	}

	response.OK(c, gin.H{
		"ai_employee_enabled":     configs["ai_employee_enabled"] == "true",
		"ai_provider":             configs["ai_provider"],
		"ai_base_url":             configs["ai_base_url"],
		"ai_model":                configs["ai_model"],
		"ai_system_prompt":        configs["ai_system_prompt"],
		"ai_api_key_configured":   apiKeyConfigured,
	})
}

// UpdateConfig 更新配置 超级管理员才能调 路由层已经校验过了
func (h *AIHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Configs map[string]string `json:"configs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 用事务包一下：多个配置要么全成要么全不成，别搞得一半生效一半没生效
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	tempHandler := &AIHandler{DB: tx}
	for key, value := range req.Configs {
		if !aiConfigKeys[key] && !strings.HasPrefix(key, "ai_api_key_") {
			response.FailMsg(c, response.CodeParamError, "不支持的配置项: "+key)
			return
		}

		// 切换服务商自动填base_url和model
		if key == "ai_provider" {
			defaultBaseURL, defaultModel := providerDefaults(value)
			if defaultBaseURL != "" {
				if _, ok := req.Configs["ai_base_url"]; !ok {
					tempHandler.configSet("ai_base_url", defaultBaseURL)
				}
				if _, ok := req.Configs["ai_model"]; !ok {
					tempHandler.configSet("ai_model", defaultModel)
				}
			}
		}

		// API Key加密存
		if key == "ai_api_key" || strings.HasPrefix(key, "ai_api_key_") {
			if value == "" {
				continue
			}
			encrypted, err := crypto.Encrypt(value)
			if err != nil {
				log.Printf("[ERROR] AI API Key 加密失败: %v", err)
				tx.Rollback()
				response.FailMsg(c, response.CodeServerError, "API Key 加密失败")
				return
			}
			value = encrypted
		}

		if err := tempHandler.configSet(key, value); err != nil {
			tx.Rollback()
			response.FailMsg(c, response.CodeServerError, "配置更新失败")
			return
		}
	}

	// 存了Key后自动选第一个有Key的服务商 兼容旧版单key配置
	currentProvider := tempHandler.configGet("ai_provider")
	if currentProvider == "" {
		currentProvider = "nvidia"
	}
	currentHasKey := tempHandler.hasConfig(fmt.Sprintf("ai_api_key_%s", currentProvider))
	if currentProvider == "nvidia" && !currentHasKey {
		currentHasKey = tempHandler.hasConfig("ai_api_key")
	}
	if !currentHasKey || tempHandler.configGet("ai_base_url") == "" || tempHandler.configGet("ai_model") == "" {
		for _, p := range aiProviders {
			pHasKey := tempHandler.hasConfig(fmt.Sprintf("ai_api_key_%s", p.Value))
			if !pHasKey && p.Value == "nvidia" {
				pHasKey = tempHandler.hasConfig("ai_api_key")
			}
			if pHasKey {
				if !currentHasKey {
					tempHandler.configSet("ai_provider", p.Value)
				}
				if tempHandler.configGet("ai_base_url") == "" {
					tempHandler.configSet("ai_base_url", p.BaseURL)
				}
				if tempHandler.configGet("ai_model") == "" && len(p.Models) > 0 {
					tempHandler.configSet("ai_model", p.Models[0].ID)
				}
				break
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[ERROR] AI配置事务提交失败: %v", err)
		response.FailMsg(c, response.CodeServerError, "配置保存失败，请重试")
		return
	}

	WriteLog(c, h.DB, "AI数字员工", "配置更新", "", "更新AI数字员工配置")
	response.OKMsg(c, "AI配置更新成功", nil)
}

// GetModels 返回服务商和模型列表 所有管理员都能调 切模型用
func (h *AIHandler) GetModels(c *gin.Context) {
	currentProvider := h.configGet("ai_provider")
	if currentProvider == "" {
		currentProvider = "nvidia"
	}
	currentModel := h.configGet("ai_model")

	// 动态填has_key
	providers := make([]ProviderInfo, len(aiProviders))
	copy(providers, aiProviders)
	for i := range providers {
		hasKey := h.hasConfig(fmt.Sprintf("ai_api_key_%s", providers[i].Value))
		if !hasKey && providers[i].Value == "nvidia" {
			hasKey = h.hasConfig("ai_api_key")
		}
		providers[i].HasKey = hasKey
	}

	response.OK(c, gin.H{
		"providers":        providers,
		"current_provider": currentProvider,
		"current_model":    currentModel,
	})
}

// SwitchModel 切模型 自动找模型属于哪个服务商 provider/base_url/model一起切
func (h *AIHandler) SwitchModel(c *gin.Context) {
	var req struct {
		Model string `json:"model" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	var found *ProviderInfo
	for i := range aiProviders {
		for _, m := range aiProviders[i].Models {
			if m.ID == req.Model {
				// 图片生成模型走独立接口 不能聊天
				if m.Icon == "image" {
					response.FailMsg(c, response.CodeParamError, m.Name+"是图片生成模型，不支持对话功能，请选择对话模型")
					return
				}
				found = &aiProviders[i]
				break
			}
		}
		if found != nil {
			break
		}
	}
	if found == nil {
		response.FailMsg(c, response.CodeParamError, "不支持的模型: "+req.Model)
		return
	}

	// 切之前先看有没有Key 没Key切过去调用才报错 体验太差
	hasKey := h.hasConfig(fmt.Sprintf("ai_api_key_%s", found.Value))
	if !hasKey && found.Value == "nvidia" {
		hasKey = h.hasConfig("ai_api_key")
	}
	if !hasKey {
		response.FailMsg(c, response.CodeParamError, found.Name+" 未配置 API Key，请先在系统配置中设置")
		return
	}

	if err := h.configSet("ai_provider", found.Value); err != nil {
		log.Printf("[ERROR] 切换AI服务商失败: %v", err)
		response.FailMsg(c, response.CodeServerError, "切换服务商失败，请稍后重试")
		return
	}
	if err := h.configSet("ai_base_url", found.BaseURL); err != nil {
		log.Printf("[ERROR] 切换AI Base URL失败: %v", err)
		response.FailMsg(c, response.CodeServerError, "切换Base URL失败，请稍后重试")
		return
	}
	if err := h.configSet("ai_model", req.Model); err != nil {
		log.Printf("[ERROR] 切换AI模型失败: %v", err)
		response.FailMsg(c, response.CodeServerError, "模型切换失败，请稍后重试")
		return
	}

	adminName, _ := c.Get("admin_name")
	WriteLog(c, h.DB, "AI数字员工", "切换模型", "", fmt.Sprintf("管理员 %v 切换AI模型为 %s (%s)", adminName, req.Model, found.Name))
	response.OKMsg(c, "模型切换成功", gin.H{"model": req.Model, "provider": found.Value, "has_key": true})
}
