package api

import (
	"context"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"bailian-studio/server/internal/config"
	"bailian-studio/server/internal/service"
)

type API struct {
	Cfg *config.Config
	Svc *service.Service
}

func NewRouter(cfg *config.Config, svc *service.Service, frontend fs.FS) *gin.Engine {
	a := &API{Cfg: cfg, Svc: svc}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg.AllowedOrigin))
	r.MaxMultipartMemory = 64 << 20

	// 访问口令（ACCESS_TOKEN 非空时启用，保护全部业务 API）
	r.Use(authMiddleware(cfg.AccessToken))
	authHandlers(r, cfg.AccessToken)

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	// Providers
	r.GET("/api/providers", a.listProviders)
	r.POST("/api/providers", a.createProvider)
	r.PUT("/api/providers/:id", a.updateProvider)
	r.DELETE("/api/providers/:id", a.deleteProvider)
	r.POST("/api/providers/:id/test", a.testProvider)

	// Models
	r.GET("/api/models", a.listModels)
	r.POST("/api/models", a.createModel)
	r.PUT("/api/models/:id", a.updateModel)
	r.DELETE("/api/models/:id", a.deleteModel)

	// Tasks
	r.POST("/api/tasks", a.createTask)
	r.GET("/api/tasks", a.listTasks)
	r.GET("/api/tasks/:id", a.getTask)
	r.POST("/api/tasks/:id/retry", a.retryTask)
	r.POST("/api/tasks/:id/cancel", a.cancelTask)
	r.DELETE("/api/tasks/:id", a.deleteTask)

	// Assets
	r.GET("/api/assets", a.listAssets)
	r.GET("/api/assets/:id/file", a.assetFile)
	r.GET("/api/assets/:id/download", a.assetDownload)
	r.DELETE("/api/assets/:id", a.deleteAsset)

	// 上传输入素材
	r.POST("/api/uploads", a.upload)

	// 音色（内置 + 复刻）
	r.GET("/api/voices", a.listVoices)
	r.POST("/api/voice-clones", a.createVoiceClone)

	// 前端静态资源（嵌入或磁盘）
	statics(r, frontend, cfg)

	return r
}

func statics(r *gin.Engine, frontend fs.FS, cfg *config.Config) {
	if frontend == nil {
		return
	}
	sub := frontend // 已是站点根（index.html 所在层级）
	lookup := func(p string) (fs.File, bool) {
		if p == "" {
			return nil, false
		}
		f, err := sub.Open(p)
		if err != nil {
			return nil, false
		}
		st, err := f.Stat()
		if err != nil || st.IsDir() {
			f.Close()
			return nil, false
		}
		return f, true
	}
	// 直接从嵌入 FS 输出（http.FileServer 会把 */index.html 301 到 ./，在 SPA 下成环）
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		f, ok := lookup(p)
		if !ok {
			p = "index.html" // SPA fallback
			f, ok = lookup(p)
			if !ok {
				c.String(http.StatusNotFound, "frontend not built")
				return
			}
		}
		defer f.Close()
		st, _ := f.Stat()
		ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(p)))
		if ct == "" {
			ct = "text/html; charset=utf-8"
		}
		c.DataFromReader(http.StatusOK, st.Size(), ct, f, nil)
	})
}

func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition, Accept-Ranges, Content-Range")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

var ctxTimeout = func() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Minute)
}
