// Package routes 注册所有API路由
// 定义了四种角色的权限隔离，不同角色访问不同接口
// 变更说明(v2)：
//  1. 中间件链升级：AccessLogger(访问日志) + Recovery(全局panic兜底) + CORS；
//  2. 新增 GET /api/chain/audit 整链完整性审计接口(监管一键校验哈希链/篡改)；
//  3. 所有 handler 统一响应 {code,msg,data}，鉴权失败返回统一错误体。
package routes

import (
	"blockchain-demo/handlers"
	"blockchain-demo/middleware"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由和中间件
func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.AccessLogger(), middleware.Recovery(), middleware.CORSMiddleware())

	// 健康检查(公开)
	r.GET("/api/health", func(c *gin.Context) {
		response.OK(c, gin.H{
			"status":  "ok",
			"service": "微碳链 - 区块链碳积分可信交易平台",
		})
	})

	// ==================== 公开接口(无需认证) ====================
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", handlers.Register) // 用户注册(仅小微企业角色)
		auth.POST("/login", handlers.Login)       // 用户登录
	}

	// ==================== 需要认证的接口 ====================
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	{
		// 当前用户信息
		api.GET("/auth/me", handlers.GetCurrentUser)

		// ==================== 区块链接口(所有角色可用，溯源/哈希展示) ====================
		chain := api.Group("/chain")
		{
			chain.POST("/upload", handlers.OnChainData)       // 上链存证(手动补链，幂等)
			chain.GET("/query", handlers.QueryByDataNo)       // 按业务单号溯源查询(展示哈希)
			chain.POST("/verify", handlers.VerifyData)        // 校验数据篡改
			chain.GET("/audit", handlers.AuditChainIntegrity) // 整链完整性审计
			chain.GET("/info", handlers.GetChainInfo)         // 区块链概览
			chain.GET("/blocks", handlers.ListBlocks)         // 区块列表
		}

		// ==================== 企业能耗接口 ====================
		enterprise := api.Group("/energy")
		enterprise.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleParkAdmin, models.RoleRegulator))
		{
			enterprise.POST("/create", handlers.CreateEnergyRecord) // 手动录入能耗(录入即上链)
			enterprise.GET("/list", handlers.ListEnergyRecords)     // 查询能耗记录(企业仅本企业)
		}

		// ==================== 碳积分接口 ====================
		carbon := api.Group("/carbon")
		carbon.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleExchange, models.RoleParkAdmin, models.RoleRegulator))
		{
			carbon.POST("/calculate", handlers.CalculateAndUpload) // 核算碳积分并上链
			carbon.GET("/my-credits", handlers.ListMyCredits)      // 查看碳积分
			carbon.GET("/stats", handlers.GetCarbonStats)          // 碳数据统计
			carbon.POST("/sell", handlers.CreateSellOrder)         // 创建挂单卖出
			carbon.GET("/sell-orders", handlers.ListSellOrders)    // 查看挂单列表
		}

		// ==================== 企业 AI 减排建议接口 ====================
		enterpriseAI := api.Group("/enterprise")
		enterpriseAI.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleParkAdmin, models.RoleRegulator))
		{
			enterpriseAI.POST("/advice", handlers.GetEmissionReductionAdvice) // 生成减排建议(AI,生成即上链)
		}

		// ==================== 园区管理员接口 ====================
		admin := api.Group("/admin")
		admin.Use(middleware.RoleRequired(models.RoleParkAdmin, models.RoleRegulator))
		{
			admin.GET("/park/overview", handlers.GetParkOverview)            // 园区碳数据总览
			admin.GET("/park/enterprises", handlers.ListParkEnterprises)     // 园区企业列表
			admin.PUT("/enterprise/status", handlers.UpdateEnterpriseStatus) // 管理企业入驻状态
			admin.POST("/park/report", handlers.GenerateParkReport)          // 生成园区低碳报告(AI,上链)
			admin.GET("/enterprise/detail", handlers.GetEnterpriseDetail)    // 企业碳数据详情
		}

		// ==================== 交易所角色接口 ====================
		exchange := api.Group("/exchange")
		exchange.Use(middleware.RoleRequired(models.RoleExchange, models.RoleRegulator))
		{
			exchange.GET("/orders", handlers.ListPendingOrders)          // 浏览卖方挂单
			exchange.GET("/buyers", handlers.ListBuyerEnterprises)       // 可作买方入驻企业(园区内/跨园区)
			exchange.POST("/match", handlers.MatchOrder)                 // 交易撮合(事务+上链)
			exchange.POST("/verify-credit", handlers.VerifyCreditSource) // 核验碳积分来源
			exchange.GET("/transactions", handlers.ListTransactions)     // 交易记录
		}

		// ==================== 监管核证角色接口(最高权限) ====================
		regulator := api.Group("/regulator")
		regulator.Use(middleware.RoleRequired(models.RoleRegulator))
		{
			regulator.GET("/chain/all", handlers.GetAllChainRecords)    // 溯源全部链上记录
			regulator.POST("/verify", handlers.VerifyAnyData)           // 校验任意数据篡改
			regulator.GET("/platform/stats", handlers.GetPlatformStats) // 监控平台碳积分发行量
			regulator.GET("/risk-alert", handlers.GenerateRiskAlert)    // 输出风险告警
			regulator.GET("/users", handlers.GetAllUsers)               // 查看所有用户
		}
	}

	return r
}
