package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Block 模拟联盟链区块结构(JSON 直接输出给前端展示)
//
// 区块哈希(BlockHash)计算规则(链式防篡改核心)：
//
//	BlockHash = SHA256( 区块高度 | 时间戳 | 业务哈希 | Merkle根 | 业务类型 | 业务单号 | 前块哈希 )
//
// 任一字段被篡改都会导致哈希不一致，通过"前块哈希+自身哈希"形成强链式关系。
type Block struct {
	Index      uint64 `json:"index"`       // 区块高度(自 0 创世块递增)
	Timestamp  int64  `json:"timestamp"`   // 区块生成时间戳(秒)
	DataHash   string `json:"data_hash"`   // 业务哈希：对业务数据核心字段做真实 SHA-256
	MerkleRoot string `json:"merkle_root"` // 简化 Merkle 根：对业务哈希再做哈希(Merkle 根占位)
	DataType   string `json:"data_type"`   // 业务类型标识: energy/credit/transaction/report/genesis
	DataID     string `json:"data_id"`     // 业务数据单号(用于链上溯源)
	PrevHash   string `json:"prev_hash"`   // 前一个区块哈希(创世块为空)
	Hash       string `json:"hash"`        // 当前区块哈希
}

// headerRaw 返回参与哈希计算的规范化字符串
// 变更说明(v2)：在原有字段基础上增加 MerkleRoot，Merkle 根一并纳入哈希，增强防篡改强度；
// 使用 '|' 分隔避免相邻字段拼接产生歧义。
func (b *Block) headerRaw() string {
	return fmt.Sprintf("%d|%d|%s|%s|%s|%s|%s",
		b.Index, b.Timestamp, b.DataHash, b.MerkleRoot, b.DataType, b.DataID, b.PrevHash)
}

// computeHash 依据区块全部关键字段计算真实 SHA-256 区块哈希
func (b *Block) computeHash() string {
	return sha256Hex([]byte(b.headerRaw()))
}

// calcMerkleRootForData 计算单个业务数据的简化 Merkle 根
// 区块当前承载单笔业务数据：叶子 = SHA256("merkle|" + 业务哈希)，根即叶子本身。
// 通过该结构直观演示 Merkle 树思想，后续若扩展为多笔打包可升级为完整二叉树归并。
func calcMerkleRootForData(dataHash string) string {
	leaf := sha256Hex([]byte("merkle|" + dataHash))
	return merkleRoot([]string{leaf})
}

// merkleRoot 对叶子哈希两两归并得到根(奇数个时复制自身补齐)
func merkleRoot(leaves []string) string {
	if len(leaves) == 0 {
		return ""
	}
	level := make([]string, len(leaves))
	copy(level, leaves)
	for len(level) > 1 {
		next := make([]string, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, sha256Hex([]byte(left+right)))
		}
		level = next
	}
	return level[0]
}

// sha256Hex 计算真实 SHA-256 并返回小写十六进制串(全项目统一入口)
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// jsonBytes 将对象序列化为 JSON(供计算业务哈希使用)
func jsonBytes(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
