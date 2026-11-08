package ai_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-star/global"

	"github.com/sirupsen/logrus"
)

// Message 是 OpenAI 兼容格式的一轮对话
type Message struct {
	Role    string `json:"role"` // system / user / assistant
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Chat 是最基础的封装：给一组消息，返回模型回复
// 所有 AI 功能都必须通过它调用，不要在别处直接发 HTTP
func Chat(ctx context.Context, messages []Message) (string, error) {
	cfg := global.Config.AI
	if !cfg.Enabled || cfg.APIKey == "" {
		return "", fmt.Errorf("ai 未配置")
	}

	// json.Marshal：把结构体转为json字符串
	body, _ := json.Marshal(chatRequest{
		Model:       cfg.Model,
		Messages:    messages,
		Temperature: 0.3, // 摘要/标签/审核都需要稳定输出，用低温度
	})

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", fmt.Errorf("解析响应失败: %w, body=%s", err, string(raw))
	}
	if cr.Error != nil {
		return "", fmt.Errorf("模型返回错误: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("模型没有返回内容，body=%s", string(raw))
	}
	if cr.Usage.TotalTokens > 0 {
		logrus.Infof("[AI] 模型=%s 输入=%d 输出=%d 总计=%d token",
			cfg.Model, cr.Usage.PromptTokens, cr.Usage.CompletionTokens, cr.Usage.TotalTokens)
	}
	return cr.Choices[0].Message.Content, nil
}
