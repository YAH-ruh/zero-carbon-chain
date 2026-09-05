package handlers

// 监管核证业务处理器：全局链上记录、任意数据篡改校验、平台运行监控、
// 风险告警、用户总览，以及企业 AI 减排建议(生成→落库→上链)。
// 数据权限：本组接口为最高权限，路由仅允许 regulator 角色访问。

import (
	"strconv"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"
	"blockchain-demo/services"

	"github.com/gin-gonic/gin"
)

// GetAllChainRecords 溯源全部链上记录
// GET /api/regulator/chain/all?page=1&page_size=20
func GetAllChainRecords(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)
	blocks, total := blockchain.Chain.GetAllBlocks(page, pageSize)
	response.OK(c, gin.H{
		"list":     blocks,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// VerifyAnyData 校验任意业务数据是否被篡改
// POST /api/regulator/verify
func VerifyAnyData(c *gin.Context) {
	var req struct {
		DataType string `json:"data_type" binding:"required"` // energy/credit/transaction/report
		DataID   string `json:"data_id" binding:"required"`   // 业务编号
	}
	if !response.BindJSON(c, &req) {
		return
	}
	currentData, err := loadBusinessData(req.DataType, req.DataID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	result := blockchain.Chain.VerifyDataIntegrity(req.DataType, req.DataID, currentData)
	response.OKMsg(c, verifyResultMsg(result), result)
}

// GetPlatformStats 监控平台碳积分发行量等统计数据
// GET /api/regulator/platform/stats
func GetPlatformStats(c *gin.Context) {
	var totalIssued, totalTraded, totalLocked, totalAvailable float64
	var enterpriseCount, transactionCount, chainBlockCount int64

	database.DB.Model(&models.CarbonCredit{}).
		Select("COALESCE(SUM(carbon_credits), 0)").Scan(&totalIssued)
	database.DB.Model(&models.CarbonCredit{}).
		Where("status = ?", "sold").
		Select("COALESCE(SUM(carbon_credits), 0)").Scan(&totalTraded)
	database.DB.Model(&models.CarbonCredit{}).
		Where("status = ?", "locked").
		Select("COALESCE(SUM(carbon_credits), 0)").Scan(&totalLocked)
	database.DB.Model(&models.CarbonCredit{}).
		Where("status = ?", "available").
		Select("COALESCE(SUM(carbon_credits), 0)").Scan(&totalAvailable)

	database.DB.Model(&models.User{}).Where("role = ?", models.RoleEnterprise).Count(&enterpriseCount)
	database.DB.Model(&models.Transaction{}).Count(&transactionCount)
	database.DB.Model(&models.BlockRecord{}).Count(&chainBlockCount)

	response.OK(c, gin.H{
		"total_issued_credits":    totalIssued,
		"total_traded_credits":    totalTraded,
		"total_locked_credits":    totalLocked,
		"total_available_credits": totalAvailable,
		"enterprise_count":        enterpriseCount,
		"transaction_count":       transactionCount,
		"chain_block_count":       chainBlockCount,
		"platform_status":         "运行正常",
		"warning":                 "",
	})
}

// GenerateRiskAlert 输出风险告警
// GET /api/regulator/risk-alert
// 结合整链完整性审计输出链健康风险，其余维持原告警逻辑。
func GenerateRiskAlert(c *gin.Context) {
	var highPriceOrders int64
	var offChainEnergy, offChainCredits int64

	database.DB.Model(&models.SellOrder{}).
		Where("unit_price > 100 AND status = ?", "pending").Count(&highPriceOrders)
	database.DB.Model(&models.EnergyRecord{}).Where("on_chain = ?", false).Count(&offChainEnergy)
	database.DB.Model(&models.CarbonCredit{}).Where("on_chain = ?", false).Count(&offChainCredits)

	alerts := make([]map[string]interface{}, 0)

	// 整链完整性审计(哈希链 + 业务数据回读比对)
	audit := blockchain.Chain.AuditChainIntegrity()
	if audit.Consistent {
		alerts = append(alerts, map[string]interface{}{
			"level":   "success",
			"type":    "chain_status",
			"message": "区块链账本完整一致，未发现篡改",
			"detail": gin.H{
				"block_count": audit.Total,
				"latest_hash": audit.LatestHash,
			},
		})
	} else {
		alerts = append(alerts, map[string]interface{}{
			"level":   "error",
			"type":    "chain_tampered",
			"message": "链完整性审计发现异常区块，疑似数据篡改",
			"detail": gin.H{
				"total":  audit.Total,
				"broken": audit.Broken,
			},
		})
	}

	if highPriceOrders > 0 {
		alerts = append(alerts, map[string]interface{}{
			"level":   "warning",
			"type":    "abnormal_price",
			"message": "存在单价异常偏高的挂单，建议关注",
			"count":   highPriceOrders,
		})
	}

	if offChainEnergy > 0 || offChainCredits > 0 {
		alerts = append(alerts, map[string]interface{}{
			"level":             "info",
			"type":              "off_chain_data",
			"message":           "存在未上链的数据，建议及时上链存证",
			"off_chain_energy":  offChainEnergy,
			"off_chain_credits": offChainCredits,
		})
	}

	response.OK(c, gin.H{
		"alerts":     alerts,
		"alertCount": len(alerts),
	})
}

// GetAllUsers 查看所有用户(监管权限)
// GET /api/regulator/users?page=1&page_size=20
func GetAllUsers(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)

	var total int64
	database.DB.Model(&models.User{}).Count(&total)

	var users []models.User
	if err := database.DB.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		response.ServerError(c, "查询用户列表失败")
		return
	}

	response.OK(c, gin.H{
		"list":     users,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetEmissionReductionAdvice 生成企业节能减排建议
// POST /api/enterprise/advice
// AI 生成(AI服务超时/异常自动降级模板)→ 建议记录落库 → SHA-256 上链存证。
// 越权防护：企业角色仅能为本企业生成建议。
func GetEmissionReductionAdvice(c *gin.Context) {
	var req struct {
		EnterpriseID uint   `json:"enterprise_id" binding:"required"`
		CompanyName  string `json:"company_name" binding:"required"`
		Industry     string `json:"industry"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	// 数据越权拦截
	if !permitOwnData(c, req.EnterpriseID) {
		return
	}

	stats := services.GetEnterpriseCarbonStats(req.EnterpriseID)
	totalEmission, _ := stats["total_emission"].(float64)

	content, source, warn := services.GenerateEmissionReductionAdvice(req.CompanyName, totalEmission, req.Industry)

	report := saveReportAndLog(c, models.ReportKindEnterprise,
		req.CompanyName+"节能减排建议", strconv.FormatUint(uint64(req.EnterpriseID), 10), content, source)
	if report == nil {
		return
	}

	response.OKMsg(c, "节能减排建议生成成功并已上链存证", gin.H{
		"company_name":   req.CompanyName,
		"total_emission": totalEmission,
		"industry":       req.Industry,
		"advice":         report.Content,
		"report_no":      report.ReportNo,
		"block_hash":     report.BlockHash,
		"on_chain":       report.OnChain,
		"source":         report.Source,
		"warn":           warn,
	})
}
