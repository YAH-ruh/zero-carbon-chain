// Package blockchain 本地模拟联盟链模块(服务端实现)
//
// 变更说明(v2)：整体替换原 mock_chain.go 的内存实现，重点修复与增强——
//  1. 创世区块持久化落库：重启后从数据库恢复完整哈希链，不再出现"创世块不落库导致链断裂"；
//  2. 区块结构增强：加入 Merkle 根(MerkleRoot)，并纳入区块哈希计算；
//  3. 业务哈希规范化：对业务模型取"核心业务要素视图"(ChainCanonical)后计算真实 SHA-256，
//     避免上链状态/权属流转等过程字段干扰哈希可比性；
//  4. 链上操作支持"加入调用方事务"(AddBlockTx)：业务落库与上链同一事务，
//     保证数据一致性；重复同一业务单号自动幂等，不产生脏区块；
//  5. 账本以数据库为唯一事实源：所有读取(溯源/列表/审计)均直查数据库，
//     天然支持重启恢复与多请求一致性，无需维护内存镜像；
//  6. 新增整链完整性审计(AuditChainIntegrity)与启动自检，支撑监管"一键校验篡改"。
//
// 说明：该模块模拟的是"联盟链背书-记账"效果(哈希链 + 持久化账本)，并未部署真实联盟链节点，
// 生产环境可按相同数据接口替换为 FISCO-BCOS 等联盟链 SDK(代码注释即对接文档)。
package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"

	"gorm.io/gorm"
)

// SimChain 模拟联盟链管理对象
// 链数据以 models.BlockRecord 形式持久化到数据库(SQLite，可平滑切换 MySQL)。
// 单节点记账模型：并发写由 SQLite 单写连接(池=1)天然串行化，区块高度单调递增；
// 读取接口每次直查数据库，保证与已提交账本严格一致。
type SimChain struct {
	db *gorm.DB
}

// Chain 包级单例：handler/service 统一通过 blockchain.Chain 访问链能力。
// 命名说明：包内类型为 SimChain，变量名为 Chain，二者不冲突，属 Go 常见"类型+单例"风格。
var (
	Chain *SimChain
	once  sync.Once
)

// InitChain 初始化模拟联盟链(进程内单例)
// 启动自愈流程：加载库内区块 → 若为空则创建并落库创世区块 → 自检链式连续性。
func InitChain(db *gorm.DB) *SimChain {
	once.Do(func() {
		Chain = &SimChain{db: db}
		if err := Chain.load(); err != nil {
			logger.Error("模拟联盟链初始化失败: %v", err)
		}
	})
	return Chain
}

// AddBlockTx 包级便捷入口：在调用方事务内追加新区块(等价于 Chain.AddBlockTx)。
// 供 handler 在业务事务中直接调用，简化"链未初始化"判断。
func AddBlockTx(tx *gorm.DB, dataType, dataID string, data interface{}) (*models.BlockRecord, error) {
	if Chain == nil {
		return nil, errors.New("模拟联盟链尚未初始化")
	}
	return Chain.AddBlockTx(tx, dataType, dataID, data)
}

// load 从数据库恢复整条链并做链式自检；首次运行则创建创世区块并持久化(重复启动不会重复创建)。
func (c *SimChain) load() error {
	var records []models.BlockRecord
	if err := c.db.Order("block_index asc").Find(&records).Error; err != nil {
		return fmt.Errorf("读取链上区块失败: %w", err)
	}

	if len(records) == 0 {
		genesis := c.buildGenesis()
		if err := c.db.Create(genesis).Error; err != nil {
			return fmt.Errorf("创世区块落库失败: %w", err)
		}
		records = append(records, *genesis)
		logger.Info("⛓ 创世区块创建并落库 height=0 hash=%s", genesis.BlockHash)
	}

	// 启动自检：发现链式断裂(前块哈希不连续)立即告警，便于演示前定位脏数据
	broken := 0
	for i := 1; i < len(records); i++ {
		if records[i].PrevHash != records[i-1].BlockHash {
			broken++
		}
	}
	if broken > 0 {
		logger.Warn("启动自检：检测到 %d 处链式断裂，请调用链完整性审计接口定位问题区块", broken)
	} else {
		logger.Info("⛓ 模拟联盟链恢复完成，区块总数=%d 最新高度=%d",
			len(records), records[len(records)-1].BlockIndex)
	}
	return nil
}

// buildGenesis 构造创世区块(高度 0，无前块哈希)
func (c *SimChain) buildGenesis() *models.BlockRecord {
	dataHash := sha256Hex([]byte("micro-carbon-chain-genesis-v2"))
	blk := Block{
		Index:      0,
		Timestamp:  time.Now().Unix(),
		DataHash:   dataHash,
		MerkleRoot: calcMerkleRootForData(dataHash),
		DataType:   models.DataTypeGenesis,
		DataID:     "GENESIS",
	}
	blk.Hash = blk.computeHash()
	return toRecord(blk)
}

// toRecord 将内存区块结构转为数据库记录结构
func toRecord(b Block) *models.BlockRecord {
	return &models.BlockRecord{
		BlockIndex: b.Index,
		BlockHash:  b.Hash,
		PrevHash:   b.PrevHash,
		DataType:   b.DataType,
		DataID:     b.DataID,
		DataHash:   b.DataHash,
		MerkleRoot: b.MerkleRoot,
		Timestamp:  b.Timestamp,
	}
}

// toBlock 将数据库记录结构转回内存区块结构(前端列表输出用)
func toBlock(r *models.BlockRecord) Block {
	return Block{
		Index:      r.BlockIndex,
		Timestamp:  r.Timestamp,
		DataHash:   r.DataHash,
		MerkleRoot: r.MerkleRoot,
		DataType:   r.DataType,
		DataID:     r.DataID,
		PrevHash:   r.PrevHash,
		Hash:       r.BlockHash,
	}
}

// BusinessHash 计算任意业务对象的规范化业务哈希(真实 SHA-256)
// 规则：优先取模型的 ChainCanonical() 核心要素视图；不具备该方法的对象直接 JSON 序列化。
func BusinessHash(v interface{}) (string, error) {
	if v == nil {
		return "", errors.New("业务数据为空，无法计算哈希")
	}
	var canonical interface{} = v
	if cd, ok := v.(interface{ ChainCanonical() interface{} }); ok {
		canonical = cd.ChainCanonical()
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("业务数据序列化失败: %w", err)
	}
	return sha256Hex(raw), nil
}

// AddBlockTx 在调用方事务内追加新区块(核心上链方法)
// 与业务写操作处于同一数据库事务：业务成功上链才提交，上链失败则整体回滚。
// 幂等保证：同一 dataType+dataID 已存在区块时直接返回已有区块，避免重复上链。
// data: 业务对象，其 ChainCanonical() 或 JSON 序列化结果参与业务哈希计算。
func (c *SimChain) AddBlockTx(tx *gorm.DB, dataType, dataID string, data interface{}) (*models.BlockRecord, error) {
	dataHash, err := BusinessHash(data)
	if err != nil {
		return nil, err
	}

	// 幂等检查：业务单号唯一，重复调用返回已上链区块
	var existed models.BlockRecord
	if err := tx.Where("data_type = ? AND data_id = ?", dataType, dataID).First(&existed).Error; err == nil {
		logger.Info("上链幂等命中 data=%s/%s hash=%s", dataType, dataID, existed.BlockHash)
		return &existed, nil
	}

	// 取当前链尾以确定高度与前块哈希(始终以事务内最新账本为准)
	var last models.BlockRecord
	nextIndex := uint64(0)
	prevHash := ""
	if err := tx.Order("block_index desc").First(&last).Error; err == nil {
		nextIndex = last.BlockIndex + 1
		prevHash = last.BlockHash
	}

	blk := Block{
		Index:      nextIndex,
		Timestamp:  time.Now().Unix(),
		DataHash:   dataHash,
		MerkleRoot: calcMerkleRootForData(dataHash),
		DataType:   dataType,
		DataID:     dataID,
		PrevHash:   prevHash,
	}
	blk.Hash = blk.computeHash()

	record := toRecord(blk)
	if err := tx.Create(record).Error; err != nil {
		return nil, fmt.Errorf("区块写入账本失败: %w", err)
	}
	logger.Info("⛓ 上链成功 height=%d type=%s data=%s hash=%s merkle=%s",
		record.BlockIndex, dataType, dataID, record.BlockHash, record.MerkleRoot)
	return record, nil
}

// AddBlock 独立事务上链(供无主事务参与的调用，如手动补录上链)
func (c *SimChain) AddBlock(dataType, dataID string, data interface{}) (*models.BlockRecord, error) {
	var record *models.BlockRecord
	err := c.db.Transaction(func(tx *gorm.DB) error {
		var e error
		record, e = c.AddBlockTx(tx, dataType, dataID, data)
		return e
	})
	return record, err
}

// QueryByDataNo 按业务单号溯源查询(返回区块与哈希信息)
// 直查数据库账本，保证返回结果与已提交区块一致。
func (c *SimChain) QueryByDataNo(dataType, dataID string) *models.ChainVerifyResult {
	var found models.BlockRecord
	q := c.db.Where("data_id = ?", dataID)
	if dataType != "" {
		q = q.Where("data_type = ?", dataType)
	}
	if err := q.Order("block_index desc").First(&found).Error; err != nil {
		return &models.ChainVerifyResult{Exists: false, DataNo: dataID}
	}
	// 重算区块哈希验证链上区块自身未被篡改
	blk := toBlock(&found)
	return &models.ChainVerifyResult{
		Exists:       true,
		BlockIndex:   found.BlockIndex,
		BlockHash:    found.BlockHash,
		PrevHash:     found.PrevHash,
		MerkleRoot:   found.MerkleRoot,
		DataNo:       found.DataID,
		DataType:     found.DataType,
		OriginalHash: found.DataHash,
		CurrentHash:  found.DataHash,
		Match:        blk.computeHash() == found.BlockHash,
		Timestamp:    found.Timestamp,
	}
}

// VerifyDataIntegrity 篡改校验：以"当前业务数据"重新计算业务哈希并与链上原始哈希比对
func (c *SimChain) VerifyDataIntegrity(dataType, dataID string, currentData interface{}) *models.ChainVerifyResult {
	var found models.BlockRecord
	if err := c.db.Where("data_type = ? AND data_id = ?", dataType, dataID).First(&found).Error; err != nil {
		return &models.ChainVerifyResult{Exists: false, DataNo: dataID, Match: false}
	}
	currentHash, err := BusinessHash(currentData)
	if err != nil {
		currentHash = ""
	}
	blk := toBlock(&found)
	return &models.ChainVerifyResult{
		Exists:       true,
		BlockIndex:   found.BlockIndex,
		BlockHash:    found.BlockHash,
		PrevHash:     found.PrevHash,
		MerkleRoot:   found.MerkleRoot,
		DataNo:       found.DataID,
		DataType:     found.DataType,
		OriginalHash: found.DataHash,
		CurrentHash:  currentHash,
		Match:        currentHash != "" && currentHash == found.DataHash && blk.computeHash() == found.BlockHash,
		Timestamp:    found.Timestamp,
	}
}

// GetChainInfo 获取区块链概览信息(前端首页/监控展示)
func (c *SimChain) GetChainInfo() map[string]interface{} {
	var count int64
	c.db.Model(&models.BlockRecord{}).Count(&count)

	genesisHash := ""
	latestIndex := uint64(0)
	latestHash := ""
	merkleRoot := ""
	var genesis, last models.BlockRecord
	if err := c.db.Where("block_index = ?", 0).First(&genesis).Error; err == nil {
		genesisHash = genesis.BlockHash
	}
	if err := c.db.Order("block_index desc").First(&last).Error; err == nil {
		latestIndex = last.BlockIndex
		latestHash = last.BlockHash
		merkleRoot = last.MerkleRoot
	}
	return map[string]interface{}{
		"block_count":        count,
		"latest_block_index": latestIndex,
		"latest_block_hash":  latestHash,
		"latest_merkle_root": merkleRoot,
		"genesis_block_hash": genesisHash,
		"chain_type":         "本地模拟联盟链(持久化账本)",
		"consensus":          "单节点记账 + 哈希链校验(模拟联盟链背书)",
		"description":        "服务端模拟联盟链：区块持久化至数据库，重启不丢失；可一键校验整链完整性。生产可替换 FISCO-BCOS 等真实联盟链。",
	}
}

// GetLatestBlock 返回链尾区块信息(前端挂单/交易成功后展示用)
func (c *SimChain) GetLatestBlock() *models.BlockRecord {
	var last models.BlockRecord
	if err := c.db.Order("block_index desc").First(&last).Error; err != nil {
		return nil
	}
	return &last
}

// GetAllBlocks 获取区块列表(分页直查数据库)，输出结构与前端兼容
func (c *SimChain) GetAllBlocks(page, pageSize int) ([]Block, int) {
	var total int64
	c.db.Model(&models.BlockRecord{}).Count(&total)

	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	var records []models.BlockRecord
	if err := c.db.Order("block_index asc").
		Offset(start).Limit(pageSize).Find(&records).Error; err != nil {
		return []Block{}, int(total)
	}

	out := make([]Block, 0, len(records))
	for i := range records {
		out = append(out, toBlock(&records[i]))
	}
	return out, int(total)
}

// AuditChainIntegrity 链完整性审计(监管"一键校验篡改")
// 直读库内全部区块：逐块重算区块哈希(校验哈希自洽) + 校验前块链式关系 + 回读业务数据
// 重算业务哈希(识别业务篡改)。返回逐块明细，便于前端红绿标记问题区块。
func (c *SimChain) AuditChainIntegrity() *models.ChainAuditReport {
	var records []models.BlockRecord
	c.db.Order("block_index asc").Find(&records)

	report := &models.ChainAuditReport{
		Items: make([]models.ChainAuditItem, 0, len(records)),
	}
	broken := 0

	for i := range records {
		r := &records[i]
		blk := toBlock(r)
		hashValid := blk.computeHash() == r.BlockHash

		linkValid := true
		if i > 0 {
			linkValid = r.PrevHash == records[i-1].BlockHash
		}

		item := models.ChainAuditItem{
			BlockIndex: r.BlockIndex,
			DataType:   r.DataType,
			DataID:     r.DataID,
			HashValid:  hashValid,
			LinkValid:  linkValid,
		}

		// 业务数据回读比对(创世区块无业务实体，跳过)
		if r.DataType != models.DataTypeGenesis {
			obj, ok := c.fetchBusiness(r.DataType, r.DataID)
			if !ok {
				// 业务实体已不存在：无法比对业务哈希，标红提示(可能是被整体删除或误清理)
				no := false
				item.DataValid = &no
				item.Issue = "链上存在该业务单号但库内业务数据缺失"
			} else {
				cur, err := BusinessHash(obj)
				valid := err == nil && cur == r.DataHash
				item.DataValid = &valid
				if !valid {
					item.Issue = "库内业务数据哈希与链上原始哈希不一致，疑似被篡改"
				}
			}
		} else {
			item.DataValid = nil
		}

		if item.Issue == "" && !(hashValid && linkValid) {
			if !hashValid {
				item.Issue = "区块自身哈希校验失败(账本字段被改动)"
			} else if !linkValid {
				item.Issue = "与前区块哈希不连续，链式关系被破坏"
			}
		}
		if item.Issue != "" {
			broken++
		}
		report.Items = append(report.Items, item)
	}

	report.Total = len(records)
	report.Valid = len(records) - broken
	report.Broken = broken
	report.Consistent = broken == 0
	if len(records) > 0 {
		report.FromHeight = records[0].BlockIndex
		report.ToHeight = records[len(records)-1].BlockIndex
		report.LatestHash = records[len(records)-1].BlockHash
	}
	return report
}

// fetchBusiness 按业务类型与单号回读库内业务实体(用于审计比对)
func (c *SimChain) fetchBusiness(dataType, dataID string) (interface{}, bool) {
	switch dataType {
	case models.DataTypeEnergy:
		var v models.EnergyRecord
		if err := c.db.Where("record_no = ?", dataID).First(&v).Error; err == nil {
			return v, true
		}
	case models.DataTypeCredit:
		var v models.CarbonCredit
		if err := c.db.Where("credit_no = ?", dataID).First(&v).Error; err == nil {
			return v, true
		}
	case models.DataTypeTransaction:
		var v models.Transaction
		if err := c.db.Where("tx_no = ?", dataID).First(&v).Error; err == nil {
			return v, true
		}
	case models.DataTypeReport:
		var v models.ReportRecord
		if err := c.db.Where("report_no = ?", dataID).First(&v).Error; err == nil {
			return v, true
		}
	}
	return nil, false
}
