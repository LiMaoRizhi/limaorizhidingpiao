package admin

import (
	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// AIHandler AI数字员工请求处理
type AIHandler struct {
	DB *gorm.DB
}

// NewAIHandler 搞个handler出来
func NewAIHandler(db *gorm.DB) *AIHandler {
	return &AIHandler{DB: db}
}

// allAIConfigs 捞所有ai_开头的配置项 API Key密文别拿出来搁内存里
func (h *AIHandler) allAIConfigs() map[string]string {
	var configs []model.SystemConfig
	h.DB.Where("config_key LIKE ?", "ai_%").
		Where("config_key NOT LIKE ?", "ai_api_key%").
		Find(&configs)
	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.ConfigKey] = cfg.ConfigValue
	}
	return result
}

// configGet 读单个配置值 没有就空字符串
// 用Limit(1).Find不用First 记录不存在GORM一直打not found日志烦死人
func (h *AIHandler) configGet(key string) string {
	var cfg model.SystemConfig
	h.DB.Where("config_key = ?", key).Limit(1).Find(&cfg)
	return cfg.ConfigValue
}

// hasConfig 看配置项存不存在 不读值 API Key那种就判断个有没有
func (h *AIHandler) hasConfig(key string) bool {
	var count int64
	h.DB.Model(&model.SystemConfig{}).Where("config_key = ?", key).Count(&count)
	return count > 0
}

// configSet 写配置 有就更新没有就插
// 并发时俩请求可能都判断没记录然后都Create 靠config_key唯一索引兜底
// Create失败了退回Update 最终一致就行 管不了那么多
func (h *AIHandler) configSet(key, value string) error {
	var cfg model.SystemConfig
	result := h.DB.Where("config_key = ?", key).Limit(1).Find(&cfg)
	// 查询出错了别误判成记录不存在去走Create分支
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if err := h.DB.Create(&model.SystemConfig{ConfigKey: key, ConfigValue: value}).Error; err != nil {
			return h.DB.Model(&model.SystemConfig{}).Where("config_key = ?", key).
				Update("config_value", value).Error
		}
		return nil
	}
	return h.DB.Model(&cfg).Update("config_value", value).Error
}
