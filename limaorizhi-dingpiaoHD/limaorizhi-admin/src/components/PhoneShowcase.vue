<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<!-- 手机界面演示：手机桌面 → 微信 → 小程序装修页面 完整流程 -->
<template>
  <div class="phone-showcase">
    <!-- 3D翻转舞台 -->
    <div class="phone-3d-stage">
      <div class="phone-3d-entrance" :style="entranceStyle">
        <!-- 品牌背面（旋转时交替可见） -->
        <div class="phone-back-face">
          <img :src="appleBackImg" class="back-face-img" />
        </div>
        <!-- 正面：手机模拟器 -->
        <div class="phone-front-face">
    <div class="phone-preview">
      <div class="phone-wrapper">
        <div class="phone-side-btn side-btn-action"></div>
        <div class="phone-side-btn side-btn-vol-up"></div>
        <div class="phone-side-btn side-btn-vol-down"></div>
        <div class="phone-frame apple" :class="screen">
          <div class="dynamic-island"><div class="dynamic-island-camera"></div></div>

          <!-- 背景图 + 液态毛玻璃（覆盖整个屏幕含状态栏，效果一致） -->
          <div v-if="screen === 'home'" class="phone-bg" :style="{ backgroundImage: `url(${phoneBg})` }"></div>
          <div v-if="screen === 'home'" class="phone-liquid-glass"></div>

          <!-- 状态栏 -->
          <div class="phone-statusbar" :class="{ 'status-on-dark': screen === 'home' }">
            <span class="status-time">9:41</span>
            <div class="status-right">
              <svg width="17" height="11" viewBox="0 0 17 11" fill="currentColor"><rect x="0" y="7" width="3" height="4" rx="0.5"/><rect x="4.5" y="5" width="3" height="6" rx="0.5"/><rect x="9" y="3" width="3" height="8" rx="0.5"/><rect x="13.5" y="1" width="3" height="10" rx="0.5"/></svg>
              <svg width="15" height="11" viewBox="0 0 15 11" fill="currentColor"><path d="M7.5 1.5C4.6 1.5 1.9 2.6 0 4.5l1.4 1.4C3 4.3 5.2 3.5 7.5 3.5s4.5 0.8 6.1 2.4L15 4.5C13.1 2.6 10.4 1.5 7.5 1.5z"/><path d="M7.5 5C5.8 5 4.2 5.7 3 6.8l1.4 1.4c0.9-0.8 2-1.2 3.1-1.2s2.2 0.4 3.1 1.2L12 6.8C10.8 5.7 9.2 5 7.5 5z"/><circle cx="7.5" cy="9" r="1.5"/></svg>
              <svg width="27" height="12" viewBox="0 0 27 12"><rect x="1" y="1" width="22" height="10" rx="3" fill="none" stroke="currentColor" stroke-width="1" opacity="0.35"/><rect x="2.5" y="2.5" width="19" height="7" rx="1.5" fill="currentColor"/><rect x="24" y="4" width="2" height="4" rx="0.5" fill="currentColor" opacity="0.4"/></svg>
            </div>
          </div>

          <div class="screen-container">
            <!-- 1. 手机桌面 -->
            <div v-show="screen === 'home'" class="screen-home">
              <div class="desktop-content">
                <!-- 桌面主功能图标（单页，可长按拖拽重排） -->
                <div class="desktop-top" :class="{ 'dragging-active': dragIndex >= 0 }">
                  <div
                    v-for="(app, i) in desktopApps"
                    :key="app.id"
                    class="top-app"
                    :class="{ 'dragging': dragIndex === i, 'drag-over': dragOverIndex === i && dragIndex >= 0 && dragIndex !== i }"
                    @mousedown="onDragStart($event, i)"
                    @touchstart="onDragStart($event, i)"
                    @click="onAppClick(app)"
                  >
                    <div class="top-app-icon" :class="{ 'shop-entry-icon': app.svg }" :style="{ background: app.bg }">
                      <img v-if="app.img" :src="app.img" class="app-img" />
                      <div v-else class="app-svg" v-html="app.svg"></div>
                    </div>
                    <div class="app-name">{{ app.name }}</div>
                  </div>
                </div>
                <!-- 底部 dock 栏：电话 / 通讯录 / 消息 / 设置 -->
                <div class="dock-bar">
                  <div class="dock-item" v-for="(app, i) in dockApps" :key="'dock'+i" @click="app.action ? app.action() : null">
                    <div class="dock-icon" :class="{ 'app-wechat': app.isWechat }" :style="{ background: app.color }">
                      <img v-if="app.img" :src="app.img" class="app-img" />
                      <span v-else class="app-glyph">{{ app.text }}</span>
                    </div>
                    <div class="dock-name">{{ app.name }}</div>
                  </div>
                </div>
              </div>
              <div class="home-indicator" @click="goScreen('wechat')"></div>
              <!-- 应用提示 toast -->
              <transition name="wx-tip-fade">
                <div v-if="appTipText" class="home-tip-toast">{{ appTipText }}</div>
              </transition>

            </div>

            <!-- 2. 微信首页 -->
            <div v-show="screen === 'wechat'" class="screen-wechat">
              <div class="wx-scroll" ref="wechatScrollRef">
                <div class="wx-header">
                  <span class="wx-header-title">微信</span>
                  <div class="wx-header-actions">
                    <span class="wx-header-icon" @click="showWxTip">
                      <svg width="22" height="22" viewBox="0 0 24 24" fill="none"><circle cx="11" cy="11" r="7" stroke="#1a1a1a" stroke-width="1.6"/><line x1="16.5" y1="16.5" x2="21" y2="21" stroke="#1a1a1a" stroke-width="1.6" stroke-linecap="round"/></svg>
                    </span>
                    <span class="wx-header-icon" @click="showWxTip">
                      <svg width="22" height="22" viewBox="0 0 24 24" fill="none"><circle cx="12" cy="12" r="10" stroke="#1a1a1a" stroke-width="1.6"/><line x1="12" y1="8" x2="12" y2="16" stroke="#1a1a1a" stroke-width="1.6" stroke-linecap="round"/><line x1="8" y1="12" x2="16" y2="12" stroke="#1a1a1a" stroke-width="1.6" stroke-linecap="round"/></svg>
                    </span>
                  </div>
                </div>
                <!-- 公众号虚拟内容 -->
                <div class="wx-section">
                  <div class="wx-section-title">公众号</div>
                  <div class="wx-content-item" @click="showWxTip">
                    <img :src="wxGzhIcon" class="wx-item-img" />
                    <div class="wx-item-main">
                      <div class="wx-item-name">热心的司机行驶在路上</div>
                      <div class="wx-item-desc">热心的司机行驶在路上</div>
                    </div>
                    <span class="wx-item-arrow">›</span>
                  </div>
                </div>
                <!-- 群组虚拟内容 -->
                <div class="wx-section">
                  <div class="wx-section-title">群组</div>
                  <div class="wx-content-item" @click="showWxTip">
                    <img :src="wxQunzuIcon" class="wx-item-img" />
                    <div class="wx-item-main">
                      <div class="wx-item-name">老师傅司机交接班群</div>
                      <div class="wx-item-desc">司机交接班交流群</div>
                    </div>
                    <span class="wx-item-arrow">›</span>
                  </div>
                </div>
                <!-- 用户分享的小程序链接 -->
                <div class="wx-section">
                  <div class="wx-section-title">用户</div>
                  <div class="wx-content-item" @click="goMiniappDetail">
                    <img :src="limaoTicketIcon" class="wx-item-img" />
                    <div class="wx-item-main">
                      <div class="wx-item-name">狸猫日志</div>
                      <div class="wx-item-desc">分享了一个小程序</div>
                    </div>
                    <span class="wx-item-arrow">›</span>
                  </div>
                </div>
                <div style="height: 60px"></div>
              </div>
              <div class="wx-tabbar">
                <div class="wx-tab-item" :class="{ active: wechatTab === 'chat' }" @click="wechatTab = 'chat'">
                  <img :src="wechatChat" class="wx-tab-icon" />
                  <span class="wx-tab-text">微信</span>
                </div>
                <div class="wx-tab-item" @click="showWxTip">
                  <img :src="wechatContacts" class="wx-tab-icon" />
                  <span class="wx-tab-text">通讯录</span>
                </div>
                <div class="wx-tab-item" @click="showWxTip">
                  <img :src="wechatDiscover" class="wx-tab-icon" />
                  <span class="wx-tab-text">发现</span>
                </div>
                <div class="wx-tab-item" @click="showWxTip">
                  <img :src="wechatMe" class="wx-tab-icon" />
                  <span class="wx-tab-text">我</span>
                </div>
              </div>
              <!-- 模拟测试提示 toast -->
              <transition name="wx-tip-fade">
                <div v-if="wxTipVisible" class="wx-tip-toast">仅作为模拟测试，无法获取更多功能</div>
              </transition>
              <div class="home-indicator home-indicator-light" @click="goScreen('home')"></div>
            </div>

            <!-- 3. 小程序分享信息详情页 -->
            <div v-show="screen === 'miniappDetail'" class="screen-miniapp-detail">
              <!-- 顶部导航栏 -->
              <div class="detail-header">
                <span class="detail-back" @click="goScreen('wechat')">
                  <svg width="12" height="20" viewBox="0 0 12 20" fill="none"><path d="M10 2L2 10L10 18" stroke="#1a1a1a" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                </span>
                <span class="detail-title">聊天信息</span>
              </div>
              <!-- 内容区 -->
              <div class="detail-body" ref="miniappScrollRef">
                <div class="detail-chat-info">
                  <div class="detail-chat-avatar">
                    <img :src="limaoTicketIcon" class="detail-chat-avatar-img" />
                  </div>
                  <div class="detail-chat-meta">
                    <div class="detail-chat-name">狸猫日志</div>
                    <div class="detail-chat-time">刚刚</div>
                  </div>
                </div>
                <div class="detail-chat-bubble">
                  <div class="detail-bubble-text">分享了一个小程序</div>
                </div>
                <!-- 分享的小程序卡片：点击跳转装修页 -->
                <div class="detail-miniapp-card" @click="goDecorate">
                  <div class="detail-card-cover">
                    <img :src="limaoTicketIcon" class="detail-cover-img" />
                  </div>
                  <div class="detail-card-info">
                    <div class="detail-card-name">狸猫日志</div>
                    <div class="detail-card-desc">点击进入小程序装修页面</div>
                  </div>
                  <span class="detail-card-arrow">›</span>
                </div>
              </div>
              <div class="home-indicator home-indicator-light" @click="goScreen('home')"></div>
            </div>

            <!-- Deleted: miniapp screen block (screen-miniapp) - moved to /design/decorate route -->

          </div>
          <!-- 关机黑屏覆盖层 -->
          <div v-if="isPoweredOff" class="power-off-screen">
            <div class="power-off-icon" v-html="powerOffIcon"></div>
          </div>
        </div>
      </div>
    </div>
        </div><!-- phone-front-face -->
      </div><!-- phone-3d-entrance -->
    </div><!-- phone-3d-stage -->

    <!-- 模拟器方向键（纵向单列：上/下/左/右 + 中间回顶），悬浮手机右上方 -->
    <div class="simulator-dpad">
      <div class="dpad-btn" @click="scrollPhone('up')" title="向上滚动"><span class="dpad-icon" v-html="navUpIcon"></span></div>
      <div class="dpad-btn" @click="scrollPhone('down')" title="向下滚动"><span class="dpad-icon" v-html="navDownIcon"></span></div>
      <div class="dpad-btn dpad-center" @click="exitToHome" title="退出回首页"><span class="dpad-icon" v-html="navCenterIcon"></span></div>
      <div class="dpad-btn" @click="goBack" title="返回上一级"><span class="dpad-icon" v-html="navLeftIcon"></span></div>
      <div class="dpad-btn" @click="goForward" title="继续前往下一级"><span class="dpad-icon" v-html="navRightIcon"></span></div>
      <div class="dpad-btn dpad-power" :class="{ 'dpad-power-off': isPoweredOff }" @click="togglePower" :title="isPoweredOff ? '开机' : '关机'"><span class="dpad-icon" v-html="powerOffIcon"></span></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import wechatAppIcon from '@/assets/icons/wechat-app.svg'
import wechatChatIcon from '@/assets/icons/wechat-tab-chat.svg'
import wechatContactsIcon from '@/assets/icons/wechat-tab-contacts.svg'
import wechatDiscoverIcon from '@/assets/icons/wechat-tab-discover.svg'
import wechatMeIcon from '@/assets/icons/wechat-tab-me.svg'
import dockSettingsIcon from '@/assets/icons/dock-settings.svg'
import dockPhoneIcon from '@/assets/icons/dock-phone.svg'
import dockContactsIcon from '@/assets/icons/dock-contacts.svg'
import dockMessageIcon from '@/assets/icons/dock-message.svg'
import phoneBg from '@/assets/phone-bg.png'
import appStoreIcon from '@/assets/icons/app-store.jpg'
import limaoTicketIcon from '@/assets/icons/limao-ticket.png'
import wxGzhIcon from '@/assets/icons/wx-gongzhonghao.jpg'
import wxQunzuIcon from '@/assets/icons/wx-qunzu.jpg'
import appleBackImg from '@/assets/icons/apple.png'
import { PHONE_LONG_PRESS_MS } from '@/utils/constants'
// 模拟器方向键图标（与装修页一致）
import navUpIcon from '@/assets/icons/nav-up.svg?raw'
import navDownIcon from '@/assets/icons/nav-down.svg?raw'
import navLeftIcon from '@/assets/icons/nav-left.svg?raw'
import navRightIcon from '@/assets/icons/nav-right.svg?raw'
import navCenterIcon from '@/assets/icons/nav-center.svg?raw'

// 入场旋转动画：比装修页切换机型力度更大、左右摆动后停留正面
const props = defineProps<{ showEntranceSpin?: boolean }>()
// 用 transition 驱动（与装修页一致），避免 @keyframes 下 backface-visibility 在插值帧失效导致后盖丢失
const spinAngle = ref(0)
const spinVisible = ref(false)
let spinTimers: ReturnType<typeof setTimeout>[] = []
const entranceStyle = computed(() => ({
  transform: `rotateY(${spinAngle.value}deg)`,
  // 不旋转时直接可见(opacity:1)；旋转时由 spinVisible 控制淡入
  opacity: props.showEntranceSpin ? (spinVisible.value ? 1 : 0) : 1,
}))
const startEntranceSpin = () => {
  // 重复触发入场动画前先清理上一轮定时器，防止堆积导致动画错乱
  spinTimers.forEach(clearTimeout)
  spinTimers = []
  spinVisible.value = true
  // 时间线复刻原 keyframes（总6秒）：淡入→右转180停留→回正→左转-180停留→回正
  const plan = [
    { t: 360,  angle: 180 },
    { t: 2160, angle: 180 },
    { t: 2760, angle: 0 },
    { t: 3360, angle: -180 },
    { t: 4560, angle: -180 },
    { t: 5160, angle: 0 },
  ]
  plan.forEach(p => {
    spinTimers.push(setTimeout(() => { spinAngle.value = p.angle }, p.t))
  })
}
// 关机/开机功能
const isPoweredOff = ref(false)
const powerOffIcon = `<svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path d="M653.329602 261.679792c-10.95023-5.935639-24.561263-1.944433-30.59924 8.903458s-1.944433 24.561263 8.903458 30.59924C726.194683 352.965821 784.937038 452.131921 784.937038 559.792125c0 162.513692-132.221467 294.735159-294.735159 294.735159S195.46672 722.305817 195.46672 559.792125c0-110.935039 61.30082-211.329202 159.955227-262.191485 11.052568-5.730961 15.35079-19.239656 9.722167-30.292225-5.730961-11.052568-19.239656-15.35079-30.292225-9.722167-54.648811 28.143114-100.701179 70.613632-133.244853 122.908655-33.464721 53.727763-51.169298 115.744953-51.169298 179.297222 0 45.847691 9.005797 90.364981 26.710374 132.221467 17.090546 40.423746 41.651809 76.753948 72.762742 107.96722 31.213272 31.213272 67.543474 55.672197 107.96722 72.762742C399.836898 890.55047 444.354187 899.556266 490.201879 899.556266c45.847691 0 90.364981-9.005797 132.221467-26.710374 40.423746-17.090546 76.753948-41.651809 107.96722-72.762742 31.213272-31.213272 55.672197-67.543474 72.762742-107.96722C820.960224 650.157106 829.96602 605.639816 829.96602 559.792125 829.96602 435.655407 762.320208 321.445533 653.329602 261.679792z" fill="#2c2c2c"/><path d="M490.201879 616.078353c12.485309 0 22.514491-10.029182 22.514491-22.514491L512.71637 160.671597c0-12.485309-10.029182-22.514491-22.514491-22.514491s-22.514491 10.029182-22.514491 22.514491l0 432.892265C467.687388 606.04917 477.71657 616.078353 490.201879 616.078353z" fill="#2c2c2c"/></svg>`
const togglePower = () => {
  isPoweredOff.value = !isPoweredOff.value
  if (isPoweredOff.value) screen.value = 'home'
}

// 屏幕导航：左键返回上一级 / 右键继续前往下一级 / 中间键退出回首页
const goBack = () => {
  if (isPoweredOff.value) return
  if (screen.value === 'miniappDetail') screen.value = 'wechat'
  else if (screen.value === 'wechat') screen.value = 'home'
}
const goForward = () => {
  if (isPoweredOff.value) return
  if (screen.value === 'home') screen.value = 'wechat'
  else if (screen.value === 'wechat') screen.value = 'miniappDetail'
}
const exitToHome = () => {
  if (isPoweredOff.value) return
  screen.value = 'home'
}

// 模拟器方向键滚动控制（上/下滚动可滚动屏；左/右/中已改导航）
const miniappScrollRef = ref<HTMLElement | null>(null)
const scrollPhone = (dir: 'up' | 'down' | 'left' | 'right' | 'center') => {
  if (isPoweredOff.value) return
  let container: HTMLElement | null = null
  if (screen.value === 'wechat') container = wechatScrollRef.value
  else if (screen.value === 'miniappDetail') container = miniappScrollRef.value
  if (!container) return
  const amount = 200
  if (dir === 'up') container.scrollBy({ top: -amount, behavior: 'smooth' })
  else if (dir === 'down') container.scrollBy({ top: amount, behavior: 'smooth' })
  else if (dir === 'center') container.scrollTo({ top: 0, behavior: 'smooth' })
  else if (dir === 'left' || dir === 'right') {
    const delta = dir === 'left' ? -amount : amount
    const allEls = container.querySelectorAll('*')
    for (const el of allEls) {
      const hEl = el as HTMLElement
      const cs = window.getComputedStyle(hEl)
      if ((cs.overflowX === 'auto' || cs.overflowX === 'scroll') && hEl.scrollWidth > hEl.clientWidth + 1) {
        hEl.scrollBy({ left: delta, behavior: 'smooth' })
        break
      }
    }
  }
}

onMounted(() => {
  if (props.showEntranceSpin) startEntranceSpin()
  // 注册全局拖拽事件（鼠标 + 触摸）
  document.addEventListener('mousemove', onDragMove)
  document.addEventListener('mouseup', onDragEnd)
  document.addEventListener('touchmove', onDragMove, { passive: false })
  document.addEventListener('touchend', onDragEnd)
})

const wechatChat = wechatChatIcon
const wechatContacts = wechatContactsIcon
const wechatDiscover = wechatDiscoverIcon
const wechatMe = wechatMeIcon
const router = useRouter()

type Screen = 'home' | 'wechat' | 'miniappDetail'
const screen = ref<Screen>('home')
const wechatTab = ref<'chat' | 'contacts' | 'discover' | 'me'>('chat')
const wechatScrollRef = ref<HTMLElement | null>(null)
let scrollTimer: ReturnType<typeof setTimeout> | null = null
const wxTipVisible = ref(false)
let wxTipTimer: ReturnType<typeof setTimeout> | null = null
// 点击通讯录/发现/我：仅模拟测试提示，不跳转
const showWxTip = () => {
  wxTipVisible.value = true
  if (wxTipTimer) clearTimeout(wxTipTimer)
  wxTipTimer = setTimeout(() => { wxTipVisible.value = false }, 2000)
}

// 应用提示（动态文字）
const appTipText = ref('')
let appTipTimer: ReturnType<typeof setTimeout> | null = null
const showAppTip = (msg: string) => {
  appTipText.value = msg
  if (appTipTimer) clearTimeout(appTipTimer)
  appTipTimer = setTimeout(() => { appTipText.value = '' }, 2000)
}

// 小店功能入口图标（白色 SVG，搭配彩色背景，仿 APP 图标风格）
const shopDecorateSvg = `<svg viewBox="0 0 1024 1024" width="34" height="34" xmlns="http://www.w3.org/2000/svg"><path d="M887.637333 170.5984H870.4a17.237333 17.237333 0 0 1-17.237333-17.237333v-68.096c0-37.546667-30.72-68.266667-68.266667-68.266667h-648.533333c-37.546667 0-68.266667 30.72-68.266667 68.266667v238.933333c0 37.546667 30.72 68.266667 68.266667 68.266667h648.533333c37.546667 0 68.266667-30.72 68.266667-68.266667v-68.096a17.237333 17.237333 0 1 1 34.474666 0v187.392a17.237333 17.237333 0 0 1-17.237333 17.237333H492.970667a68.266667 68.266667 0 0 0-68.266667 68.266667v65.467733c-36.7104 0.9728-66.491733 31.146667-66.491733 68.078934v276.1728c0 37.546667 30.72 68.266667 68.266666 68.266666h68.266667c37.546667 0 68.266667-30.72 68.266667-68.266666V662.562133c0-37.546667-30.72-68.266667-68.266667-68.266666h-1.774933v-65.297067H887.637333a68.266667 68.266667 0 0 0 68.266667-68.266667v-221.866666a68.266667 68.266667 0 0 0-68.266667-68.266667zM153.2416 119.1424c0-18.773333 15.36-34.133333 34.133333-34.133333h546.133334c18.773333 0 34.133333 15.36 34.133333 34.133333s-15.36 34.133333-34.133333 34.133333h-546.133334c-18.773333 0-34.133333-15.36-34.133333-34.133333z" fill="#fff"/></svg>`
const ticketManageSvg = `<svg viewBox="0 0 1024 1024" width="34" height="34" xmlns="http://www.w3.org/2000/svg"><path d="M106.666667 738.133333c0 42.666667 21.333333 85.333333 51.2 110.933334V938.666667c0 29.866667 21.333333 51.2 51.2 51.2h51.2c29.866667 0 51.2-21.333333 51.2-51.2v-51.2h405.333333V938.666667c0 29.866667 21.333333 51.2 51.2 51.2h51.2c29.866667 0 51.2-21.333333 51.2-51.2v-85.333334c29.866667-29.866667 51.2-68.266667 51.2-110.933333v-512c0-179.2-183.466667-204.8-405.333333-204.8s-409.6 25.6-409.6 204.8v507.733333z m179.2 51.2c-42.666667 0-76.8-34.133333-76.8-76.8S243.2 640 285.866667 640s76.8 34.133333 76.8 76.8-34.133333 72.533333-76.8 72.533333z m456.533333 0c-42.666667 0-76.8-34.133333-76.8-76.8s34.133333-76.8 76.8-76.8 76.8 34.133333 76.8 76.8-34.133333 76.8-76.8 76.8z m76.8-302.933333H209.066667v-256h610.133333v256z" fill="#fff"/></svg>`
const orderManageSvg = `<svg viewBox="0 0 1024 1024" width="34" height="34" xmlns="http://www.w3.org/2000/svg"><path d="M961.28 414.464v508.928c0 55.104-42.752 99.52-95.872 99.52H158.016c-52.928 0-95.872-44.48-95.872-99.392V100.224c0-54.784 43.072-99.392 95.552-99.392h406.016v348.544c0 35.904 28.16 65.088 62.528 65.088h335.04z m-17.472-64.96L626.176 19.008v330.368c0-0.064 0.128 0.128 0.064 0.128h317.568z m-723.52 64.96h208.128c17.28 0 31.232-14.592 31.232-32.512s-13.952-32.448-31.232-32.448H220.288c-17.216 0-31.168 14.528-31.168 32.448s13.952 32.512 31.168 32.512z m0 216.512h582.784c17.216 0 31.232-14.528 31.232-32.448 0-17.984-14.016-32.512-31.232-32.512H220.288c-17.216 0-31.168 14.528-31.168 32.512 0 17.92 13.952 32.448 31.168 32.448z" fill="#fff"/></svg>`
// 小店功能入口跳转：装修→小程序装修页 / 车票→班次管理（增加车票设定价格）/ 订单→订单列表
const goTicketManage = () => router.push('/ticket/trips')
const goOrderList = () => router.push('/order/list')

// ====== 桌面 APP 数据驱动 + 长按拖拽重排 ======
type DesktopApp = {
  id: string
  name: string
  bg: string
  svg?: string
  img?: any
  actionKey: 'wechat' | 'appstore' | 'decorate' | 'ticket' | 'order'
}
const defaultDesktopApps: DesktopApp[] = [
  { id: 'wechat',   name: '微信',     bg: 'transparent', img: wechatAppIcon, actionKey: 'wechat' },
  { id: 'appstore', name: '应用市场', bg: 'transparent', img: appStoreIcon, actionKey: 'appstore' },
  { id: 'decorate', name: '装修',     bg: '#EAB308',     svg: shopDecorateSvg,  actionKey: 'decorate' },
  { id: 'ticket',   name: '车票',     bg: '#22C55E',     svg: ticketManageSvg,  actionKey: 'ticket' },
  { id: 'order',    name: '订单',     bg: '#06B6D4',     svg: orderManageSvg,   actionKey: 'order' },
]
const STORAGE_KEY_DESKTOP = 'limao_desktop_order'
const loadDesktopApps = (): DesktopApp[] => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY_DESKTOP)
    if (saved) {
      const ids: string[] = JSON.parse(saved)
      const ordered: DesktopApp[] = []
      ids.forEach(id => {
        const found = defaultDesktopApps.find(a => a.id === id)
        if (found) ordered.push(found)
      })
      // 补充默认列表里新增的 APP
      defaultDesktopApps.forEach(a => {
        if (!ordered.find(o => o.id === a.id)) ordered.push(a)
      })
      return ordered
    }
  } catch (e) { /* localStorage 不可用时用默认顺序 */ }
  return [...defaultDesktopApps]
}
const desktopApps = ref<DesktopApp[]>(loadDesktopApps())

// 点击 APP（刚拖拽完不触发点击）
let dragJustEnded = false
const onAppClick = (app: DesktopApp) => {
  if (dragJustEnded) { dragJustEnded = false; return }
  switch (app.actionKey) {
    case 'wechat':   goScreen('wechat'); break
    case 'appstore': showAppTip('狸猫系统暂无应用'); break
    case 'decorate': goDecorate(); break
    case 'ticket':   goTicketManage(); break
    case 'order':    goOrderList(); break
  }
}

// ====== 长按拖拽重排逻辑 ======
const dragIndex = ref(-1)        // 正在拖拽的图标索引，-1 表示未拖拽
const dragOverIndex = ref(-1)   // 鼠标悬停的目标索引（让位提示）
let longPressTimer: ReturnType<typeof setTimeout> | null = null
let dragGhost: HTMLElement | null = null  // 跟随鼠标的克隆元素
let dragStartX = 0, dragStartY = 0, dragMoved = false
const LONG_PRESS_MS = PHONE_LONG_PRESS_MS
// 长按阈值：超过 PHONE_LONG_PRESS_MS 进入拖拽模式

const onDragStart = (e: MouseEvent | TouchEvent, i: number) => {
  const point = 'touches' in e ? e.touches[0] : (e as MouseEvent)
  // 同步保存事件目标引用（currentTarget 在事件结束后会被浏览器置空）
  const target = e.currentTarget as HTMLElement
  dragStartX = point.clientX
  dragStartY = point.clientY
  dragMoved = false
  longPressTimer = setTimeout(() => {
    // 达到长按阈值，进入拖拽模式
    dragIndex.value = i
    // 创建跟随鼠标的幽灵元素（原图标位置变透明占位）
    const rect = target.getBoundingClientRect()
    const ghost = target.cloneNode(true) as HTMLElement
    ghost.classList.add('drag-ghost')
    ghost.style.position = 'fixed'
    ghost.style.width = rect.width + 'px'
    ghost.style.height = rect.height + 'px'
    ghost.style.left = point.clientX - rect.width / 2 + 'px'
    ghost.style.top = point.clientY - rect.height / 2 + 'px'
    ghost.style.pointerEvents = 'none'
    ghost.style.zIndex = '9999'
    ghost.style.transform = 'scale(1.15)'
    ghost.style.opacity = '0.92'
    document.body.appendChild(ghost)
    dragGhost = ghost
  }, LONG_PRESS_MS)
}

const onDragMove = (e: MouseEvent | TouchEvent) => {
  // 未进入拖拽模式时，检测移动距离取消长按计时器
  if (dragIndex.value === -1) {
    if (longPressTimer) {
      const p = 'touches' in e ? e.touches[0] : (e as MouseEvent)
      if (Math.abs(p.clientX - dragStartX) > 6 || Math.abs(p.clientY - dragStartY) > 6) {
        clearTimeout(longPressTimer); longPressTimer = null
      }
    }
    return
  }
  const point = 'touches' in e ? e.touches[0] : (e as MouseEvent)
  if ('touches' in e) e.preventDefault()  // 阻止触摸滚动
  dragMoved = true
  // ghost 位置限制在手机屏幕范围内（避免拖出手机）
  if (dragGhost) {
    const phoneEl = document.querySelector('.phone-frame')
    const phoneRect = phoneEl ? phoneEl.getBoundingClientRect() : null
    const ghostW = parseFloat(dragGhost.style.width)
    const ghostH = parseFloat(dragGhost.style.height)
    let gx = point.clientX - ghostW / 2
    let gy = point.clientY - ghostH / 2
    if (phoneRect) {
      gx = Math.max(phoneRect.left + 4, Math.min(phoneRect.right - ghostW - 4, gx))
      gy = Math.max(phoneRect.top + 4, Math.min(phoneRect.bottom - ghostH - 4, gy))
    }
    dragGhost.style.left = gx + 'px'
    dragGhost.style.top  = gy + 'px'
  }
  // 用“最近图标”算法检测目标位置（解决鼠标在 gap 间隙时 dragOverIndex=-1 的问题）
  const allApps = Array.from(document.querySelectorAll('.desktop-top .top-app')) as HTMLElement[]
  let nearestIdx = -1
  let nearestDist = Infinity
  allApps.forEach((el, idx) => {
    if (idx === dragIndex.value) return  // 排除正在拖拽的自己
    const r = el.getBoundingClientRect()
    const cx = r.left + r.width / 2
    const cy = r.top + r.height / 2
    const dist = Math.hypot(point.clientX - cx, point.clientY - cy)
    if (dist < nearestDist) {
      nearestDist = dist
      nearestIdx = idx
    }
  })
  dragOverIndex.value = nearestIdx
}

const onDragEnd = () => {
  if (longPressTimer) { clearTimeout(longPressTimer); longPressTimer = null }
  if (dragGhost) { document.body.removeChild(dragGhost); dragGhost = null }
  // 交换两个图标的位置（符合“切换位置”直觉，避免 splice 索引偏移问题）
  if (dragIndex.value !== -1 && dragOverIndex.value !== -1 && dragMoved) {
    const apps = [...desktopApps.value]
    const tmp = apps[dragIndex.value]
    apps[dragIndex.value] = apps[dragOverIndex.value]
    apps[dragOverIndex.value] = tmp
    desktopApps.value = apps
    try { localStorage.setItem(STORAGE_KEY_DESKTOP, JSON.stringify(apps.map(a => a.id))) } catch (e) {}
  }
  if (dragMoved) dragJustEnded = true  // 防止 mouseup 后立即触发 click
  dragIndex.value = -1
  dragOverIndex.value = -1
}

const dockApps = ref<any[]>([
  { name: '电话', color: '#fff', img: dockPhoneIcon, action: () => showAppTip('未识别到SIM卡') },
  { name: '通讯录', color: '#fff', img: dockContactsIcon, action: () => showAppTip('未识别到SIM卡') },
  { name: '消息', color: '#fff', img: dockMessageIcon, action: () => showAppTip('未识别到SIM卡') },
  { name: '设置', color: '#fff', img: dockSettingsIcon },
])

// 微信公众号/群组虚拟内容（点击仅提示）

// 屏幕跳转
const goScreen = (s: Screen) => {
  if (s === 'wechat') {
    screen.value = 'wechat'
    wechatTab.value = 'chat'
    // 进入微信后，自动慢慢下滑，露出小程序入口
    nextTick(() => {
      if (scrollTimer) clearTimeout(scrollTimer)
      scrollTimer = setTimeout(() => {
        const el = wechatScrollRef.value
        if (el) {
          el.scrollTo({ top: 360, behavior: 'smooth' })
        }
      }, 600)
    })
    return
  }
  screen.value = s
}

// 微信小程序入口：跳转到真实小程序装修页面
const goDecorate = () => router.push('/design/decorate')

// 进入小程序分享信息详情页
const goMiniappDetail = () => { screen.value = 'miniappDetail' }

onUnmounted(() => {
  if (scrollTimer) clearTimeout(scrollTimer)
  // 补充清理遗漏的定时器，防止内存泄漏
  if (wxTipTimer) clearTimeout(wxTipTimer)
  if (appTipTimer) clearTimeout(appTipTimer)
  spinTimers.forEach(clearTimeout)
  if (longPressTimer) clearTimeout(longPressTimer)
  if (dragGhost) { try { document.body.removeChild(dragGhost) } catch (e) {} dragGhost = null }
  document.removeEventListener('mousemove', onDragMove)
  document.removeEventListener('mouseup', onDragEnd)
  document.removeEventListener('touchmove', onDragMove)
  document.removeEventListener('touchend', onDragEnd)
})
</script>

<style scoped>
/* ===== 组件容器 ===== */
.phone-showcase {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 8px;
}

/* ===== 模拟器方向键（纵向单列悬浮手机右上方，不挤压手机） ===== */
.simulator-dpad {
  position: absolute;
  top: 14px;
  right: -15px;
  z-index: 50;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 6px 4px;
  border: 1px solid #d0d0d0;
  border-radius: 16px;
  background: rgba(248,248,248,0.85);
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}
.dpad-btn {
  width: 24px; height: 24px; border-radius: 50%;
  background: rgba(255,255,255,0.92); border: 1px solid #dcdfe6;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; transition: all 0.15s; user-select: none;
  box-shadow: 0 2px 6px rgba(0,0,0,0.15);
}
.dpad-btn:hover { background: #f0f0f0; border-color: #c0c4cc; }
.dpad-btn:active { transform: scale(0.9); background: #e8e8e8; }
.dpad-btn.dpad-center { background: #1a1a1a; border-color: #1a1a1a; }
.dpad-btn.dpad-center:hover { background: #333; border-color: #333; }
.dpad-icon { display: inline-flex; align-items: center; justify-content: center; width: 14px; height: 14px; pointer-events: none; }
.dpad-icon :deep(svg) { width: 12px; height: 12px; }
.dpad-btn.dpad-center .dpad-icon :deep(svg) path { fill: #fff !important; }
/* 关机按钮 */
.dpad-btn.dpad-power { background: #fff; border-color: #dcdfe6; }
.dpad-btn.dpad-power:hover { background: #f0f0f0; border-color: #c0c4cc; }
.dpad-btn.dpad-power-off { background: #2c2c2c; border-color: #2c2c2c; }
.dpad-btn.dpad-power-off .dpad-icon :deep(svg) path { fill: #fff !important; }

/* 关机黑屏 */
.power-off-screen {
  position: absolute;
  inset: 0;
  background: #000;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 46px;
}
.power-off-icon { display: inline-flex; }
.power-off-icon :deep(svg) { width: 44px; height: 44px; }
.power-off-icon :deep(svg) path { fill: #fff !important; opacity: 0.6; }

/* ===== 3D翻转舞台 ===== */
.phone-3d-stage {
  perspective: 1400px;
}
.phone-3d-entrance {
  position: relative;
  transform-style: preserve-3d;
  transform: rotateY(0deg);
  /* 用 transition 驱动旋转：backface-visibility 在 transition 渲染路径下稳定生效，后盖全程可见 */
  transition: transform 0.6s cubic-bezier(0.22, 0.61, 0.36, 1), opacity 0.36s ease;
  will-change: transform;
}
/* 正反两面 */
.phone-back-face,
.phone-front-face {
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
  will-change: transform;
}
.phone-front-face {
  position: relative;
  z-index: 2;
}
.phone-back-face {
  position: absolute;
  top: 0; left: 0;
  width: 326px;
  height: min(640px, calc(100vh - 100px));
  transform: rotateY(180deg);
  border-radius: 48px;
  border: 3px solid #c4703c;
  box-shadow: 0 0 0 2px #d48048, 0 0 0 3px #a46028, 0 0 0 4px #945018, 0 12px 48px rgba(0, 0, 0, 0.25);
  background: #c4703c;
  overflow: hidden;
}
.back-face-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 45px;
  display: block;
}
.back-face-content {
  position: absolute; inset: 0;
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  color: #fff;
  border-radius: 45px;
}
.back-logo {
  font-size: 30px; font-weight: 700; letter-spacing: 4px;
  text-shadow: 0 2px 8px rgba(0,0,0,0.3);
}
.back-sub {
  font-size: 12px; letter-spacing: 6px; opacity: 0.7;
  margin-top: 6px; text-transform: uppercase;
}
.back-dot {
  width: 10px; height: 10px; border-radius: 50%;
  background: rgba(255,255,255,0.5); margin-top: 28px;
}

/* ===== 手机预览 ===== */
.phone-preview { flex-shrink: 0; display: flex; flex-direction: column; align-items: center; }
.phone-wrapper { position: relative; display: inline-block; }
.phone-side-btn {
  position: absolute; border-radius: 1px; z-index: 1;
  background: linear-gradient(to bottom, #6a6a6e 0%, #3a3a3c 50%, #6a6a6e 100%);
}
.side-btn-action { left: -4px; top: 105px; width: 4px; height: 22px; }
.side-btn-vol-up { left: -4px; top: 145px; width: 4px; height: 45px; }
.side-btn-vol-down { left: -4px; top: 200px; width: 4px; height: 45px; }

.phone-frame {
  width: 320px;
  height: min(640px, calc(100vh - 100px));
  background: #f5f6f8;
  border-radius: 48px;
  border: 3px solid #c4703c;
  box-shadow: 0 0 0 2px #d48048, 0 0 0 3px #a46028, 0 0 0 4px #945018, 0 12px 48px rgba(0, 0, 0, 0.25);
  display: flex; flex-direction: column;
  overflow: hidden;
  position: relative;
}
.dynamic-island {
  position: absolute; top: 10px; left: 50%; transform: translateX(-50%);
  width: 95px; height: 26px;
  background: #000; border-radius: 13px; z-index: 20;
}
.dynamic-island-camera {
  position: absolute; right: 10px; top: 50%; transform: translateY(-50%);
  width: 6px; height: 6px; background: #1a1a2e; border-radius: 50%;
  box-shadow: inset 0 0 2px rgba(100, 100, 255, 0.3);
}
.phone-statusbar {
  height: 44px; display: flex; align-items: center; justify-content: space-between;
  padding: 0 24px; font-size: 14px; font-weight: 600; color: #1a1a1a;
  flex-shrink: 0; position: relative; z-index: 5;
}
/* home 屏幕：状态栏透明，液态玻璃由全屏层统一提供 */
.phone-statusbar.status-on-dark {
  color: #fff;
  background: transparent;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.35);
}
.status-time { letter-spacing: 0.5px; }
.status-right { display: flex; align-items: center; gap: 6px; }

.screen-container { flex: 1; position: relative; overflow: hidden; z-index: 2; }
.home-indicator {
  position: absolute; bottom: 6px; left: 50%; transform: translateX(-50%);
  width: 120px; height: 5px; background: #1a1a1a; border-radius: 3px;
  z-index: 30; cursor: pointer; opacity: 0.3;
  transition: opacity 0.2s, transform 0.2s;
}
.home-indicator:hover { opacity: 0.7; transform: translateX(-50%) scaleY(1.4); }
.home-indicator-light { background: #1a1a1a; opacity: 0.5; }

/* ===== 手机桌面 ===== */
.screen-home {
  position: absolute; inset: 0; display: flex; flex-direction: column;
}
/* 背景图：覆盖整个手机屏幕含状态栏 */
.phone-bg {
  position: absolute; inset: 0; z-index: 0;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}
/* 透明液态毛玻璃：极淡，背景清晰透出，仅保留薄薄一层玻璃质感 */
.phone-liquid-glass {
  position: absolute; inset: 0; z-index: 1;
  background: linear-gradient(135deg, rgba(255,255,255,0.02) 0%, rgba(255,255,255,0.008) 50%, rgba(255,255,255,0.015) 100%);
  backdrop-filter: blur(2px) saturate(108%);
  -webkit-backdrop-filter: blur(2px) saturate(108%);
  box-shadow: inset 0 1px 0.5px rgba(255,255,255,0.08);
}
.desktop-content {
  position: relative; z-index: 1; flex: 1;
  display: flex; flex-direction: column; justify-content: flex-start;
  /* 上边距给状态栏下空间，左右避免贴边 */
  padding: 34px 18px 0; gap: 0; overflow: hidden;
  box-sizing: border-box;
}
/* 主功能图标区：5 个图标居中，间距适中不换行 */
.desktop-top {
  display: flex; justify-content: center; flex-wrap: nowrap;
  gap: 14px 10px; padding: 0 2px;
}
.top-app { display: flex; flex-direction: column; align-items: center; gap: 4px; cursor: pointer; }
.top-app-icon {
  width: 46px; height: 46px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.22); overflow: hidden;
  transition: transform 0.15s;
}
.top-app:active .top-app-icon { transform: scale(0.88); }
.top-app-icon .app-img { width: 100%; height: 100%; object-fit: contain; padding: 0; }
.app-img { width: 100%; height: 100%; object-fit: cover; }
.app-glyph { color: #fff; font-size: 22px; font-weight: 300; }
.app-name { color: #fff; font-size: 10px; text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5); }
/* SVG 图标容器（装修/车票/订单） */
.app-svg { display: flex; align-items: center; justify-content: center; width: 100%; height: 100%; }
.app-svg :deep(svg) { width: 30px; height: 30px; }

/* ====== 拖拽重排样式 ====== */
/* 拖拽进行中：禁止文本选择，所有图标轻微抖动 */
.desktop-top.dragging-active { user-select: none; -webkit-user-select: none; }
.desktop-top.dragging-active .top-app { animation: app-wiggle 0.3s ease-in-out infinite; }
@keyframes app-wiggle {
  0%, 100% { transform: rotate(-2deg); }
  50%      { transform: rotate(2deg); }
}
/* 正在拖拽的原图标：半透明占位 */
.top-app.dragging { opacity: 0.25; animation: none; }
/* 鼠标悬停的目标位置：轻微放大 + 高亮阴影提示让位 */
.top-app.drag-over { animation: none; }
.top-app.drag-over .top-app-icon {
  transform: scale(1.08);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.3), 0 0 0 2px rgba(255, 255, 255, 0.6);
}
/* 跟随鼠标的幽灵元素 */
.drag-ghost {
  transition: none !important;
  filter: drop-shadow(0 8px 16px rgba(0, 0, 0, 0.35));
}
.dock-bar {
  margin: auto 0 10px; padding: 9px 8px; display: flex; justify-content: space-around; gap: 6px;
  background: rgba(255, 255, 255, 0.22);
  backdrop-filter: blur(20px); -webkit-backdrop-filter: blur(20px);
  border-radius: 26px;
}
.dock-item { cursor: pointer; }
.dock-icon {
  width: 44px; height: 44px; border-radius: 11px;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.22); overflow: hidden;
  transition: transform 0.15s;
}
.dock-icon .app-img { width: 72%; height: 72%; object-fit: contain; }
.dock-item:active .dock-icon { transform: scale(0.88); }
.dock-name { color: #fff; font-size: 10px; margin-top: 4px; text-align: center; text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5); }

/* ===== 微信首页 ===== */
.screen-wechat { position: absolute; inset: 0; display: flex; flex-direction: column; background: #fff; }
.wx-scroll { flex: 1; overflow-y: auto; overflow-x: hidden; }
.wx-scroll::-webkit-scrollbar { display: none; }
/* 顶部 header：微信居中，搜索/加号在右 */
.wx-header {
  position: relative; display: flex; align-items: center; justify-content: center;
  padding: 10px 16px; background: #ededed;
}
.wx-header-title { font-size: 18px; font-weight: 600; color: #1a1a1a; }
.wx-header-actions {
  position: absolute; right: 16px; top: 50%; transform: translateY(-50%);
  display: flex; gap: 18px; align-items: center;
}
.wx-header-icon { display: inline-flex; cursor: pointer; }
/* 公众号/群组虚拟内容区域 */
.wx-section { background: #fff; margin-top: 8px; }
.wx-section-title { padding: 8px 16px 4px; font-size: 12px; color: #888; }
.wx-content-item {
  display: flex; align-items: center; gap: 12px;
  padding: 12px 16px; border-bottom: 0.5px solid #eee; cursor: pointer;
}
.wx-content-item:active { background: #f5f5f5; }
.wx-item-img {
  width: 42px; height: 42px; border-radius: 6px; object-fit: cover; flex-shrink: 0;
}
.wx-item-main { flex: 1; min-width: 0; }
.wx-item-name { font-size: 15px; color: #1a1a1a; font-weight: 500; }
.wx-item-desc {
  font-size: 12px; color: #999; margin-top: 2px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.wx-item-arrow { font-size: 18px; color: #ccc; flex-shrink: 0; }
/* 小程序分享信息详情页 */
.screen-miniapp-detail {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  background: #ededed; z-index: 2;
}
.detail-header {
  position: relative; display: flex; align-items: center; justify-content: center;
  padding: 10px 16px; background: #ededed; flex-shrink: 0;
}
.detail-back { position: absolute; left: 12px; top: 50%; transform: translateY(-50%); cursor: pointer; display: inline-flex; }
.detail-title { font-size: 18px; font-weight: 600; color: #1a1a1a; }
.detail-body { flex: 1; overflow-y: auto; padding: 12px 16px; }
.detail-body::-webkit-scrollbar { display: none; }
.detail-chat-info { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.detail-chat-avatar { width: 38px; height: 38px; border-radius: 4px; overflow: hidden; flex-shrink: 0; background: #fff; }
.detail-chat-avatar-img { width: 100%; height: 100%; object-fit: cover; }
.detail-chat-meta { flex: 1; min-width: 0; }
.detail-chat-name { font-size: 14px; color: #576b95; font-weight: 500; }
.detail-chat-time { font-size: 11px; color: #999; margin-top: 1px; }
.detail-chat-bubble {
  background: #fff; border-radius: 4px; padding: 10px 12px;
  font-size: 14px; color: #1a1a1a; display: inline-block; margin-bottom: 10px;
}
.detail-miniapp-card {
  display: flex; align-items: center; gap: 10px; width: 80%;
  padding: 10px 12px; background: #fff; border-radius: 8px; cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
}
.detail-miniapp-card:active { transform: scale(0.98); }
.detail-card-cover { width: 48px; height: 48px; border-radius: 6px; overflow: hidden; flex-shrink: 0; background: #f5f5f5; }
.detail-cover-img { width: 100%; height: 100%; object-fit: cover; }
.detail-card-info { flex: 1; min-width: 0; }
.detail-card-name { font-size: 14px; font-weight: 500; color: #1a1a1a; }
.detail-card-desc { font-size: 11px; color: #999; margin-top: 3px; }
.detail-card-arrow { font-size: 18px; color: #ccc; flex-shrink: 0; }
.wx-tabbar {
  height: 52px; display: flex; flex-shrink: 0;
  background: #f7f7f7; border-top: 0.5px solid #ddd;
  padding-bottom: 6px;
}
.wx-tab-item {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 2px;
  cursor: pointer; padding-top: 4px;
}
.wx-tab-icon { width: 24px; height: 24px; }
.wx-tab-text { font-size: 10px; color: #9a9a9a; }
.wx-tab-item.active .wx-tab-text { color: #07c160; }
.wx-tip-toast {
  position: absolute; top: 45%; left: 50%; transform: translate(-50%, -50%);
  background: rgba(0, 0, 0, 0.78); color: #fff;
  padding: 10px 18px; border-radius: 12px;
  font-size: 12px; white-space: nowrap; z-index: 50;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
}
.home-tip-toast {
  position: absolute; top: 40%; left: 50%; transform: translate(-50%, -50%);
  background: rgba(0, 0, 0, 0.78); color: #fff;
  padding: 10px 18px; border-radius: 12px;
  font-size: 13px; white-space: nowrap; z-index: 50;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
}
/* 狸猫订票确认弹窗 */
.ticket-confirm-mask {
  position: absolute; inset: 0; z-index: 60;
  background: rgba(0, 0, 0, 0.45);
  display: flex; align-items: center; justify-content: center;
}
.ticket-confirm-dialog {
  width: 78%; background: #fff; border-radius: 16px;
  overflow: hidden; box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
}
.ticket-confirm-title {
  padding: 24px 20px; text-align: center;
  font-size: 16px; font-weight: 600; color: #1a1a1a;
}
.ticket-confirm-btns { display: flex; border-top: 0.5px solid #eee; }
.tc-btn {
  flex: 1; border: none; padding: 13px 0;
  font-size: 15px; cursor: pointer;
  transition: background 0.15s;
}
.tc-btn-cancel { background: #e8e8e8; color: #888; }
.tc-btn-cancel:active { background: #ddd; }
.tc-btn-enter { background: #1a1a1a; color: #fff; }
.tc-btn-enter:active { background: #333; }
.wx-tip-fade-enter-active, .wx-tip-fade-leave-active { transition: opacity 0.25s; }
.wx-tip-fade-enter-from, .wx-tip-fade-leave-to { opacity: 0; }
/* 弹窗专用动画：弹出快、关闭快 */
.dialog-pop-enter-active { transition: opacity 0.18s; }
.dialog-pop-leave-active { transition: opacity 0.12s; }
.dialog-pop-enter-from, .dialog-pop-leave-to { opacity: 0; }
.dialog-pop-enter-active .ticket-confirm-dialog { animation: dialog-scale-in 0.18s ease; }
@keyframes dialog-scale-in { from { transform: scale(0.9); opacity: 0; } to { transform: scale(1); opacity: 1; } }

/* miniapp CSS 已删除，小程序页面移至 /design/decorate 路由 */
</style>





