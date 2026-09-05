// 微碳链 - 小微企业区块链碳积分可信交易平台
//
// 项目简介：
// 基于Go语言开发的本地模拟联盟链原型系统，面向小微企业提供碳积分可信交易服务。
// 系统使用服务端模拟联盟链(区块持久化落库)，无需部署真实区块链节点，适合竞赛演示。
//
// 四大角色：
// 1. 小微企业企业角色 (enterprise)  - 能耗数据管理、碳积分核算、挂单交易
// 2. 工业园区管理员角色 (park_admin) - 园区碳数据统计、企业账号管理、园区低碳报告
// 3. 碳交易所交易角色 (exchange)     - 挂单浏览、交易撮合、碳积分来源核验
// 4. 监管核证角色 (regulator)        - 全局监控、链上溯源、篡改校验、风险告警
//
// 变更说明(v2)：服务启动流程升级为
// 配置加载 → 分级日志初始化 → 数据库初始化(迁移+预置账号幂等) → 模拟链初始化(创世落库/恢复)
// → 路由注册(全局panic兜底) → 启动HTTP服务。
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"blockchain-demo/blockchain"
	"blockchain-demo/config"
	"blockchain-demo/database"
	"blockchain-demo/pkg/logger"
	"blockchain-demo/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置
	cfg := config.LoadConfig()

	// 2. 初始化分级日志(控制台 + 文件双写)
	logger.Init(cfg.LogLevel, cfg.LogFile)
	defer logger.Close()

	fmt.Println("========================================")
	fmt.Println("  微碳链 - 区块链碳积分可信交易平台")
	fmt.Println("  Micro Carbon Chain v2.0")
	fmt.Println("========================================")

	// 3. 初始化数据库(自动迁移 + 幂等预置4个演示账号)
	database.InitDB()

	// 4. 初始化模拟联盟链(首次创建创世区块并落库，重启自动恢复哈希链)
	blockchain.InitChain(database.DB)

	// 5. 设置路由(访问日志 + 全局panic恢复 + 角色权限隔离)
	router := routes.SetupRouter()

	// 6. 托管前端静态文件(SPA history 路由回退)
	// 修复说明：原实现先用 c.File 输出，文件不存在时已写 404 响应头，
	// 再回退 index.html 会因 "headers already written" 失败，导致刷新/直达子路由 404。
	// 现改为先用 os.Stat 判断目标是否为真实文件：存在→直接返回；否则(目录/SPA路由/不存在)→返回 index.html。
	frontendDist := "./frontend/dist"
	router.Use(func(c *gin.Context) {
		// 仅对非 /api 路径的 GET 请求处理静态文件
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Next()
			return
		}
		filePath := filepath.Join(frontendDist, filepath.Clean(c.Request.URL.Path))
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			c.File(filePath) // 真实静态资源(如 /assets/*.js、/favicon.ico)
			c.Abort()
			return
		}
		// 带扩展名的资源不存在时返回真实 404，避免将 HTML 误当 JS/CSS 交给浏览器解析
		if filepath.Ext(c.Request.URL.Path) != "" {
			c.String(http.StatusNotFound, "404 page not found")
			c.Abort()
			return
		}
		// 目录或 SPA 路由一律回退 index.html，交由前端路由渲染
		c.File(filepath.Join(frontendDist, "index.html"))
		c.Abort()
	})

	// 7. 启动服务
	addr := ":" + cfg.ServerPort
	fmt.Printf("\n🚀 服务启动成功，监听端口: %s\n", addr)
	fmt.Printf("📖 API文档: http://localhost%s/api/health\n", addr)
	fmt.Printf("📝 预置账号: 小微企业001 / 123456, 园区管理员001 / 123456, 碳交易所001 / 123456, 监管核查001 / 123456\n")
	fmt.Printf("🔗 模拟联盟链: 区块持久化至数据库，服务重启不丢失；链完整性校验: GET /api/chain/audit\n")
	fmt.Println("========================================")

	// 8. 启动HTTP服务(gin 内部已具备优雅退出所需的 ListenAndServe 语义)
	if err := router.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
