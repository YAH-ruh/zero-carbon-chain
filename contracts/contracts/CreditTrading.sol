// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import {CarbonCreditToken} from "./CarbonCreditToken.sol";
import {AccessControl} from "@openzeppelin/contracts/access/AccessControl.sol";

/**
 * @title CreditTrading
 * @notice 零碳微证 · 碳积分挂单 / 撮合成交合约
 * @dev 流程：
 *        1. 卖方(seller)提交挂单 → 锁定 CarbonCreditToken.lockForOrder
 *        2. 交易所(exchange) 撮合 findCounterOrder → 成交
 *           成交后：CreditToken.settleOrUnlock(settle=true) + 支付 USDC 到卖方
 *        3. 卖方撤销 → CreditToken.settleOrUnlock(settle=false) 解锁
 *
 *      角色：
 *        - EXCHANGE_ROLE  : 撮合（matchOrders）
 *        - REGULATOR_ROLE : 紧急撤单 / 仲裁
 *        - 任意用户       : 挂卖单 / 挂买单 / 撤自己的单
 *
 *      本合约仅管理 Order 生命周期。法币支付（如 USDC transfer）需要在更高层或链下完成，
 *      这里用 _quoteToken() 接口保留扩展位。
 */
contract CreditTrading is AccessControl {

    enum OrderSide { SELL, BUY }
    enum OrderStatus { PENDING, MATCHED, CANCELED, SETTLED }

    struct Order {
        bytes32   orderId;
        address   trader;       // 挂单方
        OrderSide side;         // SELL / BUY
        uint256   amount;       // 积分数
        uint256   pricePerUnit; // 单价（引用链下法币，单位 10^6）
        uint64    createdAt;    // block.timestamp
        OrderStatus status;
        bytes32   matchedOrder; // 撮合配对 ID
    }

    CarbonCreditToken public immutable token;

    // orderId => Order
    mapping(bytes32 => Order) public orders;
    // trader => orderIds[]
    mapping(address => bytes32[]) public traderOrders;
    // PENDING 卖单 / 买单队列（FIFO 简化版，价格优先可后续优化）
    bytes32[] private _pendingSells;
    bytes32[] private _pendingBuys;

    bytes32 public constant EXCHANGE_ROLE  = keccak256("EXCHANGE_ROLE");
    bytes32 public constant REGULATOR_ROLE = keccak256("REGULATOR_ROLE");

    // ============ 事件 ============
    event OrderPlaced(bytes32 indexed orderId, address indexed trader, OrderSide side, uint256 amount, uint256 pricePerUnit);
    event OrderMatched(bytes32 indexed sellId, bytes32 indexed buyId, uint256 amount, uint256 pricePerUnit);
    event OrderSettled(bytes32 indexed orderId, bytes32 indexed counterOrder, address indexed seller, address buyer, uint256 amount);
    event OrderCanceled(bytes32 indexed orderId, address indexed trader, string reason);
    event SettlementFailed(bytes32 indexed orderId, string reason);

    constructor(address _token) {
        token = CarbonCreditToken(_token);
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
        _grantRole(EXCHANGE_ROLE, msg.sender);
        _grantRole(REGULATOR_ROLE, msg.sender);
    }

    // ================================================================
    //                         挂单
    // ================================================================

    /**
     * @notice 企业挂卖单：锁定其积分到 escrow
     * @param amount   要卖出的积分数
     * @param pricePerUnit 单价（单位 1e6，实际法币价格由链下或 USDC 映射）
     */
    function placeSellOrder(uint256 amount, uint256 pricePerUnit) external returns (bytes32 orderId) {
        require(amount > 0, "Amount must be positive");
        require(pricePerUnit > 0, "Price must be positive");

        uint256 nonce = _nonce[msg.sender]++;
        orderId = keccak256(abi.encodePacked(msg.sender, block.timestamp, nonce));

        // 调用 Token 合约锁定积分
        token.lockForOrder(msg.sender, amount, orderId);

        orders[orderId] = Order({
            orderId:       orderId,
            trader:        msg.sender,
            side:          OrderSide.SELL,
            amount:        amount,
            pricePerUnit:  pricePerUnit,
            createdAt:     uint64(block.timestamp),
            status:        OrderStatus.PENDING,
            matchedOrder:  bytes32(0)
        });
        traderOrders[msg.sender].push(orderId);
        _pendingSells.push(orderId);

        emit OrderPlaced(orderId, msg.sender, OrderSide.SELL, amount, pricePerUnit);
    }

    /**
     * @notice 企业挂买单（买方信用 / 法币校验在更高层或链下完成）
     */
    function placeBuyOrder(uint256 amount, uint256 pricePerUnit) external returns (bytes32 orderId) {
        require(amount > 0, "Amount must be positive");
        require(pricePerUnit > 0, "Price must be positive");

        uint256 nonce = _nonce[msg.sender]++;
        orderId = keccak256(abi.encodePacked(msg.sender, block.timestamp, nonce));

        orders[orderId] = Order({
            orderId:       orderId,
            trader:        msg.sender,
            side:          OrderSide.BUY,
            amount:        amount,
            pricePerUnit:  pricePerUnit,
            createdAt:     uint64(block.timestamp),
            status:        OrderStatus.PENDING,
            matchedOrder:  bytes32(0)
        });
        traderOrders[msg.sender].push(orderId);
        _pendingBuys.push(orderId);

        emit OrderPlaced(orderId, msg.sender, OrderSide.BUY, amount, pricePerUnit);
    }

    // ================================================================
    //                         取消
    // ================================================================

    /**
     * @notice 卖方 / 买方取消挂单
     *         卖方取消 → 解锁 token（调用 settleOrUnlock(settle=false)）
     */
    function cancelOrder(bytes32 orderId, string calldata reason) external {
        Order storage o = orders[orderId];
        require(o.orderId != bytes32(0), "Order not found");
        require(o.status == OrderStatus.PENDING, "Order already processed");
        require(o.trader == msg.sender || hasRole(REGULATOR_ROLE, msg.sender), "Not authorized");

        // 卖方挂单 → 解锁
        if (o.side == OrderSide.SELL) {
            token.settleOrUnlock(o.trader, address(0), o.amount, orderId, false);
        }
        o.status = OrderStatus.CANCELED;
        emit OrderCanceled(orderId, o.trader, reason);
    }

    // ================================================================
    //                         撮合（交易所）
    // ================================================================

    /**
     * @notice 交易所撮合：指定一组 sellId / buyId 进行成交
     * @dev 价格匹配规则（演示版简化）：
     *      - sell.price <= buy.price 即可成交（取 sell.price，卖方报价优先）
     *      - 部分成交：实际成交 amount = min(sell, buy)
     */
    function matchOrders(bytes32 sellId, bytes32 buyId)
        external
        onlyRole(EXCHANGE_ROLE)
        returns (bool success)
    {
        Order storage sell = orders[sellId];
        Order storage buy  = orders[buyId];

        require(sell.orderId != bytes32(0) && buy.orderId != bytes32(0), "Order not found");
        require(sell.side == OrderSide.SELL && buy.side == OrderSide.BUY, "Wrong side");
        require(sell.status == OrderStatus.PENDING && buy.status == OrderStatus.PENDING, "Not PENDING");
        require(sell.pricePerUnit <= buy.pricePerUnit, "Price mismatch");

        uint256 execAmount = sell.amount < buy.amount ? sell.amount : buy.amount;
        uint256 execPrice  = sell.pricePerUnit; // 卖方报价优先（或可用中间价策略）

        emit OrderMatched(sellId, buyId, execAmount, execPrice);

        // 1. Token 结算：从卖方锁定中扣减并 transfer 给买方
        //    settleOrUnlock(settle=true) → 从 locked 里减并 _transfer(from,to,amount)
        try token.settleOrUnlock(sell.trader, buy.trader, execAmount, sellId, true) {
            // 2. 更新订单状态
            sell.amount       -= execAmount;
            buy.amount        -= execAmount;
            sell.matchedOrder  = buyId;
            buy.matchedOrder   = sellId;

            if (sell.amount == 0) sell.status = OrderStatus.SETTLED; else sell.status = OrderStatus.MATCHED;
            if (buy.amount  == 0) buy.status  = OrderStatus.SETTLED; else buy.status  = OrderStatus.MATCHED;

            emit OrderSettled(sellId, buyId, sell.trader, buy.trader, execAmount);
            return true;
        } catch (bytes memory reason) {
            emit SettlementFailed(sellId, string(reason));
            return false;
        }
    }

    // ================================================================
    //                         查询
    // ================================================================

    function pendingSells() external view returns (bytes32[] memory) { return _pendingSells; }
    function pendingBuys()  external view returns (bytes32[] memory) { return _pendingBuys; }

    function traderOrderCount(address trader) external view returns (uint256) {
        return traderOrders[trader].length;
    }

    // ================================================================
    //                         内部
    // ================================================================

    // trader 级 nonce，避免 orderId 碰撞
    mapping(address => uint256) private _nonce;
}
