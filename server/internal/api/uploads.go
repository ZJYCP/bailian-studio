package api

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"bailian-studio/server/internal/model"
	"bailian-studio/server/internal/service"
)

var kindByMimePrefix = map[string]string{
	"image/": "image",
	"video/": "video",
	"audio/": "audio",
}

func kindOf(mimeType, filename string) string {
	if k, ok := kindByMimePrefix[mimeType]; ok {
		return k
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp":
		return "image"
	case ".mp4", ".mov", ".webm", ".avi":
		return "video"
	case ".mp3", ".wav", ".opus", ".m4a", ".flac", ".aac":
		return "audio"
	}
	return ""
}

// upload 上传输入素材（multipart file），返回 Asset
func (a *API) upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "缺少文件字段 file")
		return
	}
	if fh.Size > 200<<20 {
		badRequest(c, "文件过大（限 200MB）")
		return
	}
	src, err := fh.Open()
	if err != nil {
		serverError(c, err)
		return
	}
	defer src.Close()

	name := service.SanitizeFilename(fh.Filename)
	buf := make([]byte, 512)
	n, _ := src.Read(buf)
	mimeType := ""
	if n > 0 {
		mimeType = http.DetectContentType(buf[:n])
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		serverError(c, err)
		return
	}
	kind := kindOf(mimeType, name)
	if kind == "" {
		badRequest(c, "不支持的文件类型（仅图片/视频/音频）")
		return
	}
	// 规范化 mime（DetectContentType 对 mp4 常给出 application/octet-stream）
	if strings.Contains(mimeType, "octet-stream") {
		mimeType = mime.TypeByExtension(filepath.Ext(name))
	}

	dir := filepath.Join("uploads", time.Now().Format("200601"))
	if err := os.MkdirAll(a.Svc.AbsPath(dir), 0o755); err != nil {
		serverError(c, err)
		return
	}
	rel := filepath.Join(dir, fmt.Sprintf("%s_%s", service.RandID(8), name))
	dst, err := os.Create(a.Svc.AbsPath(rel))
	if err != nil {
		serverError(c, err)
		return
	}
	size, err := io.Copy(dst, src)
	closeErr := dst.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		serverError(c, err)
		return
	}

	as := &model.Asset{
		Kind:     kind,
		Role:     "input",
		FilePath: rel,
		FileName: name,
		MIME:     mimeType,
		Size:     size,
	}
	if err := a.Svc.DB.Create(as).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, as)
}
