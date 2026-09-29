package handlers

import (
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListOperationLogs 审计日志列表
// GET /api/audit/logs?operation=&role=&page=1&page_size=20
func ListOperationLogs(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)
	operation := c.Query("operation")
	role := c.Query("role")
	dataType := c.Query("data_type")

	query := database.DB.Model(&models.OperationLog{})
	if operation != "" {
		query = query.Where("operation = ?", operation)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if dataType != "" {
		query = query.Where("data_type = ?", dataType)
	}

	var total int64
	query.Count(&total)

	var list []models.OperationLog
	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		response.ServerError(c, "查询审计日志失败")
		return
	}
	if list == nil {
		list = []models.OperationLog{}
	}

	response.OK(c, gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
