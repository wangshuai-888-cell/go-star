package ai_api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/ai_service"
	"go-star/service/vector_service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ragScoreThreshold 第一道闸门：Top1 低于这个分数直接拒答，不调模型
// 依据实测：相关约 0.70，无关约 0.55，取中间偏松的位置
// 宁可多放进来让模型判断，也不要误杀——误杀是"搜不到"，错放只是多花一点 token
const ragScoreThreshold = 0.5

// ragTopN 每次喂给模型几篇资料
const ragTopN = 5

// ragContentLimit 每篇资料取正文前多少字
// 5 篇 × 500 字 ≈ 2500 字，加上问题和规则约 4000 token，免费额度扛得住
const ragContentLimit = 500

// ragSystemPrompt 规则 + 资料 + 规则，%s 是资料占位
const ragSystemPrompt = `你是 go-star 技术博客的内容问答助手。

## 知识来源
以下资料摘自已发布的站内文章，这是你唯一可用的信息来源：

%s

## 回答规则
1. 只能使用上面资料中的信息。即使你自己知道答案，只要资料里没写，就不能说出来。
2. 如果资料完全无法回答这个问题，只输出这一句：「站内文章暂未涉及这个问题。」
3. 只有当资料不足以回答问题的某个部分时，才说明边界（例如「站内文章讲到了 A，没有涉及 B」）。
   如果资料足以完整回答问题，直接给出答案，不要加「站内文章只讲到了……」这类开场白。
4. 用中文回答，不超过 300 字。
5. 不要输出「根据资料」「如上文所述」这类话，直接说内容。`

type AskRequest struct {
	Question string `json:"question" binding:"required,max=500"`
}

// 返回给前端的引用来源
type sourceItem struct {
	ArticleID uint    `json:"articleID"`
	Title     string  `json:"title"`
	Score     float64 `json:"score"`
}

func (AiApi) AskStreamView(c *gin.Context) {
	var cr AskRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 检索 + 生成一共可能 10 秒以上，按长连接给
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	// ① 检索
	hits, err := vector_service.Search(ctx, cr.Question, ragTopN)
	if err != nil {
		logrus.Errorf("[RAG] 检索失败 question=%q: %v", cr.Question, err)
		writeSSE(c, map[string]string{"error": "检索失败，请稍后再试"})
		return
	}

	// ② 第一道闸门：连最相关的那篇都够不着，没必要浪费一次模型调用
	if len(hits) == 0 || hits[0].Score < ragScoreThreshold {
		top := 0.0
		if len(hits) > 0 {
			top = hits[0].Score
		}
		logrus.Infof("[RAG] 未达阈值拒答 question=%q Top1=%.4f", cr.Question, top)
		writeSSE(c, map[string]string{"content": "站内文章暂未涉及这个问题。"})
		writeSSE(c, map[string]any{"sources": []sourceItem{}})
		writeSSEDone(c)
		return
	}

	// ③ 查文章正文（只查已发布，下架文章的资料不能进模型）
	ids := make([]uint, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ArticleID)
	}
	var articles []models.ArticleModel
	if err = global.DB.Select("id", "title", "content").
		Where("id IN ? AND status = ?", ids, enum.ArticleStatusPublished).
		Find(&articles).Error; err != nil {
		logrus.Errorf("[RAG] 加载文章失败: %v", err)
		writeSSE(c, map[string]string{"error": "加载资料失败，请稍后再试"})
		return
	}

	// ④ 第二道闸门：候选全被下架/删掉，也没法答
	if len(articles) == 0 {
		logrus.Infof("[RAG] 候选文章已全部下架 question=%q", cr.Question)
		writeSSE(c, map[string]string{"content": "站内文章暂未涉及这个问题。"})
		writeSSE(c, map[string]any{"sources": []sourceItem{}})
		writeSSEDone(c)
		return
	}

	// ⑤ 拼资料：顺序跟 hits 一致，让模型看到的和分数排序一致
	scoreMap := make(map[uint]float64, len(hits))
	for _, h := range hits {
		scoreMap[h.ArticleID] = h.Score
	}
	articleMap := make(map[uint]models.ArticleModel, len(articles))
	for _, a := range articles {
		articleMap[a.ID] = a
	}
	var sb strings.Builder
	sources := make([]sourceItem, 0, len(articles))
	for i, h := range hits {
		a, ok := articleMap[h.ArticleID]
		if !ok {
			continue // 这篇未发布，跳过
		}
		sb.WriteString(fmt.Sprintf("【资料%d】《%s》\n", i+1, a.Title))
		sb.WriteString(truncateRunes(a.Content, ragContentLimit))
		sb.WriteString("\n\n")
		sources = append(sources, sourceItem{
			ArticleID: a.ID,
			Title:     a.Title,
			Score:     h.Score,
		})
	}

	logrus.Infof("[RAG] 命中 question=%q Top1=%.4f 资料数=%d",
		cr.Question, hits[0].Score, len(sources))

	// ⑥ 生成
	messages := []ai_service.Message{
		{Role: "system", Content: fmt.Sprintf(ragSystemPrompt, sb.String())},
		{Role: "user", Content: cr.Question},
	}

	err = ai_service.ChatStream(ctx, messages, func(text string) error {
		payload, _ := json.Marshal(map[string]string{"content": text})
		if _, werr := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); werr != nil {
			return werr
		}
		c.Writer.Flush()
		return nil
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			logrus.Infof("[RAG] 客户端中断 question=%q", cr.Question)
			return
		}
		logrus.Errorf("[RAG] 生成失败 question=%q: %v", cr.Question, err)
		writeSSE(c, map[string]string{"error": "生成中断，请重试"})
		return
	}

	// ⑦ 正文输出完再单独发一帧出处
	// 不让模型自己写"参考：《xxx》"——小模型经常忘，而且格式不受控
	writeSSE(c, map[string]any{"sources": sources})
	writeSSEDone(c)
}

// writeSSE 发一帧 SSE 数据，值必须能 JSON 序列化
func writeSSE(c *gin.Context, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		return
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
		return
	}
	c.Writer.Flush()
}

func writeSSEDone(c *gin.Context) {
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

// truncateRunes 按字符（不是字节）截断，避免把中文切成半个
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
