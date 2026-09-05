<template>
  <div class="register-page">
    <ParticleBg :density="34" :opacity="0.15" />
    <div class="register-card">
      <div class="register-header">
        <div class="register-logo">
          <svg width="40" height="40" viewBox="0 0 44 44" fill="none">
            <circle cx="22" cy="22" r="22" fill="url(#rg1)" />
            <path d="M22 10c-6.627 0-12 5.373-12 12s5.373 12 12 12 12-5.373 12-12-5.373-12-12-12z" fill="rgba(255,255,255,0.2)" />
            <path d="M22 15c-3.866 0-7 3.134-7 7s3.134 7 7 7 7-3.134 7-7-3.134-7-7-7z" fill="rgba(255,255,255,0.35)" />
            <defs><linearGradient id="rg1" x1="0" y1="0" x2="44" y2="44"><stop offset="0%" stop-color="#0f766e"/><stop offset="100%" stop-color="#10b981"/></linearGradient></defs>
          </svg>
        </div>
        <h1>注册企业账号</h1>
        <p>创建小微企业角色账号，参与碳积分交易</p>
      </div>

      <form @submit.prevent="handleRegister" class="form">
        <div class="form-group">
          <label>用户名</label>
          <input v-model="regForm.username" placeholder="请输入用户名" required />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="regForm.password" type="password" placeholder="请输入密码" required />
        </div>
        <div class="form-group">
          <label>企业名称</label>
          <input v-model="regForm.company" placeholder="请输入企业名称" required />
        </div>
        <div class="form-group">
          <label>所属园区ID</label>
          <input v-model.number="regForm.park_id" type="number" placeholder="默认为1" />
        </div>
        <p v-if="regError" class="error">{{ regError }}</p>
        <p v-if="regSuccess" class="success">{{ regSuccess }}</p>
        <button type="submit" class="btn-primary" :disabled="loading">
          <span v-if="loading" class="loading"><span class="spinner"></span>注册中...</span>
          <span v-else>提交注册</span>
        </button>
      </form>

      <div class="register-footer">
        <p>已有账号？<a href="/login">前往登录</a></p>
        <p class="foot-note">高权限账号（园区管理员、碳交易所、监管核查）仅通过系统预置</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { authAPI } from '../api/index.js'
import ParticleBg from '../components/ParticleBg.vue'

const loading = ref(false)
const regError = ref('')
const regSuccess = ref('')

const regForm = reactive({ username: '', password: '', company: '', park_id: 1 })

async function handleRegister() {
  loading.value = true; regError.value = ''; regSuccess.value = ''
  try {
    await authAPI.register({
      username: regForm.username,
      password: regForm.password,
      company: regForm.company,
      role: 'enterprise',
      park_id: regForm.park_id
    })
    regSuccess.value = '注册成功！请前往登录页登录'
    regForm.username = ''; regForm.password = ''; regForm.company = ''
  } catch (e) {
    regError.value = e?.msg || '注册失败'
  } finally { loading.value = false }
}
</script>

<style scoped>
.register-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 24px;
  background: radial-gradient(900px 420px at 20% -10%, rgba(204, 251, 241, 0.5), transparent 60%),
              radial-gradient(900px 420px at 110% 110%, rgba(219, 234, 254, 0.4), transparent 60%);
}
.register-card {
  width: 420px;
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
.register-header {
  text-align: center;
  padding: 32px 24px 16px;
}
.register-logo { margin-bottom: 10px; }
.register-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--text-1);
}
.register-header p {
  margin: 6px 0 0;
  color: var(--text-3);
  font-size: 13px;
}
.form { padding: 4px 24px 20px; }
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
.success { color: var(--success); font-size: 13px; margin: 8px 0; }
.register-footer {
  text-align: center;
  padding: 14px 24px;
  border-top: 1px solid #f1f5f9;
  background: #fafbfc;
}
.register-footer p { margin: 3px 0; font-size: 13px; color: var(--text-3); }
.register-footer a { color: var(--primary); text-decoration: none; font-weight: 500; }
.register-footer a:hover { text-decoration: underline; }
.foot-note { font-size: 12px !important; color: var(--text-4) !important; }
</style>