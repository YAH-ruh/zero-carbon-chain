package handlers

// 能耗上报业务处理器
// 真实业务约束：企业能耗数据必须由前端页面手动录入提交，后端不自动生成/模拟能耗。
// 变更说明(v2)：能耗记录保存与模拟联盟链上链放于同一数据库事务(记录即上链)，
// 保证"业务落库"与"哈希存证"要么同时成功、要么同时回滚，杜绝半截数据。

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

// CreateEnergyRecord 手动创建能耗数据(上报即上链)
// POST /api/energy/create
// 权限模型：小微企业角色仅允许录入本企业数据(后端强制企业ID=登录用户)，
// 园区管理员/监管核证可在核验场景为指定企业代录(仍为手动录入)。
func CreateEnergyRecord(c *gin.Context) {
	var req models.EnergyCreateRequest
	if !response.BindJSON(c, &req) {
		return
	}

	// 确定归属企业并做越权拦截
	targetID := req.EnterpriseID
	if isEnterpriseOperator(c) {
		targetID = currentUserID(c)
		if req.EnterpriseID != 0 && req.EnterpriseID != targetID {
			response.Forbidden(c, "越权操作：企业角色只能录入本企业能耗数据")
			return
		}
	}
	if targetID == 0 {
		response.BadRequest(c, "缺少企业ID(enterprise_id)")
		return
	}

	// 业务参数校验：能耗各项不得为负且至少一项大于0
	if req.Electricity < 0 || req.Gas < 0 || req.Water < 0 ||
		(req.Electricity == 0 && req.Gas == 0 && req.Water == 0) {
		response.BadRequest(c, "能耗数据需为正数(电量/气量/水量至少一项>0)")
		return
	}

	// 校验企业用户存在且为企业角色
	var enterprise models.User
	if err := database.DB.First(&enterprise, targetID).Error; err != nil {
		response.NotFound(c, "企业用户不存在")
		return
	}
	if enterprise.Role != models.RoleEnterprise {
		response.BadRequest(c, "目标用户不是小微企业账号，无法归属能耗数据")
		return
	}

	record := &models.EnergyRecord{
		RecordNo:     fmt.Sprintf("ENERGY-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()/1e6),
		EnterpriseID: targetID,
		DeviceID:     fmt.Sprintf("MANUAL-%d", targetID),
		Electricity:  req.Electricity,
		Gas:          req.Gas,
		Water:        req.Water,
		CollectTime:  time.Now(),
	}

	// 事务：能耗入库 + 真实SHA-256上链 + 业务上链状态回写 + 操作审计
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return fmt.Errorf("能耗记录保存失败: %w", err)
		}
		block, err := blockchain.AddBlockTx(tx, models.DataTypeEnergy, record.RecordNo, record)
		if err != nil {
			return err
		}
		record.BlockHash = block.BlockHash
		record.OnChain = true
		if err := tx.Model(record).Updates(map[string]interface{}{
			"block_hash": block.BlockHash,
			"on_chain":   true,
		}).Error; err != nil {
			return err
		}
		return logOperation(tx, c, models.OpEnergyCreate, models.DataTypeEnergy, record.RecordNo, block.BlockHash,
			fmt.Sprintf("能耗上报: 电%.1fkWh 气%.1fm³ 水%.1ft", record.Electricity, record.Gas, record.Water))
	})
	if err != nil {
		logger.Error("能耗上报失败: %v", err)
		response.ServerError(c, "能耗数据提交失败: "+err.Error())
		return
	}

	response.OKMsg(c, "能耗数据提交成功，已计算SHA-256并上链存证", record)
}

// ListEnergyRecords 查询能耗记录列表
// GET /api/energy/list?enterprise_id=1&page=1&page_size=20
// 数据权限：企业角色强制只返回本企业记录；园区管理员/监管核证可按 enterprise_id 查询任意企业。
func ListEnergyRecords(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)

	query := database.DB.Model(&models.EnergyRecord{})
	if isEnterpriseOperator(c) {
		// 越权防护：企业登录后不可读取他人能耗数据
		query = query.Where("enterprise_id = ?", currentUserID(c))
	} else if id := parseUint(c.Query("enterprise_id")); id > 0 {
		query = query.Where("enterprise_id = ?", id)
	}

	var total int64
	query.Count(&total)

	var records []models.EnergyRecord
	if err := query.Preload("Enterprise").
		Order("id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&records).Error; err != nil {
		response.ServerError(c, "查询能耗记录失败")
		return
	}

	response.OK(c, gin.H{
		"list":     records,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
