<template>
  <div class="login-page">
    <!-- 低碳粒子光影背景 -->
    <ParticleBg :density="40" :opacity="0.16" />
    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">
          <img src="/favicon.svg" alt="零碳微证标识" width="48" height="48" />
        </div>
        <h1>零碳微证</h1>
        <p class="login-desc">区块链碳积分可信交易平台 · 小微企业 / 园区管理员 / 碳交易所 / 监管核查</p>
      </div>

      <form @submit.prevent="handleLogin" class="form">
        <div class="form-group">
          <label>用户名</label>
          <input v-model="loginForm.username" placeholder="请输入用户名" required />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="loginForm.password" type="password" placeholder="请输入密码" required />
        </div>
        <p v-if="loginError" class="error">{{ loginError }}</p>
        <button type="submit" class="btn-primary" :disabled="loading">
          <span v-if="loading" class="loading"><span class="spinner"></span>登录中...</span>
          <span v-else class="btn-label"><Right /> 登录</span>
        </button>
      </form>

      <div class="login-footer">
        <p>还没有账号？<a href="/register">前往注册（小微企业）</a></p>
        <p class="account-hint">预置演示账号：绿恒节能科技有限公司 / 园区管理员 / 碳交易所 / 监管核查（账号密码均为 123456）</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { authAPI } from '../api/index.js'
import { ROLE_ROUTES } from '../utils/roleSwitch.js'
import ParticleBg from '../components/ParticleBg.vue'

const router = useRouter()
const loading = ref(false)
const loginError = ref('')

const loginForm = reactive({ username: '绿恒节能科技有限公司', password: '123456' })

async function handleLogin() {
  loading.value = true; loginError.value = ''
  try {
    const res = await authAPI.login(loginForm)
    localStorage.setItem('token', res.data.token)
    localStorage.setItem('user', JSON.stringify(res.data.user))
    const path = ROLE_ROUTES[res.data.user.role] || '/dashboard'
    router.push(path)
  } catch (e) {
    loginError.value = e?.msg || '登录失败'
  } finally { loading.value = false }
}
</script>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 24px;
  background: radial-gradient(900px 420px at 20% -10%, rgba(204, 251, 241, 0.5), transparent 60%),
              radial-gradient(900px 420px at 110% 110%, rgba(219, 234, 254, 0.4), transparent 60%);
}
.btn-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  justify-content: center;
}
.btn-label svg { width: 15px; height: 15px; }
.login-card {
  width: 400px;
  max-width: 100%;
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
  animation: cardIn 0.3s ease;
}
@keyframes cardIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
.login-header {
  text-align: center;
  padding: 36px 24px 20px;
}
.login-logo { margin-bottom: 12px; }
.login-logo img { border-radius: 11px; box-shadow: 0 6px 18px rgba(5, 150, 105, 0.28); }
.login-header h1 {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--text-1);
}
.login-desc {
  margin: 6px 0 0;
  color: var(--text-3);
  font-size: 13px;
}
.form { padding: 4px 24px 24px; }
.form-group { margin-bottom: 16px; }
.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-2);
}
.form-group input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--line-strong);
  border-radius: var(--radius-sm);
  font-size: 14px;
  box-sizing: border-box;
  transition: all 0.15s ease;
  background: var(--surface);
}
.form-group input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgba(13, 148, 136, 0.12);
}
.btn-primary {
  width: 100%;
  padding: 11px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--primary);
  color: #fff;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, box-shadow 0.15s ease;
  margin-top: 6px;
}
.btn-primary:hover {
  background: var(--primary-strong);
  box-shadow: var(--shadow-md);
}
.btn-primary:disabled {
  opacity: 0.55;
  cursor: not-allowed;
  box-shadow: none;
}
.error { color: var(--danger); font-size: 13px; margin: 8px 0; }
.login-footer {
  text-align: center;
  padding: 14px 24px;
  border-top: 1px solid #f1f5f9;
  background: #fafbfc;
}
.login-footer p { margin: 3px 0; font-size: 13px; color: var(--text-3); }
.login-footer a { color: var(--primary); text-decoration: none; font-weight: 500; }
.login-footer a:hover { text-decoration: underline; }
.account-hint { font-size: 12px !important; color: var(--text-4) !important; }
</style>