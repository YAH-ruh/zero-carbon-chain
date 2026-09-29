// Package services 提供业务逻辑服务
// ContractService：零碳微证 Solidity 智能合约集成层
//
// 设计思路(与 AI 助手 Mock/Real 模式一致)：
//
//	默认走 Mock 模式 → 所有上链操作写入本地 SQLite 模拟链(项目既有 SHA-256 链)，
//	  不依赖外部 EVM 节点，演示开箱即用。
//	可切换 Real 模式 → 需先启动 Hardhat 本地节点(见 contracts/README.md)，
//	  然后 .env 设置 CONTRACT_MODE=real，后端通过 go-ethereum 客户端调用真实链上合约。
//
// 三种上链动作对齐 Solidity 合约：
//  1. AttestHash   → CarbonAttestation.storeHash       (存证)
//  2. LockCredits  → CarbonCreditToken.lockForOrder    (挂卖单锁仓)
//  3. SettleCredits→ CreditTrading.matchOrders          (撮合结算)
//
// 后续可扩展：Go 端 go-ethereum 客户端代码已留 TODO 占位。
package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"blockchain-demo/config"
	"blockchain-demo/pkg/logger"
)

// ContractMode 合约集成模式
type ContractMode string

const (
	ContractModeMock ContractMode = "mock" // 默认：走本地模拟链
	ContractModeReal ContractMode = "real" // 真实 Hardhat 节点
)

// IContractService 合约服务接口
// Mock 和 Real 两个实现都要满足此接口
type IContractService interface {
	Mode() ContractMode
	// AttestHash 将业务数据 hash 锚定到链上
	AttestHash(recordHash string, atestType string, submitter string) (string, error)
	// LockCredits 卖方挂单时锁仓
	LockCredits(seller string, amount int64, orderID string) error
	// SettleCredits 撮合结算：从卖方锁仓扣出并转给买方
	SettleCredits(seller, buyer string, amount int64, orderID string) error
	// VerifyHash 校验 hash 是否存在于链上
	VerifyHash(recordHash string) (bool, error)
}

// ================================================================
//                 Mock 实现（默认，本地模拟链）
// ================================================================

type mockContractService struct {
	attestations map[string]mockAttestation
	locked       map[string]int64 // seller → locked credits
	txCounter    int64
}

type mockAttestation struct {
	RecordHash string
	Type       string
	Submitter  string
	Timestamp  int64
	PrevHash   string
}

// NewMockContractService 返回 Mock 实现（默认）
func NewMockContractService() IContractService {
	return &mockContractService{
		attestations: make(map[string]mockAttestation),
		locked:       make(map[string]int64),
	}
}

func (m *mockContractService) Mode() ContractMode { return ContractModeMock }

func (m *mockContractService) AttestHash(recordHash, atestType, submitter string) (string, error) {
	m.txCounter++
	// 生成一笔"模拟链上 tx hash"：用 recordHash + 计数器 sha256
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s", recordHash, m.txCounter, submitter)))
	txHash := "0x" + hex.EncodeToString(h[:])

	m.attestations[recordHash] = mockAttestation{
		RecordHash: recordHash,
		Type:       atestType,
		Submitter:  submitter,
		Timestamp:  m.txCounter, // 简化：用计数器代替真实时间
		PrevHash:   txHash,
	}
	logger.Info("[Contract Mock] AttestHash: type=%s submitter=%s hash=%.16s... tx=%.16s...",
		atestType, submitter, recordHash, txHash)
	return txHash, nil
}

func (m *mockContractService) LockCredits(seller string, amount int64, orderID string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	m.locked[seller] += amount
	logger.Info("[Contract Mock] LockCredits: seller=%s amount=%d order=%s locked_now=%d",
		seller, amount, orderID, m.locked[seller])
	return nil
}

func (m *mockContractService) SettleCredits(seller, buyer string, amount int64, orderID string) error {
	if m.locked[seller] < amount {
		return fmt.Errorf("insufficient locked credits for %s: have %d, need %d", seller, m.locked[seller], amount)
	}
	m.locked[seller] -= amount
	logger.Info("[Contract Mock] SettleCredits: seller=%s → buyer=%s amount=%d order=%s",
		seller, buyer, amount, orderID)
	return nil
}

func (m *mockContractService) VerifyHash(recordHash string) (bool, error) {
	_, ok := m.attestations[recordHash]
	return ok, nil
}

// ================================================================
//                 Real 实现（Hardhat 真实节点）
// ================================================================
// 启用方式：
//   1. cd contracts && npx hardhat node   → 启动 Hardhat 本地区块链(8545)
//   2. npx hardhat run scripts/deploy.js --network localhost  → 部署合约
//   3. .env 设置:
//        CONTRACT_MODE=real
//        CONTRACT_RPC_URL=http://127.0.0.1:8545
//        CONTRACT_TOKEN_ADDRESS=0x...
//        CONTRACT_TRADING_ADDRESS=0x...
//        CONTRACT_ATTEST_ADDRESS=0x...
//        CONTRACT_PRIVATE_KEY=0x...   (hardhat 默认账户私钥)
//
// 以下实现为 go-ethereum 客户端占位。
// 完整 go-ethereum 集成需引入 github.com/ethereum/go-ethereum 依赖，
// 并在 ABI 编码层适配 OpenZeppelin ERC-20 / AccessControl 合约接口。

type realContractService struct {
	rpcURL      string
	tokenAddr   string
	tradingAddr string
	attestAddr  string
}

// NewRealContractService 返回 Real 实现
// 若 env 未配置 → 返回 Mock 降级，保证不影响系统运行
func NewRealContractService() IContractService {
	rpc := config.ContractRPCURL()
	if rpc == "" {
		logger.Warn("[Contract Real] CONTRACT_RPC_URL 未配置，降级到 Mock 模式")
		return NewMockContractService()
	}
	return &realContractService{
		rpcURL:      rpc,
		tokenAddr:   config.ContractTokenAddress(),
		tradingAddr: config.ContractTradingAddress(),
		attestAddr:  config.ContractAttestAddress(),
	}
}

func (r *realContractService) Mode() ContractMode { return ContractModeReal }

func (r *realContractService) AttestHash(recordHash, atestType, submitter string) (string, error) {
	// TODO: go-ethereum → CarbonAttestation.storeHash(bytes32, uint8, string)
	// 这里先降级返回 mock 行为，等完整 go-ethereum 依赖就绪再实现
	logger.Warn("[Contract Real] AttestHash 未实现(go-ethereum 待接入)，降级 Mock")
	return NewMockContractService().AttestHash(recordHash, atestType, submitter)
}

func (r *realContractService) LockCredits(seller string, amount int64, orderID string) error {
	logger.Warn("[Contract Real] LockCredits 未实现，降级 Mock")
	return NewMockContractService().LockCredits(seller, amount, orderID)
}

func (r *realContractService) SettleCredits(seller, buyer string, amount int64, orderID string) error {
	logger.Warn("[Contract Real] SettleCredits 未实现，降级 Mock")
	return NewMockContractService().SettleCredits(seller, buyer, amount, orderID)
}

func (r *realContractService) VerifyHash(recordHash string) (bool, error) {
	logger.Warn("[Contract Real] VerifyHash 未实现，降级 Mock")
	return NewMockContractService().VerifyHash(recordHash)
}

// ================================================================
//                 工厂方法（按 config.Mode 创建）
// ================================================================

// NewContractService 按 .env 配置返回对应实现
func NewContractService() IContractService {
	mode := ContractMode(config.ContractMode())
	switch mode {
	case ContractModeReal:
		logger.Info("[Contract] 启用 Real 模式 (rpc=%s)", config.ContractRPCURL())
		return NewRealContractService()
	default:
		logger.Info("[Contract] 启用 Mock 模式 (默认，不依赖外部节点)")
		return NewMockContractService()
	}
}
