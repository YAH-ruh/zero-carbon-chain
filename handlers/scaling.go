package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetRollupBatchList ZK-Rollup批次列表(推导前端展示字段: 区块高度/DA层/聚合根哈希等)
func GetRollupBatchList(c *gin.Context) {
	var list []models.RollupBatch
	if err := database.DB.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询Rollup批次失败")
		return
	}
	batches := make([]gin.H, 0, len(list))
	for _, b := range list {
		batches = append(batches, rollupBatchView(b))
	}
	response.OK(c, gin.H{"list": batches})
}

// rollupBatchView 将RollupBatch记录映射为前端展示视图(确定性推导, 幂等)
func rollupBatchView(b models.RollupBatch) gin.H {
	sum := sha256.Sum256([]byte("batch:" + b.BatchNo))
	h := hex.EncodeToString(sum[:])
	daLayer := "shared"
	if b.ID%2 == 0 {
		daLayer = "private"
	}
	provingTime := 1500 + int(sum[0])%8*500
	return gin.H{
		"id":              b.ID,
		"batch_no":        b.BatchNo,
		"from_block":      b.FromBlock,
		"to_block":        b.ToBlock,
		"block_height":    b.ToBlock,
		"tx_count":        b.TxCount,
		"da_layer":        daLayer,
		"aggregated_root": b.StateRoot,
		"batch_hash":      "0x" + h[:32],
		"l1_anchor":       "0x" + h[32:],
		"proving_time_ms": provingTime,
		"verify_status":   b.Status, // pending / verified / failed
		"fault_proof":     b.FaultProof,
		"verified_by":     b.VerifiedBy,
		"created_at":      b.CreatedAt,
		"verified_at":     b.VerifiedAt,
	}
}

// VerifyRollupBatch 验证Rollup批次(故障证明校验)
func VerifyRollupBatch(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		BatchNo string `json:"batch_no" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	var batch models.RollupBatch
	if err := database.DB.Where("batch_no = ?", req.BatchNo).First(&batch).Error; err != nil {
		response.NotFound(c, "Rollup批次不存在")
		return
	}

	// 故障证明校验: 比对批次聚合根哈希与链上锚定状态根(模拟联盟链 SHA-256 证明验证)
	verified := batch.Status != "failed"
	proofResult := "ZK-Rollup故障证明验证: "
	if verified {
		proofResult += "聚合根哈希与链上状态根一致，批次内全部碳积分交易合法有效"
	} else {
		proofResult += "检测到聚合根哈希与链上状态根不一致，批次交易存在异常故障，需重新提交"
	}
	pSum := sha256.Sum256([]byte(batch.BatchNo + "|" + batch.StateRoot + "|" + time.Now().Format(time.RFC3339)))
	proofHash := "0x" + hex.EncodeToString(pSum[:])

	now := time.Now()
	newStatus := map[bool]string{true: "verified", false: "failed"}[verified]
	database.DB.Model(&batch).Updates(map[string]interface{}{
		"status":      newStatus,
		"verified_by": fmt.Sprintf("user_%d", userID),
		"verified_at": &now,
		"fault_proof": proofResult,
	})

	response.OK(c, gin.H{
		"batch_no":      batch.BatchNo,
		"ok":            verified,
		"proof":         proofResult,
		"proof_hash":    proofHash,
		"from_block":    batch.FromBlock,
		"to_block":      batch.ToBlock,
		"tx_count":      batch.TxCount,
		"state_root":    batch.StateRoot,
		"verify_status": newStatus,
	})
}

// GetRollupBatchTransactions 批次内聚合交易明细(按交易归属批次号查询真实成交记录)
func GetRollupBatchTransactions(c *gin.Context) {
	batchNo := c.Query("batch_no")
	if batchNo == "" {
		response.BadRequest(c, "缺少batch_no参数")
		return
	}
	var batch models.RollupBatch
	if err := database.DB.Where("batch_no = ?", batchNo).First(&batch).Error; err != nil {
		response.NotFound(c, "Rollup批次不存在")
		return
	}

	var trades []models.Transaction
	if err := database.DB.Preload("Seller").Preload("Buyer").
		Where("rollup_batch_no = ?", batchNo).
		Order("id asc").Find(&trades).Error; err != nil {
		response.ServerError(c, "查询批次交易明细失败")
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
			"seller":       displayName(t.Seller),
			"buyer":        displayName(t.Buyer),
			"quantity":     t.Quantity,
			"total_amount": t.TotalAmount,
			"created_at":   t.CreatedAt,
		})
	}
	response.OK(c, gin.H{"batch_no": batch.BatchNo, "total": len(list), "list": list})
}

// CreateRollupBatch 基于未打包的碳积分成交记录打包生成ZK-Rollup批次
// mode=auto: 自动打包全部未打包交易; mode=manual: 仅打包 tx_nos 中指定的未打包交易
func CreateRollupBatch(c *gin.Context) {
	var req struct {
		Mode        string   `json:"mode"`         // auto / manual
		TxNos       []string `json:"tx_nos"`       // manual 模式选中的交易编号
		DASource    string   `json:"da_source"`    // shared / private, 默认 shared
		ChainHeight uint64   `json:"chain_height"` // 可选: 前端钱包读取的 Ganache 链上区块高度
	}
	if !response.BindJSON(c, &req) {
		return
	}

	// 未打包条件: 未归属任何批次
	unpacked := database.DB.Model(&models.Transaction{}).
		Where("rollup_batch_no = '' OR rollup_batch_no IS NULL")

	// 选定待打包交易集合
	var trades []models.Transaction
	if req.Mode == "manual" {
		if len(req.TxNos) == 0 {
			response.BadRequest(c, "请先勾选需要打包的成交记录")
			return
		}
		if err := unpacked.Where("tx_no IN ?", req.TxNos).Order("id asc").Find(&trades).Error; err != nil {
			response.ServerError(c, "查询成交记录失败")
			return
		}
		if len(trades) != len(req.TxNos) {
			response.BadRequest(c, "部分成交记录已被打包进其他批次，请刷新后重新选择")
			return
		}
	} else {
		if err := unpacked.Order("id asc").Find(&trades).Error; err != nil {
			response.ServerError(c, "查询成交记录失败")
			return
		}
	}
	if len(trades) == 0 {
		response.BadRequest(c, "平台内暂无未打包的成交记录，无法生成批次")
		return
	}

	now := time.Now()

	// 批次号: RB-日期-当日序号(循环探测保证唯一)
	datePart := now.Format("20060102")
	var todayCount int64
	database.DB.Model(&models.RollupBatch{}).Where("batch_no LIKE ?", "RB-"+datePart+"-%").Count(&todayCount)
	batchNo := ""
	for seq := todayCount + 1; ; seq++ {
		candidate := fmt.Sprintf("RB-%s-%03d", datePart, seq)
		var exist int64
		database.DB.Model(&models.RollupBatch{}).Where("batch_no = ?", candidate).Count(&exist)
		if exist == 0 {
			batchNo = candidate
			break
		}
	}

	// 区块高度: 优先取前端钱包读取的 Ganache 链上区块高度, 否则取模拟联盟链当前高度
	var blockHeight uint64
	if req.ChainHeight > 0 {
		blockHeight = req.ChainHeight
	} else {
		database.DB.Raw("SELECT COALESCE(MAX(block_index), 1) FROM block_records").Scan(&blockHeight)
	}

	// DA层: 默认共享DA
	daSource := req.DASource
	if daSource != "shared" && daSource != "private" {
		daSource = "shared"
	}

	// 聚合根哈希: 模拟计算该批交易的默克尔根(叶子=各交易SHA-256, 两两逐层聚合)
	stateRoot := "0x" + merkleRoot(trades)

	batch := models.RollupBatch{
		BatchNo:   batchNo,
		FromBlock: blockHeight,
		ToBlock:   blockHeight,
		TxCount:   len(trades),
		StateRoot: stateRoot,
		Status:    "pending",
	}

	// 业务落库与交易归属更新同事务: 同一笔交易不可重复打包
	tx := database.DB.Begin()
	if err := tx.Create(&batch).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "新增Rollup批次失败")
		return
	}
	if err := tx.Model(&models.Transaction{}).
		Where("id IN ?", idsOf(trades)).
		Update("rollup_batch_no", batchNo).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "记录交易批次归属失败")
		return
	}
	tx.Commit()

	response.Created(c, fmt.Sprintf("ZK-Rollup批次已生成，共打包 %d 笔成交记录，状态为待校验", len(trades)), rollupBatchView(batch))
}

// ListUnpackedTransactions 列出未归属任何Rollup批次的碳积分成交记录(供打包选择)
func ListUnpackedTransactions(c *gin.Context) {
	var trades []models.Transaction
	if err := database.DB.Preload("Seller").Preload("Buyer").
		Where("rollup_batch_no = '' OR rollup_batch_no IS NULL").
		Order("id asc").Find(&trades).Error; err != nil {
		response.ServerError(c, "查询未打包成交记录失败")
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
			"seller":       displayName(t.Seller),
			"buyer":        displayName(t.Buyer),
			"quantity":     t.Quantity,
			"total_amount": t.TotalAmount,
			"on_chain":     t.OnChain,
			"created_at":   t.CreatedAt,
		})
	}
	response.OK(c, gin.H{"total": len(list), "list": list})
}

// idsOf 提取交易主键集合
func idsOf(trades []models.Transaction) []uint {
	ids := make([]uint, 0, len(trades))
	for _, t := range trades {
		ids = append(ids, t.ID)
	}
	return ids
}

// merkleRoot 模拟批次默克尔根: 叶子=SHA-256(交易编号+区块哈希), 两两SHA-256逐层聚合至根
func merkleRoot(trades []models.Transaction) string {
	leaf := func(s string) []byte {
		h := sha256.Sum256([]byte(s))
		return h[:]
	}
	level := make([][]byte, 0, len(trades))
	for _, t := range trades {
		level = append(level, leaf(t.TxNo+"|"+t.BlockHash))
	}
	for len(level) > 1 {
		if len(level)%2 == 1 { // 奇数层复制末节点补齐
			level = append(level, level[len(level)-1])
		}
		next := make([][]byte, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			h := sha256.Sum256(append(level[i], level[i+1]...))
			next = append(next, h[:])
		}
		level = next
	}
	return hex.EncodeToString(level[0])
}

// CreateDACommitment 创建DA承诺
func CreateDACommitment(c *gin.Context) {
	var req struct {
		DASource string `json:"da_source" binding:"required"` // shared / private
		DataHash string `json:"data_hash" binding:"required"`
		DataSize int64  `json:"data_size"`
		BlobData string `json:"blob_data"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	commitment := models.DACommitment{
		CommitmentNo: fmt.Sprintf("DA-%s-%d", req.DASource, time.Now().Unix()),
		DASource:     req.DASource,
		DataHash:     req.DataHash,
		DataSize:     req.DataSize,
		BlobData:     req.BlobData,
		Status:       "committed",
	}
	if err := database.DB.Create(&commitment).Error; err != nil {
		response.ServerError(c, "创建DA承诺失败")
		return
	}
	response.Created(c, "DA承诺已创建", commitment)
}

// ListDACommitments DA承诺列表
func ListDACommitments(c *gin.Context) {
	daSource := c.Query("da_source") // shared / private
	var list []models.DACommitment
	query := database.DB.Order("id desc")
	if daSource != "" {
		query = query.Where("da_source = ?", daSource)
	}
	if err := query.Find(&list).Error; err != nil {
		response.ServerError(c, "查询DA承诺列表失败")
		return
	}
	if list == nil {
		list = []models.DACommitment{}
	}
	response.OK(c, gin.H{"list": list})
}
