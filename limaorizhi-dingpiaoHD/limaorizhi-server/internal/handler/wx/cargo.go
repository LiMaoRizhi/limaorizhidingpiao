package wx

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 小程序货物托运接口（仅运费预估+下单，列表/详情/支付/取消复用统一订单接口）

type CargoHandler struct{ DB *gorm.DB }

func NewCargoHandler(db *gorm.DB) *CargoHandler { return &CargoHandler{DB: db} }

// cargoConfig 从数据库读取托运配置
type cargoConfig struct {
	PricePerKM     float64 // 每公里运费(元)
	MinFee         float64 // 最低运费(元)
	FreeWeight     float64 // 免费重量(kg)
	ExtraWeightFee float64 // 超重费(元/kg)
	MaxWeight      float64 // 最大托运重量(kg)
}

func (h *CargoHandler) getCargoConfig() cargoConfig {
	cfg := cargoConfig{
		PricePerKM:     0.5,
		MinFee:         10,
		FreeWeight:     5,
		ExtraWeightFee: 3,
		MaxWeight:      50,
	}
	var configs []model.SystemConfig
	if err := h.DB.Where("config_key LIKE ?", "cargo_%").Find(&configs).Error; err != nil {
		log.Printf("[WARN] 查询托运配置失败，使用默认值: %v", err)
		return cfg
	}
	for _, c := range configs {
		if c.ConfigValue == "" {
			continue
		}
		val, err := strconv.ParseFloat(c.ConfigValue, 64)
		if err != nil {
			log.Printf("[WARN] 托运配置 %s 值非法(%q)，使用默认值: %v", c.ConfigKey, c.ConfigValue, err)
			continue
		}
		switch c.ConfigKey {
		case "cargo_price_per_km":
			cfg.PricePerKM = val
		case "cargo_min_fee":
			cfg.MinFee = val
		case "cargo_free_weight":
			cfg.FreeWeight = val
		case "cargo_extra_weight_fee":
			cfg.ExtraWeightFee = val
		case "cargo_max_weight":
			cfg.MaxWeight = val
		}
	}
	return cfg
}

// calcCargoFee 计算运费
// 基础运费 = max(最低运费, 距离km × 每公里运费)
// 免费重量以内不额外收费，超出部分按 超重费/kg 计算
// 总运费向上取整到元，使用整数分运算避免浮点精度导致多收1元
func calcCargoFee(distanceKM, weight float64, cfg cargoConfig) float64 {
	base := math.Max(cfg.MinFee, distanceKM*cfg.PricePerKM)
	extra := 0.0
	if weight > cfg.FreeWeight {
		extra = (weight - cfg.FreeWeight) * cfg.ExtraWeightFee
	}
	total := base + extra
	// 用整数分运算避免浮点精度导致 ceil 多收1元
	// 例如 0.1*100=10.000000000000002，math.Ceil 会错误地返回11
	totalFen := money.ToFen(total)
	if totalFen%100 != 0 {
		// 有零头，向上取整到元
		return money.FromFen((totalFen/100 + 1) * 100)
	}
	return money.FromFen(totalFen)
}

// getSegmentDistance 取线路中两站之间的累计里程差（收货站累计里程 − 发货站累计里程）。
// 用 stop_order 校验收货站在发货站之后，避免里程配置异常导致负数。
func getSegmentDistance(db *gorm.DB, routeID, fromID, toID uint) (float64, error) {
	var rss []model.RouteStation
	if err := db.Where("route_id = ?", routeID).Order("stop_order ASC").Find(&rss).Error; err != nil {
		return 0, fmt.Errorf("线路站点序列不存在")
	}
	var fromOrder, toOrder int = -1, -1
	var fromDist, toDist float64
	for _, rs := range rss {
		if rs.StationID == fromID {
			fromOrder = rs.StopOrder
			fromDist = rs.DistanceKM
		}
		if rs.StationID == toID {
			toOrder = rs.StopOrder
			toDist = rs.DistanceKM
		}
	}
	if fromOrder < 0 || toOrder < 0 {
		return 0, fmt.Errorf("发货站或收货站不在该线路中")
	}
	if fromOrder >= toOrder {
		return 0, fmt.Errorf("收货站必须在发货站之后")
	}
	return toDist - fromDist, nil
}
type cargoFeePreviewRequest struct {
	TripID         string  `json:"trip_id" binding:"required"`
	FromStationID  uint    `json:"from_station_id" binding:"required"` // 发货站
	ToStationID    uint    `json:"to_station_id" binding:"required"`   // 收货站
	Weight         float64 `json:"weight" binding:"required"`
}

func (h *CargoHandler) CargoFeePreview(c *gin.Context) {
	var req cargoFeePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	var trip model.Trip
	if err := h.DB.First(&trip, req.TripID).Error; err != nil {
		response.Fail(c, response.CodeTripNotFound)
		return
	}

	// 区间里程差（发货站→收货站的累计里程之差）
	distanceKM, err := getSegmentDistance(h.DB, trip.RouteID, req.FromStationID, req.ToStationID)
	if err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}

	cfg := h.getCargoConfig()
	// 重量范围校验
	if req.Weight <= 0 || req.Weight > cfg.MaxWeight {
		response.FailMsg(c, response.CodeParamError, fmt.Sprintf("重量需在0.1~%.0fkg之间", cfg.MaxWeight))
		return
	}
	fee := calcCargoFee(distanceKM, req.Weight, cfg)
	weightFee := 0.0
	if req.Weight > cfg.FreeWeight {
		weightFee = (req.Weight - cfg.FreeWeight) * cfg.ExtraWeightFee
	}
	response.OK(c, gin.H{
		"fee":         fee,
		"distance_km": distanceKM,
		"base_fee":    math.Max(cfg.MinFee, distanceKM*cfg.PricePerKM),
		"weight_fee":  weightFee,
	})
}

// CreateCargoOrder 创建托运订单（统一到 Order 表，order_type=2）
type createCargoRequest struct {
	TripID         string  `json:"trip_id" binding:"required"`
	FromStationID  uint    `json:"from_station_id" binding:"required"` // 发货站
	ToStationID    uint    `json:"to_station_id" binding:"required"`   // 收货站
	SenderName     string  `json:"sender_name" binding:"required"`
	SenderPhone    string  `json:"sender_phone" binding:"required"`
	ReceiverName   string  `json:"receiver_name" binding:"required"`
	ReceiverPhone  string  `json:"receiver_phone" binding:"required"`
	CargoType      string  `json:"cargo_type"`
	Weight         float64 `json:"weight" binding:"required"`
	Description    string  `json:"description"`
}

func (h *CargoHandler) CreateCargoOrder(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req createCargoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 限制未支付订单数量，防刷单（与车票下单一致）
	var pendingCount int64
	h.DB.Model(&model.Order{}).Where("user_id = ? AND status = ?", userID, model.OrderStatusPending).Count(&pendingCount)
	if pendingCount >= 5 {
		response.FailMsg(c, response.CodeServerError, "您有过多未支付订单，请先支付或取消后再下单")
		return
	}

	// 手机号格式校验
	if !isValidPhone(req.SenderPhone) {
		response.FailMsg(c, response.CodeParamError, "寄件人手机号格式不正确")
		return
	}
	if !isValidPhone(req.ReceiverPhone) {
		response.FailMsg(c, response.CodeParamError, "收件人手机号格式不正确")
		return
	}

	cfg := h.getCargoConfig()
	if req.Weight <= 0 || req.Weight > cfg.MaxWeight {
		response.FailMsg(c, response.CodeParamError, fmt.Sprintf("重量需在0.1~%.0fkg之间", cfg.MaxWeight))
		return
	}
	if req.CargoType == "" {
		req.CargoType = "日用品"
	}

	var order model.Order
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 查询班次（加行锁）
		var trip model.Trip
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&trip, req.TripID).Error; err != nil {
			return errTripNotFound
		}
		// 允许已发车(status=2)的班次下单：途经站发货人在车到站前仍可托运
		if trip.Status != model.TripStatusSale && trip.Status != model.TripStatusDepart {
			return errTripUnavailable
		}
		// 检查车是否已驶过用户上车站（多源融合判断，发车后未到站可下单）
		if isStationPassed(tx, trip, req.FromStationID, nil) {
			return errTripDeparted
		}

		// 2. 查询线路获取区间里程差（发货站→收货站的累计里程之差）
		distanceKM, err := getSegmentDistance(tx, trip.RouteID, req.FromStationID, req.ToStationID)
		if err != nil {
			return err
		}

		// 2.1 查询发货站/收货站名称（冗余存储到订单，删除线路/站点后仍可显示）
		var fromStation, toStation model.Station
		tx.Select("name").Where("id = ?", req.FromStationID).First(&fromStation)
		tx.Select("name").Where("id = ?", req.ToStationID).First(&toStation)

		// 2.2 事务内再次校验未支付订单数量（加锁防并发绕过）
		var pendingCountTx int64
		if err := tx.Model(&model.Order{}).Where("user_id = ? AND status = ?", userID, model.OrderStatusPending).Count(&pendingCountTx).Error; err != nil {
			return fmt.Errorf("查询未支付订单数量失败")
		}
		if pendingCountTx >= 5 {
			return fmt.Errorf("您有过多未支付订单，请先支付或取消后再下单")
		}

		// 3. 计算运费
		fee := calcCargoFee(distanceKM, req.Weight, cfg)

		// 4. 生成订单号（加密随机后缀，防止枚举攻击）
		randomBytes := make([]byte, 4)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("生成订单号失败")
		}
		orderNo := fmt.Sprintf("HY%s%s", time.Now().Format("20060102"), hex.EncodeToString(randomBytes))

		// JSONDate.Scan 已确保格式为 "2006-01-02"，无需再手动剥离 RFC3339
		order = model.Order{
			OrderNo:        orderNo,
			OrderType:      2, // 托运
			UserID:         userID,
			TripID:         trip.ID,
			RouteID:        trip.RouteID,
			FromStationID:   req.FromStationID,
			FromStationName: fromStation.Name,
			ToStationID:     req.ToStationID,
			ToStationName:   toStation.Name,
			TripDate:        trip.TripDate,
			DepartureTime:  trip.DepartureTime,
			PassengerCount: 0,                  // 托运无乘客
			TotalPrice:     fee,                // 运费存入 total_price
			Status:         model.OrderStatusPending, // 待支付
			SenderName:     req.SenderName,
			SenderPhone:    req.SenderPhone,
			ReceiverName:   req.ReceiverName,
			ReceiverPhone:  req.ReceiverPhone,
			CargoType:      req.CargoType,
			Weight:         req.Weight,
			Description:    req.Description,
		}
		return tx.Create(&order).Error
	})

	if err != nil {
		if errors.Is(err, errTripUnavailable) || errors.Is(err, errTripDeparted) {
			response.Fail(c, response.CodeTripUnavailable)
			return
		}
		log.Printf("[ERROR] 托运下单失败 err:%v\n", err)
		response.FailMsg(c, response.CodeServerError, "托运下单失败，请稍后重试")
		return
	}

	// 重新加载完整信息
	h.DB.Preload("Trip.Route.FromStation").Preload("Trip.Route.ToStation").
		Preload("FromStation").Preload("ToStation").First(&order, order.ID)

	// 统一脱敏（与车票订单掩码一致，防止新增敏感字段遗漏）
	order.Mask()

	response.OKMsg(c, "下单成功", order)
}
