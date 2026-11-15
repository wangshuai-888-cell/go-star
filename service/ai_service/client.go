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
	Error *apiErrorBody `json:"error"`
}

// apiErrorBody OpenAI 兼容的错误体，各厂商格式一致
type apiErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"` // 智谱用字符串错误码，如 "1000" / "1302"
}

// retryableCode 判断业务错误码是否值得重试
// 白名单机制：只有「过一会儿可能恢复」的码才重试
// 鉴权(1000~1004)、余额(1110)、参数(1200/1210)、内容拦截(1301) 一律不重试
func retryableCode(code string) bool {
	switch code {
	case "1302", // QPS 超限
		"1303", // 并发数过高
		"1304", // 服务过载
		"1305": // 服务不可用
		return true
	}
	return false
}

// retryableStatus 判断 HTTP 状态码是否值得重试
func retryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// retryCall 通用重试调度，Chat 和 Embed 共用
// fn 每轮执行一次实际请求，返回 (是否值得重试, 错误)；err 为 nil 即成功
func retryCall(ctx context.Context, timeout time.Duration, name string,
	fn func(context.Context) (bool, error)) error {

	var lastErr error
	for attempt := 0; attempt <= maxRetry; attempt++ {
		// 重试前退避：1s、2s，再叠加 0-500ms 随机抖动防惊群
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			backoff += time.Duration(rand.Int63n(500)) * time.Millisecond

			// 外层 ctx 剩余时间不够等，直接放弃
			if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < backoff {
				logrus.Warnf("[AI] %s 剩余时间不足以退避 %v，放弃重试", name, backoff)
				break
			}
			logrus.Warnf("[AI] %s 第 %d 次重试，等待 %v", name, attempt, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("%s 重试等待中被取消: %w", name, ctx.Err())
			}
		}

		// 每次尝试用独立超时 ctx，上一次超时后的 ctx 不能复用
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		retryable, err := fn(attemptCtx)
		cancel()

		if err == nil {
			if attempt > 0 {
				logrus.Infof("[AI] %s 第 %d 次重试成功", name, attempt)
			}
			return nil
		}
		lastErr = err
		if !retryable {
			return err
		}
		// 外层 ctx 已结束（用户断开 / 上游超时），再试也没意义
		if ctx.Err() != nil {
			break
		}
		logrus.Warnf("[AI] %s 第 %d 次调用失败: %v", name, attempt+1, err)
	}
	return fmt.Errorf("%s 调用失败（已重试 %d 次）: %w", name, maxRetry, lastErr)
}

// 统一的超时计算：配置为 0 时用默认 60s
func aiTimeout() time.Duration {
	t := time.Duration(global.Config.AI.Timeout) * time.Second
	if t == 0 {
		t = 60 * time.Second
	}
	return t
}

// checkAPIError 解析错误体与状态码，判断是否可重试
// 返回 nil 表示响应正常，可以继续解析业务字段
func checkAPIError(statusCode int, raw []byte, apiErr *apiErrorBody) (bool, error) {
	if apiErr != nil && apiErr.Message != "" {
		code := strings.TrimSpace(apiErr.Code)
		err := fmt.Errorf("模型返回错误(code=%s): %s", code, apiErr.Message)
		return retryableCode(code), err
	}
	if statusCode >= 400 {
		err := fmt.Errorf("HTTP %d: %s", statusCode, truncate(string(raw), 300))
		return retryableStatus(statusCode), err
	}
	return false, nil
}

// Chat 是对外唯一入口：带重试的对话调用
func Chat(ctx context.Context, messages []Message) (string, error) {
	cfg := global.Config.AI
	if !cfg.Enabled || cfg.APIKey == "" {
		return "", fmt.Errorf("ai 未配置")
	}

	var content string
	err := retryCall(ctx, aiTimeout(), "chat", func(ctx context.Context) (bool, error) {
		var retryable bool
		var err error
		content, retryable, err = doChat(ctx, cfg, messages)
		return retryable, err
	})
	return content, err
}

// doChat 发一次对话请求，返回 (内容, 是否值得重试, 错误)
func doChat(ctx context.Context, cfg conf.AI, messages []Message) (string, bool, error) {
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
	setAuthHeader(req, cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", true, err // 网络层错误可重试
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", false, fmt.Errorf("解析响应失败: %w, body=%s", err, truncate(string(raw), 300))
	}

	if retryable, apiErr := checkAPIError(resp.StatusCode, raw, cr.Error); apiErr != nil {
		return "", retryable, apiErr
	}
	if len(cr.Choices) == 0 {
		return "", false, fmt.Errorf("模型没有返回内容，body=%s", truncate(string(raw), 300))
	}
	if cr.Usage.TotalTokens > 0 {
		logrus.Infof("[AI] chat 模型=%s 输入=%d 输出=%d 总计=%d token",
			cfg.Model, cr.Usage.PromptTokens, cr.Usage.CompletionTokens, cr.Usage.TotalTokens)
	}
	return cr.Choices[0].Message.Content, false, nil
}

// setAuthHeader 外部 AI 服务统一用 Bearer 认证（与本站的 token header 是两回事）
func setAuthHeader(req *http.Request, apiKey string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

// truncate 截断长字符串，避免错误日志把整个响应体打出来
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
