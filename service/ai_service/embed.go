package ai_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go-star/conf"
	"go-star/global"

	"github.com/sirupsen/logrus"
)

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"` // 部分模型支持降维
}

type embedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *apiErrorBody `json:"error"`
}

// Embed 把一批文本向量化，返回顺序与入参一致
// 走和 Chat 同一套重试 / 错误分级逻辑
func Embed(ctx context.Context, texts []string) ([][]float32, error) {
	cfg := global.Config.AI
	if !cfg.Enabled || cfg.APIKey == "" {
		return nil, fmt.Errorf("ai 未配置")
	}
	if cfg.EmbeddingModel == "" {
		return nil, fmt.Errorf("未配置 embedding_model")
	}
	if len(texts) == 0 {
		return nil, nil
	}

	var vectors [][]float32
	err := retryCall(ctx, aiTimeout(), "embed", func(ctx context.Context) (bool, error) {
		var retryable bool
		var err error
		vectors, retryable, err = doEmbed(ctx, cfg, texts)
		return retryable, err
	})
	return vectors, err
}

// doEmbed 发一次向量化请求
func doEmbed(ctx context.Context, cfg conf.AI, texts []string) ([][]float32, bool, error) {
	body, err := json.Marshal(embedRequest{
		Model:      cfg.EmbeddingModel,
		Input:      texts,
		Dimensions: cfg.EmbeddingDim,
	})
	if err != nil {
		return nil, false, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 第一步：把参数装进 request，这一步还没有发出任何网络请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		cfg.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("构造请求失败: %w", err)
	}
	setAuthHeader(req, cfg.APIKey)

	// 第二步：真正发出去，这一步才拿到响应
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, true, err // 网络层错误可重试
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var er embedResponse
	if err := json.Unmarshal(raw, &er); err != nil {
		return nil, false, fmt.Errorf("解析响应失败: %w, body=%s", err, truncate(string(raw), 300))
	}

	// 错误体与状态码统一判定，和 doChat 保持一致
	if retryable, apiErr := checkAPIError(resp.StatusCode, raw, er.Error); apiErr != nil {
		return nil, retryable, apiErr
	}
	if len(er.Data) != len(texts) {
		return nil, false, fmt.Errorf("向量数量不匹配：期望 %d，实际 %d", len(texts), len(er.Data))
	}

	// 按 index 归位，防止厂商乱序
	out := make([][]float32, len(texts))
	for _, d := range er.Data {
		if d.Index < 0 || d.Index >= len(texts) {
			return nil, false, fmt.Errorf("向量下标越界: %d", d.Index)
		}
		out[d.Index] = d.Embedding
	}
	// 校验维度一致性
	for i, v := range out {
		if len(v) == 0 {
			return nil, false, fmt.Errorf("第 %d 条向量为空", i)
		}
		if cfg.EmbeddingDim > 0 && len(v) != cfg.EmbeddingDim {
			return nil, false, fmt.Errorf("第 %d 条维度不符: 期望 %d, 实际 %d", i, cfg.EmbeddingDim, len(v))
		}
	}

	if er.Usage.TotalTokens > 0 {
		logrus.Infof("[AI] embed 模型=%s 文本数=%d 维度=%d 总计=%d token",
			cfg.EmbeddingModel, len(texts), len(out[0]), er.Usage.TotalTokens)
	}
	return out, false, nil
}
