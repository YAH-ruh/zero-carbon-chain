package models

import "time"

// 链上数据类型常量(区块 DataType / 上链请求 data_type)
// 每种业务数据均以真实 SHA-256 生成业务哈希并写入独立区块。
const (
	DataTypeEnergy       = "energy"       // 能耗上报数据
	DataTypeCredit       = "credit"       // 碳积分核算数据
	DataTypeTransaction  = "transaction"  // 碳积分交易凭证数据
	DataTypeReport       = "report"       // AI 减排建议/园区报告数据
	DataTypeGenesis      = "genesis"      // 创世区块
)

// BlockRecord 模拟联盟链区块记录(持久化到数据库，服务重启不丢)
// 区块哈希计算 = SHA256(区块高度|时间戳|业务哈希|Merkle根|业务类型|业务单号|前块哈希)
type BlockRecord struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	BlockIndex uint64    `json:"block_index" gorm:"uniqueIndex;not null"`   // 区块高度(自创世块起递增)
	BlockHash  string    `json:"block_hash" gorm:"uniqueIndex;size:128"`    // 当前区块哈希(SHA-256 十六进制)
	PrevHash   string    `json:"prev_hash" gorm:"size:128"`                 // 前一个区块哈希，创世块为空
	DataType   string    `json:"data_type" gorm:"size:32;index"`            // 业务类型标识: energy/credit/transaction/report/genesis
	DataID     string    `json:"data_id" gorm:"size:64;index"`              // 业务数据单号(记录编号/核算编号/交易编号/报告编号)
	DataHash   string    `json:"data_hash" gorm:"size:128"`                 // 业务数据哈希(对业务核心字段做 SHA-256)
	MerkleRoot string    `json:"merkle_root" gorm:"size:128"`               // 简化 Merkle 根(单业务区块对业务哈希再做一次哈希)
	Timestamp  int64     `json:"timestamp"`                                 // 区块生成时间戳(秒)
	CreatedAt  time.Time `json:"created_at"`                                // 记录创建时间
}

// ChainQueryRequest 链上溯源查询请求(可按业务单号或区块哈希查询)
type ChainQueryRequest struct {
	DataNo    string `json:"data_no"`    // 业务单号(记录编号/核算编号/交易编号/报告编号)
	BlockHash string `json:"block_hash"` // 区块哈希(二选一)
}

// ChainVerifyResult 链上校验结果(溯源/篡改校验返回结构)
type ChainVerifyResult struct {
	Exists       bool   `json:"exists"`        // 业务数据是否已上链
	BlockIndex   uint64 `json:"block_index"`   // 所在区块高度
	BlockHash    string `json:"block_hash"`    // 区块哈希
	PrevHash     string `json:"prev_hash"`     // 前块哈希
	MerkleRoot   string `json:"merkle_root"`   // Merkle 根
	DataNo       string `json:"data_no"`       // 业务单号
	DataType     string `json:"data_type"`     // 业务类型
	OriginalHash string `json:"original_hash"` // 链上原始业务哈希
	CurrentHash  string `json:"current_hash"`  // 对当前数据重算的业务哈希
	Match        bool   `json:"match"`         // 哈希是否一致(判断数据是否被篡改)
	Timestamp    int64  `json:"timestamp"`     // 上链时间戳
}

// ChainAuditItem 链完整性审计单区块结果
type ChainAuditItem struct {
	BlockIndex   uint64 `json:"block_index"`   // 区块高度
	DataType     string `json:"data_type"`     // 业务类型
	DataID       string `json:"data_id"`       // 业务单号
	HashValid    bool   `json:"hash_valid"`    // 区块哈希是否正确
	LinkValid    bool   `json:"link_valid"`    // 与前一区块链式关系是否正确
	DataValid    *bool  `json:"data_valid"`    // 业务数据哈希是否与当前数据一致(nil=无需校验,如创世块)
	Issue        string `json:"issue"`         // 问题描述(无问题为空)
}

// ChainAuditReport 链完整性审计汇总(监管一键校验)
type ChainAuditReport struct {
	Total        int               `json:"total"`         // 区块总数
	Valid        int               `json:"valid"`         // 校验通过区块数
	Broken       int               `json:"broken"`        // 校验失败区块数
	Consistent   bool              `json:"consistent"`    // 整链是否完整一致
	FromHeight   uint64            `json:"from_height"`   // 审计起始高度
	ToHeight     uint64            `json:"to_height"`     // 审计结束高度
	LatestHash   string            `json:"latest_hash"`   // 当前最新区块哈希
	Items        []ChainAuditItem  `json:"items"`         // 明细(便于前端展示问题区块)
}
