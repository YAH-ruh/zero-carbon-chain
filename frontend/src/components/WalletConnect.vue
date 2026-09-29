<!--
  WalletConnect.vue
  通用【连接钱包】组件 —— 4 个链上钱包页面共用（需求交互规则 1/6）

  使用方式：
    <WalletConnect hint="碳积分资产中心 · 链上钱包" @connected="loadBalance" />
    <WalletConnect compact />  ← 紧凑模式，适合放页面右上角工具条

  交互说明：
    - 页面渲染时绝不自动弹窗；仅当用户点击【连接钱包】按钮时弹出
      钱包账户弹窗(WalletModal)：展示网络信息(Sepolia 测试网 · 链ID 11155111 · 代币 ETH)
      与账户列表，选定账户后完成连接
    - 弹窗账户来自 MetaMask 授权（wallet_requestPermissions，全部导入账户可选）
    - 连接成功后展示钱包地址 + 网络状态；非 11155111 网络时展示"网络不匹配"并提供切换按钮
    - 错误提示（用户取消/待处理弹窗等）由 wallet store 统一弹出
-->
<template>
  <div class="wallet-bar" :class="{ compact }">
    <!-- 左侧：说明文案（可被父组件插槽覆盖） -->
    <div class="wallet-left">
      <slot name="left">
        <span class="wallet-title">
          <el-icon :size="14"><Coin /></el-icon>
          {{ hint }}
        </span>
      </slot>
    </div>

    <!-- 右侧：钱包状态区 -->
    <div class="wallet-right">
      <!-- 状态一：未连接 → 手动连接按钮（点击弹出账户弹窗） -->
      <template v-if="!wallet.isConnected">
        <span class="net-tip">Sepolia 测试网 · 链ID 11155111</span>
        <button class="wc-btn primary" :disabled="wallet.connecting" @click="showModal = true">
          <el-icon :size="14"><Link /></el-icon>
          连接钱包
        </button>
      </template>

      <!-- 状态二：已连接但网络不匹配 → 地址 + 切网按钮 -->
      <template v-else-if="!wallet.isReady">
        <span class="dot warn"></span>
        <span class="addr" :title="wallet.address" @click="copyAddress">{{ wallet.shortAddress }}</span>
        <span class="net-badge bad">网络不匹配</span>
        <button class="wc-btn warn-btn" @click="wallet.switchNetwork()">
          <el-icon :size="14"><Switch /></el-icon>
          切换 Sepolia 测试网
        </button>
        <button class="wc-btn ghost" @click="handleDisconnect">断开</button>
      </template>

      <!-- 状态三：已连接且在 Sepolia 测试网 → 正常展示 -->
      <template v-else>
        <span class="dot ok"></span>
        <span class="addr" :title="'点击复制完整地址：' + wallet.address" @click="copyAddress">
          {{ wallet.shortAddress }}
        </span>
        <span class="net-badge good">Sepolia · 11155111</span>
        <button class="wc-btn ghost" @click="handleDisconnect">
          <el-icon :size="14"><Close /></el-icon>
          断开
        </button>
      </template>
    </div>

    <!-- 钱包账户弹窗：选定账户后完成连接 -->
    <WalletModal v-model="showModal" @connect="handleConnect" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Coin, Link, Switch, Close } from '@element-plus/icons-vue'
import { useWalletStore } from '../stores/wallet.js'
import WalletModal from './WalletModal.vue'

const props = defineProps({
  /** 左侧说明文案（slot 可覆盖） */
  hint: { type: String, default: '链上钱包' },
  /** 紧凑模式：去掉边框背景，适配嵌入页面顶部工具条 */
  compact: { type: Boolean, default: false },
})

/** 连接成功时通知父页面（如自动刷新链上余额） */
const emit = defineEmits(['connected', 'disconnected'])

const wallet = useWalletStore()
const showModal = ref(false)

/** 弹窗中选定账户 → 写入连接状态 → 通知父页面刷新 */
function handleConnect(addr) {
  wallet.connectAccount(addr)
  emit('connected')
}

// 复制完整钱包地址（演示时方便核对手动输入场景）
async function copyAddress() {
  try {
    await navigator.clipboard.writeText(wallet.address)
    ElMessage.success('钱包地址已复制到剪贴板')
  } catch {
    ElMessage.warning('复制失败，请手动选择地址复制')
  }
}

// 断开连接（前端语义断开，真正的授权解除需在钱包扩展中操作）
function handleDisconnect() {
  wallet.disconnect()
  emit('disconnected')
}
</script>

<style scoped>
/* ==================== 容器 ==================== */
.wallet-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(22, 163, 74, 0.06);
}
.wallet-bar.compact {
  padding: 0;
  background: transparent;
  border: none;
  box-shadow: none;
}

/* ==================== 左侧说明 ==================== */
.wallet-left { display: flex; align-items: center; min-width: 0; }
.wallet-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}
.wallet-title .el-icon { color: var(--primary-green); }

/* ==================== 右侧状态区 ==================== */
.wallet-right { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.net-tip { font-size: 12px; color: var(--text-tertiary); }

/* 连接状态圆点：绿色脉冲=正常 / 橙色=网络不匹配 */
.dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.dot.ok { background: var(--success); box-shadow: 0 0 0 0 rgba(5, 150, 105, 0.5); animation: pulse 2s infinite; }
.dot.warn { background: var(--warning); }
@keyframes pulse {
  0% { box-shadow: 0 0 0 0 rgba(5, 150, 105, 0.45); }
  70% { box-shadow: 0 0 0 6px rgba(5, 150, 105, 0); }
  100% { box-shadow: 0 0 0 0 rgba(5, 150, 105, 0); }
}

/* 钱包地址（可点击复制） */
.addr {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  color: var(--text-primary);
  cursor: pointer;
  padding: 3px 8px;
  border-radius: 6px;
  background: var(--bg-tertiary);
  transition: all 0.2s;
}
.addr:hover { color: var(--primary-green); background: var(--card-bg-hover); }

/* 网络徽标 */
.net-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  font-weight: 500;
}
.net-badge.good { color: var(--success); background: rgba(5, 150, 105, 0.1); border: 1px solid rgba(5, 150, 105, 0.25); }
.net-badge.bad { color: var(--danger); background: rgba(220, 38, 38, 0.08); border: 1px solid rgba(220, 38, 38, 0.25); }

/* ==================== 按钮（渐变绿主按钮 + 幽灵按钮） ==================== */
.wc-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 7px 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}
.wc-btn.primary {
  color: #fff;
  background: linear-gradient(135deg, var(--primary-green), var(--primary-green-light));
  box-shadow: 0 2px 10px rgba(22, 163, 74, 0.3);
}
.wc-btn.primary:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 4px 14px rgba(22, 163, 74, 0.4); }
.wc-btn.primary:disabled { opacity: 0.7; cursor: not-allowed; }

.wc-btn.warn-btn { color: #fff; background: linear-gradient(135deg, #f59e0b, #fbbf24); box-shadow: 0 2px 10px rgba(245, 158, 11, 0.3); }
.wc-btn.warn-btn:hover { transform: translateY(-1px); }

.wc-btn.ghost { color: var(--text-secondary); background: transparent; border: 1px solid var(--line-color); }
.wc-btn.ghost:hover { color: var(--danger); border-color: var(--danger); background: rgba(220, 38, 38, 0.04); }
</style>
