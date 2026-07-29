<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="layout-wrapper">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ collapsed: isCollapsed }">
      <div class="sidebar-logo">
        <img :src="brandStore.logoUrl" alt="logo" v-if="!isCollapsed" />
        <img :src="brandStore.logoUrl" alt="logo" class="logo-collapsed" v-else />
        <span v-if="!isCollapsed" class="logo-text">{{ brandStore.systemName }}</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapsed"
        :collapse-transition="false"
        background-color="transparent"
        text-color="#1a1a1a"
        active-text-color="#1a1a1a"
        router
      >
        <!-- 首页 -->
        <el-menu-item index="/dashboard">
          <span class="menu-icon" v-html="icons.dashboard"></span>
          <template #title>首页</template>
        </el-menu-item>

        <!-- 轨迹 -->
        <el-menu-item index="/track">
          <span class="menu-icon" v-html="icons.track"></span>
          <template #title>轨迹</template>
        </el-menu-item>

        <!-- 订单 -->
        <el-sub-menu index="order">
          <template #title>
            <span class="menu-icon" v-html="icons.order"></span>
            <span>订单</span>
          </template>
          <el-menu-item index="/order/list">订单列表</el-menu-item>
          <el-menu-item index="/order/refunds">退款记录</el-menu-item>
        </el-sub-menu>

        <!-- 售票 -->
        <el-sub-menu index="ticket">
          <template #title>
            <span class="menu-icon" v-html="icons.ticket"></span>
            <span>售票</span>
          </template>
          <el-menu-item index="/ticket/stations">站点管理</el-menu-item>
          <el-menu-item index="/ticket/routes">线路管理</el-menu-item>
          <el-menu-item index="/ticket/trips">班次管理</el-menu-item>
          <el-menu-item index="/ticket/vehicles">车辆管理</el-menu-item>
          <el-menu-item index="/ticket/cargo">托运管理</el-menu-item>
        </el-sub-menu>

        <!-- 用户 -->
        <el-sub-menu index="user">
          <template #title>
            <span class="menu-icon" v-html="icons.user"></span>
            <span>用户</span>
          </template>
          <el-menu-item index="/user/list">小程序用户</el-menu-item>
          <el-menu-item index="/user/passengers">常用乘客</el-menu-item>
          <el-menu-item index="/user/idcard-verify">实名认证缓存</el-menu-item>
        </el-sub-menu>

        <!-- 设计 -->
        <el-sub-menu index="design">
          <template #title>
            <span class="menu-icon" v-html="icons.design"></span>
            <span>设计</span>
          </template>
          <el-menu-item index="/design/decorate">小程序装修</el-menu-item>
          <el-menu-item index="/design/banners">轮播图装修</el-menu-item>
          <el-menu-item index="/design/coupon-display">优惠券展示</el-menu-item>
          <el-menu-item index="/design/phone">电话设置</el-menu-item>
        </el-sub-menu>

        <!-- 营销 -->
        <el-sub-menu index="marketing">
          <template #title>
            <span class="menu-icon" v-html="icons.marketing"></span>
            <span>营销</span>
          </template>
          <el-menu-item index="/marketing/coupons">优惠券管理</el-menu-item>
          <el-menu-item index="/marketing/point-rules">积分规则</el-menu-item>
          <el-menu-item index="/marketing/user-points">用户积分明细</el-menu-item>
          <el-menu-item index="/marketing/distribution">发放记录</el-menu-item>
        </el-sub-menu>

        <!-- 管理 -->
        <el-sub-menu index="manage">
          <template #title>
            <span class="menu-icon" v-html="icons.manage"></span>
            <span>管理</span>
          </template>
          <el-menu-item index="/setting/drivers">司机管理</el-menu-item>
          <el-menu-item v-if="authStore.isSuperAdmin()" index="/setting/admins">管理员管理</el-menu-item>
          <el-menu-item v-if="authStore.isSuperAdmin()" index="/system/agreement">协议政策</el-menu-item>
        </el-sub-menu>

      </el-menu>

      <!-- 设置 - 底部，右侧飞出面板 -->
      <div class="sidebar-bottom">
        <div class="settings-trigger" :class="{ active: isSettingsActive }" @click="showSettingsPanel = !showSettingsPanel">
          <span class="menu-icon" v-html="icons.setting"></span>
          <span class="settings-label">设置</span>
        </div>
        <transition name="flyout">
          <div v-if="showSettingsPanel" class="settings-flyout">
            <!-- 品牌信息展示卡（圆形头像 + 编辑图标，点击进入修改） -->
            <div class="flyout-brand-card" @click="goSetting('/system/brand')">
              <div class="flyout-brand-avatar">
                <img :src="brandStore.logoUrl" alt="logo" />
              </div>
              <div class="flyout-brand-name">{{ brandStore.systemName }}</div>
              <span class="flyout-brand-edit">
                <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </span>
            </div>
            <div class="flyout-item" @click="goSetting('/system/password')">
              <span class="flyout-icon" v-html="icons.changePassword"></span>
              <span>修改密码</span>
            </div>
            <div class="flyout-item" @click="goSetting('/system/config')">
              <span class="flyout-icon" v-html="icons.systemConfig"></span>
              <span>系统配置</span>
            </div>
            <div class="flyout-item" @click="goSetting('/system/logs')">
              <span class="flyout-icon" v-html="icons.operationLog"></span>
              <span>操作日志</span>
            </div>
            <div class="flyout-divider"></div>
            <div class="flyout-item logout" @click="handleLogout">
              <span class="flyout-icon" v-html="icons.logout"></span>
              <span>退出登录</span>
            </div>
          </div>
        </transition>
        <!-- 遮罩层，点击外部关闭 -->
        <div v-if="showSettingsPanel" class="flyout-mask" @click="showSettingsPanel = false"></div>
      </div>
    </aside>

    <!-- 右侧内容区 -->
    <div class="main-section">
      <!-- 顶栏 -->
      <header class="topbar">
        <div class="topbar-left">
          <el-icon class="collapse-btn" @click="isCollapsed = !isCollapsed">
            <Fold v-if="!isCollapsed" />
            <Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentTitle">{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
      </header>

      <!-- 内容区 -->
      <main class="content-area">
        <router-view :key="route.path" />
      </main>
    </div>
    <!-- 数字员工：全局悬浮，独立于各页面层叠上下文，避免被遮挡 -->
    <AIAssistant />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBrandStore } from '@/stores/brand'
import { Fold, Expand } from '@element-plus/icons-vue'
import AIAssistant from '@/components/AIAssistant.vue'

// 导入SVG图标
import dashboardIcon from '@/assets/icons/dashboard.svg?raw'
import ticketIcon from '@/assets/icons/ticket.svg?raw'
import orderIcon from '@/assets/icons/order.svg?raw'
import userIcon from '@/assets/icons/user.svg?raw'
import manageIcon from '@/assets/icons/manage.svg?raw'
import designIcon from '@/assets/icons/design.svg?raw'
import marketingIcon from '@/assets/icons/marketing.svg?raw'
import settingIcon from '@/assets/icons/setting.svg?raw'
import changePasswordIcon from '@/assets/icons/change-password.svg?raw'
import systemConfigIcon from '@/assets/icons/system-config.svg?raw'
import operationLogIcon from '@/assets/icons/operation-log.svg?raw'
import logoutIcon from '@/assets/icons/logout.svg?raw'
import trackIcon from '@/assets/icons/track.svg?raw'

const icons = {
  dashboard: dashboardIcon,
  ticket: ticketIcon,
  order: orderIcon,
  user: userIcon,
  manage: manageIcon,
  design: designIcon,
  marketing: marketingIcon,
  setting: settingIcon,
  changePassword: changePasswordIcon,
  systemConfig: systemConfigIcon,
  operationLog: operationLogIcon,
  logout: logoutIcon,
  track: trackIcon,
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const brandStore = useBrandStore()
const isCollapsed = ref(false)

const activeMenu = computed(() => route.path)
const currentTitle = computed(() => route.meta.title as string)
const isSettingsActive = computed(() => route.path.startsWith('/system'))
const showSettingsPanel = ref(false)

const goSetting = (path: string) => {
  showSettingsPanel.value = false
  router.push(path)
}

const handleLogout = async () => {
  showSettingsPanel.value = false
  await authStore.logout()
  router.push('/login')
}
</script>

<style scoped>
/* 整页布局 - position:fixed 彻底钉死在视口上，任何滚动都无法移动 */
.layout-wrapper {
  display: flex;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
}

/* 侧边栏 - 毛玻璃 + 淡蓝调色 */
.sidebar {
  width: 220px;
  background: rgba(253, 254, 255, 0.85);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  display: flex;
  flex-direction: column;
  transition: width 0.3s;
  flex-shrink: 0;
  border-right: 1px solid rgba(255, 255, 255, 0.4);
  z-index: 100; /* 确保侧边栏及飞出面板在主内容区之上，避免层叠上下文遮挡 */
}
.sidebar.collapsed {
  width: 64px;
}

.sidebar-logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-bottom: 1px solid rgba(80, 120, 180, 0.06);
  padding: 0 12px;
}
.sidebar-logo img {
  width: 36px;
  height: 36px;
  border-radius: 6px;
  object-fit: contain;
}
.sidebar-logo .logo-collapsed {
  width: 36px;
  height: 36px;
  border-radius: 6px;
  object-fit: contain;
}
.sidebar-logo .logo-text {
  color: #1a1a1a;
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
}

.sidebar :deep(.el-menu) {
  border-right: none;
  flex: 1;
  overflow-y: auto;
  overflow-x: visible;
}

/* 设置 - 底部固定 */
.sidebar-bottom {
  flex-shrink: 0;
  border-top: 1px solid rgba(80, 120, 180, 0.06);
  position: relative;
}
.settings-trigger {
  display: flex;
  align-items: center;
  height: 56px;
  padding: 0 20px;
  cursor: pointer;
  color: #1a1a1a;
  font-size: 14px;
  transition: background 0.2s;
}
.settings-trigger:hover {
  background-color: rgba(0, 0, 0, 0.06);
}
.settings-trigger.active {
  color: #1a1a1a;
  background-color: rgba(0, 0, 0, 0.06);
  font-weight: 600;
}
.settings-trigger.active .menu-icon :deep(path) {
  fill: #1a1a1a !important;
}
.settings-label {
  white-space: nowrap;
}

/* 右侧飞出面板 */
.settings-flyout {
  position: absolute;
  left: 100%;
  bottom: 0;
  width: 220px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.12);
  padding: 6px 0;
  z-index: 2000;
  margin-left: 8px;
}
/* 品牌信息展示卡（精致高级风格：圆形头像 + 编辑图标） */
.flyout-brand-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
  border-bottom: 1px solid #f0f0f0;
}
.flyout-brand-card:hover {
  background: linear-gradient(135deg, #f9fbff 0%, #f0f4f8 100%);
}
.flyout-brand-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  border: 2px solid #fff;
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.12);
  background: #f5f5f5;
  display: flex;
  align-items: center;
  justify-content: center;
}
.flyout-brand-avatar img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.flyout-brand-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  letter-spacing: 0.3px;
}
.flyout-brand-edit {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #888;
  transition: background 0.15s, color 0.15s, transform 0.15s;
}
.flyout-brand-card:hover .flyout-brand-edit {
  background: #1a1a1a;
  color: #fff;
  transform: scale(1.08);
}
.flyout-item {
  padding: 0 20px;
  height: 40px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: #1a1a1a;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s;
}
.flyout-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}
.flyout-icon :deep(svg) {
  width: 16px;
  height: 16px;
}
.flyout-item:hover {
  background-color: #f5f5f5;
}
.flyout-item.logout {
  color: #1a1a1a;
}
.flyout-divider {
  height: 1px;
  background: #f0f0f0;
  margin: 4px 0;
}

/* 飞出动画 - 仅位移，不用opacity，避免滑出过程中面板透明 */
.flyout-enter-active,
.flyout-leave-active {
  transition: transform 0.2s ease;
  transform: translateX(0);
}
.flyout-enter-from,
.flyout-leave-to {
  transform: translateX(-12px);
}

/* 遮罩层 */
.flyout-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1999;
}

/* 菜单项 hover */
.sidebar :deep(.el-menu-item:hover),
.sidebar :deep(.el-sub-menu__title:hover) {
  background-color: rgba(80, 120, 180, 0.05) !important;
}

/* 选中项 - 淡蓝背景，无横线 */
.sidebar :deep(.el-menu-item.is-active) {
  background-color: rgba(80, 120, 180, 0.07) !important;
  font-weight: 600;
}
.sidebar :deep(.el-menu-item.is-active) .menu-icon :deep(path) {
  fill: #1a1a1a !important;
}

/* 菜单图标 */
.menu-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin-right: 8px;
}
.menu-icon :deep(svg) {
  width: 20px;
  height: 20px;
  fill: #1a1a1a;
}
.menu-icon :deep(path) {
  fill: #1a1a1a !important;
}

/* 折叠模式 - 强制图标显示，隐藏文字 */
.sidebar :deep(.el-menu--collapse) {
  width: 64px;
}
.sidebar :deep(.el-menu--collapse .el-menu-item),
.sidebar :deep(.el-menu--collapse .el-sub-menu__title) {
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  padding: 0 !important;
}
/* 隐藏文字span和箭头，只留图标 */
.sidebar :deep(.el-menu--collapse .el-sub-menu__title > span:not(.menu-icon)),
.sidebar :deep(.el-menu--collapse .el-menu-item > span:not(.menu-icon)) {
  display: none !important;
}
.sidebar :deep(.el-menu--collapse .el-sub-menu__icon-arrow) {
  display: none !important;
}
/* 强制图标尺寸和可见性，防止flex压缩和Element Plus隐藏 */
.sidebar :deep(.el-menu--collapse .menu-icon) {
  display: inline-flex !important;
  margin-right: 0 !important;
  width: 24px !important;
  height: 24px !important;
  flex-shrink: 0 !important;
  flex-grow: 0 !important;
  overflow: visible !important;
  visibility: visible !important;
}
.sidebar :deep(.el-menu--collapse .menu-icon svg) {
  width: 20px !important;
  height: 20px !important;
  visibility: visible !important;
}
.sidebar :deep(.el-menu--collapse .menu-icon svg path) {
  visibility: visible !important;
}

/* 主内容区 */
.main-section {
  flex: 1;
  min-width: 0; /* 关键：允许flex子项缩小到内容宽度以下，防止宽内容撑开布局 */
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 顶栏 - 毛玻璃 + 同色系 */
.topbar {
  height: 56px;
  background: rgba(253, 254, 255, 0.85);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.4);
  flex-shrink: 0;
}
.topbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  margin-right: 8px;
}
.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: #595959;
}
.collapse-btn:hover {
  color: #1a1a1a;
}

/* 内容区 - 双向滚动，内容在中间滚动，不影响sidebar和topbar */
.content-area {
  flex: 1;
  min-width: 0; /* 允许缩小，防止宽表格撑开 */
  overflow: auto; /* 双向滚动：宽内容横向滚动，长内容纵向滚动，都限制在此区域内 */
  background: #f5f6f8;
}
</style>
