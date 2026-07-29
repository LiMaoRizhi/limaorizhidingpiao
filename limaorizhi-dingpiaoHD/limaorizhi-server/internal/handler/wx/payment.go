// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/money"
	"limaorizhi-server/internal/pkg/sanitize"
	v3 "limaorizhi-server/internal/pkg/wechatpay/v3"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 微信支付 v3（JSAPI 下单 + 回调通知 + 退款）
// 路由 /api/wx/pay/notify 和 /api/wx/refund/notify 保持不变

// wxPayConfig 微信支付配置
type wxPayConfig struct {
	AppID           string
	MchID           string
	NotifyURL       string
	RefundNotifyURL string

	// v3 必填
	APIv3Key         string
	MchSerialNo      string
	MchPrivateKeyPath string
	MchCertPEMPath   string // 退款等敏感接口 mTLS 用

	// 微信支付公钥模式（可选）
	WxPayPublicKeyPath string
	WxPayPublicKeyID   string
}

// getWxPayConfig 从全局配置获取微信支付配置
func getWxPayConfig() wxPayConfig {
	cfg := config.AppConfig.Wechat
	return wxPayConfig{
		AppID:              cfg.Appid,
		MchID:              cfg.MchID,
		NotifyURL:          cfg.NotifyURL,
		RefundNotifyURL:    cfg.RefundNotifyURL,
		APIv3Key:           cfg.APIv3Key,
		MchSerialNo:        cfg.MchSerialNo,
		MchPrivateKeyPath:  cfg.MchPrivateKeyPath,
		MchCertPEMPath:     cfg.MchCertPEMPath,
		WxPayPublicKeyPath: cfg.WxPayPublicKeyPath,
		WxPayPublicKeyID:   cfg.WxPayPublicKeyID,
	}
}

// isWxPayConfigured 检查微信支付 v3 是否已配置（下单必填项检查）
func isWxPayConfigured() bool {
	cfg := getWxPayConfig()
	return cfg.AppID != "" && cfg.MchID != "" &&
		cfg.APIv3Key != "" && cfg.MchSerialNo != "" &&
		cfg.MchPrivateKeyPath != "" && cfg.NotifyURL != ""
}

// isWxRefundConfigured 检查微信退款 v3 是否已配置（额外要求 mTLS 商户证书对）
func isWxRefundConfigured() bool {
	return isWxPayConfigured() && getWxPayConfig().MchCertPEMPath != ""
}

// CreateWxRefund 导出的微信退款函数（供 admin 包调用）
// refundAmount: 实际退款金额（已扣手续费），微信退款API使用此金额
// 返回退款单号（与传入相同），失败时返回 error
func CreateWxRefund(order model.Order, refundNo, transactionID string, refundAmount float64) (string, error) {
	return createWxRefund(getWxPayConfig(), order, refundNo, transactionID, refundAmount)
}

// IsWxRefundConfigured 导出检查函数（供 admin 包调用）
func IsWxRefundConfigured() bool {
	return isWxRefundConfigured()
}

// V3Client 单例

var (
	v3ClientOnce sync.Once
	v3Client     *v3.V3Client
	v3ClientErr  error
)

// getV3Client 获取 V3Client 单例
// 首次调用时初始化，之后复用（配置在启动时加载完毕且不再变更，单例安全）
// 若配置缺失（如开发环境未配置 v3 字段），返回 error，调用方降级处理
func getV3Client() (*v3.V3Client, error) {
	v3ClientOnce.Do(func() {
		cfg := getWxPayConfig()
		v3Client, v3ClientErr = v3.NewV3Client(v3.Config{
			AppID:              cfg.AppID,
			MchID:              cfg.MchID,
			APIv3Key:           cfg.APIv3Key,
			MchSerialNo:        cfg.MchSerialNo,
			PrivateKeyPath:    cfg.MchPrivateKeyPath,
			CertPEMPath:        cfg.MchCertPEMPath,
			NotifyURL:          cfg.NotifyURL,
			RefundNotifyURL:    cfg.RefundNotifyURL,
			WxPayPublicKeyPath: cfg.WxPayPublicKeyPath,
			WxPayPublicKeyID:   cfg.WxPayPublicKeyID,
		})
		if v3ClientErr != nil {
			log.Printf("[ERROR] 微信支付 v3 客户端初始化失败: %v\n", v3ClientErr)
		}
	})
	return v3Client, v3ClientErr
}

// JSAPI 下单（v3）

// createUnifiedOrder 调用微信 v3 JSAPI 下单接口
// 保留原签名以兼容 user_order.go 调用点
// 返回 prepay_id（与 v2 行为一致）
func createUnifiedOrder(cfg wxPayConfig, order model.Order, userOpenID string, clientIP string) (string, error) {
	client, err := getV3Client()
	if err != nil {
		return "", fmt.Errorf("微信支付 v3 客户端未初始化: %w", err)
	}

	// v3 金额单位为分，用整数分运算避免浮点截断
	totalFee := int(money.ToFen(order.TotalPrice))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prepayID, err := client.CreateJSAPIPay(ctx, order.OrderNo, totalFee, userOpenID)
	if err != nil {
		return "", fmt.Errorf("微信 v3 下单失败: %w", err)
	}
	return prepayID, nil
}

// buildJSAPIParams 构建小程序 wx.requestPayment 调起参数（v3）
// 签名变化：v2 返回 map[string]string（无 error），v3 返回 (map[string]string, error)
// 因为 RSA 签名可能失败，必须暴露 error 给调用方
func buildJSAPIParams(cfg wxPayConfig, prepayID string) (map[string]string, error) {
	client, err := getV3Client()
	if err != nil {
		return nil, fmt.Errorf("微信支付 v3 客户端未初始化: %w", err)
	}
	return client.BuildJSAPIPayParams(prepayID)
}

// 支付回调通知（v3）

// PayNotify 微信支付回调通知处理（v3）
// 流程：
//   1. 读取原始 body（必须未修改，参与签名计算）
//   2. 解析 4 个签名头：Wechatpay-Timestamp/Nonce/Serial/Signature
//   3. 用平台证书公钥 RSA-SHA256 验签
//   4. 用 APIv3 密钥 AES-256-GCM 解密 resource.ciphertext
//   5. 处理订单（行锁 + 金额交叉校验 + 重复回调幂等）
//   6. 返回 200 + {"code":"SUCCESS"}，失败时返回 410/411/5xx 触发微信重试
func (h *UserHandler) PayNotify(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, v3.NotifyFailResponse("读取请求体失败"))
		return
	}

	client, err := getV3Client()
	if err != nil {
		log.Printf("[ERROR] PayNotify: v3 客户端未初始化: %v\n", err)
		c.JSON(http.StatusInternalServerError, v3.NotifyFailResponse("v3 客户端未初始化"))
		return
	}

	header := v3.ParseNotifyHeader(c.Request)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, plaintext, err := client.VerifyAndDecryptNotify(ctx, rawBody, header, getWxPayConfig().APIv3Key)
	if err != nil {
		// 验签失败：可能遭受中间人攻击或微信时间戳过期，返回 410 触发微信重试
		log.Printf("[WARN] PayNotify: 验签/解密失败: %v\n", err)
		c.JSON(http.StatusGone, v3.NotifyFailResponse("验签失败: "+err.Error()))
		return
	}

	notify, err := v3.DecryptPayNotify(plaintext)
	if err != nil {
		log.Printf("[ERROR] PayNotify: 解析明文失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, v3.NotifyFailResponse("解析明文失败"))
		return
	}

	// 仅处理 SUCCESS 状态的支付通知
	if notify.TradeState != "SUCCESS" {
		log.Printf("[INFO] PayNotify: 跳过非成功状态 trade_state=%s out_trade_no=%s\n",
			notify.TradeState, notify.OutTradeNo)
		c.JSON(http.StatusOK, v3.NotifySuccessResponse())
		return
	}

	// 更新订单状态（行锁防并发）
	now := time.Now()
	jsonNow := model.JSONTime(now)
	var paidOrder model.Order
	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ? AND status = ?", notify.OutTradeNo, model.OrderStatusPending).
			First(&paidOrder).Error; err != nil {
			return fmt.Errorf("订单不存在或已支付")
		}

		// 金额交叉校验：回调金额必须与订单金额一致，防止伪造/篡改金额
		expectedFee := int(money.ToFen(paidOrder.TotalPrice))
		if notify.Amount.Total != expectedFee {
			return fmt.Errorf("支付金额不匹配: 期望 %d 分, 实际 %d 分", expectedFee, notify.Amount.Total)
		}

		if err := tx.Model(&paidOrder).Updates(map[string]interface{}{
			"status":     1,
			"pay_time":   &jsonNow,
			"pay_method": "微信支付",
		}).Error; err != nil {
			return err
		}

		payment := model.Payment{
			OrderID:       paidOrder.ID,
			PaymentNo:     fmt.Sprintf("PAY%s%s%06d", now.Format("20060102"), now.Format("150405"), paidOrder.ID),
			TransactionID: notify.TransactionID,
			Amount:        paidOrder.TotalPrice,
			Method:        "微信支付",
			Status:        1,
			PayTime:       &jsonNow,
		}
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}

		// 积分发放移出事务（事务提交后异步发放，失败仅记日志，不影响支付确认）
		return nil
	})

	if txErr != nil {
		log.Printf("[ERROR] PayNotify: 支付回调事务失败 订单号:%s err:%v\n", notify.OutTradeNo, txErr)
		// 订单可能已被处理（重复通知）或已被自动取消（竞态条件）
		// 也可能是金额不匹配导致事务失败，订单仍为 pending
		var failedOrder model.Order
		if findErr := h.DB.Where("order_no = ?", notify.OutTradeNo).First(&failedOrder).Error; findErr == nil {
			// 订单已取消（定时任务自动取消）或仍为待支付（金额不匹配）时，触发自动退款
			if failedOrder.Status == model.OrderStatusCancelled || failedOrder.Status == model.OrderStatusPending {
				order := failedOrder
				txnID := notify.TransactionID
				refundFailed := false
				refundReason := "订单超时自动取消后收到支付回调，系统自动退款"
				preStatus := int8(model.OrderStatusCancelled)
				// 实际退款金额：使用微信回调中的实际支付金额（分→元），
				// 而非 order.TotalPrice。金额不匹配时两者可能不同，
				// 退款必须退用户实际支付的金额，否则用户损失差价。
				actualPaidAmount := money.FromFen(int64(notify.Amount.Total))
				if failedOrder.Status == model.OrderStatusPending {
					refundReason = "支付金额不匹配，系统自动退款"
					preStatus = int8(model.OrderStatusPending)
				}
			
				// 事务+行锁：检查已有退款记录 + 创建/重置退款记录，防止并发回调创建重复记录
				var existingRefund model.Refund
				var newRefund model.Refund
				var hasNewRefund bool
				var retryFailedRefund bool // 标记是否重置了已失败的退款记录（复用 refund_no 防双重退款）
				lockErr := h.DB.Transaction(func(tx *gorm.DB) error {
					// 行锁订单，防止并发回调同时创建退款记录
					var lockedOrder model.Order
					if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
						Where("order_no = ?", notify.OutTradeNo).First(&lockedOrder).Error; err != nil {
						return err
					}
					// 待支付订单（金额不匹配）：先取消订单，防止用户再次发起支付
					if lockedOrder.Status == model.OrderStatusPending {
						tx.Model(&lockedOrder).Where("id = ? AND status = ?", lockedOrder.ID, model.OrderStatusPending).
							Update("status", model.OrderStatusCancelled)
						order.Status = model.OrderStatusCancelled
					}
					// 检查是否已存在该订单的退款记录（行锁内检查，防并发重复创建）
					// 包含 failed 状态：若存在已失败的退款记录，复用其 refund_no 重置为处理中，
					// 避免创建新退款记录导致微信侧重复退款（双重退款漏洞修复）
					if err := tx.Where("order_id = ? AND status IN (?, ?)", order.ID, model.RefundStatusProcessing, model.RefundStatusSuccess).First(&existingRefund).Error; err == nil {
						return nil // 已有处理中/成功的退款记录，跳过
					}
					// 检查是否存在已失败的退款记录（复用 refund_no 重试，不创建新记录）
					var failedRefund model.Refund
					if err := tx.Where("order_id = ? AND status = ?", order.ID, model.RefundStatusFailed).First(&failedRefund).Error; err == nil {
						// 重置已失败的退款记录为处理中，复用原 refund_no
						// 微信退款 API 对同一 refund_no 的重复调用是幂等的，不会重复退款
						if err := tx.Model(&failedRefund).Where("id = ? AND status = ?", failedRefund.ID, model.RefundStatusFailed).
							Updates(map[string]interface{}{"status": model.RefundStatusProcessing, "amount": actualPaidAmount}).Error; err != nil {
							return err
						}
						existingRefund = failedRefund
						existingRefund.Status = model.RefundStatusProcessing
						existingRefund.Amount = actualPaidAmount
						retryFailedRefund = true
						return nil
					}
					// 无任何已有退款记录，创建新退款记录
					refundNo := sanitize.GenerateRefundNo(order.ID)
					newRefund = model.Refund{
						OrderID:   order.ID,
						RefundNo:  refundNo,
						Amount:    actualPaidAmount,
						Reason:    refundReason,
						Status:    model.RefundStatusProcessing,
						PreStatus: preStatus,
					}
					if err := tx.Create(&newRefund).Error; err != nil {
						return err
					}
					hasNewRefund = true
					return nil
				})
				if lockErr != nil && existingRefund.ID == 0 && !hasNewRefund {
					// 事务完全失败，返回非 SUCCESS 让微信重试回调
					log.Printf("[ERROR] 自动退款事务失败 订单号:%s err:%v\n", order.OrderNo, lockErr)
					c.JSON(http.StatusInternalServerError, v3.NotifyFailResponse("退款处理失败，等待重试"))
					return
				}
			
				// 确定用于 API 调用的退款记录
				var refundForAPI model.Refund
				if existingRefund.ID != 0 {
					refundForAPI = existingRefund
				} else if hasNewRefund {
					refundForAPI = newRefund
				}
			
				// 事务外调用微信退款 API（避免事务持有期间阻塞在外部调用上）
				if refundForAPI.ID != 0 && refundForAPI.Status == model.RefundStatusProcessing {
					if isWxRefundConfigured() {
						// 使用实际支付金额退款，totalFee 也使用实际支付金额
						// （createWxRefund 内部会用 order.TotalPrice 算 totalFee，
						// 但金额不匹配场景下需用实际支付金额，因此传入覆写后的 order）
						refundOrder := order
						refundOrder.TotalPrice = actualPaidAmount
						if _, refundErr := createWxRefund(getWxPayConfig(), refundOrder, refundForAPI.RefundNo, txnID, actualPaidAmount); refundErr != nil {
							log.Printf("[ALERT][自动退款失败] 订单号:%s 退款单号:%s err:%v\n", order.OrderNo, refundForAPI.RefundNo, refundErr)
							refundFailed = true
							h.DB.Create(&model.OperationLog{
								AdminName: "系统自动退款",
								Module:    "退款告警",
								Action:    "自动退款失败",
								Target:    order.OrderNo,
								Detail:    fmt.Sprintf("订单号:%s 退款单号:%s 退款API失败，等待补偿任务重试", order.OrderNo, refundForAPI.RefundNo),
							})
							// 退款失败时不再标记为 failed，而是保持 processing 状态，
							// 让退款补偿任务能查询到并重试（补偿任务只查 status=0 的记录）
						} else if retryFailedRefund {
							log.Printf("[INFO] 自动退款重试已发起（复用原退款单号） 订单号:%s 退款单号:%s\n", order.OrderNo, refundForAPI.RefundNo)
						} else {
							log.Printf("[INFO] 自动退款已发起 订单号:%s 退款单号:%s\n", order.OrderNo, refundForAPI.RefundNo)
						}
					} else {
						// 未配置证书时直接标记退款成功（沙箱/开发环境兑底）
						log.Printf("[WARN] 微信退款未配置 mTLS 证书，直接标记退款成功 订单号:%s\n", order.OrderNo)
						if err := service.MarkRefundSuccess(h.DB, refundForAPI); err != nil {
							log.Printf("[ERROR] MarkRefundSuccess 失败 订单号:%s err:%v\n", order.OrderNo, err)
						}
					}
				}
				// 退款失败时返回非 SUCCESS 让微信重试回调，避免"已付款但未退款"坏账
				if refundFailed {
					c.JSON(http.StatusInternalServerError, v3.NotifyFailResponse("自动退款失败，等待重试"))
					return
				}
			}
		}
		// 退款已成功发起或已存在退款记录时返回 SUCCESS
		c.JSON(http.StatusOK, v3.NotifySuccessResponse())
		return
	}

	// 支付事务成功后异步发放积分（失败仅记日志，不影响支付确认）
	if txErr == nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[PANIC] PayNotify: 异步发放积分panic 订单号:%s: %v\n", notify.OutTradeNo, r)
				}
			}()
			if err := awardPoints(h.DB, paidOrder.UserID, paidOrder.TotalPrice, paidOrder.ID, "消费赠送"); err != nil {
				log.Printf("[ERROR] PayNotify: 异步发放积分失败 订单号:%s err:%v\n", notify.OutTradeNo, err)
			}
		}()
	}

	// 订阅消息：支付成功通知（异步发送，不阻塞回调）
	service.NotifyPaymentSuccess(h.DB, paidOrder)

	c.JSON(http.StatusOK, v3.NotifySuccessResponse())
}

// 退款（v3）

// createWxRefund 调用微信 v3 退款 API（使用 mTLS）
// refundAmount: 实际退款金额（已扣手续费），微信退款API使用此金额而非订单全额
// 返回退款单号（与传入相同），失败时返回 error
func createWxRefund(cfg wxPayConfig, order model.Order, refundNo string, transactionID string, refundAmount float64) (string, error) {
	client, err := getV3Client()
	if err != nil {
		return "", fmt.Errorf("微信支付 v3 客户端未初始化: %w", err)
	}
	if !client.HasMTLS() {
		return "", fmt.Errorf("退款接口要求 mTLS 商户证书，请配置 WECHAT_MCH_CERT_PEM_PATH（apiclient_cert.pem 路径）")
	}

	totalFee := int(money.ToFen(order.TotalPrice))       // 订单原金额（分）
	refundFee := int(money.ToFen(refundAmount))          // 实际退款金额（分，已扣手续费）
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := client.CreateRefund(ctx, order.OrderNo, totalFee, refundNo, transactionID, refundFee)
	if err != nil {
		return "", fmt.Errorf("微信 v3 退款失败: %w", err)
	}
	// v3 退款响应中含 status 字段（SUCCESS/CLOSED/PROCESSING/ABNORMAL）
	// 异步退款：微信通过 RefundNotify 推送最终状态
	log.Printf("[INFO] 微信 v3 退款已发起 退款单号:%s 状态:%s\n", refundNo, resp.Status)
	return refundNo, nil
}

// InitiateWxRefund 发起微信退款公共逻辑
// 封装：检查配置 → 查找支付记录 → 调用微信退款API
// 返回nil表示退款已发起（等待回调）或已直接标记成功（未配置证书降级）
// 返回error表示退款失败（支付记录不存在或微信退款API失败）
// 调用方根据场景自行处理失败：回滚订单(RollbackRefundFailure) 或 标记退款记录失败
func InitiateWxRefund(db *gorm.DB, order model.Order, refund model.Refund) error {
	if !isWxRefundConfigured() {
		return service.MarkRefundSuccess(db, refund)
	}
	var payment model.Payment
	if err := db.Where("order_id = ? AND status = ?", order.ID, model.PaymentStatusSuccess).First(&payment).Error; err != nil {
		return fmt.Errorf("查找支付记录失败: %w", err)
	}
	_, refundErr := createWxRefund(getWxPayConfig(), order, refund.RefundNo, payment.TransactionID, refund.Amount)
	if refundErr != nil {
		return fmt.Errorf("微信退款失败: %w", refundErr)
	}
	return nil
}

// RefundNotify 微信退款回调通知处理（v3）
// 流程与 PayNotify 一致：验签 → AES-GCM 解密 → 更新退款状态
func (h *UserHandler) RefundNotify(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, v3.NotifyFailResponse("读取请求体失败"))
		return
	}

	client, err := getV3Client()
	if err != nil {
		log.Printf("[ERROR] RefundNotify: v3 客户端未初始化: %v\n", err)
		c.JSON(http.StatusInternalServerError, v3.NotifyFailResponse("v3 客户端未初始化"))
		return
	}

	header := v3.ParseNotifyHeader(c.Request)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, plaintext, err := client.VerifyAndDecryptNotify(ctx, rawBody, header, getWxPayConfig().APIv3Key)
	if err != nil {
		log.Printf("[WARN] RefundNotify: 验签/解密失败: %v\n", err)
		c.JSON(http.StatusGone, v3.NotifyFailResponse("验签失败: "+err.Error()))
		return
	}

	notify, err := v3.DecryptRefundNotify(plaintext)
	if err != nil {
		log.Printf("[ERROR] RefundNotify: 解析明文失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, v3.NotifyFailResponse("解析明文失败"))
		return
	}

	// 事务 + 行锁更新退款状态，退款失败的订单状态回滚纳入同一事务，防止“退款失败但订单已退款”不一致
	now := time.Now()
	jsonNow := model.JSONTime(now)
	var refund model.Refund
	var refundUpdatedToSuccess bool // 标记本次回调是否实际将退款从处理中更新为成功，防重复回调发送重复通知
	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		// 行锁查询退款记录
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("refund_no = ?", notify.OutRefundNo).First(&refund).Error; err != nil {
			// 退款记录不存在，视为已处理，返回 SUCCESS 避免微信重试
			return nil
		}
		// 仅当退款记录仍为处理中(0)时才正常更新，防重复回调幂等
		if refund.Status != model.RefundStatusProcessing {
			// 边缘场景修复：退款记录已被标记为失败(2)，但微信回调说退款成功
			// 发生原因：createWxRefund 超时但微信实际已退款，RollbackRefundFailure
			// 已将退款标失败+订单回滚为已支付。此时微信回调到达，需修正退款状态+重新退款订单
			if refund.Status == model.RefundStatusFailed && notify.RefundStatus == "SUCCESS" {
				if err := tx.Model(&refund).Updates(map[string]interface{}{
					"status":      model.RefundStatusSuccess,
					"refund_time": &jsonNow,
				}).Error; err != nil {
					return err
				}
				refundUpdatedToSuccess = true
				log.Printf("[WARN] 退款回调修正：退款记录原为失败状态，微信回调确认退款成功 退款单号:%s\n", refund.RefundNo)
				
				// 同步更新支付记录状态为已退款
				service.UpdatePaymentStatusRefunded(tx, refund.OrderID)
				
				// RollbackRefundFailure 已将订单回滚为 PreStatus（1=待出行/2=已完成）
				// 退款实际成功后需重新将订单标记为已退款
				var order model.Order
				if err := tx.First(&order, refund.OrderID).Error; err == nil {
					refundStatus := int8(model.OrderStatusRefunded)
					if order.OrderType == 2 {
						refundStatus = int8(model.OrderStatusCancelled)
					}
					// 条件更新：仅当订单处于退款前状态（已支付/已完成）时才更新
					tx.Model(&model.Order{}).
						Where("id = ? AND status IN (?, ?)", order.ID, model.OrderStatusPaid, model.OrderStatusCompleted).
						Update("status", refundStatus)
				}
			}
			return nil
		}
		// v3 退款回调用 refund_status 表示状态：SUCCESS/CLOSED/ABNORMAL
		if notify.RefundStatus == "SUCCESS" {
			if err := tx.Model(&refund).Updates(map[string]interface{}{
				"status":      model.RefundStatusSuccess, // 成功
				"refund_time": &jsonNow,
			}).Error; err != nil {
				return err
			}
			refundUpdatedToSuccess = true // 标记本次回调实际更新了退款状态
			log.Printf("[INFO] 退款成功 退款单号:%s 金额:%.2f\n", refund.RefundNo, refund.Amount)

			// 同步更新支付记录状态为已退款，保证财务数据一致性
			service.UpdatePaymentStatusRefunded(tx, refund.OrderID)
		} else {
			// CLOSED/ABNORMAL 等异常状态：标记退款失败 + 回滚订单状态（同一事务保证一致性）
			if err := tx.Model(&refund).Update("status", model.RefundStatusFailed).Error; err != nil {
				return err
			}
			log.Printf("[WARN] 退款异常 退款单号:%s 状态:%s\n", refund.RefundNo, notify.RefundStatus)
			var order model.Order
			if err := tx.First(&order, refund.OrderID).Error; err == nil {
				preRefundStatus := refund.PreStatus
				refundStatus := int8(model.OrderStatusRefunded) // 车票退款后状态
				if order.OrderType == 2 {
					refundStatus = int8(model.OrderStatusCancelled) // 托运退款后状态
				}
				if preRefundStatus == 0 {
					// PreStatus=0 含义模糊：可能是旧记录未设字段，也可能是 PayNotify 自动退款路径
					// 对托运订单（refundStatus=4=Cancelled）来说，WHERE status=4 会匹配到已被
					// 自动取消的订单，错误地回滚为 Paid(1)，导致已取消订单状态不一致。
					// 安全做法：PreStatus=0 时跳过回滚，记录告警日志供运维人工核实。
					log.Printf("[WARN] 退款失败回滚跳过：PreStatus=0 含义模糊，无法确定回滚目标 退款单号:%s 订单号:%s\n", refund.RefundNo, order.OrderNo)
				} else if preRefundStatus == refundStatus {
					// 回滚目标与当前状态相同（如 PayNotify 自动退款路径 PreStatus=4，托运 refundStatus=4）
					// 无需回滚，避免无意义的 UPDATE
					log.Printf("[INFO] 退款失败回滚跳过：PreStatus(%d)=退款后状态(%d)，无需回滚 退款单号:%s\n", preRefundStatus, refundStatus, refund.RefundNo)
				} else {
					if err := tx.Model(&model.Order{}).
						Where("id = ? AND status = ?", order.ID, refundStatus).
						Update("status", preRefundStatus).Error; err != nil {
						log.Printf("[ERROR] 退款失败回滚订单状态失败 退款单号:%s err:%v\n", refund.RefundNo, err)
					}
				}
			}
		}
		return nil
	})
	if txErr != nil {
		log.Printf("[ERROR] 退款回调处理事务失败 退款单号:%s err:%v\n", notify.OutRefundNo, txErr)
	}

	// 事务提交后发送订阅消息（异步，不阻塞回调）
	// 仅当本次回调实际将退款从处理中更新为成功时才发送，防止重复回调发送重复通知浪费订阅配额
	if refundUpdatedToSuccess && refund.ID != 0 {
		var latestRefund model.Refund
		if err := h.DB.First(&latestRefund, refund.ID).Error; err == nil {
			// 退款成功：发送退款到账通知
			if latestRefund.Status == model.RefundStatusSuccess {
				var order model.Order
				if err := h.DB.First(&order, latestRefund.OrderID).Error; err == nil {
					service.NotifyRefundSuccess(h.DB, latestRefund, order)
				}
			}
		}
	}

	c.JSON(http.StatusOK, v3.NotifySuccessResponse())
}

