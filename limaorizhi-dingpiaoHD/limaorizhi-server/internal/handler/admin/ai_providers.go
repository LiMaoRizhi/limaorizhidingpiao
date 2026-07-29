package admin

// 服务商列表直接写代码里了 变动不多懒得搞数据库
// 代码里写死比搞JSON直观 编译也能检查类型 写着写着服务商加了不少 都懒得改结构了
// 想参考别人的实现发现Go的几乎没有 呜呜呜

// ModelInfo 模型信息
type ModelInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Tag            string `json:"tag"`
	TagType        string `json:"tag_type"`
	SupportsVision bool   `json:"supports_vision"`
	// Icon 前端图标 brain=大脑 vision=眼睛 image=图片 空=没图标
	Icon string `json:"icon"`
}

// ProviderInfo 服务商信息
type ProviderInfo struct {
	Value     string      `json:"value"`
	Name      string      `json:"name"`
	Group     string      `json:"group"`      // nvidia | domestic
	GroupName string      `json:"group_name"` // 英伟达 | 国产
	BaseURL   string      `json:"base_url"`
	NeedsKey  bool        `json:"needs_key"`
	Tag       string      `json:"tag"`
	TagType   string      `json:"tag_type"`
	Hint      string      `json:"hint"`
	HasKey    bool        `json:"has_key"`
	Models    []ModelInfo `json:"models"`
}

// 英伟达一个Key能调所有模型 国产厂商各自要专属Key
// 旧模型名映射在Chat()里面 DB存旧名也能跑
var aiProviders = []ProviderInfo{
	// 英伟达
	{
		Value:     "nvidia",
		Name:      "英伟达 NIM",
		Group:     "nvidia",
		GroupName: "英伟达",
		BaseURL:   "https://integrate.api.nvidia.com/v1",
		NeedsKey:  true,
		Tag:       "免费",
		TagType:   "secondary",
		Hint:      "一个API Key免费调用所有英伟达模型",
		Models: []ModelInfo{
			// 狸猫员工：智能默认，自动选择最优可用模型，谁能用用谁
			// 支持对话/图片识别/图片生成/业务分析，模型下架自动降级
			{ID: "auto", Name: "狸猫员工", Description: "智能默认，自动选择最快可用模型，支持对话/图片识别/图片生成", Tag: "推荐", TagType: "primary", SupportsVision: true, Icon: "brain"},
			// 视觉模型按响应速度排序：小模型优先（谁先能识别图片让谁先上）
			{ID: "nvidia/llama-3.1-nemotron-nano-vl-8b-v1", Name: "Nemotron Nano VL 8B", Description: "8B超轻量视觉模型，图片识别极速响应", Tag: "极速", TagType: "info", SupportsVision: true, Icon: "vision"},
			{ID: "nvidia/cosmos-reason2-8b", Name: "Cosmos Reason2 8B", Description: "8B视觉推理模型，擅长物理世界理解", SupportsVision: true, Icon: "vision"},
			{ID: "meta/llama-3.2-11b-vision-instruct", Name: "Llama 3.2 11B Vision", Description: "11B视觉模型，图片推理能力强", SupportsVision: true, Icon: "vision"},
			{ID: "nvidia/nemotron-nano-12b-v2-vl", Name: "Nemotron Nano 12B VL", Description: "12B多图视觉理解，支持视频帧", SupportsVision: true, Icon: "vision"},
			{ID: "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning", Name: "Nemotron 3 Nano Omni", Description: "30B全模态推理，支持图片/视频/语音/文本", Tag: "推荐", TagType: "secondary", SupportsVision: true, Icon: "vision"},
			{ID: "moonshotai/kimi-k2.6", Name: "Kimi K2.6", Description: "1T多模态MoE，国产长上下文", SupportsVision: true, Icon: "vision"},
			{ID: "meta/llama-3.2-90b-vision-instruct", Name: "Llama 3.2 90B Vision", Description: "90B旗舰视觉模型，图片推理最强但较慢", SupportsVision: true, Icon: "vision"},
			// 纯文本对话模型
			{ID: "meta/llama-3.1-70b-instruct", Name: "Llama 3.1 70B", Description: "通用对话，响应快", Icon: "brain"},
			{ID: "meta/llama-3.2-3b-instruct", Name: "Llama 3.2 3B", Description: "轻量极速，边缘推理", Icon: "brain"},
			{ID: "nvidia/nemotron-3-super-120b-a12b", Name: "Nemotron 3 Super", Description: "NVIDIA旗舰，1M上下文", Icon: "brain"},
			{ID: "nvidia/nemotron-3-ultra-550b-a55b", Name: "Nemotron 3 Ultra", Description: "NVIDIA超旗舰，550B MoE，1M上下文", Tag: "旗舰", TagType: "primary", Icon: "brain"},
			{ID: "nvidia/llama-3.3-nemotron-super-49b-v1.5", Name: "Nemotron Super 49B v1.5", Description: "NVIDIA调优Llama，响应快", Icon: "brain"},
			{ID: "nvidia/llama-3.1-nemotron-nano-8b-v1", Name: "Nemotron Nano 8B", Description: "超轻量极速推理", Icon: "brain"},
			{ID: "google/gemma-4-31b-it", Name: "Gemma 4 31B", Description: "Google开源，前沿推理，编码强", Icon: "brain"},
			{ID: "mistralai/mistral-medium-3.5-128b", Name: "Mistral Medium 3.5", Description: "Mistral中型旗舰，编码与Agent", Icon: "brain"},
			{ID: "mistralai/mistral-nemotron", Name: "Mistral Nemotron", Description: "Mistral与NVIDIA联合调优", Icon: "brain"},
			{ID: "openai/gpt-oss-120b", Name: "GPT-OSS 120B", Description: "OpenAI开源版", Icon: "brain"},
			// 国产模型：英伟达上收费/慢，放最后
			{ID: "z-ai/glm-5.2", Name: "GLM 5.2", Description: "智谱国产，英伟达上较慢", Icon: "brain"},
			{ID: "deepseek-ai/deepseek-v4-pro", Name: "DeepSeek V4 Pro", Description: "国产推理强，但英伟达上较慢"},
			{ID: "deepseek-ai/deepseek-v4-flash", Name: "DeepSeek V4 Flash", Description: "284B MoE极速版，编码与Agent优化"},
			// 图片生成：走 Pollinations.ai 免费接口，无需 Key
			{ID: "flux-realistic", Name: "自然写实", Description: "照片级真实感，接近自然", Tag: "推荐", TagType: "secondary", Icon: "image"},
			{ID: "flux-portrait", Name: "人像摄影", Description: "专业人像摄影，背景虚化", Icon: "image"},
			{ID: "flux", Name: "FLUX 通用", Description: "高质量通用图片生成", Icon: "image"},
			{ID: "turbo", Name: "极速出图", Description: "秒级出图，快速预览", Icon: "image"},
			{ID: "sana", Name: "Sana", Description: "轻量高效图片生成", Icon: "image"},
		},
	},
	// 国产
	{
		Value:     "qwen",
		Name:      "千问3.8Max",
		Group:     "domestic",
		GroupName: "国产",
		BaseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
		NeedsKey:  true,
		Tag:       "主推预览版",
		TagType:   "primary",
		Hint:      "通义千问旗舰预览版，国产主推",
		Models: []ModelInfo{
			{ID: "qwen3.8-max", Name: "千问3.8Max", Description: "通义千问旗舰预览版", Tag: "主推预览版", TagType: "primary", Icon: "brain"},
		},
	},
	{
		Value:     "deepseek",
		Name:      "DeepSeek V4 Pro",
		Group:     "domestic",
		GroupName: "国产",
		BaseURL:   "https://api.deepseek.com/v1",
		NeedsKey:  true,
		Tag:       "推理强",
		TagType:   "secondary",
		Hint:      "DeepSeek V4 Pro，推理能力强，支持深度思考",
		Models: []ModelInfo{
			{ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro", Description: "推理强，支持深度思考", Tag: "推理强", TagType: "secondary"},
		},
	},
	{
		Value:     "kimi",
		Name:      "Kimi K3",
		Group:     "domestic",
		GroupName: "国产",
		BaseURL:   "https://api.moonshot.cn/v1",
		NeedsKey:  true,
		Hint:      "月之暗面Kimi K3",
		Models: []ModelInfo{
			{ID: "kimi-k3", Name: "Kimi K3", Description: "长上下文能力强", Icon: "brain"},
		},
	},
	{
		Value:     "glm",
		Name:      "GLM 5.2",
		Group:     "domestic",
		GroupName: "国产",
		BaseURL:   "https://open.bigmodel.cn/api/paas/v4",
		NeedsKey:  true,
		Hint:      "智谱GLM 5.2",
		Models: []ModelInfo{
			{ID: "glm-5.2", Name: "GLM 5.2", Description: "清华系，中文优秀", Icon: "brain"},
		},
	},
	{
		Value:     "doubao",
		Name:      "豆包2.1Pro",
		Group:     "domestic",
		GroupName: "国产",
		BaseURL:   "https://ark.cn-beijing.volces.com/api/v3",
		NeedsKey:  true,
		Tag:       "不推荐",
		TagType:   "warning",
		Hint:      "字节豆包2.1Pro，不推荐使用",
		Models: []ModelInfo{
			{ID: "doubao-2.1-pro", Name: "豆包2.1Pro", Description: "模型能力一般", Tag: "不推荐", TagType: "warning"},
		},
	},
	{
		Value:     "minimax",
		Name:      "MiniMax M3",
		Group:     "domestic",
		GroupName: "国产",
		BaseURL:   "https://api.minimax.chat/v1",
		NeedsKey:  true,
		Tag:       "不推荐",
		TagType:   "warning",
		Hint:      "MiniMax M3，不推荐使用",
		Models: []ModelInfo{
			{ID: "minimax-m3", Name: "MiniMax M3", Description: "模型能力一般", Tag: "不推荐", TagType: "warning"},
		},
	},
}

// providerDefaults 返回服务商默认的base_url和模型ID
func providerDefaults(provider string) (baseURL, model string) {
	for _, p := range aiProviders {
		if p.Value == provider {
			baseURL = p.BaseURL
			if len(p.Models) > 0 {
				model = p.Models[0].ID
			}
			return
		}
	}
	return
}
