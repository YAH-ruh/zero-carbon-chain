# 零碳微证 - 小微企业区块链碳积分可信交易平台

## 项目简介

零碳微证是一个面向小微企业的区块链碳积分可信交易平台原型系统，基于 Go 语言开发。项目采用本地模拟联盟链技术，无需部署真实区块链节点，适合竞赛演示和概念验证。

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP API (Gin)                         │
├──────────┬──────────┬───────────┬───────────┬───────────────┤
│ 企业角色  │ 园区管理  │  交易所   │  监管核证  │  公开接口     │
├──────────┴──────────┴───────────┴───────────┴───────────────┤
│                    JWT 认证中间件                            │
├─────────────────────────────────────────────────────────────┤
│  业务服务层：碳核算 │ AI建议 │ 交易撮合 │ 数据统计         │
├─────────────────────────────────────────────────────────────┤
│  模拟联盟链 (内存SHA-256链 + SQLite持久化)                   │
├─────────────────────────────────────────────────────────────┤
│  数据库层 (SQLite / 可切换MySQL)                             │
└─────────────────────────────────────────────────────────────┘
```

## 四大角色

| 角色 | 标识 | 职责 |
|------|------|------|
| 小微企业企业角色 | `enterprise` | 模拟能耗数据、碳积分核算、挂单交易、AI减排建议 |
| 工业园区管理员 | `park_admin` | 园区碳数据统计、企业账号管理、园区报告生成 |
| 碳交易所交易角色 | `exchange` | 挂单浏览、交易撮合、碳积分来源核验 |
| 监管核证角色 | `regulator` | 全局监控、链上溯源、篡改校验、风险告警 |

## 技术栈

- **语言**: Go 1.22+
- **Web框架**: Gin
- **数据库**: SQLite (GORM)
- **认证**: JWT (golang-jwt)
- **AI**: DeepSeek API
- **模拟区块链**: SHA-256哈希链（内存 + SQLite持久化）

## 快速启动

### 1. 环境要求

- Go 1.22+
- Git

### 2. 克隆项目

```bash
git clone <项目地址>
cd blockchain-demo
```

### 3. 配置环境变量

```bash
# 复制环境配置
cp .env.example .env
# 编辑 .env 文件，修改配置（尤其是DEEPSEEK_API_KEY）
```

### 4. 编译运行

```bash
# 下载依赖
go mod tidy

# 编译
go build -o weitanlian.exe .

# 运行
./weitanlian.exe
```

### 5. 验证服务

```bash
curl http://localhost:8080/api/health
```

## 快速测试流程

### 使用默认监管账号登录

```bash
# 默认账号: admin / admin123
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

详细API接口文档见 [docs/api.md](docs/api.md)

## 项目结构

```
blockchain-demo/
├── main.go                 # 主入口
├── go.mod / go.sum         # Go模块依赖
├── .env                    # 环境配置文件
├── config/
│   └── config.go           # 配置加载
├── database/
│   └── database.go         # 数据库初始化
├── models/
│   ├── user.go             # 用户模型
│   ├── energy.go           # 能耗数据模型
│   ├── carbon_credit.go    # 碳积分模型
│   └── transaction.go      # 交易/区块模型
├── blockchain/
│   └── mock_chain.go       # 模拟联盟链实现
├── middleware/
│   └── auth.go             # JWT认证 + 角色鉴权
├── handlers/
│   ├── auth.go             # 认证处理器
│   ├── energy.go           # 能耗处理器
│   ├── carbon.go           # 碳积分处理器
│   ├── blockchain.go       # 区块链处理器
│   ├── admin.go            # 园区管理处理器
│   ├── exchange.go         # 交易所处理器
│   └── regulator.go        # 监管核证处理器
├── services/
│   ├── ai.go               # DeepSeek API服务
│   └── carbon.go           # 碳核算服务
├── routes/
│   └── routes.go           # 路由注册
├── docs/
│   └── api.md              # API接口文档
└── README.md               # 项目说明
```

## 原型说明（竞赛答辩用）

### 真实业务实现
- **碳积分核算**: 根据能耗 × 国家碳排放因子计算碳排放量，换算为碳积分
- **交易撮合**: 完整的挂单-成交-权属变更-凭证生成业务流程
- **角色权限**: 四种角色完全隔离的权限体系
- **AI能力**: 通过DeepSeek API生成企业减排建议和园区报告

### 模拟实现
- **模拟联盟链**: 进程内内存维护SHA-256哈希链，通过SQLite持久化。演示了区块链的不可篡改特性（哈希校验），但非真实区块链节点
- **模拟设备采集**: 无真实IoT硬件，通过接口一键生成模拟电表数据（用电量、天然气、用水量）

### 可扩展性
- 数据库：当前使用SQLite，配置DB_PATH连接MySQL即可切换
- 区块链：可替换为FISCO-BCOS等真实联盟链SDK
- IoT设备：可对接真实MQTT或硬件数据采集

## 环境变量说明

| 变量 | 说明 | 默认值 |
|------|------|--------|
| SERVER_PORT | 服务端口 | 8080 |
| DB_PATH | SQLite数据库路径 | ./data/carbon_chain.db |
| JWT_SECRET | JWT签名密钥 | (需修改) |
| JWT_EXPIRE_HOURS | Token过期时间(小时) | 24 |
| DEEPSEEK_API_KEY | DeepSeek API密钥 | (需配置) |
| DEEPSEEK_API_URL | API地址 | https://api.deepseek.com/v1/chat/completions |
| DEEPSEEK_MODEL | 模型名称 | deepseek-chat |
| DEFAULT_ADMIN_USER | 默认管理员账号 | admin |
| DEFAULT_ADMIN_PASS | 默认管理员密码 | admin123 |