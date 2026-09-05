<template>
  <div class="home-page">
    <!-- 低碳粒子光影背景(氛围装饰) -->
    <ParticleBg :density="46" :opacity="0.2" />

    <!-- 首屏 Hero -->
    <section class="hero">
      <div class="hero-inner">
        <p class="eyebrow"><span class="eyebrow-icon"><Aim /></span>区块链碳积分可信交易平台</p>
        <h1>微碳链</h1>
        <p class="tagline">
          面向园区小微企业的碳资产可信管理原型系统。以本地模拟联盟链为底座，
          打通「能耗上报 → 碳积分核算 → 挂单交易 → 链上存证 → 监管溯源」的业务闭环。
        </p>
        <div class="hero-actions">
          <button class="btn btn-enter" @click="goSystem" :disabled="enterLoading">
            <span v-if="enterLoading" class="loading"><span class="spinner"></span>进入中...</span>
            <span v-else>进入系统 <Right /></span>
          </button>
        </div>
        <p class="hero-note">默认进入小微企业演示视图 · 登录后点击页面顶部"身份切换胶囊"即可在四个角色工作台间随时切换</p>
      </div>
    </section>

    <!-- 四大角色业务闭环：一图看懂系统内身份流转与协同 -->
    <section class="section">
      <div class="section-head">
        <h2>四大角色业务闭环</h2>
        <p class="section-sub">四个身份在同一可信账本上各司其职，环环相扣</p>
      </div>

      <div class="loop-board">
        <div class="loop-row">
          <template v-for="(r, idx) in roles" :key="r.name">
            <article class="loop-node">
              <span class="loop-icon" :style="{ background: r.tint, color: r.color }">
                <component :is="r.icon" />
              </span>
              <h3 class="loop-name">{{ r.name }}</h3>
              <p class="loop-act">{{ r.act }}</p>
              <span class="loop-owner">{{ r.account }}</span>
            </article>
            <span v-if="idx < roles.length - 1" class="loop-arrow">
              <component :is="r.outIcon || 'ArrowRight'" />
            </span>
          </template>
        </div>
        <!-- 闭环语义：监管核查监督全链，数据一路回归可信账本 -->
        <div class="loop-band">
          <Connection class="loop-band-icon" />
          <span>能耗上报 → 碳积分生成 → 园区统筹 → 撮合交易 → 上链存证，全程 SHA-256 哈希链支撑；监管核查对任意环节可溯源校验，形成可信闭环</span>
        </div>
      </div>
    </section>

    <!-- 核心能力 -->
    <section class="section">
      <div class="section-head">
        <h2>核心能力</h2>
        <p class="section-sub">从能耗数据到碳资产流转的完整支撑</p>
      </div>
      <div class="feature-grid">
        <article class="feature-card" v-for="(mod, idx) in modules" :key="idx">
          <span class="feature-icon" :style="{ color: mod.color, background: mod.tint }">
            <component :is="mod.icon" />
          </span>
          <h3>{{ mod.title }}</h3>
          <p>{{ mod.desc }}</p>
        </article>
      </div>
    </section>

    <!-- 技术底座 -->
    <section class="section section-tech">
      <div class="section-head">
        <h2>技术底座</h2>
      </div>
      <div class="tech-stack">
        <span class="tech-tag" v-for="(t, ti) in techStack" :key="ti">
          <CircleCheck class="tech-tick" />{{ t }}
        </span>
      </div>
      <p class="tech-note">本系统为演示原型：联盟链为服务端本地模拟实现(区块持久化至数据库，可平滑替换真实联盟链 SDK)；能耗数据须由企业用户在前端手动录入。</p>
    </section>

    <footer class="home-footer">
      <span>微碳链 · 区块链碳积分可信交易平台演示系统</span>
    </footer>
  </div>
</template>

<script setup>
// Home 门户首页：业务闭环导览 + 核心能力 + 技术底座
// 变更说明：原"平台角色/业务流转"两块合并为"四大角色业务闭环"关系卡，
// 角色展示名正式化；图标统一使用 ElementPlus 矢量图标。
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authAPI } from '../api/index.js'
import ParticleBg from '../components/ParticleBg.vue'

const router = useRouter()
const enterLoading = ref(false)

// 四大角色定义(展示名正式，无 001 后缀)
const roles = [
  { name: '小微企业', account: '小微企业001', icon: 'OfficeBuilding', tint: '#e6f7f5', color: '#0d9488', act: '手动上报企业能耗，自动核算后生成碳积分资产' },
  { name: '园区管理员', account: '园区管理员001', icon: 'Histogram', tint: '#e8f2fe', color: '#2563eb', act: '审核企业入驻，统筹园区碳排放总量与积分分布' },
  { name: '碳交易所', account: '碳交易所001', icon: 'TrendCharts', tint: '#fdf0e4', color: '#d97706', act: '发布卖方挂单，撮合成交实现碳积分权属流转' },
  { name: '监管核查', account: '监管核查001', icon: 'Monitor', tint: '#f0eefe', color: '#7c3aed', act: '监督链上全量存证，数据防篡改校验与平台风险监控' },
]

const modules = [
  {
    title: '能耗数据管理',
    desc: '企业在前端手动录入用电量、天然气与用水量，数据提交后进入核算流程并完成上链存证。',
    icon: 'Histogram', tint: '#e6f7f5', color: '#0d9488',
  },
  {
    title: '碳积分核算与交易',
    desc: '依据能耗数据自动核算碳排放量并生成碳积分，支持挂单出售、撮合成交与权属可信流转。',
    icon: 'Coin', tint: '#fdf0e4', color: '#d97706',
  },
  {
    title: '模拟联盟链存证',
    desc: '对所有关键业务数据计算 SHA-256 哈希并生成区块，支持按单号溯源查询与数据篡改校验。',
    icon: 'Link', tint: '#e8f2fe', color: '#2563eb',
  },
  {
    title: 'AI 减排建议与报告',
    desc: '对接 DeepSeek 大模型，基于历史能耗生成企业减排建议与园区低碳报告，异常自动降级不卡死。',
    icon: 'MagicStick', tint: '#f0eefe', color: '#7c3aed',
  },
]

const techStack = ['Go + Gin', 'Vue 3 + Vite', 'MySQL/SQLite', 'JWT', 'SHA-256', 'Merkle Root', 'DeepSeek API']

async function goSystem() {
  enterLoading.value = true
  try {
    // 自动登录小微企业账号，直接进入企业端演示视图
    const res = await authAPI.login({ username: '小微企业001', password: '123456' })
    localStorage.setItem('token', res.data.token)
    localStorage.setItem('user', JSON.stringify(res.data.user))
    router.push('/enterprise')
  } catch (e) {
    router.push('/login')
  } finally {
    enterLoading.value = false
  }
}
</script>

<style scoped>
.home-page {
  min-height: 100vh;
  background: transparent;
  position: relative;
}

/* ---------- Hero ---------- */
.hero {
  border-bottom: 1px solid rgba(229, 231, 235, 0.6);
  background: radial-gradient(1200px 480px at 50% -10%, rgba(204, 251, 241, 0.35), transparent 70%);
  padding: 84px 24px 60px;
  text-align: center;
  position: relative;
}
.hero-inner { max-width: 740px; margin: 0 auto; }
.eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--primary);
  background: rgba(240, 253, 250, 0.8);
  border: 1px solid #ccfbf1;
  border-radius: 999px;
  padding: 5px 14px;
  margin-bottom: 22px;
}
.eyebrow-icon { display: inline-flex; }
.eyebrow-icon svg { width: 13px; height: 13px; }
.hero h1 {
  font-size: 46px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--text-1);
}
.tagline {
  max-width: 600px;
  margin: 18px auto 0;
  font-size: 15px;
  line-height: 1.9;
  color: var(--text-3);
}
.hero-actions { margin-top: 32px; }
.btn-enter {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 12px 40px;
  font-size: 15px;
  font-weight: 500;
  border-radius: var(--radius-sm);
}
.btn-enter svg { width: 15px; height: 15px; }
.hero-note { margin-top: 14px; font-size: 12px; color: var(--text-4); }

/* ---------- 通用 Section ---------- */
.section { max-width: 1080px; margin: 0 auto; padding: 52px 24px 4px; position: relative; }
.section-head { margin-bottom: 24px; }
.section-head h2 { margin-bottom: 6px; font-size: 22px; }
.section-sub { font-size: 13px; color: var(--text-4); }

/* ---------- 四大角色业务闭环 ---------- */
.loop-board {
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid var(--line);
  border-radius: var(--radius-lg);
  padding: 26px 24px 20px;
  box-shadow: var(--shadow);
}
.loop-row {
  display: flex;
  align-items: flex-start;
  gap: 0;
  flex-wrap: nowrap;
}
.loop-node {
  flex: 1;
  min-width: 0;
  text-align: center;
  padding: 4px 6px;
}
.loop-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 14px;
  margin-bottom: 10px;
}
.loop-icon svg { width: 26px; height: 26px; }
.loop-name { margin: 0 0 6px; font-size: 15px; }
.loop-act {
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-3);
  margin: 0 auto 8px;
  max-width: 170px;
  min-height: 40px;
}
.loop-owner {
  display: inline-block;
  font-size: 11px;
  color: var(--text-4);
  background: #f8fafc;
  border: 1px dashed var(--line-strong);
  border-radius: 4px;
  padding: 2px 8px;
}
.loop-arrow {
  align-self: center;
  color: #94c6bf;
  display: inline-flex;
  margin: 30px 2px 0;
  flex-shrink: 0;
}
.loop-arrow svg { width: 18px; height: 18px; }
.loop-band {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 20px;
  padding: 12px 16px;
  background: linear-gradient(90deg, #f0fdfa, #f0f7ff);
  border: 1px solid #d9f0ec;
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: #3d5a57;
  line-height: 1.8;
}
.loop-band-icon { width: 16px; height: 16px; color: var(--primary); flex-shrink: 0; }

/* ---------- 核心能力 ---------- */
.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 16px;
}
.feature-card {
  background: rgba(255, 255, 255, 0.94);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  padding: 22px 22px 18px;
  box-shadow: var(--shadow-sm);
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}
.feature-card:hover { border-color: #99f6e4; box-shadow: var(--shadow-md); transform: translateY(-2px); }
.feature-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  margin-bottom: 12px;
}
.feature-icon svg { width: 19px; height: 19px; }
.feature-card h3 { margin: 0 0 8px; font-size: 15px; }
.feature-card p { font-size: 13px; line-height: 1.8; color: var(--text-3); margin: 0; }

/* ---------- 技术底座 ---------- */
.section-tech { padding-bottom: 8px; }
.tech-stack { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 18px; }
.tech-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--text-3);
  background: rgba(255, 255, 255, 0.94);
  border: 1px solid var(--line);
  border-radius: 4px;
  padding: 5px 12px;
}
.tech-tick { width: 13px; height: 13px; color: var(--success); }
.tech-note { font-size: 12px; color: var(--text-4); margin-top: 4px; }

/* Footer */
.home-footer {
  margin-top: 52px;
  padding: 24px;
  text-align: center;
  font-size: 12px;
  color: var(--text-4);
  border-top: 1px solid var(--line);
  background: rgba(250, 251, 252, 0.8);
}

/* 响应式 */
@media (max-width: 860px) {
  .hero { padding: 56px 20px 44px; }
  .hero h1 { font-size: 34px; }
  .section { padding: 40px 16px 4px; }
  .loop-row { flex-direction: column; }
  .loop-arrow { transform: rotate(90deg); margin: 2px 0; }
  .loop-act { min-height: 0; }
}
</style>
