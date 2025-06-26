package floatUtils

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// 将浮点数字符串转换为decimal.Decimal
// 1、限制字符串浮点数最多有两位小数，超过两位小数则返回错误
// 2、不允许负数，如果为负数则返回错误
func ParseFloatStr(str string) (result decimal.Decimal, err error) {
	str = strings.TrimSpace(str)

	// 首先尝试转换为decimal
	result, err = decimal.NewFromString(str)
	if err != nil {
		err = fmt.Errorf("转换失败：%s", str)
		return
	}

	// 检查是否为负数
	if result.LessThan(decimal.Zero) {
		err = fmt.Errorf("不允许负数：%s", str)
		return
	}
	// 检查是否等于0
	if result.Equal(decimal.Zero) {
		err = fmt.Errorf("不允许为0")
		return
	}

	// 检查小数位数
	// fmt.Printf("Str=%s\t\tCoefficient=%v\tExponent=%v\n", str, result.Coefficient(), result.Exponent())
	// Str=123     Coefficient=123    Exponent=0
	// Str=123.45  Coefficient=12345  Exponent=-4
	// Str=0.1     Coefficient=1      Exponent=-1
	// Str=0.123   Coefficient=123    Exponent=-3
	// Str=1.00    Coefficient=100    Exponent=-2
	decimalPlaces := result.Exponent() * -1 // Exponent()返回负数，所以需要取反
	// 如果小数位数超过2位，返回错误
	if decimalPlaces > 2 {
		err = fmt.Errorf("小数位数超过限制, 最多允许2位小数")
		return
	}

	return
}
