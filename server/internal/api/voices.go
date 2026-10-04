package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"bailian-studio/server/internal/model"
)

type Voice struct {
	ID     string `json:"id"`
	Source string `json:"source"` // system | custom
	Status string `json:"status,omitempty"`
}

// listVoices 合并内置系统音色（模型 schema）与复刻音色（customization 接口）
func (a *API) listVoices(c *gin.Context) {
	modelCode := c.Query("model")
	if modelCode == "" {
		badRequest(c, "缺少 model 参数")
		return
	}
	var m model.ModelDef
	if err := a.Svc.DB.Where("code = ?", modelCode).First(&m).Error; err != nil {
		notFound(c, "模型不存在")
		return
	}

	out := []Voice{}
	// 内置音色来自 schema 的 voice 字段
	var schema struct {
		Fields []struct {
			Type    string   `json:"type"`
			Options []string `json:"options"`
		} `json:"fields"`
	}
	_ = json.Unmarshal(m.ParamSchema, &schema)
	for _, f := range schema.Fields {
		if f.Type == "voice" {
			for _, v := range f.Options {
				out = append(out, Voice{ID: v, Source: "system"})
			}
			break
		}
	}

	// 复刻音色
	if pid := c.Query("provider_id"); pid != "" || m.Protocol == model.ProtoTTSHTTP || m.Protocol == model.ProtoTTSQwen {
		var provider *model.Provider
		var err error
		if pid != "" {
			id := uint(0)
			for _, r := range pid {
				if r < '0' || r > '9' {
					id = 0
					break
				}
				id = id*10 + uint(r-'0')
			}
			provider, err = a.Svc.ProviderByID(id)
		} else {
			provider, err = a.Svc.DefaultProvider()
		}
		if err == nil {
			client := a.Svc.ClientForProvider(provider)
			ctx, cancel := ctxTimeout()
			defer cancel()
			resp, err := client.VoiceAction(ctx, "voice-enrollment", map[string]any{
				"action":     "list_voices",
				"page_size":  100,
				"page_index": 0,
			})
			if err == nil {
				var o struct {
					Output struct {
						VoiceList []struct {
							VoiceID     string `json:"voice_id"`
							Voice       string `json:"voice"`
							Status      string `json:"status"`
							TargetModel string `json:"target_model"`
						} `json:"voice_list"`
					} `json:"output"`
				}
				if json.Unmarshal(mustJSON(resp), &o) == nil {
					for _, v := range o.Output.VoiceList {
						id := v.VoiceID
						if id == "" {
							id = v.Voice
						}
						if id == "" || v.Status == "DELETED" {
							continue
						}
						out = append(out, Voice{ID: id, Source: "custom", Status: v.Status})
					}
				}
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"voices": out})
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

type voiceCloneReq struct {
	ProviderID   uint   `json:"provider_id"`
	TargetModel  string `json:"target_model" binding:"required"` // 如 cosyvoice-v3.5-flash
	Prefix       string `json:"prefix" binding:"required"`       // ≤10 字符，字母数字
	AudioAssetID uint   `json:"audio_asset_id" binding:"required"`
	Language     string `json:"language"`
}

// createVoiceClone 声音复刻：上传音频 → customization create_voice
func (a *API) createVoiceClone(c *gin.Context) {
	var req voiceCloneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}
	prefix := strings.TrimSpace(req.Prefix)
	if len(prefix) > 10 {
		badRequest(c, "prefix 最长 10 字符")
		return
	}
	provider, err := a.pickProvider(req.ProviderID)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	voiceID, err := a.Svc.CreateVoiceClone(c.Request.Context(), provider, req.TargetModel, prefix, req.AudioAssetID, req.Language)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"voice_id": voiceID})
}

func (a *API) pickProvider(id uint) (*model.Provider, error) {
	if id > 0 {
		return a.Svc.ProviderByID(id)
	}
	return a.Svc.DefaultProvider()
}
