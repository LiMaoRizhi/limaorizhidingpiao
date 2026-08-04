package wx

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/idcard"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/triptime"
	"limaorizhi-server/internal/pkg/wxtoken"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// wxHTTPClient 微信API专用HTTP客户端（带超时，防止DoS）
var wxHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// 小程序用户端接口

type UserHandler struct {
	DB       *gorm.DB
	Verifier *idcard.CloudVerifier // 身份实名认证客户端
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	// 从配置初始化实名认证客户端
	v := idcard.NewCloudVerifier(
		config.AppConfig.IDCardVerify.AppCode,
		config.AppConfig.IDCardVerify.Endpoint,
		config.AppConfig.IDCardVerify.Path,
		config.AppConfig.IDCardVerify.Enabled,
		config.AppConfig.IDCardVerify.StrictMode,
		config.AppConfig.IDCardVerify.CacheTTL, // 认证结果缓存 TTL(秒)，0 表示使用默认 30 天。可通过 config.yaml 或环境变量 IDCARD_VERIFY_CACHE_TTL 调整
	)
	return &UserHandler{DB: db, Verifier: v}
}

// 辅助函数

// isTripDeparted 检查班次是否已过发车时间
// 安全策略：fail-closed，解析失败时视为已发车，拒绝操作
func isTripDeparted(tripDate, departureTime string) bool {
	t, err := triptime.Parse(tripDate, departureTime)
	if err != nil {
		return true // 解析失败视为已发车，拒绝操作（fail-closed）
	}
	return time.Now().After(t)
}

// isStationPassed 判断班次车辆是否已驶过用户上车站（多源融合，按优先级取第一个能用的）
// 优先级：1-2.GPS投影+手动标记取max(GPS优先，手动可前进覆盖) 3.每站到达时刻表(arrival_time) 4.里程比例推算 5.发车时间兜底
// 返回true=车已过该站不能上车，false=未过可上车
// fromStationID为用户上车站ID；routeStations可为nil（函数内部查db）
// 与 effectivePassedOrder 共享同一套判断逻辑，保证乘客展示与下单逻辑一致
func isStationPassed(db *gorm.DB, trip model.Trip, fromStationID uint, routeStations []model.RouteStation) bool {
	// 未传站点序列则内部查询（按站序升序）
	if len(routeStations) == 0 && db != nil {
		if err := db.Where("route_id = ?", trip.RouteID).Order("stop_order ASC").Find(&routeStations).Error; err != nil {
			// 查询失败时 fail-closed，视为已过站，拒绝下单
			return true
		}
	}

	// 定位用户上车站的 routeStation 及其站序（stop_order，从1开始）
	fromOrder := 0
	var fromRS *model.RouteStation
	for i := range routeStations {
		if routeStations[i].StationID == fromStationID {
			fromRS = &routeStations[i]
			fromOrder = routeStations[i].StopOrder
			break
		}
	}

	// 发车时间解析一下
	dep, err := triptime.Parse(string(trip.TripDate), trip.DepartureTime)
	if err != nil {
		return true // 解析失败视为已过（fail-closed，拒绝下单）
	}
	// 未到发车时间：肯定没过任何站，可下单
	if time.Now().Before(dep) {
		return false
	}

	// 优先级1-2合并：GPS投影 + 手动标记取最大值
	// GPS可用时优先使用GPS投影结果；GPS不可用时回退到手动标记
	// 手动标记可前进覆盖GPS（司机手动标记更靠后时以手动为准）
	// 统一函数同时用于乘客端进度条展示和下单逻辑判断
	if effective := service.EffectivePassedOrder(db, trip, routeStations); effective > 0 {
		return fromOrder <= effective
	}

	// 优先级3：每站到达时刻表（后台配的 arrival_time，精确到分钟）
	if fromRS != nil && fromRS.ArrivalTime != "" {
		if st, e := triptime.ParseWithOffset(string(trip.TripDate), fromRS.ArrivalTime, fromRS.ArrivalDayOffset); e == nil {
			return !time.Now().Before(st) // 当前时间 ≥ 到达时刻 = 已过
		}
	}

	// 优先级4：里程比例或站序比例推算（用累计里程或站序占比 × 起终时间自动算该站到达时间）
	if fromRS != nil && len(routeStations) > 0 {
		lastRS := routeStations[len(routeStations)-1]
		if arr, e := triptime.ParseWithOffset(string(trip.TripDate), trip.ArrivalTime, trip.ArrivalDayOffset); e == nil {
			var ratio float64
			if fromRS.DistanceKM > 0 && lastRS.DistanceKM > fromRS.DistanceKM {
				// 优先用里程比例（更精确，考虑了站间距离差异）
				ratio = fromRS.DistanceKM / lastRS.DistanceKM
			} else if lastRS.StopOrder > fromRS.StopOrder {
				// 里程未配置时降级为站序比例（等间距假设，优于直接判定全过）
				ratio = float64(fromRS.StopOrder) / float64(lastRS.StopOrder)
			}
			if ratio > 0 {
				estArrival := dep.Add(time.Duration(float64(arr.Sub(dep)) * ratio))
				return !time.Now().Before(estArrival)
			}
		}
	}

	// 优先级5：按发车→到达的时间进度线性推算（无需任何站点配置，兜底方案）
	// 避免原逻辑"发车后即视为所有站已过"的过于激进判断，导致途经站无法下单
	if len(routeStations) > 0 {
		lastRS := routeStations[len(routeStations)-1]
		if arr, e := triptime.ParseWithOffset(string(trip.TripDate), trip.ArrivalTime, trip.ArrivalDayOffset); e == nil {
			totalDuration := arr.Sub(dep)
			elapsed := time.Since(dep)
			if totalDuration > 0 && elapsed > 0 {
				progress := float64(elapsed) / float64(totalDuration)
				if progress >= 1.0 {
					return true // 已过到达时间，所有站都已过
				}
				// 按时间进度推算已驶过到第几站
				estPassedOrder := int(progress * float64(lastRS.StopOrder))
				return fromOrder <= estPassedOrder
			}
		}
	}

	// 终极兜底：到达时间也无法解析时，仅首站(及未找到的站点)视为已过，途经站允许下单
	// 原逻辑 return true 过于激进，阻断所有途经站下单
	return fromOrder <= 1
}

// getPhoneNumber 调用微信 getPhoneNumber 接口获取手机号
func getPhoneNumber(appid, secret, code string) string {
	// 1. 获取 access_token（带缓存）
	accessToken := wxtoken.GetAccessToken(appid, secret)
	if accessToken == "" {
		return ""
	}

	// 2. 使用 code 获取手机号（JSON安全编码，防止注入）
	phoneURL := fmt.Sprintf(
		"https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s",
		url.QueryEscape(accessToken),
	)
	reqBody, _ := json.Marshal(map[string]string{"code": code})
	phoneResp, err := wxHTTPClient.Post(phoneURL, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return ""
	}
	defer phoneResp.Body.Close()
	phoneBody, err := io.ReadAll(phoneResp.Body)
	if err != nil {
		return ""
	}
	var phoneResult struct {
		ErrCode int `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		PhoneInfo struct {
			PhoneNumber string `json:"phoneNumber"`
		} `json:"phone_info"`
	}
	if err := json.Unmarshal(phoneBody, &phoneResult); err != nil {
		return ""
	}
	if phoneResult.ErrCode != 0 {
		return ""
	}
	return phoneResult.PhoneInfo.PhoneNumber
}

// code2Session 调用微信 code2session 接口
func code2Session(appid, secret, code string) string {
	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(appid), url.QueryEscape(secret), url.QueryEscape(code),
	)
	resp, err := wxHTTPClient.Get(apiURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var result struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}
	if result.ErrCode != 0 {
		return ""
	}
	return result.OpenID
}

// 用户退出 & 优惠券

// Logout 用户退出登录（写入 token_invalid_before，使该用户早于此时间签发的Token失效）
func (h *UserHandler) Logout(c *gin.Context) {
	userID := c.GetUint("user_id")
	now := time.Now()
	if err := h.DB.Model(&model.User{}).Where("id = ?", userID).Update("token_invalid_before", &now).Error; err != nil {
		log.Printf("写入登出时间失败: %v\n", err)
	}
	response.OKMsg(c, "退出成功", nil)
}

// UserCoupons 获取用户可用优惠券列表
func (h *UserHandler) UserCoupons(c *gin.Context) {
	userID := c.GetUint("user_id")
	statusFilter := c.Query("status") // "" or "0" = unused, "1" = used, "2" = expired, "all" = all

	var userCoupons []model.UserCoupon
	now := time.Now()

	query := h.DB.Preload("Coupon").Where("user_id = ?", userID)
	if statusFilter == "" || statusFilter == "0" {
		// 未使用：status=0 且未过期
		query = query.Where("status = ? AND expired_at > ?", model.UserCouponStatusUnused, now)
	} else if statusFilter == "1" {
		// 已使用
		query = query.Where("status = ?", model.UserCouponStatusUsed)
	} else if statusFilter == "2" {
		// 已过期：status=0 但已过期，或 status=2
		query = query.Where("(status = ? AND expired_at <= ?) OR status = ?", model.UserCouponStatusUnused, now, model.UserCouponStatusExpired)
	}
	// statusFilter == "all" 时不加额外条件

	if err := query.Order("created_at DESC").Find(&userCoupons).Error; err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}
	// 构造返回数据
	type couponItem struct {
		ID            uint      `json:"id"`
		CouponID      uint      `json:"coupon_id"`
		Name          string    `json:"name"`
		Type          int8      `json:"type"`
		DiscountText  string    `json:"discount_text"`
		DiscountValue float64   `json:"discount_value"` // 折扣值：满减/固定金额(元)，折扣券(折数如8.5)
		MinSpend      float64   `json:"min_spend"`     // 满减门槛(元)，0为无门槛
		Desc          string    `json:"desc"`
		Status        int8      `json:"status"`
		IssuedAt      model.JSONTime `json:"issued_at"`
		ExpiredAt     model.JSONTime `json:"expired_at"`
	}
	var list []couponItem
	for _, uc := range userCoupons {
		if uc.Coupon == nil {
			continue
		}
		var discountText string
		switch uc.Coupon.Type {
		case 1:
			discountText = "¥" + fmt.Sprintf("%.0f", uc.Coupon.DiscountValue)
		case 2:
			discountText = fmt.Sprintf("%.1f折", uc.Coupon.DiscountValue)
		case 3:
			discountText = "¥" + fmt.Sprintf("%.0f", uc.Coupon.DiscountValue)
		}
		desc := "无门槛"
		if uc.Coupon.MinSpend > 0 {
			desc = fmt.Sprintf("满%.0f元可用", uc.Coupon.MinSpend)
		}
		list = append(list, couponItem{
			ID:            uc.ID,
			CouponID:      uc.CouponID,
			Name:          uc.Coupon.Name,
			Type:          uc.Coupon.Type,
			DiscountText:  discountText,
			DiscountValue: uc.Coupon.DiscountValue,
			MinSpend:      uc.Coupon.MinSpend,
			Desc:          desc,
			Status:        uc.Status,
			IssuedAt:      uc.IssuedAt,
			ExpiredAt:     uc.ExpiredAt,
		})
	}
	response.OK(c, list)
}
