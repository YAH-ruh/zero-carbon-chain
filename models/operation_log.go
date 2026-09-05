package models

import "time"

// 操作类型常量(OperationLog.Operation)
const (
	OpLogin            = "login"             // 登录
	OpRegister         = "register"          // 注册
	OpEnergyCreate     = "energy_create"     // 能耗上报
	OpCreditCalculate  = "credit_calculate"  // 碳积分核算
	OpCreditTransfer   = "credit_transfer"   // 碳积分权属变更(交易)
	OpSellOrderCreate  = "sell_order_create" // 创建卖出挂单
	OpTradeMatch       = "trade_match"       // 交易撮合
	OpOnChain          = "on_chain"          // 手动上链存证
	OpReportGenerate   = "report_generate"   // AI 报告生成
	OpUserStatusUpdate = "user_status_update" // 企业账号启停
)

// OperationLog 关键操作审计日志(落库，支撑赛事"审计留痕"展示)
// 与文件日志双写：本表可按业务单号检索操作流水。
type OperationLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Operation    string    `json:"operation" gorm:"size:32;index"`   // 操作类型
	OperatorID   uint      `json:"operator_id" gorm:"index"`         // 操作人用户ID
	OperatorName string    `json:"operator_name" gorm:"size:64"`     // 操作人用户名
	Role         string    `json:"role" gorm:"size:32"`              // 操作人角色
	DataType     string    `json:"data_type" gorm:"size:32;index"`   // 关联业务类型
	DataID       string    `json:"data_id" gorm:"size:64;index"`     // 关联业务单号
	BlockHash    string    `json:"block_hash" gorm:"size:128"`       // 关联区块哈希(若有)
	Detail       string    `json:"detail" gorm:"size:512"`           // 操作说明
	IP           string    `json:"ip" gorm:"size:64"`                // 客户端IP(可选)
	CreatedAt    time.Time `json:"created_at"`
}
