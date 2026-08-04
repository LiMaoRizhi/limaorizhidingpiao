package idcard

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 身份证号正则：18位，前17位数字，最后一位数字或X
var idCardRegex = regexp.MustCompile(`^\d{17}[\dXx]$`)

// 校验码权重因子（ISO 7064:1983.MOD 11-2）
var weightFactors = []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}

// 校验码对照表
var checkCodes = []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

// ValidateFormat 校验身份证号格式（本地校验，不调用外部API）
// 校验规则：
// 1. 18位长度
// 2. 前17位为数字，最后一位为数字或X
// 3. 出生日期合法
// 4. 校验码正确（ISO 7064:1983.MOD 11-2）
func ValidateFormat(idCard string) error {
	if len(idCard) != 18 {
		return errors.New("身份证号必须是18位")
	}

	if !idCardRegex.MatchString(idCard) {
		return errors.New("身份证号格式错误")
	}

	// 校验出生日期
	birthDate := idCard[6:14] // YYYYMMDD
	// 校验日期合理性（time.Parse会校验如2月30日等非法日期）
	t, err := time.Parse("20060102", birthDate)
	if err != nil {
		return errors.New("身份证号出生日期无效")
	}
	if t.After(time.Now()) {
		return errors.New("身份证号出生日期不能为未来日期")
	}

	// 校验码验证
	sum := 0
	for i := 0; i < 17; i++ {
		digit, err := strconv.Atoi(string(idCard[i]))
		if err != nil {
			return errors.New("身份证号格式错误")
		}
		sum += digit * weightFactors[i]
	}
	expectedCheckCode := checkCodes[sum%11]

	// 最后一位不区分大小写
	lastChar := idCard[17]
	if lastChar == 'x' {
		lastChar = 'X'
	}
	if lastChar != expectedCheckCode {
		return errors.New("身份证号校验码错误")
	}

	return nil
}

// GetGender 从身份证号获取性别（奇数=男，偶数=女）
func GetGender(idCard string) string {
	if len(idCard) != 18 {
		return ""
	}
	genderDigit, err := strconv.Atoi(string(idCard[16]))
	if err != nil {
		return ""
	}
	if genderDigit%2 == 1 {
		return "男"
	}
	return "女"
}

// GetBirthday 从身份证号获取生日（格式：2006-01-02）
func GetBirthday(idCard string) string {
	if len(idCard) != 18 {
		return ""
	}
	return idCard[6:10] + "-" + idCard[10:12] + "-" + idCard[12:14]
}

// MaskIDCard 身份证号脱敏：显示前4后4，中间用*代替
// 示例：340421190710145412 → 3404**********5412
func MaskIDCard(idCard string) string {
	if len(idCard) <= 8 {
		return idCard
	}
	return idCard[:4] + strings.Repeat("*", len(idCard)-8) + idCard[len(idCard)-4:]
}

// MaskPhone 手机号脱敏：显示前3后4，中间用*代替
// 示例：13800138001 → 138****8001
func MaskPhone(phone string) string {
	if len(phone) <= 7 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}
