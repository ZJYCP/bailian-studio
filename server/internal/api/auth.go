package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const authCookie = "bs_auth"

// authMiddleware 访问口令保护：未配置 ACCESS_TOKEN 时全放行（本地模式）。
// 校验顺序：Cookie（媒体文件 <img>/<video> 无法带自定义头，必须走 Cookie）→ X-Access-Token 头 → Bearer。
func authMiddleware(accessToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if accessToken == "" {
			c.Next()
			return
		}
		p := c.Request.URL.Path
		// 仅保护业务 API；静态前端资源放行（无 API 授权也拿不到任何数据）
		if !strings.HasPrefix(p, "/api/") {
			c.Next()
			return
		}
		if p == "/api/health" || p == "/api/auth" || p == "/api/auth/status" || p == "/api/auth/logout" {
			c.Next()
			return
		}
		presented := ""
		if ck, err := c.Cookie(authCookie); err == nil {
			presented = ck
		} else if h := c.GetHeader("X-Access-Token"); h != "" {
			presented = h
		} else if auth := c.GetHeader("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
			presented = auth[7:]
		}
		if subtle.ConstantTimeCompare([]byte(presented), []byte(accessToken)) == 1 {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "需要访问口令"})
	}
}

func authHandlers(r *gin.Engine, accessToken string) {
	// 状态探测：前端据此决定是否展示口令输入
	r.GET("/api/auth/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"auth_required": accessToken != "",
			"authenticated": accessToken == "" || cookieValid(c, accessToken),
		})
	})
	// 登录：校验口令并种 Cookie（HttpOnly；SameSite=Lax 兼容本机与同站部署）
	r.POST("/api/auth", func(c *gin.Context) {
		var req struct {
			Token string `json:"token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || subtle.ConstantTimeCompare([]byte(req.Token), []byte(accessToken)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "口令错误"})
			return
		}
		c.SetCookie(authCookie, accessToken, 365*24*3600, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.POST("/api/auth/logout", func(c *gin.Context) {
		c.SetCookie(authCookie, "", -1, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func cookieValid(c *gin.Context, accessToken string) bool {
	ck, err := c.Cookie(authCookie)
	return err == nil && subtle.ConstantTimeCompare([]byte(ck), []byte(accessToken)) == 1
}
