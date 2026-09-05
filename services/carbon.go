// Package services 提供业务逻辑服务
// 碳积分业务：核算、统计与交易撮合。
package services

import (
	"errors"
	"fmt"
	"time"

	"blockchain-demo/database"
	"blockchain-demo/models"

	"gorm.io/gorm"
)

// 碳排放因子 (kgCO₂/单位)
// 真实业务：依据国家发改委发布的区域电网基准线排放因子等标准。
const (
	EmissionFactorElectricity = 0.785 // 每kWh电力碳排放因子 (kgCO₂/kWh)
	EmissionFactorGas         = 2.165 // 每m³天然气碳排放因子 (kgCO₂/m³)
	EmissionFactorWater       = 0.298 // 每吨水碳排放因子 (kgCO₂/t)
	CreditPerKgCO2            = 1.0   // 每kgCO₂减排量对应1碳积分
)

// CalculateCarbonCredits 根据能耗数据核算碳积分
// 核算规则：能耗 × 碳排放因子 = 碳排放量(kgCO₂) → 按 CreditPerKgCO2 折算碳积分。
// 变更说明(v2)：核算编号由"能耗记录ID"派生，天然唯一，避免随机碰撞。
func CalculateCarbonCredits(energyRecord *models.EnergyRecord) *models.CarbonCredit {
	totalEmission := energyRecord.Electricity*EmissionFactorElectricity +
		energyRecord.Gas*EmissionFactorGas +
		energyRecord.Water*EmissionFactorWater
	credits := totalEmission * CreditPerKgCO2

	creditNo := fmt.Sprintf("CREDIT-%s-%04d", time.Now().Format("20060102"), energyRecord.ID)
	return &models.CarbonCredit{
		CreditNo:       creditNo,
		EnterpriseID:   energyRecord.EnterpriseID,
		EnergyRecordID: energyRecord.ID,
		TotalEmission:  totalEmission,
		CarbonCredits:  credits,
		Status:         "available",
		OwnerID:        energyRecord.EnterpriseID,
	}
}

// GetEnterpriseCarbonStats 获取企业碳数据统计
func GetEnterpriseCarbonStats(enterpriseID uint) map[string]interface{} {
	var totalEmission, totalCredits, soldCredits, availableCredits float64
	var creditCount int64

	aggr := databaseOf().Model(&models.CarbonCredit{}).Where("enterprise_id = ?", enterpriseID)
	aggr.Select("COALESCE(SUM(total_emission), 0)").Scan(&totalEmission)
	aggr.Select("COALESCE(SUM(carbon_credits), 0)").Scan(&totalCredits)
	aggr.Count(&creditCount)
	databaseOf().Model(&models.CarbonCredit{}).
		Where("enterprise_id = ? AND status = ?", enterpriseID, "sold").
		Select("COALESCE(SUM(carbon_credits), 0)").Scan(&soldCredits)
	databaseOf().Model(&models.CarbonCredit{}).
		Where("enterprise_id = ? AND status = ?", enterpriseID, "available").
		Select("COALESCE(SUM(carbon_credits), 0)").Scan(&availableCredits)

	return map[string]interface{}{
		"enterprise_id":     enterpriseID,
		"total_emission":    totalEmission,
		"total_credits":     totalCredits,
		"credit_count":      creditCount,
		"sold_credits":      soldCredits,
		"available_credits": availableCredits,
	}
}

// GetParkCarbonStats 获取园区碳数据统计(园区管理员/园区报告用)
func GetParkCarbonStats(parkID uint) map[string]interface{} {
	var enterprises []models.User
	databaseOf().Where("park_id = ? AND role = ?", parkID, models.RoleEnterprise).Find(&enterprises)

	var totalEmission, totalCredits float64
	for _, ent := range enterprises {
		var e, c float64
		databaseOf().Model(&models.CarbonCredit{}).
			Where("enterprise_id = ?", ent.ID).
			Select("COALESCE(SUM(total_emission), 0)").Scan(&e)
		databaseOf().Model(&models.CarbonCredit{}).
			Where("enterprise_id = ?", ent.ID).
			Select("COALESCE(SUM(carbon_credits), 0)").Scan(&c)
		totalEmission += e
		totalCredits += c
	}

	return map[string]interface{}{
		"park_id":          parkID,
		"enterprise_count": len(enterprises),
		"total_emission":   totalEmission,
		"total_credits":    totalCredits,
		"enterprises":      enterprises,
	}
}

// databaseOf 返回全局数据库实例
// 说明：本服务方法统一走全局 database.DB，便于 handler 层复用统计能力。
func databaseOf() *gorm.DB {
	return database.DB
}

// MatchAndTransfer 执行交易撮合(必须在调用方开启的事务内执行)
//
// 事务内顺序：核验挂单 → 条件更新挂单为已成交(防并发重复撮合) → 碳积分权属转移 →
// 生成交易凭证。任一步失败整体回滚，保证"挂单-积分-交易"三方一致。
//
// 变更说明(v2)：原 ExecuteMatch 将"上链更新"放在事务外(状态已提交后单独写哈希)，
// 存在中途失败导致业务已变但未上链的不一致；现由调用方在事务内完成上链后再提交。
func MatchAndTransfer(tx *gorm.DB, orderID, buyerID uint) (*models.Transaction, error) {
	// 1. 加载挂单并校验状态
	var order models.SellOrder
	if err := tx.First(&order, orderID).Error; err != nil {
		return nil, errors.New("挂单不存在")
	}
	if order.Status != "pending" {
		return nil, fmt.Errorf("挂单状态已变化(status=%s)，无法重复撮合", order.Status)
	}
	if order.EnterpriseID == buyerID {
		return nil, errors.New("买卖双方不能为同一企业")
	}

	// 2. 条件更新挂单(仅当仍为 pending 才生效)，避免并发重复成交
	res := tx.Model(&models.SellOrder{}).
		Where("id = ? AND status = ?", orderID, "pending").
		Updates(map[string]interface{}{"status": "matched", "buyer_id": buyerID})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errors.New("挂单已被处理，请刷新后重试")
	}

	// 3. 校验并转移碳积分权属(仅锁定中的积分允许成交，杜绝一积分多卖)
	var credit models.CarbonCredit
	if err := tx.First(&credit, order.CreditID).Error; err != nil {
		return nil, errors.New("关联碳积分不存在")
	}
	if credit.Status != "locked" {
		return nil, fmt.Errorf("碳积分状态异常(%s)，无法成交", credit.Status)
	}
	updCredit := tx.Model(&models.CarbonCredit{}).
		Where("id = ? AND status = ?", credit.ID, "locked").
		Updates(map[string]interface{}{"status": "sold", "owner_id": buyerID})
	if updCredit.Error != nil {
		return nil, updCredit.Error
	}
	if updCredit.RowsAffected == 0 {
		return nil, errors.New("碳积分状态已变化，无法成交")
	}

	// 4. 生成交易凭证(编号由挂单ID派生，天然唯一)
	txNo := fmt.Sprintf("TX-%s-%04d", time.Now().Format("20060102"), order.ID)
	transaction := &models.Transaction{
		TxNo:        txNo,
		OrderNo:     order.OrderNo,
		SellerID:    order.EnterpriseID,
		BuyerID:     buyerID,
		CreditID:    credit.ID,
		Quantity:    order.Quantity,
		UnitPrice:   order.UnitPrice,
		TotalAmount: order.Quantity * order.UnitPrice,
	}
	if err := tx.Create(transaction).Error; err != nil {
		return nil, fmt.Errorf("生成交易凭证失败: %w", err)
	}
	return transaction, nil
}
