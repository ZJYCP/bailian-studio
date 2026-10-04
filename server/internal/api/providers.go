package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"bailian-studio/server/internal/model"
)

type providerDTO struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	KeyMask   string `json:"key_mask"`
	Remark    string `json:"remark"`
	IsDefault bool   `json:"is_default"`
}

func maskKey(k string) string {
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}

func toDTO(p *model.Provider) providerDTO {
	return providerDTO{
		ID: p.ID, Name: p.Name, BaseURL: p.BaseURL,
		KeyMask: maskKey(p.APIKey), Remark: p.Remark, IsDefault: p.IsDefault,
	}
}

func (a *API) listProviders(c *gin.Context) {
	var ps []model.Provider
	a.Svc.DB.Order("id").Find(&ps)
	out := make([]providerDTO, 0, len(ps))
	for i := range ps {
		out = append(out, toDTO(&ps[i]))
	}
	c.JSON(http.StatusOK, out)
}

type providerReq struct {
	Name      string `json:"name" binding:"required"`
	BaseURL   string `json:"base_url" binding:"required"`
	APIKey    string `json:"api_key"`
	Remark    string `json:"remark"`
	IsDefault bool   `json:"is_default"`
}

func (a *API) createProvider(c *gin.Context) {
	var req providerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}
	p := model.Provider{Name: req.Name, BaseURL: strings.TrimRight(req.BaseURL, "/"), APIKey: req.APIKey, Remark: req.Remark, IsDefault: req.IsDefault}
	if err := a.Svc.DB.Create(&p).Error; err != nil {
		serverError(c, err)
		return
	}
	if p.IsDefault {
		a.clearOtherDefaults(model.Provider{}, p.ID)
	}
	c.JSON(http.StatusOK, toDTO(&p))
}

func (a *API) updateProvider(c *gin.Context) {
	var p model.Provider
	if err := a.Svc.DB.First(&p, c.Param("id")).Error; err != nil {
		notFound(c, "Provider 不存在")
		return
	}
	var req providerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}
	p.Name = req.Name
	p.BaseURL = strings.TrimRight(req.BaseURL, "/")
	if req.APIKey != "" && !strings.HasPrefix(req.APIKey, "****") {
		p.APIKey = req.APIKey
	}
	p.Remark = req.Remark
	p.IsDefault = req.IsDefault
	a.Svc.DB.Save(&p)
	if p.IsDefault {
		a.clearOtherDefaults(model.Provider{}, p.ID)
	}
	c.JSON(http.StatusOK, toDTO(&p))
}

func (a *API) deleteProvider(c *gin.Context) {
	var p model.Provider
	if err := a.Svc.DB.First(&p, c.Param("id")).Error; err != nil {
		notFound(c, "Provider 不存在")
		return
	}
	var count int64
	a.Svc.DB.Model(&model.Provider{}).Count(&count)
	if count <= 1 {
		badRequest(c, "至少保留一个服务配置")
		return
	}
	a.Svc.DB.Delete(&p)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) testProvider(c *gin.Context) {
	var p model.Provider
	if err := a.Svc.DB.First(&p, c.Param("id")).Error; err != nil {
		notFound(c, "Provider 不存在")
		return
	}
	ctx, cancel := ctxTimeout()
	defer cancel()
	client := a.Svc.ClientForProvider(&p)
	if err := client.Test(ctx); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) clearOtherDefaults(_ model.Provider, keepID uint) {
	a.Svc.DB.Model(&model.Provider{}).Where("id <> ?", keepID).Update("is_default", false)
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}
func notFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, gin.H{"error": msg})
}
func serverError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
