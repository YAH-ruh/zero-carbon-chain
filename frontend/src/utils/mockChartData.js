/**
 * utils/mockChartData.js
 * 仪表盘图表用的 Mock 数据（后续替换为真实 API）
 */

/* 近 12 个月标签 */
const MONTHS = (() => {
  const arr = []
  const now = new Date()
  for (let i = 11; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    arr.push(`${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}`)
  }
  return arr
})()

/* 近 30 天标签 */
const DAYS = (() => {
  const arr = []
  const now = new Date()
  for (let i = 29; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i)
    arr.push(`${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`)
  }
  return arr
})()

const rand = (min, max) => Math.round(Math.random() * (max - min) + min)

/* 生成近 12 个月能耗/积分趋势 */
export function genMonthlyTrend(baseEmission = 800, volatility = 0.3) {
  const labels = MONTHS
  const emission = labels.map(() => Math.round(baseEmission * (1 + (Math.random() - 0.5) * volatility)))
  const credits = emission.map(v => Math.round(v * 0.25 * (0.8 + Math.random() * 0.4)))
  return { labels, emission, credits }
}

/* 生成近 30 天日趋势 */
export function genDailyTrend(base = 40, volatility = 0.4) {
  const labels = DAYS
  const values = labels.map(() => Math.round(base * (1 + (Math.random() - 0.5) * volatility)))
  return { labels, values }
}

/* 碳积分分布（环形图数据） */
export function genCreditDistribution() {
  return [
    { name: '已挂单',   value: rand(200, 500) },
    { name: '已成交',   value: rand(300, 800) },
    { name: '可用余额', value: rand(100, 400) },
    { name: '质押中',   value: rand(50, 200) },
    { name: '已过期',   value: rand(10, 50) },
  ]
}

/* 交易类型构成 */
export function genTradeComposition() {
  return [
    { name: '现货交易',   value: rand(400, 900) },
    { name: '租赁交易',   value: rand(100, 300) },
    { name: '远期交易',   value: rand(80, 200) },
    { name: '质押融资',   value: rand(50, 150) },
  ]
}

/* 企业能耗排行柱状图 */
export function genEnterpriseRanking(count = 8) {
  // 使用平台真实注册的 5 家园区入驻企业（与数据库企业名录一致）
  const names = [
    '绿恒节能科技有限公司',
    '晨光烘焙食品有限公司',
    '恒达针织纺织品有限公司',
    '精工不锈钢制品有限公司',
    '蓝天包装制品有限公司',
  ]
  const labels = names.slice(0, count)
  const values = labels.map(() => rand(500, 1500))
  return { labels, values }
}

/* 园区分布数据（大屏用） */
export function genParkDistribution() {
  return [
    { name: '绿色科技示范园',   emission: rand(2000, 5000), credits: rand(500, 1200), enterprises: rand(8, 20) },
    { name: '低碳智造产业园',   emission: rand(1500, 4000), credits: rand(400, 1000), enterprises: rand(6, 15) },
    { name: '新能源工业园区',   emission: rand(1000, 3000), credits: rand(300, 800),  enterprises: rand(5, 12) },
    { name: '循环经济产业园',   emission: rand(800, 2500),  credits: rand(200, 600),  enterprises: rand(4, 10) },
  ]
}

/* 雷达图：企业多维评估 */
export function genEnterpriseRadar() {
  return {
    indicator: [
      { name: '能耗合规',   max: 100 },
      { name: '碳积分健康', max: 100 },
      { name: '链上存证',   max: 100 },
      { name: 'AI减排指数', max: 100 },
      { name: 'ZKP隐私性',  max: 100 },
      { name: '交易活跃度', max: 100 },
    ],
    series: [
      { name: '本企业', value: [rand(70,95), rand(60,90), rand(80,100), rand(50,85), rand(75,95), rand(40,80)] },
      { name: '园区均值', value: [rand(55,75), rand(50,70), rand(60,80), rand(40,65), rand(60,80), rand(35,65)] },
    ]
  }
}
