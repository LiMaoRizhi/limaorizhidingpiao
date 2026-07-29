// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/idcard"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/triptime"
	"limaorizhi-server/internal/pkg/verifytoken"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

// 二维码生成

type QrcodeHandler struct{ DB *gorm.DB }

func NewQrcodeHandler(db *gorm.DB) *QrcodeHandler { return &QrcodeHandler{DB: db} }

// Generate 生成订单二维码（返回PNG图片，需鉴权且校验订单归属）
// 二维码内容 = 带HMAC签名的核销凭证：LIMO|{orderNo}|{expireTs}|{sig}
// 防伪：服务端密钥签名；防重放：班次到达终点后24小时过期（跨天班次覆盖全程）
func (h *QrcodeHandler) Generate(c *gin.Context) {
	userToken := c.GetUint("user_id")
	orderNo := c.Param("order_no")
	if orderNo == "" {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 校验订单号格式（DP + 8位日期 + 8位hex），防止注入
	if err := verifytoken.ValidateOrderNo(orderNo); err != nil {
		response.FailMsg(c, response.CodeParamError, "订单号格式非法")
		return
	}

	// 验证订单存在且属于当前用户
	var order model.Order
	if err := h.DB.Where("order_no = ? AND user_id = ?", orderNo, userToken).First(&order).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}

	// 校验订单状态：只有待出行(1)的订单才能生成核销凭证
	// 防止已取消/已退款/已完成的订单仍持有有效核销凭证
	if order.Status != model.OrderStatusPaid {
		var statusText string
		switch order.Status {
		case model.OrderStatusPending:
			statusText = "未支付"
		case model.OrderStatusCompleted:
			statusText = "已完成"
		case model.OrderStatusRefunded:
			statusText = "已退款"
		case model.OrderStatusCancelled:
			statusText = "已取消"
		default:
			statusText = "异常"
		}
		response.FailMsg(c, response.CodeOrderStatusErr, "订单"+statusText+"，无法生成核销码")
		return
	}

	// 过期时间=到达终点时间+24h，覆盖沿途各站上车时段（含跨天）
	var trip model.Trip
	if err := h.DB.Select("trip_date, arrival_time, arrival_day_offset").First(&trip, order.TripID).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "班次信息异常，无法生成核销凭证")
		return
	}
	arrivalTime, parseErr := triptime.ParseWithOffset(string(trip.TripDate), trip.ArrivalTime, trip.ArrivalDayOffset)
	if parseErr != nil {
		// fail-closed：到达时间解析失败时拒绝生成二维码，防止异常数据导致凭证有效期错误
		response.FailMsg(c, response.CodeServerError, "班次到达时间格式异常，无法生成核销凭证")
		return
	}
	expireTs := arrivalTime.Add(24 * time.Hour).Unix()

	// 生成带签名的核销凭证
	token, err := verifytoken.Generate(orderNo, expireTs)
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "生成核销凭证失败")
		return
	}

	// 纠错级别用 High（30%冗余），防部分遮挡/磨损导致的伪造
	png, err := qrcode.Encode(token, qrcode.Highest, 256)
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}

	c.Data(200, "image/png", png)
}

// OrderInfo 查询订单信息（供小程序展示，需鉴权且校验订单归属）
func (h *QrcodeHandler) OrderInfo(c *gin.Context) {
	userToken := c.GetUint("user_id")
	orderNo := c.Param("order_no")
	// 校验订单号格式，与 Generate 接口保持一致
	if err := verifytoken.ValidateOrderNo(orderNo); err != nil {
		response.FailMsg(c, response.CodeParamError, "订单号格式非法")
		return
	}
	var order model.Order
	if err := h.DB.Preload("Trip.Route.FromStation").
		Preload("Trip.Route.ToStation").
		Preload("Trip.Vehicle").
		Where("order_no = ? AND user_id = ?", orderNo, userToken).
		First(&order).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}

	var passengers []model.OrderPassenger
	h.DB.Where("order_id = ?", order.ID).Find(&passengers)

	// 安全：公开接口必须脱敏，隐藏乘客身份证号和手机号中间部分
	for i := range passengers {
		passengers[i].IDCardNo = idcard.MaskIDCard(passengers[i].IDCardNo)
		passengers[i].Phone = idcard.MaskPhone(passengers[i].Phone)
	}

	// 安全：脱敏订单中的联系电话/发货人电话/收货人电话
	order.ContactPhone = idcard.MaskPhone(order.ContactPhone)
	order.SenderPhone = idcard.MaskPhone(order.SenderPhone)
	order.ReceiverPhone = idcard.MaskPhone(order.ReceiverPhone)

	response.OK(c, gin.H{
		"order":      order,
		"passengers": passengers,
	})
}


