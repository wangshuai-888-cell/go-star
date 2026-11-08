package ai_service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// 有候选集时：只能从候选集里挑
const tagPromptWithCandidates = `你是文章标签助手。从下面的「候选标签」里挑选 3-5 个最适合这篇文章的标签。

候选标签（只能从这里选，严禁自创）：
%s

文章标题：%s

文章正文：
%s

要求：
1. 只能从候选标签里选，严禁自创新标签
2. 选 3-5 个最相关的
3. 只输出 JSON 数组，不要任何解释文字
4. 格式示例：["标签1","标签2","标签3"]`

// 无候选集时（冷启动）：允许自由生成
const tagPromptFreeForm = `你是文章标签助手。为下面的文章生成 3-5 个标签。

文章标题：%s

文章正文：
%s

要求：
1. 标签要简洁（2-6 个字），是技术名词或主题词
2. 生成 3-5 个
3. 只输出 JSON 数组，不要任何解释文字
4. 格式示例：["Go","缓存","并发"]`

// GenerateeeTags 为文章生成标签
// 候选集为空时自由生成（冷启动兜底），非空时严格限定在候选集内
// 解析失败返回空切片 + nil，调用方按不加标签处理
func GenerateTags(ctx context.Context, title, content string, candidates []string) ([]string, error) {
	runes := []rune(content)
	if len(runes) > 3000 {
		content = string(runes[:3000])
	}

	var prompt string
	if len(candidates) == 0 {
		prompt = fmt.Sprintf(tagPromptFreeForm, title, content)
	} else {
		prompt = fmt.Sprintf(tagPromptWithCandidates,
			"["+strings.Join(candidates, ", ")+"]", title, content)
	}

	reply, err := Chat(ctx, []Message{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, err
	}

	tags, ok := parseTags(reply)
	if !ok {
		return nil, nil // 解析失败 = 放行，不加标签
	}

	// 有候选集时做二次校验，只保留候选集里的
	if len(candidates) > 0 {
		allowed := make(map[string]struct{}, len(candidates))
		for _, c := range candidates {
			allowed[c] = struct{}{}
		}
		out := make([]string, 0, len(tags))
		for _, t := range tags {
			if _, ok := allowed[t]; ok {
				out = append(out, t)
			}
		}
		return out, nil
	}
	return tags, nil
}

// parseTags 从模型回复里抠出 JSON 数组
// 兼容：纯 JSON / ```json 包裹 / 带前缀说明
// 都解析不了返回 (nil, false)
func parseTags(reply string) ([]string, bool) {
	start := strings.Index(reply, "[")
	end := strings.LastIndex(reply, "]")
	if start < 0 || end <= start {
		return nil, false
	}
	raw := reply[start : end+1]

	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil, false
	}

	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}
