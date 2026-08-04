package sanitize

import (
	"fmt"
	"strings"
	"time"

	"crypto/rand"
	"encoding/hex"
)

// GenerateRefundNo 统一生成退款单号：RF + 日期 + 时分秒 + 6位订单ID + 4位随机后缀
// 全项目统一格式，避免不同调用点格式不一致导致对账困难。
// 末尾追加 4 位随机串，防止同秒多笔退款单号碰撞。
func GenerateRefundNo(orderID uint) string {
	now := time.Now()
	// 4字节随机后缀，避免同秒并发退款单号重复
	randBytes := make([]byte, 2)
	_, _ = rand.Read(randBytes)
	return fmt.Sprintf("RF%s%s%06d%s", now.Format("20060102"), now.Format("150405"), orderID, hex.EncodeToString(randBytes))
}

// EscapeLikePattern 转义 SQL LIKE 模式串中的特殊字符（% _ \），防止用户输入通配符
// 导致模糊查询范围失控或注入风险。转义后配合查询使用 ESCAPE '\' 子句。
// 调用示例：query.Where("name LIKE ?", "%"+sanitize.EscapeLikePattern(name)+"%")
func EscapeLikePattern(s string) string {
	if s == "" {
		return s
	}
	// 先转义反斜杠，再转义 % 和 _
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
