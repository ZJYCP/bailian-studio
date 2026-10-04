package model

import (
	"time"

	"gorm.io/datatypes"
)

// Provider 百炼服务配置（apiUrl + API Key，可配多个）
type Provider struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	BaseURL   string    `gorm:"size:255;not null" json:"base_url"`
	APIKey    string    `gorm:"size:255;not null" json:"-"`
	HasKey    bool      `gorm:"-" json:"has_key"`
	KeyMask   string    `gorm:"-" json:"key_mask"`
	Remark    string    `gorm:"size:500" json:"remark"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ModelDef 模型目录：参数 schema 驱动前端表单与后端请求组装
type ModelDef struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Code        string         `gorm:"size:100;uniqueIndex;not null" json:"code"`
	Name        string         `gorm:"size:200" json:"name"`
	Capability  string         `gorm:"size:20;index;not null" json:"capability"` // image | video | tts
	Protocol    string         `gorm:"size:40;not null" json:"protocol"`
	ParamSchema datatypes.JSON `json:"param_schema"`
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	IsDefault   bool           `json:"is_default"`
	Sort        int            `json:"sort"`
	Remark      string         `gorm:"size:500" json:"remark"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Task 一次创作任务
type Task struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ProviderID     uint           `json:"provider_id"`
	ProviderName   string         `gorm:"size:100" json:"provider_name"`
	ModelCode      string         `gorm:"size:100;index" json:"model_code"`
	Capability     string         `gorm:"size:20;index" json:"capability"`
	Mode           string         `gorm:"size:20" json:"mode"` // t2i/i2i/t2v/i2v/r2v/kf2v/tts
	Status         string         `gorm:"size:20;index" json:"status"`
	Prompt         string         `gorm:"type:text" json:"prompt"`
	Params         datatypes.JSON `json:"params"`
	InputAssets    datatypes.JSON `json:"input_assets"` // [{"asset_id":1,"role":"img_url"}]
	ExternalTaskID string         `gorm:"size:100;index" json:"external_task_id"`
	RequestID      string         `gorm:"size:100" json:"request_id"`
	ErrorCode      string         `gorm:"size:200" json:"error_code"`
	ErrorMessage   string         `gorm:"type:text" json:"error_message"`
	Usage          datatypes.JSON `json:"usage"`
	ResultURLs     datatypes.JSON `json:"result_urls"`
	SubmittedAt    *time.Time     `json:"submitted_at"`
	FinishedAt     *time.Time     `json:"finished_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`

	Outputs []Asset `gorm:"-" json:"outputs,omitempty"` // 生成产物
	Inputs  []Asset `gorm:"-" json:"inputs,omitempty"`  // 输入素材
}

// Asset 媒体文件：生成产物（output）与上传素材（input）
type Asset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    *uint     `gorm:"index" json:"task_id"`
	Kind      string    `gorm:"size:20;index" json:"kind"` // image | video | audio
	Role      string    `gorm:"size:20" json:"role"`       // input | output
	FilePath  string    `gorm:"size:500" json:"file_path"` // 相对 DataDir
	FileName  string    `gorm:"size:255" json:"file_name"`
	MIME      string    `gorm:"size:100" json:"mime"`
	Size      int64     `json:"size"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Duration  float64   `json:"duration"`
	SourceURL string    `gorm:"size:1500" json:"source_url"`
	CreatedAt time.Time `json:"created_at"`
}

// 任务状态机: queued -> submitting -> running -> succeeded | failed | canceled
const (
	TaskQueued     = "queued"
	TaskSubmitting = "submitting"
	TaskRunning    = "running"
	TaskSucceeded  = "succeeded"
	TaskFailed     = "failed"
	TaskCanceled   = "canceled"
)

// 协议常量
const (
	ProtoImageSync       = "image_sync"        // multimodal-generation 同步（qwen-image 系/wan2.6+/wan2.7-image/z-image）
	ProtoImageLegacy     = "image_async_legacy" // text2image/image-synthesis 异步（wanx2.1/2.2/2.5-t2i）
	ProtoImageEditAsync  = "image_edit_async"   // image2image/image-synthesis 异步（wanx-imageedit/wan2.5-i2i）
	ProtoVideoMedia      = "video_media"        // video-synthesis + media[]（wan3.0/wan2.7-i2v,r2v/happyhorse-i2v,r2v）
	ProtoVideoClassic    = "video_classic"      // video-synthesis + img_url/reference_urls（wan2.6/wan2.7-t2v/happyhorse-t2v）
	ProtoTTSHTTP         = "tts_http"           // audio/tts/SpeechSynthesizer（cosyvoice/qwen-audio-tts）
	ProtoTTSQwen         = "tts_qwen"           // multimodal-generation（qwen3-tts）
)

func (t *Task) Active() bool {
	return t.Status == TaskQueued || t.Status == TaskSubmitting || t.Status == TaskRunning
}
