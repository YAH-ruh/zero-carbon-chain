// Package services 提供业务逻辑服务
// AI服务：对接DeepSeek API实现企业减排建议和园区报告生成。
//
// 变更说明(v2)：
//  1. 超时由写死 60s 改为读取 config.AITimeout()(默认25s)，AI 阻塞不会拖死接口；
//  2. 引入来源标记(source)：真实大模型成功返回 "ai"，未配置Key/网络异常/超时自动降级为 "fallback"，
//     同时返回 warn 提示文案，前端可友好展示且不会卡死；
//  3. 所有 AI 调用错误通过分级日志记录，便于排障。
package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"blockchain-demo/config"
	"blockchain-demo/pkg/logger"
)

// 来源标记常量(与 ReportRecord.Source 字段对应)
const (
	SourceAI       = "ai"       // 内容来自真实大模型
	SourceFallback = "fallback" // 内容为容错降级模板(未配置Key或调用失败)
)

// ChatSystemPrompt 聊天助手固定 system 角色提示
const ChatSystemPrompt = `你是零碳微证平台AI碳减排助手，熟悉小微企业碳排放、碳积分、ZKP隐私核算、碳足迹相关知识。
能够针对不同行业给出小微企业适用的简易碳排放策略方案。回答简洁，贴合隐私核算、区块链存证场景。
仅回答碳减排相关问题，无关问题请告知用户你只提供碳减排咨询。`

// DeepSeekMessage DeepSeek API消息结构
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekRequest DeepSeek API请求结构
type DeepSeekRequest struct {
	Model       string            `json:"model"`
	Messages    []DeepSeekMessage `json:"messages"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
}

// DeepSeekRespMessage DeepSeek API响应消息结构（含独立思考思维链字段）
type DeepSeekRespMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"` // 模型思维链（独立思考过程）
}

// DeepSeekResponse DeepSeek API响应结构
type DeepSeekResponse struct {
	Choices []struct {
		Message DeepSeekRespMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// callDeepSeekAPI 调用DeepSeek API
// 变更说明(v2)：http.Client 超时改为配置化(cfg.AITimeout())，杜绝网络异常长期占线。
func callDeepSeekAPI(systemPrompt, userPrompt string) (string, error) {
	cfg := config.AppConfig

	// 未配置有效API Key → 返回模拟内容(由上层标记 source=fallback)
	if cfg.DeepSeekAPIKey == "" || cfg.DeepSeekAPIKey == "your_deepseek_api_key_here" {
		return "", fmt.Errorf("未配置 DEEPSEEK_API_KEY")
	}

	reqBody := DeepSeekRequest{
		Model: cfg.DeepSeekModel,
		Messages: []DeepSeekMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	client := &http.Client{Timeout: cfg.AITimeout()}
	req, err := http.NewRequest("POST", cfg.DeepSeekAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		// 超时/网络异常归类处理，调用方降级
		if osIsTimeout(err) {
			return "", fmt.Errorf("AI服务响应超时(>%.0fs): %v", cfg.AITimeout().Seconds(), err)
		}
		return "", fmt.Errorf("调用AI服务失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var result DeepSeekResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("AI服务错误: %s", result.Error.Message)
	}

	if len(result.Choices) > 0 {
		content := strings.TrimSpace(result.Choices[0].Message.Content)
		if content != "" {
			return content, nil
		}
	}
	return "", fmt.Errorf("AI服务返回内容为空")
}

// osIsTimeout 判断是否超时类错误(url.Error 内部 Timeout)
func osIsTimeout(err error) bool {
	ne, ok := err.(interface{ Timeout() bool })
	return ok && ne.Timeout()
}

// callDeepSeekChat 调用 DeepSeek API（多轮对话版本）
// 与 callDeepSeekAPI 的区别：直接接受完整 messages 数组（含 system/user/assistant 历史），
// 不再由上层强制拼 system + user 两轮。
// 返回：content（最终回答）、reasoning（模型独立思考思维链，可能为空）、error
func callDeepSeekChat(messages []DeepSeekMessage) (string, string, error) {
	cfg := config.AppConfig

	if cfg.DeepSeekAPIKey == "" || cfg.DeepSeekAPIKey == "your_deepseek_api_key_here" {
		return "", "", fmt.Errorf("未配置 DEEPSEEK_API_KEY")
	}

	reqBody := DeepSeekRequest{
		Model:       cfg.DeepSeekModel,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("序列化请求失败: %w", err)
	}

	client := &http.Client{Timeout: cfg.AITimeout()}
	req, err := http.NewRequest("POST", cfg.DeepSeekAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		if osIsTimeout(err) {
			return "", "", fmt.Errorf("AI服务响应超时(>%.0fs): %v", cfg.AITimeout().Seconds(), err)
		}
		return "", "", fmt.Errorf("调用AI服务失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	var result DeepSeekResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("解析响应失败: %w (body=%s)", err, string(body))
	}

	if result.Error != nil {
		return "", "", fmt.Errorf("AI服务错误: %s", result.Error.Message)
	}

	if len(result.Choices) > 0 {
		raw := strings.TrimSpace(result.Choices[0].Message.Content)
		reasoning := strings.TrimSpace(result.Choices[0].Message.ReasoningContent)
		// 兼容部分模型把思维链以 <think>...</think> 内联在 content 中的形态
		content, inlineThink := stripThinkBlock(raw)
		if reasoning == "" {
			reasoning = inlineThink
		}
		if content != "" {
			return content, reasoning, nil
		}
	}
	return "", "", fmt.Errorf("AI服务返回内容为空")
}

// stripThinkBlock 从 content 中剥离 <think>...</think> 思维链块
// 返回：纯回答内容、剥离出的思考文本
func stripThinkBlock(raw string) (content, think string) {
	open := strings.Index(raw, "<think>")
	if open < 0 {
		return raw, ""
	}
	closeIdx := strings.Index(raw, "</think>")
	if closeIdx < 0 {
		// 未闭合：视为整段都是思考
		return strings.TrimSpace(raw[:open]), strings.TrimSpace(raw[open+len("<think>"):])
	}
	think = strings.TrimSpace(raw[open+len("<think>") : closeIdx])
	content = strings.TrimSpace(raw[:open] + raw[closeIdx+len("</think>"):])
	return content, think
}

// Chat 多轮对话入口
// 参数：前端传来的 messages（可为空或只有历史对话，不含 system 角色——此处强制前置）
// 返回：content(最终回答), reasoning(独立思考思维链), source(ai/error), warn
// 只走真实 API 模式：成功返回 source=ai，失败返回 source=error + 错误信息在 content 中。
func Chat(userMessages []DeepSeekMessage) (content, reasoning, source, warn string) {
	// 过滤非法角色 + 最多保留最近 10 条 user/assistant 对（控制 token 消耗）
	trimmed := make([]DeepSeekMessage, 0, len(userMessages))
	for _, m := range userMessages {
		switch m.Role {
		case "user", "assistant":
			if strings.TrimSpace(m.Content) != "" {
				trimmed = append(trimmed, m)
			}
		}
	}
	// 只保留最近 10 条
	if len(trimmed) > 10 {
		trimmed = trimmed[len(trimmed)-10:]
	}

	// 组装：前置固定 system 角色
	fullMessages := append([]DeepSeekMessage{
		{Role: "system", Content: ChatSystemPrompt},
	}, trimmed...)

	text, reasoning, err := callDeepSeekChat(fullMessages)
	if err == nil {
		logger.Info("AI 多轮对话成功 turns=%d reasoning_len=%d", len(trimmed), len(reasoning))
		return text, reasoning, SourceAI, ""
	}

	// 真实 API 失败 → 直接返回 error 类型，由前端提示网络请求失败
	logger.Error("AI 多轮对话失败: %v", err)
	return "", "", "error", err.Error()
}

// GenerateEmissionReductionAdvice 生成企业节能减排优化建议
// 返回：内容 content、来源 source(ai/fallback)、提示 warn。
// 变更说明(v2)：无论AI是否可用都返回可用内容，接口永不因AI异常返回500而卡死。
func GenerateEmissionReductionAdvice(companyName string, totalEmission float64, industry string) (content, source, warn string) {
	systemPrompt := `你是一位专业的碳减排咨询专家，为小微企业提供节能减排优化建议。请根据企业能耗数据，给出具体可操作的减排建议。要求：建议要具体、可执行、符合小微企业实际情况，包含预计的减排效果预估。`
	userPrompt := fmt.Sprintf(`请为以下企业生成节能减排优化建议：
企业名称：%s
总碳排放量：%.2f kgCO₂
所属行业：%s
请从以下方面给出建议：
1. 能效提升措施
2. 清洁能源替代建议
3. 生产工艺优化
4. 碳积分交易策略
5. 预计减排效果`, companyName, totalEmission, industry)

	text, err := callDeepSeekAPI(systemPrompt, userPrompt)
	if err == nil {
		logger.Info("AI 生成企业减排建议成功 company=%s emission=%.2f", companyName, totalEmission)
		return text, SourceAI, ""
	}
	logger.Warn("AI 生成企业减排建议失败，已降级模板: %v", err)
	return fallbackAdvice(companyName, totalEmission, industry), SourceFallback, "AI服务暂不可用(" + err.Error() + ")，已展示内置参考建议"
}

// GenerateParkReport 生成园区低碳发展报告
func GenerateParkReport(parkName string, parkData map[string]interface{}) (content, source, warn string) {
	systemPrompt := `你是一位工业园区低碳发展规划专家，负责为工业园区生成低碳发展报告和补贴申报材料。报告需要专业、有数据支撑、包含具体建议。`
	dataJSON, _ := json.MarshalIndent(parkData, "", "  ")
	userPrompt := fmt.Sprintf(`请为以下工业园区生成低碳发展报告和补贴申报材料：
园区名称：%s
园区数据：%s
请包含以下内容：
1. 园区碳排放总体概况
2. 各企业碳排放排名
3. 低碳发展建议
4. 补贴申报理由和可行性分析
5. 预期碳减排目标`, parkName, string(dataJSON))

	text, err := callDeepSeekAPI(systemPrompt, userPrompt)
	if err == nil {
		logger.Info("AI 生成园区低碳报告成功 park=%s", parkName)
		return text, SourceAI, ""
	}
	logger.Warn("AI 生成园区低碳报告失败，已降级模板: %v", err)
	return fallbackParkReport(parkName), SourceFallback, "AI服务暂不可用(" + err.Error() + ")，已展示内置参考报告"
}

// fallbackAdvice 企业减排建议降级模板(内容基于真实能耗数值动态生成)
func fallbackAdvice(companyName string, totalEmission float64, industry string) string {
	industryText := industry
	if industryText == "" {
		industryText = "一般制造"
	}
	return fmt.Sprintf(`【节能减排优化建议 - 参考模板】

尊敬的企业负责人，基于贵企业碳排放数据(%.2f kgCO₂)与所属行业(%s)，提出以下参考建议：

## 1. 能效提升措施
- 更换一级能效电机与变频设备，预计降低用电消耗 15%%-20%%
- 实施照明/LED 与空压系统改造，降低辅助能耗 20%%-30%%
- 建立重点用能设备台账，实施能源管理体系

## 2. 清洁能源替代
- 厂房屋顶建设分布式光伏，满足 20%%-30%% 用电需求
- 评估余热回收、空气源热泵等替代方案

## 3. 生产工艺优化
- 优化生产排程，减少设备空转与待机能耗
- 定期维护保养，保持设备最佳运行效率

## 4. 碳积分交易策略
- 富余碳积分通过平台挂牌交易，将减排量转化为收益
- 关注碳市场价格，合理择时挂单

## 5. 预计减排效果
- 综合施策预计年减排 15%%-25%%，年节约能源成本 8-15 万元

（注：本内容为系统内置参考模板；接入 DeepSeek 后将生成针对性的 AI 建议）`, totalEmission, industryText)
}

// fallbackParkReport 园区低碳报告降级模板
func fallbackParkReport(parkName string) string {
	return fmt.Sprintf(`# %s 低碳发展报告(参考模板)

## 一、碳排放总体概况
- 园区碳排放总量：XX 吨CO₂(以各企业能耗核算为准)
- 单位产值碳排放：XX 吨CO₂/万元
- 同比变化：较上期减少 X%%

## 二、企业碳排放排名
按园区内企业碳积分核算数据从高到低排序(见企业列表)

## 三、低碳发展建议
1. 推动园区集中供热供气与绿色电力交易
2. 建设园区级能源与碳数据管理平台
3. 建立"碳积分激励 + 企业绿码"机制
4. 推广绿色建筑与低碳改造补贴

## 四、补贴申报可行性
- 对照绿色低碳园区建设指南逐项自评
- 结合平台链上核算数据形成佐证材料

## 五、预期碳减排目标
- 短期(1年)：减排 5%%-8%%；中期(3年)：15%%-20%%；长期(5年)：30%%以上

（注：本内容为系统内置参考模板；接入 DeepSeek 后将生成针对性的 AI 报告）`, parkName)
}
