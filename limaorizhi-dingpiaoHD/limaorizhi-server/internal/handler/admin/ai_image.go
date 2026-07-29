package admin

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GenerateImage 调Pollinations.ai生图 返回base64给前端直接显示
// Pollinations免费不要Key 想参考别人的实现发现Go的几乎没有 呜呜呜
func (h *AIHandler) GenerateImage(c *gin.Context) {
	if h.configGet("ai_employee_enabled") != "true" {
		response.FailMsg(c, response.CodeForbidden, "AI数字员工未启用，请在系统配置中开启")
		return
	}

	var req struct {
		Prompt string `json:"prompt" binding:"required"`
		Model  string `json:"model"` // flux-realistic/flux-portrait/flux/turbo/sana
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailMsg(c, response.CodeParamError, "请输入图片描述")
		return
	}
	// 按rune（Unicode字符）计数而非字节，中文每字3字节否则500字节仅≈166个汉字
	if utf8.RuneCountInString(req.Prompt) > 800 {
		response.FailMsg(c, response.CodeParamError, "图片描述过长，请控制在800字以内")
		return
	}

	// 虚拟模型ID映射到Pollinations实际模型 加摄影后缀出图自然点
	type imgCfg struct {
		model  string
		suffix string
	}
	cfgs := map[string]imgCfg{
		"flux-realistic": {"flux", ", photorealistic, ultra realistic, natural lighting, 8k, professional photography, highly detailed"},
		"flux-portrait":  {"flux", ", portrait photography, shallow depth of field, natural lighting, bokeh, 8k, professional photo"},
		"flux":           {"flux", ""},
		"turbo":          {"turbo", ""},
		"sana":           {"sana", ""},
	}
	cfg, ok := cfgs[req.Model]
	if !ok {
		cfg = cfgs["flux-realistic"]
	}

	enhancedPrompt := req.Prompt + cfg.suffix
	// PathEscape空格编码成%20不是+ 放URL路径里用
	encodedPrompt := url.PathEscape(enhancedPrompt)
	// 随机seed防Pollinations缓存 范围拉大到9位数避免相近请求命中相同seed
	seed := rand.Intn(999999999)
	imageURL := fmt.Sprintf("https://image.pollinations.ai/prompt/%s?width=1024&height=1024&model=%s&nologo=true&seed=%d", encodedPrompt, cfg.model, seed)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		log.Printf("[ERROR] 图片生成请求失败: %v", err)
		response.FailMsg(c, response.CodeServerError, "图片生成请求失败，请稍后重试")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("图片生成返回错误 %d: %s", resp.StatusCode, string(body))
		response.FailMsg(c, response.CodeServerError, fmt.Sprintf("图片生成失败(HTTP %d)，请稍后重试", resp.StatusCode))
		return
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "读取图片数据失败")
		return
	}

	mimeType := "image/png"
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "jpeg") || strings.Contains(ct, "jpg") {
		mimeType = "image/jpeg"
	}

	response.OK(c, gin.H{
		"image":  "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(imgBytes),
		"prompt": req.Prompt,
	})
}
