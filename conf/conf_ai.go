package conf

// AI 大模型配置（兼容 OpenAI 格式：DeepSeek / 通义 / 智谱 / Ollama 都能用）
type AI struct {
	Enabled bool   `yaml:"enabled"`  // 总开关，false 时所有功能直接降级跳过
	BaseURL string `yaml:"base_url"` // 例如 https://api.deepseek.com
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`           // 例如 deepseek-chat
	Timeout int    `yaml:"timeout_seconds"` // 单次请求超时，建议 60

	// 向量模型（语义搜索用）
	EmbeddingModel string `yaml:"embedding_model"` // 例如 embedding-3
	EmbeddingDim   int    `yaml:"embedding_dim"`   // 向量维度，建议 1024
}
