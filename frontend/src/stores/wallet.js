/**
 * 钱包连接状态 —— Pinia Store
 *
 * 职责（对应需求交互规则 1/3/4/5）：
 *  1) 全局共享钱包连接状态 / 钱包地址 / 网络 ID，4 个钱包页面统一读取
 *  2) connectAccount()：用户点击【连接钱包】→ 弹出钱包账户弹窗(WalletModal)选择账户 →
 *     选定后写入连接状态（地址 + Sepolia 测试网/链ID 11155111），并绑定 provider 事件监听做状态同步
 *  3) 监听 provider 的账户切换 / 链切换事件，状态实时同步并给出友好提示
 *  4) disconnect()：前端语义断开（清空状态 + 解绑监听）
 *
 * 注意：刷新页面后不自动重连（绝不自动唤起弹窗，需用户手动点击连接）
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import {
  SEPOLIA,
  switchToSepolia,
  disconnectWalletListeners,
  formatWalletError,
  onAccountsChanged,
  onChainChanged,
} from '../utils/web3.js'

export const useWalletStore = defineStore('wallet', () => {
  // ==================== 状态 ====================
  const isConnected = ref(false)   // 是否已完成钱包授权
  const address = ref('')          // 当前钱包地址
  const chainId = ref(0)           // 当前网络链 ID（十进制）
  const connecting = ref(false)    // 连接进行中（按钮 loading，防重复点击）
  const listenerBound = ref(false) // 事件监听是否已注册（防止重复绑定）

  // ==================== 计算属性 ====================
  /** 缩略地址展示：0x1234...abcd */
  const shortAddress = computed(() => {
    if (!address.value) return ''
    return address.value.slice(0, 6) + '...' + address.value.slice(-4)
  })
  /** 钱包是否"就绪"：已连接 且 处于 Sepolia(11155111) 网络（页面据此放开合约交互按钮） */
  const isReady = computed(() => isConnected.value && chainId.value === SEPOLIA.chainIdDecimal)

  // ==================== 核心动作 ====================

  /**
   * 连接钱包（用户在【钱包账户弹窗 WalletModal】中选定账户后触发）
   * 流程：弹窗账户（MetaMask 授权账户 / 本地节点真实账户 / 内置演示账户）→ 写入连接状态 → 绑定事件监听
   * @param {string} addr 选定的钱包地址
   * @param {number} cid  当前链 ID（MetaMask 模式传真实值；缺省视为已在 Sepolia）
   */
  function connectAccount(addr, cid = SEPOLIA.chainIdDecimal) {
    if (!addr) return
    address.value = addr
    chainId.value = cid
    isConnected.value = true
    bindListeners()
    ElMessage.success('钱包连接成功：' + shortAddress.value)
  }

  /** 手动触发切换到 Sepolia 测试网（供页面"切换网络"按钮使用，需浏览器钱包插件支持） */
  async function switchNetwork() {
    try {
      await switchToSepolia()
      chainId.value = SEPOLIA.chainIdDecimal
      ElMessage.success('已切换到 Sepolia 测试网（11155111）')
    } catch (err) {
      ElMessage.error(formatWalletError(err))
    }
  }

  /**
   * 断开连接（前端语义）：清空状态 + 解绑监听
   * 如需彻底解除站点授权，请在 MetaMask 扩展 → 已连接的网站 中手动移除
   */
  function disconnect() {
    disconnectWalletListeners()
    listenerBound.value = false
    resetState()
    ElMessage.success('钱包已断开连接')
  }

  // ==================== 事件监听 ====================

  /** 绑定 MetaMask 事件（仅绑定一次；disconnect 后重新连接时再绑定） */
  function bindListeners() {
    if (listenerBound.value) return
    onAccountsChanged(handleAccountsChanged)
    onChainChanged(handleChainChanged)
    listenerBound.value = true
  }

  /** 账户切换：更新地址；切走全部账户视为断开连接 */
  function handleAccountsChanged(accounts) {
    if (!accounts || accounts.length === 0) {
      resetState()
      ElMessage.warning('钱包账户已全部断开，请重新连接')
      return
    }
    const next = accounts[0]
    if (next !== address.value) {
      address.value = next
      ElMessage.info('已切换钱包账户：' + next.slice(0, 6) + '...' + next.slice(-4))
    }
  }

  /** 链切换：同步链 ID 并提示网络状态 */
  function handleChainChanged(hexId) {
    const id = parseInt(hexId, 16)
    chainId.value = id
    if (id === SEPOLIA.chainIdDecimal) {
      ElMessage.success('已切换到 Sepolia 测试网（11155111），钱包功能可用')
    } else {
      ElMessage.warning(`检测到网络已切换为链ID ${id}，请切回 Sepolia 测试网(11155111) 再使用钱包功能`)
    }
  }

  // ==================== 内部工具 ====================

  /** 重置连接状态（保留监听绑定标记由调用方管理） */
  function resetState() {
    isConnected.value = false
    address.value = ''
    chainId.value = 0
  }

  return {
    // 状态
    isConnected,
    address,
    chainId,
    connecting,
    // 计算属性
    shortAddress,
    isReady,
    // 动作
    connectAccount,
    disconnect,
    switchNetwork,
  }
})
