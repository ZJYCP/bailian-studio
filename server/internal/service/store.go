package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"bailian-studio/server/internal/config"
	"bailian-studio/server/internal/dashscope"
	"bailian-studio/server/internal/model"
)

type Service struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{DB: db, Cfg: cfg}
}

// ClientForProvider 构造指定 Provider 的百炼客户端
func (s *Service) ClientForProvider(p *model.Provider) *dashscope.Client {
	return dashscope.New(p.BaseURL, p.APIKey)
}

// DefaultProvider 返回默认（或第一个）Provider
func (s *Service) DefaultProvider() (*model.Provider, error) {
	var p model.Provider
	if err := s.DB.Where("is_default = ?", true).First(&p).Error; err == nil {
		return &p, nil
	}
	if err := s.DB.First(&p).Error; err == nil {
		return &p, nil
	}
	return nil, fmt.Errorf("尚未配置百炼服务（apiUrl / API Key），请前往「设置」添加")
}

func (s *Service) ProviderByID(id uint) (*model.Provider, error) {
	var p model.Provider
	if err := s.DB.First(&p, id).Error; err != nil {
		return nil, fmt.Errorf("Provider 不存在")
	}
	return &p, nil
}

func (s *Service) absPath(rel string) string {
	return filepath.Join(s.Cfg.DataDir, rel)
}

// AbsPath 数据目录下的绝对路径
func (s *Service) AbsPath(rel string) string { return s.absPath(rel) }

// RemoveAssetFile 删除资产对应的本地文件（记录由调用方删除）
func (s *Service) RemoveAssetFile(a *model.Asset) {
	if a.FilePath == "" {
		return
	}
	_ = os.Remove(s.absPath(a.FilePath))
}

// ---------- 输入文件引用 ----------

type InputRef struct {
	AssetID uint   `json:"asset_id"`
	Role    string `json:"role"`
}

func parseInputRefs(raw []byte) []InputRef {
	var refs []InputRef
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &refs)
	}
	return refs
}

// ---------- 参数清洗 ----------

// cleanParams 去掉空字符串/nil 值
func cleanParams(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		switch t := v.(type) {
		case string:
			if strings.TrimSpace(t) != "" {
				out[k] = t
			}
		case nil:
		default:
			out[k] = v
		}
	}
	return out
}

func toInt(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int:
		return int64(t), true
	case int64:
		return t, true
	case json.Number:
		i, err := t.Int64()
		return i, err == nil
	case string:
		var i int64
		_, err := fmt.Sscanf(t, "%d", &i)
		return i, err == nil
	}
	return 0, false
}

// intFields 把 float64（JSON 默认数值类型）规整为整数
func intFields(m map[string]any, keys ...string) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if i, ok2 := toInt(v); ok2 {
				m[k] = i
			}
		}
	}
}

func toJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ---------- 文件名安全化 ----------

var extByKind = map[string]string{"image": "png", "video": "mp4", "audio": "mp3"}

// RandID 随机十六进制串
func RandID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SanitizeFilename 清理文件名中的非法字符
func SanitizeFilename(name string) string {
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

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
