package models

import "time"

// 报告类型常量(ReportRecord.Kind)
const (
	ReportKindEnterprise = "enterprise_advice" // 企业节能减排 AI 建议
	ReportKindPark       = "park_report"       // 园区低碳发展 AI 报告
)

// ReportRecord AI 报告/建议记录
// 企业减排建议与园区低碳报告生成后持久化并上链，供溯源核验，保证 AI 产出可审计。
type ReportRecord struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ReportNo  string    `json:"report_no" gorm:"uniqueIndex;size:64"`          // 报告编号，格式: REPORT-日期-序号
	Kind      string    `json:"kind" gorm:"size:32;index"`                     // 报告类型: enterprise_advice / park_report
	CreatorID uint      `json:"creator_id" gorm:"index"`                       // 创建人(操作人)ID
	Creator   User      `json:"creator,omitempty" gorm:"foreignKey:CreatorID"` // 创建人信息
	Title     string    `json:"title" gorm:"size:128"`                         // 报告标题(企业名/园区名 + 业务关键词)
	BizRefNo  string    `json:"biz_ref_no" gorm:"size:64"`                     // 关联业务对象(企业ID或园区ID，冗余便于检索)
	Content   string    `json:"content" gorm:"type:text"`                      // 报告正文(Markdown 文本)
	Source    string    `json:"source" gorm:"size:16;default:'ai'"`            // 内容来源: ai(真实大模型)/fallback(容错降级)
	BlockHash string    `json:"block_hash" gorm:"size:128"`                    // 报告上链后的区块哈希
	OnChain   bool      `json:"on_chain" gorm:"default:false"`                 // 是否已上链存证
	CreatedAt time.Time `json:"created_at"`
}

// ChainCanonical AI 报告链上业务哈希规范化视图
// AI 产出上链实现"AI 内容可审计、可溯源"，防止生成结果事后被篡改。
func (r ReportRecord) ChainCanonical() interface{} {
	return map[string]interface{}{
		"report_no":  r.ReportNo,
		"kind":       r.Kind,
		"creator_id": r.CreatorID,
		"title":      r.Title,
		"biz_ref_no": r.BizRefNo,
		"content":    r.Content,
	}
}

// ReportGenerateRequest AI 报告生成请求(企业建议/园区报告共用)
type ReportGenerateRequest struct {
	Kind         string `json:"kind" binding:"required"` // 报告类型
	EnterpriseID uint   `json:"enterprise_id"`           // 企业ID(企业建议用)
	CompanyName  string `json:"company_name"`            // 企业名称
	Industry     string `json:"industry"`                // 行业(可选)
	ParkID       uint   `json:"park_id"`                 // 园区ID(园区报告用)
	ParkName     string `json:"park_name"`               // 园区名称
}
