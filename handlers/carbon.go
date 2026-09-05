package handlers

// 碳积分业务处理器：核算上链、积分查询、卖出挂单。
// 变更说明(v2)：核算/挂单等写操作与链上存证同一事务提交；企业角色数据越权拦截强化。

import (
	"fmt"
	"time"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"
	"blockchain-demo/pkg/response"
	"blockchain-demo/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CalculateAndUpload 核算碳积分并上链
// POST /api/carbon/calculate
// 真实业务：能耗 × 碳排放因子 → 碳积分核算 → SHA-256 业务哈希写入模拟联盟链。
func CalculateAndUpload(c *gin.Context) {
	var req struct {
		EnergyRecordID uint `json:"energy_record_id" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}

	var energyRecord models.EnergyRecord
	if err := database.DB.First(&energyRecord, req.EnergyRecordID).Error; err != nil {
		response.NotFound(c, "能耗记录不存在")
		return
	}

	// 数据越权拦截：企业角色仅能对本企业能耗记录核算碳积分
	if !permitOwnData(c, energyRecord.EnterpriseID) {
		return
	}

	// 幂等校验：一条能耗记录仅允许核算一次碳积分
	var existed models.CarbonCredit
	if err := database.DB.Where("energy_record_id = ?", req.EnergyRecordID).First(&existed).Error; err == nil {
		response.Conflict(c, "该能耗记录已核算过碳积分(核算编号: "+existed.CreditNo+")")
		return
	}

	// 核算(能耗×排放因子)
	credit := services.CalculateCarbonCredits(&energyRecord)

	// 事务：积分入库 + 上链 + 状态回写 + 审计
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(credit).Error; err != nil {
			return fmt.Errorf("核算记录保存失败: %w", err)
		}
		block, err := blockchain.AddBlockTx(tx, models.DataTypeCredit, credit.CreditNo, credit)
		if err != nil {
			return err
		}
		credit.BlockHash = block.BlockHash
		credit.OnChain = true
		if err := tx.Model(credit).Updates(map[string]interface{}{
			"block_hash": block.BlockHash,
			"on_chain":   true,
		}).Error; err != nil {
			return err
		}
		return logOperation(tx, c, models.OpCreditCalculate, models.DataTypeCredit, credit.CreditNo, block.BlockHash,
			fmt.Sprintf("碳积分核算: 排放%.2fkgCO2 积分%.2f", credit.TotalEmission, credit.CarbonCredits))
	})
	if err != nil {
		logger.Error("碳积分核算失败: %v", err)
		response.ServerError(c, "碳积分核算失败: "+err.Error())
		return
	}

	response.OKMsg(c, "碳积分核算成功，已上链存证", gin.H{
		"credit":     credit,
		"block_hash": credit.BlockHash,
	})
}

// ListMyCredits 查看碳积分列表
// GET /api/carbon/my-credits?enterprise_id=1&page=1&page_size=20
func ListMyCredits(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)

	query := database.DB.Model(&models.CarbonCredit{})
	if isEnterpriseOperator(c) {
		query = query.Where("enterprise_id = ?", currentUserID(c))
	} else if id := parseUint(c.Query("enterprise_id")); id > 0 {
		query = query.Where("enterprise_id = ?", id)
	}

	var total int64
	query.Count(&total)

	var credits []models.CarbonCredit
	if err := query.Preload("Enterprise").
		Preload("Owner").
		Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&credits).Error; err != nil {
		response.ServerError(c, "查询碳积分失败")
		return
	}

	response.OK(c, gin.H{
		"list":     credits,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetCarbonStats 获取企业碳数据统计
// GET /api/carbon/stats?enterprise_id=1
func GetCarbonStats(c *gin.Context) {
	entID := parseUint(c.Query("enterprise_id"))
	if isEnterpriseOperator(c) {
		entID = currentUserID(c)
	}
	if entID == 0 {
		response.BadRequest(c, "缺少企业ID(enterprise_id)")
		return
	}
	response.OK(c, services.GetEnterpriseCarbonStats(entID))
}

// CreateSellOrder 创建挂单卖出
// POST /api/carbon/sell
// 事务：创建挂单 + 锁定碳积分(防止积分被重复挂卖)，业务一致后统一提交。
func CreateSellOrder(c *gin.Context) {
	var req models.SellOrderRequest
	if !response.BindJSON(c, &req) {
		return
	}
	if req.Quantity <= 0 || req.UnitPrice <= 0 {
		response.BadRequest(c, "挂单数量与单价必须大于0")
		return
	}

	var credit models.CarbonCredit
	if err := database.DB.First(&credit, req.CreditID).Error; err != nil {
		response.NotFound(c, "碳积分记录不存在")
		return
	}

	// 数据越权拦截：企业角色仅能挂卖本企业持有的积分
	if !permitOwnData(c, credit.OwnerID) {
		return
	}
	if credit.Status != "available" {
		response.BadRequest(c, "碳积分状态不可卖出: "+credit.Status)
		return
	}
	if req.Quantity > credit.CarbonCredits {
		response.BadRequest(c, "卖出数量超过可用积分")
		return
	}

	orderNo := fmt.Sprintf("SELL-%s-%04d", time.Now().Format("20060102"), credit.ID)
	sellOrder := &models.SellOrder{
		OrderNo:      orderNo,
		EnterpriseID: credit.OwnerID,
		CreditID:     credit.ID,
		Quantity:     req.Quantity,
		UnitPrice:    req.UnitPrice,
		TotalAmount:  req.Quantity * req.UnitPrice,
		Status:       "pending",
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(sellOrder).Error; err != nil {
			return fmt.Errorf("创建挂单失败: %w", err)
		}
		// 仅当积分仍为 available 时才可锁定，防止并发重复挂单
		res := tx.Model(&models.CarbonCredit{}).
			Where("id = ? AND status = ?", credit.ID, "available").
			Update("status", "locked")
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("碳积分状态已变化，无法挂单")
		}
		return logOperation(tx, c, models.OpSellOrderCreate, "sell_order", sellOrder.OrderNo, "",
			fmt.Sprintf("创建卖出挂单: %.2f积分 @%.2f元", sellOrder.Quantity, sellOrder.UnitPrice))
	})
	if err != nil {
		logger.Error("创建挂单失败: %v", err)
		response.ServerError(c, "创建挂单失败: "+err.Error())
		return
	}

	response.OKMsg(c, "挂单创建成功，碳积分已锁定", sellOrder)
}

// ListSellOrders 查看挂单列表
// GET /api/carbon/sell-orders?status=pending&page=1&page_size=20
// 数据权限：企业角色仅返回本企业挂单，其余角色可查看全部。
func ListSellOrders(c *gin.Context) {
	status := c.Query("status")
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)

	query := database.DB.Model(&models.SellOrder{})
	if isEnterpriseOperator(c) {
		query = query.Where("enterprise_id = ?", currentUserID(c))
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var orders []models.SellOrder
	if err := query.Preload("Enterprise").
		Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&orders).Error; err != nil {
		response.ServerError(c, "查询挂单列表失败")
		return
	}

	response.OK(c, gin.H{
		"list":     orders,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
