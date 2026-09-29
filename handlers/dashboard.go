package handlers

import (
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// ==================== Dashboard 统计 ====================

// GetDashboardStats 仪表盘统计(按角色返回不同指标)
// GET /api/dashboard/stats
func GetDashboardStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	role := c.GetString("role")

	result := gin.H{}

	switch role {
	case models.RoleEnterprise:
		result = getEnterpriseDashboard(userID)
	case models.RoleParkAdmin:
		result = getParkDashboard(userID)
	case models.RoleExchange:
		result = getExchangeDashboard(userID)
	case models.RoleRegulator:
		result = getRegulatorDashboard()
	}

	response.OK(c, result)
}

func getEnterpriseDashboard(userID uint) gin.H {
	var energyCount int64
	var creditCount int64
	var availableCredits, lockedCredits, soldCredits float64
	var totalEmission float64
	var sellOrderCount int64
	var pendingOrders int64

	database.DB.Model(&models.EnergyRecord{}).Where("enterprise_id = ?", userID).Count(&energyCount)
	database.DB.Model(&models.CarbonCredit{}).Where("enterprise_id = ?", userID).Count(&creditCount)
	database.DB.Model(&models.CarbonCredit{}).Where("enterprise_id = ? AND status = ?", userID, "available").Select("COALESCE(SUM(carbon_credits),0)").Scan(&availableCredits)
	database.DB.Model(&models.CarbonCredit{}).Where("enterprise_id = ? AND status = ?", userID, "locked").Select("COALESCE(SUM(carbon_credits),0)").Scan(&lockedCredits)
	database.DB.Model(&models.CarbonCredit{}).Where("enterprise_id = ? AND status = ?", userID, "sold").Select("COALESCE(SUM(carbon_credits),0)").Scan(&soldCredits)
	database.DB.Model(&models.EnergyRecord{}).Where("enterprise_id = ?", userID).Select("COALESCE(SUM(electricity*0.65 + gas*2.1 + water*0.3),0)").Scan(&totalEmission)
	database.DB.Model(&models.SellOrder{}).Where("enterprise_id = ?", userID).Count(&sellOrderCount)
	database.DB.Model(&models.SellOrder{}).Where("enterprise_id = ? AND status = ?", userID, "pending").Count(&pendingOrders)

	return gin.H{
		"role": roleName(models.RoleEnterprise),
		"stats": gin.H{
			"energy_records":   energyCount,
			"credits_total":    creditCount,
			"credits_available": availableCredits,
			"credits_locked":   lockedCredits,
			"credits_sold":     soldCredits,
			"total_emission":   totalEmission,
			"sell_orders":      sellOrderCount,
			"pending_orders":   pendingOrders,
		},
	}
}

func getParkDashboard(userID uint) gin.H {
	var enterpriseCount int64
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleEnterprise).Count(&enterpriseCount)

	var totalEnergy, totalCredits, totalTransactions float64
	database.DB.Model(&models.EnergyRecord{}).Select("COALESCE(SUM(electricity),0)").Scan(&totalEnergy)
	database.DB.Model(&models.CarbonCredit{}).Where("status = ?", "available").Select("COALESCE(SUM(carbon_credits),0)").Scan(&totalCredits)
	database.DB.Model(&models.Transaction{}).Select("COALESCE(SUM(total_amount),0)").Scan(&totalTransactions)

	var onlineDevices int64
	database.DB.Model(&models.IoTDevice{}).Where("status = ?", "online").Count(&onlineDevices)

	return gin.H{
		"role": roleName(models.RoleParkAdmin),
		"stats": gin.H{
			"enterprise_count": enterpriseCount,
			"total_energy":     totalEnergy,
			"total_credits":    totalCredits,
			"total_transactions": totalTransactions,
			"iot_devices_online": onlineDevices,
		},
	}
}

func getExchangeDashboard(userID uint) gin.H {
	var pendingOrders int64
	database.DB.Model(&models.SellOrder{}).Where("status = ?", "pending").Count(&pendingOrders)

	var totalTransactions float64
	var txCount int64
	database.DB.Model(&models.Transaction{}).Select("COALESCE(SUM(total_amount),0)").Scan(&totalTransactions)
	database.DB.Model(&models.Transaction{}).Count(&txCount)

	var arbitrations int64
	database.DB.Model(&models.ArbitrationCase{}).Where("status IN ?", []string{"pending", "under_review"}).Count(&arbitrations)

	return gin.H{
		"role": roleName(models.RoleExchange),
		"stats": gin.H{
			"pending_orders": pendingOrders,
			"tx_count":       txCount,
			"tx_total_amount": totalTransactions,
			"arbitrations_pending": arbitrations,
		},
	}
}

func getRegulatorDashboard() gin.H {
	var userCount int64
	database.DB.Model(&models.User{}).Count(&userCount)

	var energyCount, creditCount, txCount, chainBlocks int64
	database.DB.Model(&models.EnergyRecord{}).Count(&energyCount)
	database.DB.Model(&models.CarbonCredit{}).Count(&creditCount)
	database.DB.Model(&models.Transaction{}).Count(&txCount)
	database.DB.Model(&models.BlockRecord{}).Count(&chainBlocks)

	var riskAlerts int64
	database.DB.Model(&models.IoTRecord{}).Where("risk_label != ''").Count(&riskAlerts)

	var pendingArbitrations int64
	database.DB.Model(&models.ArbitrationCase{}).Where("status IN ?", []string{"pending", "under_review"}).Count(&pendingArbitrations)

	return gin.H{
		"role": roleName(models.RoleRegulator),
		"stats": gin.H{
			"user_count":          userCount,
			"energy_records":      energyCount,
			"credit_records":      creditCount,
			"transactions":        txCount,
			"chain_blocks":        chainBlocks,
			"risk_alerts":         riskAlerts,
			"pending_arbitrations": pendingArbitrations,
		},
	}
}

func roleName(r string) string {
	switch r {
	case models.RoleEnterprise:
		return "小微企业"
	case models.RoleParkAdmin:
		return "园区管理员"
	case models.RoleExchange:
		return "碳交易所"
	case models.RoleRegulator:
		return "监管核查"
	}
	return r
}
