// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// 通用保险对接框架
// 设计目标：上线前由超级管理员在管理端填入保险公司配置(API地址/商户号/密钥/产品代码)，
// 系统在订单支付成功后自动调用保险公司出单API获取真实保单号并回填到订单。
// 同一时刻仅一家保险公司 is_active=true，切换启用时其余自动置 false。
//
// 出单协议（标准JSON + HMAC-SHA256签名）：
//   请求 POST {provider.APIURL}
//   body: {app_id, product_code, order_no, trip_date, departure_time,
//          from_station, to_station, passengers:[{name,id_card_type,id_card_no,phone}],
//          contact_name, contact_phone, premium, timestamp, nonce, signature}
//   签名: 将业务字段(除 signature)按 key 升序拼成 k=v&k=v，用 app_secret 做 HMAC-SHA256，hex 输出
//   响应: {code:0成功, message, data:{policy_no, policy_url(可选)}}，code!=0 视为失败
//
// 失败处理：仅记日志 + 写 OperationLog 告警，不阻塞支付流程（支付已成功）。

const (
	insuranceHTTPTimeout = 15 * time.Second // 单次HTTP请求超时
	insuranceMaxAttempts = 3                // 含首次 + 2次重试
	insuranceRetryDelay  = 1 * time.Second  // 重试间隔
)

// insuranceIssueRequest 出单请求体（业务字段 + signature）
type insuranceIssueRequest struct {
	AppID         string               `json:"app_id"`
	ProductCode   string               `json:"product_code"`
	OrderNo       string               `json:"order_no"`
	TripDate      string               `json:"trip_date"`
	DepartureTime string               `json:"departure_time"`
	FromStation   string               `json:"from_station"`
	ToStation     string               `json:"to_station"`
	Passengers    []insurancePassenger `json:"passengers"`
	ContactName   string               `json:"contact_name"`
	ContactPhone  string               `json:"contact_phone"`
	Premium       float64              `json:"premium"`
	Timestamp     int64                `json:"timestamp"`
	Nonce         string               `json:"nonce"`
	Signature     string               `json:"signature"`
}

type insurancePassenger struct {
	Name       string `json:"name"`
	IDCardType int8   `json:"id_card_type"`
	IDCardNo   string `json:"id_card_no"`
	Phone      string `json:"phone"`
}

// insuranceIssueResponse 出单响应体
type insuranceIssueResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PolicyNo  string `json:"policy_no"`
		PolicyURL string `json:"policy_url,omitempty"`
	} `json:"data"`
}

// GetActiveProvider 返回当前启用的保险公司配置（is_active=true 第一条）。
// 无启用配置时返回 nil, nil（调用方据此回退到 system_configs 兜底）。
func GetActiveProvider(db *gorm.DB) (*model.InsuranceProvider, error) {
	var provider model.InsuranceProvider
	if err := db.Where("is_active = ?", true).First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

// IssuePolicy 调用保险公司出单API获取保单号。
// 流程：组装业务字段 → HMAC-SHA256签名 → POST JSON → 解析响应。
// 失败重试最多2次（共3次），全部失败返回错误。
// 成功后由调用方负责 UPDATE orders 写回 policy_no（防止重复出单用条件更新）。
func IssuePolicy(db *gorm.DB, order model.Order, passengers []model.OrderPassenger, provider *model.InsuranceProvider) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("保险公司配置为空")
	}
	if provider.APIURL == "" || provider.AppID == "" || provider.AppSecret == "" {
		return "", fmt.Errorf("保险公司配置不完整: name=%s", provider.Name)
	}

	// 组装业务字段
	psg := make([]insurancePassenger, 0, len(passengers))
	for _, p := range passengers {
		psg = append(psg, insurancePassenger{
			Name:       p.Name,
			IDCardType: p.IDCardType,
			IDCardNo:   p.IDCardNo,
			Phone:      p.Phone,
		})
	}
	nonce, _ := generateNonce(16)
	req := insuranceIssueRequest{
		AppID:         provider.AppID,
		ProductCode:   provider.ProductCode,
		OrderNo:       order.OrderNo,
		TripDate:      string(order.TripDate),
		DepartureTime: order.DepartureTime,
		FromStation:   order.FromStationName,
		ToStation:     order.ToStationName,
		Passengers:    psg,
		ContactName:   order.ContactName,
		ContactPhone:  order.ContactPhone,
		Premium:       order.InsuranceFee,
		Timestamp:     time.Now().Unix(),
		Nonce:         nonce,
	}
	req.Signature = signInsuranceRequest(req, provider.AppSecret)

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化出单请求失败: %w", err)
	}

	// 重试
	var lastErr error
	client := &http.Client{Timeout: insuranceHTTPTimeout}
	for attempt := 1; attempt <= insuranceMaxAttempts; attempt++ {
		policyNo, callErr := callInsuranceAPI(client, provider.APIURL, body)
		if callErr == nil {
			return policyNo, nil
		}
		lastErr = callErr
		log.Printf("[WARN] 保险公司出单失败 订单号:%s 提供商:%s 第%d次 err:%v\n",
			order.OrderNo, provider.Name, attempt, callErr)
		if attempt < insuranceMaxAttempts {
			time.Sleep(insuranceRetryDelay)
		}
	}

	// 全部失败：写告警日志
	alertInsuranceFailure(db, order, provider, lastErr)
	return "", lastErr
}

// callInsuranceAPI 单次调用保险公司出单API
func callInsuranceAPI(client *http.Client, apiURL string, body []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP调用失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP状态码 %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var result insuranceIssueResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析响应JSON失败: %w body=%s", err, truncate(string(respBody), 200))
	}
	if result.Code != 0 {
		return "", fmt.Errorf("保险公司返回失败 code=%d message=%s", result.Code, result.Message)
	}
	if result.Data.PolicyNo == "" {
		return "", fmt.Errorf("保险公司返回成功但保单号为空")
	}
	return result.Data.PolicyNo, nil
}

// signInsuranceRequest 按业务字段(除 signature)key 升序拼成 k=v&k=v，
// 用 app_secret 做 HMAC-SHA256，hex 输出。
// passengers 字段以 JSON 字符串参与签名（保证完整且顺序固定）。
func signInsuranceRequest(req insuranceIssueRequest, appSecret string) string {
	// 准备各字段值字符串（passengers 取 JSON）
	psgJSON, _ := json.Marshal(req.Passengers)
	fields := map[string]string{
		"app_id":         req.AppID,
		"product_code":   req.ProductCode,
		"order_no":       req.OrderNo,
		"trip_date":      req.TripDate,
		"departure_time": req.DepartureTime,
		"from_station":   req.FromStation,
		"to_station":     req.ToStation,
		"passengers":     string(psgJSON),
		"contact_name":   req.ContactName,
		"contact_phone":  req.ContactPhone,
		"premium":        fmt.Sprintf("%v", req.Premium),
		"timestamp":      fmt.Sprintf("%d", req.Timestamp),
		"nonce":          req.Nonce,
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(fields[k])
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(b.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

// generateNonce 生成随机 hex 字符串作为 nonce
func generateNonce(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// alertInsuranceFailure 写入 OperationLog 告警，便于运营发现出单失败
func alertInsuranceFailure(db *gorm.DB, order model.Order, provider *model.InsuranceProvider, err error) {
	log.Printf("[ALERT][保险出单失败] 订单号:%s 提供商:%s err:%v\n", order.OrderNo, provider.Name, err)
	_ = db.Create(&model.OperationLog{
		AdminName: "系统自动出单",
		Module:    "保险告警",
		Action:    "出单失败",
		Target:    order.OrderNo,
		Detail:    fmt.Sprintf("订单号:%s 保险公司:%s 出单失败: %v", order.OrderNo, provider.Name, err),
	}).Error
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
