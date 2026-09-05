package handlers

// 园区管理员业务处理器：园区碳数据总览、园区企业列表、企业账号状态管理、
// AI 园区低碳报告(落库+上链+审计)、企业碳数据详情。
// 数据权限：园区管理员仅能查看/操作本园区(park_id=自身归属园区)企业。

import (
	"strconv"

	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"
	"blockchain-demo/pkg/response"
	"blockchain-demo/services"

	"github.com/gin-gonic/gin"
)

// operatorParkID 返回操作人所属园区ID(园区管理员角色为自身归属园区，监管为0可查全部)
func operatorParkID(c *gin.Context) uint {
	var operator models.User
	if err := database.DB.First(&operator, currentUserID(c)).Error; err == nil {
		return operator.ParkID
	}
	return 0
}

// parkScope 返回园区管理员可操作的园区范围：非园区管理员(监管)返回0表示不限园区
func parkScope(c *gin.Context) uint {
	if currentRole(c) == models.RoleParkAdmin {
		return operatorParkID(c)
	}
	return 0
}

// GetParkOverview 获取园区碳数据总览
// GET /api/admin/park/overview?park_id=1
func GetParkOverview(c *gin.Context) {
	parkID := parseUint(c.Query("park_id"))
	scope := parkScope(c)
	if scope > 0 {
		parkID = scope // 园区管理员强制查看本园区
	}
	if parkID == 0 {
		parkID = 1 // 默认园区(演示数据园区ID=1)
	}
	response.OK(c, services.GetParkCarbonStats(parkID))
}

// ListParkEnterprises 查看园区企业列表
// GET /api/admin/park/enterprises?park_id=1&page=1&page_size=20
func ListParkEnterprises(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)

	query := database.DB.Model(&models.User{}).Where("role = ?", models.RoleEnterprise)
	scope := parkScope(c)
	if scope > 0 {
		query = query.Where("park_id = ?", scope)
	} else if parkID := parseUint(c.Query("park_id")); parkID > 0 {
		query = query.Where("park_id = ?", parkID)
	}

	var total int64
	query.Count(&total)

	var enterprises []models.User
	if err := query.Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&enterprises).Error; err != nil {
		response.ServerError(c, "查询园区企业失败")
		return
	}

	response.OK(c, gin.H{
		"list":     enterprises,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// UpdateEnterpriseStatus 管理企业账号入驻状态(启用/禁用)
// PUT /api/admin/enterprise/status
// 越权防护：园区管理员仅能启停本园区企业账号。
func UpdateEnterpriseStatus(c *gin.Context) {
	var req struct {
		EnterpriseID uint `json:"enterprise_id" binding:"required"`
		Status       int  `json:"status" binding:"required"` // 1-启用 0-禁用
	}
	if !response.BindJSON(c, &req) {
		return
	}
	if req.Status != 0 && req.Status != 1 {
		response.BadRequest(c, "状态值无效，只能为0(禁用)或1(启用)")
		return
	}

	var target models.User
	if err := database.DB.First(&target, req.EnterpriseID).Error; err != nil {
		response.NotFound(c, "企业账号不存在")
		return
	}
	if target.Role != models.RoleEnterprise {
		response.BadRequest(c, "仅可管理小微企业账号状态")
		return
	}
	// 园区管理员只能操作本园区企业
	if scope := parkScope(c); scope > 0 && target.ParkID != scope {
		response.Forbidden(c, "越权操作：不能管理其他园区的企业账号")
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ? AND role = ?", req.EnterpriseID, models.RoleEnterprise).
		Update("status", req.Status).Error; err != nil {
		logger.Error("更新企业账号状态失败: %v", err)
		response.ServerError(c, "更新账号状态失败")
		return
	}

	statusText := "启用"
	if req.Status == 0 {
		statusText = "禁用"
	}
	logOperation(nil, c, models.OpUserStatusUpdate, "user", target.Username, "", "企业账号"+statusText)
	response.OKMsg(c, "企业账号已"+statusText, nil)
}

// GenerateParkReport 生成园区低碳报告(AI生成 + 记录上链)
// POST /api/admin/park/report
// AI 在事务外调用(可容忍最长 AI_TIMEOUT_SECONDS 秒)，生成结果以单事务落库并 SHA-256 上链，
// 即使 AI 超时/异常也会自动降级为内置模板，接口友好返回不会卡死。
func GenerateParkReport(c *gin.Context) {
	var req struct {
		ParkID   uint   `json:"park_id" binding:"required"`
		ParkName string `json:"park_name" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	if scope := parkScope(c); scope > 0 && req.ParkID != scope {
		response.Forbidden(c, "越权操作：不能为其他园区生成报告")
		return
	}

	stats := services.GetParkCarbonStats(req.ParkID)
	content, source, warn := services.GenerateParkReport(req.ParkName, stats)

	report := saveReportAndLog(c, models.ReportKindPark,
		req.ParkName+"园区低碳发展报告", strconv.FormatUint(uint64(req.ParkID), 10), content, source)
	if report == nil {
		return
	}

	response.OKMsg(c, "园区低碳报告生成成功并已上链存证", gin.H{
		"park_name":  req.ParkName,
		"report":     report.Content,
		"report_no":  report.ReportNo,
		"block_hash": report.BlockHash,
		"on_chain":   report.OnChain,
		"source":     report.Source,
		"warn":       warn,
	})
}

// GetEnterpriseDetail 查看企业碳数据详情
// GET /api/admin/enterprise/detail?enterprise_id=1
func GetEnterpriseDetail(c *gin.Context) {
	entID := parseUint(c.Query("enterprise_id"))
	if entID == 0 {
		response.BadRequest(c, "缺少企业ID(enterprise_id)")
		return
	}

	var enterprise models.User
	if err := database.DB.First(&enterprise, entID).Error; err != nil {
		response.NotFound(c, "企业不存在")
		return
	}
	if enterprise.Role != models.RoleEnterprise {
		response.BadRequest(c, "目标账号不是小微企业")
		return
	}
	if scope := parkScope(c); scope > 0 && enterprise.ParkID != scope {
		response.Forbidden(c, "越权操作：不能查看其他园区企业详情")
		return
	}

	response.OK(c, gin.H{
		"enterprise": enterprise,
		"stats":      services.GetEnterpriseCarbonStats(enterprise.ID),
	})
}
