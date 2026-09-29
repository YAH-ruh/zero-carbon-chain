/**
 * Web3 公共工具模块 —— MetaMask(window.ethereum) × Sepolia 以太坊公共测试网
 *
 * 设计原则（对应需求交互规则）：
 *  1) 本模块只封装底层调用，绝不主动唤起 MetaMask 弹窗；
 *     所有授权动作均由"页面按钮点击 → Pinia wallet store → 本模块"链路触发
 *  2) 统一错误翻译：把 MetaMask / RPC 错误码转换为中文友好提示，
 *     页面层直接展示 err.message 即可，无需重复判断错误码
 *  3) 预留合约调用入口（sendContractTransaction / callContractRead）：
 *     碳积分 Solidity 合约部署到 Sepolia 后，把 ABI 编码结果传入 data 参数即可
 *
 * Sepolia 公链环境（线上部署固定值）：
 *  - 链 ID：11155111（十六进制 0xaa36a7）
 *  - RPC： 公共节点（publicnode 为主，sepolia.org 备用）
 *  - 区块浏览器： https://sepolia.etherscan.io
 */

// ==================== Sepolia 网络常量 ====================

export const SEPOLIA = {
  chainId: '0xaa36a7', // 11155111 的十六进制（MetaMask 接口要求 hex 格式）
  chainIdDecimal: 11155111,
  chainName: 'Sepolia 测试网',
  rpcUrl: 'https://ethereum-sepolia-rpc.publicnode.com',
  rpcUrls: ['https://ethereum-sepolia-rpc.publicnode.com', 'https://rpc.sepolia.org'],
  explorer: 'https://sepolia.etherscan.io',
  nativeCurrency: { name: 'Sepolia Ether', symbol: 'ETH', decimals: 18 },
}

// ==================== 内部工具 ====================

/**
 * 获取 MetaMask provider（未安装时抛出带中文提示的错误）
 */
function getProvider() {
  const eth = window.ethereum
  if (!eth) {
    throw makeWalletError(
      'NO_PROVIDER',
      '未检测到 MetaMask 插件，请先在浏览器中安装 MetaMask 扩展后重试'
    )
  }
  return eth
}

/**
 * 构造"已翻译"的钱包错误对象（message 为中文提示，handled 标记避免二次翻译）
 */
export function makeWalletError(code, message) {
  const err = new Error(message)
  err.code = code
  err.handled = true
  return err
}

/**
 * 将底层异常统一翻译为中文友好提示
 * 页面 / store 中统一用 formatWalletError(err) 取 message 展示
 */
export function formatWalletError(err) {
  if (!err) return '未知钱包错误'
  if (err.handled) return err.message // 已经是中文错误，直接返回
  switch (err.code) {
    case 4001:
      return '您已取消本次操作（用户拒绝授权/签名）'
    case -32002:
      return 'MetaMask 已有一个待处理的授权弹窗，请打开 MetaMask 完成或取消后再试'
    case 4902:
      return '当前钱包尚未添加 Sepolia 测试网（链ID 11155111）'
    default: {
      const msg = String(err.message || '')
      if (msg.includes('gas')) return '交易执行失败：Gas 不足或估算失败，请调整后重试'
      if (msg.includes('insufficient funds')) return '交易失败：账户 ETH 余额不足，请先从水龙头领取 Sepolia 测试币'
      if (msg.includes('Nonce')) return '交易失败：Nonce 冲突，请稍后重试'
      return msg || '钱包操作失败，请重试'
    }
  }
}

/** UTF-8 字符串 → 0x 前缀十六进制（personal_sign 签名内容要求 hex 格式） */
function utf8ToHex(str) {
  const bytes = new TextEncoder().encode(str)
  let hex = '0x'
  for (const b of bytes) hex += b.toString(16).padStart(2, '0')
  return hex
}

// ==================== 连接 / 断开 ====================

/**
 * 连接钱包：唤起 MetaMask 账户选择弹窗（wallet_requestPermissions），
 * 用户可在原生弹窗中自主勾选任一/多个账户，返回全部勾选账户地址数组
 * ⚠️ 必须由用户点击【连接钱包】按钮触发，本模块不做任何自动调用
 */
export async function connectWalletAccounts() {
  const eth = getProvider()
  // 重新发起 eth_accounts 权限申请：每次都会弹出 MetaMask 账户勾选窗（含全部导入账户）
  await eth.request({
    method: 'wallet_requestPermissions',
    params: [{ eth_accounts: {} }],
  })
  const accounts = await eth.request({ method: 'eth_accounts' })
  if (!Array.isArray(accounts) || accounts.length === 0) {
    throw makeWalletError('EMPTY_ACCOUNTS', '未获取到任何钱包账户，请在 MetaMask 弹窗中勾选账户')
  }
  return accounts
}

/**
 * 前端语义断开：仅移除事件监听（MetaMask 不支持程序化断开授权，
 * 真正的授权解除需用户在 MetaMask 扩展中手动操作）
 */
export function disconnectWalletListeners() {
  try {
    window.ethereum?.removeAllListeners?.('accountsChanged')
    window.ethereum?.removeAllListeners?.('chainChanged')
  } catch {
    /* 忽略：provider 不存在时无需清理 */
  }
}

// ==================== 网络 ====================

/**
 * 获取当前链 ID（返回十进制数字，便于与 1337 比较）
 */
export async function getChainId() {
  const eth = getProvider()
  const hex = await eth.request({ method: 'eth_chainId' })
  return parseInt(hex, 16)
}

/**
 * 切换到 Sepolia 测试网（会唤起 MetaMask 确认弹窗）
 * 若钱包中尚未添加 Sepolia 网络（错误码 4902），自动引导添加网络信息
 */
export async function switchToSepolia() {
  const eth = getProvider()
  try {
    await eth.request({
      method: 'wallet_switchEthereumChain',
      params: [{ chainId: SEPOLIA.chainId }],
    })
  } catch (err) {
    if (err.code === 4902) {
      // 钱包未收录 Sepolia → 提交网络信息让用户确认添加
      await eth.request({
        method: 'wallet_addEthereumChain',
        params: [
          {
            chainId: SEPOLIA.chainId,
            chainName: SEPOLIA.chainName,
            rpcUrls: SEPOLIA.rpcUrls,
            nativeCurrency: SEPOLIA.nativeCurrency,
            blockExplorerUrls: [SEPOLIA.explorer],
          },
        ],
      })
    } else {
      throw err
    }
  }
  return SEPOLIA.chainIdDecimal
}

// ==================== 账户列表说明 ====================
// Sepolia 为公链环境：账户列表仅能通过 MetaMask 授权（wallet_requestPermissions）获取，
// 不存在本地节点直连枚举账户的场景；未安装插件时由 WalletModal 引导用户安装 MetaMask。

// ==================== 签名 / 交易 ====================

/**
 * 对消息进行 personal_sign 签名（唤起 MetaMask 签名确认弹窗）
 * @param {string} message  待签名消息（如 ZKP 证明哈希）
 * @param {string} address  当前钱包地址
 * @returns {string} 签名结果（0x...）
 */
export async function signMessage(message, address) {
  const eth = getProvider()
  return eth.request({
    method: 'personal_sign',
    params: [utf8ToHex(message), address],
  })
}

/**
 * 钱包签名看门狗：MetaMask 在部分安全拦截场景（如 localhost 触发 Blockaid 风控）
 * 下，签名请求的 promise 可能永久挂起（不 resolve 也不 reject），导致页面 loading
 * 无法结束。用超时竞速兜底：超时后抛出可翻译错误，保证调用方的 finally 一定能执行。
 * 注意：仅竞速包装，不改动原钱包调用本身；超时后 MetaMask 侧请求仍在，
 * 立即重试会收到 -32002（已有中文翻译）。
 * @param {Promise} promise 钱包签名/交易 promise
 * @param {number} ms 超时毫秒数，默认 60s
 */
export function withWalletWatchdog(promise, ms = 60000) {
  let timer
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => {
      reject(makeWalletError(
        'WALLET_TIMEOUT',
        '钱包签名响应超时，请检查 MetaMask 是否有未处理的弹窗（如安全提示）后重试'
      ))
    }, ms)
  })
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer))
}

/**
 * 发送普通交易（唤起 MetaMask 签名确认弹窗，如积分转出）
 * @param {object} txParams 标准以太坊交易对象
 *   { from, to, value(hex), gas?, gasPrice?, data? }
 * @returns {string} 交易哈希 txHash
 */
export async function sendTransaction(txParams) {
  const eth = getProvider()
  return eth.request({ method: 'eth_sendTransaction', params: [txParams] })
}

// ==================== 合约调用预留入口（第三阶段启用） ====================

/**
 * 【预留】合约读方法：eth_call 只读调用，不消耗 gas、不弹窗
 * @param {string} to   合约地址
 * @param {string} data ABI 编码后的调用数据（functionSelector + 参数）
 */
export async function callContractRead(to, data) {
  const eth = getProvider()
  return eth.request({ method: 'eth_call', params: [{ to, data }, 'latest'] })
}

/**
 * 【预留】合约写方法：调用合约修改状态的方法，MetaMask 签名确认
 * @param {string} from 钱包地址（交易发起人）
 * @param {string} to   合约地址
 * @param {string} data ABI 编码后的方法调用数据
 * @param {string} value 附带的 ETH（hex），默认 0
 * @returns {string} 交易哈希
 */
export async function sendContractTransaction(from, to, data, value = '0x0') {
  const eth = getProvider()
  return eth.request({
    method: 'eth_sendTransaction',
    params: [{ from, to, value, data }],
  })
}

// ==================== 链上数据读取（第三阶段页面接入用） ====================

/**
 * 查询账户原生 ETH 余额（Sepolia），返回格式化字符串（保留 4 位小数）
 */
export async function getNativeBalance(address) {
  const eth = getProvider()
  const hex = await eth.request({ method: 'eth_getBalance', params: [address, 'latest'] })
  return (parseInt(hex, 16) / 1e18).toFixed(4)
}

/**
 * 查询最新区块高度（返回十进制数字）
 */
export async function getBlockNumber() {
  const eth = getProvider()
  const hex = await eth.request({ method: 'eth_blockNumber' })
  return parseInt(hex, 16)
}

/**
 * 查询最新区块摘要：{ number 区块高度, hash 区块哈希, txCount 区块内交易数 }
 */
export async function getLatestBlock() {
  const eth = getProvider()
  const block = await eth.request({ method: 'eth_getBlockByNumber', params: ['latest', false] })
  if (!block) throw makeWalletError('RPC_EMPTY', '链上未查询到区块数据，请确认当前网络为 Sepolia 测试网')
  return {
    number: parseInt(block.number, 16),
    hash: block.hash,
    txCount: (block.transactions || []).length,
  }
}

// ==================== 碳积分合约预留配置（对接 Solidity 合约时替换地址即可） ====================

/**
 * 合约地址与方法选择器配置（需求交互规则 7：预留 ABI 调用入口）
 * ⚠️ address 初始为占位地址：真实部署地址由 deploy.js 产出 frontend/public/contracts.json，
 *    运行时通过 loadDeployedContracts() 动态加载覆盖（合约重新部署后地址会变化）。
 * 方法选择器 = keccak256("函数签名") 前 4 字节：
 *   - balanceOf(address)        → 0x70a08231（标准 ERC20，真实值）
 *   - transfer(address,uint256) → 0xa9059cbb（标准 ERC20，真实值）
 *   - lockForOrder(address,uint256,bytes32) → 0x4f3b75b6（已对链上部署合约校准，eth_call 模拟验证通过）
 *   - 其余合约（zkpRegistry/rollupVerifier）仍为业务方法占位选择器，对接真实合约 ABI 后请校准
 */
export const CARBON_CONTRACTS = {
  /** 碳积分合约：余额查询 / 挂单锁定 / 转出 */
  carbonCreditToken: {
    address: '0x1111111111111111111111111111111111111111',
    methods: {
      balanceOf: '0x70a08231',                    // balanceOf(address)
      transfer: '0xa9059cbb',                     // transfer(address,uint256)
      lockForOrder: '0x4f3b75b6',                 // lockForOrder(address,uint256,bytes32)（真实选择器，已对链上部署合约校准）
    },
  },
  /** ZKP 存证合约：证明哈希上链 */
  zkpRegistry: {
    address: '0x2222222222222222222222222222222222222222',
    methods: {
      storeProof: '0xc0c0c0c0', // storeProof(bytes32)（占位，部署后校准）
    },
  },
  /** Layer2 校验合约：Rollup 批次故障证明校验 */
  rollupVerifier: {
    address: '0x3333333333333333333333333333333333333333',
    methods: {
      verifyBatch: '0xfaceface', // verifyBatch(bytes32)（占位，部署后校准）
    },
  },
  /** 交易所托管地址：撮合成交时积分先授权转移至托管（部署后由 CreditTrading 合约地址覆盖） */
  escrowAddress: '0x4444444444444444444444444444444444444444',
}

// ==================== 合约地址校验与部署产物加载 ====================

/** 占位地址集合（未部署时使用的假地址，禁止发起真实交易） */
const PLACEHOLDER_ADDRESSES = new Set([
  '0x1111111111111111111111111111111111111111',
  '0x2222222222222222222222222222222222222222',
  '0x3333333333333333333333333333333333333333',
  '0x4444444444444444444444444444444444444444',
])

/**
 * 地址格式校验（等价 ethers.isAddress：0x + 40 位十六进制）
 * 前端保持零区块链依赖设计，未引入 ethers 包，此处为等价实现
 * @param {string} addr 待校验地址
 * @returns {boolean}
 */
export function isAddress(addr) {
  return /^0x[0-9a-fA-F]{40}$/.test(String(addr || ''))
}

/**
 * 合约地址可用性校验：格式合法 且 不是占位地址
 * 用于发交易前的前置拦截，避免把无效 to 地址发给 MetaMask
 * @param {string} addr 合约地址
 * @returns {boolean}
 */
export function isUsableContractAddress(addr) {
  return isAddress(addr) && !PLACEHOLDER_ADDRESSES.has(String(addr).toLowerCase())
}

let deployedLoaded = false

/**
 * 加载 deploy.js 部署后产出的真实合约地址（frontend/public/contracts.json），
 * 并覆盖 CARBON_CONTRACTS 中的占位地址。合约重新部署后，
 * 部署脚本会同步刷新该文件，前端每次进入页面重新拉取即可拿到最新地址。
 * 加载失败时静默保留占位地址，由页面层 isUsableContractAddress 前置校验兜底。
 * @returns {Promise<object>} 合约配置对象（可继续 await 后读取最新 address）
 */
export async function loadDeployedContracts() {
  if (deployedLoaded) return CARBON_CONTRACTS
  try {
    const res = await fetch(`${import.meta.env.BASE_URL}contracts.json`, { cache: 'no-store' })
    if (res.ok) {
      const data = await res.json()
      const mapping = {
        CarbonCreditToken: 'carbonCreditToken',
        CreditTrading: 'zkpRegistry', // 结构占位复用：真实业务暂只消费 carbonCreditToken
        CarbonAttestation: 'rollupVerifier',
      }
      for (const [deployName, configKey] of Object.entries(mapping)) {
        const addr = data.contracts?.[deployName]?.address
        if (addr && isAddress(addr)) {
          CARBON_CONTRACTS[configKey].address = addr
        }
      }
      // 托管地址 = CreditTrading 撮合合约（负责 lockForOrder / settleOrUnlock）
      const tradingAddr = data.contracts?.CreditTrading?.address
      if (tradingAddr && isAddress(tradingAddr)) {
        CARBON_CONTRACTS.escrowAddress = tradingAddr
      }
    }
  } catch {
    /* 静默：保留占位地址，页面层前置校验会拦截 */
  }
  deployedLoaded = true
  return CARBON_CONTRACTS
}

// 模块加载即预取部署产物（不阻塞页面，点击交易时已就绪）
loadDeployedContracts()

/** 数值 → 32 字节 ABI 编码字（uint256），如 100 → 0x00...64 */
export function abiEncodeUint256(value) {
  const v = BigInt(value ?? 0)
  if (v < 0n) throw makeWalletError('ENCODE_ERROR', '编码数值不能为负数')
  return '0x' + v.toString(16).padStart(64, '0')
}

/** 十六进制串（地址 / 哈希）→ 32 字节 ABI 编码字，不足左补零 */
export function abiEncodeWord(hexStr) {
  const raw = String(hexStr || '').replace(/^0x/i, '')
  if (!/^[0-9a-fA-F]+$/.test(raw)) {
    throw makeWalletError('ENCODE_ERROR', '编码内容必须为十六进制字符串（地址 / 哈希）')
  }
  return '0x' + raw.toLowerCase().padStart(64, '0')
}

/** 拼接合约方法调用数据：方法选择器 + 编码参数（作为交易的 data 字段） */
export function abiEncodeCall(selector, words = []) {
  return selector + words.map(w => String(w).replace(/^0x/i, '')).join('')
}

// ==================== 事件监听（账户切换 / 链切换） ====================

/** 注册账户切换监听（用户在 MetaMask 中切换账户时回调，参数为最新账户数组） */
export function onAccountsChanged(callback) {
  window.ethereum?.on?.('accountsChanged', callback)
}

/** 注册链切换监听（切换网络时回调，参数为十六进制链 ID） */
export function onChainChanged(callback) {
  window.ethereum?.on?.('chainChanged', callback)
}
