<!--
  WalletModal.vue
  钱包账户连接弹窗 —— 仿 MetaMask 原生授权弹窗形式（点击【连接钱包】后弹出）

  连接链路（一步直连，无二次选择）：
    1) 点击「连接」→ wallet_requestPermissions 唤起 MetaMask 原生账户勾选弹窗
       （列出全部导入账户，可自主勾选）→
       - 只勾选 1 个账户：直接完成连接，弹窗关闭
       - 勾选多个账户：弹窗内快速切换勾选目标后连接
    2) 连接时若 MetaMask 不在 Sepolia 测试网(链ID 11155111)，自动唤起切换/添加网络弹窗
    3) 未检测到插件：弹窗内引导安装 MetaMask（Sepolia 为公链，账户只能经插件授权获取）
-->
<template>
  <Teleport to="body">
    <transition name="wm-fade">
      <div v-if="modelValue" class="wm-mask" @click.self="close">
        <div class="wm-card">
          <!-- 头部：狐狸徽标 + 网络名 + 意图说明 -->
          <div class="wm-hero">
            <span class="wm-fox-badge">🦊</span>
            <h3>{{ SEPOLIA.chainName }}</h3>
            <p>此网站想要：</p>
          </div>

          <!-- 账户 / 权限 页签 -->
          <div class="wm-tabs">
            <button class="wm-tab" :class="{ active: tab === 'account' }" @click="tab = 'account'">账户</button>
            <button class="wm-tab" :class="{ active: tab === 'perm' }" @click="tab = 'perm'">权限</button>
          </div>

          <!-- ===== 账户页 ===== -->
          <div v-show="tab === 'account'" class="wm-body">
            <!-- 待授权：提示点击连接（单账户勾选后立即完成，无二次选择） -->
            <div v-if="providerOk && phase === 'idle'" class="wm-idle">
              <p>点击「连接」，在 MetaMask 弹窗中勾选要使用的账户</p>
              <p class="wm-idle-sub">全部导入账户均可选择；仅勾选一个时将直接完成连接</p>
            </div>

            <!-- 未安装插件：引导安装（Sepolia 公链无本地账户枚举能力） -->
            <div v-else-if="!providerOk" class="wm-idle">
              <p>未检测到 MetaMask 插件</p>
              <p class="wm-idle-sub">
                请先安装 MetaMask 扩展并创建/导入钱包账户，
                <a href="https://metamask.io/download/" target="_blank" rel="noopener">前往安装 →</a>
              </p>
            </div>

            <!-- 多账户勾选时：快速切换目标 -->
            <template v-else>
              <div class="wm-acc-head">
                <span>选择账户</span>
                <span class="wm-acc-status ok">MetaMask 已授权 · 请选择要使用的账户</span>
              </div>

              <div class="wm-acc-list">
                  <button
                    v-for="(acc, i) in accounts"
                    :key="acc.address"
                    class="wm-acc"
                    :class="{ active: selected === acc.address }"
                    @click="selected = acc.address"
                  >
                    <span class="wm-avatar" :style="{ background: avatarTint(i) }">{{ i + 1 }}</span>
                    <span class="wm-acc-info">
                      <b>账户 {{ i }}</b>
                      <i class="mono">{{ short(acc.address) }}</i>
                    </span>
                    <span class="wm-acc-right">
                      <em>{{ acc.balance ?? '--' }} ETH</em>
                      <span class="wm-check" :class="{ on: selected === acc.address }">✓</span>
                    </span>
                  </button>
                  <div v-if="!accounts.length" class="wm-empty">未获取到账户，请重试</div>
              </div>
            </template>
          </div>

          <!-- ===== 权限页 ===== -->
          <div v-show="tab === 'perm'" class="wm-body">
            <div class="wm-net">
              <div class="wm-net-row"><span>网络名称</span><b>{{ SEPOLIA.chainName }}</b></div>
              <div class="wm-net-row"><span>RPC 地址</span><b class="mono">{{ SEPOLIA.rpcUrl }}</b></div>
              <div class="wm-net-row"><span>链 ID</span><b>{{ SEPOLIA.chainIdDecimal }}</b></div>
              <div class="wm-net-row"><span>代币符号</span><b>{{ SEPOLIA.nativeCurrency.symbol }}</b></div>
              <div class="wm-net-row"><span>区块浏览器</span><b class="mono">{{ SEPOLIA.explorer }}</b></div>
              <div class="wm-net-row"><span>请求权限</span><b>查看账户地址与 ETH 余额</b></div>
            </div>
          </div>

          <!-- 底部：取消 / 连接（MetaMask 式胶囊按钮） -->
          <div class="wm-foot">
            <button class="wm-btn cancel" @click="close">取消</button>
            <button class="wm-btn connect" :disabled="!canConnect || busy" @click="mainAction">
              {{ busy ? '连接中...' : '连接' }}
            </button>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  SEPOLIA,
  connectWalletAccounts,
  getChainId,
  switchToSepolia,
  formatWalletError,
  getNativeBalance,
} from '../utils/web3.js'

const props = defineProps({
  /** 弹窗显隐（v-model） */
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'connect'])

const tab = ref('account')     // 页签：account | perm
const providerOk = ref(false)  // 是否检测到 MetaMask 插件
const phase = ref('idle')      // idle=待授权 → list=多账户勾选后的快速选择（单账户不经过此阶段）
const mmConnecting = ref(false)
const confirming = ref(false)
const accounts = ref([])       // 多账户勾选时的候选列表
const selected = ref('')       // 当前勾选的账户地址

const busy = computed(() => mmConnecting.value || confirming.value)
const canConnect = computed(() => (providerOk.value && phase.value === 'idle') || !!selected.value)

// 每次打开弹窗重置状态；未检测到插件时展示安装引导（Sepolia 公链无本地账户枚举能力）
watch(() => props.modelValue, (open) => {
  if (!open) return
  tab.value = 'account'
  selected.value = ''
  providerOk.value = !!window.ethereum
  phase.value = 'idle'
})

/**
 * MetaMask 连接：唤起原生账户勾选弹窗（全部导入账户可选）
 * 仅勾选 1 个 → 直接完成连接；勾选多个 → 弹窗内快速切换后连接
 */
async function doMetaMaskConnect() {
  if (mmConnecting.value) return
  mmConnecting.value = true
  try {
    const addrList = await connectWalletAccounts()
    if (addrList.length === 1) {
      await finishConnect(addrList[0])
    } else {
      accounts.value = await Promise.all(
        addrList.map(async (addr) => {
          let balance = null
          try { balance = await getNativeBalance(addr) } catch { /* 余额读取失败置空 */ }
          return { address: addr, balance }
        })
      )
      selected.value = addrList[0]
      phase.value = 'list'
    }
  } catch (err) {
    ElMessage.error(formatWalletError(err))
  } finally {
    mmConnecting.value = false
  }
}

/** 完成连接：确保网络为 Sepolia 测试网(11155111)（必要时唤起切换/添加网络弹窗）后通知父组件 */
async function finishConnect(addr) {
  const cid = await getChainId()
  if (cid !== SEPOLIA.chainIdDecimal) {
    try {
      await switchToSepolia() // wallet_switchEthereumChain，未收录时自动提交添加(4902)
    } catch (err) {
      ElMessage.error(formatWalletError(err) + `；也可在 MetaMask 中手动添加网络：RPC ${SEPOLIA.rpcUrl}，链ID ${SEPOLIA.chainIdDecimal}`)
      return
    }
  }
  emit('connect', addr)
  close()
}

function short(addr) {
  return addr.slice(0, 6) + '...' + addr.slice(-4)
}

/** 账户头像底色（浅色系循环） */
const TINTS = ['#e6f7f5', '#e8f2fe', '#fdf0e4', '#f0eefe', '#fdecec']
function avatarTint(i) {
  return TINTS[i % TINTS.length]
}

function close() {
  emit('update:modelValue', false)
}

/** 底部主按钮：待授权 → 唤起 MetaMask 勾选弹窗；多账户候选 → 确认选定账户 */
function mainAction() {
  if (busy.value) return
  if (providerOk.value && phase.value === 'idle') {
    doMetaMaskConnect()
    return
  }
  if (!selected.value) return
  confirming.value = true
  finishConnect(selected.value)
    .catch((err) => ElMessage.error(formatWalletError(err)))
    .finally(() => { confirming.value = false })
}
</script>

<style scoped>
/* ==================== 遮罩与容器 ==================== */
.wm-mask {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.55);
  backdrop-filter: blur(3px);
}
.wm-card {
  width: 380px;
  max-width: calc(100vw - 32px);
  max-height: calc(100vh - 64px);
  overflow: auto;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 20px 50px rgba(15, 23, 42, 0.25);
  animation: wm-pop 0.22s ease-out;
}
@keyframes wm-pop {
  from { transform: translateY(12px) scale(0.97); opacity: 0; }
  to { transform: translateY(0) scale(1); opacity: 1; }
}
.wm-fade-enter-active, .wm-fade-leave-active { transition: opacity 0.18s ease; }
.wm-fade-enter-from, .wm-fade-leave-to { opacity: 0; }
.mono { font-family: 'Consolas', 'Monaco', monospace; font-style: normal; }

/* ==================== 头部（MetaMask 式居中布局） ==================== */
.wm-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 22px 20px 12px;
  text-align: center;
}
.wm-fox-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  border-radius: 50%;
  font-size: 24px;
  background: #e8f4ff;
  border: 1px solid #d5e9fb;
}
.wm-hero h3 { margin: 10px 0 0; font-size: 19px; color: #101828; }
.wm-hero p { margin: 4px 0 0; font-size: 13px; color: #98a2b3; }

/* ==================== 页签（账户 / 权限） ==================== */
.wm-tabs {
  display: flex;
  border-bottom: 1px solid #eaecf0;
  padding: 0 24px;
}
.wm-tab {
  position: relative;
  padding: 10px 4px 12px;
  margin-right: 28px;
  border: none;
  background: transparent;
  font-size: 14px;
  font-weight: 600;
  color: #98a2b3;
  cursor: pointer;
}
.wm-tab.active { color: #101828; }
.wm-tab.active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 2px;
  border-radius: 2px;
  background: #101828;
}

/* ==================== 主体 ==================== */
.wm-body { min-height: 168px; padding: 14px 20px 0; }
.wm-idle { padding: 26px 8px; text-align: center; }
.wm-idle p { margin: 0; font-size: 13px; color: #475467; }
.wm-idle .wm-idle-sub { margin-top: 6px; font-size: 12px; color: #98a2b3; }

.wm-acc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
  color: #101828;
}
.wm-acc-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 999px;
}
.wm-acc-status.ok { color: #059669; background: rgba(5, 150, 105, 0.1); }
.wm-acc-status.mock { color: #d97706; background: rgba(245, 158, 11, 0.12); }
.wm-refresh { cursor: pointer; }
.wm-refresh:hover { color: #16a34a; }
.wm-refresh.spin { animation: wm-rotate 0.9s linear infinite; }
@keyframes wm-rotate { to { transform: rotate(360deg); } }

.wm-acc-list {
  border: 1px solid #eaecf0;
  border-radius: 12px;
  overflow: hidden;
}
.wm-acc {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 11px 13px;
  border: none;
  border-bottom: 1px solid #eaecf0;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
}
.wm-acc:last-child { border-bottom: none; }
.wm-acc:hover { background: #f7f9fb; }
.wm-acc.active { background: #f0f7f3; }
.wm-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 700;
  color: #344054;
  flex-shrink: 0;
}
.wm-acc-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
.wm-acc-info b { font-size: 13px; color: #101828; font-weight: 600; }
.wm-acc-info i { font-size: 12px; color: #98a2b3; }
.wm-acc-right { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.wm-acc-right em { font-size: 12px; color: #475467; font-style: normal; }
.wm-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  font-size: 11px;
  color: transparent;
  border: 1.5px solid #d0d5dd;
  transition: all 0.15s;
}
.wm-check.on { color: #fff; background: #101828; border-color: #101828; }

.wm-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 26px 0;
  font-size: 13px;
  color: #98a2b3;
}
.wm-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid #eaecf0;
  border-top-color: #16a34a;
  border-radius: 50%;
  animation: wm-rotate 0.8s linear infinite;
}
.wm-empty { padding: 26px 0; text-align: center; font-size: 13px; color: #98a2b3; }

/* ==================== 权限页（网络信息） ==================== */
.wm-net {
  padding: 4px 14px;
  border: 1px solid #eaecf0;
  border-radius: 12px;
  background: #f9fafb;
}
.wm-net-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 0;
  font-size: 13px;
}
.wm-net-row span { color: #98a2b3; flex-shrink: 0; }
.wm-net-row b { color: #101828; font-weight: 600; word-break: break-all; text-align: right; }

/* ==================== 底部胶囊按钮（MetaMask 式） ==================== */
.wm-foot {
  display: flex;
  gap: 12px;
  padding: 16px 20px 20px;
}
.wm-btn {
  flex: 1;
  padding: 11px 0;
  border-radius: 999px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}
.wm-btn.cancel { color: #101828; background: #f2f4f7; }
.wm-btn.cancel:hover { background: #e5e7eb; }
.wm-btn.connect { color: #fff; background: #101828; }
.wm-btn.connect:hover:not(:disabled) { background: #293056; }
.wm-btn.connect:disabled { opacity: 0.45; cursor: not-allowed; }
</style>
