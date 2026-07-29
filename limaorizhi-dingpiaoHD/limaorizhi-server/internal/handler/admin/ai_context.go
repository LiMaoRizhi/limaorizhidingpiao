package admin

import (
	"fmt"
	"strconv"
	"strings"

	"limaorizhi-server/internal/model"

	"gorm.io/gorm"
)

// buildContext 按用户问题关键词从数据库拉相关数据拼成上下文喂给AI
func (h *AIHandler) buildContext(userMessage string) string {
	msg := strings.ToLower(userMessage)
	var parts []string

	if hasAny(msg, "价格", "票价", "计算", "线路", "路线", "站点", "每站", "多少钱") {
		parts = append(parts, h.routePricingCtx())
	}
	if hasAny(msg, "托运", "货物", "运费", "快递", "寄件") {
		parts = append(parts, h.cargoConfigCtx())
	}
	if hasAny(msg, "退票", "退款", "取消", "手续费") {
		parts = append(parts, h.orderConfigCtx())
	}
	if hasAny(msg, "班次", "发车", "调度") {
		parts = append(parts, h.tripCtx())
	}
	if hasAny(msg, "司机") {
		parts = append(parts, h.driverCtx())
	}
	if hasAny(msg, "车辆", "大巴", "客车") {
		parts = append(parts, h.vehicleCtx())
	}
	if hasAny(msg, "配置", "系统", "设置", "参数") {
		parts = append(parts, h.systemConfigCtx())
	}
	if hasAny(msg, "用户", "乘客", "注册", "会员") {
		parts = append(parts, h.userStatsCtx())
	}
	if hasAny(msg, "订单", "售票", "收入", "销量", "销售", "金额", "营业额") {
		parts = append(parts, h.orderStatsCtx())
	}
	if hasAny(msg, "优惠券", "券", "营销", "满减", "折扣") {
		parts = append(parts, h.couponCtx())
	}
	if hasAny(msg, "积分") {
		parts = append(parts, h.pointRuleCtx())
	}

	if len(parts) == 0 {
		return ""
	}
	result := "以下是从数据库拉取的当前系统真实数据，请基于这些数据回答：\n\n" + strings.Join(parts, "\n\n")
	// 上下文太长超token还费钱 按rune（Unicode字符）截断
	// 不用len()判断字节数 中文每字3字节 15000 runes ≈ 5000汉字 够覆盖20条线路×5站点的完整定价数据
	// 现代模型上下文窗口128K+，15000 runes足够安全
	const maxContextRunes = 15000
	runes := []rune(result)
	if len(runes) > maxContextRunes {
		// 智能截断：在最后一个完整行处截断，避免截断到某条线路/站点数据的中间
		truncated := string(runes[:maxContextRunes])
		if lastNL := strings.LastIndex(truncated, "\n"); lastNL > maxContextRunes/2 {
			result = truncated[:lastNL] + "\n...(数据过多，已截断)"
		} else {
			result = truncated + "\n...(数据过多，已截断)"
		}
	}
	return result
}

// maskPhone 手机号脱敏 前3后4中间打* 别让AI把手机号吐出来
func maskPhone(phone string) string {
	if len(phone) <= 7 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}

// hasAny 看s里有没有任意一个关键词
func hasAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// plainText 从content提纯文本 兼容string和多模态数组
func plainText(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var texts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					texts = append(texts, t)
				}
			}
		}
		return strings.Join(texts, " ")
	}
	return ""
}

func (h *AIHandler) routePricingCtx() string {
	var routes []model.Route
	h.DB.Preload("FromStation").Preload("ToStation").
		Preload("RouteStations", func(db *gorm.DB) *gorm.DB { return db.Order("stop_order ASC") }).
		Preload("RouteStations.Station").
		Where("status = 1").
		Limit(20).Find(&routes)

	if len(routes) == 0 {
		return "【线路票价】当前无运营中的线路"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【线路票价数据】(共显示%d条运营中线路)", len(routes)))
	for _, r := range routes {
		sb.WriteString("\n线路: " + r.Name)
		sb.WriteString(" | 起步价: " + strconv.FormatFloat(r.MinFare, 'f', 1, 64) + "元")
		sb.WriteString(" | 全程里程: " + strconv.FormatFloat(r.DistanceKM, 'f', 1, 64) + "km")
		sb.WriteString("\n  站点序列:")
		for _, rs := range r.RouteStations {
			stationName := ""
			if rs.Station != nil {
				stationName = rs.Station.Name
			}
			sb.WriteString("\n    第" + strconv.Itoa(rs.StopOrder) + "站 " + stationName +
				" | 累计里程" + strconv.FormatFloat(rs.DistanceKM, 'f', 1, 64) + "km" +
				" | 累计票价" + strconv.FormatFloat(rs.Price, 'f', 1, 64) + "元")
		}
	}
	return sb.String()
}

func (h *AIHandler) cargoConfigCtx() string {
	cfg := h.batchConfig([]string{"cargo_price_per_km", "cargo_min_fee", "cargo_free_weight", "cargo_extra_weight_fee", "cargo_max_weight"})
	return fmt.Sprintf("【货物托运配置】\n每公里运费: %s元 | 最低运费: %s元 | 免费重量: %skg | 超重费: %s元/kg | 最大重量: %skg\n运费公式: 基础运费=max(最低运费, 距离×每公里运费) + 超重费=max(0,(重量-免费重量)×超重费/kg)，总运费向上取整",
		cfg["cargo_price_per_km"], cfg["cargo_min_fee"], cfg["cargo_free_weight"], cfg["cargo_extra_weight_fee"], cfg["cargo_max_weight"])
}

func (h *AIHandler) orderConfigCtx() string {
	cfg := h.batchConfig([]string{"order_expire_minutes", "refund_before_departure_hours", "refund_fee_rate"})
	return fmt.Sprintf("【订单与退票配置】\n支付超时: %s分钟 | 退票截止: 发车前%s小时内不可退 | 退票手续费率: %s%%",
		cfg["order_expire_minutes"], cfg["refund_before_departure_hours"], cfg["refund_fee_rate"])
}

func (h *AIHandler) tripCtx() string {
	var trips []model.Trip
	h.DB.Preload("Route.FromStation").Preload("Route.ToStation").Preload("Vehicle").
		Where("status IN (1, 2)").
		Limit(20).Find(&trips)

	if len(trips) == 0 {
		return "【班次数据】当前无活跃班次"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【近期班次数据】(共显示%d条活跃班次)", len(trips)))
	for _, t := range trips {
		routeName := ""
		if t.Route != nil {
			fromName, toName := "", ""
			if t.Route.FromStation != nil {
				fromName = t.Route.FromStation.Name
			}
			if t.Route.ToStation != nil {
				toName = t.Route.ToStation.Name
			}
			routeName = fromName + "→" + toName
		}
		vehicleNo := ""
		if t.Vehicle != nil {
			vehicleNo = t.Vehicle.PlateNo
		}
		statusText := "未知"
		switch t.Status {
		case 1:
			statusText = "可售"
		case 2:
			statusText = "已发车"
		}
		sb.WriteString("\n班次" + t.TripNo + " | " + string(t.TripDate) + " " + t.DepartureTime + " | " + routeName +
			" | 车辆" + vehicleNo + " | 总座位" + strconv.Itoa(t.TotalSeats) + " | 状态" + statusText)
	}
	return sb.String()
}

func (h *AIHandler) systemConfigCtx() string {
	// 白名单模式 只注入非敏感配置 客服电话售后微信那些不能让AI吐出来
	safeConfigKeys := []string{
		"order_expire_minutes", "refund_before_departure_hours", "refund_fee_rate",
		"cargo_price_per_km", "cargo_min_fee", "cargo_free_weight",
		"cargo_extra_weight_fee", "cargo_max_weight",
		"mine_menu_layout_type", "logout_position",
	}
	cfg := h.batchConfig(safeConfigKeys)

	var sb strings.Builder
	sb.WriteString("【系统配置】")
	for _, key := range safeConfigKeys {
		if cfg[key] != "" {
			sb.WriteString("\n" + key + " = " + cfg[key])
		}
	}
	if sb.Len() == len("【系统配置】") {
		return ""
	}
	return sb.String()
}

// batchConfig 批量读配置 没有的填空字符串
func (h *AIHandler) batchConfig(keys []string) map[string]string {
	result := make(map[string]string)
	var configs []model.SystemConfig
	h.DB.Where("config_key IN ?", keys).Find(&configs)
	for _, c := range configs {
		result[c.ConfigKey] = c.ConfigValue
	}
	for _, k := range keys {
		if _, ok := result[k]; !ok {
			result[k] = ""
		}
	}
	return result
}

// driverCtx 司机列表上下文
func (h *AIHandler) driverCtx() string {
	var drivers []model.Driver
	h.DB.Where("status = 1").Limit(20).Find(&drivers)
	if len(drivers) == 0 {
		return "【司机数据】当前无在职司机"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【司机数据】(共显示%d名在职司机)", len(drivers)))
	for _, d := range drivers {
		sb.WriteString("\n工号:" + d.EmployeeNo + " | 姓名:" + d.Name + " | 电话:" + maskPhone(d.Phone))
	}
	return sb.String()
}

// vehicleCtx 车辆列表上下文
func (h *AIHandler) vehicleCtx() string {
	var vehicles []model.Vehicle
	h.DB.Where("status = 1").Limit(20).Find(&vehicles)
	if len(vehicles) == 0 {
		return "【车辆数据】当前无可用车辆"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【车辆数据】(共显示%d辆可用车辆)", len(vehicles)))
	for _, v := range vehicles {
		sb.WriteString("\n车牌:" + v.PlateNo + " | 类型:" + v.VehicleType + " | 座位数:" + strconv.Itoa(v.SeatCount))
	}
	return sb.String()
}

// userStatsCtx 用户统计上下文
func (h *AIHandler) userStatsCtx() string {
	var totalUsers, activeUsers, bannedUsers int64
	h.DB.Model(&model.User{}).Count(&totalUsers)
	h.DB.Model(&model.User{}).Where("status = 1").Count(&activeUsers)
	h.DB.Model(&model.User{}).Where("status = 0").Count(&bannedUsers)
	return fmt.Sprintf("【用户统计】\n总用户数: %d | 正常用户: %d | 封禁用户: %d", totalUsers, activeUsers, bannedUsers)
}

// orderStatsCtx 订单统计 按状态聚合算一下售票总额
func (h *AIHandler) orderStatsCtx() string {
	type stats struct {
		Status  int8
		Count   int64
		Revenue float64
	}
	var results []stats
	h.DB.Model(&model.Order{}).
		Select("status, count(*) as count, coalesce(sum(total_price), 0) as revenue").
		Group("status").
		Find(&results)
	if len(results) == 0 {
		return "【订单统计】当前无订单数据"
	}
	statusNames := map[int8]string{0: "待支付", 1: "待出行", 2: "已完成", 3: "已退票", 4: "已取消"}
	var sb strings.Builder
	sb.WriteString("【订单统计】")
	var totalRevenue float64
	var totalCount int64
	for _, r := range results {
		name := statusNames[r.Status]
		if name == "" {
			name = "状态" + strconv.Itoa(int(r.Status))
		}
		sb.WriteString(fmt.Sprintf("\n%s: %d笔 | 金额: %.2f元", name, r.Count, r.Revenue))
		// 已支付的算售票总额
		if r.Status == 1 || r.Status == 2 {
			totalRevenue += r.Revenue
		}
		totalCount += r.Count
	}
	sb.WriteString(fmt.Sprintf("\n总订单: %d笔 | 售票总额(已支付): %.2f元", totalCount, totalRevenue))
	return sb.String()
}

// couponCtx 优惠券上下文
func (h *AIHandler) couponCtx() string {
	var coupons []model.Coupon
	h.DB.Where("status = 1").Limit(20).Find(&coupons)
	if len(coupons) == 0 {
		return "【优惠券数据】当前无启用的优惠券"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【优惠券数据】(共显示%d张启用优惠券)", len(coupons)))
	for _, c := range coupons {
		typeText := ""
		switch c.Type {
		case 1:
			typeText = fmt.Sprintf("满%.0f减%.0f元", c.MinSpend, c.DiscountValue)
		case 2:
			typeText = fmt.Sprintf("%.1f折", c.DiscountValue)
		case 3:
			typeText = fmt.Sprintf("抵扣%.0f元", c.DiscountValue)
		}
		totalCountText := "不限"
		if c.TotalCount > 0 {
			totalCountText = strconv.Itoa(c.TotalCount)
		}
		sb.WriteString(fmt.Sprintf("\n%s | %s | 已发%d/%s | 已用%d",
			c.Name, typeText, c.IssuedCount, totalCountText, c.UsedCount))
	}
	return sb.String()
}

// pointRuleCtx 积分规则上下文
func (h *AIHandler) pointRuleCtx() string {
	var rules []model.PointRule
	h.DB.Where("status = 1").Limit(10).Find(&rules)
	if len(rules) == 0 {
		return "【积分规则】当前无启用的积分规则"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【积分规则】(共显示%d条启用规则)", len(rules)))
	for _, r := range rules {
		typeText := ""
		switch r.RuleType {
		case 1:
			typeText = fmt.Sprintf("消费赠送(每元%.1f积分)", r.PointsPerYuan)
		case 2:
			typeText = fmt.Sprintf("注册赠送(%d积分)", r.FixedPoints)
		case 3:
			typeText = "手动调整"
		}
		sb.WriteString("\n" + r.RuleName + " | " + typeText + " | " + r.Description)
	}
	return sb.String()
}
