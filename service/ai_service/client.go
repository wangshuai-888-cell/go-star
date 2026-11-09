package ai_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"go-star/conf"
	"go-star/global"

	"github.com/sirupsen/logrus"
)

// maxRetry 重试次数（不含首次调用）
// 免费档 RPM 约 5-10，2 次重试足以消化瞬时 429
const maxRetry = 2

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
		Code    string `json:"code"` // 智谱用字符串错误码，如 "1302"
	} `json:"error"`
}

// retryableCode 判断业务错误码是否值得重试
// 白名单机制：只有「过一会儿可能恢复」的码才重试
// 鉴权(1000~1004)、余额(1110)、参数(1200/1210)、内容拦截(1301) 一律不重试
func retryableCode(code string) bool {
	switch code {
	case "1302", "1303", "1304", "1305": // QPS超限 / 模型繁忙 / 服务过载 / 服务不可用
		return true
	}
	return false
}

// retryableStatus 判断 HTTP 状态码是否值得重试
func retryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}

// Chat 是对外唯一入口：带重试的对话调用
// 所有 AI 功能都必须通过它调用，不要在别处直接发 HTTP
func Chat(ctx context.Context, messages []Message) (string, error) {
	cfg := global.Config.AI
	if !cfg.Enabled || cfg.APIKey == "" {
		return "", fmt.Errorf("ai 未配置")
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetry; attempt++ {
		// 重试前退避：1s、2s，再叠加 0-500ms 随机抖动
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			backoff += time.Duration(rand.Int63n(500)) * time.Millisecond

			// 外层 ctx 剩余时间不够等，直接放弃，避免白等到超时
			if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < backoff {
				logrus.Warnf("[AI] 剩余时间不足以退避 %v，放弃重试", backoff)
				break
			}

			logrus.Warnf("[AI] 第 %d 次重试，等待 %v", attempt, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", fmt.Errorf("重试等待中被取消: %w", ctx.Err())
			}
		}

		// 每次尝试都用独立的超时 ctx：上一次已超时的 ctx 不能复用
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		content, retryable, err := doChat(attemptCtx, cfg, messages)
		cancel()

		if err == nil {
			if attempt > 0 {
				logrus.Infof("[AI] 第 %d 次重试成功", attempt)
			}
			return content, nil
		}

		lastErr = err
		if !retryable {
			return "", err
		}
		// 外层 ctx 已结束（用户断开 / 上游超时），再试也没意义
		if ctx.Err() != nil {
			break
		}
		logrus.Warnf("[AI] 第 %d 次调用失败: %v", attempt+1, err)
	}
	return "", fmt.Errorf("AI 调用失败（已重试 %d 次）: %w", maxRetry, lastErr)
}

// doChat 发一次 HTTP 请求，返回 (内容, 是否值得重试, 错误)
func doChat(ctx context.Context, cfg conf.AI, messages []Message) (string, bool, error) {
	// json.Marshal：把结构体转为json字符串
	body, err := json.Marshal(chatRequest{
		Model:       cfg.Model,
		Messages:    messages,
		Temperature: 0.3, // 摘要/标签/审核都需要稳定输出，用低温度
	})
	if err != nil {
		return "", false, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// 网络层错误（超时、连接失败）可重试
		return "", true, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", false, fmt.Errorf("解析响应失败: %w, body=%s", err, truncate(string(raw), 300))
	}

	// 业务错误优先看 error.code
	if cr.Error != nil {
		code := strings.TrimSpace(cr.Error.Code)
		if retryableCode(code) {
			return "", true, fmt.Errorf("模型繁忙(code=%s): %s", code, cr.Error.Message)
		}
		return "", false, fmt.Errorf("模型返回错误(code=%s): %s", code, cr.Error.Message)
	}

	// 没有 error 对象但状态码异常，按状态码判断
	if resp.StatusCode >= 400 {
		if retryableStatus(resp.StatusCode) {
			return "", true, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
		}
		return "", false, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	if len(cr.Choices) == 0 {
		return "", false, fmt.Errorf("模型没有返回内容，body=%s", truncate(string(raw), 300))
	}
	if cr.Usage.TotalTokens > 0 {
		logrus.Infof("[AI] 模型=%s 输入=%d 输出=%d 总计=%d token",
			cfg.Model, cr.Usage.PromptTokens, cr.Usage.CompletionTokens, cr.Usage.TotalTokens)
	}
	return cr.Choices[0].Message.Content, false, nil
}

// truncate 截断长字符串，避免错误日志把整个响应体打出来
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
