// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"errors"
	"fmt"
	"log"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/jwt"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 登录认证

// Login 微信登录（开发环境无appid时使用mock模式）
type wxLoginRequest struct {
	Code      string `json:"code" binding:"required"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req wxLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	var openid string
	cfg := config.AppConfig.Wechat

	if cfg.Appid != "" && cfg.Secret != "" {
		// 生产环境：调用微信 code2session 接口
		openid = code2Session(cfg.Appid, cfg.Secret, req.Code)
	} else if config.AppConfig.Server.Mode == "debug" {
		// 开发环境（仅debug模式）：用 code 生成 mock openid
		openid = "mock_openid_" + req.Code
	} else {
		// 非debug模式且未配置微信AppID，拒绝登录
		response.FailMsg(c, response.CodeServerError, "微信登录未配置，请联系管理员")
		return
	}

	if openid == "" {
		response.FailMsg(c, response.CodeServerError, "微信登录失败")
		return
	}

	// 查找或创建用户
	var user model.User
	result := h.DB.Where("open_id = ?", openid).First(&user)
	if result.Error != nil {
		// 区分"记录不存在"和数据库异常
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// openid 属于敏感标识，日志中不输出明文
			log.Printf("[ERROR] 登录查询用户失败 err:%v\n", result.Error)
			response.FailMsg(c, response.CodeServerError, "登录失败，请稍后重试")
			return
		}
		// 新用户
		user = model.User{
			OpenID:    openid,
			Nickname:  req.Nickname,
			AvatarURL: req.AvatarURL,
			Status:    1,
		}
		if err := h.DB.Create(&user).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "创建用户失败")
			return
		}
		// 新用户注册赠送积分
		awardRegisterPoints(h.DB, user.ID)
	} else if user.Status == 2 {
		// 已注销用户重新登录：自动恢复账号（不重复赠送注册积分）
		h.DB.Model(&user).Updates(map[string]interface{}{
			"nickname":             req.Nickname,
			"avatar_url":           req.AvatarURL,
			"status":               1,
			"token_invalid_before": nil,
		})
		user.Nickname = req.Nickname
		user.AvatarURL = req.AvatarURL
		user.Status = 1
		log.Printf("[INFO] 已注销用户重新登录恢复账号 userID=%d\n", user.ID)
	}

	if user.Status == 0 {
		response.FailMsg(c, response.CodeForbidden, "账号已被封禁")
		return
	}

	// 生成 JWT（从配置读取有效期秒数）
	token, err := jwt.GenerateToken(
		config.AppConfig.JWT.WXSecret,
		user.ID,
		user.Nickname,
		0,
		"wx",
		config.AppConfig.JWT.WXExpire,
	)
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}

	response.OK(c, gin.H{
		"token": token,
		"user":  h.buildUserResponse(user),
	})
}

// PhoneLogin 微信手机号一键登录（wx.login code + getPhoneNumber code 一步到位）
type phoneLoginRequest struct {
	LoginCode string `json:"login_code" binding:"required"` // wx.login() 返回的 code
	PhoneCode string `json:"phone_code"`                    // getPhoneNumber 返回的 code
}

func (h *UserHandler) PhoneLogin(c *gin.Context) {
	var req phoneLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	cfg := config.AppConfig.Wechat
	isMockMode := false

	// 1. 用 login_code 获取 openid
	var openid string
	if cfg.Appid != "" && cfg.Secret != "" {
		openid = code2Session(cfg.Appid, cfg.Secret, req.LoginCode)
	} else if config.AppConfig.Server.Mode == "debug" {
		// 开发环境（仅debug模式）：用 code 生成 mock openid
		openid = "mock_openid_" + req.LoginCode
		isMockMode = true
	} else {
		// 非debug模式且未配置微信AppID，拒绝登录
		response.FailMsg(c, response.CodeServerError, "微信登录未配置，请联系管理员")
		return
	}
	if openid == "" {
		response.FailMsg(c, response.CodeServerError, "微信登录失败")
		return
	}

	// 2. 获取手机号
	var phone string
	if cfg.Appid != "" && cfg.Secret != "" {
		// 生产环境：用 phone_code 调微信接口拿真实手机号
		if req.PhoneCode != "" {
			phone = getPhoneNumber(cfg.Appid, cfg.Secret, req.PhoneCode)
		}
	} else if isMockMode {
		// 开发环境（仅debug模式）：直接给 mock 手机号
		phone = "13800138000"
	}

	if phone == "" {
		response.FailMsg(c, response.CodeParamError, "未获取到手机号，请重新授权登录")
		return
	}

	// 3. 查找或创建用户
	var user model.User
	result := h.DB.Where("open_id = ?", openid).First(&user)
	if result.Error != nil {
		// 区分"记录不存在"和数据库异常
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// openid 属于敏感标识，日志中不输出明文
			log.Printf("[ERROR] 手机号登录查询用户失败 err:%v\n", result.Error)
			response.FailMsg(c, response.CodeServerError, "登录失败，请稍后重试")
			return
		}
		nickname := "用户"
		if phone != "" && len(phone) >= 4 {
			nickname = "用户" + phone[len(phone)-4:]
		}
		user = model.User{
			OpenID:   openid,
			Nickname: nickname,
			Phone:    phone,
			Status:   1,
		}
		if err := h.DB.Create(&user).Error; err != nil {
			response.FailMsg(c, response.CodeServerError, "创建用户失败")
			return
		}
		// 新用户注册赠送积分
		awardRegisterPoints(h.DB, user.ID)
	} else if user.Status == 2 {
		// 已注销用户重新登录：自动恢复账号（不重复赠送注册积分）
		nickname := "用户"
		if phone != "" && len(phone) >= 4 {
			nickname = "用户" + phone[len(phone)-4:]
		}
		h.DB.Model(&user).Updates(map[string]interface{}{
			"nickname":             nickname,
			"phone":                phone,
			"status":               1,
			"token_invalid_before": nil,
		})
		user.Nickname = nickname
		user.Phone = phone
		user.Status = 1
		log.Printf("[INFO] 已注销用户重新登录恢复账号 userID=%d\n", user.ID)
	} else {
		// 老用户，有新手机号就更新
		if phone != "" && user.Phone != phone {
			h.DB.Model(&user).Update("phone", phone)
			user.Phone = phone
		}
	}

	if user.Status == 0 {
		response.FailMsg(c, response.CodeForbidden, "账号已被封禁")
		return
	}

	token, err := jwt.GenerateToken(
		config.AppConfig.JWT.WXSecret,
		user.ID,
		user.Nickname,
		0,
		"wx",
		config.AppConfig.JWT.WXExpire,
	)
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}

	response.OK(c, gin.H{
		"token": token,
		"user":  h.buildUserResponse(user),
	})
}

// UpdateUser 更新用户资料（手机号由微信授权获取，不允许用户自行修改）
type updateUserRequest struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	updates := map[string]interface{}{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = req.AvatarURL
	}
	if len(updates) == 0 {
		response.FailMsg(c, response.CodeParamError, "无更新内容")
		return
	}

	if err := h.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "更新失败")
		return
	}

	var user model.User
	h.DB.First(&user, userID)
	response.OK(c, h.buildUserResponse(user))
}

// DeleteAccount 用户注销账号（软注销：清除个人敏感数据，保留用户记录避免破坏订单外键）
// 安全策略：
// 1. 拒绝有未完成订单（待支付/待出行）的用户注销，需先取消或完成
// 2. 事务内清除：常用乘客(含身份证号)、优惠券、积分
// 3. 订单保留(财务凭证)但匿名化联系人信息
// 4. 用户记录：清空昵称/头像/手机号，status=0(禁止登录)，token_invalid_before=now(踢出当前会话)
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 1. 检查是否有未完成订单（待支付=0 或 待出行=1）
	var activeCount int64
	if err := h.DB.Model(&model.Order{}).
		Where("user_id = ? AND status IN (0, 1)", userID).
		Count(&activeCount).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "查询订单失败")
		return
	}
	if activeCount > 0 {
		response.FailMsg(c, response.CodeParamError,
			fmt.Sprintf("您有%d笔未完成订单，请先取消或完成后再注销账号", activeCount))
		return
	}

	// 2. 事务内清除个人敏感数据
	emptyStr := ""
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 2a. 删除常用乘客（含加密身份证号，属于敏感个人信息）
		if err := tx.Where("user_id = ?", userID).Delete(&model.Passenger{}).Error; err != nil {
			return fmt.Errorf("删除常用乘客失败: %w", err)
		}
		// 2b. 删除用户优惠券
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserCoupon{}).Error; err != nil {
			return fmt.Errorf("删除优惠券失败: %w", err)
		}
		// 2c. 删除积分余额和明细
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserPoints{}).Error; err != nil {
			return fmt.Errorf("删除积分余额失败: %w", err)
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.PointRecord{}).Error; err != nil {
			return fmt.Errorf("删除积分明细失败: %w", err)
		}
		// 2d. 匿名化已完成/已取消/已退款订单的联系人信息（保留订单本身作为财务凭证）
		if err := tx.Model(&model.Order{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
			"contact_name":   emptyStr,
			"contact_phone":  emptyStr,
			"sender_name":    emptyStr,
			"sender_phone":   emptyStr,
			"receiver_name":  emptyStr,
			"receiver_phone": emptyStr,
		}).Error; err != nil {
			return fmt.Errorf("匿名化订单联系人失败: %w", err)
		}
		// 2d-2. 匿名化订单乘客表中的敏感信息（姓名、身份证号、手机号）
		if err := tx.Table("order_passengers").
			Joins("JOIN orders ON orders.id = order_passengers.order_id").
			Where("orders.user_id = ?", userID).
			Updates(map[string]interface{}{
				"order_passengers.name":        "已注销用户",
				"order_passengers.id_card_no":  emptyStr,
				"order_passengers.phone":       emptyStr,
			}).Error; err != nil {
			return fmt.Errorf("匿名化订单乘客信息失败: %w", err)
		}
		// 2e. 用户记录软注销：清空个人信息 + 标记已注销 + 踢出会话
		// status=2 表示已注销（与 status=0 管理员封禁区分），允许用户重新登录时自动恢复
		now := time.Now()
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"nickname":             "已注销用户",
			"avatar_url":           emptyStr,
			"phone":                emptyStr,
			"status":               2, // 已注销（可重新登录恢复）
			"token_invalid_before": &now,
		}).Error; err != nil {
			return fmt.Errorf("注销用户记录失败: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("[ERROR] 用户注销失败 userID:%d err:%v\n", userID, err)
		response.FailMsg(c, response.CodeServerError, "注销失败，请稍后重试")
		return
	}

	log.Printf("[INFO] 用户注销成功 userID:%d\n", userID)
	response.OKMsg(c, "账号已注销", nil)
}

// buildUserResponse 构建用户信息响应（含 is_driver / is_admin 标识）
// is_driver 用于小程序端判断是否显示司机核销入口：仅当用户手机号匹配启用状态的司机时为 true
// is_admin 用于小程序端判断是否显示管理后台入口：仅当用户手机号匹配启用状态的管理员时为 true
// 注：is_admin 仅控制前端入口可见性，真正的管理接口鉴权由 WxAdminAuth 中间件独立校验，前端隐藏≠安全
func (h *UserHandler) buildUserResponse(user model.User) gin.H {
	var driverCount int64
	if user.Phone != "" {
		h.DB.Model(&model.Driver{}).Where("phone = ? AND status = 1", user.Phone).Count(&driverCount)
	}
	// 管理员身份识别：手机号匹配启用状态的管理员账号（与 is_driver 同一套思路）
	isAdmin := false
	adminRole := 0
	adminRealName := ""
	if user.Phone != "" {
		var admin model.AdminUser
		h.DB.Where("phone = ? AND status = 1", user.Phone).First(&admin)
		if admin.ID > 0 {
			isAdmin = true
			adminRole = int(admin.Role)
			adminRealName = admin.RealName
		}
	}
	return gin.H{
		"id":             user.ID,
		"nickname":       user.Nickname,
		"avatar_url":     user.AvatarURL,
		"phone":          user.Phone,
		"is_driver":      driverCount > 0,
		"is_admin":       isAdmin,
		"admin_role":     adminRole,     // 1=超级管理员 2=普通管理员，前端可据此隐藏超管专属功能
		"admin_real_name": adminRealName, // 管理后台展示用
	}
}

// UserInfo 获取用户信息
func (h *UserHandler) UserInfo(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user model.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		response.Fail(c, response.CodeUserNotFound)
		return
	}
	response.OK(c, h.buildUserResponse(user))
}
