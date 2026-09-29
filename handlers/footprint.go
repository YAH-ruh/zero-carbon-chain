package handlers

import (
	"fmt"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListProductFootprints 产品碳足迹列表
// GET /api/footprint/list?category=电子
func ListProductFootprints(c *gin.Context) {
	userID := c.GetUint("user_id")
	role := c.GetString("role")
	category := c.Query("category")

	query := database.DB.Model(&models.ProductFootprint{})
	if role == models.RoleEnterprise {
		query = query.Where("enterprise_id = ?", userID)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var list []models.ProductFootprint
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询产品碳足迹失败")
		return
	}
	if list == nil {
		list = []models.ProductFootprint{}
	}

	// 汇总统计
	var totalCount int64
	var totalFootprint float64
	database.DB.Model(&models.ProductFootprint{}).Count(&totalCount)
	database.DB.Model(&models.ProductFootprint{}).Select("COALESCE(SUM(total), 0)").Scan(&totalFootprint)

	// 提取阶段数据供前端堆叠条使用
	type ProductItem struct {
		models.ProductFootprint
		Stages []float64 `json:"stages"` // [raw, mfg, transport, usage, waste]
	}
	items := make([]ProductItem, 0, len(list))
	for _, p := range list {
		items = append(items, ProductItem{
			ProductFootprint: p,
			Stages:           []float64{p.RawMaterials, p.Manufacture, p.Transport, p.Usage, p.Waste},
		})
	}

	response.OK(c, gin.H{
		"list":           items,
		"total":          totalCount,
		"total_footprint": totalFootprint,
	})
}

// CreateProductFootprint 创建产品碳足迹(创建即上链)
// POST /api/footprint/create
func CreateProductFootprint(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Name         string  `json:"name" binding:"required"`
		Category     string  `json:"category" binding:"required"`
		RawMaterials float64 `json:"raw_materials"`
		Manufacture  float64 `json:"manufacture"`
		Transport    float64 `json:"transport"`
		Usage        float64 `json:"usage"`
		Waste        float64 `json:"waste"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	total := req.RawMaterials + req.Manufacture + req.Transport + req.Usage + req.Waste
	pf := models.ProductFootprint{
		ProductNo:    fmt.Sprintf("PFP-%d-%d", userID, c.GetUint("request_id")),
		Name:         req.Name,
		Category:     req.Category,
		EnterpriseID: userID,
		RawMaterials: req.RawMaterials,
		Manufacture:  req.Manufacture,
		Transport:    req.Transport,
		Usage:        req.Usage,
		Waste:        req.Waste,
		Total:        total,
	}
	tx := database.DB.Begin()
	if err := tx.Create(&pf).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建产品碳足迹失败: "+err.Error())
		return
	}
	pf.ProductNo = fmt.Sprintf("PFP-%d", pf.ID)
	block, err := blockchain.AddBlockTx(tx, models.DataTypeFootprint, pf.ProductNo, pf.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "产品碳足迹上链失败: "+err.Error())
		return
	}
	tx.Model(&pf).Update("product_no", pf.ProductNo)
	tx.Model(&pf).Update("block_hash", block.BlockHash)
	tx.Model(&pf).Update("on_chain", true)
	tx.Commit()
	response.Created(c, "产品碳足迹核算已创建并上链", pf)
}
