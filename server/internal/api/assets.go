package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"bailian-studio/server/internal/model"
)

func (a *API) listAssets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "60"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 60
	}
	q := a.Svc.DB.Model(&model.Asset{})
	if k := c.Query("kind"); k != "" {
		q = q.Where("kind = ?", k)
	}
	if r := c.Query("role"); r != "" {
		q = q.Where("role = ?", r)
	}
	if tid := c.Query("task_id"); tid != "" {
		q = q.Where("task_id = ?", tid)
	}
	var total int64
	q.Count(&total)
	var assets []model.Asset
	q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&assets)
	c.JSON(http.StatusOK, gin.H{"total": total, "items": assets, "page": page, "page_size": size})
}

// assetFile 流式返回文件（支持 Range，视频拖动播放必需）
func (a *API) assetFile(c *gin.Context) {
	var as model.Asset
	if err := a.Svc.DB.First(&as, c.Param("id")).Error; err != nil {
		notFound(c, "文件不存在")
		return
	}
	path := a.Svc.AbsPath(as.FilePath)
	if _, err := os.Stat(path); err != nil {
		notFound(c, "文件已丢失（可能被手动清理）")
		return
	}
	if dl := c.Query("download"); dl == "1" || c.Request.URL.Query().Has("download") {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, as.FileName))
	}
	c.Header("Content-Type", as.MIME)
	http.ServeFile(c.Writer, c.Request, path) // 自带 Range/If-Modified-Since
}

func (a *API) assetDownload(c *gin.Context) {
	var as model.Asset
	if err := a.Svc.DB.First(&as, c.Param("id")).Error; err != nil {
		notFound(c, "文件不存在")
		return
	}
	path := a.Svc.AbsPath(as.FilePath)
	if _, err := os.Stat(path); err != nil {
		notFound(c, "文件已丢失")
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, as.FileName))
	c.Header("Content-Type", as.MIME)
	http.ServeFile(c.Writer, c.Request, path)
}

func (a *API) deleteAsset(c *gin.Context) {
	var as model.Asset
	if err := a.Svc.DB.First(&as, c.Param("id")).Error; err != nil {
		notFound(c, "文件不存在")
		return
	}
	a.Svc.RemoveAssetFile(&as)
	a.Svc.DB.Delete(&as)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
