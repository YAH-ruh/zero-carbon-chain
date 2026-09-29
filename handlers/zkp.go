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

// GenerateZKPProof 生成ZKP选择性披露证明
func GenerateZKPProof(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		ProofType     string   `json:"proof_type" binding:"required"`      // selective_disclosure
		FieldSelector []string `json:"field_selector" binding:"required"` // 选择的公开字段
		PublicData    string   `json:"public_data"`                       // 公开字段数据JSON
	}
	if !response.BindJSON(c, &req) {
		return
	}

	// 模拟ZKP证明：对选择的字段做SHA-256 + 随机数
	seed := fmt.Sprintf("%s-%d-%s", userID, time.Now().UnixNano(), req.PublicData)
	proofHash := fmt.Sprintf("%x", sha256.Sum256([]byte(seed)))
	proofData := fmt.Sprintf("ZKP-Simulated|Fields:%v|Proof:%s", req.FieldSelector, proofHash[:16])

	proofNo := fmt.Sprintf("ZKP-%d-%d", userID, time.Now().Unix())
	record := models.ZKProofRecord{
		ProofNo:       proofNo,
		ProofType:     req.ProofType,
		ProverID:      userID,
		FieldSelector: fmt.Sprintf("%v", req.FieldSelector),
		PublicData:    req.PublicData,
		ProofData:     proofData,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "生成ZKP证明失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeZKProof, proofNo, map[string]interface{}{
		"proof_no":   proofNo,
		"proof_type": req.ProofType,
		"prover_id":  userID,
		"fields":     req.FieldSelector,
		"proof":      proofData,
	})
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "ZKP证明上链失败: "+err.Error())
		return
	}
	tx.Model(&record).Update("block_hash", block.BlockHash)
	tx.Model(&record).Update("on_chain", true)
	tx.Commit()

	response.Created(c, "ZKP选择性披露证明已生成", gin.H{
		"proof_no":  proofNo,
		"proof":     proofData,
		"blockhash": block.BlockHash,
	})
}

// ListMyZKProofs 列出我的ZKP证明
func ListMyZKProofs(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []models.ZKProofRecord
	if err := database.DB.Where("prover_id = ?", userID).
		Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询ZKP证明列表失败")
		return
	}
	if list == nil {
		list = []models.ZKProofRecord{}
	}
	response.OK(c, gin.H{"list": list})
}

// ExportCredential 导出凭证
func ExportCredential(c *gin.Context) {
	var req struct {
		ProofNo string `json:"proof_no" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	var record models.ZKProofRecord
	if err := database.DB.Where("proof_no = ?", req.ProofNo).First(&record).Error; err != nil {
		response.NotFound(c, "ZKP证明不存在")
		return
	}
	// 模拟凭证导出：返回证明数据+区块哈希
	response.OK(c, gin.H{
		"credential": map[string]interface{}{
			"proof_no":       record.ProofNo,
			"proof_type":     record.ProofType,
			"field_selector": record.FieldSelector,
			"public_data":    record.PublicData,
			"proof_data":     record.ProofData,
			"block_hash":     record.BlockHash,
			"on_chain":       record.OnChain,
		},
		"export_format": "JSON",
	})
}

// PQCVerify 监管端PQC抗量子验证(模拟)
func PQCVerify(c *gin.Context) {
	var req struct {
		ProofNo    string `json:"proof_no" binding:"required"`
		PQCAlgo    string `json:"pqc_algo"` // 模拟: PQC-SHAKE256 / PQC-Dilithium
	}
	if !response.BindJSON(c, &req) {
		return
	}
	if req.PQCAlgo == "" {
		req.PQCAlgo = "PQC-SHAKE256"
	}
	var record models.ZKProofRecord
	if err := database.DB.Where("proof_no = ?", req.ProofNo).First(&record).Error; err != nil {
		response.NotFound(c, "待验证的证明不存在")
		return
	}
	// 模拟PQC验证：重新计算hash并与链上数据比对
	recalc := sha256.Sum256([]byte(record.ProofData + req.PQCAlgo))
	recalcHex := fmt.Sprintf("%x", recalc[:16])
	verified := recalcHex[:8] == record.ProofData[len(record.ProofData)-8:]

	response.OK(c, gin.H{
		"proof_no":   record.ProofNo,
		"pqc_algo":   req.PQCAlgo,
		"verified":   verified,
		"on_chain":   record.OnChain,
		"block_hash": record.BlockHash,
		"message":    fmt.Sprintf("PQC抗量子验证完成: %s", map[bool]string{true: "通过 ✓", false: "不通过 ✗"}[verified]),
	})
}