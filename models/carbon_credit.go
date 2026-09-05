package models

import "time"

// CarbonCredit 碳积分核算记录
// 根据能耗数据通过碳排放因子自动核算
type CarbonCredit struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	CreditNo      string    `json:"credit_no" gorm:"uniqueIndex;size:64"`  // 核算编号，格式: CREDIT-20240101-001
	EnterpriseID  uint      `json:"enterprise_id" gorm:"index;not null"`   // 所属企业ID
	EnergyRecordID uint     `json:"energy_record_id"`                       // 关联能耗记录ID
	TotalEmission  float64  `json:"total_emission"`                         // 总碳排放量(kgCO₂)
	CarbonCredits  float64  `json:"carbon_credits"`                         // 碳积分量(1积分=1kgCO₂减排量)
	Status         string   `json:"status" gorm:"size:32;default:'available'"` // available-可用, locked-锁定中, sold-已售出
	OwnerID        uint     `json:"owner_id" gorm:"index"`                  // 当前权属人ID
	BlockHash      string   `json:"block_hash" gorm:"size:128"`             // 上链区块哈希
	OnChain        bool     `json:"on_chain" gorm:"default:false"`          // 是否已上链
	CreatedAt      time.Time `json:"created_at"`
	Enterprise     User     `json:"enterprise,omitempty" gorm:"foreignKey:EnterpriseID"`
	Owner          User     `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
}

// ChainCanonical 碳积分链上业务哈希规范化视图(仅取"签发"核心要素)。
// 设计说明：block_hash/on_chain 为上链过程字段，status/owner_id 会随挂单锁定与交易成交
// 发生权属流转——这些状态变化已在"交易区块"中独立留痕，若并入本哈希将导致
// 成交后对积分的篡改校验出现假阳性。故仅对签发要素(单号/归属/排放量/积分量)计算哈希，
// 保证"签发即存证、权属变更另行记账"的语义。
func (c CarbonCredit) ChainCanonical() interface{} {
	return map[string]interface{}{
		"credit_no":       c.CreditNo,
		"enterprise_id":   c.EnterpriseID,
		"energy_record_id": c.EnergyRecordID,
		"total_emission":  c.TotalEmission,
		"carbon_credits":  c.CarbonCredits,
	}
}

// SellOrder 碳积分挂单卖出记录
type SellOrder struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	OrderNo       string    `json:"order_no" gorm:"uniqueIndex;size:64"` // 挂单编号
	EnterpriseID  uint      `json:"enterprise_id" gorm:"index;not null"` // 卖方企业ID
	CreditID      uint      `json:"credit_id"`                            // 碳积分核算记录ID
	Quantity      float64   `json:"quantity"`                             // 卖出数量
	UnitPrice     float64   `json:"unit_price"`                           // 单价(元/积分)
	TotalAmount   float64   `json:"total_amount"`                         // 总金额
	Status        string    `json:"status" gorm:"size:32;default:'pending'"` // pending-挂单中, matched-已成交, cancelled-已取消
	BuyerID       uint      `json:"buyer_id" gorm:"default:0"`            // 买方ID(成交后记录)
	BlockHash     string    `json:"block_hash" gorm:"size:128"`           // 成交上链哈希
	OnChain       bool      `json:"on_chain" gorm:"default:false"`        // 是否已上链
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Enterprise    User      `json:"enterprise,omitempty" gorm:"foreignKey:EnterpriseID"`
}

// ChainCanonical 卖出挂单链上业务哈希规范化视图
func (s SellOrder) ChainCanonical() interface{} {
	return map[string]interface{}{
		"order_no":      s.OrderNo,
		"enterprise_id": s.EnterpriseID,
		"credit_id":     s.CreditID,
		"quantity":      s.Quantity,
		"unit_price":    s.UnitPrice,
		"total_amount":  s.TotalAmount,
		"status":        s.Status,
		"buyer_id":      s.BuyerID,
	}
}

// SellOrderRequest 创建挂单请求
type SellOrderRequest struct {
	CreditID  uint    `json:"credit_id" binding:"required"`
	Quantity  float64 `json:"quantity" binding:"required"`
	UnitPrice float64 `json:"unit_price" binding:"required"`
}

// MatchOrderRequest 交易撮合请求
type MatchOrderRequest struct {
	OrderID  uint `json:"order_id" binding:"required"`
	BuyerID  uint `json:"buyer_id" binding:"required"`
}