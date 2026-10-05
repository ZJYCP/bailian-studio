// Package dashscope 封装阿里云百炼（DashScope）媒体生成 HTTP API。
// 端点与参数依据百炼官方文档核实，协议划分见 model.ModelDef 的协议常量。
package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client 百炼 API 客户端（对应一个 Provider 配置）
type Client struct {
	BaseURL string // 如 https://dashscope.aliyuncs.com（不含 /api/v1）
	APIKey  string
	HTTP    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 180 * time.Second},
	}
}

// APIError 百炼错误（DashScope 协议: {code, message, request_id}）
type APIError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	Body       string `json:"-"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("百炼接口错误 [%s]: %s (request_id=%s)", e.Code, e.Message, e.RequestID)
	}
	return fmt.Sprintf("百炼接口错误 (HTTP %d): %s", e.HTTPStatus, e.Body)
}

func (c *Client) endpoint(path string) string {
	return c.BaseURL + path
}

// doJSON 发送 JSON 请求并解析响应；非 2xx 或业务 code 非空时返回 *APIError
func (c *Client) doJSON(ctx context.Context, method, path string, headers map[string]string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	// oss:// 临时链接必须带此头才会被服务端解析（缺失时报 "url scheme must be http/https"）
	req.Header.Set("X-DashScope-OssResourceResolve", "enable")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("请求百炼失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	apiErr := &APIError{HTTPStatus: resp.StatusCode, Body: string(raw)}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, apiErr)
	}
	if resp.StatusCode >= 300 || (apiErr.Code != "" && apiErr.Code != "0") {
		return apiErr
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("解析响应失败: %w (body=%.200s)", err, string(raw))
		}
	}
	return nil
}

// ---------- 通用响应结构 ----------

type usage = map[string]any

// AsyncSubmitResponse 异步任务提交结果
type AsyncSubmitResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		Code       string `json:"code"`
		Message    string `json:"message"`
	} `json:"output"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TaskStatus 异步任务查询结果（GET /api/v1/tasks/{id}）
type TaskStatus struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"` // PENDING RUNNING SUCCEEDED FAILED CANCELED UNKNOWN
		// 图像异步任务：results[].url；视频：video_url；部分音频：audio_url
		Results []struct {
			URL string `json:"url"`
		} `json:"results"`
		VideoURL string `json:"video_url"`
		AudioURL string `json:"audio_url"`
		Code     string `json:"code"`
		Message  string `json:"message"`
	} `json:"output"`
	Usage usage `json:"usage"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GetTask 查询异步任务状态
func (c *Client) GetTask(ctx context.Context, taskID string) (*TaskStatus, error) {
	var out TaskStatus
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/tasks/"+taskID, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- 协议 1: image_sync（multimodal-generation 同步）----------

// MessageContent qwen-image 系消息内容：图片在前、文本在后
type MessageContent struct {
	Image string `json:"image,omitempty"`
	Text  string `json:"text,omitempty"`
}

type ImageSyncRequest struct {
	Model  string `json:"model"`
	Input  ImageSyncInput  `json:"input"`
	Params map[string]any `json:"parameters"`
}

type ImageSyncInput struct {
	Messages []struct {
		Role    string          `json:"role"`
		Content []MessageContent `json:"content"`
	} `json:"messages"`
}

type ImageSyncResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Choices []struct {
			Message struct {
				Content []struct {
					Image string `json:"image"`
				} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage usage `json:"usage"`
}

// ImageSync 文生图/图生图（编辑）同步调用，images 为空即文生图
func (c *Client) ImageSync(ctx context.Context, modelCode, prompt string, images []string, params map[string]any) (*ImageSyncResponse, error) {
	msg := struct {
		Role    string           `json:"role"`
		Content []MessageContent `json:"content"`
	}{Role: "user"}
	for _, img := range images {
		msg.Content = append(msg.Content, MessageContent{Image: img})
	}
	msg.Content = append(msg.Content, MessageContent{Text: prompt})
	req := ImageSyncRequest{
		Model:  modelCode,
		Input:  ImageSyncInput{Messages: []struct {
			Role    string           `json:"role"`
			Content []MessageContent `json:"content"`
		}{msg}},
		Params: params,
	}
	var out ImageSyncResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/aigc/multimodal-generation/generation", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- 协议 2: image_async_legacy（text2image/image-synthesis 异步）----------

type ImageLegacyRequest struct {
	Model  string         `json:"model"`
	Input  map[string]any `json:"input"`
	Params map[string]any `json:"parameters"`
}

// ImageLegacyAsync wanx/wan2.2/wan2.5 系文生图（异步）。input: prompt/negative_prompt
func (c *Client) ImageLegacyAsync(ctx context.Context, modelCode string, input map[string]any, params map[string]any) (*AsyncSubmitResponse, error) {
	req := ImageLegacyRequest{
		Model:  modelCode,
		Input:  input,
		Params: params,
	}
	var out AsyncSubmitResponse
	hdr := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/aigc/text2image/image-synthesis", hdr, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- 协议 3: image_edit_async（image2image/image-synthesis 异步）----------

// ImageEditAsync wanx-imageedit / wan2.5-i2i 图片编辑（异步）
func (c *Client) ImageEditAsync(ctx context.Context, modelCode, function, prompt, baseImageURL, maskURL string, params map[string]any) (*AsyncSubmitResponse, error) {
	input := map[string]any{
		"function":       function,
		"prompt":         prompt,
		"base_image_url": baseImageURL,
	}
	if maskURL != "" {
		input["mask_image_url"] = maskURL
	}
	req := ImageLegacyRequest{Model: modelCode, Input: input, Params: params}
	var out AsyncSubmitResponse
	hdr := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/aigc/image2image/image-synthesis", hdr, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- 协议 4: video_media（video-synthesis + media[]，wan3.0/wan2.7/happyhorse 系）----------

type MediaItem struct {
	Type string `json:"type"` // first_frame last_frame reference_image reference_video reference_audio
	URL  string `json:"url"`
}

type VideoMediaRequest struct {
	Model  string         `json:"model"`
	Input  map[string]any `json:"input"`
	Params map[string]any `json:"parameters"`
}

// VideoMediaAsync 全能参考视频生成（异步）。media 可为空（纯文生）。
func (c *Client) VideoMediaAsync(ctx context.Context, modelCode, prompt string, media []MediaItem, params map[string]any) (*AsyncSubmitResponse, error) {
	input := map[string]any{}
	if prompt != "" {
		input["prompt"] = prompt
	}
	if len(media) > 0 {
		input["media"] = media
	}
	req := VideoMediaRequest{Model: modelCode, Input: input, Params: params}
	var out AsyncSubmitResponse
	hdr := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/aigc/video-generation/video-synthesis", hdr, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- 协议 5: video_classic（video-synthesis + img_url/reference_urls，wan2.6 系）----------

// VideoClassicAsync 经典视频生成（异步）。inputKeys: img_url / audio_url / reference_urls
func (c *Client) VideoClassicAsync(ctx context.Context, modelCode string, inputKeys map[string]any, params map[string]any) (*AsyncSubmitResponse, error) {
	req := VideoMediaRequest{Model: modelCode, Input: inputKeys, Params: params}
	var out AsyncSubmitResponse
	hdr := map[string]string{"X-DashScope-Async": "enable"}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/aigc/video-generation/video-synthesis", hdr, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- 协议 6/7: TTS ----------

type TTSRequest struct {
	Model string         `json:"model"`
	Input map[string]any `json:"input"`
}

type TTSResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Audio struct {
			URL string `json:"url"`
		} `json:"audio"`
		FinishReason string `json:"finish_reason"`
	} `json:"output"`
	Usage struct {
		Characters int `json:"characters"`
	} `json:"usage"`
}

// TTSHTTP cosyvoice / qwen-audio-tts 系（POST audio/tts/SpeechSynthesizer，参数全部在 input）
func (c *Client) TTSHTTP(ctx context.Context, modelCode string, input map[string]any) (*TTSResponse, error) {
	req := TTSRequest{Model: modelCode, Input: input}
	var out TTSResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/audio/tts/SpeechSynthesizer", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TTSQwen qwen3-tts 系（multimodal-generation，input.text/voice/...）
func (c *Client) TTSQwen(ctx context.Context, modelCode string, input map[string]any) (*TTSResponse, error) {
	req := TTSRequest{Model: modelCode, Input: input}
	var out TTSResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/aigc/multimodal-generation/generation", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- OSS 临时上传（本地输入文件 → oss:// URL）----------

type UploadPolicy struct {
	RequestID string `json:"request_id"`
	Data      struct {
		Policy               string `json:"policy"`
		Signature            string `json:"signature"`
		UploadDir            string `json:"upload_dir"`
		UploadHost           string `json:"upload_host"`
		OSSAccessKeyID       string `json:"oss_access_key_id"`
		XOSSObjectACL        string `json:"x_oss_object_acl"`
		XOSSForbidOverwrite  string `json:"x_oss_forbid_overwrite"`
	} `json:"data"`
}

// GetUploadPolicy 获取上传凭证（文件与模型绑定，48h 有效）
func (c *Client) GetUploadPolicy(ctx context.Context, modelCode string) (*UploadPolicy, error) {
	q := url.Values{"action": {"getPolicy"}, "model": {modelCode}}
	var out UploadPolicy
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/uploads?"+q.Encode(), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UploadFile 上传文件到百炼临时 OSS，返回 oss:// URL（调用模型时需带 X-DashScope-OssResourceResolve: enable）
func (c *Client) UploadFile(ctx context.Context, modelCode, filename string, content io.Reader) (string, error) {
	pol, err := c.GetUploadPolicy(ctx, modelCode)
	if err != nil {
		return "", fmt.Errorf("获取上传凭证失败: %w", err)
	}
	key := pol.Data.UploadDir + "/" + filename
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	// file 必须为最后一个表单域
	for k, v := range map[string]string{
		"OSSAccessKeyId":        pol.Data.OSSAccessKeyID,
		"Signature":             pol.Data.Signature,
		"policy":                pol.Data.Policy,
		"x-oss-object-acl":      pol.Data.XOSSObjectACL,
		"x-oss-forbid-overwrite": pol.Data.XOSSForbidOverwrite,
		"key":                   key,
		"success_action_status": "200",
	} {
		if err := w.WriteField(k, v); err != nil {
			return "", err
		}
	}
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, content); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pol.Data.UploadHost, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %w", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("上传文件失败 (HTTP %d): %s", resp.StatusCode, string(rb))
	}
	return "oss://" + key, nil
}

// ---------- 声音复刻（audio/tts/customization）----------

// VoiceAction 声音复刻操作：create_voice / list_voices / query_voice / delete_voice
func (c *Client) VoiceAction(ctx context.Context, enrollmentModel string, input map[string]any) (map[string]any, error) {
	req := map[string]any{"model": enrollmentModel, "input": input}
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/services/audio/tts/customization", nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---------- 连通性测试 ----------

// Test 用上传凭证接口验证 base_url + api_key 有效性
func (c *Client) Test(ctx context.Context) error {
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := c.GetUploadPolicy(ctx2, "qwen-image-3.0")
	return err
}
