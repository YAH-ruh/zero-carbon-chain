// scripts/deploy.js — 零碳微证合约批量部署脚本
// 运行：npx hardhat run scripts/deploy.js --network localhost
// 流程：CreditToken → CreditTrading → CarbonAttestation → 输出地址到 .contracts.json
const fs = require("fs");
const path = require("path");

async function main() {
  const [deployer, parkAdmin, exchange, regulator, enterprise] = await ethers.getSigners();

  console.log("=".repeat(60));
  console.log("零碳微证 · Solidity 智能合约部署");
  console.log("=".repeat(60));
  console.log(`部署账户:  ${deployer.address}`);
  console.log(`ETH 余额: ${(await ethers.provider.getBalance(deployer.address)).toString()}`);
  console.log("-".repeat(60));

  // ========== 1. 部署 CarbonCreditToken ==========
  console.log("\n① 部署 CarbonCreditToken (ERC-20 碳积分)...");
  const CreditToken = await ethers.getContractFactory("CarbonCreditToken");
  const token = await CreditToken.deploy();
  await token.waitForDeployment();
  const tokenAddr = await token.getAddress();
  console.log(`   ✓ CarbonCreditToken 地址: ${tokenAddr}`);

  // ========== 2. 部署 CreditTrading（依赖 CreditToken） ==========
  console.log("\n② 部署 CreditTrading（撮合交易合约）...");
  const Trading = await ethers.getContractFactory("CreditTrading");
  const trading = await Trading.deploy(tokenAddr);
  await trading.waitForDeployment();
  const tradingAddr = await trading.getAddress();
  console.log(`   ✓ CreditTrading 地址: ${tradingAddr}`);

  // ========== 3. 部署 CarbonAttestation ==========
  console.log("\n③ 部署 CarbonAttestation（链上哈希存证）...");
  const Attest = await ethers.getContractFactory("CarbonAttestation");
  const attest = await Attest.deploy();
  await attest.waitForDeployment();
  const attestAddr = await attest.getAddress();
  console.log(`   ✓ CarbonAttestation 地址: ${attestAddr}`);

  // ========== 4. 配置角色 ==========
  console.log("\n④ 配置四类角色权限...");

  // 给 CreditTrading 合约授予 EXCHANGE_ROLE（让它能调用 token.lockForOrder / settleOrUnlock）
  const EXCHANGE_ROLE = await token.EXCHANGE_ROLE();
  await token.grantRole(EXCHANGE_ROLE, tradingAddr);
  console.log(`   ✓ CreditTrading → EXCHANGE_ROLE (CreditToken)`);

  // 给演示账户授予角色
  const MINTER_ROLE    = await token.MINTER_ROLE();
  const REGULATOR_ROLE = await token.REGULATOR_ROLE();
  const TRADING_EX_RL  = await trading.EXCHANGE_ROLE();
  const TRADING_RG_RL  = await trading.REGULATOR_ROLE();
  const ATTEST_MT_RL   = await attest.MINTER_ROLE();
  const ATTEST_EX_RL   = await attest.EXCHANGE_ROLE();
  const ATTEST_RG_RL   = await attest.REGULATOR_ROLE();

  if (parkAdmin) {
    await token.grantRole(MINTER_ROLE,    parkAdmin.address);
    await attest.grantRole(ATTEST_MT_RL,   parkAdmin.address);
    console.log(`   ✓ ParkAdmin  → MINTER_ROLE (Token + Attest)`);
  }
  if (exchange) {
    await token.grantRole(EXCHANGE_ROLE, exchange.address);
    await trading.grantRole(TRADING_EX_RL, exchange.address);
    await attest.grantRole(ATTEST_EX_RL,  exchange.address);
    console.log(`   ✓ Exchange   → EXCHANGE_ROLE (Token + Trading + Attest)`);
  }
  if (regulator) {
    await token.grantRole(REGULATOR_ROLE, regulator.address);
    await trading.grantRole(TRADING_RG_RL, regulator.address);
    await attest.grantRole(ATTEST_RG_RL,  regulator.address);
    console.log(`   ✓ Regulator  → REGULATOR_ROLE (Token + Trading + Attest)`);
  }

  // ========== 5. 初始 mint 给企业演示 ==========
  if (enterprise) {
    console.log(`\n⑤ 给企业账户 mint 1562 碳积分 (演示用)...`);
    const tx = await token.mint(enterprise.address, ethers.parseUnits("1562", 0), "demo initial mint");
    await tx.wait();
    const bal = await token.balanceOf(enterprise.address);
    console.log(`   ✓ 企业初始余额: ${bal.toString()} CCER`);
  }

  // ========== 6. 输出地址到 .contracts.json + 前端静态副本 ==========
  const deployed = {
    network: network.name,
    deployedAt: new Date().toISOString(),
    contracts: {
      CarbonCreditToken: {
        address: tokenAddr,
        abi: "artifacts/contracts/CarbonCreditToken.sol/CarbonCreditToken.json",
      },
      CreditTrading: {
        address: tradingAddr,
        abi: "artifacts/contracts/CreditTrading.sol/CreditTrading.json",
      },
      CarbonAttestation: {
        address: attestAddr,
        abi: "artifacts/contracts/CarbonAttestation.sol/CarbonAttestation.json",
      },
    },
    signers: {
      deployer:  deployer.address,
      parkAdmin: parkAdmin?.address,
      exchange:  exchange?.address,
      regulator: regulator?.address,
      enterprise: enterprise?.address,
    },
  };

  const outPath = path.join(__dirname, "..", ".contracts.json");
  fs.writeFileSync(outPath, JSON.stringify(deployed, null, 2));
  console.log(`\n⑥ 地址已写入 .contracts.json`);

  // 同步产出前端静态副本：Vite 从 public/ 提供静态文件，前端运行时
  // fetch('/contracts.json') 加载真实合约地址，覆盖 web3.js 中的占位地址。
  // Ganache 重启重新部署后重新运行本脚本即可，前端无需改代码。
  const fePath = path.join(__dirname, "..", "..", "frontend", "public", "contracts.json");
  fs.writeFileSync(fePath, JSON.stringify(deployed, null, 2));
  console.log(`   ✓ 前端地址副本已写入 frontend/public/contracts.json`);

  console.log("\n" + "=".repeat(60));
  console.log("✅ 全部部署完成！");
  console.log("=".repeat(60));
  console.log(`\n快速测试命令:`);
  console.log(`  npx hardhat console --network localhost`);
  console.log(`  > const t = await ethers.getContractAt("CarbonCreditToken", "${tokenAddr}")`);
  console.log(`  > (await t.balanceOf("${enterprise?.address || deployer.address}")).toString()`);
}

main()
  .then(() => process.exit(0))
  .catch((e) => { console.error(e); process.exit(1); });
