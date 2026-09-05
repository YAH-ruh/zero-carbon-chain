<template>
  <!-- 路由出口：直接渲染(不包 transition)，
       规避异步路由组件 + out-in 过渡在连续切换时偶发“URL 已变但视图不更新”的问题 -->
  <router-view />
</template>

<style>
/* ============================================================
   全局样式 · 克制配色 / 清晰层级 / 统一组件规范
   主题色：teal（青碧），中性色：slate 灰阶
   ============================================================ */
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body { height: 100%; }

:root {
  --primary: #0d9488;          /* 品牌主色 teal-600 */
  --primary-strong: #0f766e;   /* 深一号 teal-700 */
  --primary-bg: #f0fdfa;       /* 主色浅底 */
  --success: #059669;
  --danger: #dc2626;
  --warning: #d97706;

  --bg: #f6f7f9;               /* 页面底色 */
  --surface: #ffffff;          /* 卡片底色 */
  --line: #e5e7eb;             /* 常规描边 */
  --line-strong: #d1d5db;      /* 输入框描边 */

  --text-1: #111827;           /* 标题 */
  --text-2: #374151;           /* 正文 */
  --text-3: #6b7280;           /* 辅助说明 */
  --text-4: #9ca3af;           /* 弱化/占位 */

  --radius-sm: 6px;
  --radius: 10px;
  --radius-lg: 14px;

  --shadow-sm: 0 1px 2px rgba(17, 24, 39, 0.05);
  --shadow: 0 1px 3px rgba(17, 24, 39, 0.06), 0 1px 2px rgba(17, 24, 39, 0.04);
  --shadow-md: 0 4px 12px rgba(17, 24, 39, 0.06);

  --font: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
          'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Arial, sans-serif;
}

body {
  font-family: var(--font);
  background: var(--bg);
  color: var(--text-2);
  font-size: 14px;
  line-height: 1.65;
  -webkit-font-smoothing: antialiased;
  text-rendering: optimizeLegibility;
}

/* 页面切换过渡（轻量，仅透明度） */
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* ---------- 排版层级 ---------- */
h1, h2, h3, h4 { color: var(--text-1); line-height: 1.4; }
h2 { margin-bottom: 18px; font-size: 20px; font-weight: 600; letter-spacing: -0.01em; }
h3 { margin: 24px 0 12px; font-size: 15px; font-weight: 600; }
.page-title {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 18px;
}
.page-title h2 { margin-bottom: 0; }
.page-title .page-sub { font-size: 13px; color: var(--text-4); }

/* ---------- 页面容器 ---------- */
.page {
  max-width: 1160px;
  margin: 0 auto;
  animation: fadeUp 0.3s ease;
}
@keyframes fadeUp {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 模块说明条：浅底 + 图标，不干扰业务操作 */
.info-panel {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  background: #f0f9f8;
  border: 1px solid #e0efec;
  border-radius: var(--radius);
  margin-bottom: 22px;
  font-size: 13px;
  color: #3d5a57;
  line-height: 1.7;
}
.info-panel::before {
  content: '';
  flex: 0 0 auto;
  width: 6px;
  height: 6px;
  margin-top: 8px;
  border-radius: 50%;
  background: var(--primary);
}

/* ---------- 表单 ---------- */
.form-row {
  display: flex;
  gap: 16px;
  align-items: flex-end;
  flex-wrap: wrap;
  margin-bottom: 0;
}
.input-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 170px;
  flex: 1 1 200px;
  max-width: 320px;
}
.input-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-3);
}
input, select, textarea {
  padding: 9px 12px;
  border: 1px solid var(--line-strong);
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-family: inherit;
  color: var(--text-1);
  background: var(--surface);
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
input:focus, select:focus, textarea:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px rgba(13, 148, 136, 0.12);
}
input::placeholder { color: var(--text-4); }
input:disabled, select:disabled {
  background: #f3f4f6;
  color: var(--text-4);
}

/* 表单卡片 */
.form-card {
  padding: 24px 24px 8px;
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  margin-bottom: 22px;
}
.form-card h3 {
  margin: 0 0 18px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f3f4f6;
  font-size: 15px;
  font-weight: 600;
}

/* ---------- 按钮 ---------- */
.btn, .btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 9px 20px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--primary);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s ease, box-shadow 0.15s ease, transform 0.1s ease;
}
.btn:hover, .btn-primary:hover {
  background: var(--primary-strong);
  box-shadow: var(--shadow-md);
}
.btn:active, .btn-primary:active { transform: translateY(0); }
.btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
  box-shadow: none;
}
.btn-sm {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}
.btn-success { background: var(--success); color: #fff; }
.btn-success:hover { background: #047857; }
.btn-danger { background: var(--danger); color: #fff; }
.btn-danger:hover { background: #b91c1c; }
.btn-plain {
  background: #fff;
  color: var(--text-2);
  border: 1px solid var(--line-strong);
}
.btn-plain:hover { border-color: var(--text-3); background: #f9fafb; box-shadow: none; }
.btn-sm.btn-danger { background: #fff; border-color: #fecaca; color: var(--danger); }
.btn-sm.btn-danger:hover { background: #fef2f2; }
.btn-sm.btn-success { background: #fff; border-color: #a7f3d0; color: var(--success); }
.btn-sm.btn-success:hover { background: #ecfdf5; }

/* ---------- 表格 ---------- */
.table {
  width: 100%;
  border-collapse: collapse;
  background: var(--surface);
  font-size: 13px;
  border-radius: var(--radius);
  overflow: hidden;
  border: 1px solid var(--line);
  box-shadow: var(--shadow-sm);
}
.table th {
  padding: 11px 14px;
  text-align: left;
  font-weight: 600;
  font-size: 12px;
  color: var(--text-3);
  background: #f9fafb;
  border-bottom: 1px solid var(--line);
  white-space: nowrap;
}
.table td {
  padding: 11px 14px;
  border-bottom: 1px solid #f3f4f6;
  color: var(--text-2);
  vertical-align: middle;
}
.table tbody tr { transition: background 0.12s ease; }
.table tbody tr:hover { background: #fafcfc; }
.table tbody tr:last-child td { border-bottom: none; }
.table td code {
  font-size: 12px;
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 4px;
  color: #4b5563;
  word-break: break-all;
}
.empty { text-align: center; color: var(--text-4); padding: 36px !important; }
.tbl-scroll { overflow-x: auto; border-radius: var(--radius); box-shadow: var(--shadow-sm); }
.tbl-scroll .table { box-shadow: none; border: none; border-radius: 0; }

/* ---------- 标签 ---------- */
.tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.6;
}
.tag-available { background: #ecfdf5; color: #047857; }
.tag-locked { background: #fef2f2; color: #b91c1c; }
.tag-sold { background: #eff6ff; color: #1d4ed8; }
.tag-pending { background: #fffbeb; color: #b45309; }
.tag-matched { background: #f5f3ff; color: #6d28d9; }

/* ---------- 统计卡片 ---------- */
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 14px;
  margin-bottom: 24px;
}
.stat-card {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  padding: 18px 16px;
  text-align: center;
  box-shadow: var(--shadow-sm);
}
.stat-card .num {
  display: block;
  font-size: 26px;
  font-weight: 650;
  letter-spacing: -0.01em;
  color: var(--text-1);
  line-height: 1.2;
}
.stat-card span:last-child {
  display: block;
  font-size: 12px;
  color: var(--text-3);
  margin-top: 4px;
}

/* ---------- 子功能页工具栏(仅页面内按钮导航，顶部无导航/切换) ---------- */
.feature-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  background: #fff;
  border: 1px solid var(--line);
  border-radius: var(--radius-lg);
  padding: 9px 12px;
  margin-bottom: 18px;
  box-shadow: var(--shadow-sm);
}
.ft-back {
  display: inline-flex; align-items: center; gap: 5px;
  border: none; background: #0f766e; color: #fff;
  padding: 7px 14px; border-radius: 8px;
  cursor: pointer; font-size: 13px; white-space: nowrap;
  transition: background 0.15s ease;
}
.ft-back svg { width: 13px; height: 13px; }
.ft-back:hover { background: #115e59; }
.ft-crumb { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--text-3); white-space: nowrap; }
.ft-crumb span { color: var(--text-2); }
.ft-crumb svg { width: 12px; height: 12px; color: #9ca3af; }
.ft-crumb b { color: var(--text-1); font-weight: 600; }

/* ---------- 可视化图表区(各角色页通用，消除空白) ---------- */
.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 16px;
  margin: 2px 0 26px;
}
.charts-grid > .panel-card { min-width: 0; }
@media (max-width: 768px) {
  .charts-grid { grid-template-columns: 1fr; }
}

/* ---------- 结果 / 报告 ---------- */
.result-box, .advice-box, .chain-info {
  margin-top: 14px;
  padding: 18px 20px;
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  font-size: 13px;
  line-height: 1.8;
  box-shadow: var(--shadow-sm);
}
.result-box p, .chain-info p { margin: 5px 0; }
.result-box h3 { margin: 0 0 10px; }
.advice-box pre {
  white-space: pre-wrap;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.8;
  color: var(--text-2);
}

/* ---------- 消息状态 ---------- */
.error { color: var(--danger); font-size: 13px; }
.success { color: var(--success); font-size: 13px; }

/* ---------- 告警卡片 ---------- */
.alert-card {
  padding: 14px 16px;
  margin-bottom: 12px;
  border-radius: var(--radius);
  border: 1px solid;
  border-left-width: 3px;
  font-size: 13px;
}
.alert-info { background: #f0f6ff; border-color: #bfdbfe; color: #1e40af; }
.alert-warning { background: #fffbf0; border-color: #fde68a; color: #92400e; }
.alert-success { background: #f0fdf4; border-color: #bbf7d0; color: #065f46; }
.alert-header { display: flex; gap: 8px; margin-bottom: 2px; align-items: center; }
.alert-level { font-weight: 600; }

/* ---------- 流程步骤条（线性、克制） ---------- */
.process-flow {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 6px 0;
  margin: 4px 0 22px;
  counter-reset: pstep;
}
.process-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  counter-increment: pstep;
}
.process-step .step-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: #fff;
  border: 1px solid #c7d7d4;
  color: var(--primary-strong);
  font-weight: 600;
  font-size: 13px;
}
.process-step .step-icon::after {
  content: counter(pstep, decimal-leading-zero);
  font-size: 12px;
  font-weight: 600;
}
.process-step .step-label {
  font-size: 12px;
  color: var(--text-3);
  text-align: center;
  white-space: nowrap;
}
.process-arrow {
  color: #cbd5d1;
  margin: -14px 8px 0;
  font-size: 14px;
  flex-shrink: 0;
}
/* 圆点里的 emoji 文字隐藏，仅展示序号，保证图标风格统一 */
.process-step .step-icon { font-size: 0; }

/* ---------- 状态切换提示 ---------- */
.text-muted { color: var(--text-4); font-size: 12px; }

/* ---------- Loading ---------- */
.loading {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.35);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
/* 浅色 / 描边按钮上的加载圈配色 */
.btn-sm .spinner, .btn-plain .spinner {
  border-color: #d1d5db;
  border-top-color: #374151;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ---------- 响应式 ---------- */
@media (max-width: 768px) {
  body { font-size: 13px; }
  .form-row { gap: 12px; }
  .form-row > * { flex: 1 1 100%; }
  .form-card { padding: 20px 16px 6px; }
  .btn { width: 100%; }
  .btn-sm { width: auto; }
  .process-flow { justify-content: flex-start; overflow-x: auto; flex-wrap: nowrap; }
  .process-step { min-width: 64px; }
  .process-arrow { margin: -14px 2px 0; }
  .info-panel { flex-direction: row; }
}
</style>
