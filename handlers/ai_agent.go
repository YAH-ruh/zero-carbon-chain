package handlers

import (
	"crypto/sha256"
	"fmt"
	"time"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// TriggerTradeAgent 触发交易Agent(碳交易所使用)
func TriggerTradeAgent(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentNo := fmt.Sprintf("AGENT-TRADE-%d-%d", userID, time.Now().Unix())

	// 模拟Agent推理：分析挂单数据，推荐最优撮合策略
	inputData := `{"action":"analyze_sell_orders","params":{"sort_by":"price_asc","max_results":5}}`
	outputData := `{"recommendation":"最优撮合策略: 优先匹配低价挂单","orders_analyzed":3,"estimated_volume":1500.0}`

	zkRef := fmt.Sprintf("ZK-AGENT-%x", sha256.Sum256([]byte(agentNo)))[:32]

	record := models.AgentRecord{
		AgentNo:    agentNo,
		AgentType:  "trade",
		TriggerBy:  userID,
		InputData:  inputData,
		OutputData: outputData,
		ZKProofRef: zkRef,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "触发交易Agent失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeAgent, agentNo, record.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "Agent操作上链失败: "+err.Error())
		return
	}
	tx.Model(&record).Update("block_hash", block.BlockHash)
	tx.Model(&record).Update("on_chain", true)
	tx.Commit()

	response.OK(c, gin.H{
		"agent_no":    agentNo,
		"agent_type":  "trade",
		"output":      outputData,
		"zk_proof":    zkRef,
		"block_hash":  block.BlockHash,
	})
}

// TriggerRiskAgent 触发风险检测Agent(监管核查使用)
func TriggerRiskAgent(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentNo := fmt.Sprintf("AGENT-RISK-%d-%d", userID, time.Now().Unix())

	inputData := `{"action":"risk_scan","scope":"all_transactions","time_range":"7d"}`
	outputData := `{"risk_level":"low","anomalies":[],"total_scan":42,"recommendation":"未发现异常交易行为"}`

	zkRef := fmt.Sprintf("ZK-RISK-%x", sha256.Sum256([]byte(agentNo)))[:32]

	record := models.AgentRecord{
		AgentNo:    agentNo,
		AgentType:  "risk",
		TriggerBy:  userID,
		InputData:  inputData,
		OutputData: outputData,
		ZKProofRef: zkRef,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "触发风险检测Agent失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeAgent, agentNo, record.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "Agent操作上链失败: "+err.Error())
		return
	}
	tx.Model(&record).Update("block_hash", block.BlockHash)
	tx.Model(&record).Update("on_chain", true)
	tx.Commit()

	response.OK(c, gin.H{
		"agent_no":    agentNo,
		"agent_type":  "risk",
		"output":      outputData,
		"zk_proof":    zkRef,
		"block_hash":  block.BlockHash,
	})
}

// TriggerDispatchAgent 触发园区调度Agent(园区管理员使用)
func TriggerDispatchAgent(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentNo := fmt.Sprintf("AGENT-DISPATCH-%d-%d", userID, time.Now().Unix())

	inputData := `{"action":"dispatch_optimize","park_id":1,"resources":["energy","carbon_credits"]}`
	outputData := `{"dispatch_plan":{"energy_allocation":"optimized","carbon_credit_redistribution":true},"saved_credits":500.0}`

	zkRef := fmt.Sprintf("ZK-DISPATCH-%x", sha256.Sum256([]byte(agentNo)))[:32]

	record := models.AgentRecord{
		AgentNo:    agentNo,
		AgentType:  "park_dispatch",
		TriggerBy:  userID,
		InputData:  inputData,
		OutputData: outputData,
		ZKProofRef: zkRef,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "触发园区调度Agent失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeAgent, agentNo, record.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "Agent操作上链失败: "+err.Error())
		return
	}
	tx.Model(&record).Update("block_hash", block.BlockHash)
	tx.Model(&record).Update("on_chain", true)
	tx.Commit()

	response.OK(c, gin.H{
		"agent_no":    agentNo,
		"agent_type":  "park_dispatch",
		"output":      outputData,
		"zk_proof":    zkRef,
		"block_hash":  block.BlockHash,
	})
}

// ListAgentRecords Agent操作记录列表
func ListAgentRecords(c *gin.Context) {
	var list []models.AgentRecord
	query := database.DB.Order("id desc").Limit(50)
	if agentType := c.Query("agent_type"); agentType != "" {
		query = query.Where("agent_type = ?", agentType)
	}
	if err := query.Find(&list).Error; err != nil {
		response.ServerError(c, "查询Agent记录失败")
		return
	}
	if list == nil {
		list = []models.AgentRecord{}
	}
	response.OK(c, gin.H{"list": list})
}

// GetZKAIAnomalyAlert ZK-AI异常能耗告警
func GetZKAIAnomalyAlert(c *gin.Context) {
	// 模拟ZK-AI异常检测：检查近期IoT记录，与历史均值比对
	var anomalyCount int64
	database.DB.Model(&models.IoTRecord{}).
		Where("risk_label != '' AND created_at > ?", time.Now().Add(-72*time.Hour)).
		Count(&anomalyCount)

	alerts := []gin.H{}
	if anomalyCount > 0 {
		alerts = append(alerts, gin.H{
			"type":    "energy_anomaly",
			"level":   "medium",
			"count":   anomalyCount,
			"detail":  "检测到带风险标签的能耗记录",
			"zk_proof": fmt.Sprintf("ZK-ANOMALY-%x", sha256.Sum256([]byte(fmt.Sprintf("anomaly-%d", time.Now().Unix()))))[:16],
		})
	}
	// 始终添加模拟数据
	alerts = append(alerts, gin.H{
		"type":    "zk_ai_analysis",
		"level":   "info",
		"count":   0,
		"detail":  "ZK-AI实时分析: 当前能耗模式正常，未发现异常波动",
		"zk_proof": fmt.Sprintf("ZK-AI-%x", sha256.Sum256([]byte(time.Now().String())))[:16],
	})
	response.OK(c, gin.H{"alerts": alerts})
}