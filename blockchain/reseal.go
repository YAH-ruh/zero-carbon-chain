package blockchain

// 时间线重封存(Reseal) —— 支撑"演示数据时间线整体平移到 2026年9月以后"的需求：
//
//	业务表的展示时间(上报/核算/挂单/成交/存证等)平移后，两类链上数据需要同步：
//	 1) 区块时间戳(Timestamp)参与区块哈希计算，直接 UPDATE 会破坏链完整性校验；
//	 2) 能耗/IoT 记录的业务哈希(ChainCanonical)包含 collect_time，交易/报告的全量
//	    JSON 序列化包含 CreatedAt，时间平移会导致业务哈希漂移。
//
// 本模块按 block_index 顺序整链重封存：
//	- 平移时间戳早于基准时间的区块(保持链内相对顺序)；
//	- 按"当前业务数据"重算漂移类型的 DataHash(其余类型保留原哈希)；
//	- 逐块重算 Merkle 根与区块哈希(prev_hash 链式传导)，保证
//	  computeHash == BlockHash 的链完整性校验与 VerifyDataIntegrity 篡改校验全部通过；
//	- 回写业务行 block_hash，保证页面展示与链上一致。
//	幂等：全部一致时不产生任何更新，可重复执行。

import (
	"fmt"

	"gorm.io/gorm"

	"blockchain-demo/models"
)

// resealBusinessHashTypes 需要"按当前业务数据重算 DataHash"的类型：
//   - energy/iot_record：ChainCanonical 包含 collect_time 与 record_no，时间平移+单号改名均导致漂移；
//   - credit/pledge/arbitration/archive/agent/footprint：ChainCanonical 包含业务单号，
//     单号日期段改名(202605~202608→202609)导致漂移，按 ChainCanonical 重算可精确复现；
//   - transaction/report/zk_proof：上链快照含过程字段(伪哈希/证明体)，无法精确复现，
//     且前端无针对它们的篡改校验入口，保留原 DataHash(块自身哈希链由重封存保证完整)。
func resealBusinessHash(db *gorm.DB, dataType, dataID string) (string, bool) {
	switch dataType {
	case models.DataTypeEnergy:
		var r models.EnergyRecord
		if err := db.Where("record_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	case models.DataTypeIoTRecord:
		var r models.IoTRecord
		if err := db.Where("record_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	case models.DataTypeCredit:
		var r models.CarbonCredit
		if err := db.Where("credit_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	case models.DataTypePledge:
		var r models.PledgeOrder
		if err := db.Where("pledge_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	case models.DataTypeArbitration:
		var r models.ArbitrationCase
		if err := db.Where("case_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	case models.DataTypeArchive:
		var r models.CarbonArchive
		if err := db.Where("archive_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	case models.DataTypeAgent:
		var r models.AgentRecord
		if err := db.Where("agent_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	case models.DataTypeFootprint:
		var r models.ProductFootprint
		if err := db.Where("product_no = ?", dataID).First(&r).Error; err == nil {
			if h, err := BusinessHash(r); err == nil {
				return h, true
			}
		}
	}
	return "", false
}

// resealWriteBack 将重封存后的区块哈希回写到业务行(页面展示与链上一致)
func resealWriteBack(db *gorm.DB, dataType, dataID, blockHash string) {
	write := func(model interface{}, noCol string) {
		db.Model(model).Where(noCol+" = ?", dataID).Update("block_hash", blockHash)
	}
	switch dataType {
	case models.DataTypeEnergy:
		write(&models.EnergyRecord{}, "record_no")
	case models.DataTypeCredit:
		write(&models.CarbonCredit{}, "credit_no")
	case models.DataTypeTransaction:
		write(&models.Transaction{}, "tx_no")
	case models.DataTypeReport:
		write(&models.ReportRecord{}, "report_no")
	case models.DataTypeIoTRecord:
		write(&models.IoTRecord{}, "record_no")
	case models.DataTypePledge:
		write(&models.PledgeOrder{}, "pledge_no")
	case models.DataTypeArbitration:
		write(&models.ArbitrationCase{}, "case_no")
	case models.DataTypeAgent:
		write(&models.AgentRecord{}, "agent_no")
	case models.DataTypeZKProof:
		write(&models.ZKProofRecord{}, "proof_no")
	case models.DataTypeArchive:
		write(&models.CarbonArchive{}, "archive_no")
	case models.DataTypeFootprint:
		write(&models.ProductFootprint{}, "product_no")
	}
}

// ResealChain 整链重封存：baseUnix 之前的区块时间戳按 3 分钟间隔重排。
// 返回发生变更的区块数量。SimChain 无内存副本(实时查库)，改库即生效。
func (c *SimChain) ResealChain(baseUnix int64) int {
	if c == nil || c.db == nil {
		return 0
	}
	var blocks []models.BlockRecord
	if err := c.db.Order("block_index asc").Find(&blocks).Error; err != nil {
		fmt.Printf("  ⚠️ 链重封存失败(读取区块): %v\n", err)
		return 0
	}

	prevNew := ""
	changed := 0
	for i := range blocks {
		b := &blocks[i]

		// 1) 时间戳平移(仅早于基准时间的区块，按链内顺序 3 分钟间隔排布)
		newTs := b.Timestamp
		if b.Timestamp < baseUnix {
			newTs = baseUnix + int64(i)*180
		}

		// 2) 业务哈希：漂移类型按当前业务数据重算，其余保留
		dataHash := b.DataHash
		if h, ok := resealBusinessHash(c.db, b.DataType, b.DataID); ok {
			dataHash = h
		}

		// 3) 逐块重算 Merkle 根与区块哈希(prev_hash 链式传导)
		merkle := calcMerkleRootForData(dataHash)
		blk := Block{
			Index:      b.BlockIndex,
			Timestamp:  newTs,
			DataHash:   dataHash,
			MerkleRoot: merkle,
			DataType:   b.DataType,
			DataID:     b.DataID,
			PrevHash:   prevNew,
		}
		newHash := blk.computeHash()

		if newHash != b.BlockHash || newTs != b.Timestamp || dataHash != b.DataHash {
			if err := c.db.Model(&models.BlockRecord{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
				"timestamp":   newTs,
				"data_hash":   dataHash,
				"merkle_root": merkle,
				"block_hash":  newHash,
				"prev_hash":   prevNew,
			}).Error; err != nil {
				fmt.Printf("  ⚠️ 区块 #%d 重封存失败: %v\n", b.BlockIndex, err)
				return changed
			}
			resealWriteBack(c.db, b.DataType, b.DataID, newHash)
			changed++
		}
		prevNew = newHash
	}
	return changed
}
