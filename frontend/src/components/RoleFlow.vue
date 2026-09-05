<template>
  <!-- RoleFlow 四大角色业务闭环导览(系统公共组件，展示于各业务页顶部)
       目的：让使用者一屏看懂「小微企业-园区管理员-碳交易所-监管核查」四身份协作闭环，
       强调能耗上报→碳积分→撮合交易→链上存证→监管校验的业务完整性。 -->
  <section class="role-flow">
    <div class="rf-title">
      <span class="rf-title-icon"><Connection /></span>
      <span class="rf-title-text">四大角色业务闭环</span>
      <span class="rf-tag">碳资产管理全链路</span>
    </div>

    <div class="rf-flow">
      <div v-for="(r, i) in roles" :key="r.name" class="rf-node" :title="r.desc">
        <div class="rf-node-chip">
          <span class="rf-node-icon" :style="{ background: r.tint }">
            <component :is="r.icon" />
          </span>
          <span class="rf-node-name">{{ r.name }}</span>
        </div>
        <span class="rf-node-desc">{{ r.desc }}</span>
      </div>

      <!-- 闭环箭头 -->
      <div class="rf-arrows" aria-hidden="true">
        <span class="rf-arrow" v-for="i in 3" :key="i"><component :is="i === 3 ? 'Switch' : 'ArrowRight'" /></span>
      </div>

      <!-- 闭环回路说明(视觉上箭头折返，体现"环") -->
      <div class="rf-loop">
        <span class="rf-loop-arrow"><component :is="'RefreshLeft'" /></span>
        <span class="rf-loop-text">审核监督贯穿全链路 · 关键数据全程 SHA-256 上链存证，可溯源可校验</span>
      </div>
    </div>
  </section>
</template>

<script setup>
// 角色节点定义：业务责任一句话讲清闭环分工
const roles = [
  { name: '小微企业', icon: 'OfficeBuilding', tint: '#e6f7f5', color: '#0d9488', desc: '手动上报能耗 → 自动核算生成碳积分' },
  { name: '园区管理员', icon: 'Histogram', tint: '#e8f2fe', color: '#2563eb', desc: '入园审核 · 园区碳排放统筹总览' },
  { name: '碳交易所', icon: 'TrendCharts', tint: '#fdf0e4', color: '#d97706', desc: '撮合交易 · 碳积分权属可信流转' },
  { name: '监管核查', icon: 'Monitor', tint: '#f0eefe', color: '#7c3aed', desc: '链上存证审计 · 全链路防篡改校验' }
]
</script>

<style scoped>
.role-flow {
  border: 1px solid #dbe9e7;
  background: linear-gradient(135deg, #f6fcfb 0%, #ffffff 55%, #f3f9fb 100%);
  border-radius: var(--radius);
  padding: 12px 16px 14px;
  margin-bottom: 20px;
  box-shadow: var(--shadow-sm);
}
.rf-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.rf-title-icon { color: var(--primary); display: inline-flex; }
.rf-title-icon svg { width: 16px; height: 16px; }
.rf-title-text {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-1);
}
.rf-tag {
  font-size: 11px;
  color: var(--primary-strong);
  background: #ecfdf5;
  border: 1px solid #d1fae5;
  border-radius: 999px;
  padding: 1px 8px;
}
.rf-flow { position: relative; display: flex; align-items: center; }
.rf-node { flex: 1; min-width: 0; }
.rf-node-chip {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 3px;
}
.rf-node-icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.rf-node-icon svg { width: 16px; height: 16px; color: inherit; }
.rf-node-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-1);
  white-space: nowrap;
}
.rf-node-desc {
  display: block;
  font-size: 11px;
  color: var(--text-4);
  line-height: 1.6;
  margin-left: 35px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rf-arrows {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex-shrink: 0;
  width: 26px;
  align-items: center;
  justify-content: center;
  color: #94c6bf;
}
.rf-arrow svg { width: 14px; height: 14px; }
.rf-arrow:first-child { transform: translateY(6px); }
.rf-loop {
  position: absolute;
  right: 0;
  bottom: -14px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-4);
}
.rf-loop-arrow { color: #94a3b8; display: inline-flex; transform: rotate(-90deg); }
.rf-loop-arrow svg { width: 12px; height: 12px; }

@media (max-width: 900px) {
  .rf-node-desc { display: none; }
  .rf-node-chip { flex-direction: column; gap: 3px; text-align: center; }
  .rf-arrows { display: none; }
  .rf-loop { display: none; }
}
</style>
