package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"limaorizhi-server/internal/config"
	"limaorizhi-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// SaveImageFile 公共图片上传函数（供 admin 和 wx 包共用）
// 安全校验：扩展名白名单 + magic bytes 内容类型检测 + 随机文件名
func SaveImageFile(c *gin.Context) (string, string, bool) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, response.CodeParamError)
		return "", "", false
	}
	if file.Size > config.AppConfig.Upload.MaxSize {
		response.FailMsg(c, response.CodeParamError, "文件大小超过限制")
		return "", "", false
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExt := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowedExt[ext] {
		response.FailMsg(c, response.CodeParamError, "不支持的文件格式")
		return "", "", false
	}
	// 打开文件检测实际内容类型（magic bytes），防止伪造后缀名/Content-Type
	opened, err := file.Open()
	if err != nil {
		response.Fail(c, response.CodeServerError)
		return "", "", false
	}
	defer opened.Close()
	head := make([]byte, 512)
	n, _ := opened.Read(head)
	actualMIME := http.DetectContentType(head[:n])
	allowedMIME := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
	}
	if !allowedMIME[actualMIME] {
		response.FailMsg(c, response.CodeParamError, "文件内容不是合法图片（检测到: "+actualMIME+"）")
		return "", "", false
	}
	// 文件名使用纳秒时间戳 + 随机字节，增加不可预测性
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		response.FailMsg(c, response.CodeServerError, "生成随机文件名失败")
		return "", "", false
	}
	filename := fmt.Sprintf("%d%s%s", time.Now().UnixNano(), hex.EncodeToString(randomBytes), ext)
	uploadPath := config.AppConfig.Upload.Path
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		response.Fail(c, response.CodeServerError)
		return "", "", false
	}
	fullPath := filepath.Join(uploadPath, filename)
	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		response.Fail(c, response.CodeServerError)
		return "", "", false
	}
	url := config.AppConfig.Upload.URLPrefix + "/" + filename
	return url, filename, true
}
