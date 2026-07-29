// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/triptime"
	"limaorizhi-server/internal/pkg/wxtoken"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// notifyConcurrency 订阅消息并发发送上限，避免50座大巴瞬时spawn 50个goroutine+50次HTTP请求
const notifyConcurrency = 5

// 微信订阅消息服务
// 机制说明：
//   1. 小程序端在关键操作（如支付前）调用 wx.requestSubscribeMessage 弹窗请求用户授权
//   2. 用户授权后，小程序调用 POST /api/wx/subscribe 上报，后端为该用户增加1次发送配额
//   3. 后端在支付回调/发车/到达/退款等事件触发时，检查配额并发送订阅消息
//   4. 每次发送消耗1次配额，配额用尽后无法再发送，需用户下次操作时再次授权
//
// access_token 说明：
//   本文件与 handler/wx/user.go 共用 internal/pkg/wxtoken 包获取 access_token，
//   多实例通过 Redis 共享缓存键（wx:access_token），同一服务只需请求一次微信API。

// 订阅消息模板键（对应 SubscribeQuota.TemplateKey）
const (
	TemplatePaymentSuccess = "payment_success" // 支付成功通知
	TemplateTripDeparture  = "trip_departure"  // 班次发车通知
	TemplateTripArrival    = "trip_arrival"    // 班次到达通知
	TemplateRefundSuccess  = "refund_success"  // 退款到账通知
)

// notifyHTTPClient 订阅消息发送专用HTTP客户端
var notifyHTTPClient = &http.Client{Timeout: 10 * time.Second}

// getMiniprogramState 根据服务运行模式返回小程序状态
// formal=正式版 developer=开发版 trial=体验版
func getMiniprogramState() string {
	if config.AppConfig.Server.Mode == "release" {
		return "formal"
	}
	return "developer"
}

// getTemplateID 获取指定类型的订阅消息模板ID
// 未配置则返回空串，调用方据此跳过发送
func getTemplateID(key string) string {
	switch key {
	case TemplatePaymentSuccess:
		return config.AppConfig.Wechat.SubscribeTemplates.PaymentSuccess
	case TemplateTripDeparture:
		return config.AppConfig.Wechat.SubscribeTemplates.TripDeparture
	case TemplateTripArrival:
		return config.AppConfig.Wechat.SubscribeTemplates.TripArrival
	case TemplateRefundSuccess:
		return config.AppConfig.Wechat.SubscribeTemplates.RefundSuccess
	}
	return ""
}

// AnyTemplateConfigured 检查是否有任何订阅消息模板已配置
// 供前端判断是否需要调用 wx.requestSubscribeMessage
func AnyTemplateConfigured() bool {
	return getTemplateID(TemplatePaymentSuccess) != "" ||
		getTemplateID(TemplateTripDeparture) != "" ||
		getTemplateID(TemplateTripArrival) != "" ||
		getTemplateID(TemplateRefundSuccess) != ""
}

// GetEnabledTemplateKeys 返回所有已配置的模板键列表
// 前端调用 wx.requestSubscribeMessage 时需要传入模板ID列表，此函数返回对应的键
func GetEnabledTemplateKeys() []string {
	all := []string{TemplatePaymentSuccess, TemplateTripDeparture, TemplateTripArrival, TemplateRefundSuccess}
	var enabled []string
	for _, k := range all {
		if getTemplateID(k) != "" {
			enabled = append(enabled, k)
		}
	}
	return enabled
}

// GetTemplateIDs 返回已配置的模板键→模板ID映射，供前端获取需要请求订阅的模板ID
func GetTemplateIDs() map[string]string {
	result := make(map[string]string)
	for _, key := range []string{TemplatePaymentSuccess, TemplateTripDeparture, TemplateTripArrival, TemplateRefundSuccess} {
		if id := getTemplateID(key); id != "" {
			result[key] = id
		}
	}
	return result
}

// AddSubscribeQuota 增加用户的订阅消息发送配额
// 前端 wx.requestSubscribeMessage 返回结果后调用，用户授权的每个模板配额+1
func AddSubscribeQuota(db *gorm.DB, userID uint, templateKeys []string) {
	for _, key := range templateKeys {
		if getTemplateID(key) == "" {
			continue // 模板未配置，跳过
		}
		// 使用 upsert 原子增加配额，避免并发时重复创建或丢失更新
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "template_key"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"quota": gorm.Expr("quota + 1"),
			}),
		}).Create(&model.SubscribeQuota{UserID: userID, TemplateKey: key, Quota: 1}).Error; err != nil {
			log.Printf("[订阅配额] 增加配额失败 userID=%d key=%s err=%v", userID, key, err)
		}
	}
}

// consumeQuota 原子消耗1次发送配额（WHERE quota > 0 防并发）
// 返回true表示有配额已消耗，false表示无配额
func consumeQuota(db *gorm.DB, userID uint, templateKey string) bool {
	result := db.Model(&model.SubscribeQuota{}).
		Where("user_id = ? AND template_key = ? AND quota > 0", userID, templateKey).
		UpdateColumn("quota", gorm.Expr("quota - 1"))
	return result.RowsAffected > 0
}

// subscribeMsgData 订阅消息data字段值（{value: "xxx"}）
type subscribeMsgData struct {
	Value string `json:"value"`
}

// sendSubscribeMessage 调用微信API发送一条订阅消息
// 返回nil表示发送成功，error包含微信返回的错误信息
func sendSubscribeMessage(openid, templateID, page string, data map[string]subscribeMsgData) error {
	token := wxtoken.GetAccessToken(config.AppConfig.Wechat.Appid, config.AppConfig.Wechat.Secret)
	if token == "" {
		return fmt.Errorf("access_token获取失败")
	}

	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s",
		url.QueryEscape(token),
	)

	reqBody := map[string]interface{}{
		"touser":            openid,
		"template_id":       templateID,
		"page":              page,
		"data":              data,
		"miniprogram_state": getMiniprogramState(),
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := notifyHTTPClient.Post(apiURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}
	return nil
}

// safeSendSubscribeMessage 发送订阅消息（带 panic 恢复，不影响主流程）
// 在goroutine中调用，失败仅记录日志，不阻塞业务逻辑
func safeSendSubscribeMessage(db *gorm.DB, userID uint, templateKey, page string, data map[string]subscribeMsgData) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] wx_notify 发送订阅消息panic: %v\n", r)
		}
	}()

	templateID := getTemplateID(templateKey)
	if templateID == "" {
		return // 模板未配置，静默跳过
	}

	// 先查用户OpenID，再消耗配额
	// 顺序不能反：若OpenID为空时已消耗配额，用户授权就白白浪费了
	var user model.User
	if err := db.Select("openid").First(&user, userID).Error; err != nil {
		log.Printf("[WARN] wx_notify: 查询用户openid失败 userID=%d: %v\n", userID, err)
		return
	}
	if user.OpenID == "" {
		return // 用户无OpenID（未绑定微信），无法发送
	}

	// 检查并消耗配额（WHERE quota > 0 防并发，原子操作）
	if !consumeQuota(db, userID, templateKey) {
		return // 无配额，用户未订阅或已用完
	}

	// 发送订阅消息
	if err := sendSubscribeMessage(user.OpenID, templateID, page, data); err != nil {
		// 发送失败不回退配额：微信API可能已收到请求但响应超时
		// 宁可漏发也不重复发送（用户侧不会收到重复通知）
		log.Printf("[WARN] wx_notify: 发送%s失败 userID=%d order_page=%s: %v\n",
			templateKey, userID, page, err)
	}
}

// routeName 拼接路线名（上车站 → 下车站）
func routeName(from, to string) string {
	return from + " → " + to
}

// formatAmount 格式化金额为微信订阅消息的amount格式（¥xx.xx）
func formatAmount(amount float64) string {
	return fmt.Sprintf("¥%.2f", amount)
}

// formatTime 格式化时间为 2006-01-02 15:04
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// NotifyPaymentSuccess 发送支付成功通知（异步，不阻塞支付回调）
// 在微信支付回调 PayNotify 中事务提交成功后调用
func NotifyPaymentSuccess(db *gorm.DB, order model.Order) {
	go safeSendSubscribeMessage(db, order.UserID, TemplatePaymentSuccess,
		fmt.Sprintf("pages/order-detail/order-detail?id=%d", order.ID),
		map[string]subscribeMsgData{
			"thing1":            {Value: truncateStr(routeName(order.FromStationName, order.ToStationName), 20)},
			"character_string2": {Value: order.OrderNo},
			"amount3":           {Value: formatAmount(order.TotalPrice)},
			"time4":             {Value: formatTime(time.Now())},
		},
	)
}

// NotifyTripDeparture 发送班次发车通知（异步，批量发送给该班次所有已支付订单的用户）
// 在司机StartTrip或定时任务departTrips中班次状态 1→2 时调用
func NotifyTripDeparture(db *gorm.DB, trip model.Trip) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] wx_notify NotifyTripDeparture panic: %v\n", r)
			}
		}()

		templateID := getTemplateID(TemplateTripDeparture)
		if templateID == "" {
			return
		}

		// 查询该班次所有已支付(status=1)的车票订单
		var orders []model.Order
		if err := db.Where("trip_id = ? AND status = ? AND order_type = 1", trip.ID, model.OrderStatusPaid).Find(&orders).Error; err != nil {
			log.Printf("[WARN] wx_notify: 查询班次已支付订单失败 tripID=%d: %v\n", trip.ID, err)
			return
		}

		departureTimeStr := triptime.FormatDateTime(string(trip.TripDate), trip.DepartureTime)
		// 限制并发：信号量+WaitGroup，避免瞬时大量goroutine+HTTP请求
		sem := make(chan struct{}, notifyConcurrency)
		var wg sync.WaitGroup
		for _, order := range orders {
			order := order
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				safeSendSubscribeMessage(db, order.UserID, TemplateTripDeparture,
					fmt.Sprintf("pages/order-detail/order-detail?id=%d", order.ID),
					map[string]subscribeMsgData{
						"character_string1": {Value: trip.TripNo},
						"thing2":            {Value: truncateStr(routeName(order.FromStationName, order.ToStationName), 20)},
						"time3":             {Value: departureTimeStr},
						"thing4":            {Value: truncateStr(order.FromStationName, 20)},
					},
				)
			}()
		}
		wg.Wait()
	}()
}

// NotifyTripArrival 发送班次到达通知（异步，批量发送给该班次所有已完成订单的用户）
// 在班次到达终点(completeArrivedTrip)班次状态 2→4 时调用
func NotifyTripArrival(db *gorm.DB, trip model.Trip) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] wx_notify NotifyTripArrival panic: %v\n", r)
			}
		}()

		templateID := getTemplateID(TemplateTripArrival)
		if templateID == "" {
			return
		}

		// 查询该班次所有已完成(status=2)的车票订单
		var orders []model.Order
		if err := db.Where("trip_id = ? AND status = ? AND order_type = 1", trip.ID, model.OrderStatusCompleted).Find(&orders).Error; err != nil {
			log.Printf("[WARN] wx_notify: 查询班次已完成订单失败 tripID=%d: %v\n", trip.ID, err)
			return
		}

		arrivalTimeStr := triptime.FormatDateTime(string(trip.TripDate), trip.ArrivalTime)
		sem := make(chan struct{}, notifyConcurrency)
		var wg sync.WaitGroup
		for _, order := range orders {
			order := order
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				safeSendSubscribeMessage(db, order.UserID, TemplateTripArrival,
					fmt.Sprintf("pages/order-detail/order-detail?id=%d", order.ID),
					map[string]subscribeMsgData{
						"character_string1": {Value: trip.TripNo},
						"thing2":            {Value: truncateStr(routeName(order.FromStationName, order.ToStationName), 20)},
						"time3":             {Value: arrivalTimeStr},
						"thing4":            {Value: truncateStr(order.ToStationName, 20)},
					},
				)
			}()
		}
		wg.Wait()
	}()
}

// NotifyRefundSuccess 发送退款到账通知（异步）
// 在微信退款回调 RefundNotify 中退款状态更新为成功时调用
func NotifyRefundSuccess(db *gorm.DB, refund model.Refund, order model.Order) {
	go safeSendSubscribeMessage(db, order.UserID, TemplateRefundSuccess,
		fmt.Sprintf("pages/order-detail/order-detail?id=%d", order.ID),
		map[string]subscribeMsgData{
			"character_string2": {Value: order.OrderNo},
			"amount3":           {Value: formatAmount(refund.Amount)},
			"time4":             {Value: formatTime(time.Now())},
		},
	)
}

// truncateStr 截断字符串到指定长度（微信订阅消息thing类型限制20字符）
func truncateStr(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen])
}

// GetEnabledTemplatesForAPI 返回已配置的模板ID列表，供前端wx.requestSubscribeMessage使用
// 返回格式：[{key: "payment_success", template_id: "xxx"}, ...]
func GetEnabledTemplatesForAPI() []map[string]string {
	result := make([]map[string]string, 0)
	for _, key := range []string{TemplatePaymentSuccess, TemplateTripDeparture, TemplateTripArrival, TemplateRefundSuccess} {
		if id := getTemplateID(key); id != "" {
			result = append(result, map[string]string{
				"key":         key,
				"template_id": id,
			})
		}
	}
	return result
}

