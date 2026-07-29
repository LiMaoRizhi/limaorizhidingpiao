// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// 错误码
const (
	CodeSuccess         = 0
	CodeParamError      = 1001
	CodeUnauthorized    = 1002
	CodeForbidden       = 1003
	CodeUserNotFound    = 2001
	CodePhoneBound      = 2002
	CodeTripNotFound    = 3001
	CodeNoSeat          = 3002
	CodeTripUnavailable = 3003
	CodeOrderNotFound   = 4001
	CodeOrderStatusErr  = 4002
	CodeSeatLockFail    = 4003
	CodePayFail         = 5001
	CodeRefundFail      = 5002
	CodeDriverNotFound  = 6001
	CodeDriverDisabled  = 6002
	CodeAlreadyChecked  = 6003 // 已核销
	CodeNotThisTrip     = 6004 // 不属于该班次
	CodeIDCardInvalid   = 7001 // 身份证号格式错误
	CodeIDCardNotMatch  = 7002 // 姓名与身份证号不匹配
	CodeVerifyServiceErr = 7003 // 实名认证服务异常
	CodeServerError     = 9999
)

var codeMessages = map[int]string{
	CodeSuccess:         "成功",
	CodeParamError:      "参数错误",
	CodeUnauthorized:    "未授权或Token失效",
	CodeForbidden:       "权限不足",
	CodeUserNotFound:    "用户不存在",
	CodePhoneBound:      "手机号已被绑定",
	CodeTripNotFound:    "班次不存在",
	CodeNoSeat:          "余座不足",
	CodeTripUnavailable: "班次已发车或已取消",
	CodeOrderNotFound:   "订单不存在",
	CodeOrderStatusErr:  "订单状态不允许此操作",
	CodeSeatLockFail:    "座位锁定失败",
	CodePayFail:         "支付失败",
	CodeRefundFail:      "退款失败",
	CodeDriverNotFound:  "司机不存在",
	CodeDriverDisabled:  "司机账号已禁用",
	CodeAlreadyChecked:  "该票已核销",
	CodeNotThisTrip:      "该票不属于此班次",
	CodeIDCardInvalid:    "身份证号格式错误",
	CodeIDCardNotMatch:   "姓名与身份证号不匹配",
	CodeVerifyServiceErr: "实名认证服务异常",
	CodeServerError:      "服务器内部错误",
}

func msg(code int) string {
	if m, ok := codeMessages[code]; ok {
		return m
	}
	return "未知错误"
}

// OK 成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: msg(CodeSuccess),
		Data:    data,
	})
}

// OKMsg 成功响应带自定义消息
func OKMsg(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, code int) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: msg(code),
	})
}

// FailMsg 失败响应带自定义消息
func FailMsg(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// FailWithData 失败响应带自定义消息和附加数据（用于前端需要根据数据做后续决策的场景，如强制删除）
func FailWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Page 分页响应
func Page(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: msg(CodeSuccess),
		Data: PageResult{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}
