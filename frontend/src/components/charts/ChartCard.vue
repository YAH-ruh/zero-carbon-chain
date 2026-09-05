<template>
  <section class="panel-card">
    <header class="panel-head">
      <div class="panel-titles">
        <h4 class="panel-title">
          <slot name="icon" />
          {{ title }}
        </h4>
        <p v-if="subtitle" class="panel-sub">{{ subtitle }}</p>
      </div>
      <div class="panel-extra">
        <slot name="extra" />
      </div>
    </header>
    <div class="panel-body">
      <slot />
    </div>
  </section>
</template>

<script setup>
// ChartCard 统一图表/内容卡片容器
// 复用全局主题色与圆角，保证各角色页可视化区块视觉一致、消除页面空白感。
defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' }
})
</script>

<style scoped>
.panel-card {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  padding: 18px 20px;
  min-width: 0;
}
.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}
.panel-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-1);
}
.panel-title :deep(svg) {
  color: var(--primary);
}
.panel-sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-4);
  line-height: 1.6;
}
.panel-extra {
  flex-shrink: 0;
}
.panel-body {
  min-height: 0;
}
</style>
