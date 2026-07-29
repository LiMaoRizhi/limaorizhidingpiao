package admin

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"limaorizhi-server/internal/pkg/crypto"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// sseSemaphore 限制并发SSE连接数，防止有心人用Admin Token开大量连接耗尽fd
// channel当信号量用，满了就拒绝新连接（CodeServerError → HTTP 500）
// 20个并发够用了，轻量客运系统不需要那么多
var sseSemaphore = make(chan struct{}, 20)

// Chat SSE流式透传AI回复 前端得用fetch+ReadableStream接收
// EventSource没法带JWT头 只能这么搞
func (h *AIHandler) Chat(c *gin.Context) {
	if h.configGet("ai_employee_enabled") != "true" {
		response.FailMsg(c, response.CodeForbidden, "AI数字员工未启用，请在系统配置中开启")
		return
	}

	// 并发连接限制：获取信号量，超3秒没拿到就拒绝 防止fd耗尽
	select {
	case sseSemaphore <- struct{}{}:
		defer func() { <-sseSemaphore }()
	case <-time.After(3 * time.Second):
		response.FailMsg(c, response.CodeServerError, "当前AI数字员工使用人数过多，请稍后再试")
		return
	}

	var req struct {
		Messages []struct {
			Role    string      `json:"role"`
			Content interface{} `json:"content"` // string 或多模态数组
		} `json:"messages"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if len(req.Messages) == 0 {
		response.FailMsg(c, response.CodeParamError, "消息不能为空")
		return
	}

	// 安全限制 防止有人恶意构造超大请求 DoS倒不怕 API费用爆炸才要命
	const (
		maxMessages    = 66   // 最多66条消息（含历史）
		maxContentSize = 8000 // 单条文本最大8KB
		maxImages      = 3    // 单条消息最多3张图片
	)
	if len(req.Messages) > maxMessages {
		response.FailMsg(c, response.CodeParamError, fmt.Sprintf("消息数量不能超过%d条", maxMessages))
		return
	}
	for _, msg := range req.Messages {
		// 文本内容大小校验（按Unicode字符计数，中文每字1个rune而非3字节）
		if text := plainText(msg.Content); utf8.RuneCountInString(text) > maxContentSize {
			response.FailMsg(c, response.CodeParamError, fmt.Sprintf("单条消息不能超过%d字符", maxContentSize))
			return
		}
		// 图片数量校验（多模态消息中的 image_url 类型）
		if arr, ok := msg.Content.([]interface{}); ok {
			imgCount := 0
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if t, ok := m["type"].(string); ok && t == "image_url" {
						imgCount++
					}
				}
			}
			if imgCount > maxImages {
				response.FailMsg(c, response.CodeParamError, fmt.Sprintf("单条消息最多包含%d张图片", maxImages))
				return
			}
		}
	}

	baseURL := h.configGet("ai_base_url")
	modelName := h.configGet("ai_model")

	// 旧模型名兼容 deepseek-chat那些7月24号停用了 DB存旧名自动映射到V4
	// 不映射的话线上调用直接404
	// NIM平台下线的旧模型也一起映射
	switch modelName {
	case "deepseek-chat":
		modelName = "deepseek-v4-flash"
	case "deepseek-reasoner":
		modelName = "deepseek-v4-pro"
	case "meta/llama-3.1-405b-instruct":
		modelName = "meta/llama-3.3-70b-instruct"
	case "microsoft/phi-4":
		modelName = "openai/gpt-oss-120b"
	case "google/gemma-3-27b-it":
		modelName = "google/gemma-4-31b-it"
	case "mistralai/mistral-large-2411-instruct":
		modelName = "mistralai/mistral-medium-3.5-128b"
	case "mistralai/mixtral-8x22b-instruct-v0.1":
		modelName = "mistralai/mistral-medium-3.5-128b"
	case "nvidia/llama-3.3-nemotron-super":
		modelName = "nvidia/llama-3.3-nemotron-super-49b-v1.5"
	case "moonshotai/kimi-k2.5":
		modelName = "moonshotai/kimi-k2.6"
	}

	systemPrompt := h.configGet("ai_system_prompt")
	provider := h.configGet("ai_provider")
	if provider == "" {
		provider = "nvidia"
	}
	// auto模式强制使用nvidia平台（候选模型都是nvidia的，防止DB里provider和model不匹配）
	if modelName == "auto" && provider != "nvidia" {
		provider = "nvidia"
		baseURL = "https://integrate.api.nvidia.com/v1"
	}
	// nvidia能复用旧版统一的ai_api_key 其他服务商得有专属Key
	apiKeyEnc := h.configGet(fmt.Sprintf("ai_api_key_%s", provider))
	if apiKeyEnc == "" && provider == "nvidia" {
		apiKeyEnc = h.configGet("ai_api_key")
	}
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}

	if baseURL == "" || modelName == "" {
		response.FailMsg(c, response.CodeServerError, "AI配置不完整，请在系统配置中设置 Base URL 和模型名称")
		return
	}
	if apiKeyEnc == "" {
		response.FailMsg(c, response.CodeServerError, "AI API Key 未配置，请在系统配置中设置")
		return
	}

	apiKey, err := crypto.Decrypt(apiKeyEnc)
	if err != nil {
		log.Printf("AI API Key 解密失败: %v", err)
		response.FailMsg(c, response.CodeServerError, "AI API Key 解密失败，请重新配置")
		return
	}

	// 构造OpenAI兼容请求体
	chatMessages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
	}

	// 上下文注入 按用户最后一条消息关键词拉相关业务数据
	lastUserMsg := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMsg = plainText(req.Messages[i].Content)
			break
		}
	}
	if ctx := h.buildContext(lastUserMsg); ctx != "" {
		chatMessages = append(chatMessages, map[string]interface{}{
			"role":    "system",
			"content": ctx,
		})
	}

	for _, msg := range req.Messages {
		role := msg.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		chatMessages = append(chatMessages, map[string]interface{}{
			"role":    role,
			"content": msg.Content,
		})
	}

	// 检测消息中是否包含图片（多模态content）
	hasImages := false
	for _, msg := range req.Messages {
		if arr, ok := msg.Content.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if t, ok := m["type"].(string); ok && t == "image_url" {
						hasImages = true
						break
					}
				}
			}
		}
		if hasImages {
			break
		}
	}

	// 狸猫员工(auto)模式：自动选择最优可用模型，谁能用用谁
	// 模型下架/限流/超时自动降级到下一个
	var candidateModels []string
	if modelName == "auto" {
		candidateModels = autoCandidateModels(hasImages)
		if len(candidateModels) == 0 {
			response.FailMsg(c, response.CodeServerError, "狸猫员工无可用模型，请检查模型配置")
			return
		}
		log.Printf("[狸猫员工] 自动模式启动，候选模型 %d 个，含图片: %v", len(candidateModels), hasImages)
	} else {
		candidateModels = []string{modelName}
	}

	// 逐个尝试候选模型，第一个返回200的就用它
	var httpResp *http.Response
	var usedModelName string
	client := &http.Client{Timeout: 300 * time.Second}
	chatURL := strings.TrimRight(baseURL, "/") + "/chat/completions"

	for _, tryModel := range candidateModels {
		requestBody, err := json.Marshal(map[string]interface{}{
			"model":       tryModel,
			"messages":    chatMessages,
			"stream":      true,
			"max_tokens":  4096,
			"temperature": 0.7,
		})
		if err != nil {
			continue
		}

		httpReq, err := http.NewRequest("POST", chatURL, bytes.NewReader(requestBody))
		if err != nil {
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("[AI] 请求模型 %s 失败: %v", tryModel, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Printf("[AI] 模型 %s 返回错误 %d: %s", tryModel, resp.StatusCode, string(body))
			// auto模式：任何非200都尝试下一个模型（模型下架/限流/超时自动降级）
			if modelName == "auto" {
				continue
			}
			// 非auto模式：直接报错给前端
			writeSSEHeaders(c)
			errMsg := fmt.Sprintf("AI服务商返回错误(HTTP %d)，请检查API Key和模型配置", resp.StatusCode)
			if resp.StatusCode == 504 {
				errMsg = "模型响应超时(504)，该模型可能在英伟达平台上较慢或暂时不可用，请尝试其他模型"
			}
			c.SSEvent("message", map[string]string{"type": "content", "content": errMsg})
			c.SSEvent("message", map[string]string{"type": "done"})
			c.Writer.Flush() // 立即推送，避免数据滞留缓冲区导致前端长时间等待
			return
		}

		// 成功！用这个模型
		httpResp = resp
		usedModelName = tryModel
		break
	}

	if httpResp == nil {
		// 所有候选模型都失败了
		writeSSEHeaders(c)
		errMsg := "AI模型暂时不可用，请稍后重试或手动选择模型"
		if modelName != "auto" {
			errMsg = "请求AI服务商失败，请检查网络或API配置"
		}
		log.Printf("[AI] 所有候选模型均失败，model=%s", modelName)
		c.SSEvent("message", map[string]string{"type": "content", "content": errMsg})
		c.SSEvent("message", map[string]string{"type": "done"})
		c.Writer.Flush()
		return
	}
	defer httpResp.Body.Close()

	if modelName == "auto" {
		log.Printf("[狸猫员工] 自动选择模型: %s", usedModelName)
	}

	writeSSEHeaders(c)
	c.Writer.Flush() // 立即推送响应头，让前端尽早感知连接已建立

	c.Stream(func(w io.Writer) bool {
		scanner := bufio.NewScanner(httpResp.Body)
		// 4MB buffer：推理模型（DeepSeek-R1等）reasoning_content 可能超1MB
		// make第二个参数必须是length而非capacity，不然Scanner不会用预分配buf
		scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			// OpenAI SSE以data:开头 data: {json}和data:{json}都兼容
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			if data == "[DONE]" {
				c.SSEvent("message", map[string]string{"type": "done"})
				c.Writer.Flush() // 立即推送done，前端收到后退出等待
				return false
			}

			// delta.content正文 reasoning_content是推理模型的思考过程
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content          string `json:"content"`
						ReasoningContent string `json:"reasoning_content"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
				Error *struct {
					Message string `json:"message"`
					Type    string `json:"type"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				log.Printf("SSE解析失败: %s, raw: %s", err.Error(), data)
				continue
			}

			if chunk.Error != nil {
				log.Printf("服务商SSE错误: %s (%s)", chunk.Error.Message, chunk.Error.Type)
				c.SSEvent("message", map[string]string{
					"type":    "content",
					"content": "请求失败：" + chunk.Error.Message,
				})
				c.SSEvent("message", map[string]string{"type": "done"})
				c.Writer.Flush() // 错误路径立即推送，避免前端长时间等待
				return false
			}

			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				if delta.ReasoningContent != "" {
					c.SSEvent("message", map[string]string{
						"type":    "reasoning",
						"content": delta.ReasoningContent,
					})
					c.Writer.Flush() // 逐条推送，前端实时显示思考过程
				}
				if delta.Content != "" {
					c.SSEvent("message", map[string]string{
						"type":    "content",
						"content": delta.Content,
					})
					c.Writer.Flush() // 逐条推送，前端实时显示正文
				}
			}
			// 客户端断了就别继续处理了 白浪费带宽和API钱
			select {
			case <-c.Request.Context().Done():
				log.Printf("客户端已断开连接，终止SSE流")
				return false
			default:
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("读取SSE流失败: %v", err)
			// 通知前端连接意外断开，否则UI永久卡在"正在回答..."
			c.SSEvent("message", map[string]string{
				"type":    "content",
				"content": fmt.Sprintf("连接中断：%v，请稍后重试", err),
			})
			c.SSEvent("message", map[string]string{"type": "done"})
			c.Writer.Flush() // 错误路径立即推送
			return false
		}
		// 补发done 服务商有时候不发[DONE]标记 前端收不到done会一直卡在"正在回答"
		c.SSEvent("message", map[string]string{"type": "done"})
		c.Writer.Flush() // 立即推送done，前端收到后退出等待
		return false
	})
}

// writeSSEHeaders SSE响应头 X-Accel-Buffering:no让nginx别缓冲 不然推送不实时
func writeSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

// autoCandidateModels 返回狸猫员工自动模式的候选模型列表
// 视觉模型优先（已按参数量从小到大排序=速度从快到慢）
// 有图片时只尝试视觉模型，无图片时视觉+文本都试
func autoCandidateModels(hasImages bool) []string {
	var candidates []string
	for _, p := range aiProviders {
		if p.Value != "nvidia" {
			continue
		}
		if hasImages {
			// 有图片：只尝试视觉模型（文本模型收到image_url会报错）
			for _, m := range p.Models {
				if m.ID == "auto" || m.Icon == "image" {
					continue
				}
				if m.SupportsVision {
					candidates = append(candidates, m.ID)
				}
			}
		} else {
			// 纯文本：视觉模型优先（小模型快），然后纯文本模型
			for _, m := range p.Models {
				if m.ID == "auto" || m.Icon == "image" {
					continue
				}
				candidates = append(candidates, m.ID)
			}
		}
		break
	}
	return candidates
}
