import { createRouter, createWebHistory } from 'vue-router'

// 公共页(无需登录)
const Home = () => import('../views/Home.vue')
const Login = () => import('../views/Login.vue')
const Register = () => import('../views/Register.vue')

// 四角色独立首页面板(各自统计指标/图表/业务模块完全独立)
const EnterpriseHome = () => import('../views/EnterpriseHome.vue')
const ParkAdminHome = () => import('../views/ParkAdminHome.vue')
const ExchangeHome = () => import('../views/ExchangeHome.vue')
const RegulatorHome = () => import('../views/RegulatorHome.vue')

// 四角色业务功能页(作为各角色首页下的"子功能子路由"，内部按钮跳转进入)
const Enterprise = () => import('../views/Enterprise.vue')
const ParkAdmin = () => import('../views/ParkAdmin.vue')
const Exchange = () => import('../views/Exchange.vue')
const Regulator = () => import('../views/Regulator.vue')

// 各角色合法子功能标识(进入不存在的子路由则回退到该角色首页)
const ROLE_FEATURES = {
  enterprise: ['energy', 'credits', 'sell', 'chain', 'advice'],
  park_admin: ['enterprises', 'report'],
  exchange: ['orders', 'match', 'verify', 'history'],
  regulator: ['chain', 'verify', 'alert', 'users']
}

const routes = [
  { path: '/', name: 'Home', component: Home, meta: { noAuth: true } },
  { path: '/login', name: 'Login', component: Login, meta: { noAuth: true } },
  { path: '/register', name: 'Register', component: Register, meta: { noAuth: true } },

  // ============ 小微企业 ============
  { path: '/enterprise', name: 'EnterpriseHome', component: EnterpriseHome, meta: { role: 'enterprise' } },
  { path: '/enterprise/:feature', name: 'EnterpriseFeature', component: Enterprise, meta: { role: 'enterprise' } },

  // ============ 园区管理员 ============
  { path: '/park-admin', name: 'ParkAdminHome', component: ParkAdminHome, meta: { role: 'park_admin' } },
  { path: '/park-admin/:feature', name: 'ParkAdminFeature', component: ParkAdmin, meta: { role: 'park_admin' } },

  // ============ 碳交易所 ============
  { path: '/exchange', name: 'ExchangeHome', component: ExchangeHome, meta: { role: 'exchange' } },
  { path: '/exchange/:feature', name: 'ExchangeFeature', component: Exchange, meta: { role: 'exchange' } },

  // ============ 监管核查 ============
  { path: '/regulator', name: 'RegulatorHome', component: RegulatorHome, meta: { role: 'regulator' } },
  { path: '/regulator/:feature', name: 'RegulatorFeature', component: Regulator, meta: { role: 'regulator' } }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

// 路由守卫：登录状态 / 角色权限 / 子功能合法性
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  const userStr = localStorage.getItem('user')

  if (to.meta.noAuth) {
    if (token && to.path === '/login') return next(getDefaultRoute(userStr))
    return next()
  }
  if (!token) return next('/login')

  if (to.meta.role) {
    try {
      const user = JSON.parse(userStr || '{}')
      // 角色不匹配：回到当前用户角色首页(而非放行，避免越权页面)
      if (user.role !== to.meta.role) return next(getDefaultRoute(userStr))
      // 子功能标识校验：未收录时回退该角色首页
      if (to.params.feature) {
        const allowed = ROLE_FEATURES[user.role] || []
        if (!allowed.includes(to.params.feature)) {
          return next({ path: getDefaultRoute(userStr), query: to.query })
        }
      }
    } catch {
      return next('/login')
    }
  }

  next()
})

function getDefaultRoute(userStr) {
  try {
    const user = JSON.parse(userStr || '{}')
    const roleMap = {
      enterprise: '/enterprise',
      park_admin: '/park-admin',
      exchange: '/exchange',
      regulator: '/regulator'
    }
    return roleMap[user.role] || '/login'
  } catch {
    return '/login'
  }
}

export default router
