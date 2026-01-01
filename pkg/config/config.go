package config

import (
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

func Load() (*Config, error) {
	apiKey := os.Getenv("GHP_API_KEY")
	baseURL := os.Getenv("GHP_BASE_URL")
	model := os.Getenv("GHP_MODEL")

	if apiKey == "" {
		return nil, fmt.Errorf("请设置环境变量 GHP_API_KEY")
	}
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if model == "" {
		model = "deepseek-v3.2"
	}

	return &Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}, nil
}

func (c *Config) NewClientConfig() openai.ClientConfig {
	config := openai.DefaultConfig(c.APIKey)
	config.BaseURL = c.BaseURL
	return config
}
