// Package handlers HTTP 请求处理器
// AI 聊天接口：POST /api/chat —— 多轮对话
//
// 请求体：
//
//	{
//	  "messages": [
//	    {"role":"user", "content":"帮我看看能耗数据"},
//	    {"role":"assistant", "content":"好的..."},
//	    {"role":"user", "content":"那再看看ZKP"}
//	  ]
//	}
//
// （system 角色由后端 ChatSystemPrompt 固定注入，前端无需传）
//
// 响应：
//
//	{
//	  "code": 200,
//	  "msg": "success",
//	  "data": {
//	    "content": "AI 回答文本（Markdown 格式）",
//	    "source": "ai | fallback",
//	    "warn": "警告提示（可能为空字符串）"
//	  }
//	}
//
// 设计要点：
//   - 永不返回 500：未配置 Key / 网络异常 / 超时 均降级为内置模板（source=fallback）
//   - 返回 warn 字段让前端友好提示"这是降级回答"
package handlers

import (
	"blockchain-demo/pkg/response"
	"blockchain-demo/services"

	"github.com/gin-gonic/gin"
)

// ChatRequest 聊天接口请求体
type ChatRequest struct {
	Messages []services.DeepSeekMessage `json:"messages" binding:"required"`
}

// ChatResponseData 聊天接口响应 data 字段
type ChatResponseData struct {
	Content   string `json:"content"`             // AI 最终回答文本
	Reasoning string `json:"reasoning,omitempty"` // AI 独立思考思维链（可能为空）
	Source    string `json:"source"`              // "ai" | "fallback"
	Warn      string `json:"warn,omitempty"`      // 可选警告（如"AI服务暂不可用..."）
}

// Chat AI 多轮对话入口
func Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求体格式错误: "+err.Error())
		return
	}

	// 基本校验：至少要有一条非空消息
	if len(req.Messages) == 0 {
		response.BadRequest(c, "messages 不能为空")
		return
	}

	content, reasoning, source, warn := services.Chat(req.Messages)

	response.OK(c, ChatResponseData{
		Content:   content,
		Reasoning: reasoning,
		Source:    source,
		Warn:      warn,
	})
}
