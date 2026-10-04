// Package seed 内置模型目录种子：参数 schema 驱动前端表单与后端请求组装。
// 协议与参数依据百炼官方文档核实；用户可在「模型管理」中启用/禁用、改默认值或新增自定义模型。
package seed

import (
	"encoding/json"

	"gorm.io/gorm"

	"bailian-studio/server/internal/model"
)

type Mode struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type InputSpec struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // image | video | audio | media(图或视频)
	Modes    []string `json:"modes,omitempty"`
	Min      int      `json:"min,omitempty"`
	Max      int      `json:"max,omitempty"`
	Required bool     `json:"required,omitempty"`
	Help     string   `json:"help,omitempty"`
}

type Field struct {
	Key          string            `json:"key"`
	Label        string            `json:"label"`
	Type         string            `json:"type"` // text textarea number boolean select voice
	Options      []string          `json:"options,omitempty"`
	OptionLabels map[string]string `json:"option_labels,omitempty"`
	AllowCustom  bool              `json:"allowCustom,omitempty"`
	Default      any               `json:"default,omitempty"`
	Min          *float64          `json:"min,omitempty"`
	Max          *float64          `json:"max,omitempty"`
	Step         float64           `json:"step,omitempty"`
	Required     bool              `json:"required,omitempty"`
	Help         string            `json:"help,omitempty"`
	Placeholder  string            `json:"placeholder,omitempty"`
}

type Schema struct {
	Modes  []Mode       `json:"modes"`
	Inputs []InputSpec  `json:"inputs,omitempty"`
	Fields []Field      `json:"fields"`
}

func numPtr(v float64) *float64 { return &v }

func f(key, label, typ string) Field  { return Field{Key: key, Label: label, Type: typ} }
func ta(key, label string) Field     { return f(key, label, "textarea") }
func sel(key, label string, opts ...string) Field {
	return Field{Key: key, Label: label, Type: "select", Options: opts, AllowCustom: true}
}
func num(key, label string, min, max, def float64) Field {
	return Field{Key: key, Label: label, Type: "number", Min: numPtr(min), Max: numPtr(max), Default: def}
}
func boolean(key, label string, def bool) Field {
	return Field{Key: key, Label: label, Type: "boolean", Default: def}
}

func mustJSON(s Schema) []byte {
	b, _ := json.Marshal(s)
	return b
}

// ---------- 常用尺寸与选项 ----------

var qwenImageSizes = []string{"1328*1328", "1664*928", "936*1664", "1472*1140", "1140*1472", "1024*1024", "2048*2048"}
var wan26Sizes = []string{"1280*1280", "1696*960", "960*1696", "1472*1104", "1104*1472"}
var wanxSizes = []string{"1024*1024", "1280*720", "720*1280", "1152*864", "864*1152", "1344*768", "768*1344", "1440*720", "720*1440"}
var resolutions = []string{"1080P", "720P", "480P"}
var ratios = []string{"adaptive", "16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}

// ---------- 图像模型 schema ----------

func qwenImageSchema(editOnly bool) Schema {
	modes := []Mode{{Key: "t2i", Label: "文生图"}, {Key: "i2i", Label: "图片编辑"}}
	if editOnly {
		modes = []Mode{{Key: "i2i", Label: "图片编辑"}}
	}
	fields := []Field{
		ta("negative_prompt", "负向提示词"),
		sel("size", "尺寸", qwenImageSizes...),
		num("n", "生成数量", 1, 6, 1),
		num("seed", "随机种子", 0, 2147483647, 0),
		boolean("prompt_extend", "提示词智能改写", true),
		boolean("watermark", "添加水印", false),
	}
	var inputs []InputSpec
	if editOnly {
		inputs = []InputSpec{{Key: "images", Label: "原图（1-3 张）", Type: "image", Modes: []string{"i2i"}, Min: 1, Max: 3, Required: true}}
	} else {
		inputs = []InputSpec{{Key: "images", Label: "参考图（可多张融合）", Type: "image", Modes: []string{"i2i"}, Max: 3}}
	}
	return Schema{Modes: modes, Inputs: inputs, Fields: fields}
}

func wanT2ISyncSchema() Schema {
	return Schema{
		Modes: []Mode{{Key: "t2i", Label: "文生图"}},
		Fields: []Field{
			ta("negative_prompt", "负向提示词"),
			sel("size", "尺寸", wan26Sizes...),
			num("n", "生成数量", 1, 4, 1),
			num("seed", "随机种子", 0, 2147483647, 0),
			boolean("prompt_extend", "提示词智能改写", true),
			boolean("watermark", "添加水印", false),
		},
	}
}

func wanxT2ILegacySchema() Schema {
	return Schema{
		Modes: []Mode{{Key: "t2i", Label: "文生图"}},
		Fields: []Field{
			ta("negative_prompt", "负向提示词"),
			sel("size", "尺寸", wanxSizes...),
			num("n", "生成数量", 1, 4, 1),
			num("seed", "随机种子", 0, 2147483647, 0),
			boolean("prompt_extend", "提示词智能改写", true),
			boolean("watermark", "添加水印", false),
		},
	}
}

func wanxImageEditSchema() Schema {
	return Schema{
		Modes: []Mode{{Key: "i2i", Label: "图片编辑"}},
		Inputs: []InputSpec{
			{Key: "base_image", Label: "原图", Type: "image", Modes: []string{"i2i"}, Min: 1, Max: 1, Required: true},
			{Key: "mask_image", Label: "蒙版图（局部重绘时必填）", Type: "image", Modes: []string{"i2i"}, Max: 1, Help: "仅 description_edit_with_mask 需要"},
		},
		Fields: []Field{
			{Key: "function", Label: "编辑功能", Type: "select", Options: []string{"description_edit", "description_edit_with_mask", "stylization_all", "expand", "super_resolution", "doodle"}, Default: "description_edit"},
			num("n", "生成数量", 1, 4, 1),
			ta("negative_prompt", "负向提示词"),
		},
	}
}

// ---------- 视频模型 schema ----------

func videoMediaSchema(modes []Mode, inputs []InputSpec, extra ...Field) Schema {
	fields := []Field{
		sel("resolution", "分辨率", resolutions...),
		sel("ratio", "画面比例", ratios...),
		{Key: "duration", Label: "时长（秒）", Type: "number", Min: numPtr(-1), Max: numPtr(30), Default: 5, Help: "-1 为智能时长（仅部分模型支持）"},
		boolean("audio", "生成音频", true),
		num("seed", "随机种子", 0, 2147483647, 0),
		boolean("prompt_extend", "提示词智能改写", true),
		boolean("watermark", "添加水印", false),
	}
	fields = append(fields, extra...)
	return Schema{Modes: modes, Inputs: inputs, Fields: fields}
}

func wan3Schema() Schema {
	return videoMediaSchema(
		[]Mode{{Key: "t2v", Label: "文生视频"}, {Key: "i2v", Label: "图生视频"}, {Key: "r2v", Label: "参考生视频"}},
		[]InputSpec{
			{Key: "first_frame", Label: "首帧图片", Type: "image", Modes: []string{"i2v"}, Max: 1},
			{Key: "last_frame", Label: "尾帧图片", Type: "image", Modes: []string{"i2v"}, Max: 1, Help: "与首帧一起使用为首尾帧模式"},
			{Key: "reference_images", Label: "参考图片（最多 10 张）", Type: "image", Modes: []string{"r2v"}, Max: 10},
			{Key: "reference_videos", Label: "参考视频（最多 5 个，总时长 ≤15s）", Type: "video", Modes: []string{"r2v"}, Max: 5},
			{Key: "reference_audios", Label: "参考音频（最多 5 个）", Type: "audio", Modes: []string{"r2v"}, Max: 5},
		},
	)
}

func wan27I2VSchema() Schema {
	return videoMediaSchema(
		[]Mode{{Key: "i2v", Label: "图生视频"}, {Key: "kf2v", Label: "首尾帧生视频"}},
		[]InputSpec{
			{Key: "first_frame", Label: "首帧图片", Type: "image", Modes: []string{"i2v", "kf2v"}, Min: 1, Max: 1, Required: true},
			{Key: "last_frame", Label: "尾帧图片", Type: "image", Modes: []string{"kf2v"}, Max: 1},
			{Key: "driving_audio", Label: "驱动音频（wav/mp3 2-30s）", Type: "audio", Max: 1},
		},
	)
}

func wan27R2VSchema() Schema {
	return videoMediaSchema(
		[]Mode{{Key: "r2v", Label: "参考生视频"}},
		[]InputSpec{
			{Key: "reference_images", Label: "参考图片", Type: "image", Modes: []string{"r2v"}, Max: 10, Required: true},
			{Key: "reference_videos", Label: "参考视频", Type: "video", Modes: []string{"r2v"}, Max: 5},
			{Key: "reference_audios", Label: "参考音频（配音）", Type: "audio", Modes: []string{"r2v"}, Max: 5},
		},
	)
}

func videoClassicFields(sizeInsteadOfRatio bool) []Field {
	fields := []Field{
		ta("negative_prompt", "负向提示词"),
		{Key: "resolution", Label: "分辨率", Type: "select", Options: []string{"1080P", "720P"}},
	}
	if sizeInsteadOfRatio {
		fields = append(fields, sel("size", "画面尺寸", "1280*720", "960*1696", "1696*960", "720*1280", "1280*1280"))
	} else {
		fields = append(fields, Field{Key: "ratio", Label: "画面比例", Type: "select", Options: []string{"16:9", "9:16", "1:1", "4:3", "3:4"}})
	}
	fields = append(fields,
		num("duration", "时长（秒）", 2, 15, 5),
		Field{Key: "shot_type", Label: "镜头类型", Type: "select", Options: []string{"single", "multi"}, OptionLabels: map[string]string{"single": "单镜头", "multi": "多镜头"}, Default: "single", Help: "需开启提示词智能改写"},
		boolean("audio", "生成音频", true),
		boolean("prompt_extend", "提示词智能改写", true),
		boolean("watermark", "添加水印", false),
		num("seed", "随机种子", 0, 2147483647, 0),
	)
	return fields
}

func wanClassicT2VSchema() Schema {
	return Schema{Modes: []Mode{{Key: "t2v", Label: "文生视频"}}, Fields: videoClassicFields(false)}
}

func wanClassicI2VSchema() Schema {
	return Schema{
		Modes: []Mode{{Key: "i2v", Label: "图生视频"}},
		Inputs: []InputSpec{
			{Key: "img_url", Label: "首帧图片", Type: "image", Modes: []string{"i2v"}, Min: 1, Max: 1, Required: true},
			{Key: "audio_url", Label: "配音/背景音乐（wav/mp3 3-30s）", Type: "audio", Modes: []string{"i2v"}, Max: 1},
		},
		Fields: videoClassicFields(false),
	}
}

func wanClassicR2VSchema() Schema {
	return Schema{
		Modes: []Mode{{Key: "r2v", Label: "参考生视频"}},
		Inputs: []InputSpec{
			{Key: "reference_urls", Label: "参考图片/视频（按 prompt 中顺序）", Type: "media", Modes: []string{"r2v"}, Min: 1, Max: 5, Required: true},
		},
		Fields: videoClassicFields(true),
	}
}

// ---------- TTS 模型 schema ----------

// cosyvoice 系系统音色（来自 bl CLI 官方列表）
var cosyVoices = []string{
	"longwan_v3", "longcheng_v3", "longhua_v3", "longtian_v3", "longze_v3", "longzhe_v3",
	"longyan_v3", "longxing_v3", "longqiang_v3", "longfeifei_v3", "longhao_v3", "longanrou_v3",
	"longanyang", "longanhuan_v3", "longantai_v3", "longanwen_v3", "longanli_v3", "longanlang_v3",
	"longanmin_v3", "longanyun_v3", "longanyue_v3", "longxiaochun_v3", "longxiaoxia_v3", "longyumi_v3",
	"longyingmu_v3", "longyingxun_v3", "longyingjing_v3", "longyingling_v3", "longyingtao_v3", "longyingxiao_v3",
	"longfei_v3", "longhuhu_v3", "longpaopao_v3", "longjielidou_v3", "longxian_v3", "longling_v3",
	"longshanshan_v3", "longniuniu_v3", "longjiaxin_v3", "longjiayi_v3", "longlaotie_v3", "longshange_v3",
	"loongabby_v3", "loongandy_v3", "loongannie_v3", "loongava_v3", "loongbeth_v3", "loongbetty_v3",
	"loongcally_v3", "loongcindy_v3", "loongdavid_v3", "loongdonna_v3", "loongemily_v3", "loongeric_v3",
	"loongluna_v3", "loongluca_v3", "loongriko_v3", "loongtomoka_v3", "loongtomoya_v3", "loongyuuna_v3",
	"loongyuuma_v3", "loongkyong_v3", "loongjihun_v3", "loongindah_v3",
}

var ttsLanguages = []string{"zh", "en", "ja", "ko", "fr", "de", "ru", "es", "it", "pt", "th", "id", "vi", "ms", "ar"}

func cosyvoiceSchema(defaultVoice string) Schema {
	voiceField := Field{Key: "voice", Label: "音色", Type: "voice", Options: cosyVoices, AllowCustom: true, Default: defaultVoice, Required: true, Help: "可直接输入复刻音色 ID"}
	return Schema{
		Modes: []Mode{{Key: "tts", Label: "语音合成"}},
		Fields: []Field{
			voiceField,
			{Key: "format", Label: "音频格式", Type: "select", Options: []string{"mp3", "wav", "pcm", "opus"}, Default: "mp3"},
			{Key: "sample_rate", Label: "采样率 (Hz)", Type: "select", Options: []string{"22050", "24000", "44100", "48000", "16000", "8000", "12000"}, Default: "22050"},
			num("volume", "音量", 0, 100, 50),
			{Key: "rate", Label: "语速", Type: "number", Min: numPtr(0.5), Max: numPtr(2), Step: 0.05, Default: 1},
			{Key: "pitch", Label: "音调", Type: "number", Min: numPtr(0.5), Max: numPtr(2), Step: 0.05, Default: 1},
			num("seed", "随机种子", 0, 65535, 0),
			sel("language", "语言提示", ttsLanguages...),
			ta("instruction", "风格指令（如：用温柔的语气）"),
			boolean("enable_ssml", "解析 SSML 标记", false),
		},
	}
}

// v3.5 系仅支持复刻/设计音色（实测系统音色报 418）
func cosyvoice35Schema() Schema {
	return Schema{
		Modes: []Mode{{Key: "tts", Label: "语音合成"}},
		Fields: []Field{
			{Key: "voice", Label: "音色（复刻/设计音色 ID）", Type: "voice", AllowCustom: true, Required: true, Placeholder: "填入复刻音色 ID", Help: "该模型仅支持声音复刻/声音设计得到的音色，可在「设置」创建"},
			{Key: "format", Label: "音频格式", Type: "select", Options: []string{"mp3", "wav", "pcm", "opus"}, Default: "mp3"},
			{Key: "sample_rate", Label: "采样率 (Hz)", Type: "select", Options: []string{"22050", "24000", "44100", "48000", "16000", "8000", "12000"}, Default: "22050"},
			num("volume", "音量", 0, 100, 50),
			{Key: "rate", Label: "语速", Type: "number", Min: numPtr(0.5), Max: numPtr(2), Step: 0.05, Default: 1},
			{Key: "pitch", Label: "音调", Type: "number", Min: numPtr(0.5), Max: numPtr(2), Step: 0.05, Default: 1},
			num("seed", "随机种子", 0, 65535, 0),
			sel("language", "语言提示", ttsLanguages...),
			ta("instruction", "风格指令（如：用温柔的语气）"),
			boolean("enable_ssml", "解析 SSML 标记", false),
		},
	}
}

// qwen-tts / qwen3-tts 系系统音色（来自官方音色列表页）
var qwenVoices = []string{
	"Cherry", "Serena", "Ethan", "Chelsie", "Momo", "Vivian", "Moon", "Maia", "Kai", "Nofish",
	"Bella", "Jennifer", "Ryan", "Katerina", "Aiden", "Eldric Sage", "Mia", "Mochi", "Bellona",
	"Vincent", "Bunny", "Neil", "Elias", "Arthur", "Nini", "Seren", "Pip", "Stella", "Bodega",
	"Sonrisa", "Alek", "Dolce", "Sohee", "Ono Anna", "Lenn", "Emilien", "Andre", "Radio Gol",
	"Jada", "Dylan", "Li", "Marcus", "Roy", "Peter", "Sunny", "Eric", "Rocky", "Kiki",
}

func qwenTTSSchema(instruct bool) Schema {
	fields := []Field{
		{Key: "voice", Label: "音色", Type: "voice", Options: qwenVoices, AllowCustom: true, Default: "Cherry", Required: true},
		{Key: "language_type", Label: "语言", Type: "select", Options: []string{"Auto", "Chinese", "English", "Japanese", "Korean", "French", "German", "Russian", "Italian", "Spanish", "Portuguese"}, Default: "Auto"},
	}
	if instruct {
		fields = append(fields, ta("instructions", "合成指令（自然语言控制情感/语气，中英文，≤1600 Token）"))
	}
	return Schema{Modes: []Mode{{Key: "tts", Label: "语音合成"}}, Fields: fields}
}

func qwenAudioTTSSchema() Schema {
	s := cosyvoiceSchema("longanlingxin")
	// qwen-audio-3.0-tts 系统音色不同，且不支持 instruction
	var fields []Field
	for _, fd := range s.Fields {
		if fd.Key == "voice" {
			fd.Options = []string{"longanlingxin", "longanlufeng"}
			fd.Default = "longanlingxin"
		}
		if fd.Key == "instruction" {
			continue
		}
		fields = append(fields, fd)
	}
	s.Fields = fields
	return s
}

// ---------- 目录 ----------

type entry struct {
	code, name, capability, protocol string
	schema                            Schema
	sort                              int
	isDefault                         bool
	remark                            string
}

func catalog() []entry {
	return []entry{
		// 图像
		{code: "qwen-image-3.0-pro", name: "Qwen-Image-3.0-Pro（旗舰，图文排版最强）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(false), sort: 1, isDefault: true, remark: "支持 4.5K token 复杂提示词，文生图/图生图/编辑一体"},
		{code: "qwen-image-3.0", name: "Qwen-Image-3.0（标准版）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(false), sort: 2},
		{code: "qwen-image-max", name: "Qwen-Image-Max（真实感最强）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(false), sort: 3},
		{code: "qwen-image-2.0-pro", name: "Qwen-Image-2.0-Pro", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(false), sort: 4},
		{code: "qwen-image-2.0", name: "Qwen-Image-2.0（加速版）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(false), sort: 5},
		{code: "qwen-image-edit", name: "Qwen-Image-Edit（编辑专精）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(true), sort: 6},
		{code: "qwen-image-edit-plus", name: "Qwen-Image-Edit-Plus（编辑加速版）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(true), sort: 7},
		{code: "wan2.7-image", name: "Wan2.7-Image（万相生成编辑一体）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(false), sort: 8},
		{code: "wan2.6-t2i", name: "Wan2.6 文生图", capability: "image", protocol: model.ProtoImageSync, schema: wanT2ISyncSchema(), sort: 9},
		{code: "z-image-turbo", name: "Z-Image-Turbo（高速写实）", capability: "image", protocol: model.ProtoImageSync, schema: qwenImageSchema(false), sort: 10},
		{code: "wanx2.1-t2i-turbo", name: "万相 2.1 文生图 Turbo", capability: "image", protocol: model.ProtoImageLegacy, schema: wanxT2ILegacySchema(), sort: 11},
		{code: "wanx2.1-t2i-plus", name: "万相 2.1 文生图 Plus", capability: "image", protocol: model.ProtoImageLegacy, schema: wanxT2ILegacySchema(), sort: 12},
		{code: "wanx2.1-imageedit", name: "万相 2.1 图片编辑（指令/风格化/扩图/超分）", capability: "image", protocol: model.ProtoImageEditAsync, schema: wanxImageEditSchema(), sort: 13},

		// 视频
		{code: "wan3.0-video", name: "Wan3.0-Video（全能参考，推荐）", capability: "video", protocol: model.ProtoVideoMedia, schema: wan3Schema(), sort: 1, isDefault: true, remark: "最长 30s/30fps，支持首尾帧/多参考/参考音频/文件"},
		{code: "wan3.0-video-prime", name: "Wan3.0-Video-Prime（高速版）", capability: "video", protocol: model.ProtoVideoMedia, schema: wan3Schema(), sort: 2},
		{code: "wan2.7-t2v", name: "Wan2.7 文生视频", capability: "video", protocol: model.ProtoVideoClassic, schema: wanClassicT2VSchema(), sort: 3},
		{code: "wan2.6-t2v", name: "Wan2.6 文生视频", capability: "video", protocol: model.ProtoVideoClassic, schema: wanClassicT2VSchema(), sort: 4},
		{code: "wan2.7-i2v", name: "Wan2.7 图生视频（首帧/首尾帧/音频驱动）", capability: "video", protocol: model.ProtoVideoMedia, schema: wan27I2VSchema(), sort: 5},
		{code: "wan2.6-i2v", name: "Wan2.6 图生视频", capability: "video", protocol: model.ProtoVideoClassic, schema: wanClassicI2VSchema(), sort: 6},
		{code: "wan2.6-i2v-flash", name: "Wan2.6 图生视频 Flash（高性价比）", capability: "video", protocol: model.ProtoVideoClassic, schema: wanClassicI2VSchema(), sort: 7},
		{code: "wan2.7-r2v", name: "Wan2.7 参考生视频", capability: "video", protocol: model.ProtoVideoMedia, schema: wan27R2VSchema(), sort: 8},
		{code: "wan2.6-r2v-flash", name: "Wan2.6 参考生视频 Flash", capability: "video", protocol: model.ProtoVideoClassic, schema: wanClassicR2VSchema(), sort: 9},
		{code: "happyhorse-1.1-t2v", name: "HappyHorse 1.1 文生视频", capability: "video", protocol: model.ProtoVideoClassic, schema: wanClassicT2VSchema(), sort: 10},
		{code: "happyhorse-1.1-i2v", name: "HappyHorse 1.1 图生视频", capability: "video", protocol: model.ProtoVideoMedia, schema: wan27I2VSchema(), sort: 11},

		// 语音合成
		{code: "cosyvoice-v3-flash", name: "CosyVoice v3 Flash（推荐，66 系统音色）", capability: "tts", protocol: model.ProtoTTSHTTP, schema: cosyvoiceSchema("longwan_v3"), sort: 1, isDefault: true},
		{code: "cosyvoice-v3-plus", name: "CosyVoice v3 Plus（效果优先）", capability: "tts", protocol: model.ProtoTTSHTTP, schema: cosyvoiceSchema("longwan_v3"), sort: 2},
		{code: "cosyvoice-v3.5-flash", name: "CosyVoice v3.5 Flash（仅复刻/设计音色）", capability: "tts", protocol: model.ProtoTTSHTTP, schema: cosyvoice35Schema(), sort: 3},
		{code: "cosyvoice-v3.5-plus", name: "CosyVoice v3.5 Plus（仅复刻/设计音色）", capability: "tts", protocol: model.ProtoTTSHTTP, schema: cosyvoice35Schema(), sort: 4},
		{code: "qwen-audio-3.0-tts-plus", name: "Qwen-Audio-3.0-TTS Plus（方言/情感）", capability: "tts", protocol: model.ProtoTTSHTTP, schema: qwenAudioTTSSchema(), sort: 5},
		{code: "qwen-audio-3.0-tts-flash", name: "Qwen-Audio-3.0-TTS Flash", capability: "tts", protocol: model.ProtoTTSHTTP, schema: qwenAudioTTSSchema(), sort: 6},
		{code: "qwen3-tts-flash", name: "Qwen3-TTS Flash（17 音色）", capability: "tts", protocol: model.ProtoTTSQwen, schema: qwenTTSSchema(false), sort: 7},
		{code: "qwen3-tts-instruct-flash", name: "Qwen3-TTS Instruct Flash（自然语言控制）", capability: "tts", protocol: model.ProtoTTSQwen, schema: qwenTTSSchema(true), sort: 8},
		{code: "qwen-tts", name: "Qwen-TTS（中英混合）", capability: "tts", protocol: model.ProtoTTSQwen, schema: qwenTTSSchema(false), sort: 9},
	}
}

// Seed 写入模型目录（只插入缺失的，不覆盖用户修改）
func Seed(db *gorm.DB) error {
	for _, e := range catalog() {
		var count int64
		if err := db.Model(&model.ModelDef{}).Where("code = ?", e.code).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		m := model.ModelDef{
			Code:        e.code,
			Name:        e.name,
			Capability:  e.capability,
			Protocol:    e.protocol,
			ParamSchema: mustJSON(e.schema),
			Enabled:     true,
			IsDefault:   e.isDefault,
			Sort:        e.sort,
			Remark:      e.remark,
		}
		if err := db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}
