// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import {ERC20Pausable} from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Pausable.sol";
import {AccessControl} from "@openzeppelin/contracts/access/AccessControl.sol";
import {ERC20Burnable} from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";

/**
 * @title CarbonCreditToken
 * @notice 零碳微证 · 碳积分 ERC-20 合约
 * @dev 角色与权限映射（对应系统中的四类用户）：
 *      - MINTER_ROLE    : 园区管理员(park_admin)，可根据能耗上报结果 mint 碳积分给企业
 *      - EXCHANGE_ROLE  : 碳交易所(exchange)，可在撮合成交时转移积分
 *      - REGULATOR_ROLE : 监管核查(regulator)，可暂停 / 注销违规积分
 *      - DEFAULT_ADMIN_ROLE : 管理员，可分配角色
 *
 *      数据模型对齐 Go 后端 carbon_credits 表：
 *        - enterprise_id  映射  address
 *        - carbon_credits 映射  ERC-20 balanceOf
 *        - status         用 internal _locked / _frozen 状态追踪
 */
contract CarbonCreditToken is ERC20, ERC20Burnable, ERC20Pausable, AccessControl {

    // ============ 角色 ============
    bytes32 public constant MINTER_ROLE    = keccak256("MINTER_ROLE");
    bytes32 public constant EXCHANGE_ROLE  = keccak256("EXCHANGE_ROLE");
    bytes32 public constant REGULATOR_ROLE = keccak256("REGULATOR_ROLE");

    // ============ 锁仓 / 冻结映射 ============
    // 企业被 lock 的积分数量（挂单时由交易合约调用）
    mapping(address => uint256) private _locked;
    // 企业被 regulator 冻结的积分数量
    mapping(address => uint256) private _frozen;

    // ============ 事件 ============
    event CreditsMinted(address indexed to, uint256 amount, string reason);
    event CreditsLocked(address indexed from, uint256 amount, bytes32 indexed orderId);
    event CreditsUnlocked(address indexed to, uint256 amount, bytes32 indexed orderId);
    event CreditsFrozen(address indexed account, uint256 amount, string reason);
    event CreditsReleased(address indexed account, uint256 amount);

    // ============ 构造 ============
    constructor() ERC20("Carbon Credit", "CCER") {
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
        _grantRole(MINTER_ROLE, msg.sender);       // 部署者默认可 mint（便于演示）
        _grantRole(EXCHANGE_ROLE, msg.sender);
        _grantRole(REGULATOR_ROLE, msg.sender);
    }

    // ================================================================
    //                        Mint / Burn
    // ================================================================

    /**
     * @notice 根据能耗核算结果为企业 mint 碳积分
     * @param to     企业钱包地址
     * @param amount 碳积分数（单位 1，decimals=18 但实际用整数即可）
     * @param reason 业务凭证哈希或说明（如 能耗记录 SHA-256）
     */
    function mint(address to, uint256 amount, string calldata reason)
        external
        onlyRole(MINTER_ROLE)
        whenNotPaused
    {
        require(to != address(0), "Zero address");
        require(amount > 0, "Amount must be positive");
        _mint(to, amount);
        emit CreditsMinted(to, amount, reason);
    }

    /**
     * @notice 企业主动注销 / 销毁碳积分（如配额过期 / 合规抵消）
     */
    function burnSelf(uint256 amount) external whenNotPaused {
        _burn(msg.sender, amount);
    }

    // ================================================================
    //                    Lock / Unlock（交易结算用）
    // ================================================================

    /**
     * @notice 撮合挂单时锁定卖方积分（由 CreditTrading 合约调用）
     */
    function lockForOrder(address from, uint256 amount, bytes32 orderId)
        external
        onlyRole(EXCHANGE_ROLE)
        whenNotPaused
    {
        require(amount > 0, "Amount must be positive");
        require(availableBalance(from) >= amount, "Insufficient available balance");
        _locked[from] += amount;
        emit CreditsLocked(from, amount, orderId);
    }

    /**
     * @notice 撮合成交后：从锁定中扣减并转币到买方；或取消挂单解锁
     */
    function settleOrUnlock(
        address from,
        address to,
        uint256 amount,
        bytes32 orderId,
        bool    isSettle
    )
        external
        onlyRole(EXCHANGE_ROLE)
        whenNotPaused
    {
        require(_locked[from] >= amount, "Not enough locked credits");
        _locked[from] -= amount;
        if (isSettle) {
            _transfer(from, to, amount);
        }
        // !isSettle：仅解锁，不动余额
        emit CreditsUnlocked(isSettle ? to : from, amount, orderId);
    }

    // ================================================================
    //                Freeze / Release（监管核查用）
    // ================================================================

    function freeze(address account, uint256 amount, string calldata reason)
        external
        onlyRole(REGULATOR_ROLE)
    {
        require(balanceOf(account) >= _locked[account] + amount, "Cannot freeze locked");
        _frozen[account] += amount;
        emit CreditsFrozen(account, amount, reason);
    }

    function release(address account, uint256 amount)
        external
        onlyRole(REGULATOR_ROLE)
    {
        require(_frozen[account] >= amount, "Not enough frozen");
        _frozen[account] -= amount;
        emit CreditsReleased(account, amount);
    }

    // ================================================================
    //                         查询
    // ================================================================

    function lockedOf(address account)  external view returns (uint256) { return _locked[account]; }
    function frozenOf(address account)  external view returns (uint256) { return _frozen[account]; }

    /** @notice 实际可挂单/可转移的积分 = total - locked - frozen */
    function availableBalance(address account) public view returns (uint256) {
        uint256 total  = balanceOf(account);
        uint256 locked = _locked[account];
        uint256 frozen = _frozen[account];
        return total > locked + frozen ? total - locked - frozen : 0;
    }

    // ================================================================
    //                    Transfer 钩子（锁定/冻结校验）
    // ================================================================

    function _update(address from, address to, uint256 value)
        internal
        override(ERC20, ERC20Pausable)
    {
        if (from != address(0)) {
            require(
                balanceOf(from) >= _locked[from] + _frozen[from] + value,
                "Transfer exceeds available balance"
            );
        }
        super._update(from, to, value);
    }

    // ================================================================
    //                    Pause（监管紧急停用）
    // ================================================================

    function pause()  external onlyRole(REGULATOR_ROLE) { _pause(); }
    function unpause() external onlyRole(REGULATOR_ROLE) { _unpause(); }
}
