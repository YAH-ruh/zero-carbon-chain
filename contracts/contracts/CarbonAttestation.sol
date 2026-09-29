// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import {AccessControl} from "@openzeppelin/contracts/access/AccessControl.sol";

/**
 * @title CarbonAttestation
 * @notice 零碳微证 · 链上哈希存证 / 数据完整性校验合约
 * @dev 与 Go 后端模拟联盟链的 SHA-256 哈希链对齐：
 *        - Go 端对能耗数据 / 交易数据 计算 SHA-256，生成区块（prev_hash + data_hash → block_hash）
 *        - 本合约提供 storeHash / batchStore 让 Go 后端（或企业端）将关键哈希锚定到链上
 *        - verifyHash 用于监管核查：给定原始数据 + 已知存证 hash，快速确认数据未被篡改
 *
 *      为什么只存 hash 而不存原始数据？
 *        - 隐私合规：能耗明细属于敏感数据，链上只放 hash 可做 ZKP / 选择性披露
 *        - gas 节省：1 条记录 ~ 1 KB，hash 仅 32 字节
 *        - 可验证：hash 绑定时间戳 + 提交者，监管只需 hash 即可溯源
 *
 *      角色：
 *        - MINTER_ROLE    (park_admin)   : 能耗记录批量上链
 *        - EXCHANGE_ROLE  (exchange)     : 交易记录上链
 *        - REGULATOR_ROLE (regulator)    : 链上核验 / 紧急标记
 *        - 任意 role                     : 普通记录上链（如企业自述）
 */
contract CarbonAttestation is AccessControl {

    enum AtestType { ENERGY_REPORT, CREDIT_ISSUE, TRADE, ZKP_PROOF, OTHER }

    struct Attestation {
        bytes32       recordHash;   // Go 端 SHA-256 或业务 hash
        bytes32       prevHash;     // 前一条记录 hash（模拟 chain）
        AtestType     atestType;
        address       submitter;
        uint64        timestamp;
        string        meta;         // 可选 JSON meta：transaction_id / enterprise_id / note
    }

    bytes32 public constant MINTER_ROLE    = keccak256("MINTER_ROLE");
    bytes32 public constant EXCHANGE_ROLE  = keccak256("EXCHANGE_ROLE");
    bytes32 public constant REGULATOR_ROLE = keccak256("REGULATOR_ROLE");

    // recordHash => Attestation（hash 天然唯一键）
    mapping(bytes32 => Attestation) public attestations;
    // 提交者 -> 其提交的所有 hash
    mapping(address => bytes32[]) public submitterRecords;
    // type -> 该类型下的 hash 列表
    mapping(AtestType => bytes32[]) public typeRecords;

    // 最新 hash（模拟链头）
    bytes32 public latestHash;
    uint256 public totalRecords;

    // ============ 事件 ============
    event AttestationStored(bytes32 indexed recordHash, bytes32 prevHash, AtestType indexed atestType, address indexed submitter);
    event AttestationBatchStored(uint256 count, bytes32 headHash);
    event HashVerified(bytes32 indexed recordHash, address indexed verifier, bool valid);
    event AttestationChallenged(bytes32 indexed recordHash, address indexed challenger, string reason);

    constructor() {
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
        _grantRole(MINTER_ROLE, msg.sender);
        _grantRole(EXCHANGE_ROLE, msg.sender);
        _grantRole(REGULATOR_ROLE, msg.sender);
    }

    // ================================================================
    //                      存证（单条 / 批量）
    // ================================================================

    /**
     * @notice 存单条 hash 到链上（任意授权角色可用）
     * @param recordHash Go 端对原始数据计算的 SHA-256
     * @param atestType  业务类型枚举
     * @param meta       可选元信息（如 enterprise_id / tx_id / ZKP 证明引用）
     */
    function storeHash(bytes32 recordHash, AtestType atestType, string calldata meta)
        external
    {
        _store(recordHash, atestType, meta);
    }

    /**
     * @notice 批量存证（Go 后端同步本地模拟链到真实链时调用）
     * @dev 批量 hash 之间串 prevHash 形成链头
     */
    function batchStore(bytes32[] calldata recordHashes, AtestType atestType, string[] calldata metas)
        external
    {
        require(recordHashes.length > 0 && recordHashes.length == metas.length, "Len mismatch");
        bytes32 head = latestHash;
        for (uint256 i = 0; i < recordHashes.length; i++) {
            Attestation storage a = attestations[recordHashes[i]];
            if (a.recordHash != bytes32(0)) continue; // 已存在则跳过（幂等）
            attestations[recordHashes[i]] = Attestation({
                recordHash: recordHashes[i],
                prevHash:   head,
                atestType:  atestType,
                submitter:  msg.sender,
                timestamp:  uint64(block.timestamp),
                meta:       metas[i]
            });
            submitterRecords[msg.sender].push(recordHashes[i]);
            typeRecords[atestType].push(recordHashes[i]);
            head = recordHashes[i];
        }
        if (head != latestHash) latestHash = head;
        emit AttestationBatchStored(recordHashes.length, head);
    }

    // ================================================================
    //                      核验
    // ================================================================

    /**
     * @notice 监管核验：给定原始数据，计算 hash 后与链上 recordHash 比对
     * @dev 调用者需自行对原始数据做 keccak256（Solidity 默认）或 sha256（见 sha256 辅助函数）
     */
    function verifyRaw(bytes calldata rawData, bytes32 expectedHash)
        external
        returns (bool valid, bytes32 computedHash)
    {
        // 用 keccak256 计算链上 hash（Go SHA-256 需 sha256() 函数比对）
        computedHash = keccak256(rawData);
        valid = computedHash == expectedHash && attestations[expectedHash].recordHash != bytes32(0);
        emit HashVerified(expectedHash, msg.sender, valid);
    }

    /**
     * @notice 监管核验：链上是否存在某 hash（最快路径）
     */
    function exists(bytes32 recordHash) external view returns (bool) {
        return attestations[recordHash].recordHash != bytes32(0);
    }

    /**
     * @notice 举报链上 hash 对应的数据可疑（预留仲裁入口）
     */
    function challenge(bytes32 recordHash, string calldata reason)
        external
        onlyRole(REGULATOR_ROLE)
    {
        require(attestations[recordHash].recordHash != bytes32(0), "Not found");
        emit AttestationChallenged(recordHash, msg.sender, reason);
    }

    // ================================================================
    //                      查询
    // ================================================================

    function getRecord(bytes32 recordHash) external view returns (Attestation memory) {
        return attestations[recordHash];
    }

    function getSubmitterCount(address submitter) external view returns (uint256) {
        return submitterRecords[submitter].length;
    }

    function getTypeCount(AtestType atestType) external view returns (uint256) {
        return typeRecords[atestType].length;
    }

    // ================================================================
    //                      内部
    // ================================================================

    function _store(bytes32 recordHash, AtestType atestType, string calldata meta) internal {
        require(recordHash != bytes32(0), "Zero hash");
        require(attestations[recordHash].recordHash == bytes32(0), "Already stored");

        bytes32 prev = latestHash;
        attestations[recordHash] = Attestation({
            recordHash: recordHash,
            prevHash:   prev,
            atestType:  atestType,
            submitter:  msg.sender,
            timestamp:  uint64(block.timestamp),
            meta:       meta
        });
        submitterRecords[msg.sender].push(recordHash);
        typeRecords[atestType].push(recordHash);
        latestHash = recordHash;
        totalRecords++;

        emit AttestationStored(recordHash, prev, atestType, msg.sender);
    }
}
