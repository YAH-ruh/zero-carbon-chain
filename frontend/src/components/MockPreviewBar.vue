<template>
  <!-- 图表模拟预览开关：仅控制首页"图表 series"展示，顶部统计卡片数字不受影响 -->
  <div class="mp-bar">
    <label class="mp-switch">
      <input
        type="checkbox"
        :checked="modelValue"
        @change="emit('update:modelValue', $event.target.checked)"
      />
      <span class="mp-slider"></span>
      <span class="mp-text">图表模拟预览(内置演示数据)</span>
    </label>
    <span class="mp-note">预览仅填充图表 series，指标卡数字仍取自真实接口</span>
  </div>
</template>

<script setup>
defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue'])
</script>

<style scoped>
.mp-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin: 0 0 14px;
  padding: 8px 14px;
  background: #fff;
  border: 1px dashed #99f6e4;
  border-radius: 999px;
  width: fit-content;
}
.mp-switch { display: inline-flex; align-items: center; gap: 8px; cursor: pointer; }
.mp-switch input { position: absolute; opacity: 0; width: 0; height: 0; }
.mp-slider {
  width: 32px; height: 18px; border-radius: 999px;
  background: #d1d5db; position: relative; transition: background 0.18s ease; flex-shrink: 0;
}
.mp-slider::after {
  content: ''; position: absolute; top: 2px; left: 2px;
  width: 14px; height: 14px; border-radius: 50%; background: #fff;
  transition: transform 0.18s ease;
}
.mp-switch input:checked + .mp-slider { background: #0d9488; }
.mp-switch input:checked + .mp-slider::after { transform: translateX(14px); }
.mp-text { font-size: 12px; font-weight: 500; color: #0f766e; white-space: nowrap; }
.mp-note { font-size: 11px; color: var(--text-4, #9ca3af); }
@media (max-width: 640px) {
  .mp-bar { width: 100%; border-radius: var(--radius-sm, 8px); }
}
</style>
