package admin

import (
	"log"

	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
)

// CaptchaHandler 滑动验证码处理器
type CaptchaHandler struct{}

func NewCaptchaHandler() *CaptchaHandler {
	return &CaptchaHandler{}
}

// GetCaptcha 生成并返回滑动验证码
// GET /admin/captcha/get
func (h *CaptchaHandler) GetCaptcha(c *gin.Context) {
	result, err := service.GenerateCaptcha()
	if err != nil {
		log.Printf("[Captcha] 生成验证码失败: %v", err)
		response.FailMsg(c, response.CodeServerError, "生成验证码失败: "+err.Error())
		return
	}
	response.OK(c, result)
}

// CheckCaptcha 验证滑动结果
// POST /admin/captcha/check
type checkCaptchaRequest struct {
	Token string `json:"token" binding:"required"`
	MoveX int    `json:"moveX" binding:"required"`
}

func (h *CaptchaHandler) CheckCaptcha(c *gin.Context) {
	var req checkCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	verified, verifyToken, err := service.CheckCaptcha(req.Token, req.MoveX)
	if err != nil {
		response.FailMsg(c, response.CodeForbidden, err.Error())
		return
	}

	if !verified {
		response.OK(c, gin.H{"result": false})
		return
	}

	response.OK(c, gin.H{
		"result":      true,
		"verifyToken": verifyToken,
	})
}
