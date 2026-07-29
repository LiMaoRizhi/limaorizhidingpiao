// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package model

// 订单状态
const (
	OrderStatusPending   = 0 // 待支付
	OrderStatusPaid      = 1 // 已支付
	OrderStatusCompleted = 2 // 已完成
	OrderStatusRefunded  = 3 // 已退款
	OrderStatusCancelled = 4 // 已取消
)

// 班次状态
const (
	TripStatusSale    = 1 // 可售
	TripStatusDepart  = 2 // 已发车
	TripStatusCancel  = 3 // 已取消
	TripStatusFinish  = 4 // 已完成
)

// 优惠券状态
const (
	CouponStatusEnable  = 1 // 启用
	CouponStatusDisable = 0 // 停用
)

// 用户优惠券状态
const (
	UserCouponStatusUnused   = 0 // 未使用
	UserCouponStatusUsed     = 1 // 已使用
	UserCouponStatusExpired  = 2 // 已过期
)

// 退款状态
const (
	RefundStatusProcessing = 0 // 处理中
	RefundStatusSuccess    = 1 // 成功
	RefundStatusFailed     = 2 // 失败
)

// 支付记录状态
const (
	PaymentStatusPending = 0 // 待支付
	PaymentStatusSuccess = 1 // 成功
	PaymentStatusFailed  = 2 // 失败
	PaymentStatusRefunded = 3 // 已退款
)

// 用户状态
const (
	UserStatusEnable  = 1 // 正常
	UserStatusDisable = 0 // 封禁
	UserStatusDeleted = 2 // 已注销（可重新登录恢复）
)

// 管理员角色
const (
	AdminRoleSuper  = 1 // 超级管理员
	AdminRoleNormal = 2 // 普通管理员
)

// 司机状态
const (
	DriverStatusEnable  = 1 // 启用
	DriverStatusDisable = 0 // 禁用
)

// 线路状态
const (
	RouteStatusEnable  = 1 // 运营
	RouteStatusDisable = 0 // 停运
)
