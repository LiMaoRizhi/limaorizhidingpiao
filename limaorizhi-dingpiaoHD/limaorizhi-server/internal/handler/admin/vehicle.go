// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package admin

import (
	"fmt"
	"strconv"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/sanitize"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	v := model.Vehicle{
		PlateNo:     req.PlateNo,
		VehicleType: req.VehicleType,
		SeatCount:   req.SeatCount,
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
	if err := h.DB.Model(&v).Updates(map[string]interface{}{
		"plate_no": req.PlateNo, "vehicle_type": req.VehicleType,
		"seat_count": req.SeatCount, "status": req.Status,
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
		// 强制删除：取消活跃班次，退款已支付订单
		var activeTrips []model.Trip
		h.DB.Where("vehicle_id = ? AND status IN (1, 2)", id).Find(&activeTrips)
		for _, trip := range activeTrips {
			// 取消该班次的待支付订单
			h.DB.Model(&model.Order{}).Where("trip_id = ? AND status = 0", trip.ID).Update("status", 4)
			// 已支付订单创建退款记录（事务外调微信退款API）
			var paidOrders []model.Order
			h.DB.Where("trip_id = ? AND status = 1", trip.ID).Find(&paidOrders)
			for _, o := range paidOrders {
				refundRecord := model.Refund{
					OrderID: o.ID, RefundNo: sanitize.GenerateRefundNo(o.ID),
					Amount: o.TotalPrice, Reason: "车辆删除-自动退款", Status: 0, PreStatus: 1,
				}
				h.DB.Create(&refundRecord)
				// 标记订单为已取消
				h.DB.Model(&o).Update("status", 4)
			}
			// 更新班次状态为已取消
			h.DB.Model(&trip).Update("status", 3)
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
