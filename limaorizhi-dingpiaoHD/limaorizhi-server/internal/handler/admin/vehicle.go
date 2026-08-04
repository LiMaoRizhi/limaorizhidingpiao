package admin

import (
	"fmt"
	"log"
	"strconv"

	wxpay "limaorizhi-server/internal/handler/wx"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/sanitize"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)
// 车辆管理

type VehicleHandler struct {
	DB *gorm.DB
}

func NewVehicleHandler(db *gorm.DB) *VehicleHandler {
	return &VehicleHandler{DB: db}
}

func (h *VehicleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.DB.Model(&model.Vehicle{})
	if plateNo := c.Query("plate_no"); plateNo != "" {
		query = query.Where("plate_no LIKE ?", "%"+sanitize.EscapeLikePattern(plateNo)+"%")
	}
	query.Count(&total)

	var list []model.Vehicle
	query.Order("id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	response.Page(c, list, total, page, pageSize)
}

func (h *VehicleHandler) All(c *gin.Context) {
	var list []model.Vehicle
	h.DB.Where("status = 1").Order("id ASC").Find(&list)
	response.OK(c, list)
}

// createVehicleRequest 车辆创建请求（DTO白名单）
type createVehicleRequest struct {
	PlateNo     string `json:"plate_no" binding:"required"`
	VehicleType string `json:"vehicle_type"`
	SeatCount   int    `json:"seat_count" binding:"required"`
	SeatLayout  string `json:"seat_layout"` // 座位布局JSON，空则自动生成默认布局
	Status      int8   `json:"status"`
}

func (h *VehicleHandler) Create(c *gin.Context) {
	var req createVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.PlateNo == "" {
		response.FailMsg(c, response.CodeParamError, "车牌号不能为空")
		return
	}
	if req.SeatCount <= 0 {
		response.FailMsg(c, response.CodeParamError, "座位数必须大于0")
		return
	}
	// 校验座位布局与座位数一致性（防止恶意/错误的JSON存入数据库）
	if req.SeatLayout != "" {
		if err := service.ValidateSeatLayout(req.SeatLayout, req.SeatCount); err != nil {
			response.FailMsg(c, response.CodeParamError, err.Error())
			return
		}
	}
	v := model.Vehicle{
		PlateNo:     req.PlateNo,
		VehicleType: req.VehicleType,
		SeatCount:   req.SeatCount,
		SeatLayout:  req.SeatLayout,
		Status:      1, // 默认可用
	}
	if req.Status == 0 || req.Status == 1 {
		v.Status = req.Status
	}
	if err := h.DB.Create(&v).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "创建失败")
		return
	}
	WriteLog(c, h.DB, "车辆", "新增", v.PlateNo, fmt.Sprintf("车辆ID:%d 车牌:%s", v.ID, v.PlateNo))
	response.OK(c, v)
}

func (h *VehicleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var v model.Vehicle
	if err := h.DB.First(&v, id).Error; err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	var req struct {
		PlateNo     string `json:"plate_no"`
		VehicleType string `json:"vehicle_type"`
		SeatCount   int    `json:"seat_count"`
		SeatLayout  string `json:"seat_layout"`
		Status      int8   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if req.PlateNo == "" {
		response.FailMsg(c, response.CodeParamError, "车牌号不能为空")
		return
	}
	if req.SeatCount <= 0 {
		response.FailMsg(c, response.CodeParamError, "座位数必须大于0")
		return
	}
	// 座位布局和座位数得对得上
	if req.SeatLayout != "" {
		if err := service.ValidateSeatLayout(req.SeatLayout, req.SeatCount); err != nil {
			response.FailMsg(c, response.CodeParamError, err.Error())
			return
		}
	}
	if err := h.DB.Model(&v).Updates(map[string]interface{}{
		"plate_no": req.PlateNo, "vehicle_type": req.VehicleType,
		"seat_count": req.SeatCount, "seat_layout": req.SeatLayout, "status": req.Status,
	}).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	h.DB.First(&v, id)
	WriteLog(c, h.DB, "车辆", "编辑", v.PlateNo, fmt.Sprintf("车辆ID:%s 车牌:%s", id, v.PlateNo))
	response.OK(c, v)
}

func (h *VehicleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	force := c.Query("force") == "1"

	// 区分活跃班次和历史班次
	// 活跃班次(status=1可售 / 2已发车)：阻止删除，需先处理
	var activeTripCount int64
	h.DB.Model(&model.Trip{}).Where("vehicle_id = ? AND status IN (1, 2)", id).Count(&activeTripCount)

	if activeTripCount > 0 && !force {
		response.FailWithData(c, response.CodeServerError, "该车辆被活跃班次引用，需先处理或强制删除", map[string]int64{
			"active_trip_count": activeTripCount,
		})
		return
	}

	if activeTripCount > 0 && force {
		// 强制删除：取消活跃班次，退款已支付订单（与班次删除逻辑一致，事务内处理 + 事务外调微信退款API）
		var activeTrips []model.Trip
		h.DB.Where("vehicle_id = ? AND status IN (1, 2)", id).Find(&activeTrips)

		type refundItem struct {
			order  model.Order
			refund model.Refund
		}
		var refundItems []refundItem
		feeRate := readRefundFeeRate(h.DB)

		txErr := h.DB.Transaction(func(tx *gorm.DB) error {
			for _, trip := range activeTrips {
				// 行锁班次（与下单锁序一致，阻断并发新订单）
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model.Trip{}, trip.ID).Error; err != nil {
					return fmt.Errorf("班次ID:%d 不存在或已被删除", trip.ID)
				}
				// 待支付订单取消 + 归还优惠券
				var pendingOrders []model.Order
				if err := tx.Where("trip_id = ? AND status = ?", trip.ID, model.OrderStatusPending).Find(&pendingOrders).Error; err != nil {
					return err
				}
				for _, po := range pendingOrders {
					if err := tx.Model(&model.Order{}).Where("id = ? AND status = ?", po.ID, model.OrderStatusPending).Update("status", model.OrderStatusCancelled).Error; err != nil {
						return err
					}
					if err := service.ReturnOrderCoupon(tx, po.ID); err != nil {
						return err
					}
				}
				// 已支付订单 → 已退款(3) + 创建退款记录（事务内当前读，与班次删除一致）
				if err := tx.Model(&model.Order{}).
					Where("trip_id = ? AND order_type = 1 AND status = ?", trip.ID, model.OrderStatusPaid).
					Update("status", model.OrderStatusRefunded).Error; err != nil {
					return fmt.Errorf("退款状态更新失败: %v", err)
				}
				// 托运已支付订单 → 已取消(4)
				if err := tx.Model(&model.Order{}).
					Where("trip_id = ? AND order_type = 2 AND status = ?", trip.ID, model.OrderStatusPaid).
					Update("status", model.OrderStatusCancelled).Error; err != nil {
					return fmt.Errorf("托运订单状态更新失败: %v", err)
				}
				var refundedOrders []model.Order
				if err := tx.Where("trip_id = ? AND order_type = 1 AND status = ?", trip.ID, model.OrderStatusRefunded).Find(&refundedOrders).Error; err != nil {
					return err
				}
				for _, o := range refundedOrders {
					// 检查是否已存在退款记录（防止重复创建）
					var existingRefund model.Refund
					if err := tx.Where("order_id = ? AND status IN (?, ?)", o.ID, model.RefundStatusProcessing, model.RefundStatusSuccess).First(&existingRefund).Error; err == nil {
						continue
					}
					// 归还已支付订单的优惠券（用户未乘车，券应退回）
					if err := service.ReturnOrderCoupon(tx, o.ID); err != nil {
						return err
					}
					refundRecord := model.Refund{
						OrderID: o.ID, RefundNo: sanitize.GenerateRefundNo(o.ID),
						Amount: money.CalcRefundAmount(o.TotalPrice, feeRate), Reason: "车辆删除-自动退款", Status: model.RefundStatusProcessing, PreStatus: model.OrderStatusPaid,
					}
					if err := tx.Create(&refundRecord).Error; err != nil {
						return fmt.Errorf("创建退款记录失败: %v", err)
					}
					refundItems = append(refundItems, refundItem{order: o, refund: refundRecord})
				}
				// 班次标成已取消
				if err := tx.Model(&model.Trip{}).Where("id = ? AND status IN (1, 2)", trip.ID).Update("status", model.TripStatusCancel).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			response.FailMsg(c, response.CodeServerError, "强制删除失败: "+txErr.Error())
			return
		}
		// 事务外：调用微信退款API（退款失败时回滚订单状态，与班次删除逻辑一致）
		for _, item := range refundItems {
			if err := wxpay.InitiateWxRefund(h.DB, item.order, item.refund); err != nil {
				log.Printf("[ERROR] 删除车辆退款失败 订单号:%s err:%v\n", item.order.OrderNo, err)
				service.RollbackRefundFailure(h.DB, item.order, item.refund, model.OrderStatusPaid)
			}
		}
	}

	// 断开历史班次的车辆引用（车辆删除后班次记录保留，订单历史不受影响）
	// 订单已冗余存储 from_station_name/to_station_name/trip_date/departure_time，不受影响
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 历史班次（status=0/3/4）的 vehicle_id 置0，保留班次记录
		if err := tx.Model(&model.Trip{}).Where("vehicle_id = ?", id).Update("vehicle_id", 0).Error; err != nil {
			return err
		}
		// 删除车辆
		if err := tx.Delete(&model.Vehicle{}, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}

	note := "删除车辆ID:" + id
	if activeTripCount > 0 {
		note = fmt.Sprintf("删除车辆ID:%s 强制删除，取消活跃班次%d，历史班次保留（vehicle_id置0）", id, activeTripCount)
	}
	WriteLog(c, h.DB, "车辆", "删除", id, note)
	response.OKMsg(c, "删除成功", nil)
}
