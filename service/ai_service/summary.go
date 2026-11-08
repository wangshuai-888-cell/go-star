package ai_service

import (
	"context"
	"strings"
)

const summarySystemPrompt = `你是专业的中文文章摘要助手。
请为下面的文章生成一段不超过 100 字的中文摘要。

严格要求：
1. 只输出摘要正文，不要加"摘要："之类的任何前缀
2. 不要使用引号包住整段
3. 保留核心观点，不要罗列章节标题
4. 语气与原文保持一致`

// GenerateSummary 生成文章摘要
// 注意：部分轻量模型（如智谱 glm-4-flash-250414）不支持 system role，
// 因此把系统指令合并进 user 消息，保证跨厂商兼容
func GenerateSummary(ctx context.Context, title, content string) (string, error) {
	runes := []rune(content)
	if len(runes) > 3000 {
		content = string(runes[:3000])
	}
	reply, err := Chat(ctx, []Message{
		{
			Role:    "user",
			Content: summarySystemPrompt + "\n\n【文章标题】" + title + "\n\n【正文】\n" + content,
		},
	})
	if err != nil {
		return "", err
	}
	// 防御性截断：字段是 varchar(256)，留 20 字符余量
	out := []rune(strings.TrimSpace(reply))
	if len(out) > 240 {
		out = out[:240]
	}
	return string(out), nil
}
