package handlers

// 本文件为 handlers 公共工具函数：
// 操作人信息提取、企业角色数据越权拦截、关键操作审计日志(OperationLog)落库、
// AI 报告上链复用、业务实体装载等，减少各业务处理器重复代码。

import (
	"fmt"
	"time"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------- 操作人信息(从 JWT 中间件写入的上下文获取) ----------

func currentUserID(c *gin.Context) uint { return c.GetUint("user_id") }

func currentRole(c *gin.Context) string { return c.GetString("role") }

func currentUserName(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return ""
}

func isEnterpriseOperator(c *gin.Context) bool {
	return currentRole(c) == models.RoleEnterprise
}

// permitOwnData 企业角色数据越权拦截
// 安全机制：小微企业在后端被强制只能访问/操作本企业(ownerID)数据，
// 即便前端伪造 enterprise_id 参数也会被后端拦截，实现前后端双重权限隔离。
func permitOwnData(c *gin.Context, ownerID uint) bool {
	if isEnterpriseOperator(c) && currentUserID(c) != ownerID {
		response.Forbidden(c, "越权访问：企业角色仅能操作本企业数据")
		return false
	}
	return true
}

// logOperation 关键操作审计日志(落库 OperationLog + 分级文件日志双写)
// db 可为当前事务句柄(与业务同事务提交，失败则随业务一并回滚)；为 nil 时使用全局连接。
// 返回 error：供事务内调用方直接 return，保证"审计失败 → 业务整体回滚"。
func logOperation(db *gorm.DB, c *gin.Context, operation, dataType, dataID, blockHash, detail string) error {
	if db == nil {
		db = database.DB
	}
	entry := &models.OperationLog{
		Operation:    operation,
		OperatorID:   currentUserID(c),
		OperatorName: currentUserName(c),
		Role:         currentRole(c),
		DataType:     dataType,
		DataID:       dataID,
		BlockHash:    blockHash,
		Detail:       detail,
		IP:           c.ClientIP(),
	}
	if err := db.Create(entry).Error; err != nil {
		logger.Error("操作审计日志落库失败 op=%s err=%v", operation, err)
		return err
	}
	logger.Info("[操作审计] op=%s operator=%s(id=%d,role=%s) biz=%s/%s hash=%s detail=%s ip=%s",
		operation, entry.OperatorName, entry.OperatorID, entry.Role, dataType, dataID, blockHash, detail, entry.IP)
	return nil
}

// ---------- 业务实体装载(链上存证/篡改校验共用) ----------

// loadBusinessData 按业务类型与单号回读业务实体(返回实体指针)，错误信息可直接返回前端。
func loadBusinessData(dataType, dataID string) (interface{}, error) {
	switch dataType {
	case models.DataTypeEnergy:
		var v models.EnergyRecord
		if err := database.DB.Where("record_no = ?", dataID).First(&v).Error; err != nil {
			return nil, fmt.Errorf("能耗记录未找到(单号: %s)", dataID)
		}
		return &v, nil
	case models.DataTypeCredit:
		var v models.CarbonCredit
		if err := database.DB.Where("credit_no = ?", dataID).First(&v).Error; err != nil {
			return nil, fmt.Errorf("碳积分记录未找到(单号: %s)", dataID)
		}
		return &v, nil
	case models.DataTypeTransaction:
		var v models.Transaction
		if err := database.DB.Where("tx_no = ?", dataID).First(&v).Error; err != nil {
			return nil, fmt.Errorf("交易凭证未找到(单号: %s)", dataID)
		}
		return &v, nil
	case models.DataTypeReport:
		var v models.ReportRecord
		if err := database.DB.Where("report_no = ?", dataID).First(&v).Error; err != nil {
			return nil, fmt.Errorf("报告记录未找到(单号: %s)", dataID)
		}
		return &v, nil
	default:
		return nil, fmt.Errorf("不支持的业务类型: %s", dataType)
	}
}

// markOnChainStatus 更新业务表上链状态与区块哈希
// 调用链完整性：/chain/upload 手动补链与存量数据处理场景使用。
func markOnChainStatus(db *gorm.DB, dataType, dataID, blockHash string) error {
	fields := map[string]interface{}{"block_hash": blockHash, "on_chain": true}
	switch dataType {
	case models.DataTypeEnergy:
		return db.Model(&models.EnergyRecord{}).Where("record_no = ?", dataID).Updates(fields).Error
	case models.DataTypeCredit:
		return db.Model(&models.CarbonCredit{}).Where("credit_no = ?", dataID).Updates(fields).Error
	case models.DataTypeTransaction:
		return db.Model(&models.Transaction{}).Where("tx_no = ?", dataID).Updates(fields).Error
	case models.DataTypeReport:
		return db.Model(&models.ReportRecord{}).Where("report_no = ?", dataID).Updates(fields).Error
	}
	return nil
}

// ---------- AI 报告上链复用 ----------

// genReportNo 生成报告编号 REPORT-日期-微秒，保证唯一
func genReportNo() string {
	return fmt.Sprintf("REPORT-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()/1000)
}

// createReportInTx 在事务内创建 AI 报告记录并上链(记录即上链)
// 返回报告记录与区块哈希，供调用方继续写审计日志并提交事务。
func createReportInTx(tx *gorm.DB, c *gin.Context, kind, title, bizRefNo, content, source string) (*models.ReportRecord, string, error) {
	report := &models.ReportRecord{
		ReportNo:  genReportNo(),
		Kind:      kind,
		CreatorID: currentUserID(c),
		Title:     title,
		BizRefNo:  bizRefNo,
		Content:   content,
		Source:    source,
	}
	if err := tx.Create(report).Error; err != nil {
		return nil, "", fmt.Errorf("报告记录保存失败: %w", err)
	}
	// 业务数据(报告内容)上链
	block, err := blockchain.AddBlockTx(tx, models.DataTypeReport, report.ReportNo, report)
	if err != nil {
		return nil, "", fmt.Errorf("报告上链失败: %w", err)
	}
	report.BlockHash = block.BlockHash
	report.OnChain = true
	if err := tx.Model(report).Updates(map[string]interface{}{
		"block_hash": block.BlockHash,
		"on_chain":   true,
	}).Error; err != nil {
		return nil, "", err
	}
	return report, block.BlockHash, nil
}

// saveReportAndLog 开启事务保存 AI 报告→上链→写审计日志(供 AI 报告类处理器复用)
// 失败时已统一返回错误响应，调用方直接 return。
func saveReportAndLog(c *gin.Context, kind, title, bizRefNo, content, source string) *models.ReportRecord {
	var report *models.ReportRecord
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		r, hash, err := createReportInTx(tx, c, kind, title, bizRefNo, content, source)
		if err != nil {
			return err
		}
		report = r
		return logOperation(tx, c, models.OpReportGenerate, models.DataTypeReport, r.ReportNo, hash,
			"AI报告生成: "+title+" (source="+source+")")
	})
	if err != nil {
		logger.Error("AI 报告保存上链失败: %v", err)
		response.ServerError(c, "报告保存上链失败: "+err.Error())
		return nil
	}
	return report
}

// ---------- 分页参数解析 ----------

// parseInt 字符串转正整数(非法/非正数回退默认值)
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		val = val*10 + int(ch-'0')
	}
	if val <= 0 {
		return defaultVal
	}
	return val
}

// parseUint 字符串转无符号整数(非法返回0)
func parseUint(s string) uint {
	var val uint
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		val = val*10 + uint(ch-'0')
	}
	return val
}
