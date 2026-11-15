package ai_service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-star/global"

	"github.com/sirupsen/logrus"
)

// streamChatRequest 流式请求体：和普通请求唯一的区别就是 Stream=true
type streamChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
}

// streamChunk 流式响应里的一小块
// 注意 Delta 而不是 Message：流式返回的是增量，每块只有几个字
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *apiErrorBody `json:"error"`
}

// ChatStream 流式对话：每收到一段文本就回调一次
// 和 Chat 的区别：不重试。已经推给前端的内容收不回来，
// 中途重试会变成"同一段话说两遍"，比直接报错更糟
func ChatStream(ctx context.Context, messages []Message, onChunk func(string) error) error {
	cfg := global.Config.AI
	if !cfg.Enabled || cfg.APIKey == "" {
		return fmt.Errorf("ai 未配置")
	}

	body, err := json.Marshal(streamChatRequest{
		Model:       cfg.Model,
		Messages:    messages,
		Temperature: 0.7, // 对话比摘要松一些，摘要用 0.3 是为了稳定
		Stream:      true,
	})
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	setAuthHeader(req, cfg.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 出错时上游返回的是普通 JSON，不是流，得单独处理
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		var er struct {
			Error *apiErrorBody `json:"error"`
		}
		_ = json.Unmarshal(raw, &er)
		if _, apiErr := checkAPIError(resp.StatusCode, raw, er.Error); apiErr != nil {
			return apiErr
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	scanner := bufio.NewScanner(resp.Body)
	// 默认单行上限 64KB，长段中文可能超，放大到 1MB
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue // 跳过空行分隔符和 event:/id: 这类非数据行
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // 个别厂商会插注释行，忽略而不是整条流中断
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return fmt.Errorf("模型返回错误(code=%s): %s", chunk.Error.Code, chunk.Error.Message)
		}
		if chunk.Usage.TotalTokens > 0 {
			logrus.Infof("[AI] chat-stream 模型=%s 总计=%d token", cfg.Model, chunk.Usage.TotalTokens)
		}
		if len(chunk.Choices) == 0 {
			continue // 最后那块只有 usage，没有 choices
		}
		text := chunk.Choices[0].Delta.Content
		if text == "" {
			continue // 首块只有 role，末块只有 finish_reason，都没正文
		}
		if err := onChunk(text); err != nil {
			return err // 回调失败一般是浏览器断开，必须立刻停，别继续烧 token
		}
	}
	return scanner.Err()
}
