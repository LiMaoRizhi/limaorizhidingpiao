// limaorizhi-server  狸猫日志售票系统  联系微信：lihao68681818
package wx

import (
	"github.com/gin-gonic/gin"

	"limaorizhi-server/internal/pkg/response"
	"limaorizhi-server/internal/pkg/upload"
)

// Upload 小程序用户端文件上传（头像等图片）
func (h *UserHandler) Upload(c *gin.Context) {
	url, filename, ok := upload.SaveImageFile(c)
	if !ok {
		return // SaveImageFile 内部已返回错误响应
	}
	response.OK(c, gin.H{"url": url, "filename": filename})
}
