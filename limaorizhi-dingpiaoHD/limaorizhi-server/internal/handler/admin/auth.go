package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/jwt"
	"limaorizhi-server/internal/pkg/response"
	redis "limaorizhi-server/internal/pkg/redis"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 弱口令校验
// 校验密码强度：长度>=8、必须同时包含字母与数字、拒绝常见弱口令
func validatePasswordStrength(pwd string) error {
	if len(pwd) < 8 {
		return fmt.Errorf("密码长度不能少于8位")
	}
	hasLetter, hasDigit := false, false
	for _, ch := range pwd {
		switch {
		case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
			hasLetter = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("密码必须同时包含字母和数字")
	}
	// 常见弱口令黑名单（小写比对）
	weakPasswords := map[string]bool{
		"12345678": true, "123456789": true, "1234567890": true,
		"admin123": true, "admin1234": true, "adminadmin": true,
		"password": true, "password1": true, "password12": true,
		"abc12345": true, "qwerty123": true, "11111111": true,
		"00000000": true, "88888888": true, "root1234": true,
	}
	if weakPasswords[strings.ToLower(pwd)] {
		return fmt.Errorf("密码过于简单，请更换为更复杂的密码")
	}
	return nil
}

// 管理员登录失败计数器（IP+用户名维度）
// 连续失败5次后锁定15分钟
// 优先使用 Redis，不可用时降级为进程内存计数
type loginAttempt struct {
	failCount int
	lockUntil time.Time
}

var loginAttempts sync.Map // key: "ip:username"（降级方案）

const loginLockDuration = 15 * time.Minute

// isLoginLocked 检查是否已锁定
func isLoginLocked(ip, username string) bool {
	// 优先检查 Redis
	locked, fallback := redis.IsLoginLocked(ip, username)
	if !fallback {
		return locked
	}

	// 降级：进程内存检查
	key := ip + ":" + username
	val, ok := loginAttempts.Load(key)
	if !ok {
		return false
	}
	attempt := val.(*loginAttempt)
	return time.Now().Before(attempt.lockUntil)
}

// recordLoginFail 记录登录失败，达到阈值时锁定
func recordLoginFail(ip, username string) {
	// 优先使用 Redis
	if fallback := redis.RecordLoginFail(ip, username, loginLockDuration); !fallback {
		return
	}

	// 降级：进程内存计数
	key := ip + ":" + username
	var attempt *loginAttempt
	val, ok := loginAttempts.Load(key)
	if ok {
		attempt = val.(*loginAttempt)
	} else {
		attempt = &loginAttempt{}
	}
	attempt.failCount++
	if attempt.failCount >= 5 {
		attempt.lockUntil = time.Now().Add(loginLockDuration)
		attempt.failCount = 0 // 重置计数
	}
	loginAttempts.Store(key, attempt)
}

// clearLoginFail 登录成功时清除失败计数
func clearLoginFail(ip, username string) {
	// 优先使用 Redis
	if fallback := redis.ClearLoginFail(ip, username); !fallback {
		return
	}

	// 降级：进程内存清除
	loginAttempts.Delete(ip + ":" + username)
}

type AuthHandler struct {
	DB *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

type loginRequest struct {
	Username             string `json:"username" binding:"required"`
	Password              string `json:"password" binding:"required"`
	CaptchaVerification  string `json:"captchaVerification" binding:"required"`
}

// Login 管理员登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}

	// 校验滑动验证码token（一次性消费，防止绕过验证码直接调用登录接口）
	if !service.VerifyCaptchaToken(req.CaptchaVerification) {
		response.FailMsg(c, response.CodeForbidden, "验证码校验失败，请重新验证")
		return
	}

	ip := c.ClientIP()
	if isLoginLocked(ip, req.Username) {
		response.FailMsg(c, response.CodeForbidden, "登录失败次数过多，请15分钟后再试")
		return
	}

	var admin model.AdminUser
	if err := h.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		recordLoginFail(ip, req.Username)
		response.FailMsg(c, response.CodeForbidden, "用户名或密码错误")
		return
	}

	if admin.Status != 1 {
		recordLoginFail(ip, req.Username)
		response.FailMsg(c, response.CodeForbidden, "用户名或密码错误")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		recordLoginFail(ip, req.Username)
		response.FailMsg(c, response.CodeForbidden, "用户名或密码错误")
		return
	}

	// 登录成功，清除失败计数
	clearLoginFail(ip, req.Username)

	// 登录时间更新一下
	now := time.Now()
	if err := h.DB.Model(&admin).Update("last_login_at", &now).Error; err != nil {
		// 非关键错误，仅记录日志
		log.Printf("更新登录时间失败: %v\n", err)
	}

	// 生成Token
	cfg := config.AppConfig.JWT
	token, err := jwt.GenerateToken(cfg.AdminSecret, admin.ID, admin.Username, admin.Role, "admin", cfg.AdminExpire)
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return
	}

	response.OK(c, gin.H{
		"token":               token,
		"id":                  admin.ID,
		"username":            admin.Username,
		"real_name":           admin.RealName,
		"avatar_url":          admin.AvatarURL,
		"role":                admin.Role,
		"must_change_password": admin.MustChangePassword, // 前端据此强制跳转改密页
	})
}

// Logout 退出登录（写入 token_invalid_before，使该管理员早于此时间签发的Token失效）
// 此接口经过 JWTAuth 中间件，Token已在context中验证有效
func (h *AuthHandler) Logout(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	now := time.Now()
	// 写入登出时间到数据库，重启不丢失、多实例共享
	if err := h.DB.Model(&model.AdminUser{}).Where("id = ?", adminID).Update("token_invalid_before", &now).Error; err != nil {
		log.Printf("写入登出时间失败: %v\n", err)
	}
	response.OKMsg(c, "退出成功", nil)
}

// Profile 获取当前管理员信息
func (h *AuthHandler) Profile(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var admin model.AdminUser
	if err := h.DB.First(&admin, adminID).Error; err != nil {
		response.Fail(c, response.CodeUserNotFound)
		return
	}
	response.OK(c, gin.H{
		"id":                  admin.ID,
		"username":            admin.Username,
		"real_name":           admin.RealName,
		"avatar_url":          admin.AvatarURL,
		"role":                admin.Role,
		"last_login_at":       admin.LastLoginAt,
		"must_change_password": admin.MustChangePassword,
	})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamError)
		return
	}
	if err := validatePasswordStrength(req.NewPassword); err != nil {
		response.FailMsg(c, response.CodeParamError, err.Error())
		return
	}
	// 新密码不能与原密码相同
	if req.NewPassword == req.OldPassword {
		response.FailMsg(c, response.CodeParamError, "新密码不能与原密码相同")
		return
	}

	adminID := c.GetUint("admin_id")
	var admin model.AdminUser
	if err := h.DB.First(&admin, adminID).Error; err != nil {
		response.Fail(c, response.CodeUserNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.OldPassword)); err != nil {
		response.FailMsg(c, response.CodeForbidden, "原密码错误")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.FailMsg(c, response.CodeServerError, "密码加密失败")
		return
	}

	if err := h.DB.Model(&admin).Update("password_hash", string(hash)).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "密码更新失败")
		return
	}
	// 清除「必须改密」标记，并使旧Token失效：更新token_invalid_before为当前时间，旧Token立即失效
	now := time.Now()
	if err := h.DB.Model(&admin).Updates(map[string]interface{}{
		"token_invalid_before":  &now,
		"must_change_password": false,
	}).Error; err != nil {
		response.FailMsg(c, response.CodeServerError, "Token失效标记写入失败")
		return
	}
	response.OKMsg(c, "密码修改成功，请重新登录", nil)
}

// WriteLog 写入操作日志（供各 handler 调用）
func WriteLog(c *gin.Context, db *gorm.DB, module, action, target, detail string) {
	adminID := c.GetUint("admin_id")
	adminName, _ := c.Get("admin_name")
	nameStr, ok := adminName.(string)
	if !ok {
		nameStr = fmt.Sprintf("%v", adminName)
	}
	logEntry := model.OperationLog{
		AdminID:   adminID,
		AdminName: nameStr,
		Module:    module,
		Action:    action,
		Target:    target,
		Detail:    detail,
		IPAddress: c.ClientIP(),
	}
	if err := db.Create(&logEntry).Error; err != nil {
		log.Printf("写入操作日志失败: %v\n", err)
	}
}

// InitAdmin 初始化默认管理员账号（使用随机临时密码）
func InitAdmin(db *gorm.DB) error {
	var count int64
	db.Model(&model.AdminUser{}).Count(&count)
	if count > 0 {
		return nil
	}
	// 生成随机临时密码（admin前缀 + 16位随机hex = 21位），避免使用可预测的弱口令
	// 也可通过环境变量 ADMIN_INIT_PASSWORD 指定初始密码
	tmpPassword := os.Getenv("ADMIN_INIT_PASSWORD")
	if tmpPassword == "" {
		tmpPasswordBytes := make([]byte, 8)
		if _, err := rand.Read(tmpPasswordBytes); err != nil {
			return fmt.Errorf("生成随机密码失败: %w", err)
		}
		tmpPassword = "admin" + hex.EncodeToString(tmpPasswordBytes)
		// 随机生成的密码写入文件供首次登录读取（避免明文打印到日志）
		passwordFile := ".admin_initial_password"
		if err := os.WriteFile(passwordFile, []byte(tmpPassword), 0600); err != nil {
			log.Printf("[WARN] 无法写入初始密码文件: %v\n", err)
		} else {
			log.Printf("[安全警告] 已创建默认管理员账号 admin，初始密码已写入 %s\n", passwordFile)
		}
	} else {
		log.Println("[安全警告] 已创建默认管理员账号 admin（使用环境变量指定的密码）")
	}
	log.Println("[安全警告] 请立即登录并修改密码！")
	hash, err := bcrypt.GenerateFromPassword([]byte(tmpPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}
	admin := model.AdminUser{
		Username:           "admin",
		PasswordHash:       string(hash),
		RealName:           "超级管理员",
		Role:               1,
		Status:             1,
		MustChangePassword: true, // 默认口令必须在首次登录后修改
	}
	return db.Create(&admin).Error
}

// InitConfig 初始化默认系统配置（自动补全缺失的配置项）
func InitConfig(db *gorm.DB) error {
	configs := []model.SystemConfig{
		{ConfigKey: "site_name", ConfigValue: "狸猫日志售票", Description: "站点名称"},
		{ConfigKey: "order_expire_minutes", ConfigValue: "15", Description: "待支付订单超时分钟数"},
		{ConfigKey: "refund_before_departure_hours", ConfigValue: "2", Description: "发车前可退款小时数"},
		{ConfigKey: "customer_service_phone", ConfigValue: "", Description: "客服电话"},
		{ConfigKey: "after_sales_wechat", ConfigValue: "lihao68681818", Description: "售后微信"},
		{ConfigKey: "refund_fee_rate", ConfigValue: "10", Description: "退票手续费率(%)"},
		{ConfigKey: "notice", ConfigValue: "", Description: "首页公告内容"},
		// 托运配置
		{ConfigKey: "cargo_price_per_km", ConfigValue: "0.5", Description: "托运每公里运费(元)"},
		{ConfigKey: "cargo_min_fee", ConfigValue: "10", Description: "托运最低运费(元)"},
		{ConfigKey: "cargo_free_weight", ConfigValue: "5", Description: "托运免费重量(kg)"},
		{ConfigKey: "cargo_extra_weight_fee", ConfigValue: "3", Description: "托运超重费(元/kg)"},
		{ConfigKey: "cargo_max_weight", ConfigValue: "50", Description: "托运最大重量限制(kg)"},
		{ConfigKey: "homepage_coupon_ids", ConfigValue: "", Description: "首页展示的优惠券ID(逗号分隔)"},
		// 协议政策默认内容（上线前请在管理后台「协议政策」页面完善内容）
		{ConfigKey: "user_agreement", ConfigValue: defaultUserAgreement, Description: "用户协议"},
		{ConfigKey: "privacy_policy", ConfigValue: defaultPrivacyPolicy, Description: "隐私政策"},
	}
	for _, cfg := range configs {
		var count int64
		db.Model(&model.SystemConfig{}).Where("config_key = ?", cfg.ConfigKey).Count(&count)
		if count == 0 {
			db.Create(&cfg)
		}
	}

	// 初始化默认积分规则（按 rule_type 逐条检查，避免表中有旧数据导致漏初始化）
	var consumeRule model.PointRule
	if err := db.Where("rule_type = 1").First(&consumeRule).Error; err != nil {
		db.Create(&model.PointRule{
			RuleName: "消费赠送", RuleType: 1, PointsPerYuan: 1, FixedPoints: 0,
			Description: "用户消费每1元获得1积分", Status: 1,
		})
		log.Println(`[初始化] 已创建 消费赠送 积分规则`)
	}
	var registerRule model.PointRule
	if err := db.Where("rule_type = 2").First(&registerRule).Error; err != nil {
		db.Create(&model.PointRule{
			RuleName: "注册赠送", RuleType: 2, PointsPerYuan: 0, FixedPoints: 50,
			Description: "新用户注册赠送50积分", Status: 1,
		})
		log.Println(`[初始化] 已创建 注册赠送 积分规则`)
	}

	return nil
}

// InitTestData 初始化测试数据（仅 debug 模式调用）
// 旧的内置测试数据已清除，请通过管理后台手动新增站点、线路、车辆、司机、班次等业务数据。
// 如需在此函数中硬编码测试数据，请参考 model 包中各结构体的字段定义。
func InitTestData(db *gorm.DB) error {
	// 在此添加自定义测试数据，示例：
	//
	// var stationCount int64
	// db.Model(&model.Station{}).Count(&stationCount)
	// if stationCount == 0 {
	// 	db.Create(&[]model.Station{
	// 		{Name: "站点A", Pinyin: "zhandiana", SortOrder: 1, Status: 1},
	// 	})
	// }
	return nil
}

// 协议政策默认内容

const defaultUserAgreement = `狸猫日志售票平台用户协议

欢迎使用狸猫日志售票平台（以下简称"本平台"）。本协议是您与本平台之间就使用本平台提供的售票、托运及相关服务所订立的协议。请您在使用本平台服务前，仔细阅读并充分理解本协议的全部内容。一旦您点击"同意"或使用本平台服务，即视为您已阅读并同意接受本协议的约束。

一、服务内容
本平台为用户提供城乡公交、城际客运、旅游专线等票务预订服务，以及货物托运服务。具体服务内容以本平台实际提供的为准。

二、用户注册与登录
1. 用户通过微信授权手机号登录本平台，授权即视为同意本协议。
2. 用户应确保提供的手机号码真实有效，因手机号码失效导致的损失由用户自行承担。
3. 用户不得将账号转让、出借给他人使用。

三、购票规则
1. 实名制购票：购票需提供乘车人真实姓名及有效身份证件号码，乘车时请携带与购票信息一致的证件。
2. 班次时间：请以实际发车时间为准，建议提前到达乘车站点。
3. 座位分配：系统自动分配座位，如需调换请联系司机。
4. 订单支付：下单后请在规定时间内完成支付，超时订单将自动取消。

四、退票与改签
1. 退票需在发车前规定时间内申请，超出时限将无法退票。
2. 退票将收取一定比例的手续费，具体费率以退票页面提示为准。
3. 退款将在受理后原路退回，到账时间以支付渠道为准。
4. 改签规则以实际线路政策为准，部分班次不支持改签。

五、托运服务
1. 托运物品需符合国家相关规定，违禁品不予受理。
2. 托运重量超出免费额度将收取额外费用。
3. 请如实填写寄收件人信息，因信息错误导致的损失由用户承担。

六、用户行为规范
1. 用户不得利用本平台从事违法违规活动。
2. 用户不得恶意下单、囤票或干扰平台正常运营。
3. 用户应文明乘车，遵守乘车秩序。

七、免责声明
1. 因不可抗力（自然灾害、交通管制等）导致的班次取消或延误，本平台不承担责任。
2. 因网络故障、系统维护等导致的服务中断，本平台将尽快恢复。
3. 本平台不对第三方提供的服务质量承担责任。

八、协议修改
本平台保留随时修改本协议的权利，修改后的协议自发布之日起生效。继续使用本平台服务即视为同意修改后的协议。

九、争议解决
本协议的订立、执行和解释均适用中华人民共和国法律。因本协议引起的争议，双方应协商解决，协商不成的，可向本平台所在地人民法院提起诉讼。

如有疑问，请联系客服。`

const defaultPrivacyPolicy = `狸猫日志售票平台隐私政策

本政策适用于您使用狸猫日志售票平台（以下简称"本平台"）提供的售票、托运及相关服务。本政策将告知您我们如何收集、使用、存储和保护您的个人信息。请您在使用本平台服务前，仔细阅读并充分理解本政策的全部内容。

一、我们收集的个人信息
1. 手机号码：通过微信授权获取，用于账号登录和联系。
2. 身份证件信息：包括姓名和身份证号码，用于实名制购票和乘车核验。
3. 位置信息：司机端用于车辆实时定位和到站追踪（仅司机端使用）。
4. 订单信息：包括购票、托运订单的详细记录。
5. 联系人信息：取件人、寄件人姓名和电话（托运服务）。
6. 用户资料：昵称、头像等微信公开信息。

二、信息使用目的
1. 提供售票、托运服务：处理订单、分配座位、核验乘车人身份。
2. 客户服务：处理退票、退款、投诉等售后问题。
3. 安全保障：实名制核验、防止恶意购票、保障乘车安全。
4. 服务改进：分析使用数据以优化服务体验。
5. 法律合规：遵守道路客运实名制等相关法律法规要求。

三、信息存储与保护
1. 身份证件号码采用 AES-256 加密存储，确保数据安全。
2. 我们采取技术和管理措施保护您的个人信息，防止泄露、损毁或丢失。
3. 个人信息保存期限为服务所必需的最短时间，超出期限将删除或匿名化处理。
4. 未经您授权，我们不会向任何第三方提供您的个人信息。

四、信息共享
在以下情形中，我们可能共享您的个人信息：
1. 获得您的明确同意。
2. 根据法律法规或行政/司法机关的要求。
3. 为完成退票退款等服务，向支付渠道提供必要信息。

五、您的权利
1. 查看与修改：您可在"个人中心"查看和修改个人信息。
2. 账号注销：您可通过"注销账号"功能删除所有个人数据。
3. 撤回授权：您可通过关闭微信授权停止信息收集。

六、未成年人保护
本平台不面向未满14周岁的未成年人提供服务。

七、隐私政策修改
本平台保留随时修改本隐私政策的权利，修改后的政策自发布之日起生效。继续使用本平台服务即视为同意修改后的政策。

八、联系我们
如有任何关于个人信息保护的问题，请联系客服。

本政策最后更新日期：发布之日。`
