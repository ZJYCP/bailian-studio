package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bailian-studio/server/internal/dashscope"
	"bailian-studio/server/internal/model"
)

// CreateTaskInput 创建任务的入参
type CreateTaskInput struct {
	ProviderID uint           `json:"provider_id"`
	ModelCode  string         `json:"model_code"`
	Mode       string         `json:"mode"`
	Prompt     string         `json:"prompt"`
	Params     map[string]any `json:"params"`
	Inputs     map[string][]uint `json:"inputs"` // input key -> asset IDs
}

// CreateTask 创建并异步分发任务
func (s *Service) CreateTask(in CreateTaskInput) (*model.Task, error) {
	var m model.ModelDef
	if err := s.DB.Where("code = ? AND enabled = ?", in.ModelCode, true).First(&m).Error; err != nil {
		return nil, fmt.Errorf("模型 %s 不可用", in.ModelCode)
	}
	provider, err := s.pickProvider(in.ProviderID)
	if err != nil {
		return nil, err
	}

	var refs []InputRef
	for role, ids := range in.Inputs {
		for _, id := range ids {
			refs = append(refs, InputRef{AssetID: id, Role: role})
		}
	}

	task := &model.Task{
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ModelCode:    m.Code,
		Capability:   m.Capability,
		Mode:         in.Mode,
		Status:       model.TaskQueued,
		Prompt:       in.Prompt,
		Params:       toJSON(cleanParams(in.Params)),
		InputAssets:  toJSON(refs),
	}
	if err := s.DB.Create(task).Error; err != nil {
		return nil, err
	}

	go s.dispatch(task.ID)
	return task, nil
}

func (s *Service) pickProvider(id uint) (*model.Provider, error) {
	if id > 0 {
		return s.ProviderByID(id)
	}
	return s.DefaultProvider()
}

// RetryTask 以相同参数重新创建任务
func (s *Service) RetryTask(id uint) (*model.Task, error) {
	var old model.Task
	if err := s.DB.First(&old, id).Error; err != nil {
		return nil, fmt.Errorf("任务不存在")
	}
	var params map[string]any
	_ = json.Unmarshal(old.Params, &params)
	inputs := map[string][]uint{}
	for _, r := range parseInputRefs(old.InputAssets) {
		inputs[r.Role] = append(inputs[r.Role], r.AssetID)
	}
	return s.CreateTask(CreateTaskInput{
		ProviderID: old.ProviderID,
		ModelCode:  old.ModelCode,
		Mode:       old.Mode,
		Prompt:     old.Prompt,
		Params:     params,
		Inputs:     inputs,
	})
}

// dispatch 按协议调用百炼（goroutine 中执行）
func (s *Service) dispatch(taskID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var task model.Task
	if err := s.DB.First(&task, taskID).Error; err != nil {
		return
	}
	provider, err := s.ProviderByID(task.ProviderID)
	if err != nil {
		s.failTask(&task, "provider_missing", err.Error())
		return
	}
	var m model.ModelDef
	if err := s.DB.Where("code = ?", task.ModelCode).First(&m).Error; err != nil {
		s.failTask(&task, "model_missing", err.Error())
		return
	}

	now := time.Now()
	task.Status = model.TaskSubmitting
	task.SubmittedAt = &now
	s.DB.Save(&task)

	client := s.ClientForProvider(provider)
	err = s.callProtocol(ctx, client, &task, &m)
	if err != nil {
		s.failTask(&task, errorCode(err), err.Error())
		return
	}
}

func errorCode(err error) string {
	var apiErr *dashscope.APIError
	if e, ok := err.(*dashscope.APIError); ok {
		apiErr = e
	}
	if apiErr != nil && apiErr.Code != "" {
		return apiErr.Code
	}
	return "internal_error"
}

func (s *Service) failTask(task *model.Task, code, msg string) {
	now := time.Now()
	task.Status = model.TaskFailed
	task.ErrorCode = code
	task.ErrorMessage = msg
	task.FinishedAt = &now
	s.DB.Save(task)
	log.Printf("[task %d] 失败: %s: %s", task.ID, code, msg)
}

// callProtocol 组装并执行各协议请求
func (s *Service) callProtocol(ctx context.Context, c *dashscope.Client, task *model.Task, m *model.ModelDef) error {
	var params map[string]any
	_ = json.Unmarshal(task.Params, &params)
	params = cleanParams(params)
	refs := parseInputRefs(task.InputAssets)

	switch m.Protocol {
	case model.ProtoImageSync:
		imgs, err := s.resolveInputs(ctx, c, task, refs, "images", true)
		if err != nil {
			return err
		}
		intFields(params, "n", "seed")
		resp, err := c.ImageSync(ctx, m.Code, task.Prompt, imgs, params)
		if err != nil {
			return err
		}
		task.RequestID = resp.RequestID
		task.Usage = toJSON(resp.Usage)
		var urls []string
		for _, ch := range resp.Output.Choices {
			for _, ct := range ch.Message.Content {
				if ct.Image != "" {
					urls = append(urls, ct.Image)
				}
			}
		}
		return s.finishWithDownloads(ctx, task, urls, "image")

	case model.ProtoImageLegacy:
		input := map[string]any{"prompt": task.Prompt}
		if np, ok := params["negative_prompt"]; ok {
			input["negative_prompt"] = np
			delete(params, "negative_prompt")
		}
		intFields(params, "n", "seed")
		resp, err := c.ImageLegacyAsync(ctx, m.Code, input, params)
		if err != nil {
			return err
		}
		return s.awaitAsync(task, resp)

	case model.ProtoImageEditAsync:
		baseImgs, err := s.resolveInputs(ctx, c, task, refs, "base_image", false)
		if err != nil {
			return err
		}
		if len(baseImgs) == 0 {
			return fmt.Errorf("缺少原图（base_image）")
		}
		maskImgs, _ := s.resolveInputs(ctx, c, task, refs, "mask_image", false)
		function, _ := params["function"].(string)
		if function == "" {
			function = "description_edit"
		}
		delete(params, "function")
		intFields(params, "n", "seed")
		mask := ""
		if len(maskImgs) > 0 {
			mask = maskImgs[0]
		}
		resp, err := c.ImageEditAsync(ctx, m.Code, function, task.Prompt, baseImgs[0], mask, params)
		if err != nil {
			return err
		}
		return s.awaitAsync(task, resp)

	case model.ProtoVideoMedia:
		media, err := s.buildMedia(ctx, c, task, refs)
		if err != nil {
			return err
		}
		intFields(params, "duration", "seed")
		resp, err := c.VideoMediaAsync(ctx, m.Code, task.Prompt, media, params)
		if err != nil {
			return err
		}
		return s.awaitAsync(task, resp)

	case model.ProtoVideoClassic:
		input := map[string]any{}
		if task.Prompt != "" {
			input["prompt"] = task.Prompt
		}
		if np, ok := params["negative_prompt"]; ok {
			input["negative_prompt"] = np
			delete(params, "negative_prompt")
		}
		if urls, err := s.resolveInputs(ctx, c, task, refs, "img_url", true); err != nil {
			return err
		} else if len(urls) > 0 {
			input["img_url"] = urls[0]
		}
		if urls, err := s.resolveInputs(ctx, c, task, refs, "audio_url", false); err != nil {
			return err
		} else if len(urls) > 0 {
			input["audio_url"] = urls[0]
		}
		if urls, err := s.resolveInputs(ctx, c, task, refs, "reference_urls", false); err != nil {
			return err
		} else if len(urls) > 0 {
			input["reference_urls"] = urls
		}
		intFields(params, "duration", "seed")
		resp, err := c.VideoClassicAsync(ctx, m.Code, input, params)
		if err != nil {
			return err
		}
		return s.awaitAsync(task, resp)

	case model.ProtoTTSHTTP:
		input := map[string]any{"text": task.Prompt}
		for k, v := range params {
			input[k] = v
		}
		if sr, ok := input["sample_rate"]; ok {
			if i, ok2 := toInt(sr); ok2 {
				input["sample_rate"] = i
			}
		}
		if v, ok := input["volume"]; ok {
			if i, ok2 := toInt(v); ok2 {
				input["volume"] = i
			}
		}
		if i, ok2 := toInt(input["seed"]); ok2 {
			input["seed"] = i
		}
		if lang, ok := input["language"].(string); ok && lang != "" {
			input["language_hints"] = []string{lang}
			delete(input, "language")
		}
		resp, err := c.TTSHTTP(ctx, m.Code, input)
		if err != nil {
			return err
		}
		task.RequestID = resp.RequestID
		task.Usage = toJSON(map[string]any{"characters": resp.Usage.Characters})
		return s.finishWithDownloads(ctx, task, []string{resp.Output.Audio.URL}, "audio")

	case model.ProtoTTSQwen:
		input := map[string]any{"text": task.Prompt}
		for k, v := range params {
			input[k] = v
		}
		resp, err := c.TTSQwen(ctx, m.Code, input)
		if err != nil {
			return err
		}
		task.RequestID = resp.RequestID
		task.Usage = toJSON(map[string]any{"characters": resp.Usage.Characters})
		return s.finishWithDownloads(ctx, task, []string{resp.Output.Audio.URL}, "audio")

	default:
		return fmt.Errorf("未知协议: %s", m.Protocol)
	}
}

// awaitAsync 记录远端任务 ID，交给轮询器跟踪
func (s *Service) awaitAsync(task *model.Task, resp *dashscope.AsyncSubmitResponse) error {
	if resp.Output.TaskID == "" {
		return fmt.Errorf("未返回任务 ID: %s", resp.Output.Message)
	}
	task.ExternalTaskID = resp.Output.TaskID
	task.RequestID = resp.RequestID
	task.Status = model.TaskRunning
	return s.DB.Save(task).Error
}

// finishWithDownloads 同步协议完成后立即下载产物落盘
func (s *Service) finishWithDownloads(ctx context.Context, task *model.Task, urls []string, kind string) error {
	var kept []string
	for _, u := range urls {
		if u != "" {
			kept = append(kept, u)
		}
	}
	task.ResultURLs = toJSON(kept)
	if len(kept) == 0 {
		return fmt.Errorf("接口未返回结果 URL")
	}
	for i, u := range kept {
		if _, err := s.downloadOutput(ctx, task, u, kind, i); err != nil {
			return fmt.Errorf("下载产物失败: %w", err)
		}
	}
	now := time.Now()
	task.Status = model.TaskSucceeded
	task.FinishedAt = &now
	return s.DB.Save(task).Error
}

// buildMedia wan3/wan2.7 media 协议：input key -> media type
var mediaTypes = map[string]string{
	"first_frame":      "first_frame",
	"last_frame":       "last_frame",
	"reference_images": "reference_image",
	"reference_videos": "reference_video",
	"reference_audios": "reference_audio",
	"driving_audio":    "driving_audio",
}

func (s *Service) buildMedia(ctx context.Context, c *dashscope.Client, task *model.Task, refs []InputRef) ([]dashscope.MediaItem, error) {
	var media []dashscope.MediaItem
	for key, typ := range mediaTypes {
		// driving_audio 在 wan2.7-i2v 中的 media type 为 driving_audio
		urls, err := s.resolveInputs(ctx, c, task, refs, key, false)
		if err != nil {
			return nil, err
		}
		for _, u := range urls {
			media = append(media, dashscope.MediaItem{Type: typ, URL: u})
		}
	}
	return media, nil
}

// resolveInputs 把输入素材解析为百炼可用的 URL：
// 图片 ≤10MB 直接 base64 内联，其余上传百炼临时 OSS
func (s *Service) resolveInputs(ctx context.Context, c *dashscope.Client, task *model.Task, refs []InputRef, role string, inlineImage bool) ([]string, error) {
	var urls []string
	for _, r := range refs {
		if r.Role != role {
			continue
		}
		var a model.Asset
		if err := s.DB.First(&a, r.AssetID).Error; err != nil {
			return nil, fmt.Errorf("输入素材 %d 不存在", r.AssetID)
		}
		path := s.absPath(a.FilePath)
		fi, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("输入文件缺失: %s", a.FileName)
		}
		if inlineImage && a.Kind == "image" && fi.Size() <= 10<<20 {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			mime := a.MIME
			if mime == "" {
				mime = "image/png"
			}
			urls = append(urls, "data:"+mime+";base64,"+base64.StdEncoding.EncodeToString(data))
			continue
		}
		ossURL, err := s.uploadToDashScope(ctx, c, task.ModelCode, path, a.FileName)
		if err != nil {
			return nil, err
		}
		urls = append(urls, ossURL)
	}
	return urls, nil
}

func (s *Service) uploadToDashScope(ctx context.Context, c *dashscope.Client, modelCode, path, name string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// 避免同名覆盖导致 OSS 409
	safe := RandID(6) + "_" + sanitizeFilename(name)
	return c.UploadFile(ctx, modelCode, safe, f)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	var b strings.Builder
	for _, r := range name {
		if r > 31 && r != '/' && r != '\\' && r != '"' && r < 127 {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "file"
	}
	return b.String()
}

// downloadOutput 下载生成产物到本地资产目录
func (s *Service) downloadOutput(ctx context.Context, task *model.Task, url, kind string, index int) (*model.Asset, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		a, err := s.tryDownload(ctx, task, url, kind, index)
		if err == nil {
			return a, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 3 * time.Second):
		}
	}
	return nil, lastErr
}

func (s *Service) tryDownload(ctx context.Context, task *model.Task, rawURL, kind string, index int) (*model.Asset, error) {
	dlCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("下载失败 HTTP %d", resp.StatusCode)
	}

	ext := extFromURL(rawURL, kind)
	dir := filepath.Join("assets", time.Now().Format("200601"))
	absDir := s.absPath(dir)
	if err := ensureDir(absDir); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("task%d_%d_%s.%s", task.ID, index+1, RandID(4), ext)
	rel := filepath.Join(dir, name)
	abs := s.absPath(rel)

	tmp := abs + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	size, err := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, abs); err != nil {
		return nil, err
	}

	a := &model.Asset{
		TaskID:    &task.ID,
		Kind:      kind,
		Role:      "output",
		FilePath:  rel,
		FileName:  name,
		MIME:      resp.Header.Get("Content-Type"),
		Size:      size,
		SourceURL: rawURL,
	}
	if a.MIME == "" {
		a.MIME = mimeByKind(kind, ext)
	}
	if kind == "image" {
		probeImage(abs, a)
	}
	if err := s.DB.Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func extFromURL(u, kind string) string {
	idx := strings.IndexAny(u, "?#")
	path := u
	if idx > 0 {
		path = u[:idx]
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	switch ext {
	case "png", "jpg", "jpeg", "webp", "gif", "bmp", "mp4", "mov", "webm", "mp3", "wav", "opus", "pcm", "m4a", "flac":
		return ext
	}
	return extByKind[kind]
}

func mimeByKind(kind, ext string) string {
	if m := map[string]string{
		"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg", "webp": "image/webp", "gif": "image/gif",
		"mp4": "video/mp4", "mov": "video/quicktime", "webm": "video/webm",
		"mp3": "audio/mpeg", "wav": "audio/wav", "opus": "audio/opus", "m4a": "audio/mp4", "flac": "audio/flac",
	}[ext]; m != "" {
		return m
	}
	return kind + "/octet-stream"
}

// SaveUsageRecord 保存任务 usage（轮询器用）
func (s *Service) SaveUsageRecord(task *model.Task, usage map[string]any) {
	if len(usage) > 0 {
		task.Usage = toJSON(usage)
	}
}
