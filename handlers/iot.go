package handlers

import (
	"fmt"
	"time"

	"blockchain-demo/blockchain"
	"blockchain-demo/database"
	"blockchain-demo/models"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListIoTDevices 列出当前企业的IoT设备
func ListIoTDevices(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []models.IoTDevice
	if err := database.DB.Where("enterprise_id = ?", userID).
		Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询IoT设备失败")
		return
	}
	if list == nil {
		list = []models.IoTDevice{}
	}
	response.OK(c, gin.H{"list": list})
}

// RegisterIoTDevice 注册IoT设备
func RegisterIoTDevice(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		DeviceName string `json:"device_name" binding:"required"`
		DeviceType string `json:"device_type" binding:"required"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	device := models.IoTDevice{
		DeviceID:     fmt.Sprintf("IOT-%d-%d", userID, time.Now().Unix()),
		DeviceName:   req.DeviceName,
		EnterpriseID: userID,
		DeviceType:   req.DeviceType,
		Status:       "online",
		LastOnline:   time.Now(),
	}
	if err := database.DB.Create(&device).Error; err != nil {
		response.ServerError(c, "注册IoT设备失败")
		return
	}
	response.Created(c, "设备注册成功", device)
}

// CreateIoTRecord 模拟IoT自动采集记录
func CreateIoTRecord(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		DeviceID    string  `json:"device_id" binding:"required"`
		Electricity float64 `json:"electricity"`
		Gas         float64 `json:"gas"`
		Water       float64 `json:"water"`
	}
	if !response.BindJSON(c, &req) {
		return
	}
	recordNo := fmt.Sprintf("IOTR-%d-%d", userID, time.Now().UnixNano())
	record := models.IoTRecord{
		RecordNo:     recordNo,
		DeviceID:     req.DeviceID,
		EnterpriseID: userID,
		Source:       "auto",
		RiskLabel:    "",
		Electricity:  req.Electricity,
		Gas:          req.Gas,
		Water:        req.Water,
		CollectTime:  time.Now(),
	}
	tx := database.DB.Begin()
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建IoT采集记录失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeIoTRecord, recordNo, record.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "IoT记录上链失败: "+err.Error())
		return
	}
	tx.Model(&record).Update("block_hash", block.BlockHash)
	tx.Model(&record).Update("on_chain", true)
	tx.Commit()
	response.Created(c, "IoT自动采集记录已创建并上链", record)
}

// CreateManualRecord 手动录入IoT记录(带风险标签)
func CreateManualRecord(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		DeviceID    string  `json:"device_id"`
		Electricity float64 `json:"electricity"`
		Gas         float64 `json:"gas"`
		Water       float64 `json:"water"`
		RiskLabel   string  `json:"risk_label"` // 人工标记-低/中/高
	}
	if !response.BindJSON(c, &req) {
		return
	}
	if req.RiskLabel == "" {
		req.RiskLabel = "人工标记-低"
	}
	recordNo := fmt.Sprintf("IOTM-%d-%d", userID, time.Now().UnixNano())
	record := models.IoTRecord{
		RecordNo:     recordNo,
		DeviceID:     req.DeviceID,
		EnterpriseID: userID,
		Source:       "manual",
		RiskLabel:    req.RiskLabel,
		Electricity:  req.Electricity,
		Gas:          req.Gas,
		Water:        req.Water,
		CollectTime:  time.Now(),
	}
	tx := database.DB.Begin()
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "创建手动录入记录失败")
		return
	}
	block, err := blockchain.AddBlockTx(tx, models.DataTypeIoTRecord, recordNo, record.ChainCanonical())
	if err != nil {
		tx.Rollback()
		response.ServerError(c, "手动记录上链失败: "+err.Error())
		return
	}
	tx.Model(&record).Update("block_hash", block.BlockHash)
	tx.Model(&record).Update("on_chain", true)
	tx.Commit()
	response.Created(c, "手动录入记录已创建(风险标签:"+req.RiskLabel+")", record)
}

// ListIoTRecords 查看IoT记录列表
func ListIoTRecords(c *gin.Context) {
	userID := c.GetUint("user_id")
	var list []models.IoTRecord
	query := database.DB.Where("enterprise_id = ?", userID)
	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}
	if err := query.Order("id desc").Find(&list).Error; err != nil {
		response.ServerError(c, "查询IoT记录失败")
		return
	}
	if list == nil {
		list = []models.IoTRecord{}
	}
	response.OK(c, gin.H{"list": list})
}