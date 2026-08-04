<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="decorate-wrapper">
      <!-- 左侧：组件配置区 -->
      <div class="config-panel">
        <!-- 首页装修 -->
        <template v-if="currentPage==='home'">
        <div class="panel-header">
          <span class="page-title">小程序首页装修</span>
          <button class="save-btn-black" @click="handleSave" :disabled="saving">
            <span class="save-btn-icon" v-html="saveConfigIcon"></span>
            <span>{{ saving ? '保存中...' : '保存配置' }}</span>
          </button>
        </div>
        <el-alert
          title="拖拽组件可调整首页展示顺序，切换开关控制是否显示。点击组件可在右侧预览中定位。"
          type="info" :closable="false" show-icon
          style="margin-bottom: 16px"
        />

        <div class="component-list" v-loading="loading">
          <div
            v-for="(item, index) in layout"
            :key="item.type"
            class="component-item"
            :class="{ hidden: !item.visible, active: activeIndex === index, 'drag-over': dragOverIndex === index }"
            draggable="true"
            @click="scrollToComponent(index)"
            @dragstart="onDragStart(index)"
            @dragover.prevent="onDragOver(index)"
            @drop="onDrop(index)"
            @dragend="onDragEnd"
            :style="{ opacity: dragIndex === index ? 0.4 : 1, transform: dragIndex === index ? 'scale(0.98)' : '' }"
          >
            <div class="drag-handle">
              <el-icon><Rank /></el-icon>
            </div>
            <div class="component-icon" v-html="componentIcons[item.type]"></div>
            <div class="component-info">
              <div class="component-title">{{ item.title }}</div>
              <div class="component-desc">{{ componentDesc[item.type] }}</div>
            </div>
            <div class="component-actions">
              <el-button
                size="small" :disabled="index === 0" circle
                @click.stop="moveUp(index)"
              ><span class="move-btn-icon" v-html="moveUpIcon"></span></el-button>
              <el-button
                size="small" :disabled="index === layout.length - 1" circle
                @click.stop="moveDown(index)"
              ><span class="move-btn-icon" v-html="moveDownIcon"></span></el-button>
              <el-switch v-model="item.visible" size="small" @click.stop />
            </div>
          </div>
        </div>

        <!-- 公告编辑区（可直接修改首页公告） -->
        <div class="notice-editor" v-if="hasNoticeComponent">
          <div class="notice-editor-header">
            <span class="notice-editor-title">公告内容</span>
            <button class="save-btn-black" @click="saveNotice" :disabled="savingNotice">
                        <span class="save-btn-icon" v-html="saveNoticeIcon"></span>
                        <span>{{ savingNotice ? '保存中...' : '保存公告' }}</span>
                      </button>
          </div>
          <el-input
            v-model="noticeEditing"
            type="textarea"
            :rows="3"
            maxlength="200"
            show-word-limit
            placeholder="请输入首页公告内容，如：欢迎乘坐巴士出行..."
            @input="noticeText = noticeEditing"
          />
          <div class="notice-editor-hint">修改后点击「保存公告」生效，右侧预览实时同步</div>
        </div>

        <div class="quick-tips">
          <div class="tip-item">
            <el-icon><InfoFilled /></el-icon>
            <span>轮播图：在「设计 - 轮播图装修」中管理图片</span>
          </div>
          <div class="tip-item">
            <el-icon><InfoFilled /></el-icon>
            <span>优惠券：在「设计 - 优惠券展示」中选择展示的券</span>
          </div>
        </div>
        </template>

        <!-- 订单页装修 -->
        <template v-else-if="currentPage==='order'">
          <div class="panel-header">
            <span class="page-title">订单页装修</span>
            <button class="save-btn-black" @click="handleSave" :disabled="saving">
              <span class="save-btn-icon" v-html="saveConfigIcon"></span>
              <span>{{ saving ? '保存中...' : '保存配置' }}</span>
            </button>
          </div>
          <el-alert
            title="拖拽调整订单筛选标签的展示顺序，切换开关控制是否显示。"
            type="info" :closable="false" show-icon
            style="margin-bottom: 16px"
          />
          <div class="component-list" v-loading="loadingOrder">
            <div
              v-for="(item, index) in orderTabsLayout"
              :key="item.type"
              class="component-item"
              :class="{ hidden: !item.visible }"
            >
              <div class="drag-handle"><el-icon><Rank /></el-icon></div>
              <div class="component-info">
                <div class="component-title">{{ item.title }}</div>
                <div class="component-desc">订单筛选标签</div>
              </div>
              <div class="component-actions">
                <el-button size="small" :disabled="index === 0" circle @click.stop="moveUpGeneric(orderTabsLayout, index)"><span class="move-btn-icon" v-html="moveUpIcon"></span></el-button>
                <el-button size="small" :disabled="index === orderTabsLayout.length - 1" circle @click.stop="moveDownGeneric(orderTabsLayout, index)"><span class="move-btn-icon" v-html="moveDownIcon"></span></el-button>
                <el-switch v-model="item.visible" size="small" @click.stop />
              </div>
            </div>
          </div>
        </template>

        <!-- 我的页装修 -->
        <template v-else-if="currentPage==='mine'">
          <div class="panel-header">
            <span class="page-title">我的页装修</span>
            <button class="save-btn-black" @click="handleSave" :disabled="saving">
              <span class="save-btn-icon" v-html="saveConfigIcon"></span>
              <span>{{ saving ? '保存中...' : '保存配置' }}</span>
            </button>
          </div>
          <el-alert
            title="上下移动调整菜单展示顺序，切换开关控制是否显示。名称与图标固定使用小程序真实样式。"
            type="info" :closable="false" show-icon
            style="margin-bottom: 16px"
          />
          <div class="section-label">我的订单分类</div>
          <div class="component-list" v-loading="loadingMine">
            <div
              v-for="(item, index) in mineOrderGridLayout"
              :key="item.type"
              class="component-item"
              :class="{ hidden: !item.visible }"
            >
              <img class="menu-thumb" :src="mineIcons[item.type]" />
              <div class="component-info">
                <div class="component-title">{{ item.title }}</div>
                <div class="component-desc">订单分类入口</div>
              </div>
              <div class="component-actions">
                <el-button size="small" :disabled="index === 0" circle @click.stop="moveUpGeneric(mineOrderGridLayout, index)"><span class="move-btn-icon" v-html="moveUpIcon"></span></el-button>
                <el-button size="small" :disabled="index === mineOrderGridLayout.length - 1" circle @click.stop="moveDownGeneric(mineOrderGridLayout, index)"><span class="move-btn-icon" v-html="moveDownIcon"></span></el-button>
                <el-switch v-model="item.visible" size="small" @click.stop />
              </div>
            </div>
          </div>
          <div class="mine-menu-layout-row">
            <div class="section-label">功能菜单</div>
            <div class="layout-switch">
              <div class="layout-btn" :class="{ active: mineMenuLayoutType === 'list' }" @click="mineMenuLayoutType = 'list'">
                <span class="layout-btn-icon" v-html="layoutListIcon"></span>
                <span>列表</span>
              </div>
              <div class="layout-btn" :class="{ active: mineMenuLayoutType === 'grid' }" @click="mineMenuLayoutType = 'grid'">
                <span class="layout-btn-icon" v-html="layoutGridIcon"></span>
                <span>网格</span>
              </div>
            </div>
          </div>
          <div class="component-list">
            <div
              v-for="(item, index) in mineMenuLayout"
              :key="item.type"
              class="component-item"
              :class="{ hidden: !item.visible }"
            >
              <img class="menu-thumb" :src="mineIcons[item.type]" />
              <div class="component-info">
                <div class="component-title">{{ item.title }}</div>
                <div class="component-desc">功能菜单入口</div>
              </div>
              <div class="component-actions">
                <el-button size="small" :disabled="index === 0" circle @click.stop="moveUpGeneric(mineMenuLayout, index)"><span class="move-btn-icon" v-html="moveUpIcon"></span></el-button>
                <el-button size="small" :disabled="index === mineMenuLayout.length - 1" circle @click.stop="moveDownGeneric(mineMenuLayout, index)"><span class="move-btn-icon" v-html="moveDownIcon"></span></el-button>
                <el-switch v-model="item.visible" size="small" @click.stop />
              </div>
            </div>
          </div>
          <div class="logout-position-config">
            <div class="logout-position-config-label">
              <span class="logout-position-config-icon" v-html="logoutIcon"></span>
              <span>退出按钮位置</span>
            </div>
            <div class="logout-position-switch">
              <div class="logout-position-btn" :class="{ active: logoutPosition === 'mine_bottom' }" @click="logoutPosition = 'mine_bottom'">我的页底部</div>
              <div class="logout-position-btn" :class="{ active: logoutPosition === 'profile_detail' }" @click="logoutPosition = 'profile_detail'">个人中心详情页</div>
            </div>
          </div>
        </template>

        <!-- 页面切换（底部固定） -->
        <div class="page-tabs page-tabs-bottom">
          <div class="page-tab" :class="{active: currentPage==='home'}" @click="switchPage('home')">首页</div>
          <div class="page-tab" :class="{active: currentPage==='order'}" @click="switchPage('order')">订单</div>
          <div class="page-tab" :class="{active: currentPage==='mine'}" @click="switchPage('mine')">我的</div>
        </div>
      </div>

      <!-- 右侧：手机预览区 -->
      <div class="preview-panel">
        <!-- 左侧控制栏（D-pad+机型切换） -->
        <div class="left-controls">
        <!-- 模拟器导航控制 -->
        <div class="simulator-dpad">
          <div class="dpad-row">
            <div class="dpad-spacer"></div>
            <div class="dpad-btn" @click="scrollPhone('up')" title="向上滚动">
              <span class="dpad-icon" v-html="navUpIcon"></span>
            </div>
            <div class="dpad-spacer"></div>
          </div>
          <div class="dpad-row">
            <div class="dpad-btn" @click="scrollPhone('left')" title="向左滚动">
              <span class="dpad-icon" v-html="navLeftIcon"></span>
            </div>
            <div class="dpad-btn dpad-center" @click="scrollPhone('center')" title="回到顶部">
              <span class="dpad-icon" v-html="navCenterIcon"></span>
            </div>
            <div class="dpad-btn" @click="scrollPhone('right')" title="向右滚动">
              <span class="dpad-icon" v-html="navRightIcon"></span>
            </div>
          </div>
          <div class="dpad-row">
            <div class="dpad-spacer"></div>
            <div class="dpad-btn" @click="scrollPhone('down')" title="向下滚动">
              <span class="dpad-icon" v-html="navDownIcon"></span>
            </div>
            <div class="dpad-spacer"></div>
          </div>
        </div>

        <!-- 机型切换器包装 -->
        <div class="model-switcher-wrapper">
        <!-- 当前机型显示 -->
        <div class="model-name-display">
          {{ modelInfo[currentModel].name }}
          <span class="model-os-badge">{{ modelInfo[currentModel].sub }}</span>
        </div>

        <!-- 机型切换按钮 -->
        <div class="model-switch-btn" @click="showModelPicker = !showModelPicker">
          <span class="model-switch-icon" v-html="phoneSwitchIcon"></span>
          <span>切换机型</span>
        </div>

        <!-- 点击外部关闭遮罩 -->
        <transition name="model-picker-fade">
          <div v-if="showModelPicker" class="model-picker-overlay" @click="showModelPicker = false"></div>
        </transition>

        <!-- 机型选择弹窗 -->
        <transition name="model-picker">
          <div v-if="showModelPicker" class="model-picker-popup">
            <!-- 标题栏 + 关闭按钮 -->
            <div class="model-picker-header">
              <span class="model-picker-title">选择机型</span>
              <div class="model-picker-close" @click="showModelPicker = false">×</div>
            </div>
            <div class="model-option" :class="{ active: currentModel==='apple' }" @click="switchPhoneModel('apple')">
              <div class="model-option-back" :style="{ backgroundImage: `url(${appleBackImg})` }">
                <div class="model-option-logo" v-html="brandLogos.apple"></div>
              </div>
              <div class="model-option-info">
                <div class="model-option-name">iPhone 17 Pro Max</div>
                <div class="model-option-os">iOS 系统</div>
              </div>
            </div>
            <div class="model-option" :class="{ active: currentModel==='pixel' }" @click="switchPhoneModel('pixel')">
              <div class="model-option-back" :style="{ backgroundImage: `url(${pixelBackImg})` }">
                <div class="model-option-logo" v-html="brandLogos.pixel"></div>
              </div>
              <div class="model-option-info">
                <div class="model-option-name">Pixel 10</div>
                <div class="model-option-os">Android 系统</div>
              </div>
            </div>
            <div class="model-option" :class="{ active: currentModel==='huawei' }" @click="switchPhoneModel('huawei')">
              <div class="model-option-back" :style="{ backgroundImage: `url(${huaweiBackImg})` }">
                <div class="model-option-logo" v-html="brandLogos.huawei"></div>
              </div>
              <div class="model-option-info">
                <div class="model-option-name">Pura 80 Pro</div>
                <div class="model-option-os">HarmonyOS 系统</div>
              </div>
            </div>
            <div class="model-option" :class="{ active: currentModel==='samsung' }" @click="switchPhoneModel('samsung')">
              <div class="model-option-back" :style="{ backgroundImage: `url(${samsungBackImg})` }">
                <div class="model-option-logo" v-html="brandLogos.samsung"></div>
              </div>
              <div class="model-option-info">
                <div class="model-option-name">Galaxy S26 Ultra</div>
                <div class="model-option-os">One UI 系统</div>
              </div>
            </div>
          </div>
        </transition>
        </div><!-- model-switcher-wrapper -->
        </div><!-- left-controls -->

        <!-- 3D翻转容器 -->
        <div class="phone-3d-container">
        <div class="phone-3d-flipper" :class="[displayModel, { flipping: isFlipping }]">
          <!-- 背面（真实手机背面图片，已裁剪白边满屏铺满） -->
          <div class="phone-back-face">
            <img :src="modelInfo[displayModel].back" class="back-face-img" />
          </div>
          <!-- 正面 -->
          <div class="phone-front-face">
        <div class="phone-wrapper">
        <!-- 侧边按键（仅苹果） -->
        <div class="phone-side-btn side-btn-action" v-if="currentModel==='apple'"></div>
        <div class="phone-side-btn side-btn-vol-up" v-if="currentModel==='apple'"></div>
        <div class="phone-side-btn side-btn-vol-down" v-if="currentModel==='apple'"></div>
        <div class="phone-frame" :class="currentModel">
          <!-- Apple: 灵动岛 -->
          <div v-if="currentModel==='apple'" class="dynamic-island"><div class="dynamic-island-camera"></div></div>
          <!-- Pixel: 打孔屏 -->
          <div v-else-if="currentModel==='pixel'" class="punch-hole-camera"></div>
          <!-- Huawei: 圆形打孔屏（Pura系列） -->
          <div v-else-if="currentModel==='huawei'" class="huawei-punch-hole"></div>
          <!-- Samsung: 圆形打孔屏（居中偏上） -->
          <div v-else-if="currentModel==='samsung'" class="samsung-punch-hole"></div>
          <!-- 状态栏 -->
          <div class="phone-statusbar" :class="'sb-' + currentModel">
            <span class="status-time">9:41</span>
            <!-- Apple iOS状态栏：圆角信号条 + 3弧WiFi + 胶囊电池 -->
            <div class="status-right" v-if="currentModel==='apple'">
              <svg width="17" height="11" viewBox="0 0 17 11" fill="#1a1a1a"><rect x="0" y="7" width="3" height="4" rx="0.5"/><rect x="4.5" y="5" width="3" height="6" rx="0.5"/><rect x="9" y="3" width="3" height="8" rx="0.5"/><rect x="13.5" y="1" width="3" height="10" rx="0.5"/></svg>
              <svg width="15" height="11" viewBox="0 0 15 11" fill="#1a1a1a"><path d="M7.5 1.5C4.6 1.5 1.9 2.6 0 4.5l1.4 1.4C3 4.3 5.2 3.5 7.5 3.5s4.5 0.8 6.1 2.4L15 4.5C13.1 2.6 10.4 1.5 7.5 1.5z"/><path d="M7.5 5C5.8 5 4.2 5.7 3 6.8l1.4 1.4c0.9-0.8 2-1.2 3.1-1.2s2.2 0.4 3.1 1.2L12 6.8C10.8 5.7 9.2 5 7.5 5z"/><circle cx="7.5" cy="9" r="1.5"/></svg>
              <svg width="27" height="12" viewBox="0 0 27 12"><rect x="1" y="1" width="22" height="10" rx="3" fill="none" stroke="#1a1a1a" stroke-width="1" opacity="0.35"/><rect x="2.5" y="2.5" width="19" height="7" rx="1.5" fill="#1a1a1a"/><rect x="24" y="4" width="2" height="4" rx="0.5" fill="#1a1a1a" opacity="0.4"/></svg>
            </div>
            <!-- Pixel Android状态栏：直角信号条 + Material WiFi + 宽胶囊电池 -->
            <div class="status-right" v-else-if="currentModel==='pixel'">
              <svg width="18" height="12" viewBox="0 0 18 12" fill="#1a1a1a"><rect x="0" y="9" width="3" height="3"/><rect x="4.5" y="6" width="3" height="6"/><rect x="9" y="3" width="3" height="9"/><rect x="13.5" y="0" width="3" height="12"/></svg>
              <svg width="16" height="12" viewBox="0 0 16 12" fill="#1a1a1a"><path d="M8 1.5C4.6 1.5 1.8 2.8 0 4.8l1.8 1.8C3.2 5.4 5.4 4.5 8 4.5s4.8 0.9 6.2 2.1L16 4.8C14.2 2.8 11.4 1.5 8 1.5z"/><path d="M8 5.5C5.8 5.5 3.9 6.3 2.6 7.6l1.8 1.8C5.2 8.8 6.5 8.3 8 8.3s2.8 0.5 3.6 1.3l1.8-1.8C12.1 6.3 10.2 5.5 8 5.5z"/><circle cx="8" cy="11" r="1.5"/></svg>
              <svg width="28" height="13" viewBox="0 0 28 13"><rect x="1" y="1" width="23" height="11" rx="2.5" fill="none" stroke="#1a1a1a" stroke-width="1"/><rect x="2.5" y="2.5" width="18" height="8" rx="1.5" fill="#1a1a1a"/><rect x="25" y="4" width="2" height="5" rx="1" fill="#1a1a1a" opacity="0.4"/></svg>
            </div>
            <!-- Huawei HarmonyOS状态栏：微圆角信号条 + HarmonyOS WiFi + 紧凑电池 -->
            <div class="status-right" v-else-if="currentModel==='huawei'">
              <svg width="17" height="12" viewBox="0 0 17 12" fill="#1a1a1a"><rect x="0" y="8" width="3" height="4" rx="0.5"/><rect x="4.5" y="5" width="3" height="7" rx="0.5"/><rect x="9" y="3" width="3" height="9" rx="0.5"/><rect x="13.5" y="0" width="3" height="12" rx="0.5"/></svg>
              <svg width="15" height="12" viewBox="0 0 15 12" fill="#1a1a1a"><path d="M7.5 1C4 1 1 2.5 0 4l1.5 1.5C2.5 4 5 3 7.5 3s5 1 6 2.5L15 4C14 2.5 11 1 7.5 1z"/><path d="M7.5 4.5C5.5 4.5 3.5 5.5 2.5 6.5L4 8c0.8-0.8 2-1.5 3.5-1.5s2.7 0.7 3.5 1.5l1.5-1.5C11 5.5 9.5 4.5 7.5 4.5z"/><circle cx="7.5" cy="10" r="1.5"/></svg>
              <svg width="25" height="12" viewBox="0 0 25 12"><rect x="0.5" y="1" width="21" height="10" rx="2.5" fill="none" stroke="#1a1a1a" stroke-width="1"/><rect x="2" y="2.5" width="16" height="7" rx="1" fill="#1a1a1a"/><rect x="22.5" y="4" width="2" height="4" rx="0.5" fill="#1a1a1a" opacity="0.4"/></svg>
            </div>
            <!-- Samsung One UI状态栏：圆角信号条 + One UI WiFi + 宽电池 -->
            <div class="status-right" v-else-if="currentModel==='samsung'">
              <svg width="18" height="12" viewBox="0 0 18 12" fill="#1a1a1a"><rect x="0" y="8" width="3" height="4" rx="1"/><rect x="4.5" y="6" width="3" height="6" rx="1"/><rect x="9" y="3" width="3" height="9" rx="1"/><rect x="13.5" y="0" width="3" height="12" rx="1"/></svg>
              <svg width="16" height="12" viewBox="0 0 16 12" fill="#1a1a1a"><path d="M8 1.5C5 1.5 2.2 2.7 0.3 4.7l1.5 1.5C3.3 5.3 5.5 4.5 8 4.5s4.7 0.8 6.2 2.2l1.5-1.5C13.8 2.7 11 1.5 8 1.5z"/><path d="M8 5C6 5 4.2 5.8 3 7l1.5 1.5C5.3 7.7 6.6 7.3 8 7.3s2.7 0.4 3.5 1.2L13 7C11.8 5.8 10 5 8 5z"/><circle cx="8" cy="10.5" r="1.3"/></svg>
              <svg width="27" height="13" viewBox="0 0 27 13"><rect x="0.5" y="1" width="22" height="11" rx="3" fill="none" stroke="#1a1a1a" stroke-width="1" opacity="0.35"/><rect x="2" y="2.5" width="17" height="8" rx="2" fill="#1a1a1a"/><rect x="23.5" y="4" width="2" height="5" rx="1" fill="#1a1a1a" opacity="0.4"/></svg>
            </div>
          </div>

          <!-- 导航栏（毛玻璃效果，匹配真实小程序） -->
          <div class="phone-navbar">
            <span v-if="previewSubPage" class="navbar-back" @click="closeSubPage">‹</span>
            <span class="navbar-title">{{ navbarTitle }}</span>
          </div>

          <!-- 预览滚动内容 -->
          <div class="phone-scroll" :class="{ 'phone-scroll-detail': previewSubPage }" ref="phoneScrollRef">
            <!-- 主页面预览（子页面打开时隐藏） -->
            <template v-if="!previewSubPage">
            <!-- 首页预览 -->
            <template v-if="currentPage==='home'">
            <template v-for="(item, idx) in visibleLayout" :key="item.type">
              <!-- 轮播图预览（真实图片 + 自动轮播） -->
              <div v-if="item.type === 'banner'" class="preview-banner" :class="{ 'preview-highlight': activeIndex === getOriginalIndex(item.type) }" :ref="el => setComponentRef(item.type, el)">
                <div class="banner-viewport">
                  <!-- 有真实轮播图 -->
                  <div v-if="bannerList.length > 0" class="banner-track" :style="{ transform: `translateX(-${currentSlide * 100}%)` }">
                    <div
                      v-for="(banner, bi) in bannerList" :key="bi"
                      class="banner-slide slide-real"
                    >
                      <img :src="banner.image_url" class="banner-img" />
                      <div v-if="banner.title" class="banner-text" :class="getEffectClass(banner.title_effect)">
                        <div class="banner-title" :style="getTitleStyle(banner)">{{ banner.title }}</div>
                      </div>
                    </div>
                  </div>
                  <!-- 无轮播图空状态 -->
                  <div v-else class="banner-empty">
                    <div class="banner-empty-icon" v-html="busSvg"></div>
                    <span class="banner-empty-text">暂无轮播图，请在「轮播图装修」中添加</span>
                  </div>
                  <!-- 指示点 -->
                  <div v-if="bannerList.length > 1" class="banner-dots">
                    <span
                      v-for="(banner, bi) in bannerList" :key="'dot'+bi"
                      class="dot" :class="{ active: currentSlide === bi }"
                      @click="goToSlide(bi)"
                    ></span>
                  </div>
                </div>
              </div>

              <!-- 优惠券预览（真实数据 + 横向滑动） -->
              <div v-else-if="item.type === 'coupon'" class="preview-coupon" :class="{ 'preview-highlight': activeIndex === getOriginalIndex(item.type) }" :ref="el => setComponentRef(item.type, el)">
                <div class="coupon-section-header">
                  <span class="coupon-section-icon" v-html="couponTicketSvg"></span>
                  <span class="coupon-section-title">优惠福利</span>
                </div>
                <!-- 有优惠券 -->
                <div v-if="couponList.length > 0" class="preview-coupon-scroll">
                  <div v-for="coupon in couponList" :key="coupon.id" class="preview-coupon-card" :class="{ 'coupon-claimed': couponClaimedIds.has(coupon.id) }" @click.stop="claimCoupon(coupon.id)">
                    <div class="preview-coupon-left" :class="coupon.themeClass">
                      <span class="coupon-value">{{ coupon.valueText }}</span>
                      <span class="coupon-desc">{{ coupon.desc }}</span>
                    </div>
                    <div class="preview-coupon-right">
                      <div class="preview-coupon-label" :class="coupon.labelClass">{{ coupon.label }}</div>
                      <div class="preview-coupon-name">{{ coupon.name }}</div>
                      <div class="preview-coupon-claim-btn" :class="{ claimed: couponClaimedIds.has(coupon.id) }">{{ couponClaimedIds.has(coupon.id) ? '已领取' : '立即领取' }}</div>
                    </div>
                  </div>
                </div>
                <!-- 无优惠券空状态 -->
                <div v-else class="preview-empty-state">
                  <span>暂无展示优惠券，请在「优惠券展示」中选择</span>
                </div>
              </div>

              <!-- 公告预览（真实内容 + 跑马灯滚动） -->
              <div v-else-if="item.type === 'notice'" class="preview-notice" :class="{ 'preview-highlight': activeIndex === getOriginalIndex(item.type) }" :ref="el => setComponentRef(item.type, el)">
                <span class="notice-megaphone-icon" v-html="megaphoneSvg"></span>
                <div class="notice-marquee">
                  <span v-if="noticeText" class="notice-text">{{ noticeText }}</span>
                  <span v-else class="notice-text notice-placeholder">暂无公告，请在下方编辑框中输入公告内容</span>
                </div>
              </div>

              <!-- 搜索筛选预览 -->
              <div v-else-if="item.type === 'search'" class="preview-search" :class="{ 'preview-highlight': activeIndex === getOriginalIndex(item.type) }" :ref="el => setComponentRef(item.type, el)" @click="openSubPage('trip-search')">
                <div class="preview-station-filter">
                  <div class="preview-station-box">
                    <span class="preview-label">出发地</span>
                    <span class="preview-station-name">全部地区</span>
                  </div>
                  <div class="preview-swap" v-html="routeArrowSvg"></div>
                  <div class="preview-station-box">
                    <span class="preview-label">到达地</span>
                    <span class="preview-station-name">全部地区</span>
                  </div>
                </div>
                <div class="preview-date-bar">
                  <span class="preview-label">出行日期</span>
                  <span class="preview-date">{{ searchDateLabel }}</span>
                  <div class="preview-date-bar-right">
                    <div class="preview-glass-refresh">刷新</div>
                  </div>
                </div>
              </div>

              <!-- 车次列表预览（真实数据） -->
              <div v-else-if="item.type === 'trips'" class="preview-trips" :class="{ 'preview-highlight': activeIndex === getOriginalIndex(item.type) }" :ref="el => setComponentRef(item.type, el)">
                <!-- 有车次 -->
                <div class="preview-trips-list" v-if="tripList.length > 0">
                  <!-- 排序栏 -->
                  <div class="preview-sort-bar">
                    <span class="preview-sort-chip active">直达优先</span>
                    <span class="preview-sort-chip">发车时间</span>
                    <span class="preview-sort-chip">价格</span>
                    <span class="preview-sort-chip">耗时</span>
                  </div>
                  <div v-for="trip in tripList" :key="trip.id" class="preview-route-card">
                    <div class="preview-route-top">
                      <div class="preview-route-path">
                        <span class="preview-route-from">{{ trip.from }}</span>
                        <span class="preview-route-arrow" v-html="routeArrowSvg"></span>
                        <span class="preview-route-to">{{ trip.to }}</span>
                      </div>
                      <span class="preview-route-seats" :class="{ 'seats-low': trip.seatsLow, 'seats-out': trip.soldOut }">{{ trip.seats }}</span>
                    </div>
                    <!-- 途经走向 -->
                    <div class="preview-route-via">
                      <span class="preview-via-tag">直达</span>
                      <span class="preview-via-total">全程站</span>
                      <span class="preview-via-more">站点详情›</span>
                    </div>
                    <div class="preview-route-bottom">
                      <div class="preview-route-info">
                        <span class="preview-route-bus" v-html="busSvg"></span>
                        <span class="preview-route-duration">{{ trip.duration }}</span>
                        <span class="preview-route-arrival" v-if="trip.arrivalText">{{ trip.arrivalText }}到</span>
                      </div>
                      <div class="preview-route-action">
                        <span class="preview-route-price">¥{{ trip.price }}<span class="preview-route-price-unit" v-if="!trip.priceIsInterval">起</span></span>
                        <span class="preview-route-btn" :class="{ 'btn-disabled': trip.soldOut }" @click.stop="openSubPage('trip-detail', trip)">{{ trip.soldOut ? '售罄' : '订票' }}</span>
                      </div>
                    </div>
                  </div>
                </div>
                <!-- 无车次空状态 -->
                <div v-else class="preview-empty-state">
                  <span>今日暂无可购车次</span>
                </div>
              </div>
            </template>
            </template>

            <!-- 订单页预览 -->
            <template v-else-if="currentPage==='order'">
              <div class="preview-filter-tabs">
                <div
                  v-for="tab in visibleOrderTabs"
                  :key="tab.type"
                  class="preview-filter-tab"
                  :class="{ active: tab.type === orderActiveTab }"
                  @click="orderActiveTab = tab.type"
                >
                  <span>{{ tab.title }}</span>
                  <div class="preview-tab-underline" v-if="tab.type === orderActiveTab"></div>
                </div>
              </div>
              <!-- 订单列表（与详情页一致的订单卡片） -->
              <div class="detail-order-list">
                <div v-for="order in filteredMockOrders" :key="order.id" class="detail-order-card" @click="openSubPage('trip-detail', order)">
                  <div class="detail-order-header">
                    <span class="detail-order-no">
                      <span class="detail-type-tag" :class="order.isCargo ? 'cargo' : 'ticket'">{{ order.typeText }}</span>
                      {{ order.orderNo }}
                    </span>
                    <span class="detail-order-status" :class="'detail-status-' + order.status">{{ order.statusText }}</span>
                  </div>
                  <div class="detail-order-route">
                    <span class="detail-route-city">{{ order.from }}</span>
                    <div class="detail-route-arrow">
                      <div class="detail-arrow-line"></div>
                      <span class="detail-arrow-icon" v-html="routeArrowSvg"></span>
                      <div class="detail-arrow-line"></div>
                    </div>
                    <span class="detail-route-city">{{ order.to }}</span>
                  </div>
                  <div class="detail-order-info">
                    <span class="detail-info-time">{{ order.date }} {{ order.time }} 发车</span>
                    <div class="detail-info-right">
                      <span class="detail-info-seats">{{ order.infoText }}</span>
                      <span class="detail-info-price">¥{{ order.price }}</span>
                    </div>
                  </div>
                  <div class="detail-order-footer">
                    <span class="detail-order-action" :class="'detail-action-' + order.status">{{ order.actionText }}</span>
                  </div>
                </div>
              </div>
              <div v-if="filteredMockOrders.length === 0" class="preview-order-empty">
                <span>暂无订单</span>
              </div>
            </template>

            <!-- 我的页预览 -->
            <template v-else-if="currentPage==='mine'">
              <div class="preview-mine-user-card" @click="openSubPage('mine-profile')">
                <img class="preview-mine-avatar" :src="mineAvatar" />
                <div class="preview-mine-user-info">
                  <span class="preview-mine-name">狸猫日志</span>
                  <div class="preview-mine-phone-row" @click.stop="togglePreviewPhone">
                    <span class="preview-mine-phone">{{ previewPhoneMasked ? previewMaskedPhone : previewPhone }}</span>
                    <img class="preview-phone-eye-icon" :src="previewPhoneMasked ? eyeCloseIcon : eyeOpenIcon" />
                  </div>
                </div>
                <span class="preview-mine-user-arrow">›</span>
              </div>
              <div class="preview-mine-order-section">
                <div class="preview-mine-section-header">
                  <span class="preview-mine-section-title">我的订单</span>
                  <span class="preview-mine-more" @click="openSubPage('search')">全部订单 ›</span>
                </div>
                <div class="preview-mine-order-grid">
                  <div v-for="item in visibleMineOrderGrid" :key="item.type" class="preview-mine-order-item" @click="openSubPage('mine-order', item)">
                    <div class="preview-mine-order-icon-wrap">
                      <img class="preview-mine-order-icon" :src="mineIcons[item.type]" />
                    </div>
                    <span class="preview-mine-order-text">{{ item.title }}</span>
                  </div>
                </div>
              </div>
              <div class="preview-mine-menu-section" :class="'layout-' + mineMenuLayoutType">
                <div v-for="item in visibleMineMenu" :key="item.type" class="preview-mine-menu-item" @click="openSubPage('mine-' + item.type, item)">
                  <img class="preview-mine-menu-icon" :class="{ 'icon-wechat-service': item.type === 'wechat_service' }" :src="mineIcons[item.type]" />
                  <span class="preview-mine-menu-name">{{ item.title }}</span>
                  <span class="preview-mine-menu-arrow" v-if="mineMenuLayoutType==='list'">›</span>
                </div>
              </div>
              <!-- 退出登录（仅 mine_bottom 模式显示） -->
              <div v-if="logoutPosition==='mine_bottom'" class="preview-mine-logout-section">
                <div class="preview-mine-logout-btn" @click="handlePreviewLogout">
                  <span class="preview-mine-logout-icon" v-html="logoutIcon"></span>
                  <span>退出登录</span>
                </div>
                <div class="preview-mine-delete-account-link">注销账号</div>
              </div>
            </template>
            </template><!-- /主页面预览 -->

            <!-- 详情页预览 -->
            <template v-else>
              <!-- 行程搜索页（可输入关键词 + 切换日期 + 筛选车次） -->
              <template v-if="previewSubPage === 'trip-search'">
                <!-- 站点筛选（白底圆角卡片） -->
                <div class="search-station-filter">
                  <div class="search-station-box" @click="cycleFromStation">
                    <span class="search-station-label">出发地</span>
                    <span class="search-station-name">{{ searchFromStation || '全部地区' }}</span>
                  </div>
                  <div class="search-swap" v-html="routeArrowSvg" @click="swapSearchStations"></div>
                  <div class="search-station-box" @click="cycleToStation">
                    <span class="search-station-label">到达地</span>
                    <span class="search-station-name">{{ searchToStation || '全部地区' }}</span>
                  </div>
                </div>
                <!-- 日期栏（文字+日期值+清除筛选） -->
                <div class="search-date-bar">
                  <span class="search-date-label">出行日期</span>
                  <span class="search-date-value" @click="cycleSearchDate">{{ searchDateLabel }}</span>
                  <span v-if="searchFromStation || searchToStation" class="search-clear-btn" @click="searchFromStation = ''; searchToStation = ''">清除筛选</span>
                </div>
                <!-- 搜索框（圆角白底） -->
                <div class="search-box-round">
                  <span class="search-box-icon" v-html="searchIconSvg"></span>
                  <input class="search-box-input" v-model="searchKeyword" placeholder="输入目的地搜索车次" />
                  <span v-if="searchKeyword" class="search-box-clear" @click="searchKeyword = ''">✕</span>
                </div>
                <!-- 车次列表 -->
                <div class="search-result-list">
                  <div v-for="trip in filteredSearchTrips" :key="trip.id" class="search-result-card" @click="openSubPage('trip-detail', trip)">
                    <div class="search-result-top">
                      <div class="search-result-path">
                        <span class="search-result-from">{{ trip.from }}</span>
                        <span class="search-result-arrow" v-html="routeArrowSvg"></span>
                        <span class="search-result-to">{{ trip.to }}</span>
                      </div>
                      <span class="search-result-seats">{{ trip.seats }}</span>
                    </div>
                    <div class="search-result-bottom">
                      <div class="search-result-info">
                        <span class="search-result-bus" v-html="busSvg"></span>
                        <span class="search-result-duration">{{ trip.duration }}</span>
                        <span class="search-result-arrival" v-if="trip.arrivalText">{{ trip.arrivalText }}到</span>
                      </div>
                      <div class="search-result-action">
                        <span class="search-result-price">¥{{ trip.price }}</span>
                        <span class="search-result-btn">订票</span>
                      </div>
                    </div>
                  </div>
                  <div v-if="filteredSearchTrips.length === 0" class="search-empty">
                    <span>未找到相关车次</span>
                  </div>
                </div>
              </template>

              <!-- 个人中心详情页 -->
              <template v-if="previewSubPage === 'mine-profile'">
                <div class="profile-detail">
                  <div class="profile-user-card">
                    <img class="profile-avatar" :src="mineAvatar" />
                    <div class="profile-info">
                      <div class="profile-name">狸猫日志</div>
                      <div class="profile-phone">{{ previewPhoneMasked ? previewMaskedPhone : previewPhone }}</div>
                    </div>
                  </div>
                  <div v-if="logoutPosition==='profile_detail'" class="preview-mine-logout-section">
                    <div class="preview-mine-logout-btn" @click="handlePreviewLogout">
                      <span class="preview-mine-logout-icon" v-html="logoutIcon"></span>
                      <span>退出登录</span>
                    </div>
                  </div>
                </div>
              </template>

              <!-- 搜索/订单列表详情（匹配小程序 search.wxml 1:1） -->
              <template v-if="previewSubPage === 'search' || previewSubPage === 'mine-order'">
                <!-- 筛选标签 -->
                <div class="detail-filter-tabs">
                  <div v-for="tab in visibleOrderTabs" :key="tab.type"
                       class="detail-filter-tab"
                       :class="{ active: tab.type === orderActiveTab }"
                       @click="orderActiveTab = tab.type">
                    <span>{{ tab.title }}</span>
                    <div class="detail-tab-underline" v-if="tab.type === orderActiveTab"></div>
                  </div>
                </div>
                <!-- 订单列表 -->
                <div class="detail-order-list">
                  <div v-for="order in filteredMockOrders" :key="order.id" class="detail-order-card">
                    <div class="detail-order-header">
                      <span class="detail-order-no">
                        <span class="detail-type-tag" :class="order.isCargo ? 'cargo' : 'ticket'">{{ order.typeText }}</span>
                        {{ order.orderNo }}
                      </span>
                      <span class="detail-order-status" :class="'detail-status-' + order.status">{{ order.statusText }}</span>
                    </div>
                    <div class="detail-order-route">
                      <span class="detail-route-city">{{ order.from }}</span>
                      <div class="detail-route-arrow">
                        <div class="detail-arrow-line"></div>
                        <span class="detail-arrow-icon" v-html="routeArrowSvg"></span>
                        <div class="detail-arrow-line"></div>
                      </div>
                      <span class="detail-route-city">{{ order.to }}</span>
                    </div>
                    <div class="detail-order-info">
                      <span class="detail-info-time">{{ order.date }} {{ order.time }} 发车</span>
                      <div class="detail-info-right">
                        <span class="detail-info-seats">{{ order.infoText }}</span>
                        <span class="detail-info-price">¥{{ order.price }}</span>
                      </div>
                    </div>
                    <div class="detail-order-footer">
                      <span class="detail-order-action" :class="'detail-action-' + order.status">{{ order.actionText }}</span>
                    </div>
                  </div>
                </div>
                <div v-if="filteredMockOrders.length === 0" class="detail-empty">暂无订单</div>
              </template>

              <!-- 班次详情/订票详情（匹配小程序 trip-detail.wxml 1:1） -->
              <template v-else-if="previewSubPage === 'trip-detail'">
                <!-- 班次信息卡片 -->
                <div class="detail-trip-card">
                  <div class="detail-trip-route">
                    <span class="detail-trip-from">{{ previewSubPageData?.from || '商丘' }}</span>
                    <div class="detail-trip-arrow">
                      <div class="detail-arrow-line"></div>
                      <span class="detail-arrow-icon" v-html="routeArrowSvg"></span>
                      <div class="detail-arrow-line"></div>
                    </div>
                    <span class="detail-trip-to">{{ previewSubPageData?.to || '郑州' }}</span>
                  </div>
                  <div class="detail-trip-info">
                    <div class="detail-info-item">
                      <span class="detail-info-label">发车时间</span>
                      <span class="detail-info-value">{{ previewSubPageData?.time || '07:30' }} 发车</span>
                    </div>
                    <div class="detail-info-item">
                      <span class="detail-info-label">行程时长</span>
                      <span class="detail-info-value">{{ previewSubPageData?.duration || '约2小时' }}</span>
                    </div>
                    <div class="detail-info-item">
                      <span class="detail-info-label">票价</span>
                      <span class="detail-info-value detail-price">¥{{ previewSubPageData?.price || '45' }}</span>
                    </div>
                    <div class="detail-info-item">
                      <span class="detail-info-label">余座</span>
                      <span class="detail-info-value">{{ previewSubPageData?.seats || '有票' }}</span>
                    </div>
                  </div>
                  <div class="detail-track-btn">
                    <span class="detail-track-icon" v-html="busSvg"></span>
                    <span>查看车辆位置</span>
                  </div>
                </div>

                <!-- 乘客信息 -->
                <div class="detail-section">
                  <div class="detail-section-header">
                    <div class="detail-title-wrap">
                      <span class="detail-title-icon" v-html="passengerInfoSvg"></span>
                      <span class="detail-section-title">乘客信息</span>
                    </div>
                    <span class="detail-add-passenger">+ 手动添加</span>
                  </div>
                  <div class="detail-passenger-item">
                    <div class="detail-passenger-form">
                      <div class="detail-p-input">乘客姓名</div>
                      <div class="detail-p-input">身份证号</div>
                      <div class="detail-p-input">手机号（选填）</div>
                    </div>
                    <div class="detail-remove-passenger"><span class="detail-remove-text">删除</span></div>
                  </div>
                  <div class="detail-saved-passenger-btn">从常用乘客选择</div>
                </div>

                <!-- 联系人信息 -->
                <div class="detail-section">
                  <div class="detail-section-header">
                    <div class="detail-title-wrap">
                      <span class="detail-title-icon" v-html="contactInfoSvg"></span>
                      <span class="detail-section-title">联系人信息</span>
                    </div>
                  </div>
                  <div class="detail-contact-form">
                    <div class="detail-c-input">联系人姓名</div>
                    <div class="detail-c-input">联系人手机号</div>
                  </div>
                </div>

                <!-- 优惠券 -->
                <div class="detail-section">
                  <div class="detail-section-header">
                    <div class="detail-title-wrap">
                      <span class="detail-title-icon" v-html="couponSelectSvg"></span>
                      <span class="detail-section-title">优惠券</span>
                    </div>
                  </div>
                  <div class="detail-coupon-select">
                    <span class="detail-coupon-text">选择优惠券</span>
                    <span class="detail-coupon-arrow">›</span>
                  </div>
                </div>

                <!-- 底部下单栏 -->
                <div class="detail-bottom-bar">
                  <div class="detail-total-price">
                    <span class="detail-price-label">合计</span>
                    <span class="detail-price-num">¥{{ previewSubPageData?.price || '80' }}</span>
                    <span class="detail-price-count">×1人</span>
                  </div>
                  <span class="detail-order-btn">立即下单</span>
                </div>
              </template>

              <!-- 我的页菜单项详情 -->
              <template v-else-if="previewSubPage && previewSubPage.startsWith('mine-')">
                <!-- 常用乘客 -->
                <template v-if="previewSubPage === 'mine-passenger'">
                  <div class="detail-section" style="margin-top: 16px;">
                    <div class="detail-section-header">
                      <span class="detail-section-title">常用乘客</span>
                      <span class="detail-add-passenger">+ 添加乘客</span>
                    </div>
                    <div v-for="p in mockPassengers" :key="p.id" class="detail-passenger-card">
                      <div class="detail-passenger-info">
                        <span class="detail-passenger-name">{{ p.name }}</span>
                        <span class="detail-passenger-card-no">{{ p.id_card_no }}</span>
                      </div>
                      <span class="detail-passenger-arrow">›</span>
                    </div>
                    <div v-if="mockPassengers.length === 0" class="detail-empty">暂无常用乘客</div>
                  </div>
                </template>

                <!-- 行程管理 -->
                <template v-else-if="previewSubPage === 'mine-orders'">
                  <div class="detail-section" style="margin-top: 16px;">
                    <div class="detail-section-header">
                      <span class="detail-section-title">行程管理</span>
                    </div>
                    <div v-for="trip in tripList" :key="trip.id" class="detail-trip-manage-card">
                      <div class="detail-trip-manage-route">
                        <span class="detail-trip-manage-city">{{ trip.from }}</span>
                        <span class="detail-trip-manage-arrow">→</span>
                        <span class="detail-trip-manage-city">{{ trip.to }}</span>
                      </div>
                      <div class="detail-trip-manage-info">
                        <span>{{ trip.time }} 发车</span>
                        <span>¥{{ trip.price }}</span>
                      </div>
                    </div>
                    <div v-if="tripList.length === 0" class="detail-empty">暂无行程</div>
                  </div>
                </template>

                <!-- 货物托运 -->
                <template v-else-if="previewSubPage === 'mine-cargo'">
                  <div class="detail-section" style="margin-top: 16px;">
                    <div class="detail-section-header">
                      <span class="detail-section-title">货物托运</span>
                      <span class="detail-add-passenger">+ 新建托运</span>
                    </div>
                    <div class="detail-empty">暂无托运记录</div>
                  </div>
                </template>

                <!-- 微信客服（模拟微信开发者工具客服会话页面） -->
                <template v-else-if="previewSubPage === 'mine-wechat_service'">
                  <div class="wx-chat-page">
                    <div class="wx-chat-messages">
                      <div class="wx-chat-system-msg">
                        <span>您已接入客服会话</span>
                      </div>
                      <div class="wx-chat-row wx-chat-incoming">
                        <div class="wx-chat-avatar wx-chat-avatar-cs"></div>
                        <div class="wx-chat-bubble wx-chat-bubble-in">您好，我是狸猫日志客服，有什么可以帮您？</div>
                      </div>
                      <div class="wx-chat-row wx-chat-outgoing">
                        <div class="wx-chat-bubble wx-chat-bubble-out">我想咨询购票问题</div>
                        <div class="wx-chat-avatar wx-chat-avatar-user"></div>
                      </div>
                      <div class="wx-chat-row wx-chat-incoming">
                        <div class="wx-chat-avatar wx-chat-avatar-cs"></div>
                        <div class="wx-chat-bubble wx-chat-bubble-in">好的，请问您需要咨询班次查询、退改签还是其他问题？</div>
                      </div>
                    </div>
                    <div class="wx-chat-input-bar">
                      <div class="wx-chat-input-placeholder">请输入消息</div>
                      <div class="wx-chat-send-btn">发送</div>
                    </div>
                  </div>
                </template>

                <!-- 电话热线（背景页内容，拨打提示弹窗在 phone-frame 层） -->
                <template v-else-if="previewSubPage === 'mine-phone'">
                  <div class="detail-section" style="margin-top: 16px;">
                    <div class="detail-contact-card">
                      <img class="detail-contact-icon" :src="mineIcons.phone" />
                      <div class="detail-contact-info">
                        <span class="detail-contact-title">电话热线</span>
                        <span class="detail-contact-desc">点击拨打客服电话</span>
                      </div>
                    </div>
                  </div>
                </template>

                <!-- 司机核销 -->
                <template v-else-if="previewSubPage === 'mine-verify'">
                  <div class="detail-section" style="margin-top: 16px;">
                    <div class="detail-section-header">
                      <span class="detail-section-title">司机核销</span>
                    </div>
                    <div class="detail-verify-input">
                      <div class="detail-c-input">请输入核销码</div>
                      <div class="detail-verify-btn">核销</div>
                    </div>
                  </div>
                </template>

                <!-- 惊喜礼包 -->
                <template v-else-if="previewSubPage === 'mine-gift'">
                  <!-- 主标签栏 -->
                  <div class="preview-gift-tabs">
                    <div class="preview-gift-tab" :class="{ active: previewGiftTab === 0 }" @click="switchGiftTab(0)">优惠券</div>
                    <div class="preview-gift-tab" :class="{ active: previewGiftTab === 1 }" @click="switchGiftTab(1)">积分</div>
                  </div>

                  <!-- 优惠券 -->
                  <template v-if="previewGiftTab === 0">
                    <!-- 优惠券子标签 -->
                    <div class="preview-gift-subtabs">
                      <div class="preview-gift-subtab" :class="{ active: previewCouponTab === 0 }" @click="switchCouponTab(0)">未使用</div>
                      <div class="preview-gift-subtab" :class="{ active: previewCouponTab === 1 }" @click="switchCouponTab(1)">已使用</div>
                      <div class="preview-gift-subtab" :class="{ active: previewCouponTab === 2 }" @click="switchCouponTab(2)">已过期</div>
                    </div>

                    <!-- 未使用优惠券 -->
                    <template v-if="previewCouponTab === 0">
                      <div class="preview-coupon-card">
                        <div class="preview-coupon-left">
                          <span class="preview-coupon-value">¥10</span>
                          <span class="preview-coupon-desc">满50可用</span>
                        </div>
                        <div class="preview-coupon-right">
                          <span class="preview-coupon-label">满减券</span>
                          <span class="preview-coupon-name">新人立减券</span>
                          <span class="preview-coupon-expire">有效期至 2025-08-15</span>
                        </div>
                        <div class="preview-coupon-status usable"><span>去使用</span></div>
                      </div>
                      <div class="preview-coupon-card">
                        <div class="preview-coupon-left">
                          <span class="preview-coupon-value">8.5折</span>
                          <span class="preview-coupon-desc">无门槛</span>
                        </div>
                        <div class="preview-coupon-right">
                          <span class="preview-coupon-label">折扣券</span>
                          <span class="preview-coupon-name">出行特惠折扣</span>
                          <span class="preview-coupon-expire">有效期至 2025-09-01</span>
                        </div>
                        <div class="preview-coupon-status usable"><span>去使用</span></div>
                      </div>
                    </template>

                    <!-- 已使用优惠券 -->
                    <template v-else-if="previewCouponTab === 1">
                      <div class="preview-coupon-card">
                        <div class="preview-coupon-left gray">
                          <span class="preview-coupon-value">¥10</span>
                          <span class="preview-coupon-desc">满50可用</span>
                        </div>
                        <div class="preview-coupon-right">
                          <span class="preview-coupon-label">满减券</span>
                          <span class="preview-coupon-name">新人立减券</span>
                          <span class="preview-coupon-expire">有效期至 2025-06-15</span>
                        </div>
                        <div class="preview-coupon-status used"><span>已使用</span></div>
                      </div>
                    </template>

                    <!-- 已过期优惠券 -->
                    <template v-else-if="previewCouponTab === 2">
                      <div class="preview-coupon-card">
                        <div class="preview-coupon-left gray">
                          <span class="preview-coupon-value">¥5</span>
                          <span class="preview-coupon-desc">无门槛</span>
                        </div>
                        <div class="preview-coupon-right">
                          <span class="preview-coupon-label">抵用券</span>
                          <span class="preview-coupon-name">暑期出行券</span>
                          <span class="preview-coupon-expire">有效期至 2025-05-01</span>
                        </div>
                        <div class="preview-coupon-status expired"><span>已过期</span></div>
                      </div>
                    </template>
                  </template>

                  <!-- 积分 -->
                  <template v-if="previewGiftTab === 1">
                    <!-- 积分卡片 -->
                    <div class="preview-gift-points-card">
                      <img class="preview-gift-icon" :src="minePoints" />
                      <div class="preview-points-info">
                        <span class="preview-points-label">我的积分</span>
                        <span class="preview-points-value">1,280</span>
                        <span class="preview-points-desc">消费产生积分，积分可兑换权益</span>
                      </div>
                    </div>

                    <!-- 积分明细 -->
                    <div class="detail-section" style="margin-top: 12px;">
                      <div class="detail-section-header">
                        <span class="detail-section-title">积分明细</span>
                      </div>
                      <div class="preview-record-item">
                        <div class="preview-record-left">
                          <span class="preview-record-remark">购票消费</span>
                          <span class="preview-record-date">7月15日 14:30</span>
                        </div>
                        <span class="preview-record-points plus">+80</span>
                      </div>
                      <div class="preview-record-item">
                        <div class="preview-record-left">
                          <span class="preview-record-remark">购票消费</span>
                          <span class="preview-record-date">7月10日 09:15</span>
                        </div>
                        <span class="preview-record-points plus">+160</span>
                      </div>
                      <div class="preview-record-item">
                        <div class="preview-record-left">
                          <span class="preview-record-remark">退款扣除</span>
                          <span class="preview-record-date">7月8日 16:00</span>
                        </div>
                        <span class="preview-record-points minus">-80</span>
                      </div>
                    </div>

                    <!-- 积分说明 -->
                    <div class="preview-gift-tips">
                      <span class="preview-tips-title">积分说明</span>
                      <span class="preview-tips-text">1. 购票消费后自动产生积分，每消费1元获得相应积分</span>
                      <span class="preview-tips-text">2. 积分可用于兑换优惠券等权益</span>
                      <span class="preview-tips-text">3. 退款订单将扣除对应积分</span>
                    </div>
                  </template>
                </template>
              </template>
            </template>
          </div>

          <!-- 模拟微信开发者工具弹窗（电话拨打提示） -->
          <div v-if="previewSubPage === 'mine-phone'" class="wx-modal-mask">
            <div class="wx-modal-dialog">
              <div class="wx-modal-title">提示</div>
              <div class="wx-modal-content" v-if="servicePhone">是否拨打电话？<br/>{{ servicePhone }}</div>
              <div class="wx-modal-content" v-else>未设置客服电话<br/>请在「设计 - 电话设置」中配置</div>
              <div class="wx-modal-footer">
                <div class="wx-modal-btn wx-modal-btn-cancel" @click="closeSubPage">取消</div>
                <div class="wx-modal-btn wx-modal-btn-confirm" @click="closeSubPage">拨打</div>
              </div>
            </div>
          </div>

          <!-- 优惠券领取提示 toast -->
          <div v-if="couponToast" class="coupon-toast">{{ couponToast }}</div>

          <!-- 底部 TabBar（毛玻璃 + 真实SVG图标，与小程序一致：首页/订单/我的） -->
          <div class="phone-tabbar" v-if="!previewSubPage">
            <div class="tabbar-item" :class="{ active: currentPage==='home' }" @click="switchPage('home')">
              <img :src="currentPage==='home' ? tabHomeActive : tabHome" class="tabbar-icon" />
              <span class="tabbar-text">首页</span>
            </div>
            <div class="tabbar-item" :class="{ active: currentPage==='order' }" @click="switchPage('order')">
              <img :src="currentPage==='order' ? tabOrderActive : tabOrder" class="tabbar-icon" />
              <span class="tabbar-text">订单</span>
            </div>
            <div class="tabbar-item" :class="{ active: currentPage==='mine' }" @click="switchPage('mine')">
              <img :src="currentPage==='mine' ? tabMineActive : tabMine" class="tabbar-icon" />
              <span class="tabbar-text">我的</span>
            </div>
          </div>
        </div>
        </div>
          </div><!-- phone-front-face -->
        </div><!-- phone-3d-flipper -->
        </div><!-- phone-3d-container -->

        <!-- 悬浮保存按钮 -->
        <transition name="fade-slide">
          <div v-if="showFloatBtn" class="float-save-btn" @click="handleSave">
            <span class="float-save-icon" v-html="saveConfigIcon"></span>
            <span>保存</span>
          </div>
        </transition>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Rank, InfoFilled, Check } from '@element-plus/icons-vue'
import { designApi, bannerApi, configApi, couponApi, tripApi } from '@/api'
import routeArrowSvg from '@/assets/icons/decorate/route-arrow.svg?raw'
import searchIconSvg from '@/assets/icons/decorate/search-icon.svg?raw'
import busSvg from '@/assets/icons/decorate/bus.svg?raw'
import megaphoneSvg from '@/assets/icons/decorate/notice-megaphone.svg?raw'
// 购票详情页标题图标（与小程序一致：黑色）
import passengerInfoSvg from '@/assets/icons/decorate/passenger-info.svg?raw'
import contactInfoSvg from '@/assets/icons/decorate/contact-info.svg?raw'
import couponSelectSvg from '@/assets/icons/decorate/coupon-select.svg?raw'
import couponTicketSvg from '@/assets/icons/decorate/coupon-ticket.svg?raw'
import bannerCarouselSvg from '@/assets/icons/decorate/banner-carousel.svg?raw'
import couponDisplaySvg from '@/assets/icons/decorate/coupon-display.svg?raw'
import searchFilterSvg from '@/assets/icons/decorate/search-filter.svg?raw'
import tripListSvg from '@/assets/icons/decorate/trip-list.svg?raw'
import tabHomeActive from '@/assets/icons/decorate/home-active.svg'
import tabOrder from '@/assets/icons/decorate/order.svg'
import tabMine from '@/assets/icons/decorate/mine.svg'
import tabHome from '@/assets/icons/decorate/home.svg'
import tabOrderActive from '@/assets/icons/decorate/order-active.svg'
import tabMineActive from '@/assets/icons/decorate/mine-active.svg'
// 我的页真实图标（与小程序一致）
import minePendingPay from '@/assets/icons/decorate/mine/pending-pay.svg'
import minePendingTravel from '@/assets/icons/decorate/mine/pending-travel.svg'
import mineCompleted from '@/assets/icons/decorate/mine/completed.svg'
import mineRefund from '@/assets/icons/decorate/mine/refund.svg'
import minePassenger from '@/assets/icons/decorate/mine/passenger-info.svg'
import mineOrders from '@/assets/icons/decorate/mine/trip-track.svg'
import mineCargo from '@/assets/icons/decorate/mine/cargo.svg'
import mineWechat from '@/assets/icons/decorate/mine/wechat-service.svg'
import minePhone from '@/assets/icons/decorate/mine/phone-hotline.svg'
import mineVerify from '@/assets/icons/decorate/mine/verify.svg'
import mineGift from '@/assets/icons/decorate/mine/surprise-gift.svg'
import minePoints from '@/assets/icons/decorate/mine/points.svg'
import logoutIcon from '@/assets/icons/decorate/mine/logout.svg?raw'
import mineAvatar from '@/assets/icons/decorate/mine/avatar.png'
import eyeCloseIcon from '@/assets/icons/decorate/eye-close.svg'
import eyeOpenIcon from '@/assets/icons/decorate/eye-open.svg'
// 装修页保存/布局按钮图标（白色）
import saveConfigIcon from '@/assets/icons/save-config.svg?raw'
import saveNoticeIcon from '@/assets/icons/save-notice.svg?raw'
import layoutListIcon from '@/assets/icons/layout-list.svg?raw'
import layoutGridIcon from '@/assets/icons/layout-grid.svg?raw'
// 上下移动按钮图标（装修页排序）
import moveUpIcon from '@/assets/icons/decorate/move-up.svg?raw'
import moveDownIcon from '@/assets/icons/decorate/move-down.svg?raw'
// 模拟器导航键图标
import navUpIcon from '@/assets/icons/nav-up.svg?raw'
import navDownIcon from '@/assets/icons/nav-down.svg?raw'
import navLeftIcon from '@/assets/icons/nav-left.svg?raw'
import navRightIcon from '@/assets/icons/nav-right.svg?raw'
import navCenterIcon from '@/assets/icons/nav-center.svg?raw'
// 机型切换图标
import phoneSwitchIcon from '@/assets/icons/phone-switch.svg?raw'
// 手机背面图片
import appleBackImg from '@/assets/icons/apple.png'
import pixelBackImg from '@/assets/icons/android.png'
import huaweiBackImg from '@/assets/icons/huawei.jpg'
import samsungBackImg from '@/assets/icons/samsung.png'

// TODO: 这个文件5000多行了，后面装修面板拆分时得拆成独立组件，不然改一个地方要滚半天
// 机型品牌Logo（弹窗缩略图叠加用，不影响3D翻转背面）
const brandLogos: Record<PhoneModel, string> = {
  apple: `<svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path d="M791.488 544.095c-1.28-129.695 105.76-191.871 110.528-194.975-60.16-88.032-153.856-100.064-187.232-101.472-79.744-8.064-155.584 46.944-196.064 46.944-40.352 0-102.816-45.76-168.96-44.544-86.912 1.28-167.072 50.528-211.808 128.384-90.304 156.703-23.136 388.831 64.896 515.935 43.008 62.208 94.304 132.064 161.632 129.568 64.832-2.592 89.376-41.952 167.744-41.952s100.416 41.952 169.056 40.672c69.76-1.312 113.984-63.392 156.704-125.792 49.376-72.16 69.728-142.048 70.912-145.632-1.536-0.704-136.064-52.224-137.408-207.136zM662.56 163.52C698.304 120.16 722.432 60 715.84 0c-51.488 2.112-113.888 34.304-150.816 77.536-33.152 38.368-62.144 99.616-54.368 158.432 57.472 4.48 116.128-29.216 151.904-72.448z" fill="#2c2c2c"/></svg>`,
  pixel: `<svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path d="M210.04288 260.05504a30.72 30.72 0 0 1 41.90208 11.42784l73.3184 128.32768A438.64064 438.64064 0 0 1 512 358.4c60.416 0 118.00576 12.16512 170.3936 34.16064l69.2224-121.07776a30.72 30.72 0 0 1 53.32992 30.47424l-67.584 118.3744a440.36096 440.36096 0 0 1 212.7872 334.39744 30.72 30.72 0 0 1-30.55616 33.75104H104.448a30.72 30.72 0 0 1-30.55616-33.75104c13.5168-136.11008 88.96512-253.952 197.75488-325.0176L198.656 301.95712a30.72 30.72 0 0 1 11.42784-41.90208zM139.8784 727.04h744.2432c-33.50528-174.98112-187.392-307.2-372.16256-307.2-184.7296 0-338.61632 132.21888-372.08064 307.2zM327.68 655.36a40.96 40.96 0 1 0 0-81.92 40.96 40.96 0 0 0 0 81.92z m368.64 0a40.96 40.96 0 1 0 0-81.92 40.96 40.96 0 0 0 0 81.92z" fill="#2c2c2c"/></svg>`,
  huawei: `<svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path d="M446.50124 183.429683s-71.363328-3.247974-116.870777 55.568596c-45.528939 58.814524-11.337209 148.651598 10.248411 194.994066 21.584598 46.341445 135.480624 228.190119 139.664923 232.190223 4.151553 4.010337 9.154497 2.322905 9.42772-1.276063 0.275269-3.569292 13.010315-264.227893 2.76088-328.498686-10.248412-64.272839-40.591487-145.927557-45.231157-152.978136zM199.86091 304.604486c-8.156774 0.87288-66.697051 59.253522-71.973218 116.331495-5.301749 57.07388 19.128663 95.057983 87.341231 140.499941 68.664869 48.689932 231.852532 138.264017 234.849796 130.547265 3.001357-7.696286-63.235207-125.387755-117.44076-210.986295-54.201459-85.596494-124.656091-177.262216-132.777049-176.392406z m21.946848 526.487969c49.38578 22.079878 126.322034-27.508517 147.604756-41.930994 19.792788-15.121396 56.353472-42.919509 56.353473-42.919509l-279.836383 7.489578s26.491351 55.276954 75.878154 77.360925z m7.030114-226.13532c-50.083675-25.069979-153.122423-82.662675-157.154249-81.560574-4.030803 1.100054-19.553335 72.743768 12.670577 126.742613 32.223912 53.998845 94.978165 70.511937 123.777071 74.924433 32.379455 4.613064 222.75841 2.551102 221.482347-1.277086-1.122567-3.341095-150.662395-93.782943-200.775746-118.829386z m464.281185-365.957833c-45.503357-58.81657-116.869754-55.568596-116.869754-55.568596-4.63353 7.048533-34.983769 88.703251-45.231157 152.978137-10.249435 64.271816 2.517333 324.928371 2.789532 328.500732 0.275269 3.595898 5.244444 5.281283 9.423627 1.27504 4.1536-4.002151 118.081349-185.850825 139.667993-232.19227 21.553898-46.342468 55.748698-136.177496 10.219759-194.993043zM951.064874 523.395538c-4.02978-1.103124-107.068528 56.492642-157.151179 81.560574-50.083675 25.045419-199.652156 115.488291-200.748117 118.829385-1.302669 3.827165 189.076286 5.891174 221.455741 1.277087 28.79686-4.412496 91.551112-20.925588 123.777071-74.924434 32.253588-53.997821 16.697287-125.643582 12.666484-126.742612zM653.344169 789.161461c21.278629 14.421454 98.245581 64.010873 147.634432 41.930994 49.384757-22.086018 75.881225-77.360925 75.881224-77.360925l-279.837406-7.489578c-0.002047 0 36.556591 27.802206 56.32175 42.919509z m241.551428-368.226503c-5.304819-57.07695-63.84305-115.457592-71.970148-116.331495-8.151657-0.86981-78.608336 90.795912-132.809795 176.393429-54.207599 85.59854-120.439046 203.293079-117.438713 210.986295 3.000334 7.715729 166.212556-81.857333 234.851842-130.547265 68.208475-45.442982 92.672656-83.426061 87.366814-140.500964z" fill="#2c2c2c"/></svg>`,
  samsung: `<svg viewBox="0 0 240 70" xmlns="http://www.w3.org/2000/svg"><text x="120" y="50" text-anchor="middle" font-family="Helvetica, Arial, sans-serif" font-size="42" font-weight="700" fill="#1428a0" letter-spacing="4">SAMSUNG</text></svg>`,
}

const loading = ref(false)
const saving = ref(false)
const layout = ref<any[]>([])

const dragIndex = ref(-1)
const dragOverIndex = ref(-1)
const activeIndex = ref(-1)
const showFloatBtn = ref(false)

// 页面切换
const currentPage = ref<'home' | 'order' | 'mine'>('home')
const previewSubPage = ref<string | null>(null)
const previewSubPageData = ref<any>(null)
// 惊喜礼包预览标签切换
const previewGiftTab = ref(0) // 0=优惠券 1=积分
const previewCouponTab = ref(0) // 0=未使用 1=已使用 2=已过期
const switchGiftTab = (tab: number) => { previewGiftTab.value = tab }
const switchCouponTab = (tab: number) => { previewCouponTab.value = tab }

// 手机机型切换
type PhoneModel = 'apple' | 'pixel' | 'huawei' | 'samsung'
const currentModel = ref<PhoneModel>('apple')
const pendingModel = ref<PhoneModel>('apple')
const isFlipping = ref(false)
// isLocked 在整个翻转+回转周期(1.6s)内保持true，防止回转动画进行中再次触发切换
const isLocked = ref(false)
const showModelPicker = ref(false)
// 翻转定时器引用，组件卸载时清理
let flipTimer1: ReturnType<typeof setTimeout> | null = null
let flipTimer2: ReturnType<typeof setTimeout> | null = null
const modelInfo: Record<PhoneModel, { name: string; sub: string; back: string }> = {
  apple: { name: 'iPhone 17 Pro Max', sub: 'iOS', back: appleBackImg },
  pixel: { name: 'Pixel 10', sub: 'Android', back: pixelBackImg },
  huawei: { name: 'Pura 80 Pro', sub: 'HarmonyOS', back: huaweiBackImg },
  samsung: { name: 'Galaxy S26 Ultra', sub: 'One UI', back: samsungBackImg },
}
const switchPhoneModel = (model: PhoneModel) => {
  // 无论后续是否实际切换，点击选项都先关闭弹窗
  showModelPicker.value = false
  // 使用 isLocked 锁定整个动画周期（翻转800ms + 回转800ms = 1600ms）
  if (model === currentModel.value || isLocked.value) return
  isLocked.value = true
  pendingModel.value = model
  isFlipping.value = true
  // 800ms：翻转到背面时切换正面内容并触发回转
  flipTimer1 = setTimeout(() => {
    currentModel.value = model
    isFlipping.value = false
  }, 800)
  // 1600ms：回转动画完成后解锁
  flipTimer2 = setTimeout(() => {
    isLocked.value = false
  }, 1600)
}
// 当前显示的机型（翻转中用pendingModel，否则用currentModel）
const displayModel = computed(() => isFlipping.value ? pendingModel.value : currentModel.value)
const navbarTitle = computed(() => {
  if (previewSubPage.value === 'trip-search') return '搜索车次'
  if (previewSubPage.value === 'search') return '我的订单'
  if (previewSubPage.value === 'mine-order') return '我的订单'
  if (previewSubPage.value === 'trip-detail') return '班次详情'
  if (previewSubPage.value === 'mine-passenger') return '常用乘客'
  if (previewSubPage.value === 'mine-orders') return '行程管理'
  if (previewSubPage.value === 'mine-cargo') return '货物托运'
  if (previewSubPage.value === 'mine-wechat_service') return '微信客服'
  if (previewSubPage.value === 'mine-phone') return '电话热线'
  if (previewSubPage.value === 'mine-verify') return '司机核销'
  if (previewSubPage.value === 'mine-gift') return '惊喜礼包'
  if (previewSubPage.value === 'mine-profile') return '个人中心'
  if (currentPage.value === 'order') return '我的订单'
  if (currentPage.value === 'mine') return ''
  return ''
})

// 订单页/我的页布局数据
const orderTabsLayout = ref<any[]>([])
const mineOrderGridLayout = ref<any[]>([])
const mineMenuLayout = ref<any[]>([])
const mineMenuLayoutType = ref<'list' | 'grid'>('list')
const logoutPosition = ref<'mine_bottom' | 'profile_detail'>('mine_bottom')
const previewPhoneMasked = ref(true)
const previewPhone = '188****8888'
const previewMaskedPhone = computed(() => {
  return previewPhone.substring(0, 3) + '****' + previewPhone.substring(previewPhone.length - 4)
})
const togglePreviewPhone = () => {
  previewPhoneMasked.value = !previewPhoneMasked.value
}
// 退出登录（装修预览模式提示）
const handlePreviewLogout = () => {
  ElMessage.info('装修预览模式，无法退出登录')
}
const loadingOrder = ref(false)
const loadingMine = ref(false)
const orderActiveTab = ref('all')

// 真实电话号码（从电话设置加载）
const servicePhone = ref('')

// 搜索功能
const searchKeyword = ref('')
const searchDateIndex = ref(0)
const searchFromStation = ref('')
const searchToStation = ref('')
// 从车次列表提取站点选项
const stationOptions = computed(() => {
  const set = new Set<string>()
  tripList.value.forEach((t: any) => {
    set.add(t.from)
    set.add(t.to)
  })
  return Array.from(set)
})
const searchDateOptions = computed(() => {
  const opts: { label: string; value: string; short: string }[] = []
  const now = new Date()
  const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  for (let i = 0; i < 3; i++) {
    const d = new Date(now)
    d.setDate(d.getDate() + i)
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    opts.push({
      label: i === 0 ? `今天 ${weekdays[d.getDay()]}` : i === 1 ? `明天 ${weekdays[d.getDay()]}` : `后天 ${weekdays[d.getDay()]}`,
      short: `${mm}-${dd}`,
      value: `${d.getFullYear()}-${mm}-${dd}`,
    })
  }
  return opts
})
const searchDateLabel = computed(() => {
  const opt = searchDateOptions.value[searchDateIndex.value]
  return opt ? opt.label : todayDate.value
})
const filteredSearchTrips = computed(() => {
  let result = tripList.value
  if (searchFromStation.value) {
    result = result.filter((t: any) => t.from === searchFromStation.value)
  }
  if (searchToStation.value) {
    result = result.filter((t: any) => t.to === searchToStation.value)
  }
  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.trim()
    result = result.filter((t: any) => t.from.includes(kw) || t.to.includes(kw))
  }
  return result
})
const cycleSearchDate = () => {
  searchDateIndex.value = (searchDateIndex.value + 1) % 3
}
const cycleFromStation = () => {
  const opts = stationOptions.value
  if (opts.length === 0) return
  const idx = opts.indexOf(searchFromStation.value)
  if (idx === -1) {
    // 当前是"全部"，切换到第一个站点
    searchFromStation.value = opts[0]
  } else if (idx === opts.length - 1) {
    // 到达最后一个，回到"全部"
    searchFromStation.value = ''
  } else {
    searchFromStation.value = opts[idx + 1]
  }
}
const cycleToStation = () => {
  const opts = stationOptions.value
  if (opts.length === 0) return
  const idx = opts.indexOf(searchToStation.value)
  if (idx === -1) {
    searchToStation.value = opts[0]
  } else if (idx === opts.length - 1) {
    searchToStation.value = ''
  } else {
    searchToStation.value = opts[idx + 1]
  }
}
// 交换出发/到达站
const swapSearchStations = () => {
  const tmp = searchFromStation.value
  searchFromStation.value = searchToStation.value
  searchToStation.value = tmp
}

// 优惠券领取模拟
const couponClaimedIds = ref<Set<number>>(new Set())
const couponToast = ref<string>('')
let couponToastTimer: ReturnType<typeof setTimeout> | null = null
const claimCoupon = (couponId: number) => {
  if (couponClaimedIds.value.has(couponId)) {
    couponToast.value = '该优惠券已领取，请勿重复领取'
  } else {
    couponClaimedIds.value.add(couponId)
    couponToast.value = '领取成功！'
  }
  if (couponToastTimer) clearTimeout(couponToastTimer)
  couponToastTimer = setTimeout(() => { couponToast.value = '' }, 2000)
}

// 子页面（详情页）导航
const openSubPage = async (page: string, data?: any) => {
  previewSubPage.value = page
  previewSubPageData.value = data || null
  if (page === 'mine-order' && data) {
    orderActiveTab.value = data.type === 'pending_pay' ? '0' : data.type === 'pending_travel' ? '1' : data.type === 'completed' ? '2' : data.type === 'refund' ? '3' : 'all'
  }
  // 搜索/订单详情需要筛选标签数据
  if ((page === 'search' || page === 'mine-order') && !pageLoaded.value['order']) {
    pageLoaded.value['order'] = true
    await loadOrderLayout()
  }
  nextTick(() => {
    if (phoneScrollRef.value) phoneScrollRef.value.scrollTop = 0
  })
}
const closeSubPage = () => {
  previewSubPage.value = null
  previewSubPageData.value = null
}

// 模拟订单数据（与小程序真实样式一致）
const mockOrders = [
  { id: 1, typeText: '车票', isCargo: false, orderNo: 'LM20250117001', status: 1, statusText: '待出行', from: '商丘', to: '郑州', date: '2025-01-18', time: '07:30', infoText: '2座', price: '45', actionText: '查看详情' },
  { id: 2, typeText: '车票', isCargo: false, orderNo: 'LM20250116002', status: 2, statusText: '已完成', from: '郑州', to: '商丘', date: '2025-01-16', time: '14:00', infoText: '1座', price: '45', actionText: '查看详情' },
  { id: 3, typeText: '货物', isCargo: true, orderNo: 'LM20250115003', status: 0, statusText: '待支付', from: '商丘', to: '洛阳', date: '2025-01-17', time: '09:00', infoText: '50kg', price: '60', actionText: '去支付' },
]

// 模拟乘客数据
const mockPassengers = [
  { id: 1, name: '张三', id_card_no: '411402199001011234' },
  { id: 2, name: '李四', id_card_no: '411402199203045678' },
]

// 按筛选标签过滤订单
const filteredMockOrders = computed(() => {
  if (orderActiveTab.value === 'all') return mockOrders
  return mockOrders.filter(o => o.status === parseInt(orderActiveTab.value))
})

// 我的页真实图标映射
const mineIcons: Record<string, string> = {
  pending_pay: minePendingPay,
  pending_travel: minePendingTravel,
  completed: mineCompleted,
  refund: mineRefund,
  passenger: minePassenger,
  orders: mineOrders,
  cargo: mineCargo,
  wechat_service: mineWechat,
  phone: minePhone,
  verify: mineVerify,
  gift: mineGift,
  points: minePoints,
}

// 默认布局（与后端一致，接口为空时兜底）
const defaultOrderTabs = () => [
  { type: 'all', title: '全部', visible: true },
  { type: '0', title: '待支付', visible: true },
  { type: '1', title: '待出行', visible: true },
  { type: '2', title: '已完成', visible: true },
  { type: '3', title: '已退款', visible: true },
  { type: '4', title: '已取消', visible: true },
]
const defaultMineOrderGrid = () => [
  { type: 'pending_pay', title: '待付款', visible: true },
  { type: 'pending_travel', title: '待出行', visible: true },
  { type: 'completed', title: '已完成', visible: true },
  { type: 'refund', title: '退款', visible: true },
]
const defaultMineMenu = () => [
  { type: 'passenger', title: '常用乘客', visible: true },
  { type: 'gift', title: '惊喜礼包', visible: true },
  { type: 'orders', title: '行程管理', visible: true },
  { type: 'cargo', title: '货物托运', visible: true },
  { type: 'wechat_service', title: '微信客服', visible: true },
  { type: 'phone', title: '电话热线', visible: true },
  { type: 'verify', title: '司机核销', visible: true },
]

// 真实预览数据
const bannerList = ref<any[]>([])
const couponList = ref<any[]>([])
const noticeText = ref('')
const noticeEditing = ref('')
const savingNotice = ref(false)
const tripList = ref<any[]>([])
const todayDate = ref('')

// 轮播图自动播放
const currentSlide = ref(0)
let slideTimer: ReturnType<typeof setInterval> | null = null
const startSlideTimer = () => {
  if (slideTimer) clearInterval(slideTimer)
  if (bannerList.value.length <= 1) return
  slideTimer = setInterval(() => {
    currentSlide.value = (currentSlide.value + 1) % bannerList.value.length
  }, 3500)
}
const goToSlide = (i: number) => {
  currentSlide.value = i
  if (slideTimer) clearInterval(slideTimer)
  startSlideTimer()
}

// 标题颜色映射
const titleColorMap: Record<string, string> = {
  white: '#ffffff', black: '#000000', yellow: '#ffcc00', red: '#ff3333',
  blue: '#2689ff', green: '#33cc33', cyan: '#00cccc', purple: '#9933ff',
}

// 标题特效类名（1=阴影 2=玻璃 3=液态）
const getEffectClass = (effect: number) => {
  if (effect === 1) return 'banner-effect-shadow'
  if (effect === 2) return 'banner-effect-glass'
  if (effect === 3) return 'banner-effect-liquid'
  return ''
}

const getTitleStyle = (banner: any) => {
  const color = titleColorMap[banner.title_color] || '#ffffff'
  return { color }
}

// 优惠券数据转一下（跟小程序保持一致）
const formatCoupon = (item: any) => {
  let label = '', valueText = ''
  if (item.type === 1) { label = '满减券'; valueText = '¥' + item.discount_value }
  else if (item.type === 2) { label = '折扣券'; valueText = item.discount_value + '折' }
  else { label = '抵用券'; valueText = '¥' + item.discount_value }
  const desc = item.min_spend > 0 ? '满' + item.min_spend + '元可用' : '无门槛'
  const themeClass = item.type === 1 ? '' : item.type === 2 ? 'coupon-left-orange' : 'coupon-left-green'
  const labelClass = item.type === 1 ? '' : item.type === 2 ? 'coupon-label-orange' : 'coupon-label-green'
  return { id: item.id, name: item.name, label, valueText, desc, themeClass, labelClass }
}

// 车次数据转一下（跟小程序保持一致）
const formatTrip = (item: any) => {
  const fromName = item.route && item.route.from_station ? item.route.from_station.name : ''
  const toName = item.route && item.route.to_station ? item.route.to_station.name : ''
  const durationMin = item.route ? item.route.duration_minutes : 0
  const durationStr = durationMin >= 60
    ? `约${Math.floor(durationMin/60)}小时${durationMin%60 > 0 ? durationMin%60 + '分钟' : ''}`
    : `约${durationMin}分钟`
  const seatsText = item.available_seats === 0 ? '无票' : item.available_seats <= 5 ? `余${item.available_seats}座` : '有票'
  const seatsLow = item.available_seats > 0 && item.available_seats <= 5
  const soldOut = item.available_seats === 0
  let arrivalText = ''
  if (item.departure_time) {
    const [h, m] = item.departure_time.split(':').map(Number)
    const total = h * 60 + m + durationMin
    const ah = Math.floor(total / 60) % 24
    const am = total % 60
    arrivalText = `${String(ah).padStart(2, '0')}:${String(am).padStart(2, '0')}`
  }
  return {
    id: item.id, from: fromName, to: toName,
    time: item.departure_time, duration: durationStr,
    price: parseFloat(item.base_price).toFixed(0),
    seats: seatsText, seatsLow, soldOut,
    arrivalText, priceIsInterval: false
  }
}

const formatDate = (d: Date) => {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const phoneScrollRef = ref<HTMLElement | null>(null)
const componentRefs: Record<string, HTMLElement | null> = {}

// 模拟器导航键滚动控制
const scrollPhone = (dir: 'up' | 'down' | 'left' | 'right' | 'center') => {
  const container = phoneScrollRef.value
  if (!container) return
  const scrollAmount = 200
  if (dir === 'up') {
    container.scrollBy({ top: -scrollAmount, behavior: 'smooth' })
  } else if (dir === 'down') {
    container.scrollBy({ top: scrollAmount, behavior: 'smooth' })
  } else if (dir === 'center') {
    container.scrollTo({ top: 0, behavior: 'smooth' })
  } else if (dir === 'left' || dir === 'right') {
    const delta = dir === 'left' ? -scrollAmount : scrollAmount
    // 遍历所有子元素，找到可水平滚动且未到边界的区域
    const allEls = container.querySelectorAll('*')
    let scrolled = false
    for (const el of allEls) {
      const hEl = el as HTMLElement
      const cs = window.getComputedStyle(hEl)
      if ((cs.overflowX === 'auto' || cs.overflowX === 'scroll') && hEl.scrollWidth > hEl.clientWidth + 1) {
        const atRightEnd = hEl.scrollLeft + hEl.clientWidth >= hEl.scrollWidth - 1
        const atLeftEnd = hEl.scrollLeft <= 0
        if ((dir === 'right' && !atRightEnd) || (dir === 'left' && !atLeftEnd)) {
          hEl.scrollBy({ left: delta, behavior: 'smooth' })
          scrolled = true
          break
        }
        // 已到边界，继续查找下一个可滚动元素或切换页面
      }
    }
    // 主容器不参与横滑判断，直接走页面切换
    // 如果没有可横滑内容或已到边界，左右键切换主页面
    if (!scrolled) {
      const pages: Array<'home' | 'order' | 'mine'> = ['home', 'order', 'mine']
      const idx = pages.indexOf(currentPage.value)
      if (dir === 'left' && idx > 0) {
        switchPage(pages[idx - 1])
      } else if (dir === 'right' && idx < pages.length - 1) {
        switchPage(pages[idx + 1])
      }
    }
  }
}

const componentDesc: Record<string, string> = {
  banner: '首页顶部轮播广告图',
  coupon: '优惠券横滑展示区域',
  notice: '平台公告/通知消息',
  search: '出发到达筛选 + 日期 + 搜索框',
  trips: '当日可购票班次列表',
}

const componentIcons: Record<string, string> = {
  banner: bannerCarouselSvg,
  coupon: couponDisplaySvg,
  notice: megaphoneSvg,
  search: searchFilterSvg,
  trips: tripListSvg,
}

const visibleLayout = computed(() => layout.value.filter(item => item.visible))
const visibleOrderTabs = computed(() => orderTabsLayout.value.filter(item => item.visible))
const visibleMineOrderGrid = computed(() => mineOrderGridLayout.value.filter(item => item.visible))
const visibleMineMenu = computed(() => mineMenuLayout.value.filter(item => item.visible))

const hasNoticeComponent = computed(() => layout.value.some(item => item.type === 'notice'))

const saveNotice = async () => {
  savingNotice.value = true
  try {
    await configApi.update({ configs: { notice: noticeEditing.value } })
    noticeText.value = noticeEditing.value
    ElMessage.success('公告保存成功')
  } catch {
    ElMessage.error('保存失败，可能需要超级管理员权限')
  } finally {
    savingNotice.value = false
  }
}

const getOriginalIndex = (type: string) => layout.value.findIndex(item => item.type === type)

const setComponentRef = (type: string, el: any) => {
  if (el) componentRefs[type] = el as HTMLElement
}

const loadData = async () => {
  loading.value = true
  try {
    const res: any = await designApi.getLayout()
    layout.value = res.data || []
    if (layout.value.length === 0) {
      layout.value = [
        { type: 'banner', title: '轮播图', visible: true },
        { type: 'coupon', title: '优惠券展示', visible: true },
        { type: 'notice', title: '公告通知', visible: true },
        { type: 'search', title: '搜索筛选', visible: true },
        { type: 'trips', title: '车次列表', visible: true },
      ]
    }
    // 拉取真实预览数据
    await loadPreviewData()
  } catch {
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
}

// 拉取真实预览数据（使用admin接口，每个接口独立容错，互不影响）
const loadPreviewData = async () => {
  todayDate.value = formatDate(new Date())

  // 轮播图（admin接口返回全部，需过滤 status=1）
  try {
    const res: any = await bannerApi.list()
    const allBanners = res.data || []
    bannerList.value = allBanners.filter((b: any) => b.status === 1)
    startSlideTimer()
  } catch (e) {
    console.error('[装修预览] 轮播图加载失败:', e)
  }

  // 系统配置（公告 + 优惠券ID）
  let configData: any = {}
  try {
    const res: any = await configApi.get()
    configData = res.data || {}
    noticeText.value = configData.notice || ''
    noticeEditing.value = noticeText.value
    servicePhone.value = configData.customer_service_phone || ''
  } catch (e) {
    console.error('[装修预览] 系统配置加载失败:', e)
  }

  // 优惠券（按 homepage_coupon_ids 过滤，只展示 status=1 的）
  try {
    const idsStr = configData.homepage_coupon_ids || ''
    const couponIds = idsStr.split(',').map((s: string) => parseInt(s.trim())).filter((n: number) => !isNaN(n) && n > 0)
    if (couponIds.length > 0) {
      const res: any = await couponApi.list({ page: 1, page_size: 100 })
      const allCoupons = (res.data && res.data.list) || []
      couponList.value = allCoupons
        .filter((c: any) => couponIds.includes(c.id) && c.status === 1)
        .map(formatCoupon)
    }
  } catch (e) {
    console.error('[装修预览] 优惠券加载失败:', e)
  }

  // 车次（今日可售班次，过滤 status=1）
  try {
    const res: any = await tripApi.list({ trip_date: todayDate.value, status: 1, page: 1, page_size: 10 })
    const allTrips = (res.data && res.data.list) || []
    tripList.value = allTrips.map(formatTrip)
  } catch (e) {
    console.error('[装修预览] 车次加载失败:', e)
  }
  // 无真实车次时用模拟数据兜底，保证装修预览效果完整
  if (tripList.value.length === 0) {
    tripList.value = [
      { id: 1, from: '市区客运站', to: '青田镇', time: '15:30', duration: '约1小时30分钟', price: '25', seats: '有票', seatsLow: false, soldOut: false, arrivalText: '17:00', priceIsInterval: false },
      { id: 2, from: '青田镇', to: '市区客运站', time: '08:00', duration: '约1小时30分钟', price: '25', seats: '余3座', seatsLow: true, soldOut: false, arrivalText: '09:30', priceIsInterval: false },
      { id: 3, from: '市区客运站', to: '青田镇', time: '09:00', duration: '约1小时30分钟', price: '25', seats: '无票', seatsLow: false, soldOut: true, arrivalText: '10:30', priceIsInterval: false },
    ]
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    if (currentPage.value === 'home') {
      await designApi.updateLayout(layout.value)
    } else if (currentPage.value === 'order') {
      await designApi.updatePageLayout('order_tabs', orderTabsLayout.value)
    } else if (currentPage.value === 'mine') {
      await designApi.updatePageLayout('mine_order_grid', mineOrderGridLayout.value)
      await designApi.updatePageLayout('mine_menu', mineMenuLayout.value)
      await configApi.update({ configs: { mine_menu_layout_type: mineMenuLayoutType.value, logout_position: logoutPosition.value } })
    }
    ElMessage.success('保存成功')
    showFloatBtn.value = false
  } catch {
    ElMessage.error('保存失败，可能需要超级管理员权限')
  } finally {
    saving.value = false
  }
}

const moveUp = (index: number) => {
  if (index === 0) return
  const temp = layout.value[index]
  layout.value[index] = layout.value[index - 1]
  layout.value[index - 1] = temp
  showFloatBtn.value = true
}

const moveDown = (index: number) => {
  if (index === layout.value.length - 1) return
  const temp = layout.value[index]
  layout.value[index] = layout.value[index + 1]
  layout.value[index + 1] = temp
  showFloatBtn.value = true
}

// 通用上下移动（订单/我的页用，操作任意布局数组）
const moveUpGeneric = (list: any[], index: number) => {
  if (index === 0) return
  const temp = list[index]
  list[index] = list[index - 1]
  list[index - 1] = temp
  showFloatBtn.value = true
}
const moveDownGeneric = (list: any[], index: number) => {
  if (index === list.length - 1) return
  const temp = list[index]
  list[index] = list[index + 1]
  list[index + 1] = temp
  showFloatBtn.value = true
}

// 切页面（用到了才加载对应布局）
const pageLoaded = ref<Record<string, boolean>>({ home: true })
const switchPage = async (page: 'home' | 'order' | 'mine') => {
  closeSubPage()
  if (currentPage.value === page) return
  currentPage.value = page
  showFloatBtn.value = false
  if (!pageLoaded.value[page]) {
    pageLoaded.value[page] = true
    if (page === 'order') await loadOrderLayout()
    else if (page === 'mine') await loadMineLayout()
  }
}

// 订单页标签布局
const loadOrderLayout = async () => {
  loadingOrder.value = true
  try {
    const res: any = await designApi.getPageLayout('order_tabs')
    orderTabsLayout.value = res.data && res.data.length ? res.data : defaultOrderTabs()
  } catch {
    orderTabsLayout.value = defaultOrderTabs()
  } finally {
    loadingOrder.value = false
  }
}

// 我的页布局（订单分类 + 功能菜单）
const loadMineLayout = async () => {
  loadingMine.value = true
  try {
    const [gridRes, menuRes, cfgRes]: any = await Promise.all([
      designApi.getPageLayout('mine_order_grid'),
      designApi.getPageLayout('mine_menu'),
      configApi.get(),
    ])
    mineOrderGridLayout.value = gridRes.data && gridRes.data.length ? gridRes.data : defaultMineOrderGrid()
    mineMenuLayout.value = menuRes.data && menuRes.data.length ? menuRes.data : defaultMineMenu()
    mineMenuLayoutType.value = (cfgRes.data && cfgRes.data.mine_menu_layout_type === 'grid') ? 'grid' : 'list'
    logoutPosition.value = cfgRes.data && cfgRes.data.logout_position === 'profile_detail' ? 'profile_detail' : 'mine_bottom'
  } catch {
    mineOrderGridLayout.value = defaultMineOrderGrid()
    mineMenuLayout.value = defaultMineMenu()
  } finally {
    loadingMine.value = false
  }
}

// 点击组件 → 预览区滚动定位
const scrollToComponent = (index: number) => {
  activeIndex.value = index
  const item = layout.value[index]
  if (!item || !item.visible) return
  nextTick(() => {
    const el = componentRefs[item.type]
    if (el && phoneScrollRef.value) {
      const scrollContainer = phoneScrollRef.value
      const targetTop = el.offsetTop - scrollContainer.offsetTop
      scrollContainer.scrollTo({ top: targetTop - 8, behavior: 'smooth' })
    }
  })
}

// 拖拽排序
const onDragStart = (index: number) => {
  dragIndex.value = index
}

const onDragOver = (index: number) => {
  dragOverIndex.value = index
}

const onDrop = (index: number) => {
  if (dragIndex.value === -1 || dragIndex.value === index) return
  const dragItem = layout.value[dragIndex.value]
  layout.value.splice(dragIndex.value, 1)
  layout.value.splice(index, 0, dragItem)
  dragIndex.value = -1
  dragOverIndex.value = -1
  showFloatBtn.value = true
}

const onDragEnd = () => {
  dragIndex.value = -1
  dragOverIndex.value = -1
}

onMounted(loadData)

onUnmounted(() => {
  if (slideTimer) clearInterval(slideTimer)
  if (flipTimer1) clearTimeout(flipTimer1)
  if (flipTimer2) clearTimeout(flipTimer2)
  // 补充清理遗漏的定时器，防止内存泄漏
  if (couponToastTimer) clearTimeout(couponToastTimer)
})
</script>

<style scoped>
.decorate-wrapper {
  display: flex;
  gap: 20px;
  padding: 0 0 20px;
  min-width: 820px;
}

/* 左侧配置区 */
.config-panel {
  flex: 1;
  min-width: 340px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
}

.component-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 24px;
}

.component-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
}

.component-item:hover {
  border-color: #409eff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.12);
  transform: translateY(-1px);
}

.component-item.active {
  border-color: #409eff;
  background: #ecf5ff;
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.1);
}

.component-item.drag-over {
  border-color: #67c23a;
  border-style: dashed;
  border-width: 2px;
}

.component-item.hidden {
  opacity: 0.55;
  background: #f9f9f9;
}

.component-item:active {
  cursor: grabbing;
}

.drag-handle {
  cursor: grab;
  color: #c0c4cc;
  display: flex;
  align-items: center;
  transition: color 0.2s;
}

.drag-handle:hover {
  color: #909399;
}

.component-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  background: #f5f7fa;
  border-radius: 8px;
  flex-shrink: 0;
  transition: background 0.2s;
}

.component-icon :deep(svg) {
  width: 22px;
  height: 22px;
}

.component-item:hover .component-icon {
  background: #eef4fc;
}

.component-info {
  flex: 1;
  min-width: 0;
}

.component-title {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
}

.component-desc {
  font-size: 12px;
  color: #999;
  margin-top: 2px;
}

.component-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.notice-editor {
  margin-bottom: 16px;
  padding: 16px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
}

.notice-editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.notice-editor-title {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
}

.notice-editor-hint {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
}

/* 黑色保存按钮 */
.save-btn-black {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 14px;
  background: #1a1a1a;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.save-btn-black:hover {
  background: #333;
}
.save-btn-black:active {
  transform: scale(0.97);
}
.save-btn-black:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.save-btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.save-btn-icon svg {
  width: 14px;
  height: 14px;
}

/* 布局切换按钮 */
.layout-switch {
  display: flex;
  gap: 0;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #dcdfe6;
}
.layout-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: #fff;
  color: #303133;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}
.layout-btn:hover {
  background: #f5f5f5;
}
.layout-btn.active {
  background: #1a1a1a;
  color: #fff;
}
.layout-btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.layout-btn-icon :deep(svg) {
  width: 16px;
  height: 16px;
  /* 图标颜色跟随按钮 color（SVG fill=currentColor）：未选中深灰可见，选中白色 */
}
/* 上下移动按钮：透明背景无边框，纯图标显示，放大尺寸更醒目 */
.component-actions .el-button {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  padding: 0 !important;
  width: 30px !important;
  height: 30px !important;
  transition: transform 0.15s, color 0.15s;
}
.component-actions .el-button:hover {
  background: transparent !important;
  color: #1a1a1a;
}
.component-actions .el-button:active {
  transform: scale(0.9);
}
.component-actions .el-button.is-disabled {
  opacity: 0.35;
}
.component-actions .move-btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
}
.component-actions .move-btn-icon :deep(svg) {
  width: 22px;
  height: 22px;
}

.quick-tips {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 10px;
}

.tip-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #909399;
}

/* 右侧预览区 */
.preview-panel {
  width: 460px;
  flex-shrink: 0;
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  padding-top: 10px;
  padding-bottom: 10px;
  position: sticky;
  top: 16px;
  align-self: flex-start;
  overflow: visible;
}

/* 手机外壳容器 */
.phone-wrapper {
  position: relative;
  display: inline-block;
}

/* 侧边按键（钛金属质感） */
.phone-side-btn {
  position: absolute;
  background: linear-gradient(to bottom, #6a6a6e 0%, #3a3a3c 50%, #6a6a6e 100%);
  border-radius: 1px;
  z-index: 1;
}
.side-btn-action {
  left: -4px;
  top: 105px;
  width: 4px;
  height: 22px;
}
.side-btn-vol-up {
  left: -4px;
  top: 145px;
  width: 4px;
  height: 45px;
}
.side-btn-vol-down {
  left: -4px;
  top: 200px;
  width: 4px;
  height: 45px;
}

.phone-frame {
  width: 320px;
  height: min(640px, calc(100vh - 100px));
  background: #f5f6f8;
  border-radius: 48px;
  border: 3px solid #1a1a1a;
  box-shadow:
    0 0 0 2px #545456,
    0 0 0 3px #3a3a3c,
    0 0 0 4px #2a2a2c,
    0 12px 48px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
/* Apple iPhone 17 Pro Max: 暖橙边框 */
.phone-frame.apple,
.phone-3d-flipper.apple .phone-back-face {
  border-color: #c4703c;
  box-shadow: 0 0 0 2px #d48048, 0 0 0 3px #a46028, 0 0 0 4px #945018, 0 12px 48px rgba(0, 0, 0, 0.25);
}
.phone-3d-flipper.apple .phone-back-face {
  background: #c4703c;
}

/* Pixel 10: 薄荷绿边框 */
.phone-frame.pixel,
.phone-3d-flipper.pixel .phone-back-face {
  border-color: #6a9888;
  box-shadow: 0 0 0 2px #7aa898, 0 0 0 3px #5a8878, 0 0 0 4px #4a7868, 0 12px 48px rgba(0, 0, 0, 0.25);
}
.phone-3d-flipper.pixel .phone-back-face {
  background: #6a9888;
}

/* Huawei Pura 80 Pro: 玫瑰金边框 + 圆角略小 */
.phone-frame.huawei {
  border-radius: 38px;
  border-color: #b87878;
  box-shadow: 0 0 0 2px #c88888, 0 0 0 3px #a86868, 0 0 0 4px #985858, 0 12px 48px rgba(0, 0, 0, 0.25);
}
.phone-3d-flipper.huawei .phone-back-face {
  border-radius: 38px;
  border-color: #b87878;
  box-shadow: 0 0 0 2px #c88888, 0 0 0 3px #a86868, 0 0 0 4px #985858, 0 12px 48px rgba(0, 0, 0, 0.25);
  background: #b87878;
}
.phone-3d-flipper.huawei .back-face-img {
  border-radius: 35px;
}

/* Samsung Galaxy S26 Ultra: 黑色边框 + 方形圆角18px + Samsung细字体 */
.phone-frame.samsung {
  border-radius: 18px;
  border-color: #2a2a2a;
  box-shadow: 0 0 0 2px #3a3a3a, 0 0 0 3px #252525, 0 0 0 4px #1a1a1a, 0 12px 48px rgba(0, 0, 0, 0.25);
  font-family: 'Roboto', 'Noto Sans SC', 'Microsoft YaHei', 'Segoe UI', sans-serif;
  font-weight: 300;
}
.phone-frame.samsung .phone-statusbar,
.phone-frame.samsung .phone-scroll,
.phone-frame.samsung .phone-scroll *:not(.phone-statusbar) {
  font-family: 'Roboto', 'Noto Sans SC', 'Microsoft YaHei', 'Segoe UI', sans-serif;
  font-weight: 300 !important;
}
.phone-frame.samsung .phone-scroll h1,
.phone-frame.samsung .phone-scroll h2,
.phone-frame.samsung .phone-scroll h3,
.phone-frame.samsung .phone-scroll .card-title,
.phone-frame.samsung .phone-scroll .nav-text,
.phone-frame.samsung .phone-scroll p,
.phone-frame.samsung .phone-scroll span,
.phone-frame.samsung .phone-scroll div {
  font-weight: 300 !important;
}
.phone-3d-flipper.samsung .phone-back-face {
  border-radius: 18px;
  border-color: #2a2a2a;
  box-shadow: 0 0 0 2px #3a3a3a, 0 0 0 3px #252525, 0 0 0 4px #1a1a1a, 0 12px 48px rgba(0, 0, 0, 0.25);
  background: #2a2a2a;
}
.phone-3d-flipper.samsung .back-face-img {
  border-radius: 15px;
}

/* 灵动岛 */
.dynamic-island {
  position: absolute;
  top: 10px;
  left: 50%;
  transform: translateX(-50%);
  width: 95px;
  height: 26px;
  background: #000;
  border-radius: 13px;
  z-index: 20;
}

.dynamic-island-camera {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  width: 6px;
  height: 6px;
  background: #1a1a2e;
  border-radius: 50%;
  box-shadow: inset 0 0 2px rgba(100, 100, 255, 0.3);
}

/* 状态栏 */
.phone-statusbar {
  height: 44px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
  flex-shrink: 0;
  position: relative;
  z-index: 5;
}

.status-time {
  letter-spacing: 0.5px;
}

.status-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

/* Pixel 打孔屏摄像头 */
.punch-hole-camera {
  position: absolute;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  width: 12px;
  height: 12px;
  background: #000;
  border-radius: 50%;
  z-index: 20;
}

/* Huawei 圆形打孔屏（Pura系列） */
.huawei-punch-hole {
  position: absolute;
  top: 11px;
  left: 50%;
  transform: translateX(-50%);
  width: 14px;
  height: 14px;
  background: #000;
  border-radius: 50%;
  z-index: 20;
}

/* Samsung 圆形打孔屏 */
.samsung-punch-hole {
  position: absolute;
  top: 11px;
  left: 50%;
  transform: translateX(-50%);
  width: 13px;
  height: 13px;
  background: #000;
  border-radius: 50%;
  z-index: 20;
}

/* 机型切换 */
.left-controls {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex-shrink: 0;
  width: 120px;
  align-self: stretch;
  justify-content: flex-start;
}

/* 3D翻转 */
.phone-3d-container {
  perspective: 1200px;
}
.phone-3d-flipper {
  position: relative;
  transform-style: preserve-3d;
  /* 显式设置初始transform，确保移除.flipping类时从rotateY(180deg)回转到rotateY(0deg)的transition在所有浏览器中均触发 */
  transform: rotateY(0deg);
  transition: transform 0.8s cubic-bezier(0.4, 0, 0.2, 1);
}
.phone-3d-flipper.flipping {
  transform: rotateY(180deg);
}
.phone-back-face,
.phone-front-face {
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
}
.phone-front-face {
  position: relative;
  z-index: 2;
}
.phone-back-face {
  position: absolute;
  top: 0;
  left: 0;
  width: 320px;
  height: min(640px, calc(100vh - 100px));
  transform: rotateY(180deg);
  border-radius: 48px;
  border: 3px solid #1a1a1a;
  box-shadow: 0 0 0 2px #545456, 0 0 0 3px #3a3a3c, 0 0 0 4px #2a2a2c, 0 12px 48px rgba(0,0,0,0.25);
  overflow: hidden;
  background: #000;
}
.back-face-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 45px;
  display: block;
}

/* 机型名称显示 */
/* 机型切换器包装 */
.model-switcher-wrapper {
  position: relative;
  margin-top: auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
}

/* 机型名称显示 */
.model-name-display {
  font-size: 11px;
  font-weight: 600;
  color: #1a1a1a;
  text-align: center;
  line-height: 1.3;
  width: 100%;
  word-break: break-word;
}
.model-os-badge {
  display: inline-block;
  font-size: 9px;
  padding: 1px 5px;
  border-radius: 4px;
  background: #1a1a1a;
  color: #fff;
  font-weight: 500;
  margin-top: 2px;
}

/* 机型切换按钮 - 滑动开关样式 */
.model-switch-btn {
  margin-top: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 10px;
  background: linear-gradient(135deg, #2a2a2a, #1a1a1a);
  color: #fff;
  border: none;
  border-radius: 20px;
  font-size: 10px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  user-select: none;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
  position: relative;
  overflow: hidden;
  width: 100%;
  max-width: 110px;
}
.model-switch-btn::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.15), transparent);
  transition: left 0.5s ease;
}
.model-switch-btn:hover::before {
  left: 100%;
}
.model-switch-btn:hover {
  background: linear-gradient(135deg, #3a3a3a, #2a2a2a);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.25);
}
.model-switch-btn:active {
  transform: scale(0.95);
}
.model-switch-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}
.model-switch-icon svg {
  width: 14px;
  height: 14px;
}
.model-switch-icon :deep(svg path) {
  fill: #fff !important;
}

/* 机型选择弹窗 */
.model-picker-popup {
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  margin-bottom: 6px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.15);
  padding: 0 0 8px 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  z-index: 100;
  white-space: nowrap;
  overflow: hidden;
}
/* 弹窗标题栏 */
.model-picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid #f0f0f0;
}
.model-picker-title {
  font-size: 12px;
  font-weight: 600;
  color: #606266;
  letter-spacing: 0.5px;
}
/* 弹窗关闭按钮 */
.model-picker-close {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  line-height: 1;
  color: #909399;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.2s;
}
.model-picker-close:hover {
  background: #f0f0f0;
  color: #303133;
}
.model-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  margin: 0 8px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}
.model-option:hover {
  background: #f5f5f5;
}
.model-option.active {
  background: #f0f0f0;
}
.model-option-back {
  position: relative;
  width: 32px;
  height: 44px;
  border-radius: 6px;
  flex-shrink: 0;
  background-size: cover;
  background-position: center;
  overflow: hidden;
}
/* 缩略图品牌Logo叠加层 - 白色背景完全遮盖手机图 */
.model-option-logo {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
}
.model-option-logo :deep(svg) {
  width: 22px;
  height: 22px;
}
.model-option-name {
  font-size: 13px;
  font-weight: 600;
  color: #1a1a1a;
}
.model-option-os {
  font-size: 11px;
  color: #909399;
}

/* 点击外部关闭遮罩 */
.model-picker-overlay {
  position: fixed;
  inset: 0;
  z-index: 99;
  /* 透明遮罩，捕获弹窗外的点击 */
}
.model-picker-fade-enter-active,
.model-picker-fade-leave-active {
  transition: opacity 0.25s ease;
}
.model-picker-fade-enter-from,
.model-picker-fade-leave-to {
  opacity: 0;
}

/* 弹窗动画 - 向上滑出 */
.model-picker-enter-active,
.model-picker-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.model-picker-enter-from,
.model-picker-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(12px) scale(0.95);
}

/* 导航栏（毛玻璃） */
.phone-navbar {
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.65);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border-bottom: 0.5px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.navbar-title {
  font-size: 16px;
  font-weight: 700;
  color: #1a1a1a;
}

/* 可滚动内容区 */
.phone-scroll {
  flex: 1;
  overflow-y: auto;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scroll-behavior: smooth;
  scrollbar-width: none;
}

.phone-scroll::-webkit-scrollbar {
  display: none;
}

/* 模拟器导航控制 D-pad */
.simulator-dpad {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  margin-right: 8px;
  flex-shrink: 0;
}
.dpad-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
.dpad-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #fff;
  border: 1px solid #dcdfe6;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
}
.dpad-btn:hover {
  background: #f0f0f0;
  border-color: #c0c4cc;
}
.dpad-btn:active {
  transform: scale(0.9);
  background: #e8e8e8;
}
.dpad-btn.dpad-center {
  background: #1a1a1a;
  border-color: #1a1a1a;
}
.dpad-btn.dpad-center:hover {
  background: #333;
  border-color: #333;
}
.dpad-spacer {
  width: 36px;
  height: 36px;
}
.dpad-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  pointer-events: none;
}
.dpad-icon :deep(svg) {
  width: 18px;
  height: 18px;
}
.dpad-btn.dpad-center .dpad-icon :deep(svg) path {
  fill: #fff !important;
}

/* 高亮选中组件 */
.preview-highlight {
  position: relative;
  box-shadow: 0 0 0 2px #409eff, 0 0 12px rgba(64, 158, 255, 0.25);
  border-radius: 12px;
  z-index: 1;
}

/* 轮播图 */
.preview-banner {
  padding: 0 16px;
}

.banner-viewport {
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  height: 180px;
}

.banner-track {
  display: flex;
  height: 100%;
  transition: transform 0.5s cubic-bezier(0.25, 0.1, 0.25, 1);
}

.banner-slide {
  position: relative;
  flex: 0 0 100%;
  height: 100%;
  overflow: hidden;
  display: flex;
  align-items: flex-end;
}

/* 轮播图主题背景 */
.slide-blue {
  background: linear-gradient(135deg, #4facfe 0%, #00c6fb 40%, #6a82fb 100%);
}
.slide-orange {
  background: linear-gradient(135deg, #ff9a56 0%, #ff6a88 50%, #ff5e62 100%);
}
.slide-green {
  background: linear-gradient(135deg, #43e97b 0%, #38d39f 50%, #2bb673 100%);
}

/* 装饰元素 */
.banner-bg-elements {
  position: absolute;
  inset: 0;
  overflow: hidden;
}

.bg-circle {
  position: absolute;
  border-radius: 50%;
}

.bg-circle-1 {
  top: -20px;
  right: -10px;
  width: 80px;
  height: 80px;
  background: rgba(255, 255, 255, 0.12);
}

.bg-circle-2 {
  bottom: 30px;
  right: 20px;
  width: 40px;
  height: 40px;
  background: rgba(255, 255, 255, 0.08);
}

.bg-road {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 45px;
  background: linear-gradient(180deg, transparent, rgba(0,0,0,0.2));
}

/* 巴士插图 */
.banner-bus {
  position: absolute;
  right: 10px;
  bottom: 8px;
  z-index: 1;
  opacity: 0.9;
}

.banner-bus :deep(svg) {
  width: 80px;
  height: auto;
}

.banner-bus :deep(svg path) {
  fill: rgba(255, 255, 255, 0.85) !important;
}

/* 文案区 */
.banner-text {
  position: absolute;
  top: 14px;
  left: 16px;
  z-index: 2;
  max-width: calc(100% - 32px);
}

.banner-effect-shadow .banner-title {
  text-shadow: 0 1px 6px rgba(0,0,0,0.45);
}

.banner-effect-glass {
  background: rgba(255, 255, 255, 0.18);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border-radius: 6px;
  padding: 4px 10px;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.banner-effect-glass .banner-title {
  text-shadow: none;
}

/* 液态玻璃特效 - 更透明更通透 */
.banner-effect-liquid {
  background: rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(2px) saturate(180%);
  -webkit-backdrop-filter: blur(2px) saturate(180%);
  border-radius: 6px;
  padding: 4px 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 1px 3px rgba(0,0,0,0.06), inset 0 1px 0 rgba(255,255,255,0.12);
}

.banner-effect-liquid .banner-title {
  text-shadow: none;
}

.banner-badge {
  display: inline-block;
  padding: 1px 8px;
  background: rgba(255, 255, 255, 0.25);
  backdrop-filter: blur(4px);
  border-radius: 10px;
  font-size: 9px;
  font-weight: 800;
  color: #fff;
  letter-spacing: 1px;
  margin-bottom: 6px;
}

.banner-title {
  font-size: 16px;
  font-weight: 800;
  text-shadow: 0 1px 6px rgba(0,0,0,0.25);
}

.banner-sub {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.92);
  margin-top: 3px;
  text-shadow: 0 1px 3px rgba(0,0,0,0.15);
}

/* 指示点 */
.banner-dots {
  position: absolute;
  bottom: 6px;
  right: 14px;
  display: flex;
  gap: 4px;
  z-index: 3;
}

.banner-dots .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  transition: all 0.3s;
}

.banner-dots .dot.active {
  width: 16px;
  border-radius: 3px;
  background: #fff;
}

/* 真实轮播图图片 */
.slide-real {
  background: #f0f0f0;
}

.banner-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* 轮播图空状态 */
.banner-empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: linear-gradient(135deg, #e8eef5 0%, #f5f7fa 100%);
}

.banner-empty-icon :deep(svg) {
  width: 40px;
  height: auto;
  opacity: 0.3;
}

.banner-empty-icon :deep(svg path) {
  fill: #909399 !important;
}

.banner-empty-text {
  font-size: 12px;
  color: #909399;
}

/* 通用空状态 */
.preview-empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  font-size: 13px;
  color: #bbb;
}

/* 优惠券 */
.preview-coupon {
  margin: 12px 16px 0;
}

.coupon-section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
}

.coupon-section-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
}

.coupon-section-icon :deep(svg) {
  width: 24px;
  height: 24px;
}

.coupon-section-title {
  font-size: 17px;
  font-weight: 700;
  color: #1a1a1a;
}

.preview-coupon-scroll {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  overflow-y: hidden;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  scroll-snap-type: x mandatory;
  padding-bottom: 4px;
}

.preview-coupon-scroll::-webkit-scrollbar {
  display: none;
}

.preview-coupon-card {
  position: relative;
  display: flex;
  width: 150px;
  height: 100px;
  border-radius: 10px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
  scroll-snap-align: start;
}

/* 穿孔虚线已移除：与小程序客户端保持一致，客户端无此装饰元素 */

.preview-coupon-left {
  width: 60px;
  background: linear-gradient(135deg, #2689FF, #1a6dd0);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.coupon-left-orange {
  background: linear-gradient(135deg, #52a4ff, #2689FF);
}

.coupon-left-green {
  background: linear-gradient(135deg, #1a6dd0, #0d5fb0);
}

.coupon-value {
  font-size: 16px;
  font-weight: bold;
  color: #fff;
}

.coupon-desc {
  font-size: 9px;
  color: rgba(255, 255, 255, 0.9);
  margin-top: 4px;
}

.preview-coupon-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 0 10px;
  overflow: hidden;
}

.preview-coupon-claim-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-top: 6px;
  padding: 3px 12px;
  background: linear-gradient(135deg, #2689FF, #1a6dd0);
  border-radius: 14px;
  font-size: 11px;
  color: #fff;
  font-weight: 500;
  align-self: flex-start;
}

.preview-coupon-claim-btn.claimed {
  background: #e0e0e0;
  color: #999;
}

.preview-coupon-label {
  font-size: 11px;
  color: #2689FF;
  font-weight: 500;
}

.coupon-label-orange {
  color: #52a4ff;
}

.coupon-label-green {
  color: #1a6dd0;
}

.preview-coupon-name {
  font-size: 13px;
  color: #333;
  margin-top: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 公告 */
.preview-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 12px 16px;
  padding: 10px 16px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #1a1a1a;
  overflow: hidden;
}

.notice-megaphone-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.notice-megaphone-icon :deep(svg) {
  width: 20px;
  height: 20px;
}

.notice-marquee {
  flex: 1;
  overflow: hidden;
}

.notice-text {
  display: inline-block;
  font-size: 13px;
  color: #1a1a1a;
  white-space: nowrap;
  animation: notice-scroll 12s linear infinite;
}

@keyframes notice-scroll {
  0% { transform: translateX(100%); }
  100% { transform: translateX(-100%); }
}

.notice-placeholder {
  color: #bbb !important;
  animation: none !important;
}

/* 搜索筛选 */
.preview-search {
  margin: 24px 16px 0;
}

.preview-station-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #fff;
  border-radius: 12px;
}

.preview-station-box {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.preview-label {
  font-size: 12px;
  color: #999;
}

.preview-station-name {
  font-size: 16px;
  font-weight: 600;
  color: #1296db;
  margin-top: 4px;
}

.preview-swap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
}

.preview-swap :deep(svg) {
  width: 22px;
  height: 22px;
}

.preview-date-bar {
  display: flex;
  flex-direction: row;
  align-items: center;
  margin: 10px 0 0;
  padding: 8px 16px;
  font-size: 14px;
  color: #1a1a1a;
}

/* 日期栏内 label 样式覆盖通用 .preview-label（小程序中 date-label 是 14px/#1a1a1a，不同于站点筛选的 12px/#999） */
.preview-date-bar .preview-label {
  font-size: 14px;
  color: #1a1a1a;
  margin-right: 10px;
}

.preview-date-bar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}

.preview-glass-refresh {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 6px 22px;
  border-radius: 18px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  font-size: 13px;
  font-weight: 600;
  color: #1a1a1a;
  letter-spacing: 3px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.preview-glass-refresh:hover {
  transform: scale(0.96);
}

.preview-date {
  font-size: 16px;
  font-weight: 600;
  color: #1296db;
}

.preview-search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  background: #fff;
  border-radius: 22px;
  padding: 0 16px;
  margin-top: 4px;
}

.preview-search-icon {
  display: flex;
  align-items: center;
}

.preview-search-icon :deep(svg) {
  width: 20px;
  height: 20px;
}

.preview-search-placeholder {
  font-size: 14px;
  color: #bbb;
}

/* 车次列表 */
.preview-trips {
  padding: 0 16px 12px;
}
.preview-trips-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.preview-route-card {
  background: #fff;
  border-radius: 12px;
  padding: 18px 20px;
  border: 1px solid #e0e0e0;
  box-shadow: 0 1px 4px rgba(0,0,0,0.05);
}

.preview-route-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.preview-route-path {
  display: flex;
  align-items: center;
  flex: 1;
}

.preview-route-from,
.preview-route-to {
  font-size: 19px;
  font-weight: bold;
  color: #1a1a1a;
}

.preview-route-arrow {
  display: flex;
  align-items: center;
  margin: 0 10px;
}

.preview-route-arrow :deep(svg) {
  width: 20px;
  height: 20px;
}

.preview-route-seats {
  display: inline-block;
  font-size: 12px;
  font-weight: 500;
  color: #1a1a1a;
  padding: 2px 10px;
  background: transparent;
  border: 1px solid #d0d0d0;
  border-radius: 12px;
}

.preview-route-seats.seats-low {
  color: #1a1a1a;
}

.preview-route-seats.seats-out {
  color: #ccc;
  border-color: #e5e5e5;
}

.preview-route-btn.btn-disabled {
  background: #e0e0e0;
  color: #999;
  box-shadow: none;
  cursor: not-allowed;
}

.preview-route-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
}

.preview-route-info {
  display: flex;
  flex-direction: row;
  align-items: center;
  flex-wrap: wrap;
  flex: 1;
  min-width: 0;
}

.preview-route-bus {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  margin-right: 8px;
}

.preview-route-bus :deep(svg) {
  width: 20px;
  height: 20px;
}

.preview-route-bus :deep(svg path) {
  fill: #2c2c2c !important;
}

/* 排序栏 */
.preview-sort-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0 12px;
  overflow-x: auto;
  white-space: nowrap;
}
.preview-sort-chip {
  flex-shrink: 0;
  font-size: 13px;
  color: #1a1a1a;
  padding: 6px 16px;
  background: #fff;
  border: 1px solid #1a1a1a;
  border-radius: 16px;
  cursor: pointer;
}
.preview-sort-chip.active {
  color: #fff;
  background: #1a1a1a;
  border-color: #1a1a1a;
  font-weight: 500;
}

/* 途经走向 */
.preview-route-via {
  display: flex;
  align-items: center;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #f5f5f5;
}
.preview-via-tag {
  flex-shrink: 0;
  font-size: 11px;
  color: #1a1a1a;
  padding: 2px 8px;
  border: 1px solid #1a1a1a;
  border-radius: 4px;
  margin-right: 10px;
}
.preview-via-total {
  font-size: 12px;
  color: #666;
}
.preview-via-more {
  flex: 1;
  text-align: right;
  font-size: 12px;
  color: #666;
}

.preview-route-time-group {
  display: flex;
  flex-direction: row;
  align-items: center;
}

.preview-route-time {
  font-size: 16px;
  color: #333;
  font-weight: 500;
}

.preview-route-duration {
  font-size: 13px;
  color: #666;
  margin-left: 8px;
  flex-shrink: 0;
  white-space: nowrap;
}

.preview-route-arrival {
  font-size: 13px;
  color: #666;
  margin-left: 8px;
  flex-shrink: 0;
  white-space: nowrap;
}

.preview-route-price-unit {
  font-size: 12px;
  font-weight: 400;
  color: #999;
}

.preview-route-action {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.preview-route-price {
  font-size: 22px;
  font-weight: bold;
  color: #1a1a1a;
  margin-right: 14px;
}

.preview-route-btn {
  background: #1a1a1a;
  color: #fff;
  font-size: 16px;
  font-weight: 500;
  padding: 8px 24px;
  border-radius: 22px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

/* 底部 TabBar */
.phone-tabbar {
  height: 56px;
  background: rgba(255, 255, 255, 0.65);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border-top: 0.5px solid rgba(0, 0, 0, 0.06);
  display: flex;
  flex-shrink: 0;
}

.tabbar-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  cursor: pointer;
  transition: opacity 0.2s;
}
.tabbar-item:hover {
  opacity: 0.7;
}

.tabbar-icon {
  width: 22px;
  height: 22px;
  opacity: 0.45;
}

.tabbar-item.active .tabbar-icon {
  opacity: 1;
}

.tabbar-text {
  font-size: 10px;
  color: #8a8a8a;
  line-height: 1.2;
}

.tabbar-item.active .tabbar-text {
  color: #1a1a1a;
  font-weight: 600;
}

/* 悬浮保存按钮 */
.float-save-btn {
  position: fixed;
  bottom: 32px;
  right: 32px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 24px;
  background: #1a1a1a;
  color: #fff;
  border-radius: 28px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.3);
  z-index: 100;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.float-save-btn:hover {
  transform: translateY(-2px) scale(1.03);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.4);
}

.float-save-btn:active {
  transform: translateY(0) scale(0.98);
}
.float-save-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.float-save-icon svg {
  width: 16px;
  height: 16px;
}

/* 过渡动画 */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(20px) scale(0.9);
}

/* 页面切换 tabs */
.page-tabs {
  display: flex;
  gap: 0;
  margin-bottom: 16px;
  background: #fff;
  border-radius: 10px;
  padding: 4px;
  border: 1px solid #ebeef5;
}
.page-tabs-bottom {
  position: sticky;
  bottom: 0;
  margin-bottom: 0;
  margin-top: 16px;
  box-shadow: 0 -2px 12px rgba(0, 0, 0, 0.08);
  z-index: 10;
}
.page-tab {
  flex: 1;
  text-align: center;
  padding: 8px 0;
  font-size: 14px;
  color: #595959;
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.2s;
}
.page-tab:hover {
  color: #1a1a1a;
}
.page-tab.active {
  background: #1a1a1a;
  color: #fff;
  font-weight: 600;
}

/* 分组标签 */
.section-label {
  font-size: 13px;
  font-weight: 600;
  color: #595959;
  margin: 12px 0 8px;
}
.mine-menu-layout-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 12px 0 8px;
}
.mine-menu-layout-row .section-label {
  margin: 0;
}

/* 退出按钮位置配置卡片 */
.logout-position-config {
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 12px 14px;
  margin: 14px 0 10px;
}
.logout-position-config-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 10px;
}
.logout-position-config-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.logout-position-config-icon :deep(svg) {
  width: 16px;
  height: 16px;
}
.logout-position-switch {
  display: flex;
  gap: 0;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #dcdfe6;
}
.logout-position-btn {
  flex: 1;
  text-align: center;
  padding: 8px 4px;
  background: #fff;
  color: #606266;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}
.logout-position-btn:hover {
  background: #f5f5f5;
}
.logout-position-btn.active {
  background: #1a1a1a;
  color: #fff;
}

/* 我的页菜单缩略图 */
.menu-thumb {
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  object-fit: contain;
}

/* 订单页预览 */
.preview-filter-tabs {
  display: flex;
  background: #fff;
  padding: 0 16px;
}
.preview-filter-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 40px;
  position: relative;
  cursor: pointer;
}
.preview-filter-tab span {
  font-size: 14px;
  color: #8a8a8a;
}
.preview-filter-tab.active span {
  color: #1a1a1a;
  font-weight: 600;
}
.preview-tab-underline {
  position: absolute;
  bottom: 0;
  width: 24px;
  height: 3px;
  background: #1296db;
  border-radius: 2px;
}
.preview-order-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  font-size: 14px;
  color: #bbb;
}

/* 我的页预览 */
.preview-mine-user-card {
  display: flex;
  align-items: center;
  margin: 14px 16px;
  padding: 24px 20px;
  background: #fff;
  border-radius: 12px;
}
.preview-mine-avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  border: 2px solid #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
}
.preview-mine-user-info {
  display: flex;
  flex-direction: column;
  margin-left: 16px;
}
.preview-mine-name {
  font-size: 20px;
  font-weight: bold;
  color: #1a1a1a;
}
.preview-mine-phone {
  font-size: 15px;
  color: #999;
}
.preview-mine-phone-row {
  display: flex;
  align-items: center;
  margin-top: 6px;
  cursor: pointer;
}
.preview-phone-eye-icon {
  width: 22px;
  height: 22px;
  margin-left: 6px;
}
.preview-mine-user-arrow {
  font-size: 24px;
  color: #ccc;
  flex-shrink: 0;
}

.preview-mine-order-section {
  margin: 0 16px 14px;
  padding: 20px;
  background: #fff;
  border-radius: 12px;
}
.preview-mine-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}
.preview-mine-section-title {
  font-size: 18px;
  font-weight: bold;
  color: #1a1a1a;
}
.preview-mine-more {
  font-size: 15px;
  color: #1a1a1a;
}
.preview-mine-order-grid {
  display: flex;
  justify-content: space-around;
}
.preview-mine-order-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.preview-mine-order-icon-wrap {
  width: 54px;
  height: 54px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.preview-mine-order-icon {
  width: 38px;
  height: 38px;
}
.preview-mine-order-text {
  font-size: 14px;
  color: #1a1a1a;
  margin-top: 8px;
}

.preview-mine-menu-section {
  margin: 0 16px;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
}
/* 列表布局（上下排列） */
.preview-mine-menu-section.layout-list .preview-mine-menu-item {
  display: flex;
  align-items: center;
  padding: 18px 20px;
  border-bottom: 0.5px solid #f0f0f0;
}
.preview-mine-menu-section.layout-list .preview-mine-menu-item:last-child {
  border-bottom: none;
}
.preview-mine-menu-section.layout-list .preview-mine-menu-icon {
  width: 38px;
  height: 38px;
  margin-right: 14px;
}
.preview-mine-menu-section.layout-list .preview-mine-menu-name {
  flex: 1;
  font-size: 17px;
}
.preview-mine-menu-section.layout-list .preview-mine-menu-arrow {
  font-size: 22px;
  color: #1a1a1a;
}
/* 网格布局（一排三个） */
.preview-mine-menu-section.layout-grid {
  padding: 8px 0;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
}
.preview-mine-menu-section.layout-grid .preview-mine-menu-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0 12px;
}
.preview-mine-menu-section.layout-grid .preview-mine-menu-icon {
  width: 46px;
  height: 46px;
  margin-bottom: 8px;
}
.preview-mine-menu-section.layout-grid .preview-mine-menu-name {
  font-size: 13px;
}
/* 微信客服图标尺寸微调：聊天气泡 SVG 内容占比偏小，适当调大 */
.preview-mine-menu-icon.icon-wechat-service {
  width: 38px;
  height: 38px;
}
.preview-mine-menu-section.layout-grid .preview-mine-menu-icon.icon-wechat-service {
  width: 46px;
  height: 46px;
}
/* 共用 */
.preview-mine-menu-icon {
  flex-shrink: 0;
}
.preview-mine-menu-name {
  color: #1a1a1a;
}

/* 退出登录 */
.preview-mine-logout-section {
  margin: 20px 16px 30px;
  padding: 0 20px;
  background: #fff;
  border-radius: 12px;
}
.preview-mine-logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 50px;
  font-size: 16px;
  color: #1a1a1a;
  cursor: pointer;
  transition: opacity 0.2s;
}
.preview-mine-logout-btn:active {
  opacity: 0.6;
}
.preview-mine-logout-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
}
.preview-mine-logout-icon :deep(svg) {
  width: 24px;
  height: 24px;
}

/* 注销账号链接 */
.preview-mine-delete-account-link {
  text-align: center;
  font-size: 13px;
  color: #999;
  padding: 12px 0 16px;
  border-top: 0.5px solid #f0f0f0;
  cursor: pointer;
}

/* 个人中心详情页 */
.profile-detail {
  padding: 16px 12px;
}
.profile-user-card {
  display: flex;
  align-items: center;
  padding: 20px;
  background: #fff;
  border-radius: 12px;
  margin-bottom: 16px;
}
.profile-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  flex-shrink: 0;
}
.profile-info {
  margin-left: 14px;
  flex: 1;
}
.profile-name {
  font-size: 18px;
  font-weight: 700;
  color: #1a1a1a;
}
.profile-phone {
  font-size: 14px;
  color: #999;
  margin-top: 4px;
}

/* 导航栏返回按钮 */
.phone-navbar {
  position: relative;
}
.navbar-back {
  position: absolute;
  left: 12px;
  font-size: 28px;
  color: #1a1a1a;
  cursor: pointer;
  line-height: 1;
  font-weight: 300;
  z-index: 1;
}

/* 子页面背景 */
.phone-scroll-detail {
  background: #f5f6f8;
}

/* 详情页通用 */
.detail-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  font-size: 14px;
  color: #999;
}

/* 搜索/订单列表详情（匹配 search.wxml） */
.detail-filter-tabs {
  display: flex;
  background: #fff;
  padding: 0 16px;
}
.detail-filter-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 40px;
  position: relative;
  cursor: pointer;
}
.detail-filter-tab span {
  font-size: 14px;
  color: #8a8a8a;
}
.detail-filter-tab.active span {
  color: #1a1a1a;
  font-weight: 600;
}
.detail-tab-underline {
  position: absolute;
  bottom: 0;
  width: 24px;
  height: 3px;
  border-radius: 2px;
  background: #1296db;
}
.detail-order-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 16px;
}
.detail-order-card {
  background: #fff;
  border-radius: 12px;
  padding: 16px 20px;
  cursor: pointer;
}
.detail-order-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}
.detail-order-no {
  font-size: 13px;
  color: #999;
}
.detail-type-tag {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 500;
  margin-right: 4px;
}
.detail-type-tag.ticket {
  background: transparent;
  border: 1px solid #d0d0d0;
  color: #1a1a1a;
}
.detail-type-tag.cargo {
  background: transparent;
  border: 1px solid #d0d0d0;
  color: #1a1a1a;
}
.detail-order-status {
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  border: 1px solid #d0d0d0;
  background: transparent;
}
.detail-status-0 { color: #ff3b30; border-color: rgba(255, 59, 48, 0.5); }
.detail-status-1 { color: #34c759; border-color: rgba(52, 199, 89, 0.5); }
.detail-status-2 { color: #8a8a8a; }
.detail-status-3 { color: #ff3b30; border-color: rgba(255, 59, 48, 0.5); }
.detail-status-4 { color: #8a8a8a; }
.detail-order-route {
  display: flex;
  align-items: center;
  margin-bottom: 14px;
}
.detail-route-city {
  font-size: 19px;
  font-weight: bold;
  color: #1a1a1a;
}
.detail-route-arrow {
  flex: 1;
  display: flex;
  align-items: center;
  margin: 0 12px;
}
.detail-arrow-line {
  flex: 1;
  height: 1px;
  background: #e0e0e0;
}
.detail-arrow-icon {
  width: 20px;
  height: 20px;
  margin: 0 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.detail-arrow-icon :deep(svg) {
  width: 20px !important;
  height: 20px !important;
}
.detail-order-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.detail-info-time {
  font-size: 14px;
  color: #666;
}
.detail-info-right {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.detail-info-seats {
  font-size: 13px;
  color: #999;
}
.detail-info-price {
  font-size: 20px;
  font-weight: bold;
  color: #1a1a1a;
}
.detail-order-footer {
  display: flex;
  flex-direction: row;
  justify-content: flex-end;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 0.5px solid #f0f0f0;
  gap: 10px;
}

.detail-order-action-secondary {
  padding: 8px 20px;
  border-radius: 22px;
  font-size: 15px;
  font-weight: 500;
  color: #999;
  border: 1px solid #e0e0e0;
  background: transparent;
  cursor: pointer;
  white-space: nowrap;
}
.detail-order-action {
  padding: 8px 24px;
  border-radius: 22px;
  font-size: 15px;
  font-weight: 500;
  box-sizing: border-box;
  border: 1px solid transparent;
  cursor: pointer;
  white-space: nowrap;
}
.detail-action-0 { background: #1a1a1a; color: #fff; }
.detail-action-1 { background: #1a1a1a; color: #fff; }
.detail-action-2 { background: #1a1a1a; color: #fff; }
.detail-action-3 { background: transparent; color: #666; border-color: #d0d0d0; }
.detail-action-4 { background: transparent; color: #666; border-color: #d0d0d0; }

/* 班次详情（匹配 trip-detail.wxml） */
.detail-trip-card {
  background: #fff;
  border-radius: 16px;
  padding: 24px 20px;
  margin: 16px 16px 0;
  box-shadow: 0 2px 16px rgba(0, 0, 0, 0.05);
}
.detail-trip-route {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #f0f0f0;
}
.detail-trip-from, .detail-trip-to {
  font-size: 22px;
  font-weight: bold;
  color: #1a1a1a;
}
.detail-trip-arrow {
  flex: 1;
  display: flex;
  align-items: center;
  margin: 0 12px;
}
.detail-trip-info {
  display: flex;
  flex-wrap: wrap;
}
.detail-info-item {
  width: 50%;
  margin-bottom: 14px;
}
.detail-info-label {
  display: block;
  font-size: 12px;
  color: #999;
  margin-bottom: 6px;
  letter-spacing: 0.5px;
}
.detail-info-value {
  font-size: 16px;
  color: #333;
  font-weight: 500;
}
.detail-info-value.detail-price {
  color: #1a1a1a;
  font-weight: bold;
}
.detail-track-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 18px;
  padding: 12px 0;
  background: #1a1a1a;
  border-radius: 10px;
  font-size: 15px;
  color: #fff;
  font-weight: 500;
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.15);
  cursor: pointer;
}
.detail-track-icon {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.detail-track-icon :deep(svg) {
  width: 26px !important;
  height: 26px !important;
}
.detail-track-icon :deep(svg path) {
  fill: #fff !important;
}
.detail-section {
  background: #fff;
  border-radius: 16px;
  padding: 20px;
  margin: 12px 16px;
  box-shadow: 0 2px 16px rgba(0, 0, 0, 0.05);
}
.detail-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.detail-section-title {
  font-size: 17px;
  font-weight: 600;
  color: #1a1a1a;
}
.detail-title-wrap {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
}
.detail-title-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.detail-title-icon :deep(svg) {
  width: 20px !important;
  height: 20px !important;
}
.detail-add-passenger {
  font-size: 13px;
  color: #1a1a1a;
  padding: 5px 14px;
  background: transparent;
  border: 1px solid #d0d0d0;
  border-radius: 16px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
}
.detail-passenger-item {
  background: #fff;
  border-radius: 12px;
  padding: 14px;
  margin-bottom: 12px;
  border: 1px solid #eee;
}
.detail-passenger-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.detail-p-input {
  background: #fff;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 15px;
  color: #999;
  height: 44px;
  box-sizing: border-box;
  border: 1px solid #eee;
  display: flex;
  align-items: center;
}
.detail-remove-passenger {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: flex-end;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #f0f0f0;
  cursor: pointer;
}
.detail-remove-text {
  font-size: 13px;
  color: #ff3b30;
  font-weight: 500;
}
.detail-saved-passenger-btn {
  text-align: center;
  font-size: 14px;
  color: #1a1a1a;
  padding: 12px 0;
  border: 1px dashed #ccc;
  border-radius: 10px;
  margin-top: 8px;
  background: #fafafa;
  font-weight: 500;
  cursor: pointer;
}
.detail-coupon-select {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  background: #1a1a1a;
  border-radius: 10px;
  gap: 8px;
  cursor: pointer;
}
.detail-coupon-text {
  flex: 1;
  font-size: 15px;
  color: rgba(255,255,255,0.6);
}
.detail-coupon-arrow {
  font-size: 20px;
  color: rgba(255,255,255,0.6);
}
.detail-contact-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.detail-c-input {
  background: #fff;
  border-radius: 10px;
  padding: 12px 16px;
  font-size: 15px;
  color: #999;
  height: 48px;
  box-sizing: border-box;
  border: 1px solid #e5e5e5;
  display: flex;
  align-items: center;
}
.detail-bottom-bar {
  position: sticky;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #fff;
  border-top: 0.5px solid #eee;
  z-index: 100;
  box-shadow: 0 -2px 16px rgba(0, 0, 0, 0.06);
}
.detail-total-price {
  display: flex;
  align-items: baseline;
  flex-shrink: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
}
.detail-price-label {
  font-size: 14px;
  color: #999;
  margin-right: 6px;
}
.detail-price-num {
  font-size: 26px;
  font-weight: bold;
  color: #1a1a1a;
}
.detail-price-count {
  font-size: 14px;
  color: #999;
  margin-left: 6px;
}
.detail-order-btn {
  background: linear-gradient(135deg, #1a1a1a 0%, #3d3d3d 100%);
  color: #fff;
  font-size: 16px;
  font-weight: 500;
  padding: 12px 28px;
  border-radius: 24px;
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.15);
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
  margin-left: 10px;
}

/* 我的页菜单项详情 */
.detail-passenger-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 12px;
  margin-bottom: 10px;
  border: 1px solid #f0f0f0;
}
.detail-passenger-info {
  display: flex;
  flex-direction: column;
}
.detail-passenger-name {
  font-size: 16px;
  font-weight: 500;
  color: #333;
}
.detail-passenger-card-no {
  font-size: 13px;
  color: #999;
  margin-top: 4px;
}
.detail-passenger-arrow {
  font-size: 20px;
  color: #ccc;
}
.detail-trip-manage-card {
  padding: 14px;
  background: #f8f9fa;
  border-radius: 12px;
  margin-bottom: 10px;
  border: 1px solid #f0f0f0;
}
.detail-trip-manage-route {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}
.detail-trip-manage-city {
  font-size: 17px;
  font-weight: bold;
  color: #1a1a1a;
}
.detail-trip-manage-arrow {
  color: #ccc;
  font-size: 16px;
}
.detail-trip-manage-info {
  display: flex;
  justify-content: space-between;
  font-size: 14px;
  color: #666;
}
.detail-contact-card {
  display: flex;
  align-items: center;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 12px;
}
.detail-contact-icon {
  width: 40px;
  height: 40px;
  margin-right: 16px;
  flex-shrink: 0;
}
.detail-contact-info {
  display: flex;
  flex-direction: column;
}
.detail-contact-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}
.detail-contact-desc {
  font-size: 14px;
  color: #999;
  margin-top: 4px;
}
.detail-verify-input {
  display: flex;
  gap: 10px;
  align-items: center;
}
.detail-verify-input .detail-c-input {
  flex: 1;
}
.detail-verify-btn {
  background: #1a1a1a;
  color: #fff;
  font-size: 15px;
  padding: 12px 30px;
  border-radius: 24px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
}

/* 微信客服聊天页（模拟微信开发者工具客服会话） */
.wx-chat-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 560px;
  background: #ededed;
}
.wx-chat-messages {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px 12px;
  overflow-y: auto;
}
.wx-chat-system-msg {
  text-align: center;
}
.wx-chat-system-msg span {
  display: inline-block;
  font-size: 12px;
  color: #999;
  background: rgba(0, 0, 0, 0.05);
  border-radius: 4px;
  padding: 3px 12px;
}
.wx-chat-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.wx-chat-incoming {
  flex-direction: row;
}
.wx-chat-outgoing {
  flex-direction: row-reverse;
}
.wx-chat-avatar {
  width: 38px;
  height: 38px;
  border-radius: 4px;
  flex-shrink: 0;
}
.wx-chat-avatar-cs {
  background: #07c160;
  position: relative;
}
.wx-chat-avatar-cs::after {
  content: '客服';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #fff;
  font-size: 11px;
  white-space: nowrap;
}
.wx-chat-avatar-user {
  background: #f0f0f0;
  position: relative;
}
.wx-chat-avatar-user::after {
  content: '我';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #666;
  font-size: 14px;
}
.wx-chat-bubble {
  max-width: 220px;
  padding: 10px 14px;
  font-size: 15px;
  line-height: 1.5;
  border-radius: 4px;
  word-break: break-word;
}
.wx-chat-bubble-in {
  background: #fff;
  color: #333;
}
.wx-chat-bubble-out {
  background: #95ec69;
  color: #333;
}
.wx-chat-input-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f7f7f7;
  border-top: 0.5px solid #e0e0e0;
  flex-shrink: 0;
}
.wx-chat-input-placeholder {
  flex: 1;
  height: 36px;
  background: #fff;
  border-radius: 4px;
  display: flex;
  align-items: center;
  padding: 0 12px;
  font-size: 14px;
  color: #999;
}
.wx-chat-send-btn {
  background: #07c160;
  color: #fff;
  font-size: 14px;
  padding: 6px 16px;
  border-radius: 4px;
  white-space: nowrap;
}

/* 微信模拟弹窗（电话拨打提示） */
.wx-modal-mask {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
}
.wx-modal-dialog {
  width: 260px;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 40px rgba(0, 0, 0, 0.2);
}
.wx-modal-title {
  text-align: center;
  font-size: 17px;
  font-weight: 600;
  color: #333;
  padding: 20px 20px 8px;
}
.wx-modal-content {
  text-align: center;
  font-size: 15px;
  color: #666;
  line-height: 1.6;
  padding: 8px 20px 24px;
}
.wx-modal-footer {
  display: flex;
  border-top: 0.5px solid #e0e0e0;
}
.wx-modal-btn {
  flex: 1;
  text-align: center;
  font-size: 17px;
  padding: 14px 0;
  cursor: pointer;
  transition: background 0.15s;
}
.wx-modal-btn:hover {
  background: #f5f5f5;
}
.wx-modal-btn-cancel {
  color: #333;
  border-right: 0.5px solid #e0e0e0;
}
.wx-modal-btn-confirm {
  color: #07c160;
  font-weight: 600;
}

/* 行程搜索子页面（1:1还原小程序home.wxss） */

/* 站点筛选（白底圆角卡片） */
.search-station-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 14px 12px 0;
  padding: 12px 16px;
  background: #fff;
  border-radius: 12px;
}
.search-station-box {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  cursor: pointer;
}
.search-station-label {
  font-size: 12px;
  color: #999;
  margin-bottom: 4px;
}
.search-station-name {
  font-size: 16px;
  font-weight: 600;
  color: #1296db;
}
.search-swap {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
}
.search-swap :deep(svg) {
  width: 22px;
  height: 22px;
}

/* 日期栏 */
.search-date-bar {
  display: flex;
  align-items: center;
  margin: 10px 12px 0;
  padding: 8px 16px;
}
.search-date-label {
  font-size: 14px;
  color: #1a1a1a;
  margin-right: 10px;
}
.search-date-value {
  font-size: 16px;
  font-weight: 600;
  color: #1296db;
  cursor: pointer;
}
.search-clear-btn {
  margin-left: auto;
  font-size: 13px;
  color: #999;
  cursor: pointer;
}

/* 搜索框（圆角白底） */
.search-box-round {
  display: flex;
  align-items: center;
  margin: 14px 12px;
  padding: 0 16px;
  height: 50px;
  background: #fff;
  border-radius: 25px;
}
.search-box-icon {
  display: flex;
  align-items: center;
  margin-right: 10px;
  flex-shrink: 0;
}
.search-box-icon :deep(svg) {
  width: 22px;
  height: 22px;
}
.search-box-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 17px;
  color: #333;
  height: 100%;
}
.search-box-input::placeholder {
  color: #999;
}
.search-box-clear {
  color: #999;
  font-size: 16px;
  cursor: pointer;
  flex-shrink: 0;
  padding: 0 4px;
}

/* 车次列表 */
.search-result-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0 12px;
}
.search-result-card {
  background: #fff;
  border-radius: 12px;
  padding: 18px 20px;
  cursor: pointer;
}
.search-result-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}
.search-result-path {
  display: flex;
  align-items: center;
  flex: 1;
}
.search-result-from,
.search-result-to {
  font-size: 19px;
  font-weight: bold;
  color: #1a1a1a;
}
.search-result-arrow {
  display: flex;
  align-items: center;
  margin: 0 10px;
}
.search-result-arrow :deep(svg) {
  width: 20px;
  height: 20px;
}
.search-result-seats {
  font-size: 12px;
  color: #1a1a1a;
  font-weight: 500;
  padding: 2px 10px;
  border: 1px solid #d0d0d0;
  border-radius: 12px;
  background: transparent;
}
.search-result-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.search-result-info {
  display: flex;
  flex-direction: row;
  align-items: center;
  flex-wrap: wrap;
  flex: 1;
  min-width: 0;
}
.search-result-bus :deep(svg) {
  width: 20px;
  height: 20px;
}
.search-result-bus {
  margin-right: 8px;
  flex-shrink: 0;
}
.search-result-time {
  font-size: 16px;
  color: #333;
  font-weight: 500;
}
.search-result-duration {
  font-size: 13px;
  color: #666;
  margin-left: 8px;
  flex-shrink: 0;
  white-space: nowrap;
}

.search-result-arrival {
  font-size: 13px;
  color: #666;
  margin-left: 8px;
  flex-shrink: 0;
  white-space: nowrap;
}
.search-result-action {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.search-result-price {
  font-size: 22px;
  font-weight: bold;
  color: #1a1a1a;
  margin-right: 14px;
}
.search-result-btn {
  background: #1a1a1a;
  color: #fff;
  font-size: 16px;
  font-weight: 500;
  padding: 8px 24px;
  border-radius: 22px;
  white-space: nowrap;
  flex-shrink: 0;
}
.search-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  font-size: 15px;
  color: #999;
}

/* 优惠券领取提示 toast */
.coupon-toast {
  position: absolute;
  bottom: 80px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.75);
  color: #fff;
  font-size: 14px;
  padding: 10px 24px;
  border-radius: 20px;
  z-index: 998;
  white-space: nowrap;
  animation: couponToastIn 0.3s ease;
}
@keyframes couponToastIn {
  from { opacity: 0; transform: translateX(-50%) translateY(10px); }
  to { opacity: 1; transform: translateX(-50%) translateY(0); }
}

/* 优惠券已领取状态 */
.preview-coupon-card {
  cursor: pointer;
  transition: opacity 0.2s;
}
.preview-coupon-card.coupon-claimed {
  opacity: 0.6;
}

/* 惊喜礼包 - 积分卡片 */
.preview-gift-points-card {
  display: flex;
  align-items: center;
  margin: 14px 12px;
  padding: 24px 20px;
  background: linear-gradient(135deg, #1296db, #0d7fb8);
  border-radius: 16px;
}
.preview-gift-icon {
  width: 52px;
  height: 52px;
  flex-shrink: 0;
  margin-right: 18px;
  /* 白色变体在蓝色背景上显示 */
  filter: brightness(0) invert(1);
}
.preview-points-info {
  display: flex;
  flex-direction: column;
}
.preview-points-label {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.85);
}
.preview-points-value {
  font-size: 36px;
  font-weight: bold;
  color: #fff;
  line-height: 1.2;
  margin: 2px 0;
}
.preview-points-desc {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}

/* 积分明细列表 */
.preview-record-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  border-bottom: 0.5px solid #f0f0f0;
}
.preview-record-item:last-child {
  border-bottom: none;
}
.preview-record-left {
  display: flex;
  flex-direction: column;
  flex: 1;
}
.preview-record-remark {
  font-size: 14px;
  color: #333;
}
.preview-record-date {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}
.preview-record-points {
  font-size: 16px;
  font-weight: 600;
}
.preview-record-points.plus { color: #34c759; }
.preview-record-points.minus { color: #ff3b30; }

/* 积分说明 */
.preview-gift-tips {
  margin: 12px;
  padding: 16px 18px;
  background: #fff;
  border-radius: 12px;
}
.preview-tips-title {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: #666;
  margin-bottom: 8px;
}
.preview-tips-text {
  display: block;
  font-size: 13px;
  color: #999;
  line-height: 1.8;
}

/* 惊喜礼包预览 - 标签栏 */
.preview-gift-tabs {
  display: flex;
  flex-direction: row;
  background: #fff;
  height: 40px;
  border-bottom: 1px solid #f0f0f0;
}
.preview-gift-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: #666;
  position: relative;
}
.preview-gift-tab.active {
  color: #1296db;
  font-weight: 600;
}
.preview-gift-tab.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 24px;
  height: 2px;
  background: #1296db;
  border-radius: 1px;
}
.preview-gift-subtabs {
  display: flex;
  flex-direction: row;
  padding: 0 12px;
  height: 32px;
  align-items: center;
  gap: 16px;
}
.preview-gift-subtab {
  font-size: 12px;
  color: #999;
}
.preview-gift-subtab.active {
  color: #1296db;
  font-weight: 600;
}
.preview-gift-divider {
  height: 1px;
  background: #f0f0f0;
  margin: 12px 0 0;
}

/* 礼包页优惠券卡片预览 - 覆盖首页横滑卡片的180px宽度 */
.phone-scroll .preview-coupon-card {
  display: flex;
  flex-direction: row;
  width: calc(100% - 24px);
  height: 90px;
  margin: 0 12px 10px;
  border-radius: 10px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
  position: relative;
  flex-shrink: 0;
}
.phone-scroll .preview-coupon-left {
  width: 100px;
  background: linear-gradient(135deg, #1296db, #0d7fb8);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.phone-scroll .preview-coupon-left.gray {
  background: linear-gradient(135deg, #bbb, #999);
}

/* 优惠券状态标签 */
.phone-scroll .preview-coupon-status {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 10px;
  font-weight: 500;
}
.phone-scroll .preview-coupon-status.usable {
  background: rgba(18, 150, 219, 0.1);
  color: #1296db;
}
.phone-scroll .preview-coupon-status.used {
  background: rgba(153, 153, 153, 0.1);
  color: #999;
}
.phone-scroll .preview-coupon-status.expired {
  background: rgba(153, 153, 153, 0.1);
  color: #999;
}
.phone-scroll .preview-coupon-value {
  font-size: 20px;
  font-weight: bold;
  color: #fff;
}
.phone-scroll .preview-coupon-desc {
  font-size: 11px;
  color: rgba(255,255,255,0.9);
  margin-top: 4px;
}
.phone-scroll .preview-coupon-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 0 56px 0 16px;
  overflow: hidden;
}
.phone-scroll .preview-coupon-label {
  font-size: 12px;
  color: #1296db;
  font-weight: 500;
}
.phone-scroll .preview-coupon-name {
  font-size: 15px;
  color: #333;
  margin-top: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.phone-scroll .preview-coupon-expire {
  font-size: 11px;
  color: #999;
  margin-top: 4px;
}
</style>
