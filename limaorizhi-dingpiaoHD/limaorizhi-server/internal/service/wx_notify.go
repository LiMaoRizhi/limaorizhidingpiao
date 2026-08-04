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

// notifyConcurrency 发消息并发上限5个，50座大巴全员推通知时，
// 别一下子50个goroutine+50个HTTP请求糊微信脸上，糊多了人家限流
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

// AddSubscribeQuota 给用户加订阅消息配额
// 前端 wx.requestSubscribeMessage 出结果后调这个，用户点授权的模板每种+1次
func AddSubscribeQuota(db *gorm.DB, userID uint, templateKeys []string) {
	for _, key := range templateKeys {
		if getTemplateID(key) == "" {
			continue // 模板未配置，跳过
		}
		// upsert 原子加配额，俩人同时授权也不会重复建记录或者把次数搞丢
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

// consumeQuota 原子扣1次配额（WHERE quota > 0 防并发）
// 返回true=扣上了，false=没配额了（用户没订阅或者次数用光了）
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

// sendSubscribeMessage 调微信API发一条订阅消息
// 返回nil=发送成功，error=微信那边给的错误信息
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

// safeSendSubscribeMessage 发订阅消息（带panic兜底，崩了也不拖累主流程）
// 在goroutine里跑的，失败了记个日志拉倒，别耽误正经业务
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

	// 先查OpenID再扣配额，这顺序可不能反了
	// 要是OpenID是空的先把配额扣了，人家授权一次就算白授权了，多亏
	var user model.User
	if err := db.Select("openid").First(&user, userID).Error; err != nil {
		log.Printf("[WARN] wx_notify: 查询用户openid失败 userID=%d: %v\n", userID, err)
		return
	}
	if user.OpenID == "" {
		return // 用户无OpenID（未绑定微信），无法发送
	}

	// 扣配额（WHERE quota > 0 防并发，原子操作）
	if !consumeQuota(db, userID, templateKey) {
		return // 无配额，用户未订阅或已用完
	}

	// 给用户推订阅消息
	if err := sendSubscribeMessage(user.OpenID, templateID, page, data); err != nil {
		// 发失败了不退配额：微信那边可能其实收到了，只是回包超时
		// 宁肯漏一条也不重复推，重复推了用户不烦死也嫌得慌
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

// NotifyTripDeparture 班次发车通知（异步，一车已付钱的订单全推一遍）
// 司机点发车或者定时任务把班次从1整到2的时候调
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

		// 这趟车已付钱(status=1)的订单都查出来
		var orders []model.Order
		if err := db.Where("trip_id = ? AND status = ? AND order_type = 1", trip.ID, model.OrderStatusPaid).Find(&orders).Error; err != nil {
			log.Printf("[WARN] wx_notify: 查询班次已支付订单失败 tripID=%d: %v\n", trip.ID, err)
			return
		}

		departureTimeStr := triptime.FormatDateTime(string(trip.TripDate), trip.DepartureTime)
		// 信号量+WaitGroup限一下并发，别一趟车人满为患时HTTP请求乌泱乌泱的
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

// NotifyTripArrival 班次到达通知（异步，到站的订单全推）
// 车到终点站(completeArrivedTrip)班次2→4的时候调
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

		// 这趟车已完成(status=2)的订单都查出来
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

// NotifyRefundSuccess 退款到账通知（异步）
// 退款回调 RefundNotify 把钱退成功的时候调
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

// truncateStr 字符串截断（微信订阅消息thing字段最多20字符，多了不让发）
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

