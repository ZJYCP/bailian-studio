package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"bailian-studio/server/internal/model"
)

func (a *API) listModels(c *gin.Context) {
	var ms []model.ModelDef
	q := a.Svc.DB.Order("capability").Order("sort").Order("id")
	if cap := c.Query("capability"); cap != "" {
		q = q.Where("capability = ?", cap)
	}
	if c.Query("enabled") == "true" {
		q = q.Where("enabled = ?", true)
	}
	q.Find(&ms)
	c.JSON(http.StatusOK, ms)
}

type modelReq struct {
	Code        string         `json:"code" binding:"required"`
	Name        string         `json:"name"`
	Capability  string         `json:"capability" binding:"required"`
	Protocol    string         `json:"protocol" binding:"required"`
	ParamSchema datatypes.JSON `json:"param_schema"`
	Enabled     *bool          `json:"enabled"`
	IsDefault   *bool          `json:"is_default"`
	Sort        int            `json:"sort"`
	Remark      string         `json:"remark"`
}

func (a *API) createModel(c *gin.Context) {
	var req modelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}
	m := model.ModelDef{
		Code: req.Code, Name: req.Name, Capability: req.Capability, Protocol: req.Protocol,
		ParamSchema: req.ParamSchema, Enabled: true, IsDefault: req.IsDefault != nil && *req.IsDefault,
		Sort: req.Sort, Remark: req.Remark,
	}
	if m.ParamSchema == nil {
		badRequest(c, "param_schema 不能为空")
		return
	}
	if err := a.Svc.DB.Create(&m).Error; err != nil {
		badRequest(c, "创建失败（模型 code 可能重复）")
		return
	}
	if m.IsDefault {
		a.Svc.DB.Model(&model.ModelDef{}).Where("capability = ? AND id <> ?", m.Capability, m.ID).Update("is_default", false)
	}
	c.JSON(http.StatusOK, m)
}

func (a *API) updateModel(c *gin.Context) {
	var m model.ModelDef
	if err := a.Svc.DB.First(&m, c.Param("id")).Error; err != nil {
		notFound(c, "模型不存在")
		return
	}
	var req modelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}
	if req.Name != "" {
		m.Name = req.Name
	}
	if req.ParamSchema != nil {
		m.ParamSchema = req.ParamSchema
	}
	if req.Sort != 0 {
		m.Sort = req.Sort
	}
	if req.Remark != "" {
		m.Remark = req.Remark
	}
	if req.Enabled != nil {
		m.Enabled = *req.Enabled
	}
	if req.IsDefault != nil {
		m.IsDefault = *req.IsDefault
		if m.IsDefault {
			a.Svc.DB.Model(&model.ModelDef{}).Where("capability = ? AND id <> ?", m.Capability, m.ID).Update("is_default", false)
		}
	}
	a.Svc.DB.Save(&m)
	c.JSON(http.StatusOK, m)
}

func (a *API) deleteModel(c *gin.Context) {
	var m model.ModelDef
	if err := a.Svc.DB.First(&m, c.Param("id")).Error; err != nil {
		notFound(c, "模型不存在")
		return
	}
	a.Svc.DB.Delete(&m)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
