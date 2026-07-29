// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // 注册JPEG解码器（背景图实际是JPEG格式）
	"image/png"
	"math/big"
	"strings"
	"sync"
	"time"

	redis "limaorizhi-server/internal/pkg/redis"
)

//go:embed captcha_bg/*.png
var captchaBgFS embed.FS

// 画布和拼图块尺寸常量（与前端保持一致）
const (
	CaptchaW    = 310
	CaptchaH    = 310
	PieceBodyW  = 34
	PieceTabR   = 10
	PieceTotalW = PieceBodyW + PieceTabR // 44
	PieceTotalH = 44                     // 正方形拼图块
	PieceYMin   = 5
	PieceYMax   = CaptchaH - PieceTotalH - 5 // 261
	Tolerance   = 5                      // 容差像素
	MaxDrag     = 266                    // CaptchaW - PieceTotalW
	TargetMin   = 80
	TargetMax   = 256                    // MaxDrag - 10
	CaptchaTTL  = 5 * time.Minute       // 验证码有效期
	VerifyTTL   = 2 * time.Minute       // 验证通过token有效期（登录时使用）
)

// CaptchaResult 验证码生成结果
type CaptchaResult struct {
	Token       string `json:"token"`
	BgImage     string `json:"bgImage"`     // 带缺口的背景图 base64
	PuzzleImage string `json:"puzzleImage"` // 拼图块 base64
	YPosition   int    `json:"yPosition"`   // 拼图块Y轴位置
}

// captchaState 验证码状态（存储在Redis或进程内存中）
type captchaState struct {
	TargetX  int       `json:"target_x"`
	TargetY  int       `json:"target_y"`
	ExpireAt time.Time `json:"expire_at"`
}

// 进程内存存储（Redis不可用时降级）
var memCaptchaStore sync.Map // key: token -> *captchaState
var memVerifyStore sync.Map  // key: verifyToken -> time.Time (expireAt)

func init() {
	// 定期清理过期的内存存储条目（Redis不可用时的降级方案内存回收，防止泄漏）
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			memCaptchaStore.Range(func(key, value any) bool {
				state, ok := value.(*captchaState)
				if ok && now.After(state.ExpireAt) {
					memCaptchaStore.Delete(key)
				}
				return true
			})
			memVerifyStore.Range(func(key, value any) bool {
				expireAt, ok := value.(time.Time)
				if ok && now.After(expireAt) {
					memVerifyStore.Delete(key)
				}
				return true
			})
		}
	}()
}

// --- 公开API ---

// GenerateCaptcha 生成滑动验证码
func GenerateCaptcha() (*CaptchaResult, error) {
	// 1. 加载随机背景图
	bgRaw, err := loadRandomBackground()
	if err != nil {
		return nil, fmt.Errorf("加载背景图失败: %w", err)
	}

	// 2. 等比缩放并居中裁剪到标准尺寸（cover模式，不拉伸不变形）
	bgImg := resizeCover(bgRaw, CaptchaW, CaptchaH)

	// 3. 随机选取拼图块目标位置
	targetX := randInt(TargetMin, TargetMax)
	targetY := randInt(PieceYMin, PieceYMax)

	// 4. 生成带缺口的背景图
	bgWithHole := createBackgroundWithHole(bgImg, targetX, targetY)

	// 5. 生成拼图块图片
	puzzlePiece := createPuzzlePiece(bgImg, targetX, targetY)

	// 6. 编码为base64
	bgBase64, err := encodePNGBase64(bgWithHole)
	if err != nil {
		return nil, fmt.Errorf("编码背景图失败: %w", err)
	}
	puzzleBase64, err := encodePNGBase64(puzzlePiece)
	if err != nil {
		return nil, fmt.Errorf("编码拼图块失败: %w", err)
	}

	// 7. 生成token并存储状态
	token := generateToken()
	state := &captchaState{
		TargetX:  targetX,
		TargetY:  targetY,
		ExpireAt: time.Now().Add(CaptchaTTL),
	}
	storeCaptchaState(token, state)

	return &CaptchaResult{
		Token:       token,
		BgImage:     bgBase64,
		PuzzleImage: puzzleBase64,
		YPosition:   targetY,
	}, nil
}

// CheckCaptcha 验证滑动结果
// 返回: verified=是否验证通过, verifyToken=验证通过后的一次性登录令牌, err=错误信息
func CheckCaptcha(token string, moveX int) (verified bool, verifyToken string, err error) {
	// 1. 加载状态（一次性消费，取出即删除）
	state, ok := loadAndDeleteCaptchaState(token)
	if !ok {
		return false, "", fmt.Errorf("验证码不存在或已使用")
	}

	// 2. 检查是否过期
	if time.Now().After(state.ExpireAt) {
		return false, "", fmt.Errorf("验证码已过期")
	}

	// 3. 检查容差
	if absInt(moveX-state.TargetX) > Tolerance {
		return false, "", nil
	}

	// 4. 生成一次性验证token，供登录接口校验
	verifyToken = generateToken()
	storeVerifyToken(verifyToken)

	return true, verifyToken, nil
}

// VerifyCaptchaToken 校验登录时携带的验证码token（一次性消费）
func VerifyCaptchaToken(token string) bool {
	return consumeVerifyToken(token)
}

// --- 内部辅助函数 ---

// randInt 生成[min, max]范围内的随机整数（使用crypto/rand）
func randInt(min, max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return min // 降级
	}
	return min + int(n.Int64())
}

// generateToken 生成32位hex随机token
func generateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败极少见，降级用时间戳保证唯一性
		return hex.EncodeToString([]byte(fmt.Sprintf("%032d", time.Now().UnixNano())))
	}
	return hex.EncodeToString(b)
}

// absInt 整数绝对值
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// loadRandomBackground 从嵌入的图片中随机加载一张
func loadRandomBackground() (image.Image, error) {
	entries, err := captchaBgFS.ReadDir("captcha_bg")
	if err != nil {
		return nil, err
	}
	var pngFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".png") {
			pngFiles = append(pngFiles, e.Name())
		}
	}
	if len(pngFiles) == 0 {
		return nil, fmt.Errorf("未找到验证码背景图")
	}
	idx := randInt(0, len(pngFiles)-1)
	data, err := captchaBgFS.ReadFile("captcha_bg/" + pngFiles[idx])
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// lerp 线性插值（uint32 先转 float64 再计算，避免下溢）
func lerp(a, b uint32, t float64) uint32 {
	fa := float64(a)
	fb := float64(b)
	return uint32(fa + (fb-fa)*t)
}

// resizeBilinear 双线性插值缩放（标准库，无额外依赖，质量远优于最近邻）
func resizeBilinear(src image.Image, newW, newH int) *image.RGBA {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))

	scaleX := float64(srcW) / float64(newW)
	scaleY := float64(srcH) / float64(newH)

	for y := 0; y < newH; y++ {
		srcY := float64(y) * scaleY
		y0 := int(srcY)
		y1 := y0 + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		fy := srcY - float64(y0)

		for x := 0; x < newW; x++ {
			srcX := float64(x) * scaleX
			x0 := int(srcX)
			x1 := x0 + 1
			if x1 >= srcW {
				x1 = srcW - 1
			}
			fx := srcX - float64(x0)

			c00 := src.At(bounds.Min.X+x0, bounds.Min.Y+y0)
			c10 := src.At(bounds.Min.X+x1, bounds.Min.Y+y0)
			c01 := src.At(bounds.Min.X+x0, bounds.Min.Y+y1)
			c11 := src.At(bounds.Min.X+x1, bounds.Min.Y+y1)

			r00, g00, b00, a00 := c00.RGBA()
			r10, g10, b10, a10 := c10.RGBA()
			r01, g01, b01, a01 := c01.RGBA()
			r11, g11, b11, a11 := c11.RGBA()

			rTop := lerp(r00, r10, fx)
			gTop := lerp(g00, g10, fx)
			bTop := lerp(b00, b10, fx)
			aTop := lerp(a00, a10, fx)

			rBot := lerp(r01, r11, fx)
			gBot := lerp(g01, g11, fx)
			bBot := lerp(b01, b11, fx)
			aBot := lerp(a01, a11, fx)

			r := lerp(rTop, rBot, fy)
			g := lerp(gTop, gBot, fy)
			b := lerp(bTop, bBot, fy)
			a := lerp(aTop, aBot, fy)

			dst.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

// resizeCover 等比缩放并裁剪（cover模式，保持宽高比不变形）
// 水平居中裁剪，垂直底部对齐裁剪（保留底部水印区域）
func resizeCover(src image.Image, newW, newH int) *image.RGBA {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// 计算覆盖缩放比（取较大的缩放比，确保填满目标尺寸）
	scaleX := float64(newW) / float64(srcW)
	scaleY := float64(newH) / float64(srcH)
	scale := scaleX
	if scaleY > scale {
		scale = scaleY
	}

	// 等比缩放后的尺寸
	scaledW := int(float64(srcW) * scale)
	scaledH := int(float64(srcH) * scale)
	if scaledW < newW {
		scaledW = newW
	}
	if scaledH < newH {
		scaledH = newH
	}

	// 先双线性插值等比缩放
	scaled := resizeBilinear(src, scaledW, scaledH)

	// 裁剪：水平居中，垂直底部对齐（水印在图片底部，保留底部区域）
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	offsetX := (scaledW - newW) / 2
	offsetY := scaledH - newH // 底部对齐，保留水印
	if offsetY < 0 {
		offsetY = 0
	}
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			dst.Set(x, y, scaled.At(offsetX+x, offsetY+y))
		}
	}
	return dst
}

// isInsidePiece 判断像素是否在拼图块形状内
// rx, ry 是相对于拼图块左上角的坐标
func isInsidePiece(rx, ry int) bool {
	// 矩形主体部分
	if rx >= 0 && rx < PieceBodyW && ry >= 0 && ry < PieceTotalH {
		return true
	}
	// 右侧半圆形凸起
	if rx >= PieceBodyW && rx < PieceTotalW {
		dx := float64(rx - PieceBodyW)
		dy := float64(ry - PieceTotalH/2)
		return dx*dx+dy*dy <= float64(PieceTabR)*float64(PieceTabR)
	}
	return false
}

// isOnPieceBorder 判断像素是否在拼图块边缘
func isOnPieceBorder(rx, ry int) bool {
	if !isInsidePiece(rx, ry) {
		return false
	}
	neighbors := [4][2]int{{rx - 1, ry}, {rx + 1, ry}, {rx, ry - 1}, {rx, ry + 1}}
	for _, n := range neighbors {
		if !isInsidePiece(n[0], n[1]) {
			return true
		}
	}
	return false
}

// createBackgroundWithHole 在背景图上挖出拼图块形状的缺口
func createBackgroundWithHole(bg *image.RGBA, targetX, targetY int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, CaptchaW, CaptchaH))
	draw.Draw(dst, dst.Bounds(), bg, image.Point{0, 0}, draw.Src)

	// 在缺口区域内填充白色，让缺口清晰可见
	for py := 0; py < CaptchaH; py++ {
		for px := 0; px < CaptchaW; px++ {
			rx := px - targetX
			ry := py - targetY
			if isInsidePiece(rx, ry) {
				dst.Set(px, py, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}

	// 绘制缺口边框（深灰色，在白色缺口上清晰可见）
	for py := 0; py < CaptchaH; py++ {
		for px := 0; px < CaptchaW; px++ {
			rx := px - targetX
			ry := py - targetY
			if isOnPieceBorder(rx, ry) {
				dst.Set(px, py, color.RGBA{R: 80, G: 80, B: 80, A: 255})
			}
		}
	}

	return dst
}

// createPuzzlePiece 从背景图中裁剪拼图块
func createPuzzlePiece(bg *image.RGBA, targetX, targetY int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, PieceTotalW, PieceTotalH))

	// 从背景图中复制拼图块区域内的像素
	for ry := 0; ry < PieceTotalH; ry++ {
		for rx := 0; rx < PieceTotalW; rx++ {
			if isInsidePiece(rx, ry) {
				px := targetX + rx
				py := targetY + ry
				if px >= 0 && px < CaptchaW && py >= 0 && py < CaptchaH {
					dst.Set(rx, ry, bg.At(px, py))
				}
			}
		}
	}

	// 绘制拼图块边框（白色不透明）
	for ry := 0; ry < PieceTotalH; ry++ {
		for rx := 0; rx < PieceTotalW; rx++ {
			if isOnPieceBorder(rx, ry) {
				dst.Set(rx, ry, color.RGBA{R: 255, G: 255, B: 255, A: 220})
			}
		}
	}

	return dst
}

// encodePNGBase64 将图片编码为base64 data URL
func encodePNGBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// --- 状态存储（Redis优先，降级进程内存）---

// storeCaptchaState 存储验证码状态
func storeCaptchaState(token string, state *captchaState) {
	if redis.Enabled() {
		ctx, cancel := redisCtx()
		defer cancel()
		key := "captcha:" + token
		data, _ := json.Marshal(state)
		redis.Client().Set(ctx, key, data, CaptchaTTL)
		return
	}
	// 降级：进程内存
	memCaptchaStore.Store(token, state)
}

// loadAndDeleteCaptchaState 读取并删除验证码状态（一次性消费）
func loadAndDeleteCaptchaState(token string) (*captchaState, bool) {
	if redis.Enabled() {
		ctx, cancel := redisCtx()
		defer cancel()
		key := "captcha:" + token
		// 先删除（防并发重放），再读取
		val, err := redis.Client().GetDel(ctx, key).Bytes()
		if err != nil || len(val) == 0 {
			return nil, false
		}
		var state captchaState
		if err := json.Unmarshal(val, &state); err != nil {
			return nil, false
		}
		return &state, true
	}
	// 降级：进程内存
	val, ok := memCaptchaStore.LoadAndDelete(token)
	if !ok {
		return nil, false
	}
	return val.(*captchaState), true
}

// storeVerifyToken 存储验证通过的一次性token
func storeVerifyToken(token string) {
	if redis.Enabled() {
		ctx, cancel := redisCtx()
		defer cancel()
		key := "captcha:verified:" + token
		redis.Client().Set(ctx, key, "1", VerifyTTL)
		return
	}
	// 降级：进程内存
	memVerifyStore.Store(token, time.Now().Add(VerifyTTL))
}

// consumeVerifyToken 消费验证token（一次性，取出即删除）
func consumeVerifyToken(token string) bool {
	if redis.Enabled() {
		ctx, cancel := redisCtx()
		defer cancel()
		key := "captcha:verified:" + token
		deleted, err := redis.Client().Del(ctx, key).Result()
		if err != nil || deleted == 0 {
			return false
		}
		return true
	}
	// 降级：进程内存
	val, ok := memVerifyStore.LoadAndDelete(token)
	if !ok {
		return false
	}
	expireAt := val.(time.Time)
	return time.Now().Before(expireAt)
}

// redisCtx 创建2秒超时的context（避免Redis卡住时阻塞请求）
func redisCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}
