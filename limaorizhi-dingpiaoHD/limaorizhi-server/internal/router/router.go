package router

import (
	"strings"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/handler/admin"
	wxhandler "limaorizhi-server/internal/handler/wx"
	"limaorizhi-server/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB) *gin.Engine {
	// 使用 gin.New() 替代 gin.Default()，避免 gin.Logger() 与 middleware.Logger() 重复输出日志
	r := gin.New()
	r.Use(gin.Recovery()) // gin.Default() 内置的 panic 恢复

	// 可信代理配置：留空则不信任任何代理，防X-Forwarded-For伪造IP绕过限流
	if tp := config.AppConfig.Server.TrustedProxies; tp != "" {
		_ = r.SetTrustedProxies(strings.Split(tp, ","))
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	// 中间件
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Logger())

	// 静态文件（上传目录）
	r.Static("/uploads", "./uploads")

	// 健康检查接口（运维必需，负载均衡器/K8s探活）
	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(503, gin.H{"status": "error", "error": "数据库连接失败"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "error", "error": "数据库不可达"})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "service": "limaorizhi-server"})
	})

	// 管理端
	authHandler := admin.NewAuthHandler(db)
	stationHandler := admin.NewStationHandler(db)
	routeHandler := admin.NewRouteHandler(db)
	vehicleHandler := admin.NewVehicleHandler(db)
	tripHandler := admin.NewTripHandler(db)
	orderHandler := admin.NewOrderHandler(db)
	userHandler := admin.NewWxUserHandler(db)
	configHandler := admin.NewConfigHandler(db)
	adminUserHandler := admin.NewAdminUserHandler(db)
	uploadHandler := admin.NewUploadHandler()
	dashboardHandler := admin.NewDashboardHandler(db)
	bannerHandler := admin.NewBannerHandler(db)
	driverHandler := admin.NewDriverHandler(db)
	couponHandler := admin.NewCouponHandler(db)
	userCouponHandler := admin.NewUserCouponHandler(db)
	pointRuleHandler := admin.NewPointRuleHandler(db)
	userPointsHandler := admin.NewUserPointsHandler(db)
	designHandler := admin.NewDesignHandler(db)
	idCardVerifyHandler := admin.NewIDCardVerifyHandler(db) // 身份认证缓存管理（监控 + 主动失效）
	trackHandler := admin.NewTrackHandler(db)               // 轨迹可视化（实时监控 + 历史回放）
	aiHandler := admin.NewAIHandler(db)
	insuranceProviderHandler := admin.NewInsuranceProviderHandler(db)
	wxDriverHandler := wxhandler.NewDriverHandler(db)
	wxUserHandler := wxhandler.NewUserHandler(db)
	wxCargoHandler := wxhandler.NewCargoHandler(db)

	// 公开接口
	r.POST("/admin/login", middleware.RateLimit(), authHandler.Login)

	// 滑动验证码（公开接口，加限流防刷）
	captchaHandler := admin.NewCaptchaHandler()
	r.GET("/admin/captcha/get", middleware.RateLimit(), captchaHandler.GetCaptcha)
	r.POST("/admin/captcha/check", middleware.RateLimit(), captchaHandler.CheckCaptcha)

	// 小程序公开接口（无需鉴权，登录接口加限流防刷）
	r.GET("/api/banners", bannerHandler.PublicList)
	r.GET("/api/homepage/coupons", couponHandler.PublicHomeCoupons)
	r.GET("/api/homepage/layout", designHandler.PublicLayout)
	r.GET("/api/design/page-layout", designHandler.PublicPageLayout)
	r.GET("/api/wx/stations", middleware.RateLimit(), wxUserHandler.Stations)
	r.GET("/api/wx/routes", middleware.RateLimit(), wxUserHandler.Routes)
	r.GET("/api/wx/trips", middleware.RateLimit(), wxUserHandler.Trips)
	r.GET("/api/wx/trips/:id", middleware.RateLimit(), wxUserHandler.TripDetail)
	r.GET("/api/wx/trips/:id/available-seats", middleware.RateLimit(), wxUserHandler.TripAvailableSeats)
	r.GET("/api/wx/config", middleware.RateLimit(), wxUserHandler.PublicConfig)
	r.POST("/api/wx/login", middleware.RateLimit(), wxUserHandler.Login)
	r.POST("/api/wx/phone-login", middleware.RateLimit(), wxUserHandler.PhoneLogin)

	// 需要鉴权的管理端接口
	adminGroup := r.Group("/admin", middleware.JWTAuth(db))
	{
		// 管理员信息
		adminGroup.POST("/logout", authHandler.Logout)
		adminGroup.GET("/profile", authHandler.Profile)
		adminGroup.PUT("/password", authHandler.ChangePassword)

		// 仪表盘
		adminGroup.GET("/dashboard", dashboardHandler.Stats)

		// 站点管理
		adminGroup.GET("/stations", stationHandler.List)
		adminGroup.GET("/stations/all", stationHandler.All)
		adminGroup.GET("/stations/:id/routes", stationHandler.StationRoutes) // 站点枢纽视图：该站关联的所有线路
		adminGroup.POST("/stations", stationHandler.Create)
		adminGroup.PUT("/stations/:id", stationHandler.Update)
		adminGroup.DELETE("/stations/:id", stationHandler.Delete)

		// 线路管理
		adminGroup.GET("/routes", routeHandler.List)
		adminGroup.GET("/routes/all", routeHandler.All) // 不分页全部线路（下拉选择/线路总览用）
		adminGroup.POST("/routes", routeHandler.Create)
		adminGroup.PUT("/routes/:id", routeHandler.Update)
		adminGroup.DELETE("/routes/:id", routeHandler.Delete)
		adminGroup.GET("/routes/:id/stations", routeHandler.Stations)

		// 车辆管理
		adminGroup.GET("/vehicles", vehicleHandler.List)
		adminGroup.GET("/vehicles/all", vehicleHandler.All)
		adminGroup.POST("/vehicles", vehicleHandler.Create)
		adminGroup.PUT("/vehicles/:id", vehicleHandler.Update)
		adminGroup.DELETE("/vehicles/:id", vehicleHandler.Delete)

		// 班次管理
		adminGroup.GET("/trips", tripHandler.List)
		adminGroup.POST("/trips", tripHandler.Create)
		adminGroup.PUT("/trips/:id", tripHandler.Update)
		adminGroup.DELETE("/trips/:id", tripHandler.Delete)
		adminGroup.POST("/trips/batch", tripHandler.BatchCreate)
		adminGroup.POST("/trips/cleanup", tripHandler.CleanupHistory)

		// 轨迹可视化（实时监控运行中班次 + 历史轨迹回放）
		adminGroup.GET("/trips/active", trackHandler.ActiveTrips)
		adminGroup.GET("/trips/:id/track", trackHandler.TripTrack)

		// 订单管理
		adminGroup.GET("/orders", orderHandler.List)
		adminGroup.GET("/orders/export", orderHandler.Export) // 订单导出CSV
		adminGroup.GET("/orders/:id", orderHandler.Detail)
		adminGroup.PUT("/orders/:id/status", orderHandler.UpdateStatus)
		adminGroup.POST("/orders/:id/refund", middleware.RequireSuperAdmin(), orderHandler.Refund)

		// 用户管理
		adminGroup.GET("/users", userHandler.List)
		adminGroup.GET("/users/:id", userHandler.Detail)
		adminGroup.PUT("/users/:id/status", userHandler.UpdateStatus)
		adminGroup.GET("/passengers", userHandler.PassengerList)

		// 实名认证缓存管理（命中率监控 + 主动失效）
		// 注意路由顺序：/passengers/:id/idcard-cache 比 /passengers 更具体，不会冲突
		adminGroup.GET("/idcard-verify/stats", idCardVerifyHandler.Stats)
		adminGroup.POST("/idcard-verify/stats/reset", idCardVerifyHandler.ResetStats)
		adminGroup.DELETE("/passengers/:id/idcard-cache", idCardVerifyHandler.InvalidatePassengerCache)

		// 轮播图管理
		adminGroup.GET("/banners", bannerHandler.List)
		adminGroup.POST("/banners", bannerHandler.Create)
		adminGroup.PUT("/banners/:id", bannerHandler.Update)
		adminGroup.DELETE("/banners/:id", bannerHandler.Delete)

		// 首页装修
		adminGroup.GET("/homepage/layout", designHandler.GetLayout)
		adminGroup.PUT("/homepage/layout", middleware.RequireSuperAdmin(), designHandler.UpdateLayout)

		// 多页面装修（订单页标签/我的页订单分类/我的页功能菜单）
		adminGroup.GET("/design/page-layout", designHandler.GetPageLayout)
		adminGroup.PUT("/design/page-layout", middleware.RequireSuperAdmin(), designHandler.UpdatePageLayout)

		// 系统配置（仅超级管理员可修改，普通管理员可查看）
		adminGroup.GET("/config", configHandler.Get)
		adminGroup.PUT("/config", middleware.RequireSuperAdmin(), configHandler.Update)

		// 管理员管理（仅超级管理员可操作）
		superAdminGroup := adminGroup.Group("", middleware.RequireSuperAdmin())
		{
			superAdminGroup.GET("/admins", adminUserHandler.List)
			superAdminGroup.POST("/admins", adminUserHandler.Create)
			superAdminGroup.PUT("/admins/:id", adminUserHandler.Update)
			superAdminGroup.PUT("/admins/:id/reset-password", adminUserHandler.ResetPassword)
			superAdminGroup.DELETE("/admins/:id", adminUserHandler.Delete)
		}

		// 操作日志（仅超级管理员可查看，防止普通管理员窥探其他管理员操作记录）
		adminGroup.GET("/logs", middleware.RequireSuperAdmin(), adminUserHandler.Logs)
		adminGroup.GET("/logs/export", middleware.RequireSuperAdmin(), adminUserHandler.LogsExport) // 导出操作日志CSV

		// 退款记录
		adminGroup.GET("/refunds", configHandler.RefundList)

		// 优惠券管理
		adminGroup.GET("/coupons", couponHandler.List)
		adminGroup.POST("/coupons", couponHandler.Create)
		adminGroup.PUT("/coupons/:id", couponHandler.Update)
		adminGroup.DELETE("/coupons/:id", couponHandler.Delete)

		// 发放记录
		adminGroup.GET("/user-coupons", userCouponHandler.List)

		// 积分规则
		adminGroup.GET("/point-rules", pointRuleHandler.List)
		adminGroup.POST("/point-rules", pointRuleHandler.Create)
		adminGroup.PUT("/point-rules/:id", pointRuleHandler.Update)
		adminGroup.DELETE("/point-rules/:id", pointRuleHandler.Delete)

		// 用户积分
		adminGroup.GET("/user-points", userPointsHandler.List)
		adminGroup.GET("/user-points/:id/records", userPointsHandler.Records)
		adminGroup.POST("/user-points/:id/adjust", middleware.RequireSuperAdmin(), userPointsHandler.Adjust)

		// 文件上传
		adminGroup.POST("/upload", uploadHandler.Upload)

		// 司机管理（CRUD需超管权限，/all和/availability供班次管理时选择司机用）
		adminGroup.GET("/drivers", middleware.RequireSuperAdmin(), driverHandler.List)
		adminGroup.GET("/drivers/all", driverHandler.All) // 不分页全部司机（下拉选择用，普通管理员可访问）
		adminGroup.POST("/drivers", middleware.RequireSuperAdmin(), driverHandler.Create)
		adminGroup.PUT("/drivers/:id", middleware.RequireSuperAdmin(), driverHandler.Update)
		adminGroup.DELETE("/drivers/:id", middleware.RequireSuperAdmin(), driverHandler.Delete)
		adminGroup.GET("/drivers/availability", driverHandler.DriverAvailability) // 司机可用性查询（班次分配用）
		adminGroup.GET("/drivers/verify-stats", middleware.RequireSuperAdmin(), driverHandler.VerifyStats)
		adminGroup.GET("/trips/:id/passengers", driverHandler.TripPassengers)                                  // 班次乘客名单（班次管理用）
		adminGroup.PUT("/trips/:id/assign-driver", middleware.RequireSuperAdmin(), driverHandler.AssignDriver) // 分配司机（班次管理用，超管权限）

		// AI 数字员工
		adminGroup.POST("/ai/chat", middleware.RateLimit(), aiHandler.Chat)
		adminGroup.GET("/ai/config", middleware.RequireSuperAdmin(), aiHandler.GetConfig)
		adminGroup.PUT("/ai/config", middleware.RequireSuperAdmin(), aiHandler.UpdateConfig)
		adminGroup.GET("/ai/models", aiHandler.GetModels)
		adminGroup.PUT("/ai/model", middleware.RequireSuperAdmin(), aiHandler.SwitchModel)
		adminGroup.POST("/ai/image", middleware.RateLimit(), aiHandler.GenerateImage)

		// 保险公司配置（通用保险对接框架，仅超级管理员可操作）
		adminGroup.GET("/insurance-providers", middleware.RequireSuperAdmin(), insuranceProviderHandler.List)
		adminGroup.POST("/insurance-providers", middleware.RequireSuperAdmin(), insuranceProviderHandler.Create)
		adminGroup.PUT("/insurance-providers/:id", middleware.RequireSuperAdmin(), insuranceProviderHandler.Update)
		adminGroup.DELETE("/insurance-providers/:id", middleware.RequireSuperAdmin(), insuranceProviderHandler.Delete)
		adminGroup.PUT("/insurance-providers/:id/activate", middleware.RequireSuperAdmin(), insuranceProviderHandler.Activate)
	}

	// 小程序端（司机核销）
	wxGroup := r.Group("/api/wx")
	{
		wxGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "狸猫日志售票系统 - 小程序接口"})
		})

		// 司机登录（公开接口，加限流防刷）
		wxGroup.POST("/driver/login", middleware.RateLimit(), wxDriverHandler.Login)

		// 司机鉴权接口
		wxDriver := wxGroup.Group("", middleware.DriverAuth(db))
		{
			wxDriver.GET("/driver/trips", wxDriverHandler.Trips)
			wxDriver.GET("/driver/trips/:id/passengers", wxDriverHandler.TripPassengers)
			wxDriver.GET("/driver/trips/:id/station-stats", wxDriverHandler.TripStationStats)
			// 核销接口加限流，防获取Token后暴力枚举订单号
			wxDriver.POST("/driver/verify", middleware.RateLimit(), wxDriverHandler.Verify)
			wxDriver.POST("/driver/verify-by-no", middleware.RateLimit(), wxDriverHandler.VerifyByOrderNo)
			wxDriver.POST("/driver/location", wxDriverHandler.ReportLocation)
			wxDriver.PUT("/driver/trips/:id/start", wxDriverHandler.StartTrip)
			wxDriver.PUT("/driver/trips/:id/end", wxDriverHandler.EndTrip)
			wxDriver.PUT("/driver/trips/:id/mark-station", wxDriverHandler.MarkStation)
			wxDriver.POST("/driver/logout", wxDriverHandler.Logout)
		}

		qrcodeHandler := wxhandler.NewQrcodeHandler(db)

		// 微信支付回调（公开接口，微信服务器调用）
		r.POST("/api/wx/pay/notify", wxUserHandler.PayNotify)
		// 微信退款回调（公开接口，微信服务器调用）
		r.POST("/api/wx/refund/notify", wxUserHandler.RefundNotify)

		// 小程序用户端接口（需用户鉴权）
		wxUser := wxGroup.Group("", middleware.WxUserAuth(db))
		{
			wxUser.POST("/logout", wxUserHandler.Logout)
			wxUser.DELETE("/account", wxUserHandler.DeleteAccount) // 用户注销账号
			wxUser.GET("/user", wxUserHandler.UserInfo)
			wxUser.PUT("/user", wxUserHandler.UpdateUser)
			wxUser.POST("/upload", wxUserHandler.Upload)
			wxUser.POST("/orders", wxUserHandler.CreateOrder)
			wxUser.GET("/orders", wxUserHandler.OrderList)
			wxUser.GET("/orders/stats", wxUserHandler.OrderStats)
			wxUser.GET("/orders/:id", wxUserHandler.OrderDetail)
			wxUser.GET("/trips/:id/location", wxUserHandler.TripLocation)
			wxUser.GET("/trips/:id/seats", middleware.RateLimit(), wxUserHandler.TripSeatMap)
			wxUser.GET("/order/:order_no", qrcodeHandler.OrderInfo)
			wxUser.GET("/qrcode/:order_no", qrcodeHandler.Generate)
			wxUser.POST("/orders/:id/pay", wxUserHandler.PayOrder)
			wxUser.POST("/orders/:id/cancel", wxUserHandler.CancelOrder)
			wxUser.POST("/orders/:id/refund", wxUserHandler.RefundOrder)
			wxUser.POST("/orders/:id/change", wxUserHandler.ChangeOrder) // 改签（同线路换班次）
			wxUser.POST("/orders/:id/hide", wxUserHandler.HideOrder) // 用户隐藏/删除订单（软删除）
			wxUser.GET("/passengers", wxUserHandler.PassengerList)
			wxUser.POST("/passengers", wxUserHandler.PassengerCreate)
			wxUser.PUT("/passengers/:id", wxUserHandler.PassengerUpdate)
			wxUser.DELETE("/passengers/:id", wxUserHandler.PassengerDelete)

			// 优惠券
			wxUser.GET("/coupons", wxUserHandler.UserCoupons)
			wxUser.GET("/coupons/available", wxUserHandler.AvailableCoupons)
			wxUser.POST("/coupons/claim", wxUserHandler.ClaimCoupon)

			// 积分
			wxUser.GET("/points", wxUserHandler.GetPoints)
			wxUser.GET("/points/records", wxUserHandler.GetPointRecords)

			// 订阅消息（用户授权上报 + 模板列表查询）
			wxUser.GET("/subscribe/templates", wxUserHandler.SubscribeTemplates)
			wxUser.POST("/subscribe/report", wxUserHandler.SubscribeReport)

			// 货物托运
			wxUser.POST("/cargo/preview", wxCargoHandler.CargoFeePreview)
			wxUser.POST("/cargo", wxCargoHandler.CreateCargoOrder)
		}

		// 小程序管理员端接口（仅管理员可见的管理后台入口）
		// 纯增量：复用现有 admin handler，不动 /admin/* 路由，网页后台行为完全不受影响。
		// 鉴权链：WxUserAuth（验 wx token、写 user_id）→ WxAdminAuth（按手机号匹配 admin、写 admin_id/name/role）。
		// 写操作（创建/修改/删除/退款等）按阶段逐步开放，避免一次性暴露过大攻击面。
		wxAdmin := wxGroup.Group("/admin", middleware.WxUserAuth(db), middleware.WxAdminAuth(db))
		{
			// 管理员信息（含 role，前端据此隐藏超管专属功能）
			wxAdmin.GET("/profile", authHandler.Profile)

			// 首页统计看板
			wxAdmin.GET("/dashboard", dashboardHandler.Stats)

			// 订单管理（只读 + 状态流转；退款需超管）
			wxAdmin.GET("/orders", orderHandler.List)
			wxAdmin.GET("/orders/:id", orderHandler.Detail)
			wxAdmin.PUT("/orders/:id/status", orderHandler.UpdateStatus)
			wxAdmin.POST("/orders/:id/refund", middleware.RequireSuperAdmin(), orderHandler.Refund)

			// 售票主数据（列表只读，编辑能力开放给超管，复用现有 handler）
			wxAdmin.GET("/stations", stationHandler.List)
			wxAdmin.GET("/stations/all", stationHandler.All)
			wxAdmin.GET("/routes", routeHandler.List)
			wxAdmin.GET("/routes/all", routeHandler.All)
			wxAdmin.GET("/routes/:id/stations", routeHandler.Stations)
			wxAdmin.GET("/trips", tripHandler.List)
			wxAdmin.GET("/vehicles", vehicleHandler.List)
			wxAdmin.GET("/vehicles/all", vehicleHandler.All)

			// 售票主数据写操作（仅超管，与网页后台权限一致）
			wxAdminWrite := wxAdmin.Group("", middleware.RequireSuperAdmin())
			{
				// 站点
				wxAdminWrite.POST("/stations", stationHandler.Create)
				wxAdminWrite.PUT("/stations/:id", stationHandler.Update)
				wxAdminWrite.DELETE("/stations/:id", stationHandler.Delete)
				// 线路
				wxAdminWrite.POST("/routes", routeHandler.Create)
				wxAdminWrite.PUT("/routes/:id", routeHandler.Update)
				wxAdminWrite.DELETE("/routes/:id", routeHandler.Delete)
				// 车辆
				wxAdminWrite.POST("/vehicles", vehicleHandler.Create)
				wxAdminWrite.PUT("/vehicles/:id", vehicleHandler.Update)
				wxAdminWrite.DELETE("/vehicles/:id", vehicleHandler.Delete)
				// 班次
				wxAdminWrite.POST("/trips", tripHandler.Create)
				wxAdminWrite.PUT("/trips/:id", tripHandler.Update)
				wxAdminWrite.DELETE("/trips/:id", tripHandler.Delete)
				wxAdminWrite.POST("/trips/batch", tripHandler.BatchCreate)
				wxAdminWrite.POST("/trips/cleanup", tripHandler.CleanupHistory)
				wxAdminWrite.PUT("/trips/:id/assign-driver", driverHandler.AssignDriver)
			}

			// 乘客名单（与网页后台权限一致，管理员只读）
			wxAdmin.GET("/trips/:id/passengers", driverHandler.TripPassengers)

			// 司机管理（列表需超管，与网页后台权限一致；/all 供班次选择用）
			wxAdmin.GET("/drivers", middleware.RequireSuperAdmin(), driverHandler.List)
			wxAdmin.GET("/drivers/all", driverHandler.All)
			// 司机写操作（仅超管）
			wxAdmin.POST("/drivers", middleware.RequireSuperAdmin(), driverHandler.Create)
			wxAdmin.PUT("/drivers/:id", middleware.RequireSuperAdmin(), driverHandler.Update)
			wxAdmin.DELETE("/drivers/:id", middleware.RequireSuperAdmin(), driverHandler.Delete)

			// 用户管理（只读 + 封禁）
			wxAdmin.GET("/users", userHandler.List)
			wxAdmin.GET("/users/:id", userHandler.Detail)
			wxAdmin.PUT("/users/:id/status", userHandler.UpdateStatus)

			// 营销（优惠券/积分规则列表只读，写操作开放给超管；发放记录/用户积分只读）
			wxAdmin.GET("/coupons", couponHandler.List)
			wxAdmin.GET("/user-coupons", userCouponHandler.List)
			wxAdmin.GET("/point-rules", pointRuleHandler.List)
			wxAdmin.GET("/user-points", userPointsHandler.List)
			wxAdmin.GET("/user-points/:id/records", userPointsHandler.Records)
			// 营销写操作（仅超管）
			wxAdmin.POST("/coupons", middleware.RequireSuperAdmin(), couponHandler.Create)
			wxAdmin.PUT("/coupons/:id", middleware.RequireSuperAdmin(), couponHandler.Update)
			wxAdmin.DELETE("/coupons/:id", middleware.RequireSuperAdmin(), couponHandler.Delete)
			wxAdmin.POST("/point-rules", middleware.RequireSuperAdmin(), pointRuleHandler.Create)
			wxAdmin.PUT("/point-rules/:id", middleware.RequireSuperAdmin(), pointRuleHandler.Update)
			wxAdmin.DELETE("/point-rules/:id", middleware.RequireSuperAdmin(), pointRuleHandler.Delete)
			wxAdmin.POST("/user-points/:id/adjust", middleware.RequireSuperAdmin(), userPointsHandler.Adjust)

			// 退款记录
			wxAdmin.GET("/refunds", configHandler.RefundList)

			// 轨迹可视化（实时监控 + 历史回放）
			wxAdmin.GET("/trips/active", trackHandler.ActiveTrips)
			wxAdmin.GET("/trips/:id/track", trackHandler.TripTrack)

			// 数字员工（AI 聊天，加限流防滥用；切换模型为全局写操作，仅超管）
			wxAdmin.POST("/ai/chat", middleware.RateLimit(), aiHandler.Chat)
			wxAdmin.GET("/ai/models", aiHandler.GetModels)
			wxAdmin.PUT("/ai/model", middleware.RequireSuperAdmin(), aiHandler.SwitchModel)
			wxAdmin.POST("/ai/image", middleware.RateLimit(), aiHandler.GenerateImage)

			// 保险公司配置（只读列表，写操作需超管，与网页后台权限一致）
			wxAdmin.GET("/insurance-providers", middleware.RequireSuperAdmin(), insuranceProviderHandler.List)
		}
	}

	return r
}
