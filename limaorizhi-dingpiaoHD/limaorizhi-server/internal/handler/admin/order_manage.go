package admin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"strconv"
	"time"

	wxpay "limaorizhi-server/internal/handler/wx"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/sanitize"
	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 订单管理

type OrderHandler struct{ DB *gorm.DB }

func NewOrderHandler(db *gorm.DB) *OrderHandler { return &OrderHandler{DB: db} }

// buildOrderQuery 构建订单查询的公共方法，List 和 Export 共用
func (h *OrderHandler) buildOrderQuery(c *gin.Context) *gorm.DB {
	query := h.DB.Model(&model.Order{})
	if orderNo := c.Query("order_no"); orderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+sanitize.EscapeLikePattern(orderNo)+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if phone := c.Query("contact_phone"); phone != "" {
		query = query.Where("contact_phone LIKE ?", "%"+sanitize.EscapeLikePattern(phone)+"%")
	}
	if orderType := c.Query("order_type"); orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("CAST(created_at AS DATE) >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("CAST(created_at AS DATE) <= ?", endDate)
	}
	return query
}

func (h *OrderHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	var total int64
	query := h.buildOrderQuery(c)
	query.Count(&total)

	var list []model.Order
	query.Preload("FromStation").Preload("ToStation").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	model.MaskOrders(list)
	response.Page(c, list, total, page, pageSize)
}

// Export 导出订单为CSV（F3：管理端订单导出功能）
func (h *OrderHandler) Export(c *gin.Context) {
	// 复用公共查询逻辑，最多导出10000条
	query := h.buildOrderQuery(c)

	var list []model.Order
	query.Preload("FromStation").Preload("ToStation").
		Order("created_at DESC").Limit(10000).Find(&list)
	model.MaskOrders(list) // 脱敏手机号

	// 生成CSV（带BOM头防止Excel乱码）
	buf := &bytes.Buffer{}
	buf.Write([]byte("\xEF\xBB\xBF"))
	w := csv.NewWriter(buf)
	w.UseCRLF = true
	w.Write([]string{"订单号", "类型", "线路", "日期", "时间", "人数/重量", "金额", "联系人", "联系电话", "状态", "创建时间"})

	statusText := func(s, t int8) string {
		if t == 2 {
			return map[int8]string{model.OrderStatusPending: "待支付", model.OrderStatusPaid: "待运输", model.OrderStatusCompleted: "运输中", model.OrderStatusRefunded: "已到达", model.OrderStatusCancelled: "已取消", 5: "已取件"}[s]
		}
		return map[int8]string{model.OrderStatusPending: "待支付", model.OrderStatusPaid: "待出行", model.OrderStatusCompleted: "已完成", model.OrderStatusRefunded: "已退款", model.OrderStatusCancelled: "已取消"}[s]
	}

	for _, o := range list {
		// 优先使用冗余站名（删除线路/站点后仍可显示），回退到Preload的关联站名
		fromName := o.FromStationName
		if fromName == "" && o.FromStation != nil {
			fromName = o.FromStation.Name
		}
		toName := o.ToStationName
		if toName == "" && o.ToStation != nil {
			toName = o.ToStation.Name
		}
		route := fromName + " → " + toName
		count := strconv.Itoa(o.PassengerCount) + "人"
		if o.OrderType == 2 {
			count = strconv.FormatFloat(o.Weight, 'f', 1, 64) + "kg"
		}
		contact, phone := o.ContactName, o.ContactPhone
		if o.OrderType == 2 {
			contact, phone = o.SenderName, o.SenderPhone
		}
		w.Write([]string{
			o.OrderNo,
			map[int8]string{1: "车票", 2: "托运"}[o.OrderType],
			"\t" + route, "\t" + string(o.TripDate), "\t" + o.DepartureTime, count,
			strconv.FormatFloat(o.TotalPrice, 'f', 2, 64),
			contact, phone, statusText(o.Status, o.OrderType),
			"\t" + o.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()

	c.Header("Content-Disposition", "attachment; filename=orders_"+time.Now().Format("20060102_150405")+".csv")
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
}

func (h *OrderHandler) Detail(c *gin.Context) {
	id := c.Param("id")
	var order model.Order
	if err := h.DB.Preload("FromStation").Preload("ToStation").
		Preload("Trip").Preload("User").
		First(&order, id).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}
	var passengers []model.OrderPassenger
	h.DB.Where("order_id = ?", id).Find(&passengers)
	model.MaskPassengers(passengers)
	order.Mask()
	if order.User != nil {
		order.User.Mask()
	}
	// 补查支付流水：支付回调可能没送达，订单状态是"假的"（待支付/已取消）但微信钱已扣。
	// 支付流水是微信真实扣款的铁证，管理后台靠它看出"实际已付款"，退款也要用它。
	var payments []model.Payment
	h.DB.Where("order_id = ?", id).Order("created_at ASC").Find(&payments)
	response.OK(c, gin.H{"order": order, "passengers": passengers, "payments": payments})
}

type updateOrderStatusRequest struct {
	Status int8 `json:"status" binding:"required"`
}

func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req updateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	// 校验状态值合法性（5=托运已取件，仅托运订单使用）
	validStatus := map[int8]bool{model.OrderStatusPending: true, model.OrderStatusPaid: true, model.OrderStatusCompleted: true, model.OrderStatusRefunded: true, model.OrderStatusCancelled: true, 5: true}
	if !validStatus[req.Status] {
		response.FailMsg(c, response.CodeParamError, "非法的订单状态值")
		return
	}
	var order model.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}
	// 按订单类型区分状态转换规则
	// 车票订单：0待支付 1待出行 2已完成 3已退款 4已取消
	// 托运订单：0待支付 1待运输 2运输中 3已到达 4已取消 5已取件
	var allowedTransitions map[int8]map[int8]bool
	if order.OrderType == 2 {
		// 托运订单转换规则
		allowedTransitions = map[int8]map[int8]bool{
			model.OrderStatusPending:   {model.OrderStatusPending: true, model.OrderStatusCancelled: true},                // 待支付 → 待支付/已取消
			model.OrderStatusPaid:      {model.OrderStatusPaid: true, model.OrderStatusCompleted: true},       // 待运输 → 待运输/运输中（退款走专用Refund接口，禁止直接改已取消）
			model.OrderStatusCompleted: {model.OrderStatusCompleted: true, model.OrderStatusRefunded: true},               // 运输中 → 运输中/已到达
			model.OrderStatusRefunded:  {model.OrderStatusRefunded: true, 5: true},               // 已到达 → 已到达/已取件
			model.OrderStatusCancelled: {model.OrderStatusCancelled: true},                        // 已取消 → 已取消（终态）
			5:                          {5: true},                        // 已取件 → 已取件（终态）
		}
	} else {
		// 车票订单转换规则（退款走专用Refund接口）
		allowedTransitions = map[int8]map[int8]bool{
			model.OrderStatusPending:   {model.OrderStatusPending: true, model.OrderStatusCancelled: true},                // 待支付 → 待支付/已取消
			model.OrderStatusPaid:      {model.OrderStatusPaid: true, model.OrderStatusCompleted: true},        // 待出行 → 待出行/已完成（退款走Refund接口，禁止直接改已取消）
			model.OrderStatusCompleted: {model.OrderStatusCompleted: true},                          // 已完成 → 已完成（退款走Refund接口）
			model.OrderStatusRefunded:  {model.OrderStatusRefunded: true},                          // 已退款 → 已退款（终态）
			model.OrderStatusCancelled: {model.OrderStatusCancelled: true},                          // 已取消 → 已取消（终态）
		}
	}
	if allowed, ok := allowedTransitions[order.Status]; ok {
		if !allowed[req.Status] {
			// 已支付订单不可直接改为已取消，需走退款流程退还用户资金
			if order.Status == model.OrderStatusPaid && req.Status == model.OrderStatusCancelled {
				response.FailMsg(c, response.CodeOrderStatusErr, "已支付订单不可直接改为已取消，请使用退款功能退还用户资金")
				return
			}
			response.FailMsg(c, response.CodeOrderStatusErr, fmt.Sprintf("订单状态不允许从 %d 转换为 %d", order.Status, req.Status))
			return
		}
	}
	// 状态转换使用事务保证一致性
	oldStatus := order.Status
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 行锁查询订单
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return fmt.Errorf("订单不存在")
		}
		// 再次校验状态转换合法性（行锁内，防并发）
		if allowed, ok := allowedTransitions[order.Status]; ok {
			if !allowed[req.Status] {
				return fmt.Errorf("订单状态不允许从 %d 转换为 %d", order.Status, req.Status)
			}
		}
		if err := tx.Model(&order).Update("status", req.Status).Error; err != nil {
			return err
		}
		// 订单被取消（待支付→已取消）时用户没坐车，优惠券必须退回，否则用户白丢券
		if order.Status == model.OrderStatusPending && req.Status == model.OrderStatusCancelled {
			if err := service.ReturnOrderCoupon(tx, order.ID); err != nil {
				return err
			}
		}
		// 区间复用模型：座位容量按区间实时计算，状态转为已取消/已退款后区间容量自然恢复，无需回补 available_seats
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeOrderStatusErr, err.Error())
		return
	}
	WriteLog(c, h.DB, "订单", "状态更新", order.OrderNo, fmt.Sprintf("订单ID:%s 订单号:%s 状态:%d→%d", id, order.OrderNo, oldStatus, req.Status))
	response.OKMsg(c, "状态更新成功", order)
}

type refundRequest struct {
	Reason string `json:"reason"`
}

func (h *OrderHandler) Refund(c *gin.Context) {
	id := c.Param("id")
	var req refundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// reason为可选字段，客户端可能不传body，降级为空reason继续
		req.Reason = ""
	}

	// 读取退票手续费率配置（与用户端退票逻辑一致）
	var feeRateCfg model.SystemConfig
	feeRate := 0.0
	if err := h.DB.Where("config_key = ?", "refund_fee_rate").First(&feeRateCfg).Error; err == nil {
		if v, err := strconv.ParseFloat(feeRateCfg.ConfigValue, 64); err == nil && v >= 0 {
			feeRate = v
		}
	}
	// 手续费率封顶100%，防止配置异常导致退款金额为负数
	if feeRate > 100 {
		feeRate = 100
	}

	// 使用行锁防止并发双退款
	var refundedOrder model.Order
	var refundRecord model.Refund
	var refundFee, refundAmount float64
	var preRefundStatus int8 // 退款前订单状态（1=待出行 / 2=已完成），退款失败回滚时使用

	// 退款前先查单对账：支付回调没送达时订单可能显示"待支付/已取消"，
	// 但微信钱已扣，直接退会被"订单状态不允许退款"挡掉（钱就卡死了）。
	// 先救回真实状态：已支付就确认、已取消就登记自动退款。
	if err := h.DB.First(&refundedOrder, id).Error; err != nil {
		response.Fail(c, response.CodeOrderNotFound)
		return
	}
	if handled, rerr := wxpay.ReconcileOrderPaidState(h.DB, refundedOrder); rerr != nil {
		response.FailMsg(c, response.CodeRefundFail, "退款失败："+rerr.Error())
		return
	} else if handled {
		response.OKMsg(c, "订单已取消，系统将自动退款", nil)
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 加行锁查询订单
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&refundedOrder, id).Error; err != nil {
			return fmt.Errorf("订单不存在")
		}
		if refundedOrder.Status != model.OrderStatusPaid && refundedOrder.Status != model.OrderStatusCompleted {
			return fmt.Errorf("订单状态不允许退款")
		}
		preRefundStatus = refundedOrder.Status

		// 用整数分计算退款手续费，与用户端退票共用 money.CalcRefundAmount（统一四舍五入），保证对账一致
		refundAmount = money.CalcRefundAmount(refundedOrder.TotalPrice, feeRate)
		refundFee = money.Sub(refundedOrder.TotalPrice, refundAmount)

		// 2. 更新订单状态（车票→3已退款，托运→4已取消）
		var refundStatus int8 = model.OrderStatusRefunded // 车票退款后状态
		if refundedOrder.OrderType == 2 {
			refundStatus = model.OrderStatusCancelled // 托运退款后状态（已取消）
		}
		if err := tx.Model(&refundedOrder).Update("status", refundStatus).Error; err != nil {
			return err
		}
		// 3. 区间复用模型：座位容量按区间实时计算，退款后区间容量自然恢复，无需回补 available_seats
		// 4. 创建/复用退款记录（复用失败退款单号，防双重退款）
		// 注意：err 由同语句 `err := h.DB.Transaction(...)` 声明，作用域不覆盖闭包内，故用局部变量接收
		newRefund, _, prepareErr := service.PrepareRefundRecord(tx, refundedOrder.ID, refundAmount, req.Reason, preRefundStatus, 0)
		if prepareErr != nil {
			return prepareErr
		}
		refundRecord = newRefund
		// 5. 未出行（待出行）订单退款后归还优惠券；已完成订单用户已乘车，券不归还
		if preRefundStatus == model.OrderStatusPaid {
			return service.ReturnOrderCoupon(tx, refundedOrder.ID)
		}
		return nil
	})
	if err != nil {
		response.FailMsg(c, response.CodeRefundFail, "退款失败")
		return
	}
	WriteLog(c, h.DB, "订单", "退款", refundedOrder.OrderNo, fmt.Sprintf("订单ID:%s 订单号:%s 退款金额:%.2f 手续费:%.2f", id, refundedOrder.OrderNo, refundAmount, refundFee))

	// 0元订单退款（优惠券全额抵扣那种）：本来就没收钱，微信也不让退0元，
	// 直接标记退款成功就行，别真去调微信退款接口给自己找麻烦
	if refundRecord.Amount <= 0 {
		if err := service.MarkRefundSuccess(h.DB, refundRecord); err != nil {
			log.Printf("[ERROR] 管理端退款：MarkRefundSuccess 失败 订单号:%s err:%v\n", refundedOrder.OrderNo, err)
			service.RollbackRefundFailure(h.DB, refundedOrder, refundRecord, preRefundStatus)
			response.FailMsg(c, response.CodeRefundFail, "退款失败：标记退款成功时出错")
			return
		}
	} else if !wxpay.IsWxRefundConfigured() {
		// 未配置证书：直接标记退款成功
		if err := service.MarkRefundSuccess(h.DB, refundRecord); err != nil {
			log.Printf("[ERROR] 管理端退款：MarkRefundSuccess 失败 订单号:%s err:%v\n", refundedOrder.OrderNo, err)
			service.RollbackRefundFailure(h.DB, refundedOrder, refundRecord, preRefundStatus)
			response.FailMsg(c, response.CodeRefundFail, "退款失败：标记退款成功时出错")
			return
		}
	} else {
		// 查支付记录拿微信交易号
		var payment model.Payment
		if err := h.DB.Where("order_id = ? AND status = ?", refundedOrder.ID, model.PaymentStatusSuccess).First(&payment).Error; err != nil {
			log.Printf("[ERROR] 管理端退款：查找支付记录失败 订单ID:%d err:%v\n", refundedOrder.ID, err)
			// 找不到支付记录时回滚（订单状态回滚+座位重打+退款记录标记失败）
			service.RollbackRefundFailure(h.DB, refundedOrder, refundRecord, preRefundStatus)
			response.FailMsg(c, response.CodeRefundFail, "退款失败：找不到支付记录")
			return
		}
		_, refundErr := wxpay.CreateWxRefund(refundedOrder, refundRecord.RefundNo, payment.TransactionID, refundRecord.Amount)
		if refundErr != nil {
			log.Printf("[ERROR] 管理端微信退款失败 订单号:%s err:%v\n", refundedOrder.OrderNo, refundErr)
			// 退款API失败后调用公共回滚函数（订单状态回滚+座位重打+退款记录标记失败）
			service.RollbackRefundFailure(h.DB, refundedOrder, refundRecord, preRefundStatus)
			response.FailMsg(c, response.CodeRefundFail, "退款失败："+refundErr.Error())
			return
		}
		log.Printf("[INFO] 管理端微信退款已发起 订单号:%s 退款单号:%s\n", refundedOrder.OrderNo, refundRecord.RefundNo)
	}

	response.OKMsg(c, "退款申请已提交，退款将在1-3个工作日内到账", nil)
}
