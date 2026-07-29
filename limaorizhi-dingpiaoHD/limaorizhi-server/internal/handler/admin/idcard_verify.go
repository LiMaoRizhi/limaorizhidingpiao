// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"strconv"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/idcard"
	"limaorizhi-server/internal/pkg/redis"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// IDCardVerifyHandler 身份实名认证缓存管理（监控 + 主动失效）
type IDCardVerifyHandler struct{ DB *gorm.DB }

func NewIDCardVerifyHandler(db *gorm.DB) *IDCardVerifyHandler {
	return &IDCardVerifyHandler{DB: db}
}

// idCardPricePerCall 云市场二要素核验单价（元/次）
const idCardPricePerCall = 0.3

// Stats 返回实名认证缓存的统计快照
// 失败也计费（云市场按调用计费），所以用 APICalls 而非 (APICalls - APIErrors)
func (h *IDCardVerifyHandler) Stats(c *gin.Context) {
	stats := idcard.GetVerifyStats()
	// 估算累计节省成本：每次缓存命中省 1 次云市场 API 调用 = 0.3 元
	savedCostYuan := float64(stats.CacheHits) * idCardPricePerCall
	// 估算累计花费成本：实际 API 调用次数 × 0.3 元
	// 注意：失败也计费（云市场按调用计费），所以用 APICalls 而非 (APICalls - APIErrors)
	spentCostYuan := float64(stats.APICalls) * idCardPricePerCall
	response.OK(c, gin.H{
		"stats":             stats,
		"saved_cost_yuan":   savedCostYuan,
		"spent_cost_yuan":   spentCostYuan,
		"price_per_call":    idCardPricePerCall,
		"cost_unit":         "CNY",
		"note":              "成本为估算值，实际以云市场账单为准。统计数据进程级累计，重启清零。",
	})
}

// ResetStats 重置所有计数器（不影响 Redis 缓存本身）
func (h *IDCardVerifyHandler) ResetStats(c *gin.Context) {
	idcard.ResetVerifyStats()
	WriteLog(c, h.DB, "实名认证", "重置统计", "-", "重置实名认证缓存命中率计数器")
	response.OKMsg(c, "统计计数器已重置", nil)
}

// InvalidatePassengerCache 删除指定常用乘客的认证缓存，强制下次走云市场 API
func (h *IDCardVerifyHandler) InvalidatePassengerCache(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.FailMsg(c, response.CodeParamError, "无效的乘客ID")
		return
	}

	// 查询乘客信息（Passenger.AfterFind Hook 自动解密 IDCardNo 为明文）
	// 这里直接拿到的是明文身份证号，可以用于计算缓存 key（key 用 sha256，不会泄露明文）
	var p model.Passenger
	if err := h.DB.First(&p, id).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "乘客不存在")
		return
	}

	// 删除 Redis 缓存（fallback=true 表示 Redis 不可用，原本就没有缓存可删，下次本就走 API）
	redis.DeleteIDCardVerify(p.Name, p.IDCardNo)
	idcard.IncCacheDelete()

	// 日志中脱敏身份证号（与 MaskPassengerList 风格一致，避免审计日志泄露明文证件号）
	WriteLog(c, h.DB, "实名认证", "清除缓存", idStr,
		fmt.Sprintf("乘客ID:%d 姓名:%s 证件:%s", id, p.Name, idcard.MaskIDCard(p.IDCardNo)))

	response.OKMsg(c, "缓存已清除，下次该乘客认证将走云市场 API", gin.H{
		"passenger_id": id,
		"name":         p.Name,
		"id_card_mask": idcard.MaskIDCard(p.IDCardNo),
	})
}
