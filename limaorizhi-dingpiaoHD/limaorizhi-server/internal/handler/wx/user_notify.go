// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
)

// 订阅消息相关接口（用户授权上报 + 模板列表查询）

// subscribeReportRequest 用户上报订阅授权结果
// 前端 wx.requestSubscribeMessage 成功后调用，将用户授权的模板列表上报给后端
type subscribeReportRequest struct {
	TemplateKeys []string `json:"template_keys" binding:"required"` // 用户授权的模板键列表（如 ["payment_success","trip_departure"]）
}

// SubscribeReport 用户上报订阅授权结果
// 前端 wx.requestSubscribeMessage 返回后，将用户同意的模板上报，后端为每个模板增加1次发送配额
func (h *UserHandler) SubscribeReport(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req subscribeReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 增加配额（service层会过滤未配置的模板）
	service.AddSubscribeQuota(h.DB, userID, req.TemplateKeys)

	response.OKMsg(c, "订阅成功", nil)
}

// SubscribeTemplates 获取已配置的订阅消息模板列表
// 前端调用此接口获取需要请求订阅的模板ID列表，用于 wx.requestSubscribeMessage 的 tmplIds 参数
func (h *UserHandler) SubscribeTemplates(c *gin.Context) {
	templates := service.GetEnabledTemplatesForAPI()
	if len(templates) == 0 {
		response.OK(c, gin.H{
			"templates": []map[string]string{},
			"enabled":   false,
		})
		return
	}
	response.OK(c, gin.H{
		"templates": templates,
		"enabled":   true,
	})
}

