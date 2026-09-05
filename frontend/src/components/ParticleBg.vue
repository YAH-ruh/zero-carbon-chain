<template>
  <canvas ref="canvas" class="particle-bg"></canvas>
</template>

<script setup>
// ParticleBg 低碳粒子光影背景
// 轻量 Canvas 粒子系统：仅作氛围背景，pointer-events:none 不影响任何操作；
// 颜色沿用青-碧主题(#0d9488/#22d3ee)，低透明度保证正文可读性。
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'

const props = defineProps({
  density: { type: Number, default: 42 },  // 粒子数量
  linkDist: { type: Number, default: 120 }, // 连线判定距离(px)
  opacity: { type: Number, default: 0.18 } // 全局透明度
})

const canvas = ref(null)
let ctx = null
let raf = 0
let particles = []
let w = 0
let h = 0

function resize() {
  const el = canvas.value
  if (!el) return
  w = el.clientWidth
  h = el.clientHeight
  // 适配高分屏
  const dpr = Math.min(window.devicePixelRatio || 1, 2)
  el.width = w * dpr
  el.height = h * dpr
  ctx = el.getContext('2d')
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
}

function initParticles() {
  const count = Math.max(12, props.density)
  particles = Array.from({ length: count }, () => ({
    x: Math.random() * w,
    y: Math.random() * h,
    vx: (Math.random() - 0.5) * 0.35,
    vy: (Math.random() - 0.5) * 0.35,
    r: Math.random() * 1.8 + 0.6,
    phase: Math.random() * Math.PI * 2
  }))
}

function step() {
  if (!ctx) return
  ctx.clearRect(0, 0, w, h)
  const t = Date.now() / 1200
  for (const p of particles) {
    // 缓慢漂浮 + 呼吸
    p.x += p.vx
    p.y += p.vy
    const px = Math.sin(t + p.phase) * 0.25
    p.x += px
    if (p.x < -10) p.x = w + 10
    if (p.x > w + 10) p.x = -10
    if (p.y < -10) p.y = h + 10
    if (p.y > h + 10) p.y = -10
    ctx.beginPath()
    ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2)
    ctx.fillStyle = `rgba(13,148,136,${props.opacity + 0.1})`
    ctx.fill()
  }
  // 邻近连线
  for (let i = 0; i < particles.length; i++) {
    for (let j = i + 1; j < particles.length; j++) {
      const a = particles[i]
      const b = particles[j]
      const dx = a.x - b.x
      const dy = a.y - b.y
      const dist = Math.hypot(dx, dy)
      if (dist < props.linkDist) {
        const alpha = (1 - dist / props.linkDist) * props.opacity
        ctx.beginPath()
        ctx.moveTo(a.x, a.y)
        ctx.lineTo(b.x, b.y)
        ctx.strokeStyle = `rgba(34,211,238,${alpha})`
        ctx.lineWidth = 0.6
        ctx.stroke()
      }
    }
  }
  raf = requestAnimationFrame(step)
}

function start() {
  resize()
  initParticles()
  cancelAnimationFrame(raf)
  raf = requestAnimationFrame(step)
}

onMounted(() => {
  start()
  window.addEventListener('resize', start)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', start)
  cancelAnimationFrame(raf)
})
watch(() => props.density, () => start())
</script>

<style scoped>
.particle-bg {
  position: fixed;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}
</style>
