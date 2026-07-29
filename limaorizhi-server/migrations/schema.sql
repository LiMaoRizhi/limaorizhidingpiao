-- limaorizhi-server 数据库脚本  狸猫日志  联系微信：lihao68681818
-- 跟GORM AutoMigrate生成的表结构一样 不想等程序启动自动迁移的话手动跑这个
-- 身份证号字段AES加密 GORM Hook自动加解密 不用管

CREATE DATABASE IF NOT EXISTS limaorizhi_dingpiao DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE limaorizhi_dingpiao;

-- 管理员表
CREATE TABLE IF NOT EXISTS `admin_users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(64) NOT NULL,
  `password_hash` VARCHAR(255) NOT NULL,
  `real_name` VARCHAR(64) DEFAULT '',
  `phone` VARCHAR(20) DEFAULT '',
  `avatar_url` VARCHAR(512) DEFAULT '',
  `role` TINYINT DEFAULT 2 COMMENT '1=超级管理员 2=普通管理员',
  `status` TINYINT DEFAULT 1 COMMENT '1=启用 0=禁用',
  `must_change_password` BOOLEAN DEFAULT FALSE,
  `token_invalid_before` DATETIME NULL COMMENT '登出失效：早于此时间签发的Token视为已踢出',
  `last_login_at` DATETIME NULL,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_admin_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员';

-- 小程序用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `open_id` VARCHAR(64) NOT NULL,
  `union_id` VARCHAR(64) DEFAULT '',
  `nickname` VARCHAR(64) DEFAULT '',
  `avatar_url` VARCHAR(512) DEFAULT '',
  `phone` VARCHAR(20) DEFAULT '',
  `status` TINYINT DEFAULT 1 COMMENT '1=正常 0=封禁',
  `token_invalid_before` DATETIME NULL COMMENT '登出失效：早于此时间签发的Token视为已踢出',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_open_id` (`open_id`),
  KEY `idx_users_phone` (`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='小程序用户';

-- 站点表
CREATE TABLE IF NOT EXISTS `stations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL,
  `pinyin` VARCHAR(128) DEFAULT '',
  `sort_order` INT DEFAULT 0,
  `longitude` DECIMAL(10,6) DEFAULT 0 COMMENT '经度',
  `latitude` DECIMAL(10,6) DEFAULT 0 COMMENT '纬度',
  `status` TINYINT DEFAULT 1 COMMENT '1=启用 0=禁用',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='站点';

-- 线路表
CREATE TABLE IF NOT EXISTS `routes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `route_type` TINYINT DEFAULT 1 COMMENT '1=城乡公交 2=城际客运 3=旅游专线',
  `from_station_id` BIGINT UNSIGNED NOT NULL,
  `to_station_id` BIGINT UNSIGNED NOT NULL,
  `distance_km` DECIMAL(6,1) DEFAULT 0,
  `duration_minutes` INT DEFAULT 0,
  `min_fare` DECIMAL(8,2) DEFAULT 0 COMMENT '起步价 0=不启用',
  `status` TINYINT DEFAULT 1 COMMENT '1=运营 0=停运',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_routes_from_station_id` (`from_station_id`),
  KEY `idx_routes_to_station_id` (`to_station_id`),
  CONSTRAINT `fk_routes_from_station` FOREIGN KEY (`from_station_id`) REFERENCES `stations` (`id`),
  CONSTRAINT `fk_routes_to_station` FOREIGN KEY (`to_station_id`) REFERENCES `stations` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='线路';

-- 线路站点序列表
CREATE TABLE IF NOT EXISTS `route_stations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `route_id` BIGINT UNSIGNED NOT NULL,
  `station_id` BIGINT UNSIGNED NOT NULL,
  `stop_order` INT NOT NULL,
  `distance_km` DECIMAL(8,2) DEFAULT 0 COMMENT '从起点到该站累计里程',
  `price` DECIMAL(8,2) DEFAULT 0 COMMENT '从起点到该站的票价(区间价=下车站price-上车站price)',
  `arrival_time` VARCHAR(8) DEFAULT '' COMMENT '该站到达时刻(如08:35)，发车后按站下单判断；空则走里程推算',
  `arrival_day_offset` INT DEFAULT 0 COMMENT '该站到达相对发车日的天数偏移(0=当天,1=次日...)，跨省长途跨天班次用',
  `created_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_route_station_order` (`route_id`,`stop_order`),
  UNIQUE KEY `idx_route_station_sid` (`route_id`,`station_id`),
  KEY `idx_route_stations_route` (`route_id`),
  CONSTRAINT `fk_route_stations_route` FOREIGN KEY (`route_id`) REFERENCES `routes` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_route_stations_station` FOREIGN KEY (`station_id`) REFERENCES `stations` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='线路站点序列';

-- 车辆表
CREATE TABLE IF NOT EXISTS `vehicles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `plate_no` VARCHAR(20) NOT NULL,
  `vehicle_type` VARCHAR(32) DEFAULT '',
  `seat_count` INT DEFAULT 0,
  `status` TINYINT DEFAULT 1 COMMENT '1=可用 0=维修',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vehicles_plate_no` (`plate_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='车辆';

-- 司机表
CREATE TABLE IF NOT EXISTS `drivers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `employee_no` VARCHAR(32) DEFAULT '' COMMENT '工号（可选，用于区分重名司机）',
  `name` VARCHAR(64) NOT NULL,
  `phone` VARCHAR(20) NOT NULL,
  `password_hash` VARCHAR(255) NOT NULL,
  `license_no` VARCHAR(32) DEFAULT '',
  `status` TINYINT DEFAULT 1 COMMENT '1=启用 0=禁用',
  `token_invalid_before` DATETIME NULL,
  `last_login_at` DATETIME NULL,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_drivers_phone` (`phone`),
  KEY `idx_drivers_employee_no` (`employee_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='司机';

-- 班次表
CREATE TABLE IF NOT EXISTS `trips` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `route_id` BIGINT UNSIGNED NOT NULL,
  `vehicle_id` BIGINT UNSIGNED NOT NULL,
  `driver_id` BIGINT UNSIGNED DEFAULT 0,
  `trip_no` VARCHAR(32) NOT NULL,
  `trip_date` DATE NOT NULL,
  `departure_time` VARCHAR(8) NOT NULL,
  `arrival_time` VARCHAR(8) NOT NULL,
  `arrival_day_offset` INT DEFAULT 0 COMMENT '终点到达相对发车日的天数偏移(0=当天,1=次日...)，跨省长途跨天班次用',
  `total_seats` INT NOT NULL,
  `available_seats` INT NOT NULL,
  `base_price` DECIMAL(8,2) NOT NULL,
  `status` TINYINT DEFAULT 1 COMMENT '1=可售 2=已发车 3=已取消 0=下架',
  `current_passed_order` INT DEFAULT 0 COMMENT '手动标记车已驶过到第几站序(0=未标记)，>0时覆盖其他到站判断',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_trips_driver_id` (`driver_id`),
  KEY `idx_route_date` (`route_id`, `trip_date`),
  KEY `idx_date_status` (`trip_date`, `status`),
  CONSTRAINT `fk_trips_route` FOREIGN KEY (`route_id`) REFERENCES `routes` (`id`),
  CONSTRAINT `fk_trips_vehicle` FOREIGN KEY (`vehicle_id`) REFERENCES `vehicles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班次(车次)';

-- 订单表（车票+托运统一）
CREATE TABLE IF NOT EXISTS `orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_no` VARCHAR(32) NOT NULL,
  `order_type` TINYINT DEFAULT 1 COMMENT '1=车票 2=托运',
  `user_id` BIGINT UNSIGNED NOT NULL,
  `trip_id` BIGINT UNSIGNED NOT NULL,
  `route_id` BIGINT UNSIGNED NOT NULL,
  `from_station_id` BIGINT UNSIGNED NOT NULL,
  `from_station_name` VARCHAR(64) DEFAULT '' COMMENT '冗余站名 删线路后还能显示',
  `to_station_id` BIGINT UNSIGNED NOT NULL,
  `to_station_name` VARCHAR(64) DEFAULT '' COMMENT '冗余站名',
  `trip_date` DATE NOT NULL,
  `departure_time` VARCHAR(8) NOT NULL,
  `passenger_count` INT DEFAULT 1,
  `total_price` DECIMAL(10,2) NOT NULL,
  `status` TINYINT DEFAULT 0 COMMENT '0=待支付 1=待出行 2=已完成 3=已退款 4=已取消 5=已取件',
  `contact_name` VARCHAR(64) DEFAULT '',
  `contact_phone` VARCHAR(20) DEFAULT '',
  `sender_name` VARCHAR(64) DEFAULT '',
  `sender_phone` VARCHAR(20) DEFAULT '',
  `receiver_name` VARCHAR(64) DEFAULT '',
  `receiver_phone` VARCHAR(20) DEFAULT '',
  `cargo_type` VARCHAR(32) DEFAULT '',
  `weight` DECIMAL(8,2) DEFAULT 0,
  `description` VARCHAR(255) DEFAULT '',
  `pay_time` DATETIME NULL,
  `pay_method` VARCHAR(16) DEFAULT '',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_orders_order_no` (`order_no`),
  KEY `idx_orders_order_type` (`order_type`),
  KEY `idx_orders_user_id` (`user_id`),
  KEY `idx_orders_trip_id` (`trip_id`),
  KEY `idx_orders_status` (`status`),
  KEY `idx_orders_created_at` (`created_at`),
  CONSTRAINT `fk_orders_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_orders_trip` FOREIGN KEY (`trip_id`) REFERENCES `trips` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单（统一订单：车票+托运）';

-- 订单乘客表
CREATE TABLE IF NOT EXISTS `order_passengers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(64) NOT NULL,
  `id_card_type` TINYINT DEFAULT 1 COMMENT '1=身份证 2=护照',
  `id_card_no` VARCHAR(128) NOT NULL COMMENT '身份证号 AES加密',
  `phone` VARCHAR(20) DEFAULT '',
  `seat_no` VARCHAR(8) DEFAULT '',
  `check_status` TINYINT DEFAULT 0 COMMENT '0=未核销 1=已核销',
  `checked_at` DATETIME NULL,
  `checked_by` BIGINT UNSIGNED DEFAULT 0 COMMENT '司机ID',
  `created_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_order_passengers_order_id` (`order_id`),
  CONSTRAINT `fk_order_passengers_order` FOREIGN KEY (`order_id`) REFERENCES `orders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单乘客';

-- 常用乘客表
CREATE TABLE IF NOT EXISTS `passengers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(64) NOT NULL,
  `id_card_type` TINYINT DEFAULT 1,
  `id_card_no` VARCHAR(128) NOT NULL COMMENT '身份证号 AES加密',
  `phone` VARCHAR(20) DEFAULT '',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_passengers_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='常用乘客';

-- 支付记录表
CREATE TABLE IF NOT EXISTS `payments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `payment_no` VARCHAR(64) NOT NULL,
  `transaction_id` VARCHAR(64) DEFAULT '',
  `amount` DECIMAL(10,2) NOT NULL,
  `method` VARCHAR(32) DEFAULT '微信支付',
  `status` TINYINT DEFAULT 0 COMMENT '0=待支付 1=成功 2=失败 3=已退款',
  `pay_time` DATETIME NULL,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_payments_payment_no` (`payment_no`),
  KEY `idx_payments_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='支付记录';

-- 退款记录表
CREATE TABLE IF NOT EXISTS `refunds` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `refund_no` VARCHAR(64) NOT NULL,
  `amount` DECIMAL(10,2) NOT NULL,
  `reason` VARCHAR(255) DEFAULT '',
  `status` TINYINT DEFAULT 0 COMMENT '0=处理中 1=成功 2=失败',
  `refund_time` DATETIME NULL,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_refunds_refund_no` (`refund_no`),
  KEY `idx_refunds_order_id` (`order_id`),
  CONSTRAINT `fk_refunds_order` FOREIGN KEY (`order_id`) REFERENCES `orders` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='退款记录';

-- 轮播图表
CREATE TABLE IF NOT EXISTS `banners` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(128) DEFAULT '',
  `title_color` VARCHAR(32) DEFAULT '' COMMENT '标题颜色: white/black/yellow/red/blue/green/cyan/purple',
  `title_effect` TINYINT DEFAULT 0 COMMENT '标题特效: 0=无 1=阴影 2=磨砂玻璃',
  `image_url` VARCHAR(512) NOT NULL,
  `link_type` TINYINT DEFAULT 0 COMMENT '0=无 1=车次 2=外链',
  `link_value` VARCHAR(512) DEFAULT '',
  `sort_order` INT DEFAULT 0,
  `status` TINYINT DEFAULT 1 COMMENT '1=显示 0=隐藏',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='轮播图';

-- 系统配置表
CREATE TABLE IF NOT EXISTS `system_configs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `config_key` VARCHAR(64) NOT NULL,
  `config_value` TEXT,
  `description` VARCHAR(255) DEFAULT '',
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_system_configs_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置';

-- 操作日志表
CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `admin_id` BIGINT UNSIGNED NOT NULL,
  `admin_name` VARCHAR(64) NOT NULL,
  `module` VARCHAR(32) NOT NULL,
  `action` VARCHAR(32) NOT NULL,
  `target` VARCHAR(128) DEFAULT '',
  `detail` TEXT,
  `ip_address` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_operation_logs_admin_id` (`admin_id`),
  KEY `idx_operation_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作日志';

-- 班次车辆实时位置表
CREATE TABLE IF NOT EXISTS `vehicle_locations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `trip_id` BIGINT UNSIGNED NOT NULL COMMENT '关联班次',
  `vehicle_id` BIGINT UNSIGNED COMMENT '关联车辆',
  `driver_id` BIGINT UNSIGNED COMMENT '上报司机',
  `longitude` DECIMAL(10,6) COMMENT '经度(gcj02)',
  `latitude` DECIMAL(10,6) COMMENT '纬度(gcj02)',
  `speed` DECIMAL(6,1) DEFAULT 0 COMMENT '速度m/s 前端展示×3.6转km/h',
  `heading` DECIMAL(5,1) DEFAULT 0 COMMENT '方向角',
  `reported_at` DATETIME COMMENT '上报时间',
  `created_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_vehicle_locations_trip_id` (`trip_id`),
  KEY `idx_vehicle_locations_vehicle_id` (`vehicle_id`),
  KEY `idx_vehicle_locations_driver_id` (`driver_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班次车辆实时位置（司机上报GPS）';

-- 优惠券模板表
CREATE TABLE IF NOT EXISTS `coupons` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT '优惠券名称',
  `type` TINYINT DEFAULT 1 COMMENT '1=满减 2=折扣 3=固定金额',
  `discount_value` DECIMAL(10,2) NOT NULL COMMENT '满减/固定金额=元, 折扣=折(如8.5)',
  `min_spend` DECIMAL(10,2) DEFAULT 0 COMMENT '最低消费门槛(元)',
  `valid_days` INT DEFAULT 30 COMMENT '领取后有效天数',
  `total_count` INT DEFAULT 0 COMMENT '发放总量, 0=不限',
  `issued_count` INT DEFAULT 0 COMMENT '已发放数量',
  `used_count` INT DEFAULT 0 COMMENT '已使用数量',
  `status` TINYINT DEFAULT 1 COMMENT '1=启用 0=停用',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='优惠券模板';

-- 用户优惠券表
CREATE TABLE IF NOT EXISTS `user_coupons` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `coupon_id` BIGINT UNSIGNED NOT NULL,
  `status` TINYINT DEFAULT 0 COMMENT '0=未使用 1=已使用 2=已过期',
  `issued_at` DATETIME,
  `expired_at` DATETIME,
  `used_at` DATETIME NULL,
  `order_id` BIGINT UNSIGNED DEFAULT 0,
  `created_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_user_coupons_user_id` (`user_id`),
  KEY `idx_user_coupons_coupon_id` (`coupon_id`),
  KEY `idx_user_coupons_status` (`status`),
  KEY `idx_user_coupons_created_at` (`created_at`),
  CONSTRAINT `fk_user_coupons_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_user_coupons_coupon` FOREIGN KEY (`coupon_id`) REFERENCES `coupons` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户优惠券（发放记录）';

-- 积分规则表
CREATE TABLE IF NOT EXISTS `point_rules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `rule_name` VARCHAR(128) NOT NULL COMMENT '规则名称',
  `rule_type` TINYINT DEFAULT 1 COMMENT '1=消费赠送 2=注册赠送 3=手动调整',
  `points_per_yuan` DECIMAL(10,2) DEFAULT 1 COMMENT '每消费1元获得积分(消费赠送时使用)',
  `fixed_points` INT DEFAULT 0 COMMENT '固定积分(注册赠送时使用)',
  `description` VARCHAR(255) DEFAULT '' COMMENT '规则说明',
  `status` TINYINT DEFAULT 1 COMMENT '1=启用 0=停用',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='积分规则';

-- 用户积分余额表
CREATE TABLE IF NOT EXISTS `user_points` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `balance` INT DEFAULT 0 COMMENT '当前积分余额',
  `total_earned` INT DEFAULT 0 COMMENT '累计获得',
  `total_spent` INT DEFAULT 0 COMMENT '累计消耗',
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_points_user_id` (`user_id`),
  CONSTRAINT `fk_user_points_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户积分余额';

-- 积分明细表
CREATE TABLE IF NOT EXISTS `point_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `change_type` TINYINT NOT NULL COMMENT '1=获得 2=消耗',
  `points` INT NOT NULL COMMENT '变动积分(正数)',
  `source` VARCHAR(32) DEFAULT '' COMMENT '来源: order/manual/register',
  `order_id` BIGINT UNSIGNED DEFAULT 0,
  `remark` VARCHAR(255) DEFAULT '',
  `admin_id` BIGINT UNSIGNED DEFAULT 0,
  `admin_name` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME,
  PRIMARY KEY (`id`),
  KEY `idx_point_records_user_id` (`user_id`),
  KEY `idx_point_records_created_at` (`created_at`),
  CONSTRAINT `fk_point_records_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='积分明细';

-- 订阅消息配额表
CREATE TABLE IF NOT EXISTS `subscribe_quotas` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `template_key` VARCHAR(32) NOT NULL COMMENT 'payment_success, trip_departure, trip_arrival, refund_success',
  `quota` INT DEFAULT 0 COMMENT '剩余可发送次数',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sub_quota_user_template` (`user_id`, `template_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅消息发送配额';

-- 种子数据 系统默认配置
-- 业务数据站点线路那些去后台手动加

-- 默认系统配置
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES
('site_name', '狸猫日志售票', '站点名称'),
('order_expire_minutes', '15', '待支付订单超时分钟数'),
('refund_before_departure_hours', '2', '发车前可退款小时数'),
('customer_service_phone', '', '客服电话'),
('after_sales_wechat', 'lihao68681818', '售后微信'),
('refund_fee_rate', '10', '退票手续费率(%)'),
('notice', '', '首页公告内容'),
('user_agreement', '狸猫日志售票平台用户协议...（详见管理后台协议政策页面）', '用户协议'),
('privacy_policy', '狸猫日志售票平台隐私政策...（详见管理后台协议政策页面）', '隐私政策');

-- 默认积分规则
INSERT INTO `point_rules` (`rule_name`, `rule_type`, `points_per_yuan`, `fixed_points`, `description`, `status`) VALUES
('消费赠送', 1, 1.00, 0, '每消费1元获得1积分', 1),
('注册赠送', 2, 0.00, 100, '新用户注册赠送100积分', 1);

-- 默认管理员 admin/admin123 程序启动时InitAdmin自动创建
-- 不在这直接写INSERT 因为bcrypt哈希每次不一样
-- 登录后赶紧改密码
