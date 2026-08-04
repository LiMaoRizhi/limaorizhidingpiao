<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="verify-wrapper" :style="{ backgroundImage: `url(${cosmicBg})` }">
    <canvas ref="neuronCanvas" class="neuron-canvas"></canvas>
    <!-- 验证区域 - 直接在宇宙背景上 -->
    <div class="captcha-area">
        <!-- 图片区域 -->
        <div class="image-container">
          <img v-if="bgImageUrl" :src="bgImageUrl" class="bg-image" alt="验证码背景" />
          <img
            v-if="puzzleImageUrl"
            :src="puzzleImageUrl"
            class="puzzle-piece"
            :class="{ dragging: isDragging }"
            :style="{ transform: `translate(${blockX}px, ${yPosition}px)` }"
            alt="拼图块"
          />
          <!-- 结果遮罩 -->
          <transition name="verify-fade">
            <div v-if="result !== 'idle'" class="result-overlay" :class="result">
              <span class="result-icon">{{ result === 'success' ? '✓' : '✕' }}</span>
              <span class="result-text">{{ result === 'success' ? '你是碳基生命' : '你是一串数字？' }}</span>
            </div>
          </transition>
          <!-- 加载遮罩 -->
          <div v-if="loading" class="loading-overlay">
            <span class="loading-spinner"></span>
          </div>
        </div>

        <!-- 滑动条 -->
        <div class="slider-container">
          <div
            class="slider-track"
            :style="{ '--trail-width': blockX + 'px' }"
            :class="result"
          >
            <span class="slider-hint" v-if="result === 'idle' && !isDragging">向右滑动完成验证</span>
            <span class="slider-hint success" v-else-if="result === 'success'">你是碳基生命</span>
            <span class="slider-hint fail" v-else-if="result === 'fail'">你是一串数字？</span>
            <div
              class="slider-handle"
              :class="{ dragging: isDragging, success: result === 'success', fail: result === 'fail' }"
              :style="{ transform: `translateX(${blockX}px)` }"
              @mousedown.prevent="startDrag"
              @touchstart.prevent="startDrag"
            >
              <svg v-if="result === 'success'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <svg v-else-if="result === 'fail'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
              <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="5" y1="12" x2="19" y2="12" />
                <polyline points="12 5 19 12 12 19" />
              </svg>
            </div>
          </div>
        </div>

        <!-- 刷新 + 返回 -->
        <div class="action-bar">
          <span class="action-link" @click="goBack">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="19" y1="12" x2="5" y2="12" />
              <polyline points="12 19 5 12 12 5" />
            </svg>
            返回登录
          </span>
          <span class="action-link" @click="refresh">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M23 4v6h-6" />
              <path d="M1 20v-6h6" />
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
            </svg>
            刷新验证码
          </span>
        </div>
      </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { http } from '@/utils/request'
import { ElMessage } from 'element-plus'
import cosmicBg from '@/assets/cosmic-bg.jpg'

const router = useRouter()
const authStore = useAuthStore()

// 常量
const CANVAS_W = 310
const PIECE_TOTAL_W = 44
const MAX_DRAG = CANVAS_W - PIECE_TOTAL_W // 266

// 响应式状态
const loading = ref(false)
const blockX = ref(0)
const yPosition = ref(0)
const isDragging = ref(false)
const result = ref<'idle' | 'success' | 'fail'>('idle')
const bgImageUrl = ref('')
const puzzleImageUrl = ref('')

// 非响应式状态
let captchaToken = ''
let dragStartX = 0
let dragStartBlockX = 0
let timers: ReturnType<typeof setTimeout>[] = []
let pendingLogin: { username: string; password: string } | null = null

// 宇宙神经元背景动画
const neuronCanvas = ref<HTMLCanvasElement | null>(null)
let neuronAnimId = 0
let neuronCleanup: (() => void) | null = null

function initNeuronCanvas() {
  const c = neuronCanvas.value
  if (!c) return
  const ctx = c.getContext('2d')!

  let w = (c.width = window.innerWidth)
  let h = (c.height = window.innerHeight)

  const onResize = () => {
    w = c.width = window.innerWidth
    h = c.height = window.innerHeight
  }
  window.addEventListener('resize', onResize)

  // 神经元节点
  const nodes: { x: number; y: number; vx: number; vy: number; phase: number }[] = []
  for (let i = 0; i < 50; i++) {
    nodes.push({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: (Math.random() - 0.5) * 0.25,
      vy: (Math.random() - 0.5) * 0.25,
      phase: Math.random() * Math.PI * 2,
    })
  }

  const maxDist = 180
  let conns: [number, number][] = []
  function rebuildConns() {
    conns = []
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        const dx = nodes[i].x - nodes[j].x
        const dy = nodes[i].y - nodes[j].y
        if (dx * dx + dy * dy < maxDist * maxDist) conns.push([i, j])
      }
    }
  }
  rebuildConns()

  // 光脉冲（沿连线行走的光）
  const pulses: { from: number; to: number; t: number; speed: number }[] = []

  let frame = 0
  function animate() {
    ctx.clearRect(0, 0, w, h)
    frame++

    for (const n of nodes) {
      n.x += n.vx
      n.y += n.vy
      if (n.x < 0 || n.x > w) n.vx *= -1
      if (n.y < 0 || n.y > h) n.vy *= -1
    }
    if (frame % 60 === 0) rebuildConns()

    // 画连线（微弱）
    ctx.lineWidth = 0.5
    for (const [i, j] of conns) {
      const dx = nodes[i].x - nodes[j].x
      const dy = nodes[i].y - nodes[j].y
      const dist = Math.sqrt(dx * dx + dy * dy)
      const alpha = (1 - dist / maxDist) * 0.15
      ctx.strokeStyle = `rgba(255,255,255,${alpha})`
      ctx.beginPath()
      ctx.moveTo(nodes[i].x, nodes[i].y)
      ctx.lineTo(nodes[j].x, nodes[j].y)
      ctx.stroke()
    }

    // 画节点（发光星点）
    for (const n of nodes) {
      const pulse = 0.5 + 0.5 * Math.sin(frame * 0.02 + n.phase)
      const r = 1 + pulse * 1.5
      const glow = ctx.createRadialGradient(n.x, n.y, 0, n.x, n.y, r * 4)
      glow.addColorStop(0, `rgba(255,255,255,${0.6 * pulse + 0.2})`)
      glow.addColorStop(1, 'rgba(255,255,255,0)')
      ctx.fillStyle = glow
      ctx.beginPath()
      ctx.arc(n.x, n.y, r * 4, 0, Math.PI * 2)
      ctx.fill()
      ctx.fillStyle = `rgba(255,255,255,${0.7 * pulse + 0.3})`
      ctx.beginPath()
      ctx.arc(n.x, n.y, r, 0, Math.PI * 2)
      ctx.fill()
    }

    // 随机生成光脉冲
    if (Math.random() < 0.03 && conns.length > 0) {
      const [f, t] = conns[Math.floor(Math.random() * conns.length)]
      pulses.push({ from: f, to: t, t: 0, speed: 0.015 + Math.random() * 0.015 })
    }

    // 画光脉冲（沿连线行走的光点）
    for (let i = pulses.length - 1; i >= 0; i--) {
      const p = pulses[i]
      p.t += p.speed
      if (p.t >= 1) { pulses.splice(i, 1); continue }
      const fn = nodes[p.from]
      const tn = nodes[p.to]
      const x = fn.x + (tn.x - fn.x) * p.t
      const y = fn.y + (tn.y - fn.y) * p.t
      const g = ctx.createRadialGradient(x, y, 0, x, y, 10)
      g.addColorStop(0, 'rgba(255,255,255,0.9)')
      g.addColorStop(0.4, 'rgba(255,255,255,0.5)')
      g.addColorStop(1, 'rgba(255,255,255,0)')
      ctx.fillStyle = g
      ctx.beginPath()
      ctx.arc(x, y, 10, 0, Math.PI * 2)
      ctx.fill()
    }

    neuronAnimId = requestAnimationFrame(animate)
  }
  animate()

  neuronCleanup = () => {
    cancelAnimationFrame(neuronAnimId)
    window.removeEventListener('resize', onResize)
  }
}

function clearTimers() {
  timers.forEach((t) => clearTimeout(t))
  timers = []
}

function addTimer(fn: () => void, delay: number) {
  const t = setTimeout(fn, delay)
  timers.push(t)
}

// 换/刷新验证码
async function init() {
  loading.value = true
  result.value = 'idle'
  blockX.value = 0
  isDragging.value = false
  bgImageUrl.value = ''
  puzzleImageUrl.value = ''

  try {
    const res = await http.get<{
      token: string
      bgImage: string
      puzzleImage: string
      yPosition: number
    }>('/admin/captcha/get')

    captchaToken = res.data.token
    bgImageUrl.value = res.data.bgImage
    puzzleImageUrl.value = res.data.puzzleImage
    yPosition.value = res.data.yPosition
  } catch {
    addTimer(() => init(), 2000)
  } finally {
    loading.value = false
  }
}

function refresh() {
  clearTimers()
  init()
}

// 返回登录页
function goBack() {
  clearTimers()
  sessionStorage.removeItem('limao_pending_login')
  router.push('/login')
}

// 拖拽处理
function startDrag(e: MouseEvent | TouchEvent) {
  if (result.value === 'success' || loading.value) return
  e.preventDefault()

  isDragging.value = true
  result.value = 'idle'

  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX
  dragStartX = clientX
  dragStartBlockX = blockX.value

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  document.addEventListener('touchmove', onTouchMove, { passive: false })
  document.addEventListener('touchend', onTouchEnd)
}

function onMouseMove(e: MouseEvent) {
  if (!isDragging.value) return
  e.preventDefault()
  const delta = e.clientX - dragStartX
  blockX.value = Math.max(0, Math.min(MAX_DRAG, dragStartBlockX + delta))
}

function onMouseUp() {
  if (!isDragging.value) return
  isDragging.value = false
  removeEventListeners()
  checkResult()
}

function onTouchMove(e: TouchEvent) {
  if (!isDragging.value) return
  e.preventDefault()
  const delta = e.touches[0].clientX - dragStartX
  blockX.value = Math.max(0, Math.min(MAX_DRAG, dragStartBlockX + delta))
}

function onTouchEnd() {
  if (!isDragging.value) return
  isDragging.value = false
  removeEventListeners()
  checkResult()
}

function removeEventListeners() {
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  document.removeEventListener('touchmove', onTouchMove)
  document.removeEventListener('touchend', onTouchEnd)
}

// 验证 + 登录
async function checkResult() {
  if (blockX.value === 0) return

  loading.value = true
  try {
    // 1. 调后端验证滑动结果
    const checkRes = await http.post<{
      result: boolean
      verifyToken?: string
    }>('/admin/captcha/check', {
      token: captchaToken,
      moveX: Math.round(blockX.value),
    })

    if (!checkRes.data.result || !checkRes.data.verifyToken) {
      result.value = 'fail'
      addTimer(() => { blockX.value = 0 }, 400)
      addTimer(() => { refresh() }, 1000)
      return
    }

    // 2. 验证成功，执行登录
    result.value = 'success'
    if (!pendingLogin) {
      ElMessage.error('登录信息已过期，请重新登录')
      addTimer(() => goBack(), 1500)
      return
    }

    addTimer(async () => {
      try {
        await authStore.login(
          pendingLogin!.username,
          pendingLogin!.password,
          checkRes.data.verifyToken!
        )
        sessionStorage.removeItem('limao_pending_login')
        ElMessage.success('登录成功')
        router.push('/')
      } catch {
        // 登录失败（用户名密码错误等），返回登录页
        ElMessage.error('登录失败，请重新尝试')
        addTimer(() => goBack(), 1500)
      }
    }, 800)
  } catch {
    result.value = 'fail'
    addTimer(() => { blockX.value = 0 }, 400)
    addTimer(() => { refresh() }, 1000)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  initNeuronCanvas()
  // 从 sessionStorage 读取登录页传过来的账号密码
  const stored = sessionStorage.getItem('limao_pending_login')
  if (!stored) {
    router.push('/login')
    return
  }
  try {
    pendingLogin = JSON.parse(stored)
  } catch {
    router.push('/login')
    return
  }
  init()
})

onUnmounted(() => {
  clearTimers()
  removeEventListeners()
  neuronCleanup?.()
})
</script>

<style scoped>
/* 整页宇宙神经元背景 */
.verify-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #020410;
  background-size: cover;
  background-position: center bottom;
  background-repeat: no-repeat;
  position: relative;
  overflow: hidden;
}

/* Canvas 神经元动画层 */
.neuron-canvas {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
  pointer-events: none;
}

/* 验证区域 - 直接在宇宙背景上 */
.captcha-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  overflow-x: auto;
  position: relative;
  z-index: 1;
}

/* 图片容器 */
.image-container {
  position: relative;
  width: 310px;
  height: 310px;
  flex-shrink: 0; /* 防止flex布局压缩固定尺寸的验证码图片 */
  border-radius: 10px;
  overflow: hidden;
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.15);
}
.bg-image {
  display: block;
  width: 310px;
  height: 310px;
  user-select: none;
  -webkit-user-drag: none;
}
.puzzle-piece {
  position: absolute;
  top: 0;
  left: 0;
  width: 44px;
  height: 44px;
  user-select: none;
  -webkit-user-drag: none;
  transition: transform 0.35s ease-out;
}
.puzzle-piece.dragging {
  transition: none;
}

/* 结果遮罩 */
.result-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  z-index: 10;
  font-size: 16px;
  font-weight: 700;
}
.result-overlay.success {
  background: rgba(255, 255, 255, 0.55);
  color: #fff;
}
.result-overlay.fail {
  background: rgba(255, 80, 80, 0.45);
  color: #fff;
}
.result-icon {
  font-size: 36px;
  line-height: 1;
}

/* 加载遮罩 */
.loading-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  z-index: 5;
}
.loading-spinner {
  width: 24px;
  height: 24px;
  border: 3px solid rgba(255, 255, 255, 0.15);
  border-top-color: rgba(255, 255, 255, 0.8);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 滑动条 */
.slider-container {
  width: 310px;
  flex-shrink: 0;
  margin-top: 16px;
}
.slider-track {
  position: relative;
  width: 100%;
  height: 40px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  overflow: hidden;
}
.slider-track::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: var(--trail-width, 0px);
  height: 100%;
  background: rgba(255, 255, 255, 0.2);
  transition: width 0s;
}
.slider-track.success::before {
  background: rgba(255, 255, 255, 0.35);
}
.slider-track.fail::before {
  background: rgba(255, 100, 100, 0.25);
}
.slider-track:not(.success):not(.fail)::before {
  transition: width 0.35s ease-out;
}
.slider-hint {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 14px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.5);
  pointer-events: none;
  user-select: none;
  white-space: nowrap;
  z-index: 3;
}
.slider-hint.success {
  color: #fff;
  font-weight: 600;
}
.slider-hint.fail {
  color: #fff;
  font-weight: 600;
}

/* 滑块按钮 */
.slider-handle {
  position: absolute;
  top: 0;
  left: 0;
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: rgba(255, 255, 255, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: grab;
  z-index: 2;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  transition: transform 0.35s ease-out, background 0.2s, box-shadow 0.2s;
}
.slider-handle:active { cursor: grabbing; }
.slider-handle.dragging {
  transition: background 0.2s, box-shadow 0.2s;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
}
.slider-handle.success { background: rgba(255, 255, 255, 0.35); }
.slider-handle.fail { background: rgba(255, 100, 100, 0.3); }

/* 操作栏 */
.action-bar {
  width: 310px;
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  margin-top: 16px;
}
.action-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  transition: color 0.2s;
  user-select: none;
}
.action-link:hover { color: rgba(255, 255, 255, 0.9); }

/* 过渡动画 */
.verify-fade-enter-active,
.verify-fade-leave-active {
  transition: opacity 0.3s;
}
.verify-fade-enter-from,
.verify-fade-leave-to {
  opacity: 0;
}

/* 窄屏无需额外调整 */
</style>
