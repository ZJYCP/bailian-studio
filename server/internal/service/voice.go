package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"bailian-studio/server/internal/model"
)

var prefixRe = regexp.MustCompile(`^[a-z0-9]{1,10}$`)

// CreateVoiceClone 声音复刻：音频上传百炼 OSS 后调 customization create_voice
func (s *Service) CreateVoiceClone(ctx context.Context, provider *model.Provider, targetModel, prefix string, audioAssetID uint, language string) (string, error) {
	if !prefixRe.MatchString(prefix) {
		return "", fmt.Errorf("prefix 仅支持 1-10 位小写字母/数字")
	}
	var audio model.Asset
	if err := s.DB.First(&audio, audioAssetID).Error; err != nil {
		return "", fmt.Errorf("音频素材不存在")
	}
	if audio.Kind != "audio" {
		return "", fmt.Errorf("请上传音频文件（wav/mp3 等）")
	}
	path := s.absPath(audio.FilePath)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("音频文件缺失")
	}

	client := s.ClientForProvider(provider)
	ossURL, err := s.uploadToDashScope(ctx, client, "voice-enrollment", path, audio.FileName)
	if err != nil {
		return "", fmt.Errorf("上传音频失败: %w", err)
	}

	input := map[string]any{
		"action":        "create_voice",
		"target_model":  targetModel,
		"prefix":        prefix,
		"url":           ossURL,
	}
	if language != "" {
		input["language_hints"] = []string{language}
	}
	resp, err := client.VoiceAction(ctx, "voice-enrollment", input)
	if err != nil {
		return "", err
	}
	var out struct {
		Output struct {
			VoiceID string `json:"voice_id"`
		} `json:"output"`
	}
	b, _ := json.Marshal(resp)
	if err := json.Unmarshal(b, &out); err != nil || out.Output.VoiceID == "" {
		return "", fmt.Errorf("复刻结果解析失败: %s", string(b))
	}
	return out.Output.VoiceID, nil
}
