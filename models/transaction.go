package models

import "time"

// Transaction 碳积分交易凭证记录
// 交易撮合成功后生成，记录买卖双方、成交数量、金额等核心交易要素，
// 并携带上链区块哈希，保证交易可追溯、不可篡改。
type Transaction struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TxNo        string    `json:"tx_no" gorm:"uniqueIndex;size:64"` // 交易编号，格式: TX-日期-序号
	OrderNo     string    `json:"order_no" gorm:"size:64;index"`    // 关联卖方挂单编号
	SellerID    uint      `json:"seller_id" gorm:"index;not null"`  // 卖方企业ID(碳积分原权属人)
	BuyerID     uint      `json:"buyer_id" gorm:"index;not null"`   // 买方企业ID(碳积分受让人)
	CreditID    uint      `json:"credit_id"`                        // 关联碳积分核算记录ID
	Quantity    float64   `json:"quantity"`                         // 交易碳积分数量
	UnitPrice   float64   `json:"unit_price"`                       // 成交单价(元/碳积分)
	TotalAmount float64   `json:"total_amount"`                     // 成交总金额(元)
	BlockHash   string    `json:"block_hash" gorm:"size:128"`       // 交易上链后的区块哈希
	OnChain     bool      `json:"on_chain" gorm:"default:false"`    // 是否已上链存证
	RollupBatchNo string  `json:"rollup_batch_no" gorm:"size:64;index;default:''"` // 归属ZK-Rollup批次号(空=未打包, 同一交易不可重复打包)
	CreatedAt   time.Time `json:"created_at"`
	Seller      User      `json:"seller,omitempty" gorm:"foreignKey:SellerID"` // 卖方用户信息
	Buyer       User      `json:"buyer,omitempty" gorm:"foreignKey:BuyerID"`   // 买方用户信息
}

// ChainCanonical 交易凭证链上业务哈希规范化视图
func (t Transaction) ChainCanonical() interface{} {
	return map[string]interface{}{
		"tx_no":        t.TxNo,
		"order_no":     t.OrderNo,
		"seller_id":    t.SellerID,
		"buyer_id":     t.BuyerID,
		"credit_id":    t.CreditID,
		"quantity":     t.Quantity,
		"unit_price":   t.UnitPrice,
		"total_amount": t.TotalAmount,
	}
}
