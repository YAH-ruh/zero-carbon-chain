package models

import "time"

// ProductFootprint 产品碳足迹(全生命周期阶段分解)
type ProductFootprint struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	ProductNo    string    `json:"product_no" gorm:"uniqueIndex;size:64"`
	Name         string    `json:"name" gorm:"size:128"`
	Category     string    `json:"category" gorm:"size:32"`            // 电子/建材/食品/纺织/机械
	EnterpriseID uint      `json:"enterprise_id" gorm:"index;not null"` // 归属企业
	RawMaterials float64   `json:"raw_materials"`                       // 原材料阶段 kgCO₂e
	Manufacture  float64   `json:"manufacture"`                         // 生产制造阶段
	Transport    float64   `json:"transport"`                           // 运输阶段
	Usage        float64   `json:"usage"`                               // 使用阶段
	Waste        float64   `json:"waste"`                               // 废弃阶段
	Total        float64   `json:"total"`                               // 合计
	VerifiedBy   string    `json:"verified_by" gorm:"size:64"`          // 核查机构(可为空)
	BlockHash    string    `json:"block_hash" gorm:"size:128"`
	OnChain      bool      `json:"on_chain" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Enterprise   User      `json:"enterprise,omitempty" gorm:"foreignKey:EnterpriseID"`
}

// ChainCanonical 返回 ProductFootprint 的链上规范数据
func (p ProductFootprint) ChainCanonical() interface{} {
	return map[string]interface{}{
		"product_no":    p.ProductNo,
		"enterprise_id": p.EnterpriseID,
		"category":      p.Category,
		"raw_materials": p.RawMaterials,
		"manufacture":   p.Manufacture,
		"transport":     p.Transport,
		"usage":         p.Usage,
		"waste":         p.Waste,
		"total":         p.Total,
	}
}

// 阶段键名数组(前端渲染堆叠条使用)
var FootprintStages = []string{"raw_materials", "manufacture", "transport", "usage", "waste"}
