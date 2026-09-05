// utils/roleSwitch.js
// 页面内"跳转其他角色面板"的辅助函数：
// 使用预置演示账号静默重登，再交给调用方完成 vue-router 跳转，
// 从而保证目标角色页面调用后端接口时权限(角色JWT)正确、数据可正常加载。
// 说明：仅保留 4 个预置账号具备切换能力，账号标识(用户名含编号)与后端鉴权逻辑不变。
import { authAPI } from '../api/index.js'

// 预置账号：仅这 4 个支持演示期全角色体验
const PRESET_ACCOUNTS = {
  enterprise: { username: '小微企业001', password: '123456' },
  park_admin: { username: '园区管理员001', password: '123456' },
  exchange: { username: '碳交易所001', password: '123456' },
  regulator: { username: '监管核查001', password: '123456' }
}

// 各角色基础首页地址(与路由/守卫约定一致)
export const ROLE_ROUTES = {
  enterprise: '/enterprise',
  park_admin: '/park-admin',
  exchange: '/exchange',
  regulator: '/regulator'
}

// 角色展示元信息(供各角色首页渲染“其他角色入口”使用)
export const ROLE_META = [
  { key: 'enterprise', name: '小微企业', desc: '能耗上报 · 碳积分核算', icon: 'OfficeBuilding' },
  { key: 'park_admin', name: '园区管理员', desc: '入园统筹 · 园区碳数据', icon: 'Histogram' },
  { key: 'exchange', name: '碳交易所', desc: '撮合交易 · 积分流转', icon: 'TrendCharts' },
  { key: 'regulator', name: '监管核查', desc: '链上审计 · 防篡改校验', icon: 'Monitor' }
]

/**
 * 静默登录指定角色预置账号，写入 token 与 user
 * @param {string} role enterprise|park_admin|exchange|regulator
 */
export async function loginPreset(role) {
  const account = PRESET_ACCOUNTS[role]
  if (!account) throw new Error('未知角色')
  const res = await authAPI.login({ username: account.username, password: account.password })
  localStorage.setItem('token', res.data.token)
  localStorage.setItem('user', JSON.stringify(res.data.user))
  return res.data.user
}

/**
 * 当前登录用户是否为预置账号
 */
export function isPresetUser() {
  const user = JSON.parse(localStorage.getItem('user') || '{}')
  return Object.values(PRESET_ACCOUNTS).some(a => a.username === user.username)
}
