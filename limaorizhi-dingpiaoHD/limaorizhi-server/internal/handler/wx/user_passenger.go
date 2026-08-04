package wx

import (
	"log"
	"strings"

	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/crypto"
	"limaorizhi-server/internal/pkg/idcard"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// 常用乘客管理

// PassengerList 常用乘客列表（脱敏返回，下单时通过 passenger_id 引用完整信息）
func (h *UserHandler) PassengerList(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []model.Passenger
	h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&list)
	// 脱敏：隐藏身份证号和手机号中间部分
	for i := range list {
		list[i].IDCardNo = idcard.MaskIDCard(list[i].IDCardNo)
		list[i].Phone = idcard.MaskPhone(list[i].Phone)
	}
	response.OK(c, list)
}

// PassengerCreate 添加常用乘客
type passengerRequest struct {
	Name       string `json:"name" binding:"required"`
	IDCardType int8   `json:"id_card_type"`
	IDCardNo   string `json:"id_card_no" binding:"required"`
	Phone      string `json:"phone"`
}

func (h *UserHandler) PassengerCreate(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req passengerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 身份验证：本地格式校验 + 实名认证
	if err := idcard.ValidateFormat(req.IDCardNo); err != nil {
		response.FailMsg(c, response.CodeIDCardInvalid, err.Error())
		return
	}
	result, err := h.Verifier.Verify(req.Name, req.IDCardNo)
	if err != nil {
		if h.Verifier.IsStrictMode() {
			response.FailMsg(c, response.CodeVerifyServiceErr, "实名认证服务异常，请稍后重试")
			return
		}
		log.Printf("[WARN] 实名认证服务异常: %v\n", err)
	} else if !result.Matched {
		response.FailMsg(c, response.CodeIDCardNotMatch, "姓名与身份证号不匹配")
		return
	}

	idCardType := req.IDCardType
	if idCardType == 0 {
		idCardType = 1
	}

	p := model.Passenger{
		UserID:     userID,
		Name:       req.Name,
		IDCardType: idCardType,
		IDCardNo:   req.IDCardNo,
		Phone:      req.Phone,
	}
	if err := h.DB.Create(&p).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "添加失败")
		return
	}
	// 脱敏：隐藏身份证号中间部分
	// 注意：p.IDCardNo 已被 BeforeCreate Hook 加密为密文，这里用原始明文(req.IDCardNo)脱敏
	p.IDCardNo = idcard.MaskIDCard(req.IDCardNo)
	response.OK(c, p)
}

// PassengerUpdate 编辑常用乘客
// 身份证号字段处理：前端编辑常用乘客时回显的是脱敏值（含*，如 1101**********1234），
// 若用户未修改身份证号（仍为脱敏值），则跳过校验与更新，保留数据库原明文；
// 若用户输入了新的完整身份证号（不含*），则正常校验+实名认证+加密更新。
func (h *UserHandler) PassengerUpdate(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var req passengerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 判断身份证号/手机号是否为脱敏值（含*，前端编辑回显的脱敏占位）
	isMasked := strings.Contains(req.IDCardNo, "*")
	phoneMasked := strings.Contains(req.Phone, "*")

	// 先查出原乘客信息（用于判断姓名是否变更，以及脱敏时获取原明文身份证号重新验证）
	var p model.Passenger
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&p).Error; err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 脱敏身份证号未修改时用原明文验证；姓名变更需重新验证
	verifyIDCardNo := req.IDCardNo
	needVerify := !isMasked // 身份证号被修改时必须验证
	if isMasked {
		verifyIDCardNo = p.IDCardNo // 用原明文身份证号
		if req.Name != p.Name {
			needVerify = true // 姓名变更，需用原身份证号+新姓名重新验证
		}
	}

	if needVerify {
		// 身份验证：本地格式校验 + 实名认证
		if err := idcard.ValidateFormat(verifyIDCardNo); err != nil {
			response.FailMsg(c, response.CodeIDCardInvalid, err.Error())
			return
		}
		result, err := h.Verifier.Verify(req.Name, verifyIDCardNo)
		if err != nil {
			if h.Verifier.IsStrictMode() {
				response.FailMsg(c, response.CodeVerifyServiceErr, "实名认证服务异常，请稍后重试")
				return
			}
			log.Printf("[WARN] 实名认证服务异常: %v\n", err)
		} else if !result.Matched {
			response.FailMsg(c, response.CodeIDCardNotMatch, "姓名与身份证号不匹配")
			return
		}
	}

	idCardType := req.IDCardType
	if idCardType == 0 {
		idCardType = 1
	}

	updates := map[string]interface{}{
		"name":         req.Name,
		"id_card_type": idCardType,
	}
	// 仅当用户输入了新的手机号（非脱敏值）时，才更新手机号
	if !phoneMasked {
		updates["phone"] = req.Phone
	}
	// 仅当用户输入了新的身份证号（非脱敏值）时，才更新身份证号字段
	if !isMasked {
		encIDCard, err := crypto.Encrypt(req.IDCardNo)
		if err != nil {
			response.FailMsg(c, response.CodeServerError, "身份证号加密失败")
			return
		}
		updates["id_card_no"] = encIDCard
	}
	if err := h.DB.Model(&p).Updates(updates).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}
	response.OKMsg(c, "更新成功", nil)
}

// PassengerDelete 删除常用乘客
func (h *UserHandler) PassengerDelete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Passenger{}).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "删除失败")
		return
	}
	response.OKMsg(c, "删除成功", nil)
}
