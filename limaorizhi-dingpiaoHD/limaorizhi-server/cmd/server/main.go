// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
// 啊啊啊啊啊要疯了，Saas商城系统搞多了老是跑题，感觉没有Java好搞GitHub没找到有Go的大巴车订票开源的代码
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/handler/admin"
	"limaorizhi-server/internal/handler/wx"
	"limaorizhi-server/internal/model"
	"limaorizhi-server/internal/pkg/crypto"
	redis "limaorizhi-server/internal/pkg/redis"
	"limaorizhi-server/internal/router"
	"limaorizhi-server/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//go:embed all:dist
var distFS embed.FS

// extractFrontend 把打包进二进制的前端文件解压到同目录的dist/ nginx好直接读
func extractFrontend() {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("[前端] 无法获取可执行文件路径: %v", err)
		return
	}
	distDir := filepath.Join(filepath.Dir(exe), "dist")
	count := 0
	err = fs.WalkDir(distFS, "dist", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel("dist", path)
		targetPath := filepath.Join(distDir, relPath)
		os.MkdirAll(filepath.Dir(targetPath), 0755)
		content, err := distFS.ReadFile(path)
		if err != nil {
			log.Printf("[前端] 读取嵌入文件失败 %s: %v", path, err)
			return nil
		}
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			log.Printf("[前端] 写入文件失败 %s: %v", targetPath, err)
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		log.Printf("[前端] 释放前端文件失败: %v", err)
	} else {
		log.Printf("[前端] 已释放 %d 个前端文件到 %s", count, distDir)
	}
}

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		exe, _ := os.Executable()
		configPath = filepath.Join(filepath.Dir(exe), "config.yaml")
	}
	if err := config.Load(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 强制使用东八区，避免服务器时区不一致导致发车/退票等时间判断出错
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		time.Local = loc
		log.Println("[时区] 已设置为 Asia/Shanghai")
	} else {
		log.Printf("[时区] 加载 Asia/Shanghai 失败，依赖系统时区: %v", err)
	}

	cfg := config.AppConfig

	// 打印实名认证配置状态 方便排查"假名字能通过"这种破事
	log.Printf("[实名认证] enabled=%v, strict_mode=%v, appcode_configured=%v, endpoint=%s%s",
		cfg.IDCardVerify.Enabled,
		cfg.IDCardVerify.StrictMode,
		cfg.IDCardVerify.AppCode != "",
		cfg.IDCardVerify.Endpoint,
		cfg.IDCardVerify.Path)
	if cfg.IDCardVerify.Enabled && cfg.IDCardVerify.AppCode == "" {
		log.Println("[严重警告] 实名认证开了但AppCode没配！strict_mode=true时所有认证都会被拒 .env里配IDCARD_VERIFY_APPCODE")
	}
	// 生产环境强制开严格模式 没开的话自动给开了
	if cfg.Server.Mode == "release" && cfg.IDCardVerify.Enabled && !cfg.IDCardVerify.StrictMode {
		log.Println("[安全警告] 生产环境strict_mode=false API挂了会降级放行 不安全 已自动强制开启")
		cfg.IDCardVerify.StrictMode = true
	}

	// 身份证AES密钥 生产必填 开发可以不配但写身份证号会报错
	if err := crypto.InitKey(cfg.Security.IDCardAESKey); err != nil {
		log.Fatalf("初始化身份证加密密钥失败: %v", err)
	}
	if cfg.Server.Mode == "release" && !crypto.Enabled() {
		log.Fatalf("生产环境必须配IDCARD_AES_KEY 身份证不能明文存")
	}
	if !crypto.Enabled() {
		log.Println("[警告] 身份证加密密钥没配 身份证号明文存 生产环境用IDCARD_AES_KEY配")
	}

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
		if cfg.CORSOrigins == "" {
			log.Println("[警告] 生产环境没配cors_origins 跨域请求全拒 环境变量CORS_ORIGINS配")
		}
		// 没配可信代理的话拿不到真实IP
		if cfg.Server.TrustedProxies == "" {
			log.Println("[警告] 生产环境没配trusted_proxies 不信任何代理 部署在nginx后面的话ClientIP拿不到 环境变量TRUSTED_PROXIES配")
		}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.Username, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		// 不建物理外键 trips.driver_id=0是没分配司机的合法状态 强制外键迁移会挂
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("连接MySQL失败: %v\n请确认MySQL已启动且配置正确: %s", err, configPath)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库连接池失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)  // 连接最多活5分钟 防止用旧连接
	sqlDB.SetConnMaxIdleTime(3 * time.Minute) // 空闲连接3分钟回收
	log.Println("MySQL 连接成功")

	// Redis 可选的 没配或连失败就降级用进程内存
	if err := redis.Init(cfg.Redis); err != nil {
		log.Printf("Redis 初始化失败: %v", err)
	}

	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Println("数据库迁移完成")

	if err := admin.InitAdmin(db); err != nil {
		log.Printf("初始化管理员失败: %v", err)
	}
	if err := admin.InitConfig(db); err != nil {
		log.Printf("初始化系统配置失败: %v", err)
	}
	// 测试数据 只有debug模式才跑 InitTestData现在是空函数 想加自己加
	if cfg.Server.Mode == "debug" {
		if err := admin.InitTestData(db); err != nil {
			log.Printf("初始化测试数据失败: %v", err)
		}
	} else {
		log.Printf("当前模式: %s 跳过测试数据", cfg.Server.Mode)
	}

	if err := os.MkdirAll(cfg.Upload.Path, 0755); err != nil {
		log.Fatalf("创建上传目录失败: %v", err)
	}

	// 一堆定时任务
	service.StartOrderExpireChecker(db)  // 订单超时自动取消
	service.StartTripAutoCompleter(db)    // 发车后自动完成
	service.StartLocationCleaner(db)      // 车辆位置记录清理
	service.StartCouponExpireChecker(db)  // 优惠券过期标记
	service.StartLogCleaner(db)          // 操作日志清理 保留30天

	// 退款补偿兜底：PayNotify挂了导致退款记录卡在处理中 这个定时任务会补
	service.SetRefundRetryFunc(func(refund model.Refund, transactionID string) error {
		var order model.Order
		if err := db.First(&order, refund.OrderID).Error; err != nil {
			return err
		}
		_, err := wx.CreateWxRefund(order, refund.RefundNo, transactionID, refund.Amount)
		return err
	})
	// 退款对账兜底：退款回调没送达时退款记录卡"处理中"但微信侧钱已退，
	// 这个函数主动查微信退款状态，补偿任务对账时先用它判断是标记成功还是重试发起。
	service.SetRefundQueryFunc(wx.QueryRefundStatus)
	// 支付对账兜底：支付回调没送达时订单卡"待支付/已取消"但微信钱已扣，
	// 这个任务主动把这类订单拿去问微信，已支付就确认、已取消就登记退款，
	// 然后上面的退款补偿任务负责把真金白银退回去。存量卡死单就这么救。
	service.SetPayReconcileFunc(func(db *gorm.DB, order model.Order) (bool, error) {
		return wx.ReconcileOrderPaidState(db, order)
	})
	service.StartPayReconcileChecker(db)
	service.StartRefundCompensator(db)

	// 前端文件解压到dist/ nginx直接读
	extractFrontend()

	r := router.Setup(db)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("狸猫日志售票系统后端服务启动: http://localhost%s", addr)
	if cfg.Server.Mode != "release" {
		log.Printf("默认管理员: admin（首次启动时已生成随机临时密码，请查看上方安全警告输出）")
	}
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
