package ai_service

import (
	"context"
	"fmt"
	"strings"
)

const auditSystemPrompt = `你是内容审核助手。判断下面的用户评论是否包含以下任一违规内容：
1. 辱骂、人身攻击、歧视性言论
2. 色情、低俗内容
3. 广告、推广、引流（加微信、扫码、联系方式）
4. 政治敏感、违法信息
5. 恶意刷屏、无意义重复

只输出一个词，不要有任何其他内容：
- 违规 → 输出 block
- 正常 → 输出 pass`

// AuditComment 审核评论内容
// 返回 (true, nil) 表示通过
// 返回 (false, nil) 表示违规
// 返回 (_, err)     表示审核服务不可用，由调用方决定降级策略
func AuditComment(ctx context.Context, content string) (bool, error) {
	runes := []rune(content)
	if len(runes) > 500 {
		content = string(runes[:500])
	}

	reply, err := Chat(ctx, []Message{
		{
			Role:    "user",
			Content: auditSystemPrompt + "\n\n【评论内容】\n" + content,
		},
	})
	if err != nil {
		return false, err
	}

	// 模型偶尔会回 "Block。" / "判定为 block" / "这条必须 block"，统一小写后模糊匹配
	lower := strings.ToLower(strings.TrimSpace(reply))
	// 保守优先：同时出现两个词时按违规处理
	if strings.Contains(lower, "block") {
		return false, nil
	}
	if strings.Contains(lower, "pass") {
		return true, nil
	}
	// 两个词都没有，说明模型没按格式回，返回 error 让调用方走 fail-open 降级
	return false, fmt.Errorf("审核模型返回格式异常: %q", reply)
}
