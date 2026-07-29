// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package money

// 金额计算辅助函数
// 数据库 decimal(10,2) 经 GORM 读出后为 float64，直接做乘法/累加存在二进制浮点误差。
// 本包统一通过"整数分"做运算，避免 10.99*3 != 32.97 类问题。

// ToFen 将元（float64）转换为分（int64），四舍五入
func ToFen(yuan float64) int64 {
	if yuan >= 0 {
		return int64(yuan*100 + 0.5)
	}
	return int64(yuan*100 - 0.5)
}

// FromFen 将分（int64）转换为元（float64）
func FromFen(fen int64) float64 {
	return float64(fen) / 100
}

// Mul 元 * 数量 = 元（结果四舍五入到分）
func Mul(yuan float64, count int) float64 {
	return FromFen(ToFen(yuan) * int64(count))
}

// Sub 元 - 元 = 元（结果四舍五入到分）
func Sub(a, b float64) float64 {
	return FromFen(ToFen(a) - ToFen(b))
}

// Add 元 + 元 = 元（结果四舍五入到分）
func Add(a, b float64) float64 {
	return FromFen(ToFen(a) + ToFen(b))
}

// Discount 折扣计算：orderTotal * (1 - discountValue/10)，结果四舍五入到分
func Discount(orderTotal, discountValue float64) float64 {
	fen := ToFen(orderTotal)
	// discountValue 为折扣（如 8.5 表示 85 折），discountFen = fen * (10 - discountValue) / 10
	discountFen := int64(float64(fen) * (10 - discountValue) / 10)
	return FromFen(discountFen)
}
