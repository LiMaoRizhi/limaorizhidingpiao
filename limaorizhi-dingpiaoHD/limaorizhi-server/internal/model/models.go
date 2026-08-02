// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package model

import (
	"time"

	"gorm.io/gorm"
)

// 兼容别名，确保旧代码中直接使用 time.Time 的地方仍可编译
// 新代码应使用 JSONTime/JSONDate

// AdminUser 管理员
type AdminUser struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Username           string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash       string     `gorm:"size:255;not null" json:"-"`
	RealName           string     `gorm:"size:64;default:''" json:"real_name"`
	Phone              string     `gorm:"size:20;default:''" json:"phone"`
	AvatarURL          string     `gorm:"size:512;default:''" json:"avatar_url"`
	Role               int8       `gorm:"default:2" json:"role"`   // 1=超级管理员 2=普通管理员
	Status             int8       `gorm:"default:1" json:"status"` // 1=启用 0=禁用
	MustChangePassword bool       `gorm:"default:false" json:"-"`  // 是否必须在下次登录后修改密码（默认管理员口令等）
	TokenInvalidBefore *time.Time `json:"-"`                       // 登出失效：早于此时间签发的Token视为已踢出（重启不丢失）
	LastLoginAt        *JSONTime  `json:"last_login_at"`
	CreatedAt          JSONTime   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          JSONTime   `gorm:"autoUpdateTime" json:"updated_at"`
}

// User 小程序用户
type User struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	OpenID             string     `gorm:"size:64;uniqueIndex;not null" json:"openid"`
	UnionID            string     `gorm:"size:64;default:''" json:"unionid"`
	Nickname           string     `gorm:"size:64;default:''" json:"nickname"`
	AvatarURL          string     `gorm:"size:512;default:''" json:"avatar_url"`
	Phone              string     `gorm:"size:20;index;default:''" json:"phone"`
	Status             int8       `gorm:"default:1" json:"status"` // 1=正常 0=封禁 2=已注销
	TokenInvalidBefore *time.Time `json:"-"`                       // 登出失效：早于此时间签发的Token视为已踢出（重启不丢失）
	CreatedAt          JSONTime   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          JSONTime   `gorm:"autoUpdateTime" json:"updated_at"`
}

// Station 站点
type Station struct {
	ID        uint     `gorm:"primaryKey" json:"id"`
	Name      string   `gorm:"size:64;not null" json:"name"`
	Pinyin    string   `gorm:"size:128;default:''" json:"pinyin"`
	SortOrder int      `gorm:"default:0" json:"sort_order"`
	Longitude float64  `gorm:"type:decimal(10,6);default:0" json:"longitude"` // 经度
	Latitude  float64  `gorm:"type:decimal(10,6);default:0" json:"latitude"`  // 纬度
	Status    int8     `gorm:"default:1" json:"status"`                       // 1=启用 0=禁用
	CreatedAt JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// Route 线路
// 注：from_station_id/to_station_id 为首站/末站冗余缓存（由 route_stations 派生），仅用于列表展示；
// 售票/区间判断一律基于 route_stations 站点序列。
// RouteType: 1=城乡公交（多站点途经，如商丘→界沟镇沿途各站）
//            2=城际客运（跨城市，如商丘→郑州途经开封）
//            3=旅游专线（A地直达B地，如景区直达）
type Route struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"size:128;not null" json:"name"`
	RouteType       int8           `gorm:"default:1" json:"route_type"` // 1=城乡公交 2=城际客运 3=旅游专线
	FromStationID   uint           `gorm:"not null" json:"from_station_id"`
	ToStationID     uint           `gorm:"not null" json:"to_station_id"`
	FromStation     *Station       `gorm:"foreignKey:FromStationID" json:"from_station,omitempty"`
	ToStation       *Station       `gorm:"foreignKey:ToStationID" json:"to_station,omitempty"`
	RouteStations   []RouteStation `gorm:"foreignKey:RouteID" json:"route_stations,omitempty"`
	DistanceKM      float64        `gorm:"type:decimal(6,1);default:0" json:"distance_km"`
	DurationMinutes int            `gorm:"default:0" json:"duration_minutes"`
	MinFare         float64        `gorm:"type:decimal(8,2);default:0" json:"min_fare"` // 起步价（最低票价），0=不启用；实际票价=max(起步价, 下车站price-上车站price)
	Status          int8           `gorm:"default:1" json:"status"`                     // 1=运营 0=停运
	CreatedAt       JSONTime       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       JSONTime       `gorm:"autoUpdateTime" json:"updated_at"`
}

// RouteStation 线路站点序列（一条线路经过的有序站点 + 到站票价 + 累计里程）
// 区间票价 = 下车站 Price − 上车站 Price；区间里程 = 下车站 DistanceKM − 上车站 DistanceKM
type RouteStation struct {
	ID               uint     `gorm:"primaryKey" json:"id"`
	RouteID          uint     `gorm:"not null;index:idx_route_stations_route;uniqueIndex:idx_route_station_order;uniqueIndex:idx_route_station_sid" json:"route_id"`
	Route            *Route   `gorm:"foreignKey:RouteID" json:"route,omitempty"`
	StationID        uint     `gorm:"not null;uniqueIndex:idx_route_station_sid" json:"station_id"`
	Station          *Station `gorm:"foreignKey:StationID" json:"station,omitempty"`
	StopOrder        int      `gorm:"not null;uniqueIndex:idx_route_station_order" json:"stop_order"` // 第几站，从1开始
	DistanceKM       float64  `gorm:"type:decimal(8,2);default:0" json:"distance_km"`                 // 从起点到该站累计里程（托运运费用）
	Price            float64  `gorm:"type:decimal(8,2);default:0" json:"price"`                       // 从起点到该站的票价
	ArrivalTime      string   `gorm:"type:varchar(8);default:''" json:"arrival_time"`                 // 该站到达时刻(如08:35)，发车后按站下单判断；空则走里程推算
	ArrivalDayOffset int      `gorm:"default:0" json:"arrival_day_offset"`                            // 该站到达相对发车日的天数偏移(0=当天,1=次日...)，跨省长途跨天班次用
	CreatedAt        JSONTime `gorm:"autoCreateTime" json:"created_at"`
}

// Vehicle 车辆
type Vehicle struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	PlateNo     string   `gorm:"size:20;uniqueIndex;not null" json:"plate_no"`
	VehicleType string   `gorm:"size:32;default:''" json:"vehicle_type"`
	SeatCount   int      `gorm:"default:0" json:"seat_count"`
	Status      int8     `gorm:"default:1" json:"status"` // 1=可用 0=维修
	CreatedAt   JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// Driver 司机
type Driver struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	EmployeeNo         string     `gorm:"size:32;index;default:''" json:"employee_no"` // 工号（可选，用于区分重名司机）
	Name               string     `gorm:"size:64;not null" json:"name"`
	Phone              string     `gorm:"size:20;uniqueIndex;not null" json:"phone"`
	PasswordHash       string     `gorm:"size:255;not null" json:"-"`
	LicenseNo          string     `gorm:"size:32;default:''" json:"license_no"` // 驾驶证号
	Status             int8       `gorm:"default:1" json:"status"`              // 1=启用 0=禁用
	TokenInvalidBefore *time.Time `json:"-"`                                    // 登出失效：早于此时间签发的Token视为已踢出（重启不丢失）
	LastLoginAt        *JSONTime  `json:"last_login_at"`
	CreatedAt          JSONTime   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          JSONTime   `gorm:"autoUpdateTime" json:"updated_at"`
}

// Trip 班次(车次)
type Trip struct {
	ID                 uint     `gorm:"primaryKey" json:"id"`
	RouteID            uint     `gorm:"not null;index:idx_route_date" json:"route_id"`
	Route              *Route   `gorm:"foreignKey:RouteID" json:"route,omitempty"`
	VehicleID          uint     `gorm:"not null" json:"vehicle_id"`
	Vehicle            *Vehicle `gorm:"foreignKey:VehicleID" json:"vehicle,omitempty"`
	DriverID           uint     `gorm:"default:0;index" json:"driver_id"`
	Driver             *Driver  `gorm:"foreignKey:DriverID" json:"driver,omitempty"`
	TripNo             string   `gorm:"size:32;not null;uniqueIndex:idx_trips_trip_no" json:"trip_no"` // 班次号唯一约束，防止重复
	TripDate           JSONDate `gorm:"type:date;not null;index:idx_route_date" json:"trip_date"`
	DepartureTime      string   `gorm:"type:varchar(8);not null" json:"departure_time"`
	ArrivalTime        string   `gorm:"type:varchar(8);not null" json:"arrival_time"`
	ArrivalDayOffset   int      `gorm:"default:0" json:"arrival_day_offset"` // 终点到达相对发车日的天数偏移(0=当天,1=次日...)，跨省长途跨天班次用
	TotalSeats         int      `gorm:"not null" json:"total_seats"`
	AvailableSeats     int      `gorm:"not null" json:"available_seats"` // 初始可售座位数（创建时=TotalSeats）。实际区间可用座位由AvailableSeatsForSegment实时计算，此字段不随订单变化更新
	BasePrice          float64  `gorm:"type:decimal(8,2);not null" json:"base_price"`
	Status             int8     `gorm:"default:1;index:idx_date_status" json:"status"` // 1=可售 2=已发车 3=已取消 0=下架 4=已完成
	CurrentPassedOrder int      `gorm:"default:0" json:"current_passed_order"`         // 手动标记车已驶过到第几站序(0=未标记)，>0时覆盖其他到站判断
	CreatedAt          JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// Order 订单（统一订单：车票+托运）
type Order struct {
	ID                  uint     `gorm:"primaryKey" json:"id"`
	OrderNo             string   `gorm:"size:32;uniqueIndex;not null" json:"order_no"`
	OrderType           int8     `gorm:"default:1;index" json:"order_type"` // 1=车票 2=托运
	UserID              uint     `gorm:"not null;index" json:"user_id"`
	User                *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TripID              uint     `gorm:"not null;index" json:"trip_id"`
	Trip                *Trip    `gorm:"foreignKey:TripID" json:"trip,omitempty"`
	RouteID             uint     `gorm:"not null" json:"route_id"`
	FromStationID       uint     `gorm:"not null" json:"from_station_id"`
	FromStation         *Station `gorm:"foreignKey:FromStationID" json:"from_station,omitempty"`
	FromStationName     string   `gorm:"size:64;default:''" json:"from_station_name"` // 冗余存储上车站名（删除线路/站点后仍可显示）
	ToStationID         uint     `gorm:"not null" json:"to_station_id"`
	ToStation           *Station `gorm:"foreignKey:ToStationID" json:"to_station,omitempty"`
	ToStationName       string   `gorm:"size:64;default:''" json:"to_station_name"` // 冗余存储下车站名
	TripDate            JSONDate `gorm:"type:date;not null" json:"trip_date"`
	DepartureTime       string   `gorm:"type:varchar(8);not null" json:"departure_time"`
	PassengerCount      int      `gorm:"default:1" json:"passenger_count"`
	TotalPrice          float64  `gorm:"type:decimal(10,2);not null" json:"total_price"`
	InsuranceFee        float64  `gorm:"type:decimal(10,2);default:0" json:"insurance_fee"` // 保险费小计（单价×乘客数，0=未购买保险）
	InsuranceProviderID uint     `gorm:"default:0;index" json:"insurance_provider_id"`      // 出单保险公司ID（0=未配置保险公司或未购买）
	InsurancePolicyNo   string   `gorm:"size:64;default:''" json:"insurance_policy_no"`     // 保单号（保险公司出单后回填，空=未出单）
	Status              int8     `gorm:"default:0;index" json:"status"`
	// 车票字段
	ContactName  string `gorm:"size:64;default:''" json:"contact_name"`
	ContactPhone string `gorm:"size:20;default:''" json:"contact_phone"`
	// 托运字段（order_type=2时使用）
	SenderName    string    `gorm:"size:64;default:''" json:"sender_name"`
	SenderPhone   string    `gorm:"size:20;default:''" json:"sender_phone"`
	ReceiverName  string    `gorm:"size:64;default:''" json:"receiver_name"`
	ReceiverPhone string    `gorm:"size:20;default:''" json:"receiver_phone"`
	CargoType     string    `gorm:"size:32;default:''" json:"cargo_type"`
	Weight        float64   `gorm:"type:decimal(8,2);default:0" json:"weight"`
	Description   string    `gorm:"size:255;default:''" json:"description"`
	PayTime       *JSONTime `json:"pay_time"`
	PayMethod     string    `gorm:"size:16;default:''" json:"pay_method"`
	UserHidden    bool      `gorm:"default:false" json:"user_hidden"` // 用户端软删除：true=用户已删除（仅前端隐藏，管理端仍可见）
	CreatedAt     JSONTime  `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt     JSONTime  `gorm:"autoUpdateTime" json:"updated_at"`
}

// OrderPassenger 订单乘客
// OrderID 通过外键约束关联到 orders 表，确保乘客记录始终引用有效订单
type OrderPassenger struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	OrderID    uint   `gorm:"not null;index;constraint:OnDelete:CASCADE" json:"order_id"`
	Order      *Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Name       string `gorm:"size:64;not null" json:"name"`
	IDCardType int8   `gorm:"default:1" json:"id_card_type"` // 1=身份证 2=护照
	// 身份证号加密存储（AES-256-GCM），读取时自动解密。API响应仍需脱敏。
	IDCardNo    string    `gorm:"size:128;not null" json:"id_card_no"`
	Phone       string    `gorm:"size:20;default:''" json:"phone"`
	SeatNo      string    `gorm:"size:8;default:''" json:"seat_no"`
	CheckStatus int8      `gorm:"default:0" json:"check_status"` // 0=未核销 1=已核销
	CheckedAt   *JSONTime `json:"checked_at"`
	CheckedBy   uint      `gorm:"default:0" json:"checked_by"` // 司机ID
	CreatedAt   JSONTime  `gorm:"autoCreateTime" json:"created_at"`
}

// Passenger 常用乘客
type Passenger struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"not null;index" json:"user_id"`
	Name       string `gorm:"size:64;not null" json:"name"`
	IDCardType int8   `gorm:"default:1" json:"id_card_type"`
	// 身份证号加密存储（AES-256-GCM），读取时自动解密。API响应仍需脱敏。
	IDCardNo  string   `gorm:"size:128;not null" json:"id_card_no"`
	Phone     string   `gorm:"size:20;default:''" json:"phone"`
	CreatedAt JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// Payment 支付记录
type Payment struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	OrderID       uint      `gorm:"not null;index" json:"order_id"`
	PaymentNo     string    `gorm:"size:64;uniqueIndex;not null" json:"payment_no"`
	TransactionID string    `gorm:"size:64;default:''" json:"transaction_id"`
	Amount        float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Method        string    `gorm:"size:32;default:微信支付" json:"method"`
	Status        int8      `gorm:"default:0" json:"status"` // 0=待支付 1=成功 2=失败 3=已退款
	PayTime       *JSONTime `json:"pay_time"`
	CreatedAt     JSONTime  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     JSONTime  `gorm:"autoUpdateTime" json:"updated_at"`
}

// Refund 退款记录
type Refund struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	OrderID    uint      `gorm:"not null;index" json:"order_id"`
	Order      *Order    `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	RefundNo   string    `gorm:"size:64;uniqueIndex;not null" json:"refund_no"`
	Amount     float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Reason     string    `gorm:"size:255;default:''" json:"reason"`
	Status     int8      `gorm:"default:0" json:"status"`     // 0=处理中 1=成功 2=失败
	PreStatus  int8      `gorm:"default:0" json:"pre_status"` // 退款前订单状态（用于退款失败回滚：1=待出行 2=已完成 4=已取消）
	RefundTime *JSONTime `json:"refund_time"`
	CreatedAt  JSONTime  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  JSONTime  `gorm:"autoUpdateTime" json:"updated_at"`
}

// Banner 轮播图
type Banner struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	Title       string   `gorm:"size:128;default:''" json:"title"`
	TitleColor  string   `gorm:"size:32;default:''" json:"title_color"` // 标题颜色: white/black/yellow/red/blue/green/cyan/purple
	TitleEffect int8     `gorm:"default:0" json:"title_effect"`         // 标题特效: 0=无 1=阴影 2=磨砂玻璃
	ImageURL    string   `gorm:"size:512;not null" json:"image_url"`
	LinkType    int8     `gorm:"default:0" json:"link_type"` // 0=无 1=车次 2=外链
	LinkValue   string   `gorm:"size:512;default:''" json:"link_value"`
	SortOrder   int      `gorm:"default:0" json:"sort_order"`
	Status      int8     `gorm:"default:1" json:"status"` // 1=显示 0=隐藏
	CreatedAt   JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	ConfigKey   string   `gorm:"size:64;uniqueIndex;not null" json:"config_key"`
	ConfigValue string   `gorm:"type:text" json:"config_value"`
	Description string   `gorm:"size:255;default:''" json:"description"`
	UpdatedAt   JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// InsuranceProvider 保险公司配置（通用保险对接框架）
// 上线时由超级管理员在管理端填入要用哪家保险公司的配置即可对接出真实保单；
// 同一时刻仅允许一家 is_active=true，切换启用时其余自动置 false。
// AppSecret 在 API 响应中脱敏（仅返回 app_secret_masked），不出现在公开配置中。
type InsuranceProvider struct {
	ID              uint     `gorm:"primaryKey" json:"id"`
	Name            string   `gorm:"size:64;not null" json:"name"`                                       // 保险公司名称
	APIURL          string   `gorm:"size:512;not null" json:"api_url"`                                   // 出单API地址（POST JSON）
	AppID           string   `gorm:"size:64;not null" json:"app_id"`                                     // 商户号/应用ID
	AppSecret       string   `gorm:"size:255;not null" json:"-"`                                         // 商户密钥（不返回明文）
	AppSecretMasked string   `gorm:"-" json:"app_secret_masked,omitempty"`                               // 脱敏后的密钥（仅展示用）
	ProductCode     string   `gorm:"size:64;default:''" json:"product_code"`                             // 保险产品代码
	Fee             float64  `gorm:"type:decimal(8,2);default:0" json:"fee"`                             // 保险费单价(元/人)
	IsActive        bool     `gorm:"default:false;index:idx_insurance_provider_active" json:"is_active"` // 是否当前启用（同时间仅一家）
	Required        bool     `gorm:"default:false" json:"required"`                                      // 是否强制购买
	Remark          string   `gorm:"size:255;default:''" json:"remark"`                                  // 备注
	CreatedAt       JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// OperationLog 操作日志
type OperationLog struct {
	ID        uint     `gorm:"primaryKey" json:"id"`
	AdminID   uint     `gorm:"not null;index" json:"admin_id"`
	AdminName string   `gorm:"size:64;not null" json:"admin_name"`
	Module    string   `gorm:"size:32;not null" json:"module"`
	Action    string   `gorm:"size:32;not null" json:"action"`
	Target    string   `gorm:"size:128;default:''" json:"target"`
	Detail    string   `gorm:"type:text" json:"detail"`
	IPAddress string   `gorm:"size:64;default:''" json:"ip_address"`
	CreatedAt JSONTime `gorm:"autoCreateTime;index" json:"created_at"`
}

// VehicleLocation 班次车辆实时位置（司机上报GPS，供乘客查看车辆位置）
type VehicleLocation struct {
	ID         uint     `gorm:"primaryKey" json:"id"`
	TripID     uint     `gorm:"index;not null" json:"trip_id"`              // 关联班次
	VehicleID  uint     `gorm:"index" json:"vehicle_id"`                    // 关联车辆
	DriverID   uint     `gorm:"index" json:"driver_id"`                     // 上报司机
	Longitude  float64  `gorm:"type:decimal(10,6)" json:"longitude"`        // 经度(gcj02)
	Latitude   float64  `gorm:"type:decimal(10,6)" json:"latitude"`         // 纬度(gcj02)
	Speed      float64  `gorm:"type:decimal(6,1);default:0" json:"speed"`   // 速度m/s（微信API返回m/s，前端展示时×3.6转km/h）
	Heading    float64  `gorm:"type:decimal(5,1);default:0" json:"heading"` // 方向角
	ReportedAt JSONTime `json:"reported_at"`                                // 上报时间
	CreatedAt  JSONTime `gorm:"autoCreateTime" json:"created_at"`
}

// 营销模块

// Coupon 优惠券模板
// 1=满减券(满X减Y元) 2=折扣券(打X折) 3=固定金额券(直接抵扣X元)
type Coupon struct {
	ID            uint     `gorm:"primaryKey" json:"id"`
	Name          string   `gorm:"size:128;not null" json:"name"`                     // 优惠券名称
	Type          int8     `gorm:"default:1" json:"type"`                             // 1=满减 2=折扣 3=固定金额
	DiscountValue float64  `gorm:"type:decimal(10,2);not null" json:"discount_value"` // 满减/固定金额=元, 折扣=折(如8.5)
	MinSpend      float64  `gorm:"type:decimal(10,2);default:0" json:"min_spend"`     // 最低消费门槛(元)
	ValidDays     int      `gorm:"default:30" json:"valid_days"`                      // 领取后有效天数
	TotalCount    int      `gorm:"default:0" json:"total_count"`                      // 发放总量, 0=不限
	IssuedCount   int      `gorm:"default:0" json:"issued_count"`                     // 已发放数量
	UsedCount     int      `gorm:"default:0" json:"used_count"`                       // 已使用数量
	Status        int8     `gorm:"default:1" json:"status"`                           // 1=启用 0=停用
	CreatedAt     JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// UserCoupon 用户优惠券（发放记录）
type UserCoupon struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_coupon" json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CouponID  uint      `gorm:"not null;uniqueIndex:idx_user_coupon" json:"coupon_id"`
	Coupon    *Coupon   `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
	Status    int8      `gorm:"default:0;index" json:"status"` // 0=未使用 1=已使用 2=已过期
	IssuedAt  JSONTime  `json:"issued_at"`
	ExpiredAt JSONTime  `json:"expired_at"`
	UsedAt    *JSONTime `json:"used_at"`
	OrderID   uint      `gorm:"default:0" json:"order_id"`
	CreatedAt JSONTime  `gorm:"autoCreateTime;index" json:"created_at"`
}

// PointRule 积分规则
// 1=消费赠送 2=注册赠送 3=手动调整
type PointRule struct {
	ID            uint     `gorm:"primaryKey" json:"id"`
	RuleName      string   `gorm:"size:128;not null" json:"rule_name"`                  // 规则名称
	RuleType      int8     `gorm:"default:1" json:"rule_type"`                          // 1=消费赠送 2=注册赠送 3=手动调整
	PointsPerYuan float64  `gorm:"type:decimal(10,2);default:1" json:"points_per_yuan"` // 每消费1元获得积分(消费赠送时使用)
	FixedPoints   int      `gorm:"default:0" json:"fixed_points"`                       // 固定积分(注册赠送时使用)
	Description   string   `gorm:"size:255;default:''" json:"description"`              // 规则说明
	Status        int8     `gorm:"default:1" json:"status"`                             // 1=启用 0=停用
	CreatedAt     JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// UserPoints 用户积分余额
type UserPoints struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	UserID      uint     `gorm:"uniqueIndex;not null" json:"user_id"`
	User        *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Balance     int      `gorm:"default:0" json:"balance"`      // 当前积分余额
	TotalEarned int      `gorm:"default:0" json:"total_earned"` // 累计获得
	TotalSpent  int      `gorm:"default:0" json:"total_spent"`  // 累计消耗
	UpdatedAt   JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// PointRecord 积分明细
type PointRecord struct {
	ID         uint     `gorm:"primaryKey" json:"id"`
	UserID     uint     `gorm:"not null;index" json:"user_id"`
	User       *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ChangeType int8     `gorm:"not null" json:"change_type"`      // 1=获得 2=消耗
	Points     int      `gorm:"not null" json:"points"`           // 变动积分(正数)
	Source     string   `gorm:"size:32;default:''" json:"source"` // 来源: order/manual/register
	OrderID    uint     `gorm:"default:0" json:"order_id"`
	Remark     string   `gorm:"size:255;default:''" json:"remark"`
	AdminID    uint     `gorm:"default:0" json:"admin_id"`
	AdminName  string   `gorm:"size:64;default:''" json:"admin_name"`
	CreatedAt  JSONTime `gorm:"autoCreateTime;index" json:"created_at"`
}

// SubscribeQuota 订阅消息发送配额
// 微信订阅消息机制：用户每次通过 wx.requestSubscribeMessage 授权一个模板，后端获得1次发送配额
// 用完后无法再向该用户发送该模板的消息，需用户下次操作时再次授权
// user_id + template_key 唯一约束，防止同一用户同一模板出现多条配额记录
type SubscribeQuota struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	UserID      uint     `gorm:"not null;uniqueIndex:idx_sub_quota_user_template" json:"user_id"`
	TemplateKey string   `gorm:"size:32;not null;uniqueIndex:idx_sub_quota_user_template" json:"template_key"` // payment_success, trip_departure, trip_arrival, refund_success
	Quota       int      `gorm:"default:0" json:"quota"`                                                       // 剩余可发送次数
	CreatedAt   JSONTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   JSONTime `gorm:"autoUpdateTime" json:"updated_at"`
}

// AutoMigrate 自动迁移所有表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AdminUser{},
		&User{},
		&Station{},
		&Route{},
		&RouteStation{},
		&Vehicle{},
		&Driver{},
		&Trip{},
		&Order{},
		&OrderPassenger{},
		&Passenger{},
		&Payment{},
		&Refund{},
		&Banner{},
		&SystemConfig{},
		&InsuranceProvider{},
		&OperationLog{},
		&VehicleLocation{},
		&Coupon{},
		&UserCoupon{},
		&PointRule{},
		&UserPoints{},
		&PointRecord{},
		&SubscribeQuota{},
	)
}
