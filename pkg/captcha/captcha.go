package captcha

import (
	"image/color"
	"math/rand"
	"time"

	"github.com/mojocn/base64Captcha"
)

var store = base64Captcha.DefaultMemStore

// Generate 生成验证码
func Generate() (id, b64s string, err error) {
	// 配置验证码参数
	driver := base64Captcha.NewDriverDigit(
		80,  // 高度
		240, // 宽度
		4,   // 验证码长度
		0.7, // 最大倾斜角度
		80,  // 点的数量
	)

	// 生成验证码
	captcha := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err = captcha.Generate()
	return id, b64s, err
}

// Verify 验证验证码
func Verify(id, answer string) bool {
	return store.Verify(id, answer, true)
}

// GenerateColorful 生成彩色字符验证码
func GenerateColorful() (id, b64s string, err error) {
	driver := base64Captcha.NewDriverString(
		80,                                 // 高度
		240,                                // 宽度
		6,                                  // 干扰线数量
		base64Captcha.OptionShowHollowLine, // 显示空心线
		4,                                  // 验证码长度
		"abcdefghjkmnpqrstuvwxyz23456789",  // 字符源
		&color.RGBA{R: 240, G: 240, B: 246, A: 246}, // 背景颜色
		nil, // 使用默认字体
		[]string{"wqy-microhei.ttc"},
	)

	rand.Seed(time.Now().UnixNano())
	captcha := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err = captcha.Generate()
	return id, b64s, err
}
