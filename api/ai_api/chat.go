package ai_api

import (
	"context"
	"encoding/json"
	"fmt"
	"go-star/common/res"
	"go-star/service/ai_service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ChatStreamRequest struct {
	Content string `json:"content" binding:"required,max=2000"`
}

const chatSystemPrompt = `你是 go-star 技术博客站点的助手，回答中文技术问题。要求：
1. 简洁，不超过 300 字
2. 涉及代码给出关键片段，不要贴整个文件
3. 不确定的直接说"我不确定"，不要编造`

func (AiApi) ChatStreamView(c *gin.Context) {
	// 先校验参数：这一步失败还能正常返回JSON
	// 一旦设了 text/event-stream 头，就再也回不去普通响应了
	var cr ChatStreamRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	// 将来上 Nginx 时，不加这行流会被攒着一次性吐出，等于白做
	// X-Accel-Buffering 是 Nginx 自己认的一个"内部约定"响应头，跟 HTTP 标准没关系。它告诉 Nginx：这个响应别缓冲，收到多少立刻转发多少。
	c.Header("X-Accel-Buffering", "no")

	// 流式是长连接，不能用8秒那种短超时
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	messages := []ai_service.Message{
		{Role: "system", Content: chatSystemPrompt},
		{Role: "user", Content: cr.Content},
	}

	err := ai_service.ChatStream(ctx, messages, func(text string) error {
		// 必须包 JSON：模型输出里的换行会破坏 SSE 的帧分隔符
		payload, _ := json.Marshal(map[string]string{"content": text})
		if _, werr := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); werr != nil {
			return werr // 写不进去 = 浏览器已经断开
		}
		c.Writer.Flush() // 不 flush 就只是"假流式"
		return nil
	})

	if err != nil {
		// 已经吐过内容，HTTP 状态码改不了了，只能发一个 error 事件
		if ctx.Err() == context.Canceled {
			logrus.Infof("流式对话被客户端中断")
			return // 用户自己关的页面，不算错误
		}
		logrus.Errorf("流式对话失败: %v", err)
		payload, _ := json.Marshal(map[string]string{"error": "生成中断，请重试"})
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		c.Writer.Flush()
		return
	}

	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}
