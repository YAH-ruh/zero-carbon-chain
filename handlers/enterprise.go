package handlers

// 企业名录处理器(仅查看)：返回平台全部注册企业清单，供工作台【企业名录】模块展示。
// 只读接口，不修改任何业务数据，与碳积分/隐私核算/IoT 等原有模块互不影响。

import (
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListEnterprises 企业名录
// GET /api/enterprises
// 展示信息：企业名称、企业类型、注册时间、企业状态
func ListEnterprises(c *gin.Context) {
	var users []models.User
	if err := database.DB.Where("role = ?", models.RoleEnterprise).
		Order("id asc").Find(&users).Error; err != nil {
		response.ServerError(c, "查询企业名录失败")
		return
	}

	list := make([]gin.H, 0, len(users))
	for _, u := range users {
		// 全部为园区入驻企业(不存在平台管理主体)
		list = append(list, gin.H{
			"id":              u.ID,
			"company":         u.Company, // 企业名称(未填写时前端回退显示账号)
			"username":        u.Username,
			"enterprise_type": "园区入驻企业",
			"park_id":         u.ParkID,
			"status":          u.Status, // 1-正常 0-禁用
			"created_at":      u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	response.OK(c, gin.H{"list": list, "total": len(list)})
}
