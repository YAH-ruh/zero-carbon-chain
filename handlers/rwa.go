package handlers

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==================== RWA交易 ====================

// orderTypeLabel RWA 交易类型中文名
func orderTypeLabel(t string) string {
	switch t {
	case "lease":
		return "碳积分租赁"
	case "forward":
		return "远期合约"
	default:
		return "现货"
	}
}

// ListRWATradeOrders 按挂单类型(orderType)列出现货/租赁/远期挂单(真实数据)
// 与现货市场同口径：仅返回 pending 在售挂单，按 order_type 过滤，
// 并批量补充挂单关联的碳积分凭证编号(credit_no)，供撮合弹窗展示与校验。
func ListRWATradeOrders(c *gin.Context) {
	orderType := c.DefaultQuery("type", "spot") // spot / lease / forward
	if orderType != "lease" && orderType != "forward" && orderType != "spot" {
		orderType = "spot"
	}
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 50)

	query := database.DB.Model(&models.SellOrder{}).
		Where("status = ? AND order_type = ?", "pending", orderType)

	var total int64
	query.Count(&total)

	var list0 []models.SellOrder
	if err := query.Preload("Enterprise").Order("id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&list0).Error; err != nil {
		response.ServerError(c, "查询RWA交易挂单失败")
		return
	}

	// 批量补充挂单关联的碳积分凭证编号(与现货挂单同口径，撮合必带)
	creditNoMap := loadCreditNoMap(list0)
	list := make([]gin.H, 0, len(list0))
	for _, o := range list0 {
		list = append(list, gin.H{
			"id": o.ID, "order_no": o.OrderNo, "enterprise_id": o.EnterpriseID,
			"credit_id": o.CreditID, "credit_no": creditNoMap[o.CreditID],
			"order_type": o.OrderType,
			"quantity":   o.Quantity, "unit_price": o.UnitPrice, "total_amount": o.TotalAmount,
			"status": o.Status, "buyer_id": o.BuyerID,
			"block_hash": o.BlockHash, "on_chain": o.OnChain,
			"lease_start_date": o.LeaseStartDate, "lease_end_date": o.LeaseEndDate,
			"lease_cycle": o.LeaseCycle, "return_rule": o.ReturnRule,
			"delivery_date": o.DeliveryDate, "expire_at": o.ExpireAt,
			"created_at": o.CreatedAt, "updated_at": o.UpdatedAt,
			"enterprise": o.Enterprise,
		})
	}
	response.OK(c, gin.H{"list": list, "total": total, "type": orderType})
}

// typedOrderReq 租赁/远期挂单公共请求要素
type typedOrderReq struct {
	CreditID  uint    `json:"credit_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"required"`
	UnitPrice float64 `json:"unit_price" binding:"required"`
}

// createTypedOrder 租赁/远期挂单创建核心(共用)：
// 必须绑定本人持有的碳积分凭证(credit_id) → 校验权属与积分状态 → "挂单即锁定" →
// 记录类型专属字段 → 后续统一走 /api/exchange/match 撮合流程(核验积分编号 → 成交上链)。
func createTypedOrder(c *gin.Context, orderType string, req typedOrderReq, extra func(o *models.SellOrder)) {
	if req.Quantity <= 0 || req.UnitPrice <= 0 {
		response.BadRequest(c, "挂单数量与单价必须大于0")
		return
	}

	var credit models.CarbonCredit
	if err := database.DB.First(&credit, req.CreditID).Error; err != nil {
		response.NotFound(c, "碳积分记录不存在")
		return
	}

	// 数据越权拦截：企业角色仅能挂卖本企业持有的积分(交易所/监管代客挂单放行)
	if !permitOwnData(c, credit.OwnerID) {
		return
	}
	if credit.Status != "available" {
		response.BadRequest(c, "碳积分状态不可挂单: "+credit.Status)
		return
	}
	if req.Quantity > credit.CarbonCredits {
		response.BadRequest(c, "挂单数量超过可用积分")
		return
	}

	sellOrder := &models.SellOrder{
		OrderNo:      fmt.Sprintf("RWA-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()),
		EnterpriseID: credit.OwnerID,
		CreditID:     credit.ID,
		OrderType:    orderType,
		Quantity:     req.Quantity,
		UnitPrice:    req.UnitPrice,
		TotalAmount:  req.Quantity * req.UnitPrice,
		Status:       "pending",
	}
	if extra != nil {
		extra(sellOrder)
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(sellOrder).Error; err != nil {
			return fmt.Errorf("创建挂单失败: %w", err)
		}
		// 仅当积分仍为 available 时才可锁定，防止同一积分重复挂单
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
			fmt.Sprintf("创建%s挂单: %.2f积分 @%.2f元 (凭证 %s)",
				orderTypeLabel(orderType), sellOrder.Quantity, sellOrder.UnitPrice, credit.CreditNo))
	})
	if err != nil {
		response.ServerError(c, "创建挂单失败: "+err.Error())
		return
	}
	response.Created(c, "挂单创建成功，碳积分已锁定", sellOrder)
}

// CreateLeaseOrder 发布租赁挂单
// POST /api/rwa/lease-order
// 业务规则：企业只出租碳积分使用权，到期归还卖方，所有权不变。
// 额外字段：租赁起止时间(必填，结束须晚于开始)、租赁周期(默认按月)、到期归还规则(默认自动归还)。
func CreateLeaseOrder(c *gin.Context) {
	var req struct {
		typedOrderReq
		LeaseStartDate string `json:"lease_start_date" binding:"required"`
		LeaseEndDate   string `json:"lease_end_date" binding:"required"`
		LeaseCycle     string `json:"lease_cycle"` // day / week / month，默认 month
		ReturnRule     string `json:"return_rule"` // auto / manual，默认 auto
	}
	if !response.BindJSON(c, &req) {
		return
	}
	start, err1 := time.ParseInLocation("2006-01-02", req.LeaseStartDate, time.Local)
	end, err2 := time.ParseInLocation("2006-01-02", req.LeaseEndDate, time.Local)
	if err1 != nil || err2 != nil {
		response.BadRequest(c, "租赁起止时间格式应为 YYYY-MM-DD")
		return
	}
	if !end.After(start) {
		response.BadRequest(c, "租赁结束时间必须晚于开始时间")
		return
	}
	if req.LeaseCycle == "" {
		req.LeaseCycle = "month"
	}
	if req.LeaseCycle != "day" && req.LeaseCycle != "week" && req.LeaseCycle != "month" {
		response.BadRequest(c, "租赁周期仅支持 day/week/month")
		return
	}
	if req.ReturnRule == "" {
		req.ReturnRule = "auto"
	}
	if req.ReturnRule != "auto" && req.ReturnRule != "manual" {
		response.BadRequest(c, "到期归还规则仅支持 auto/manual")
		return
	}
	createTypedOrder(c, "lease", req.typedOrderReq, func(o *models.SellOrder) {
		o.LeaseStartDate = req.LeaseStartDate
		o.LeaseEndDate = req.LeaseEndDate
		o.LeaseCycle = req.LeaseCycle
		o.ReturnRule = req.ReturnRule
	})
}

// CreateForwardOrder 发布远期挂单
// POST /api/rwa/forward-order
// 业务规则：现在锁定远期价格，约定在未来指定日期完成碳积分交割上链。
// 额外字段：约定交割日期(必填)、挂单有效期默认30天(服务端计算 expire_at)。
func CreateForwardOrder(c *gin.Context) {
	var req struct {
		typedOrderReq
		DeliveryDate string `json:"delivery_date" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	delivery, err := time.ParseInLocation("2006-01-02", req.DeliveryDate, time.Local)
	if err != nil {
		response.BadRequest(c, "交割日期格式应为 YYYY-MM-DD")
		return
	}
	if !delivery.After(time.Now()) {
		response.BadRequest(c, "交割日期必须是未来日期")
		return
	}
	expireAt := time.Now().AddDate(0, 0, 30) // 远期挂单有效期30天
	createTypedOrder(c, "forward", req.typedOrderReq, func(o *models.SellOrder) {
		o.DeliveryDate = req.DeliveryDate
		o.ExpireAt = &expireAt
	})
}

// ==================== 质押融资 ====================

// CreatePledge 创建碳积分质押融资
func CreatePledge(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		CreditID     uint    `json:"credit_id"`
		PledgeAmount float64 `json:"pledge_amount" binding:"required"`
		TermMonths   int     `json:"term_months" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	// 期限→年利率白名单(与页面展示一致)：3个月6% / 6个月7% / 12个月8%，利率由服务端决定
	termRates := map[int]float64{3: 0.06, 6: 0.07, 12: 0.08}
	rate, ok := termRates[req.TermMonths]
	if !ok {
		response.BadRequest(c, "质押期限仅支持 3/6/12 个月")
		return
	}
	// 融资额服务端按业务规则计算：积分单价 50 元 × 质押率 80%，与前端预估一致
	loanAmount := math.Round(req.PledgeAmount*50*0.8*100) / 100
	pledgeNo := fmt.Sprintf("PLEDGE-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
	pledge := models.PledgeOrder{
		PledgeNo:     pledgeNo,
		EnterpriseID: userID,
		CreditID:     req.CreditID,
		PledgeAmount: req.PledgeAmount,
		LoanAmount:   loanAmount,
		TermMonths:   req.TermMonths,
		AnnualRate:   rate,
		Status:       "active",
	}
	tx := database.DB.Begin()
	if err := tx.Create(&pledge).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建质押融资失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypePledge, pledgeNo, pledge.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "质押上链失败: "+err.Error())
		return
	}
	tx.Model(&pledge).Update("block_hash", block.BlockHash)
	tx.Model(&pledge).Update("on_chain", true)
	tx.Commit()
	response.Created(c, "碳积分质押融资已创建并上链", pledge)
}

// ListPledges 质押列表
func ListPledges(c *gin.Context) {
	userID := c.GetUint("user_id")
	role := c.GetString("user_role")
	var list []models.PledgeOrder
	query := database.DB
	if role == models.RoleEnterprise {
		query = query.Where("enterprise_id = ?", userID)
	}
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询质押列表失败")
		return
	}
	if list == nil {
		list = []models.PledgeOrder{}
	}
	response.OK(c, gin.H{"list": list})
}

// RedeemPledge 质押赎回(业务联动锁定规则：赎回后扣减质押锁定积分，可用余额恢复)
// POST /api/pledge/redeem  body: {pledge_no}
// 数据越权：仅能赎回本企业质押单；状态机：active → cleared。
func RedeemPledge(c *gin.Context) {
	var req struct {
		PledgeNo string `json:"pledge_no" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}

	var pledge models.PledgeOrder
	if err := database.DB.Where("pledge_no = ? AND enterprise_id = ?", req.PledgeNo, currentUserID(c)).
		First(&pledge).Error; err != nil {
		response.NotFound(c, "质押单不存在或不属于本企业")
		return
	}
	if pledge.Status != "active" {
		response.BadRequest(c, "质押单状态不可赎回: "+pledge.Status)
		return
	}

	// 还款金额 = 本金 + 利息(按年利率×期限计算)，赎回即全额结清
	repayTotal := math.Round(pledge.LoanAmount*(1+pledge.AnnualRate*float64(pledge.TermMonths)/12)*100) / 100

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.PledgeOrder{}).
			Where("id = ? AND status = ?", pledge.ID, "active").
			Updates(map[string]interface{}{"status": "cleared", "repaid_amount": repayTotal})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("质押单状态已变化，无法赎回")
		}
		return logOperation(tx, c, models.OpPledgeRedeem, "pledge_order", pledge.PledgeNo,
			"", fmt.Sprintf("质押赎回: %.2f积分 解除锁定, 偿还本息 %.2f 元", pledge.PledgeAmount, repayTotal))
	})
	if err != nil {
		logger.Error("质押赎回失败: %v", err)
		response.ServerError(c, "质押赎回失败: "+err.Error())
		return
	}
	response.OKMsg(c, fmt.Sprintf("质押已赎回，已偿还本息 ¥%.2f，锁定积分已恢复可用", repayTotal), gin.H{"pledge_no": pledge.PledgeNo})
}

// ==================== 仲裁 ====================

// ListOnchainTransactions 返回全部已上链碳积分交易，供【发起新仲裁】弹窗的
// "关联交易号"下拉选择(数据源为数据库真实上链交易，不使用模拟数据)。
// 返回交易编号、交易双方(名称+ID)、交易时间、成交金额，前端选中后自动回填被诉方与争议金额。
func ListOnchainTransactions(c *gin.Context) {
	var trades []models.Transaction
	if err := database.DB.Preload("Seller").Preload("Buyer").
		Where("on_chain = ?", true).
		Order("id desc").Find(&trades).Error; err != nil {
		response.ServerError(c, "查询已上链交易失败")
		return
	}
	displayName := func(u models.User) string {
		if u.Company != "" {
			return u.Company
		}
		return u.Username
	}
	list := make([]gin.H, 0, len(trades))
	for _, t := range trades {
		list = append(list, gin.H{
			"tx_no":        t.TxNo,
			"seller_id":    t.SellerID,
			"buyer_id":     t.BuyerID,
			"seller_name":  displayName(t.Seller),
			"buyer_name":   displayName(t.Buyer),
			"quantity":     t.Quantity,
			"unit_price":   t.UnitPrice,
			"total_amount": t.TotalAmount,
			"created_at":   t.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}

// CreateArbitration 提交仲裁
// 案件必须关联一笔已上链的碳积分交易(transaction_no)：
// 原告=申诉人，被告=被诉方企业；未指定被诉方/争议金额时自动取链上交易的对手方与成交金额。
func CreateArbitration(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		RespondentID  uint    `json:"respondent_id"`
		Respondent    string  `json:"respondent"`                   // 被诉方企业名称(与库内企业名称匹配)
		TransactionNo string  `json:"transaction_no"`               // 关联已上链交易凭证号
		CaseType      string  `json:"case_type" binding:"required"` // credit_dispute / pledge_dispute / other
		Description   string  `json:"description" binding:"required"`
		Evidence      string  `json:"evidence"`
		DisputeAmount float64 `json:"dispute_amount"`
	}
	if !response.BindJSON(c, &req) {
		return
	}

	// 1. 校验关联交易必须真实上链(无上链交易不生成仲裁案件)
	if req.TransactionNo == "" {
		response.BadRequest(c, "请填写关联的已上链交易编号")
		return
	}
	var trade models.Transaction
	if err := database.DB.Where("tx_no = ? AND on_chain = ?", req.TransactionNo, true).
		First(&trade).Error; err != nil {
		response.BadRequest(c, "关联交易不存在或未上链，无法生成仲裁案件")
		return
	}

	// 2. 解析被诉方：优先ID；其次按企业名称/账号匹配；都缺省时取交易对手方
	var respondent models.User
	if req.RespondentID > 0 {
		if err := database.DB.First(&respondent, req.RespondentID).Error; err != nil {
			response.NotFound(c, "被诉方企业不存在")
			return
		}
	} else if req.Respondent != "" {
		if err := database.DB.Where("role = ? AND (company = ? OR username = ?)",
			models.RoleEnterprise, req.Respondent, req.Respondent).First(&respondent).Error; err != nil {
			response.NotFound(c, "未找到被诉方企业，请填写平台已注册的企业名称")
			return
		}
	} else {
		opponentID := trade.SellerID
		if userID == trade.SellerID {
			opponentID = trade.BuyerID
		}
		if err := database.DB.First(&respondent, opponentID).Error; err != nil {
			response.NotFound(c, "被诉方企业不存在")
			return
		}
	}

	// 3. 争议金额缺省取链上交易成交金额
	amount := req.DisputeAmount
	if amount <= 0 {
		amount = trade.TotalAmount
	}

	caseNo := fmt.Sprintf("ARB-%d-%d", userID, time.Now().Unix())
	arb := models.ArbitrationCase{
		CaseNo:        caseNo,
		ApplicantID:   userID,
		RespondentID:  respondent.ID,
		TransactionNo: trade.TxNo,
		DisputeAmount: amount,
		CaseType:      req.CaseType,
		Description:   req.Description,
		Evidence:      req.Evidence,
		Status:        "pending",
	}
	tx := database.DB.Begin()
	if err := tx.Create(&arb).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "提交仲裁失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeArbitration, caseNo, arb.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "仲裁上链失败: "+err.Error())
		return
	}
	tx.Model(&arb).Update("block_hash", block.BlockHash)
	tx.Model(&arb).Update("on_chain", true)
	tx.Commit()
	response.Created(c, "仲裁案件已提交并上链", arb)
}

// ListArbitrations 仲裁列表
// 返回字段与前端卡片一一对应：原告/被告/关联交易/争议金额/案件描述全部来自真实数据
// (Preload 双方用户取企业名称，verdict 映射为 resolution，ResolvedBy 映射为裁决人名称)。
func ListArbitrations(c *gin.Context) {
	userID := c.GetUint("user_id")
	role := c.GetString("user_role")
	var list []models.ArbitrationCase
	query := database.DB.Preload("Applicant").Preload("Respondent")
	if role == models.RoleEnterprise {
		query = query.Where("applicant_id = ? OR respondent_id = ?", userID, userID)
	}
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询仲裁列表失败")
		return
	}

	// 裁决人名称映射(ResolvedBy → 企业名称/账号)
	resolverNames := map[uint]string{}
	ids := make([]uint, 0, len(list))
	for _, a := range list {
		if a.ResolvedBy > 0 {
			ids = append(ids, a.ResolvedBy)
		}
	}
	if len(ids) > 0 {
		var users []models.User
		database.DB.Select("id", "username", "company").Where("id IN ?", ids).Find(&users)
		for _, u := range users {
			if u.Company != "" {
				resolverNames[u.ID] = u.Company
			} else {
				resolverNames[u.ID] = u.Username
			}
		}
	}

	items := make([]gin.H, 0, len(list))
	for _, a := range list {
		applicantName := a.Applicant.Company
		if applicantName == "" {
			applicantName = a.Applicant.Username
		}
		respondentName := a.Respondent.Company
		if respondentName == "" {
			respondentName = a.Respondent.Username
		}
		items = append(items, gin.H{
			"id":             a.ID,
			"case_no":        a.CaseNo,
			"case_type":      a.CaseType,
			"applicant":      applicantName,  // 原告(申诉方)
			"respondent":     respondentName, // 被告(被诉方)
			"applicant_id":   a.ApplicantID,
			"respondent_id":  a.RespondentID,
			"transaction_no": a.TransactionNo, // 关联上链交易凭证号
			"dispute_amount": a.DisputeAmount,
			"description":    a.Description,
			"status":         a.Status,
			"resolution":     a.Verdict, // 裁决结果
			"resolver":       resolverNames[a.ResolvedBy],
			"resolved_at":    a.ResolvedAt,
			"on_chain":       a.OnChain,
			"block_hash":     a.BlockHash,
			"created_at":     a.CreatedAt,
		})
	}
	response.OK(c, gin.H{"list": items})
}

// ResolveArbitration 裁决仲裁(交易所/监管)
func ResolveArbitration(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		CaseNo  string `json:"case_no" binding:"required"`
		Verdict string `json:"verdict" binding:"required"`
		Status  string `json:"status"` // resolved / dismissed
	}
	if !response.BindJSON(c, &req) {
		return
	}
	if req.Status == "" {
		req.Status = "resolved"
	}
	now := time.Now()
	if err := database.DB.Model(&models.ArbitrationCase{}).
		Where("case_no = ?", req.CaseNo).
		Updates(map[string]interface{}{
			"status":      req.Status,
			"verdict":     req.Verdict,
			"resolved_by": userID,
			"resolved_at": &now,
		}).Error; err != nil {
		response.ServerError(c, "仲裁裁决失败")
		return
	}
	response.OKMsg(c, "仲裁裁决已完成", gin.H{"case_no": req.CaseNo, "status": req.Status})
}

// ==================== 激励池 ====================

// GetIncentivePool 激励池信息
func GetIncentivePool(c *gin.Context) {
	var pool models.IncentivePool
	if err := database.DB.First(&pool).Error; err != nil {
		// 未预置时返回空结构
		response.OK(c, gin.H{
			"pool": models.IncentivePool{
				PoolName: "碳链生态激励池", TotalCredits: 0, Remaining: 0, RewardRate: 0, Status: "inactive",
			},
		})
		return
	}
	response.OK(c, gin.H{"pool": pool})
}

// ClaimIncentive 领取激励
func ClaimIncentive(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	var pool models.IncentivePool
	if err := database.DB.First(&pool).Error; err != nil {
		response.BadRequest(c, "激励池未初始化")
		return
	}
	if pool.Remaining < req.Amount {
		response.BadRequest(c, "激励池余额不足")
		return
	}
	database.DB.Model(&pool).Update("remaining", pool.Remaining-req.Amount)
	response.OKMsg(c, fmt.Sprintf("成功领取 %.2f 碳积分激励", req.Amount), gin.H{
		"claimed":   req.Amount,
		"remaining": pool.Remaining - req.Amount,
	})
}

// ==================== 碳信用档案 ====================

// archiveView 档案视图: 真实档案 + 关联企业名 + 自动档案标记
type archiveView struct {
	models.CarbonArchive
	Company string `json:"company"` // 企业名称(关联 users 表)
	Auto    bool   `json:"auto"`    // 是否由真实积分记录自动映射
	Status  string `json:"status"`  // 积分状态(auto 档案才有意义)
}

// ListCarbonArchive 碳信用档案列表(真实数据)
// 数据源1: carbon_archives 表(手工建档/核查建档, 关联企业名)
// 数据源2: carbon_credits 表中尚未建档的真实积分记录 → 自动映射为链上档案,
//
//	其 archive_no/credit_no/block_hash 均为链上真实存证数据, 可直接核验
func ListCarbonArchive(c *gin.Context) {
	var list []models.CarbonArchive
	if err := database.DB.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询碳信用档案失败")
		return
	}

	users := make(map[uint]models.User)
	loadUser := func(id uint) models.User {
		if u, ok := users[id]; ok {
			return u
		}
		var u models.User
		database.DB.First(&u, id)
		users[id] = u
		return u
	}

	archived := make(map[string]bool, len(list))
	out := make([]archiveView, 0, len(list)+8)
	for _, a := range list {
		archived[a.CreditNo] = true
		out = append(out, archiveView{CarbonArchive: a, Company: loadUser(a.EnterpriseID).Company})
	}

	// 未建档的真实碳积分 → 链上自动档案
	var credits []models.CarbonCredit
	database.DB.Order("id desc").Find(&credits)
	for _, cr := range credits {
		if archived[cr.CreditNo] {
			continue
		}
		out = append(out, archiveView{
			CarbonArchive: models.CarbonArchive{
				ArchiveNo:     "CA-" + strings.TrimPrefix(cr.CreditNo, "CREDIT-"),
				CreditNo:      cr.CreditNo,
				EnterpriseID:  cr.EnterpriseID,
				TotalEmission: cr.TotalEmission,
				CarbonCredits: cr.CarbonCredits,
				SourceDesc:    "链上自动存证",
				VerifiedBy:    "零碳微证智能合约",
				BlockHash:     cr.BlockHash,
				OnChain:       cr.OnChain,
				CreatedAt:     cr.CreatedAt,
			},
			Company: loadUser(cr.EnterpriseID).Company,
			Auto:    true,
			Status:  cr.Status,
		})
	}
	response.OK(c, gin.H{"list": out})
}

// CreateCarbonArchive 创建碳资产档案
func CreateCarbonArchive(c *gin.Context) {
	var req struct {
		CreditNo      string  `json:"credit_no" binding:"required"`
		EnterpriseID  uint    `json:"enterprise_id" binding:"required"`
		TotalEmission float64 `json:"total_emission"`
		CarbonCredits float64 `json:"carbon_credits" binding:"required"`
		SourceDesc    string  `json:"source_desc"`
		VerifiedBy    string  `json:"verified_by"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	archiveNo := fmt.Sprintf("ARC-%d", time.Now().Unix())
	archive := models.CarbonArchive{
		ArchiveNo:     archiveNo,
		CreditNo:      req.CreditNo,
		EnterpriseID:  req.EnterpriseID,
		TotalEmission: req.TotalEmission,
		CarbonCredits: req.CarbonCredits,
		SourceDesc:    req.SourceDesc,
		VerifiedBy:    req.VerifiedBy,
	}
	tx := database.DB.Begin()
	if err := tx.Create(&archive).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建碳资产档案失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeArchive, archiveNo, archive.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "档案上链失败: "+err.Error())
		return
	}
	tx.Model(&archive).Update("block_hash", block.BlockHash)
	tx.Model(&archive).Update("on_chain", true)
	tx.Commit()
	response.Created(c, "碳资产档案已创建并上链", archive)
}
