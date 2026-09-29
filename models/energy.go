package models

import "time"

// 能耗记录核算状态：只有核算完成(calculated)的能耗，其对应碳积分才计入累计核算积分。
const (
	EnergyStatusPending    = "pending"    // 待核算
	EnergyStatusCalculated = "calculated" // 核算完成(已生成碳凭证入账)
)

// EnergyRecord 企业能耗数据记录(能耗上报核心业务表)
// 数据必须由企业用户在前端页面手动录入提交，后端禁止自动生成/模拟能耗；
// 记录生成后自动计算 SHA-256 业务哈希并写入模拟联盟链，实现"记录即上链"。
type EnergyRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RecordNo     string    `json:"record_no" gorm:"uniqueIndex;size:64"`       // 记录编号，格式: ENERGY-日期-序号
	EnterpriseID uint      `json:"enterprise_id" gorm:"index;not null"`        // 上报企业用户ID(数据归属)
	DeviceID     string    `json:"device_id" gorm:"size:64"`                   // 数据来源设备标识(手动录入统一 MANUAL-企业ID)
	Electricity  float64   `json:"electricity"`                                // 用电量(kWh)
	Gas          float64   `json:"gas"`                                        // 天然气用量(m³)
	Water        float64   `json:"water"`                                      // 用水量(t)
	CollectTime  time.Time `json:"collect_time"`                               // 能耗采集时间
	BlockHash    string    `json:"block_hash" gorm:"size:128"`                 // 上链后的区块哈希
	OnChain      bool      `json:"on_chain" gorm:"default:false"`              // 是否已上链存证
	Status       string    `json:"status" gorm:"size:32;default:'calculated'"` // 待核算/核算完成
	CreatedAt    time.Time `json:"created_at"`
	Enterprise   User      `json:"enterprise,omitempty" gorm:"foreignKey:EnterpriseID"` // 企业用户信息
}

// ChainCanonical 返回参与链上业务哈希的规范化数据(仅核心业务要素)。
// 说明：区块哈希/上链状态等"过程字段"不纳入业务哈希，避免上链状态自身变动造成哈希漂移；
// 上链与篡改校验两侧对同一模型取同一视图，保证哈希可比性。
func (r EnergyRecord) ChainCanonical() interface{} {
	return map[string]interface{}{
		"record_no":     r.RecordNo,
		"enterprise_id": r.EnterpriseID,
		"device_id":     r.DeviceID,
		"electricity":   r.Electricity,
		"gas":           r.Gas,
		"water":         r.Water,
		"collect_time":  r.CollectTime.Format(time.RFC3339),
	}
}

// EnergyCreateRequest 能耗上报请求
type EnergyCreateRequest struct {
	EnterpriseID uint    `json:"enterprise_id"` // 企业ID(企业角色可省略，后端以登录用户为准)
	Electricity  float64 `json:"electricity"`   // 用电量(kWh)，必须大于 0
	Gas          float64 `json:"gas"`           // 天然气用量(m³)，必须大于 0
	Water        float64 `json:"water"`         // 用水量(t)，必须大于 0
}
