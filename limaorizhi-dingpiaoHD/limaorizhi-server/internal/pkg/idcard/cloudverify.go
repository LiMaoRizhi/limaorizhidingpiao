// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package idcard

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"limaorizhi-server/internal/pkg/redis"
)

// 云市场API专用HTTP客户端（带超时，防止DoS）
var cloudHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// CloudVerifier 云市场实名认证客户端
type CloudVerifier struct {
	AppCode     string
	Endpoint    string // API地址，如 https://idenauthen.market.alicloudapi.com
	Path        string // API路径，如 /idenAuthentication
	Enabled     bool   // 是否启用实名认证
	StrictMode  bool   // 严格模式：API失败时返回错误而非降级
	CacheTTL    time.Duration // 认证结果缓存有效期，<=0 时使用 redis 包默认 30 天。从 config.IDCardVerifyConfig.CacheTTL 注入，实现不重启调参
}

// NewCloudVerifier 从配置创建实名认证客户端
// cacheTTL: 认证结果缓存有效期(秒)，传 0 使用默认 30 天。对应 config.IDCardVerifyConfig.CacheTTL
func NewCloudVerifier(appCode, endpoint, path string, enabled, strictMode bool, cacheTTL int) *CloudVerifier {
	return &CloudVerifier{
		AppCode:    appCode,
		Endpoint:   endpoint,
		Path:       path,
		Enabled:    enabled,
		StrictMode: strictMode,
		CacheTTL:   time.Duration(cacheTTL) * time.Second,
	}
}

// cloudVerifyResponse 云市场API返回结构（杭州快证签科技有限公司 二要素核验）
// 返回示例：{"msg":"成功","success":true,"code":200,"data":{"birthday":"19840816","result":0,"address":"浙江省杭州市","orderNo":"xxx","sex":"男","desc":"一致"}}
type cloudVerifyResponse struct {
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Data    struct {
		Birthday string `json:"birthday"`
		Result   int    `json:"result"` // 0=一致, 1=不一致
		Address  string `json:"address"`
		OrderNo  string `json:"orderNo"`
		Sex      string `json:"sex"`
		Desc     string `json:"desc"`
	} `json:"data"`
}

// VerifyResult 实名认证结果
type VerifyResult struct {
	Matched  bool   // 姓名和身份证号是否匹配
	Message   string // 结果描述
	Province string // 省份
	City      string // 城市
	Sex       string // 性别
}

// Verify 调用云市场API进行姓名+身份证号二要素核验
// 返回：匹配结果和错误信息（错误表示API调用失败，不代表不匹配）
// 包含最多2次重试机制，减少因网络抖动导致的认证失败
//
// 集成 Redis 缓存：同一姓名+身份证号组合在 30 天内复用认证结果，避免重复调用云市场 API（0.3 元/次）
// 缓存 key 同时包含姓名和身份证号的 sha256，防止同一身份证号被不同姓名冒用时错误命中“匹配”缓存
// Redis 不可用或调用失败时自动降级为直接调用 API，不影响业务
func (v *CloudVerifier) Verify(name, idCardNo string) (*VerifyResult, error) {
	// 规范化：身份证号末位小写x转大写X
	// 防止：1）云市场API不识别小写x返回不匹配 2）缓存key不一致导致同一身份证号缓存不命中
	idCardNo = strings.ToUpper(idCardNo)

	if !v.Enabled {
		// 未启用实名认证，直接返回匹配（跳过验证）
		return &VerifyResult{Matched: true, Message: "实名认证未启用"}, nil
	}
	if v.AppCode == "" {
		// fail-closed：已启用但未配置AppCode时拒绝认证
		return nil, errors.New("实名认证已启用但AppCode未配置，请联系管理员设置IDCARD_VERIFY_APPCODE")
	}

	// 1. 先查 Redis 缓存，命中则直接返回，避免重复调用云市场 API
	// fallback=true 表示 Redis 不可用，降级为直接调用 API；cache!=nil 表示命中缓存
	// 计数器：fallback=true 不计入 hit/miss（Redis 不可用时本就没有缓存可命中）
	if cache, fallback := redis.GetIDCardVerify(name, idCardNo); !fallback && cache != nil {
		incCacheHit()
		return &VerifyResult{
			Matched: cache.Matched,
			Message: "实名认证缓存命中（30天内复用）",
		}, nil
	} else if !fallback {
		// Redis 可用但未命中缓存，计为 miss（便于计算命中率）
		incCacheMiss()
	}

	// 2. 未命中缓存或 Redis 不可用，走正常 API 调用流程
	var lastErr error
	const maxRetries = 2
	for attempt := 0; attempt < maxRetries; attempt++ {
		incAPICall() // 计 API 调用次数（含重试，反映真实请求量，包含 0.3 元/次成本）
		result, err := v.doVerify(name, idCardNo)
		if err == nil {
			// 3. API 调用成功，写入缓存（仅缓存明确结果 matched=true/false，错误不缓存）
			// 写入失败不影响业务，下次仍会调用 API
			// TTL 从 CloudVerifier 配置注入，<=0 时由 redis 包使用默认 30 天
			if fallback := redis.SetIDCardVerify(name, idCardNo, result.Matched, v.CacheTTL); !fallback {
				incCacheWrite() // 仅在成功写入时计数（Redis 不可用时不计）
			}
			return result, nil
		}
		incAPIError() // 计 API 失败次数（网络错误/超时/状态码非200等）
		lastErr = err
		// 重试前等待短时间
		if attempt < maxRetries-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil, lastErr
}

// doVerify 单次调用云市场API进行认证
func (v *CloudVerifier) doVerify(name, idCardNo string) (*VerifyResult, error) {
	// 构造请求体
	formData := url.Values{}
	formData.Set("name", name)
	formData.Set("idcard", idCardNo)

	req, err := http.NewRequest("POST", v.Endpoint+v.Path, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建认证请求失败: %w", err)
	}

	// 设置请求头：AppCode认证
	req.Header.Set("Authorization", "APPCODE "+v.AppCode)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := cloudHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("实名认证服务请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取认证结果失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		// 常见错误：403=AppCode无效或欠费，429=超出QPS限制
		// 打印响应体帮助诊断（如 "Unauthorized"=AppCode不匹配，"Quota Exhausted"=次数用完）
		log.Printf("[ERROR] 实名认证API返回状态码:%d, 响应体:%s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("实名认证服务异常(状态码:%d, 响应:%s)", resp.StatusCode, string(body))
	}

	var result cloudVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析认证结果失败: %w", err)
	}

	// data.result: 0=一致, 1=不一致；success=true 且 code=200 表示API调用成功
	matched := result.Success && result.Code == 200 && result.Data.Result == 0

	return &VerifyResult{
		Matched:  matched,
		Message:  result.Data.Desc,
		Province: result.Data.Address,
		Sex:      result.Data.Sex,
	}, nil
}

// IsStrictMode 返回是否启用严格模式
func (v *CloudVerifier) IsStrictMode() bool {
	return v.StrictMode
}
