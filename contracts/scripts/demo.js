// scripts/demo.js — 零碳微证智能合约完整业务流程演示
// 运行: npx hardhat run scripts/demo.js --network localhost
// 覆盖: 角色权限 / 挂单锁仓 / 撮合结算 / 监管冻结 / 链上存证 / 哈希核验 / 撤单

const { ethers } = require("hardhat");

function line() { console.log("-".repeat(64)); }
function ok(msg) { console.log("  ✓ " + msg); }
function step(n, t) { console.log(`\n${n}. ${t}`); }

async function main() {
  // ====== 账户 ======
  const [deployer, parkAdmin, exchange, regulator, enterpriseA, enterpriseB] = await ethers.getSigners();
  console.log("=".repeat(64));
  console.log("零碳微证 · 智能合约全流程演示");
  console.log("=".repeat(64));
  console.log(`链 ID:     ${(await ethers.provider.getNetwork()).chainId}`);
  console.log(`区块高度:  ${await ethers.provider.getBlockNumber()}`);

  // ====== 加载合约 ======
  const TOKEN_ADDR   = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
  const TRADING_ADDR = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
  const ATTEST_ADDR  = "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0";

  const token   = await ethers.getContractAt("CarbonCreditToken",   TOKEN_ADDR);
  const trading = await ethers.getContractAt("CreditTrading",       TRADING_ADDR);
  const attest  = await ethers.getContractAt("CarbonAttestation",  ATTEST_ADDR);

  line();

  // ================================================================
  // 场景 1: 角色权限核验
  // ================================================================
  step(1, "角色权限核验 (AccessControl)");

  const MINTER_ROLE    = await token.MINTER_ROLE();
  const EXCHANGE_ROLE  = await token.EXCHANGE_ROLE();
  const REGULATOR_ROLE = await token.REGULATOR_ROLE();

  ok(`园区管理员 → MINTER_ROLE:     ${await token.hasRole(MINTER_ROLE, parkAdmin.address)}`);
  ok(`交易所     → EXCHANGE_ROLE:   ${await token.hasRole(EXCHANGE_ROLE, exchange.address)}`);
  ok(`监管核查   → REGULATOR_ROLE:  ${await token.hasRole(REGULATOR_ROLE, regulator.address)}`);
  ok(`CreditTrading → EXCHANGE_ROLE (Token): ${await token.hasRole(EXCHANGE_ROLE, TRADING_ADDR)}`);

  // ================================================================
  // 场景 2: Mint 园区管理员给企业 mint 碳积分
  // ================================================================
  step(2, "园区管理员 mint 碳积分给企业 A");

  const beforeA = await token.balanceOf(enterpriseA.address);
  console.log(`  Mint 前企业 A 余额: ${beforeA.toString()} CCER`);

  const mintTx = await token.connect(parkAdmin).mint(
    enterpriseA.address,
    ethers.parseUnits("800", 0),
    "园区2024年Q1能耗核算: 800吨减排"
  );
  const mintReceipt = await mintTx.wait();
  const afterA = await token.balanceOf(enterpriseA.address);

  ok(`mint 800 CCER  → gasUsed: ${mintReceipt.gasUsed.toString()}`);
  ok(`企业 A 新余额: ${afterA.toString()} CCER`);

  // ================================================================
  // 场景 3: 企业 A 挂卖单 (approve + placeSellOrder)
  // ================================================================
  step(3, "企业 A 挂卖单 (approve → placeSellOrder)");

  // approve Trading 合约可以调用 lockForOrder
  await token.connect(enterpriseA).approve(TRADING_ADDR, ethers.parseUnits("200", 0));

  // 挂卖单: 200 吨 @ 55元/吨 (合约自动取 msg.sender)
  const placeTx = await trading.connect(enterpriseA).placeSellOrder(
    ethers.parseUnits("200", 0),
    55
  );
  const placeReceipt = await placeTx.wait();

  const orderEvent = placeReceipt.logs.find(l => l.eventName === "OrderPlaced");
  const orderId = orderEvent.args.orderId.toString();
  ok(`卖单已挂: orderId=${orderId.slice(0, 16)}..., amount=200, price=55元/吨`);
  console.log(`  gasUsed: ${placeReceipt.gasUsed.toString()}`);

  // 验证锁仓成功
  const locked = await token.lockedOf(enterpriseA.address);
  ok(`企业 A 锁定中积分: ${locked.toString()} CCER`);

  // ================================================================
  // 场景 4: 企业 B 挂买单
  // ================================================================
  step(4, "企业 B 挂买单 (150 吨 @ 60元/吨)");

  // 给企业 B 也 mint 一些
  await token.connect(parkAdmin).mint(enterpriseB.address, ethers.parseUnits("500", 0), "企业B初始配额");
  ok(`企业 B 初始余额: ${(await token.balanceOf(enterpriseB.address)).toString()} CCER`);

  // 挂买单
  const buyTx = await trading.connect(enterpriseB).placeBuyOrder(
    ethers.parseUnits("150", 0),
    60
  );
  const buyReceipt = await buyTx.wait();
  const buyEvent = buyReceipt.logs.find(l => l.eventName === "OrderPlaced");
  const buyOrderId = buyEvent.args.orderId.toString();
  ok(`买单已挂: orderId=${buyOrderId.slice(0, 16)}..., amount=150, price=60元/吨`);

  // ================================================================
  // 场景 5: 交易所撮合 + 结算
  // ================================================================
  step(5, "交易所撮合结算 (matchOrders)");

  const pendingSells = await trading.pendingSells();
  const pendingBuys  = await trading.pendingBuys();
  console.log(`  当前卖单队列: ${pendingSells.length} 条, 买单队列: ${pendingBuys.length} 条`);

  // matchOrders: min(sell, buy) 自动取 150 吨, sell.price=55 成交价
  const matchTx = await trading.connect(exchange).matchOrders(orderId, buyOrderId);
  const matchReceipt = await matchTx.wait();

  const matchedEvent = matchReceipt.logs.find(l => l.eventName === "OrderMatched");
  const settledEvent = matchReceipt.logs.find(l => l.eventName === "OrderSettled");
  ok(`撮合成功: 成交=${matchedEvent.args.amount.toString()} 吨, 成交价=${matchedEvent.args.pricePerUnit} 元/吨`);
  ok(`结算完成: seller=${settledEvent.args.seller.slice(0, 10)}..., buyer=${settledEvent.args.buyer.slice(0, 10)}...`);
  console.log(`  gasUsed: ${matchReceipt.gasUsed.toString()}`);

  // 撮合后锁仓变化
  const lockedAfter = await token.lockedOf(enterpriseA.address);
  ok(`企业 A 剩余锁定: ${lockedAfter.toString()} CCER (原200 - 成交150 = 50剩余锁定)`);

  const balA = await token.balanceOf(enterpriseA.address);
  const balB = await token.balanceOf(enterpriseB.address);
  ok(`企业 A 余额: ${balA.toString()} CCER`);
  ok(`企业 B 余额: ${balB.toString()} CCER`);

  // ================================================================
  // 场景 6: 监管核查冻结违规积分
  // ================================================================
  step(6, "监管冻结违规积分 (freeze)");

  const frozenBefore = await token.frozenOf(enterpriseA.address);
  console.log(`  冻结前企业 A 冻结额: ${frozenBefore.toString()}`);

  const freezeTx = await token.connect(regulator).freeze(
    enterpriseA.address,
    ethers.parseUnits("30", 0),
    "2024Q1 能耗数据存疑，冻结待核查"
  );
  await freezeTx.wait();

  const frozenAfter = await token.frozenOf(enterpriseA.address);
  ok(`监管已冻结 30 吨 → 企业 A 冻结额: ${frozenAfter.toString()} CCER`);

  const availA = await token.availableBalance(enterpriseA.address);
  ok(`企业 A 可用余额: ${availA.toString()} CCER`);

  // ========== 场景 7: 链上哈希存证 ==========
  step(7, "链上哈希存证 (CarbonAttestation.storeHash)");

  // AtestType enum: ENERGY_REPORT=0, CREDIT_ISSUE=1, TRADE=2, ZKP_PROOF=3, OTHER=4
  const ENERGY_REPORT = 0, CREDIT_ISSUE = 1, TRADE = 2;
  const ts = Date.now(); // 让每次 hash 唯一，支持重跑

  const sampleRecords = [
    { hash: ethers.keccak256(ethers.toUtf8Bytes(`能耗记录#${ts}: 企业A, 2024Q1, 用电1500kWh`)), type: ENERGY_REPORT,  meta: "2024Q1能耗明细" },
    { hash: ethers.keccak256(ethers.toUtf8Bytes(`减排凭证#${ts}: 企业A, 光伏项目, 800吨`)),         type: CREDIT_ISSUE,  meta: "CCER光伏项目" },
    { hash: ethers.keccak256(ethers.toUtf8Bytes(`交易流水#${ts}: A→B, 150吨@55`)),                type: TRADE,         meta: "撮合交易#1" },
  ];

  for (const rec of sampleRecords) {
    const tx = await attest.connect(parkAdmin).storeHash(rec.hash, rec.type, rec.meta);
    const r = await tx.wait();
    ok(`锚定成功: type=${["ENERGY","CREDIT","TRADE","ZKP","OTHER"][rec.type]} · ${rec.meta} → tx=${r.hash.slice(0, 12)}...`);
  }

  const totalRecs = await attest.totalRecords();
  ok(`链上存证总数: ${totalRecs.toString()}`);

  // ========== 场景 8: 监管核验哈希完整性 ==========
  step(8, "监管核验哈希完整性 (exists + getRecord)");

  const targetHash = sampleRecords[0].hash;
  const recordExists = await attest.exists(targetHash);
  ok(`哈希 ${targetHash.slice(0, 14)}... → exists=${recordExists}`);

  const rec = await attest.getRecord(targetHash);
  ok(`  类型:   ${["ENERGY_REPORT","CREDIT_ISSUE","TRADE","ZKP_PROOF","OTHER"][rec.atestType]}`);
  ok(`  提交者: ${rec.submitter}`);
  ok(`  时间:   ${new Date(Number(rec.timestamp) * 1000).toISOString()}`);
  ok(`  Meta:   ${rec.meta}`);

  // 假哈希核验
  const fakeExists = await attest.exists(ethers.ZeroHash);
  ok(`假哈希核验 → exists=${fakeExists} (应为 false)`);

  // ========== 场景 9: 挂新单 + 立即撤销 ==========
  step(9, "企业挂新单后立即撤销 (cancelOrder PENDING)");

  // 撮合后的卖单已从 pendingSells 移除（状态 MATCHED），挂一个新的卖单再撤销
  await token.connect(enterpriseA).approve(TRADING_ADDR, ethers.parseUnits("10", 0));
  const newPlaceTx = await trading.connect(enterpriseA).placeSellOrder(
    ethers.parseUnits("10", 0),
    62
  );
  const newPlaceReceipt = await newPlaceTx.wait();
  const newOrderEvent = newPlaceReceipt.logs.find(l => l.eventName === "OrderPlaced");
  const newOrderId = newOrderEvent.args.orderId.toString();
  ok(`新卖单已挂: ${newOrderId.slice(0, 16)}..., amount=10, price=62, status=PENDING`);

  const lockedBeforeCancel = await token.lockedOf(enterpriseA.address);
  ok(`挂新单后企业 A 锁仓: ${lockedBeforeCancel.toString()} CCER`);

  // 撤销
  const cancelTx = await trading.connect(enterpriseA).cancelOrder(newOrderId, "行情变动，手动撤销");
  const cancelReceipt = await cancelTx.wait();
  const cancelEvent = cancelReceipt.logs.find(l => l.eventName === "OrderCanceled");
  ok(`卖单已撤销: orderId=${cancelEvent.args.orderId.slice(0, 16)}..., trader=${cancelEvent.args.trader.slice(0, 10)}...`);

  const lockedAfterCancel = await token.lockedOf(enterpriseA.address);
  ok(`撤销后锁仓释放: ${lockedBeforeCancel} → ${lockedAfterCancel} CCER`);

  // 挂单队列变化
  const finalSells = await trading.pendingSells();
  const finalBuys  = await trading.pendingBuys();
  ok(`最终队列: 卖单=${finalSells.length}, 买单=${finalBuys.length}`);

  // ========== 汇总 ==========
  console.log("\n" + "=".repeat(64));
  console.log("📊 演示汇总");
  console.log("=".repeat(64));
  console.log(`  区块高度:        ${await ethers.provider.getBlockNumber()}`);
  console.log(`  企业 A 余额:     ${(await token.balanceOf(enterpriseA.address)).toString()} CCER`);
  console.log(`  企业 A 锁定:     ${(await token.lockedOf(enterpriseA.address)).toString()} CCER`);
  console.log(`  企业 A 冻结:     ${(await token.frozenOf(enterpriseA.address)).toString()} CCER`);
  console.log(`  企业 A 可用:     ${(await token.availableBalance(enterpriseA.address)).toString()} CCER`);
  console.log(`  企业 B 余额:     ${(await token.balanceOf(enterpriseB.address)).toString()} CCER`);
  console.log(`  挂单队列-卖:     ${(await trading.pendingSells()).length}`);
  console.log(`  挂单队列-买:     ${(await trading.pendingBuys()).length}`);
  console.log(`  链上存证总数:    ${(await attest.totalRecords()).toString()}`);
  console.log("=".repeat(64));
  console.log("✅ 全部 9 个场景演示完成！");
  console.log("=".repeat(64));
}

// 辅助: 在指定 orderIds 列表里查找属于某 trader 的订单
async function findOrderByTrader(trading, orderIds, traderAddress, side) {
  for (const oid of orderIds) {
    const o = await trading.orders(oid);
    if (o.trader === traderAddress && Number(o.side) === side) return oid;
  }
  return null;
}

main().catch(e => { console.error(e); process.exit(1); });
