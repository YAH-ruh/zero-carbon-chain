// hardhat.config.js — Hardhat v2 CommonJS 配置
// 零碳微证智能合约开发环境
require("@nomicfoundation/hardhat-toolbox");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: {
    version: '0.8.27',
    // evmVersion 用 paris：Ganache GUI 不支持 shanghai 的 PUSH0 指令（会报 invalid opcode）
    settings: {
      optimizer: { enabled: true, runs: 200 },
      evmVersion: 'paris',
    },
  },
  networks: {
    hardhat: {
      chainId: 1337,
    },
    localhost: {
      url: 'http://127.0.0.1:8545',
      chainId: 1337,
    },
    // Ganache GUI 默认端口 7545：npx hardhat run scripts/deploy.js --network ganache
    // 部署账户：优先取环境变量 DEPLOYER_PRIVATE_KEY（在 Ganache GUI 点账户钥匙图标复制私钥），
    // 未配置时回退 hardhat 默认测试账户（需 Ganache 导入对应助记词才有余额）
    ganache: {
      url: 'http://127.0.0.1:7545',
      chainId: 1337,
      accounts: process.env.DEPLOYER_PRIVATE_KEY
        ? [process.env.DEPLOYER_PRIVATE_KEY.startsWith('0x') ? process.env.DEPLOYER_PRIVATE_KEY : '0x' + process.env.DEPLOYER_PRIVATE_KEY]
        : undefined,
    },
    // Sepolia 公共测试网（线上部署， GitHub Pages 方案配套）：
    //   部署前设置环境变量 DEPLOYER_PRIVATE_KEY（0x 前缀私钥，钱包需持有 Sepolia 测试币作 gas）
    //   npx hardhat run scripts/deploy.js --network sepolia
    //   RPC 优先用公共节点 publicnode（免注册），可用 SEPOLIA_RPC_URL 覆盖
    sepolia: {
      url: process.env.SEPOLIA_RPC_URL || 'https://ethereum-sepolia-rpc.publicnode.com',
      chainId: 11155111,
      accounts: process.env.DEPLOYER_PRIVATE_KEY
        ? [process.env.DEPLOYER_PRIVATE_KEY.startsWith('0x') ? process.env.DEPLOYER_PRIVATE_KEY : '0x' + process.env.DEPLOYER_PRIVATE_KEY]
        : undefined,
    },
  },
  paths: {
    sources: './contracts',
    tests: './test',
    cache: './cache',
    artifacts: './artifacts',
  },
}
