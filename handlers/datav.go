package handlers

import (
	"fmt"

	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// ==================== 数据中台大屏 ====================

// GetDatavOverview 数据大屏 KPI 概览
// GET /api/datav/overview?range=30
func GetDatavOverview(c *gin.Context) {
	var (
		enterpriseCount   int64
		totalEnergy       float64
		totalCredits      float64
		totalTransactions float64
		txCount           int64
		chainBlocks       int64
	)
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleEnterprise).Count(&enterpriseCount)
	database.DB.Model(&models.EnergyRecord{}).Select("COALESCE(SUM(electricity*0.65 + gas*2.1 + water*0.3),0)").Scan(&totalEnergy)
	database.DB.Model(&models.CarbonCredit{}).Select("COALESCE(SUM(carbon_credits),0)").Scan(&totalCredits)
	database.DB.Model(&models.Transaction{}).Select("COALESCE(SUM(total_amount),0)").Scan(&totalTransactions)
	database.DB.Model(&models.Transaction{}).Count(&txCount)
	database.DB.Model(&models.BlockRecord{}).Count(&chainBlocks)

	// 实时趋势（近 6 个月按月）
	type MonthPoint struct {
		Month        string  `json:"month"`
		Energy       float64 `json:"energy"`
		Credits      float64 `json:"credits"`
		Transactions float64 `json:"transactions"`
	}
	var monthly []MonthPoint
	for i := 5; i >= 0; i-- {
		m := MonthPoint{}
		// 能耗
		database.DB.Model(&models.EnergyRecord{}).
			Where("created_at >= datetime('now', ?)", "start of month", "datetime('now', ?)", fmt.Sprintf("-%d months start of month", i)).
			Select("COALESCE(SUM(electricity*0.65 + gas*2.1 + water*0.3),0)").Scan(&m.Energy)
		// 碳积分
		database.DB.Model(&models.CarbonCredit{}).
			Where("created_at >= datetime('now', ?)", fmt.Sprintf("-%d months start of month", i)).
			Select("COALESCE(SUM(carbon_credits),0)").Scan(&m.Credits)
		// 交易金额
		database.DB.Model(&models.Transaction{}).
			Where("created_at >= datetime('now', ?)", fmt.Sprintf("-%d months start of month", i)).
			Select("COALESCE(SUM(total_amount),0)").Scan(&m.Transactions)
		monthly = append(monthly, m)
	}

	response.OK(c, gin.H{
		"kpi": gin.H{
			"enterprise_count":   enterpriseCount,
			"total_emission":     totalEnergy,
			"total_credits":      totalCredits,
			"total_transactions": totalTransactions,
			"tx_count":           txCount,
			"chain_blocks":       chainBlocks,
		},
		"trend": monthly,
	})
}

// GetParkMap 园区分布(模拟 5 个园区)
// GET /api/datav/park-map
func GetParkMap(c *gin.Context) {
	parks := []gin.H{
		{"name": "绿色科技示范园区", "enterprise_count": 12, "credits": 3200, "energy": 18500, "status": "active", "lat": 31.23, "lng": 121.47},
		{"name": "低碳智造产业园", "enterprise_count": 8, "credits": 2100, "energy": 12300, "status": "active", "lat": 30.27, "lng": 120.16},
		{"name": "生态农业示范区", "enterprise_count": 5, "credits": 1450, "energy": 6800, "status": "active", "lat": 32.06, "lng": 118.80},
		{"name": "智慧能源科技园", "enterprise_count": 9, "credits": 2600, "energy": 14200, "status": "warning", "lat": 22.54, "lng": 113.26},
		{"name": "循环经济试验区", "enterprise_count": 6, "credits": 1780, "energy": 9500, "status": "active", "lat": 28.23, "lng": 112.94},
	}
	response.OK(c, gin.H{"parks": parks, "total": len(parks)})
}

// GetDatavCharts 大屏图表数据(碳积分构成、企业排行、风险告警)
// GET /api/datav/charts
func GetDatavCharts(c *gin.Context) {
	// 碳积分状态分布
	var available, locked, sold, pending float64
	database.DB.Model(&models.CarbonCredit{}).Where("status = ?", "available").Select("COALESCE(SUM(carbon_credits),0)").Scan(&available)
	database.DB.Model(&models.CarbonCredit{}).Where("status = ?", "locked").Select("COALESCE(SUM(carbon_credits),0)").Scan(&locked)
	database.DB.Model(&models.CarbonCredit{}).Where("status = ?", "sold").Select("COALESCE(SUM(carbon_credits),0)").Scan(&sold)

	// 企业碳积分排行 Top 10
	type RankItem struct {
		ID       uint    `json:"id"`
		Name     string  `json:"name"`
		Credits  float64 `json:"credits"`
		Emission float64 `json:"emission"`
	}
	var rank []RankItem
	database.DB.Raw(`
		SELECT u.id, u.username as name, COALESCE(SUM(cc.carbon_credits),0) as credits
		FROM users u
		LEFT JOIN carbon_credits cc ON cc.enterprise_id = u.id
		WHERE u.role = ?
		GROUP BY u.id
		ORDER BY credits DESC
		LIMIT 10
	`, models.RoleEnterprise).Scan(&rank)
	if rank == nil {
		rank = []RankItem{}
	}

	// 最新风险告警
	var risks []models.IoTRecord
	database.DB.Where("risk_label != ''").Order("created_at desc").Limit(10).Find(&risks)

	response.OK(c, gin.H{
		"credit_distribution": gin.H{
			"available": available,
			"locked":    locked,
			"sold":      sold,
			"pending":   pending,
		},
		"enterprise_ranking": rank,
		"risk_alerts":        risks,
	})
}
