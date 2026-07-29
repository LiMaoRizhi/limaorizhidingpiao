<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div ref="containerRef" class="characters-container">
    <!-- 橙色角色 (最前, 最矮, 圆顶) -->
    <div ref="orangeRef" class="character orange-char">
      <div ref="orangeFaceRef" class="face orange-face">
        <div class="pupil" data-max-distance="5" style="width:12px;height:12px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
        <div class="pupil" data-max-distance="5" style="width:12px;height:12px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
      </div>
    </div>

    <!-- 紫色角色 (最高, 方形, 可伸长脖子) -->
    <div ref="purpleRef" class="character purple-char">
      <div ref="purpleFaceRef" class="face purple-face">
        <div class="eyeball" data-max-distance="5" style="width:18px;height:18px;border-radius:50%;background-color:white;display:flex;align-items:center;justify-content:center;overflow:hidden;will-change:height;">
          <div class="eyeball-pupil" style="width:7px;height:7px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
        </div>
        <div class="eyeball" data-max-distance="5" style="width:18px;height:18px;border-radius:50%;background-color:white;display:flex;align-items:center;justify-content:center;overflow:hidden;will-change:height;">
          <div class="eyeball-pupil" style="width:7px;height:7px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
        </div>
      </div>
    </div>

    <!-- 黑色角色 (中等高度, 方形) -->
    <div ref="blackRef" class="character black-char">
      <div ref="blackFaceRef" class="face black-face">
        <div class="eyeball" data-max-distance="4" style="width:16px;height:16px;border-radius:50%;background-color:white;display:flex;align-items:center;justify-content:center;overflow:hidden;will-change:height;">
          <div class="eyeball-pupil" style="width:6px;height:6px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
        </div>
        <div class="eyeball" data-max-distance="4" style="width:16px;height:16px;border-radius:50%;background-color:white;display:flex;align-items:center;justify-content:center;overflow:hidden;will-change:height;">
          <div class="eyeball-pupil" style="width:6px;height:6px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
        </div>
      </div>
    </div>

    <!-- 黄色角色 (圆顶, 有嘴巴) -->
    <div ref="yellowRef" class="character yellow-char">
      <div ref="yellowFaceRef" class="face yellow-face">
        <div class="pupil" data-max-distance="5" style="width:12px;height:12px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
        <div class="pupil" data-max-distance="5" style="width:12px;height:12px;border-radius:50%;background-color:#2D2D2D;will-change:transform;"></div>
      </div>
      <div ref="yellowMouthRef" class="yellow-mouth"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import gsap from 'gsap'

const props = withDefaults(defineProps<{
  isTyping?: boolean
  showPassword?: boolean
  passwordLength?: number
}>(), {
  isTyping: false,
  showPassword: false,
  passwordLength: 0,
})

// ─── 模板引用 ──────────────────────────────────────────────
const containerRef = ref<HTMLElement | null>(null)
const orangeRef = ref<HTMLElement | null>(null)
const purpleRef = ref<HTMLElement | null>(null)
const blackRef = ref<HTMLElement | null>(null)
const yellowRef = ref<HTMLElement | null>(null)
const orangeFaceRef = ref<HTMLElement | null>(null)
const purpleFaceRef = ref<HTMLElement | null>(null)
const blackFaceRef = ref<HTMLElement | null>(null)
const yellowFaceRef = ref<HTMLElement | null>(null)
const yellowMouthRef = ref<HTMLElement | null>(null)

// ─── 内部状态 ──────────────────────────────────────────────
const mouseRef = { x: 0, y: 0 }
let rafId = 0

let purpleBlinkTimer: ReturnType<typeof setTimeout> | undefined
let blackBlinkTimer: ReturnType<typeof setTimeout> | undefined
let purplePeekTimer: ReturnType<typeof setTimeout> | undefined
let lookingTimer: ReturnType<typeof setTimeout> | undefined
const isLookingRef = ref(false)

// 状态快照（避免 raf 闭包读取过期值）
const stateRef = ref({
  isTyping: false,
  isHidingPassword: false,
  isShowingPassword: false,
  isLooking: false,
})

// ─── quickTo 函数集合 ──────────────────────────────────────
interface QuickToSet {
  purpleSkew: gsap.QuickToFunc
  blackSkew: gsap.QuickToFunc
  orangeSkew: gsap.QuickToFunc
  yellowSkew: gsap.QuickToFunc
  purpleX: gsap.QuickToFunc
  blackX: gsap.QuickToFunc
  purpleHeight: gsap.QuickToFunc
  purpleFaceLeft: gsap.QuickToFunc
  purpleFaceTop: gsap.QuickToFunc
  blackFaceLeft: gsap.QuickToFunc
  blackFaceTop: gsap.QuickToFunc
  orangeFaceX: gsap.QuickToFunc
  orangeFaceY: gsap.QuickToFunc
  yellowFaceX: gsap.QuickToFunc
  yellowFaceY: gsap.QuickToFunc
  mouthX: gsap.QuickToFunc
  mouthY: gsap.QuickToFunc
}
let qt: QuickToSet | null = null

// ─── 计算函数 ──────────────────────────────────────────────
function calcPos(el: HTMLElement) {
  const rect = el.getBoundingClientRect()
  const cx = rect.left + rect.width / 2
  const cy = rect.top + rect.height / 3
  const dx = mouseRef.x - cx
  const dy = mouseRef.y - cy
  return {
    faceX: Math.max(-15, Math.min(15, dx / 20)),
    faceY: Math.max(-10, Math.min(10, dy / 30)),
    bodySkew: Math.max(-6, Math.min(6, -dx / 120)),
  }
}

function calcEyePos(el: HTMLElement, maxDist: number) {
  const r = el.getBoundingClientRect()
  const cx = r.left + r.width / 2
  const cy = r.top + r.height / 2
  const dx = mouseRef.x - cx
  const dy = mouseRef.y - cy
  const dist = Math.min(Math.sqrt(dx ** 2 + dy ** 2), maxDist)
  const angle = Math.atan2(dy, dx)
  return { x: Math.cos(angle) * dist, y: Math.sin(angle) * dist }
}

// ─── 主循环 tick ────────────────────────────────────────────
function tick() {
  const container = containerRef.value
  if (!container || !qt) {
    rafId = requestAnimationFrame(tick)
    return
  }

  const { isTyping: typing, isHidingPassword: hiding, isShowingPassword: showing, isLooking: looking } = stateRef.value

  // 紫色角色（不在显示密码状态时才更新）
  if (purpleRef.value && !showing) {
    const pp = calcPos(purpleRef.value)
    if (typing || hiding) {
      qt.purpleSkew(pp.bodySkew - 12)
      qt.purpleX(40)
      qt.purpleHeight(440)
    } else {
      qt.purpleSkew(pp.bodySkew)
      qt.purpleX(0)
      qt.purpleHeight(400)
    }
  }

  // 黑色角色
  if (blackRef.value && !showing) {
    const bp = calcPos(blackRef.value)
    if (looking) {
      qt.blackSkew(bp.bodySkew * 1.5 + 10)
      qt.blackX(20)
    } else if (typing || hiding) {
      qt.blackSkew(bp.bodySkew * 1.5)
      qt.blackX(0)
    } else {
      qt.blackSkew(bp.bodySkew)
      qt.blackX(0)
    }
  }

  // 橙色角色
  if (orangeRef.value && !showing) {
    const op = calcPos(orangeRef.value)
    qt.orangeSkew(op.bodySkew)
  }

  // 黄色角色
  if (yellowRef.value && !showing) {
    const yp = calcPos(yellowRef.value)
    qt.yellowSkew(yp.bodySkew)
  }

  // 紫色脸部位置
  if (purpleRef.value && !showing && !looking) {
    const pp = calcPos(purpleRef.value)
    const purpleFaceX = pp.faceX >= 0 ? Math.min(25, pp.faceX * 1.5) : pp.faceX
    qt.purpleFaceLeft(45 + purpleFaceX)
    qt.purpleFaceTop(40 + pp.faceY)
  }

  // 黑色脸部位置
  if (blackRef.value && !showing && !looking) {
    const bp = calcPos(blackRef.value)
    qt.blackFaceLeft(26 + bp.faceX)
    qt.blackFaceTop(32 + bp.faceY)
  }

  // 橙色脸部位置
  if (orangeRef.value && !showing) {
    const op = calcPos(orangeRef.value)
    qt.orangeFaceX(op.faceX)
    qt.orangeFaceY(op.faceY)
  }

  // 黄色脸部位置
  if (yellowRef.value && !showing) {
    const yp = calcPos(yellowRef.value)
    qt.yellowFaceX(yp.faceX)
    qt.yellowFaceY(yp.faceY)
  }

  // 黄色嘴巴位置
  if (yellowRef.value && !showing) {
    const yp = calcPos(yellowRef.value)
    qt.mouthX(yp.faceX)
    qt.mouthY(yp.faceY)
  }

  // 简单瞳孔跟随鼠标（橙/黄）
  if (!showing) {
    const allPupils = container.querySelectorAll('.pupil')
    allPupils.forEach((p) => {
      const el = p as HTMLElement
      const maxDist = Number(el.dataset.maxDistance) || 5
      const ePos = calcEyePos(el, maxDist)
      gsap.set(el, { x: ePos.x, y: ePos.y })
    })

    // 眼球瞳孔跟随鼠标（紫/黑），不在对视状态时
    if (!looking) {
      const allEyeballs = container.querySelectorAll('.eyeball')
      allEyeballs.forEach((eb) => {
        const el = eb as HTMLElement
        const maxDist = Number(el.dataset.maxDistance) || 10
        const pupil = el.querySelector('.eyeball-pupil') as HTMLElement
        if (!pupil) return
        const ePos = calcEyePos(el, maxDist)
        gsap.set(pupil, { x: ePos.x, y: ePos.y })
      })
    }
  }

  rafId = requestAnimationFrame(tick)
}

// ─── 眨眼动画 ──────────────────────────────────────────────
function scheduleBlink(
  charRef: { value: HTMLElement | null },
  eyeSize: number,
  timerKey: 'purpleBlink' | 'blackBlink'
) {
  const eyeballs = charRef.value?.querySelectorAll('.eyeball')
  if (!eyeballs?.length) return

  const timer = setTimeout(() => {
    eyeballs.forEach((el) => {
      gsap.to(el, { height: 2, duration: 0.08, ease: 'power2.in' })
    })
    const innerTimer = setTimeout(() => {
      eyeballs.forEach((el) => {
        gsap.to(el, { height: eyeSize, duration: 0.08, ease: 'power2.out' })
      })
      scheduleBlink(charRef, eyeSize, timerKey)
    }, 150)
    // 跟踪内层timer，防止组件卸载后内层回调仍触发递归
    if (timerKey === 'purpleBlink') {
      purpleBlinkTimer = innerTimer
    } else {
      blackBlinkTimer = innerTimer
    }
  }, Math.random() * 4000 + 3000)

  if (timerKey === 'purpleBlink') {
    purpleBlinkTimer = timer
  } else {
    blackBlinkTimer = timer
  }
}

// ─── 对视动画 ──────────────────────────────────────────────
function applyLookAtEachOther() {
  if (qt) {
    qt.purpleFaceLeft(55)
    qt.purpleFaceTop(65)
    qt.blackFaceLeft(32)
    qt.blackFaceTop(12)
  }
  purpleRef.value?.querySelectorAll('.eyeball-pupil').forEach((p) => {
    gsap.to(p, { x: 3, y: 4, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
  })
  blackRef.value?.querySelectorAll('.eyeball-pupil').forEach((p) => {
    gsap.to(p, { x: 0, y: -4, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
  })
}

// ─── 密码隐藏时偷看 ────────────────────────────────────────
function applyHidingPassword() {
  if (qt) {
    qt.purpleFaceLeft(55)
    qt.purpleFaceTop(65)
  }
}

// ─── 显示密码时回避 ────────────────────────────────────────
function applyShowPassword() {
  if (qt) {
    qt.purpleSkew(0)
    qt.blackSkew(0)
    qt.orangeSkew(0)
    qt.yellowSkew(0)
    qt.purpleX(0)
    qt.blackX(0)
    qt.purpleHeight(400)

    qt.purpleFaceLeft(20)
    qt.purpleFaceTop(35)
    qt.blackFaceLeft(10)
    qt.blackFaceTop(28)
    qt.orangeFaceX(50 - 82)
    qt.orangeFaceY(85 - 90)
    qt.yellowFaceX(20 - 52)
    qt.yellowFaceY(35 - 40)
    qt.mouthX(10 - 40)
    qt.mouthY(0)
  }

  purpleRef.value?.querySelectorAll('.eyeball-pupil').forEach((p) => {
    gsap.to(p, { x: -4, y: -4, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
  })
  blackRef.value?.querySelectorAll('.eyeball-pupil').forEach((p) => {
    gsap.to(p, { x: -4, y: -4, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
  })
  orangeRef.value?.querySelectorAll('.pupil').forEach((p) => {
    gsap.to(p, { x: -5, y: -4, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
  })
  yellowRef.value?.querySelectorAll('.pupil').forEach((p) => {
    gsap.to(p, { x: -5, y: -4, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
  })
}

// ─── 紫色偷瞄（显示密码时随机偷看） ────────────────────────
function schedulePeek() {
  const purpleEyePupils = purpleRef.value?.querySelectorAll('.eyeball-pupil')
  if (!purpleEyePupils?.length) return

  purplePeekTimer = setTimeout(() => {
    purpleEyePupils.forEach((p) => {
      gsap.to(p, { x: 4, y: 5, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
    })
    if (qt) {
      qt.purpleFaceLeft(20)
      qt.purpleFaceTop(35)
    }

    purplePeekTimer = setTimeout(() => {
      purpleEyePupils.forEach((p) => {
        gsap.to(p, { x: -4, y: -4, duration: 0.3, ease: 'power2.out', overwrite: 'auto' })
      })
      schedulePeek()
    }, 800)
  }, Math.random() * 3000 + 2000)
}

// ─── 监听状态变化 ──────────────────────────────────────────
watch(() => [props.isTyping, props.showPassword, props.passwordLength], () => {
  const isHidingPassword = props.passwordLength > 0 && !props.showPassword
  const isShowingPassword = props.passwordLength > 0 && props.showPassword

  stateRef.value = {
    isTyping: props.isTyping,
    isHidingPassword,
    isShowingPassword,
    isLooking: isLookingRef.value,
  }

  // 对视逻辑
  if (props.isTyping && !isShowingPassword) {
    isLookingRef.value = true
    stateRef.value.isLooking = true
    applyLookAtEachOther()

    clearTimeout(lookingTimer)
    lookingTimer = setTimeout(() => {
      isLookingRef.value = false
      stateRef.value.isLooking = false
      purpleRef.value?.querySelectorAll('.eyeball-pupil').forEach((p) => {
        gsap.killTweensOf(p)
      })
    }, 800)
  } else {
    clearTimeout(lookingTimer)
    isLookingRef.value = false
    stateRef.value.isLooking = false
  }

  // 显示/隐藏密码
  if (isShowingPassword) {
    applyShowPassword()
    // 启动紫色偷瞄
    clearTimeout(purplePeekTimer)
    schedulePeek()
  } else if (isHidingPassword) {
    applyHidingPassword()
    clearTimeout(purplePeekTimer)
  } else {
    clearTimeout(purplePeekTimer)
  }
})

// ─── 生命周期 ──────────────────────────────────────────────
onMounted(() => {
  // 初始化 quickTo 函数
  if (
    purpleRef.value && blackRef.value && orangeRef.value && yellowRef.value &&
    purpleFaceRef.value && blackFaceRef.value && orangeFaceRef.value && yellowFaceRef.value &&
    yellowMouthRef.value
  ) {
    qt = {
      purpleSkew: gsap.quickTo(purpleRef.value, 'skewX', { duration: 0.3, ease: 'power2.out' }),
      blackSkew: gsap.quickTo(blackRef.value, 'skewX', { duration: 0.3, ease: 'power2.out' }),
      orangeSkew: gsap.quickTo(orangeRef.value, 'skewX', { duration: 0.3, ease: 'power2.out' }),
      yellowSkew: gsap.quickTo(yellowRef.value, 'skewX', { duration: 0.3, ease: 'power2.out' }),
      purpleX: gsap.quickTo(purpleRef.value, 'x', { duration: 0.3, ease: 'power2.out' }),
      blackX: gsap.quickTo(blackRef.value, 'x', { duration: 0.3, ease: 'power2.out' }),
      purpleHeight: gsap.quickTo(purpleRef.value, 'height', { duration: 0.3, ease: 'power2.out' }),
      purpleFaceLeft: gsap.quickTo(purpleFaceRef.value, 'left', { duration: 0.3, ease: 'power2.out' }),
      purpleFaceTop: gsap.quickTo(purpleFaceRef.value, 'top', { duration: 0.3, ease: 'power2.out' }),
      blackFaceLeft: gsap.quickTo(blackFaceRef.value, 'left', { duration: 0.3, ease: 'power2.out' }),
      blackFaceTop: gsap.quickTo(blackFaceRef.value, 'top', { duration: 0.3, ease: 'power2.out' }),
      orangeFaceX: gsap.quickTo(orangeFaceRef.value, 'x', { duration: 0.2, ease: 'power2.out' }),
      orangeFaceY: gsap.quickTo(orangeFaceRef.value, 'y', { duration: 0.2, ease: 'power2.out' }),
      yellowFaceX: gsap.quickTo(yellowFaceRef.value, 'x', { duration: 0.2, ease: 'power2.out' }),
      yellowFaceY: gsap.quickTo(yellowFaceRef.value, 'y', { duration: 0.2, ease: 'power2.out' }),
      mouthX: gsap.quickTo(yellowMouthRef.value, 'x', { duration: 0.2, ease: 'power2.out' }),
      mouthY: gsap.quickTo(yellowMouthRef.value, 'y', { duration: 0.2, ease: 'power2.out' }),
    }
  }

  // 初始设置
  gsap.set('.pupil', { x: 0, y: 0 })
  gsap.set('.eyeball-pupil', { x: 0, y: 0 })

  // 启动眨眼
  scheduleBlink(purpleRef, 18, 'purpleBlink')
  scheduleBlink(blackRef, 16, 'blackBlink')

  // 鼠标监听
  const onMove = (e: MouseEvent) => {
    mouseRef.x = e.clientX
    mouseRef.y = e.clientY
  }
  window.addEventListener('mousemove', onMove, { passive: true })

  // 启动主循环
  rafId = requestAnimationFrame(tick)

  // 清理函数
  onUnmounted(() => {
    window.removeEventListener('mousemove', onMove)
    cancelAnimationFrame(rafId)
    clearTimeout(purpleBlinkTimer)
    clearTimeout(blackBlinkTimer)
    clearTimeout(purplePeekTimer)
    clearTimeout(lookingTimer)
  })
})
</script>

<style scoped>
.characters-container {
  position: relative;
  width: 550px;
  height: 400px;
}

.character {
  position: absolute;
  bottom: 0;
  transform-origin: bottom center;
  will-change: transform;
}

/* 橙色角色 - 最前, 最矮, 圆顶 */
.orange-char {
  left: 0;
  width: 240px;
  height: 200px;
  background-color: #FF9B6B;
  border-radius: 120px 120px 0 0;
  z-index: 3;
}

/* 紫色角色 - 最高, 方形 */
.purple-char {
  left: 70px;
  width: 180px;
  height: 400px;
  background-color: #6C3FF5;
  border-radius: 10px 10px 0 0;
  z-index: 1;
}

/* 黑色角色 - 中等高度, 方形 */
.black-char {
  left: 240px;
  width: 120px;
  height: 310px;
  background-color: #2D2D2D;
  border-radius: 8px 8px 0 0;
  z-index: 2;
}

/* 黄色角色 - 圆顶, 有嘴巴 */
.yellow-char {
  left: 310px;
  width: 140px;
  height: 230px;
  background-color: #E8D754;
  border-radius: 70px 70px 0 0;
  z-index: 4;
}

/* 脸部容器 */
.face {
  position: absolute;
  display: flex;
}

.orange-face {
  gap: 32px;
  left: 82px;
  top: 90px;
}

.purple-face {
  gap: 32px;
  left: 45px;
  top: 40px;
}

.black-face {
  gap: 24px;
  left: 26px;
  top: 32px;
}

.yellow-face {
  gap: 24px;
  left: 52px;
  top: 40px;
}

/* 黄色嘴巴 */
.yellow-mouth {
  position: absolute;
  width: 80px;
  height: 4px;
  background-color: #2D2D2D;
  border-radius: 9999px;
  left: 40px;
  top: 88px;
}
</style>
