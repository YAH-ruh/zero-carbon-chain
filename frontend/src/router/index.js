import { createRouter, createWebHashHistory } from 'vue-router'
import LayoutShell from '../components/layout/LayoutShell.vue'
// 采用 hash 路由：GitHub Pages 为静态托管，history 模式刷新子路径会 404；
// hash 模式(#/xxx)全程走 index.html，线上部署零配置可用。

/* ====== 公共页（不进 LayoutShell） ====== */
const Home      = () => import('../views/Home.vue')
const Login     = () => import('../views/Login.vue')
const Register  = () => import('../views/Register.vue')

/* ====== 业务页（进 LayoutShell，扁平化路由） ====== */
const Dashboard     = () => import('../views/dashboard.vue')
const DataV         = () => import('../views/datav.vue')
const EnergyReport  = () => import('../views/energy-report.vue')
const IoTDevices    = () => import('../views/iot-devices.vue')
const ZkpProof      = () => import('../views/zkp-proof.vue')
const CreditsCenter = () => import('../views/credits-center.vue')
const Exchange      = () => import('../views/exchange.vue')
const Pledge        = () => import('../views/pledge.vue')
const Archive       = () => import('../views/archive.vue')
const Footprint     = () => import('../views/footprint.vue')
const AiAgent       = () => import('../views/ai-agent.vue')
const Arbitration   = () => import('../views/arbitration.vue')
const RollupVerify  = () => import('../views/rollup-verify.vue')
const Incentive     = () => import('../views/incentive.vue')
const Permission    = () => import('../views/permission.vue')

/* 菜单权限矩阵：key -> 允许访问的角色数组 */
const ROLE_MENU = {
  'dashboard':      ['enterprise','park_admin','exchange','regulator'],
  'datav':          ['park_admin','regulator'],
  'energy-report':  ['enterprise'],
  'iot-devices':    ['enterprise'],
  'zkp-proof':      ['enterprise','regulator'],
  'credits-center': ['enterprise','park_admin','exchange','regulator'],
  'exchange':       ['exchange'],
  'pledge':         ['enterprise'],
  'archive':        ['exchange','regulator'],
  'footprint':      ['enterprise'],
  'ai-agent':       ['exchange','regulator'],
  'arbitration':    ['enterprise','exchange','regulator'],
  'rollup-verify':  ['regulator'],
  'incentive':      ['exchange','park_admin'],
  'permission':     ['regulator'],
}

/* 业务子路由表（扁平化，不嵌套角色前缀） */
const bizChildren = [
  { path: 'dashboard',      name: 'Dashboard',      component: Dashboard     },
  { path: 'datav',          name: 'DataV',          component: DataV,          meta: { fullscreen: true } },
  { path: 'energy-report',  name: 'EnergyReport',   component: EnergyReport  },
  { path: 'iot-devices',    name: 'IoTDevices',     component: IoTDevices    },
  { path: 'zkp-proof',      name: 'ZkpProof',       component: ZkpProof      },
  { path: 'credits-center', name: 'CreditsCenter',  component: CreditsCenter },
  { path: 'exchange',       name: 'ExchangePage',   component: Exchange      },
  { path: 'pledge',         name: 'Pledge',         component: Pledge        },
  { path: 'archive',        name: 'Archive',        component: Archive       },
  { path: 'footprint',      name: 'Footprint',      component: Footprint     },
  { path: 'ai-agent',       name: 'AiAgent',        component: AiAgent       },
  { path: 'arbitration',    name: 'Arbitration',    component: Arbitration   },
  { path: 'rollup-verify',  name: 'RollupVerify',   component: RollupVerify  },
  { path: 'incentive',      name: 'Incentive',      component: Incentive     },
  { path: 'permission',     name: 'Permission',     component: Permission    },
]

const routes = [
  { path: '/', name: 'Home', component: Home, meta: { noAuth: true } },
  { path: '/login', name: 'Login', component: Login, meta: { noAuth: true } },
  { path: '/register', name: 'Register', component: Register, meta: { noAuth: true } },

  {
    path: '/',
    component: LayoutShell,
    children: [
      ...bizChildren,
      // 默认跳仪表盘
      { path: '', redirect: '/dashboard' }
    ]
  },

  // 兜底：跳到仪表盘
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

/* 路由守卫：登录状态 + 角色权限 */
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  const userStr = localStorage.getItem('user')

  if (to.meta.noAuth) {
    if (token && to.path === '/login') return next('/dashboard')
    return next()
  }
  if (!token) return next('/login')

  // 业务页角色权限校验
  const pathKey = to.path.replace(/^\//, '')
  const allowed = ROLE_MENU[pathKey]
  if (allowed) {
    try {
      const user = JSON.parse(userStr || '{}')
      if (!allowed.includes(user.role)) {
        // 无权限：回到仪表盘（LayoutShell 会按 user.role 渲染对应菜单）
        return next('/dashboard')
      }
    } catch {
      return next('/login')
    }
  }

  next()
})

export default router
