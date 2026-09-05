package handlers

// 模拟联盟链相关接口处理器
// 提供：上链存证、按单号溯源查询、篡改校验、链概览、区块分页列表、整链完整性审计。
// 变更说明(v2)：区块结构新增 MerkleRoot 并由接口透出；新增 /chain/audit 供监管一键审计。

import (
	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OnChainData 上链存证接口(手动补链)
// POST /api/chain/upload
// 幂等：同一业务单号重复调用返回已有区块，不会产生重复区块。
func OnChainData(c *gin.Context) {
	var req struct {
		DataType string `json:"data_type" binding:"required"` // energy/credit/transaction/report
		DataID   string `json:"data_id" binding:"required"`   // 业务数据编号
	}
	if !response.BindJSON(c, &req) {
		return
	}

	data, err := loadBusinessData(req.DataType, req.DataID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	var blockHash string
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		block, err := blockchain.AddBlockTx(tx, req.DataType, req.DataID, data)
		if err != nil {
			return err
		}
		blockHash = block.BlockHash
		if err := markOnChainStatus(tx, req.DataType, req.DataID, blockHash); err != nil {
			return err
		}
		return logOperation(tx, c, models.OpOnChain, req.DataType, req.DataID, blockHash, "手动上链存证")
	})
	if txErr != nil {
		response.ServerError(c, "上链存证失败: "+txErr.Error())
		return
	}

	response.OKMsg(c, "上链存证成功", gin.H{
		"block_hash": blockHash,
		"data_type":  req.DataType,
		"data_id":    req.DataID,
	})
}

// QueryByDataNo 根据业务单号溯源查询(哈希值供前端页面展示)
// GET /api/chain/query?data_no=ENERGY-20260905-xxx[&data_type=energy]
func QueryByDataNo(c *gin.Context) {
	dataNo := c.Query("data_no")
	if dataNo == "" {
		response.BadRequest(c, "缺少业务单号(data_no)")
		return
	}
	result := blockchain.Chain.QueryByDataNo(c.Query("data_type"), dataNo)
	if result == nil || !result.Exists {
		response.NotFound(c, "未找到该业务单号的链上记录")
		return
	}
	response.OKMsg(c, "溯源查询成功，区块信息如下", result)
}

// VerifyData 校验数据是否被篡改
// POST /api/chain/verify
// 重新读取库内业务数据并计算 SHA-256，与链上原始业务哈希比对。
func VerifyData(c *gin.Context) {
	var req struct {
		DataType string `json:"data_type" binding:"required"`
		DataID   string `json:"data_id" binding:"required"`
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
	msg := verifyResultMsg(result)
	response.OKMsg(c, msg, result)
}

// verifyResultMsg 将校验结果转成面向用户的中文结论(含哈希值信息)
func verifyResultMsg(r *models.ChainVerifyResult) string {
	if r == nil || !r.Exists {
		return "该业务数据未上链，无法校验"
	}
	if !r.Match {
		return "数据完整性校验失败：当前数据哈希与链上原始哈希不一致，疑似被篡改"
	}
	return "数据完整性校验通过，数据未被篡改(哈希链完整)"
}

// GetChainInfo 获取区块链概览信息
// GET /api/chain/info
func GetChainInfo(c *gin.Context) {
	response.OK(c, blockchain.Chain.GetChainInfo())
}

// ListBlocks 获取区块列表(前端展示哈希字符串)
// GET /api/chain/blocks?page=1&page_size=20
func ListBlocks(c *gin.Context) {
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

// AuditChainIntegrity 整链完整性审计(识别数据篡改)
// GET /api/chain/audit
// 监管一键校验：逐块重算哈希 + 校验前块链式关系 + 回读业务数据比对业务哈希。
func AuditChainIntegrity(c *gin.Context) {
	report := blockchain.Chain.AuditChainIntegrity()
	msg := "整链校验通过，区块完整一致"
	if !report.Consistent {
		msg = "整链校验发现异常区块，请关注明细"
	}
	response.OKMsg(c, msg, report)
}
