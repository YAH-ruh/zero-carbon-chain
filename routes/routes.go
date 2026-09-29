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
			"service": "零碳微证 - 区块链碳积分可信交易平台",
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

		// 企业名录(所有登录角色可查看，仅展示不编辑)
		api.GET("/enterprises", handlers.ListEnterprises)

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
			carbon.POST("/calculate", handlers.CalculateAndUpload)       // 核算碳积分并上链
			carbon.GET("/my-credits", handlers.ListMyCredits)            // 查看碳积分
			carbon.GET("/stats", handlers.GetCarbonStats)                // 碳数据统计
			carbon.POST("/sell", handlers.CreateSellOrder)               // 创建挂单卖出
			carbon.POST("/sell-orders/cancel", handlers.CancelSellOrder) // 撤销挂单(解除积分冻结)
			carbon.GET("/sell-orders", handlers.ListSellOrders)          // 查看挂单列表
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

	// ==================== 创新功能1: IoT设备管理 ====================
	iot := api.Group("/iot")
	iot.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleParkAdmin, models.RoleRegulator))
	{
		iot.GET("/devices", handlers.ListIoTDevices)
		iot.POST("/devices/register", handlers.RegisterIoTDevice)
		iot.POST("/record", handlers.CreateIoTRecord)
		iot.POST("/manual-record", handlers.CreateManualRecord)
		iot.GET("/records", handlers.ListIoTRecords)
	}

	// ==================== 创新功能2: ZKP隐私证明 & PQC抗量子 ====================
	zkp := api.Group("/zkp")
	zkp.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleRegulator))
	{
		zkp.POST("/proof", handlers.GenerateZKPProof)
		zkp.GET("/my-proofs", handlers.ListMyZKProofs)
		zkp.POST("/export-credential", handlers.ExportCredential)
	}
	pqc := api.Group("/pqc")
	pqc.Use(middleware.RoleRequired(models.RoleRegulator))
	{
		pqc.POST("/verify", handlers.PQCVerify)
	}

	// ==================== 创新功能3: RWA碳资产拓展 ====================
	rwa := api.Group("/rwa")
	rwa.Use(middleware.RoleRequired(models.RoleExchange, models.RoleRegulator))
	{
		rwa.GET("/trade-orders", handlers.ListRWATradeOrders)
		rwa.POST("/lease-order", handlers.CreateLeaseOrder)     // 发布租赁挂单(绑定碳积分凭证)
		rwa.POST("/forward-order", handlers.CreateForwardOrder) // 发布远期挂单(绑定碳积分凭证)
	}
	pledge := api.Group("/pledge")
	pledge.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleExchange, models.RoleRegulator))
	{
		pledge.POST("/create", handlers.CreatePledge)
		pledge.POST("/redeem", handlers.RedeemPledge) // 质押赎回(解除积分锁定)
		pledge.GET("/list", handlers.ListPledges)
	}
	arbitration := api.Group("/arbitration")
	arbitration.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleExchange, models.RoleRegulator))
	{
		arbitration.POST("/create", handlers.CreateArbitration)
		arbitration.GET("/list", handlers.ListArbitrations)
		arbitration.GET("/onchain-transactions", handlers.ListOnchainTransactions) // 已上链交易下拉数据源
		arbitration.POST("/resolve", handlers.ResolveArbitration)
	}
	incentive := api.Group("/incentive")
	incentive.Use(middleware.RoleRequired(models.RoleExchange, models.RoleRegulator))
	{
		incentive.GET("/pool", handlers.GetIncentivePool)
		incentive.POST("/claim", handlers.ClaimIncentive)
	}
	archive := api.Group("/archive")
	{
		// 档案列表所有登录角色可见(企业查看自身积分档案, 核查方查看全部)
		archive.GET("/list", handlers.ListCarbonArchive)
		// 核查建档仅交易所/监管可操作
		archive.POST("/create", middleware.RoleRequired(models.RoleExchange, models.RoleRegulator), handlers.CreateCarbonArchive)
	}

	// ==================== 创新功能4: 区块链AI Agent ====================
	agent := api.Group("/agent")
	agent.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleParkAdmin, models.RoleExchange, models.RoleRegulator))
	{
		agent.POST("/trade", handlers.TriggerTradeAgent)
		agent.POST("/risk", handlers.TriggerRiskAgent)
		agent.POST("/dispatch", handlers.TriggerDispatchAgent)
		agent.GET("/records", handlers.ListAgentRecords)
		agent.GET("/zk-anomaly", handlers.GetZKAIAnomalyAlert)
	}

	// ==================== 创新功能5: 扩容架构 ====================
	rollup := api.Group("/rollup")
	rollup.Use(middleware.RoleRequired(models.RoleRegulator))
	{
		rollup.GET("/batches", handlers.GetRollupBatchList)
		rollup.GET("/unpacked-transactions", handlers.ListUnpackedTransactions)
		rollup.POST("/create", handlers.CreateRollupBatch)
		rollup.POST("/verify", handlers.VerifyRollupBatch)
		rollup.GET("/batch/transactions", handlers.GetRollupBatchTransactions)
	}
	da := api.Group("/da")
	da.Use(middleware.RoleRequired(models.RoleRegulator))
	{
		da.POST("/commit", handlers.CreateDACommitment)
		da.GET("/commitments", handlers.ListDACommitments)
	}

	// ==================== 创新功能6: 国产主权链 ====================
	sovereign := api.Group("/sovereign")
	sovereign.Use(middleware.RoleRequired(models.RoleRegulator))
	{
		sovereign.GET("/permissions", handlers.GetPermissionPolicies)
		sovereign.POST("/permissions", handlers.CreatePermissionPolicy)
		sovereign.GET("/anonymous-identity", handlers.GetAnonymousIdentity)
		sovereign.POST("/anonymous-identity", handlers.GenerateAnonymousIdentity)
		sovereign.POST("/cross-chain-report", handlers.CreateCrossChainReport)
		sovereign.GET("/cross-chain-reports", handlers.ListCrossChainReports)
	}

	// ==================== Dashboard & 数据大屏 ====================
	dashboard := api.Group("/dashboard")
	{
		dashboard.GET("/stats", handlers.GetDashboardStats)
	}
	datav := api.Group("/datav")
	datav.Use(middleware.RoleRequired(models.RoleParkAdmin, models.RoleRegulator))
	{
		datav.GET("/overview", handlers.GetDatavOverview)
		datav.GET("/park-map", handlers.GetParkMap)
		datav.GET("/charts", handlers.GetDatavCharts)
	}

	// ==================== 产品碳足迹 ====================
	footprint := api.Group("/footprint")
	footprint.Use(middleware.RoleRequired(models.RoleEnterprise, models.RoleRegulator))
	{
		footprint.GET("/list", handlers.ListProductFootprints)
		footprint.POST("/create", handlers.CreateProductFootprint)
	}

	// ==================== 审计日志 ====================
	audit := api.Group("/audit")
	audit.Use(middleware.RoleRequired(models.RoleRegulator))
	{
		audit.GET("/logs", handlers.ListOperationLogs)
	}

	// ==================== AI 多轮聊天助手（所有角色可用） ====================
	api.POST("/chat", handlers.Chat)

	return r
}
