package models

import "time"

// ========== IoT Device Management ==========

// IoTDevice IoT设备注册信息
type IoTDevice struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	DeviceID     string    `json:"device_id" gorm:"uniqueIndex;size:64"`
	DeviceName   string    `json:"device_name" gorm:"size:128"`
	EnterpriseID uint      `json:"enterprise_id" gorm:"index;not null"`
	DeviceType   string    `json:"device_type" gorm:"size:32"`             // sensor/gateway/meter
	Status       string    `json:"status" gorm:"size:16;default:'online'"` // online/offline/error
	LastOnline   time.Time `json:"last_online"`
	CreatedAt    time.Time `json:"created_at"`
	Enterprise   User      `json:"enterprise,omitempty" gorm:"foreignKey:EnterpriseID"`
}

// IoTRecord IoT采集/手动录入记录
type IoTRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RecordNo     string    `json:"record_no" gorm:"uniqueIndex;size:64"`
	DeviceID     string    `json:"device_id" gorm:"size:64;index"`
	EnterpriseID uint      `json:"enterprise_id" gorm:"index;not null"`
	Source       string    `json:"source" gorm:"size:16"`     // auto / manual
	RiskLabel    string    `json:"risk_label" gorm:"size:32"` // 手动录入: 人工标记-低/中/高; 自动采集: ""
	Electricity  float64   `json:"electricity"`
	Gas          float64   `json:"gas"`
	Water        float64   `json:"water"`
	CollectTime  time.Time `json:"collect_time"`
	BlockHash    string    `json:"block_hash" gorm:"size:128"`
	OnChain      bool      `json:"on_chain" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	Enterprise   User      `json:"enterprise,omitempty" gorm:"foreignKey:EnterpriseID"`
}

// ChainCanonical 返回IoTRecord的链上规范数据
func (r IoTRecord) ChainCanonical() interface{} {
	return map[string]interface{}{
		"record_no":     r.RecordNo,
		"device_id":     r.DeviceID,
		"enterprise_id": r.EnterpriseID,
		"source":        r.Source,
		"risk_label":    r.RiskLabel,
		"electricity":   r.Electricity,
		"gas":           r.Gas,
		"water":         r.Water,
		"collect_time":  r.CollectTime.Format(time.RFC3339),
	}
}

// ========== RWA Carbon Asset Expansion ==========

// PledgeOrder 碳积分质押融资订单
type PledgeOrder struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	PledgeNo     string    `json:"pledge_no" gorm:"uniqueIndex;size:64"`
	EnterpriseID uint      `json:"enterprise_id" gorm:"index;not null"`
	CreditID     uint      `json:"credit_id"`
	PledgeAmount float64   `json:"pledge_amount"`
	LoanAmount   float64   `json:"loan_amount"`
	TermMonths   int       `json:"term_months"`   // 质押期限(月)：3/6/12
	AnnualRate   float64   `json:"annual_rate"`   // 年利率(小数)：0.06/0.07/0.08
	RepaidAmount float64   `json:"repaid_amount"` // 已偿还金额(本金+利息)，赎回时写入
	Status       string    `json:"status" gorm:"size:32;default:'active'"` // active / cleared / overdue
	BlockHash    string    `json:"block_hash" gorm:"size:128"`
	OnChain      bool      `json:"on_chain" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	Enterprise   User      `json:"enterprise,omitempty" gorm:"foreignKey:EnterpriseID"`
}

// ChainCanonical 返回PledgeOrder的链上规范数据
func (p PledgeOrder) ChainCanonical() interface{} {
	return map[string]interface{}{
		"pledge_no":     p.PledgeNo,
		"enterprise_id": p.EnterpriseID,
		"credit_id":     p.CreditID,
		"pledge_amount": p.PledgeAmount,
		"loan_amount":   p.LoanAmount,
		"status":        p.Status,
	}
}

// ArbitrationCase 交易仲裁案件
// 案件必须由已上链的碳积分交易产生纠纷后生成：原告/被告/关联交易/争议金额均对齐链上交易数据
type ArbitrationCase struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	CaseNo        string     `json:"case_no" gorm:"uniqueIndex;size:64"`
	ApplicantID   uint       `json:"applicant_id" gorm:"index"`
	RespondentID  uint       `json:"respondent_id" gorm:"index"`
	TransactionNo string     `json:"transaction_no" gorm:"size:64;index"` // 关联已上链交易凭证号(TX-*)，无上链交易不生成案件
	DisputeAmount float64    `json:"dispute_amount"`                      // 争议金额(元，缺省取关联交易成交金额)
	CaseType      string     `json:"case_type" gorm:"size:32"`            // credit_dispute / pledge_dispute / other
	Description   string     `json:"description" gorm:"type:text"`
	Evidence      string     `json:"evidence" gorm:"type:text"`               // JSON 证据引用
	Status        string     `json:"status" gorm:"size:32;default:'pending'"` // pending / under_review / resolved / dismissed
	Verdict       string     `json:"verdict" gorm:"type:text"`
	ResolvedBy    uint       `json:"resolved_by"` // 裁决人ID(交易所/监管)
	BlockHash     string     `json:"block_hash" gorm:"size:128"`
	OnChain       bool       `json:"on_chain" gorm:"default:false"`
	CreatedAt     time.Time  `json:"created_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	Applicant     User       `json:"applicant,omitempty" gorm:"foreignKey:ApplicantID"`
	Respondent    User       `json:"respondent,omitempty" gorm:"foreignKey:RespondentID"`
}

// ChainCanonical 返回ArbitrationCase的链上规范数据
func (a ArbitrationCase) ChainCanonical() interface{} {
	return map[string]interface{}{
		"case_no":        a.CaseNo,
		"applicant_id":   a.ApplicantID,
		"respondent_id":  a.RespondentID,
		"transaction_no": a.TransactionNo,
		"case_type":      a.CaseType,
		"status":         a.Status,
	}
}

// IncentivePool 链上激励池
type IncentivePool struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	PoolName     string    `json:"pool_name" gorm:"size:128"`
	TotalCredits float64   `json:"total_credits"`
	Remaining    float64   `json:"remaining"`
	RewardRate   float64   `json:"reward_rate"`
	Status       string    `json:"status" gorm:"size:32;default:'active'"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CarbonArchive 链上碳信用档案
type CarbonArchive struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ArchiveNo     string    `json:"archive_no" gorm:"uniqueIndex;size:64"`
	CreditNo      string    `json:"credit_no" gorm:"size:64;index"`
	EnterpriseID  uint      `json:"enterprise_id" gorm:"index"`
	TotalEmission float64   `json:"total_emission"`
	CarbonCredits float64   `json:"carbon_credits"`
	SourceDesc    string    `json:"source_desc" gorm:"type:text"`
	VerifiedBy    string    `json:"verified_by" gorm:"size:64"`
	BlockHash     string    `json:"block_hash" gorm:"size:128"`
	OnChain       bool      `json:"on_chain" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
}

// ChainCanonical 返回CarbonArchive的链上规范数据
func (a CarbonArchive) ChainCanonical() interface{} {
	return map[string]interface{}{
		"archive_no":     a.ArchiveNo,
		"credit_no":      a.CreditNo,
		"enterprise_id":  a.EnterpriseID,
		"total_emission": a.TotalEmission,
		"carbon_credits": a.CarbonCredits,
		"verified_by":    a.VerifiedBy,
	}
}

// ========== Blockchain AI ==========

// AgentRecord AI Agent操作记录(全部上链存证)
type AgentRecord struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	AgentNo    string    `json:"agent_no" gorm:"uniqueIndex;size:64"`
	AgentType  string    `json:"agent_type" gorm:"size:32"` // trade / risk / park_dispatch
	TriggerBy  uint      `json:"trigger_by"`
	InputData  string    `json:"input_data" gorm:"type:text"`
	OutputData string    `json:"output_data" gorm:"type:text"`
	ZKProofRef string    `json:"zk_proof_ref" gorm:"size:128"`
	BlockHash  string    `json:"block_hash" gorm:"size:128"`
	OnChain    bool      `json:"on_chain" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at"`
}

// ChainCanonical 返回AgentRecord的链上规范数据
func (a AgentRecord) ChainCanonical() interface{} {
	return map[string]interface{}{
		"agent_no":   a.AgentNo,
		"agent_type": a.AgentType,
		"trigger_by": a.TriggerBy,
		"input":      a.InputData,
		"output":     a.OutputData,
	}
}

// ZKProofRecord ZKP证明记录(选择性披露 / PQC / ZK-AI)
type ZKProofRecord struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ProofNo       string    `json:"proof_no" gorm:"uniqueIndex;size:64"`
	ProofType     string    `json:"proof_type" gorm:"size:32"` // selective_disclosure / pqc / zk_ai
	ProverID      uint      `json:"prover_id" gorm:"index"`
	FieldSelector string    `json:"field_selector" gorm:"type:text"` // JSON: 选择的公开字段
	PublicData    string    `json:"public_data" gorm:"type:text"`    // 公开字段数据JSON
	ProofData     string    `json:"proof_data" gorm:"type:text"`     // 模拟ZKP证明
	BlockHash     string    `json:"block_hash" gorm:"size:128"`
	OnChain       bool      `json:"on_chain" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at"`
}

// ========== Scaling Architecture ==========

// RollupBatch ZK-Rollup批次
type RollupBatch struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	BatchNo    string     `json:"batch_no" gorm:"uniqueIndex;size:64"`
	FromBlock  uint64     `json:"from_block"`
	ToBlock    uint64     `json:"to_block"`
	TxCount    int        `json:"tx_count"`
	StateRoot  string     `json:"state_root" gorm:"size:128"`
	FaultProof string     `json:"fault_proof" gorm:"type:text"`
	Status     string     `json:"status" gorm:"size:32;default:'pending'"` // pending / verified / failed
	VerifiedBy string     `json:"verified_by" gorm:"size:64"`
	CreatedAt  time.Time  `json:"created_at"`
	VerifiedAt *time.Time `json:"verified_at"`
}

// DACommitment 模块化DA层承诺
type DACommitment struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	CommitmentNo string    `json:"commitment_no" gorm:"uniqueIndex;size:64"`
	DASource     string    `json:"da_source" gorm:"size:32"` // shared / private
	DataHash     string    `json:"data_hash" gorm:"size:128"`
	DataSize     int64     `json:"data_size"`
	BlobData     string    `json:"blob_data" gorm:"type:text"`
	Status       string    `json:"status" gorm:"size:32;default:'committed'"` // committed / verified
	CreatedAt    time.Time `json:"created_at"`
}

// ========== Domestic Sovereign Chain ==========

// PermissionPolicy 细粒度权限策略
type PermissionPolicy struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	PolicyNo    string    `json:"policy_no" gorm:"uniqueIndex;size:64"`
	TargetRole  string    `json:"target_role" gorm:"size:32"`
	Resource    string    `json:"resource" gorm:"size:64"`
	Action      string    `json:"action" gorm:"size:32"`      // read / write / admin
	Granularity string    `json:"granularity" gorm:"size:32"` // fine / coarse
	CreatedAt   time.Time `json:"created_at"`
}

// AnonymousIdentity 可控匿名身份
type AnonymousIdentity struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	RealUserID uint      `json:"real_user_id" gorm:"uniqueIndex"`
	AnonID     string    `json:"anon_id" gorm:"uniqueIndex;size:128"`
	PublicKey  string    `json:"public_key" gorm:"size:256"`
	Status     string    `json:"status" gorm:"size:16;default:'active'"`
	CreatedAt  time.Time `json:"created_at"`
}

// CrossChainReport 政务跨链上报记录
type CrossChainReport struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	ReportNo    string     `json:"report_no" gorm:"uniqueIndex;size:64"`
	FromChain   string     `json:"from_chain" gorm:"size:64"`
	ToChain     string     `json:"to_chain" gorm:"size:64"`
	DataPayload string     `json:"data_payload" gorm:"type:text"`
	Status      string     `json:"status" gorm:"size:32;default:'pending'"` // pending / sent / confirmed
	BlockHash   string     `json:"block_hash" gorm:"size:128"`
	OnChain     bool       `json:"on_chain" gorm:"default:false"`
	CreatedAt   time.Time  `json:"created_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
}

// ChainCanonical 返回CrossChainReport的链上规范数据
func (r CrossChainReport) ChainCanonical() interface{} {
	return map[string]interface{}{
		"report_no":  r.ReportNo,
		"from_chain": r.FromChain,
		"to_chain":   r.ToChain,
		"status":     r.Status,
		"payload":    r.DataPayload,
	}
}
