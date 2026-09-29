package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// ==================== 权限管控 ====================

// GetPermissionPolicies 权限策略列表
func GetPermissionPolicies(c *gin.Context) {
	var list []models.PermissionPolicy
	if err := database.DB.Order("id asc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询权限策略失败")
		return
	}
	if list == nil {
		list = []models.PermissionPolicy{}
	}
	response.OK(c, gin.H{"list": list})
}

// CreatePermissionPolicy 创建细粒度权限策略
func CreatePermissionPolicy(c *gin.Context) {
	var req struct {
		TargetRole  string `json:"target_role" binding:"required"`
		Resource    string `json:"resource" binding:"required"`
		Action      string `json:"action" binding:"required"` // read / write / admin
		Granularity string `json:"granularity"`               // fine / coarse
	}
	if !response.BindJSON(c, &req) {
		return
	}
	if req.Granularity == "" {
		req.Granularity = "fine"
	}
	policy := models.PermissionPolicy{
		PolicyNo:    fmt.Sprintf("POL-%d", time.Now().Unix()),
		TargetRole:  req.TargetRole,
		Resource:    req.Resource,
		Action:      req.Action,
		Granularity: req.Granularity,
	}
	if err := database.DB.Create(&policy).Error; err != nil {
		response.ServerError(c, "创建权限策略失败")
		return
	}
	response.Created(c, "权限策略已创建", policy)
}

// ==================== 可控匿名身份 ====================

// GetAnonymousIdentity 匿名身份列表
func GetAnonymousIdentity(c *gin.Context) {
	var list []models.AnonymousIdentity
	if err := database.DB.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询匿名身份失败")
		return
	}
	if list == nil {
		list = []models.AnonymousIdentity{}
	}
	response.OK(c, gin.H{"list": list})
}

// GenerateAnonymousIdentity 生成可控匿名身份
func GenerateAnonymousIdentity(c *gin.Context) {
	var req struct {
		RealUserID uint `json:"real_user_id" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	// 检查是否已存在
	var existing models.AnonymousIdentity
	if err := database.DB.Where("real_user_id = ?", req.RealUserID).First(&existing).Error; err == nil {
		response.Conflict(c, "该用户已存在匿名身份")
		return
	}

	// 生成模拟匿名身份
	anonBytes := make([]byte, 16)
	rand.Read(anonBytes)
	anonID := fmt.Sprintf("ANON-%x", sha256.Sum256(anonBytes))[:36]
	pubKey := fmt.Sprintf("PK-%x", sha256.Sum256([]byte(anonID+time.Now().String())))[:48]

	identity := models.AnonymousIdentity{
		RealUserID: req.RealUserID,
		AnonID:     anonID,
		PublicKey:  pubKey,
		Status:     "active",
	}
	if err := database.DB.Create(&identity).Error; err != nil {
		response.ServerError(c, "生成匿名身份失败")
		return
	}
	response.Created(c, "可控匿名身份已生成", identity)
}

// ==================== 政务跨链上报 ====================

// CreateCrossChainReport 创建政务跨链上报
func CreateCrossChainReport(c *gin.Context) {
	var req struct {
		ToChain     string `json:"to_chain" binding:"required"` // 模拟政府链
		DataPayload string `json:"data_payload" binding:"required"`
		FromChain   string `json:"from_chain"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	if req.FromChain == "" {
		req.FromChain = "carbon-credit-chain"
	}
	reportNo := fmt.Sprintf("CCR-%d", time.Now().Unix())
	report := models.CrossChainReport{
		ReportNo:    reportNo,
		FromChain:   req.FromChain,
		ToChain:     req.ToChain,
		DataPayload: req.DataPayload,
		Status:      "pending",
	}
	tx := database.DB.Begin()
	if err := tx.Create(&report).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建跨链上报失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeCrossChain, reportNo, report.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "跨链上报上链失败: "+err.Error())
		return
	}
	tx.Model(&report).Update("block_hash", block.BlockHash)
	tx.Model(&report).Update("on_chain", true)
	// 模拟跨链确认
	now := time.Now()
	tx.Model(&report).Update("status", "confirmed")
	tx.Model(&report).Update("confirmed_at", &now)
	tx.Commit()
	response.Created(c, "政务跨链上报已创建并上链确认", report)
}

// ListCrossChainReports 跨链上报列表
func ListCrossChainReports(c *gin.Context) {
	var list []models.CrossChainReport
	if err := database.DB.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询跨链上报列表失败")
		return
	}
	if list == nil {
		list = []models.CrossChainReport{}
	}
	response.OK(c, gin.H{"list": list})
}