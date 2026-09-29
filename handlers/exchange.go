package handlers

// 碳交易所交易处理器：挂单浏览、交易撮合(原子事务+上链)、碳积分来源核验、交易记录。
// 变更说明(v2)：撮合由"业务先提交、链上后补写"改为"撮合-积分转移-交易凭证-上链"单事务，
// 任一环节失败整体回滚；服务层 MatchAndTransfer 使用条件更新防并发重复成交。

import (
	"fmt"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"
	"blockchain-demo/pkg/response"
	"blockchain-demo/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// transactionDetail 构造交易撮合审计详情文本
func transactionDetail(t *models.Transaction) string {
	return fmt.Sprintf("交易撮合成交: 数量%.2f积分 单价%.2f元 总额%.2f元", t.Quantity, t.UnitPrice, t.TotalAmount)
}

// ListBuyerEnterprises 获取可作买方的园区入驻企业(跨园区候选)
// GET /api/exchange/buyers
// 业务规则：买方不受"只能小微企业互买"限制——园区内所有入驻企业均可作为买家，
// 且不限制买卖双方必须同属一个园区(支持跨园区交易)；园区管理员只做企业入驻与园区
// 数据统计，不参与撮合，全部撮合操作统一由碳交易所完成。故此处返回全部启用状态
// 的小微企业账号(含其所属园区)，供交易所撮合时选择买方。
func ListBuyerEnterprises(c *gin.Context) {
	var list []models.User
	if err := database.DB.Select("id", "username", "company", "park_id", "status", "created_at").
		Where("role = ? AND status = ?", models.RoleEnterprise, 1).
		Order("park_id asc, id asc").
		Find(&list).Error; err != nil {
		logger.Error("查询可作买方企业失败: %v", err)
		response.ServerError(c, "查询可作买方企业失败")
		return
	}
	response.OK(c, gin.H{"list": list})
}

// ListPendingOrders 浏览卖方挂单货源
// GET /api/exchange/orders?page=1&page_size=20
func ListPendingOrders(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)

	var total int64
	// 仅现货挂单(order_type 兜底空值为历史数据)；租赁/远期走 /api/rwa/trade-orders
	database.DB.Model(&models.SellOrder{}).
		Where("status = ? AND (order_type = ? OR order_type = '' OR order_type IS NULL)", "pending", "spot").
		Count(&total)

	var orders []models.SellOrder
	if err := database.DB.Where("status = ? AND (order_type = ? OR order_type = '' OR order_type IS NULL)", "pending", "spot").
		Preload("Enterprise").
		Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&orders).Error; err != nil {
		response.ServerError(c, "查询挂单失败")
		return
	}

	// 批量补充挂单关联的碳积分凭证编号(撮合弹窗展示与校验用)
	creditNoMap := loadCreditNoMap(orders)
	list := make([]gin.H, 0, len(orders))
	for _, o := range orders {
		list = append(list, gin.H{
			"id": o.ID, "order_no": o.OrderNo, "enterprise_id": o.EnterpriseID,
			"credit_id": o.CreditID, "credit_no": creditNoMap[o.CreditID],
			"order_type": o.OrderType,
			"quantity":   o.Quantity, "unit_price": o.UnitPrice, "total_amount": o.TotalAmount,
			"status": o.Status, "buyer_id": o.BuyerID, "block_hash": o.BlockHash,
			"lease_start_date": o.LeaseStartDate, "lease_end_date": o.LeaseEndDate,
			"lease_cycle": o.LeaseCycle, "return_rule": o.ReturnRule,
			"delivery_date": o.DeliveryDate, "expire_at": o.ExpireAt,
			"on_chain": o.OnChain, "created_at": o.CreatedAt, "updated_at": o.UpdatedAt,
			"enterprise": o.Enterprise,
		})
	}

	response.OK(c, gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// loadCreditNoMap 批量查询挂单关联碳积分的凭证编号映射(credit_id → credit_no)
func loadCreditNoMap(orders []models.SellOrder) map[uint]string {
	ids := make([]uint, 0, len(orders))
	seen := map[uint]bool{}
	for _, o := range orders {
		if o.CreditID > 0 && !seen[o.CreditID] {
			ids = append(ids, o.CreditID)
			seen[o.CreditID] = true
		}
	}
	result := map[uint]string{}
	if len(ids) == 0 {
		return result
	}
	var credits []models.CarbonCredit
	if err := database.DB.Select("id", "credit_no").Where("id IN ?", ids).Find(&credits).Error; err != nil {
		return result
	}
	for _, cr := range credits {
		result[cr.ID] = cr.CreditNo
	}
	return result
}

// MatchOrder 交易撮合
// POST /api/exchange/match
// 单事务完成：核验挂单→挂单成交→积分权属转移→生成交易凭证→上链存证→操作审计。
// 业务规则：买方可为园区内任意入驻企业(同园区即"园区内部交易")，亦可为其他园区
// 入驻企业(跨园区交易)，系统不限制买卖双方同园；园区管理员不参与撮合。
func MatchOrder(c *gin.Context) {
	var req models.MatchOrderRequest
	if !response.BindJSON(c, &req) {
		return
	}

	// 买方必须是"启用中的园区入驻小微企业"账号(积分受让人)，可跨园区
	var buyer models.User
	if err := database.DB.First(&buyer, req.BuyerID).Error; err != nil {
		response.NotFound(c, "买方用户不存在")
		return
	}
	if buyer.Role != models.RoleEnterprise {
		response.BadRequest(c, "买方必须是园区入驻小微企业账号")
		return
	}
	if buyer.Status != 1 {
		response.BadRequest(c, "买方企业账号未入驻或已被禁用，无法参与交易")
		return
	}

	var (
		transaction *models.Transaction
		blockHash   string
	)

	// 前置校验：请求携带的碳积分编号必须与挂单实际关联的积分编号一致(防传错/篡改)
	var order models.SellOrder
	if err := database.DB.First(&order, req.OrderID).Error; err != nil {
		response.NotFound(c, "挂单不存在")
		return
	}
	var linkedCredit models.CarbonCredit
	if err := database.DB.Select("credit_no").First(&linkedCredit, order.CreditID).Error; err != nil {
		response.BadRequest(c, "挂单关联碳积分缺失，无法撮合")
		return
	}
	if linkedCredit.CreditNo != req.CreditNo {
		response.BadRequest(c, "碳积分编号与挂单不匹配，无法撮合")
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		t, err := services.MatchAndTransfer(tx, req.OrderID, req.BuyerID)
		if err != nil {
			return err
		}
		transaction = t

		// 交易凭证上链(真实 SHA-256 业务哈希写入新区块)
		block, err := blockchain.AddBlockTx(tx, models.DataTypeTransaction, t.TxNo, t)
		if err != nil {
			return err
		}
		blockHash = block.BlockHash

		// 回写上链状态到交易凭证与挂单
		if err := tx.Model(&models.Transaction{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
			"block_hash": block.BlockHash,
			"on_chain":   true,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SellOrder{}).Where("id = ?", req.OrderID).Updates(map[string]interface{}{
			"block_hash": block.BlockHash,
			"on_chain":   true,
		}).Error; err != nil {
			return err
		}

		return logOperation(tx, c, models.OpTradeMatch, models.DataTypeTransaction, t.TxNo, block.BlockHash,
			transactionDetail(t))
	})
	if err != nil {
		logger.Error("交易撮合失败: %v", err)
		response.ServerError(c, "交易撮合失败: "+err.Error())
		return
	}

	// 重新查询完整数据(带买卖双方信息)用于前端展示
	database.DB.Preload("Seller").Preload("Buyer").First(transaction, transaction.ID)

	response.OKMsg(c, "交易撮合成功，交易凭证已生成并上链存证", gin.H{
		"transaction": transaction,
		"block_hash":  blockHash,
	})
}

// VerifyCreditSource 核验碳积分来源真实性
// POST /api/exchange/verify-credit
// 交易所交易前核验：查询链上记录并重新计算业务哈希比对，杜绝伪造/篡改积分。
func VerifyCreditSource(c *gin.Context) {
	var req struct {
		CreditNo string `json:"credit_no" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}

	query := blockchain.Chain.QueryByDataNo(models.DataTypeCredit, req.CreditNo)
	if query == nil || !query.Exists {
		response.NotFound(c, "该碳积分未在链上找到记录，来源不可信")
		return
	}

	// 回读库内当前积分数据重算哈希做二次比对
	var credit models.CarbonCredit
	if err := database.DB.Where("credit_no = ?", req.CreditNo).First(&credit).Error; err != nil {
		response.NotFound(c, "碳积分记录在数据库中未找到")
		return
	}
	result := blockchain.Chain.VerifyDataIntegrity(models.DataTypeCredit, req.CreditNo, credit)
	if !result.Match {
		response.OKMsg(c, "碳积分数据与链上哈希不一致，来源不可信", result)
		return
	}
	response.OKMsg(c, "碳积分来源核验通过，数据完整可信", result)
}

// ListTransactions 查看交易记录
// GET /api/exchange/transactions?page=1&page_size=20
func ListTransactions(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)

	var total int64
	database.DB.Model(&models.Transaction{}).Count(&total)

	var transactions []models.Transaction
	if err := database.DB.Preload("Seller").Preload("Buyer").
		Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&transactions).Error; err != nil {
		response.ServerError(c, "查询交易记录失败")
		return
	}

	response.OK(c, gin.H{
		"list":     transactions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
