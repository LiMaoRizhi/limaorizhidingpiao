<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <!-- 悬浮按钮 -->
  <div class="ai-trigger" :class="{ 'ai-dark': darkMode }" @click="togglePanel" title="打开数字员工">
    <svg class="ai-trigger-icon ai-cat-sway" viewBox="0 0 800 800"><g transform="translate(143.608259,631.739000) scale(0.056999,-0.056999)" fill="currentColor" stroke="none"><path :d="catPath"/></g></svg>
    <span class="ai-trigger-text">数字员工</span>
  </div>

  <!-- ChatGPT风格左侧边栏 -->
  <transition name="ai-sidebar-slide">
    <div v-if="showSidebar && showPanel" class="ai-sidebar" :class="{ 'ai-dark': darkMode }">
      <div class="ai-sidebar-header">
        <span class="ai-sidebar-title">对话历史</span>
        <span class="ai-sidebar-close" @click="showSidebar = false" title="收起侧边栏">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>
        </span>
      </div>
      <div class="ai-sidebar-newchat" @click="handleNewChat">
        <svg viewBox="0 0 1024 1024" width="14" height="14" fill="currentColor"><path d="M213.333333 896V234.666667h512v405.333333h85.333334V192c0-23.466667-19.2-42.666667-42.666667-42.666667H170.666667c-23.466667 0-42.666667 19.2-42.666667 42.666667v725.333333c0 23.466667 19.2 42.666667 42.666667 42.666667h341.333333v-85.333333H213.333333z m725.333334-85.333333v-85.333334H810.666667v-128h-85.333334v128h-128v85.333334h128v85.333333h85.333334v-85.333333h128z"/></svg>
        <span>新对话</span>
      </div>
      <div class="ai-sidebar-list">
        <div v-if="conversations.length === 0" class="ai-sidebar-empty">暂无对话记录</div>
        <div v-for="conv in conversations" :key="conv.id" class="ai-sidebar-item" @click="loadConversation(conv.id)">
          <svg class="ai-sidebar-item-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
          <div class="ai-sidebar-item-info">
            <span class="ai-sidebar-item-title">{{ conv.title }}</span>
            <span class="ai-sidebar-item-time">{{ conv.time }}</span>
          </div>
          <span class="ai-sidebar-item-delete" @click.stop="deleteConversation(conv.id)" title="删除对话">
            <svg viewBox="0 0 1024 1024" width="13" height="13" fill="currentColor"><path d="M202.666667 256h-42.666667a32 32 0 0 1 0-64h704a32 32 0 0 1 0 64H266.666667v565.333333a53.333333 53.333333 0 0 0 53.333333 53.333334h384a53.333333 53.333333 0 0 0 53.333333-53.333334V352a32 32 0 0 1 64 0v469.333333c0 64.8-52.533333 117.333333-117.333333 117.333334H320c-64.8 0-117.333333-52.533333-117.333333-117.333334V256z m224-106.666667a32 32 0 0 1 0-64h170.666666a32 32 0 0 1 0 64H426.666667z m-32 288a32 32 0 0 1 64 0v256a32 32 0 0 1-64 0V437.333333z m170.666666 0a32 32 0 0 1 64 0v256a32 32 0 0 1-64 0V437.333333z" fill="currentColor"/></svg>
          </span>
        </div>
      </div>
    </div>
  </transition>

  <!-- 聊天面板 -->
  <transition name="ai-slide">
    <div v-if="showPanel" class="ai-panel" :class="{ 'ai-dark': darkMode }">
      <!-- 面板头部 -->
      <div class="ai-header">
        <div class="ai-header-left">
          <!-- 侧边栏开关 -->
          <span class="ai-header-btn" :class="{ active: showSidebar }" @click="toggleSidebar" :title="showSidebar ? '收起侧边栏' : '打开侧边栏'">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="14" rx="7"/><line :x1="showSidebar ? 15 : 9" y1="5" :x2="showSidebar ? 15 : 9" y2="19"/></svg>
          </span>
        <div class="ai-header-info">
          <svg class="ai-header-avatar ai-cat-sway" viewBox="0 0 800 800"><g transform="translate(143.608259,631.739000) scale(0.056999,-0.056999)" fill="currentColor" stroke="none"><path :d="catPath"/></g></svg>
          <div class="ai-header-meta">
            <span class="ai-header-name">数字员工</span>
            <span class="ai-header-status">{{ isReasoning ? '正在思考...' : isThinking ? '正在回答...' : '在线' }}</span>
          </div>
        </div>
        </div>
        <div class="ai-header-actions">
          <!-- 新对话 -->
          <span class="ai-header-btn" @click="handleNewChat" title="新对话">
            <svg viewBox="0 0 1024 1024" width="16" height="16"><path d="M333.141333 987.477333a52.736 52.736 0 0 1-18.688-3.754666 47.274667 47.274667 0 0 1-16.042666-10.496 44.544 44.544 0 0 1-10.581334-15.36 45.738667 45.738667 0 0 1-4.096-18.346667l-1.024-96.938667a256.853333 256.853333 0 0 1-82.346666-27.392 239.786667 239.786667 0 0 1-36.181334-24.149333 218.88 218.88 0 0 1-31.488-29.610667A240.384 240.384 0 0 1 76.8 646.826667a237.397333 237.397333 0 0 1-3.584-42.410667V346.197333c0-15.616 1.450667-30.976 4.352-46.336a277.333333 277.333333 0 0 1 13.994667-45.056A242.432 242.432 0 0 1 269.568 112.64a230.4 230.4 0 0 1 47.445333-4.778667H486.4a42.154667 42.154667 0 1 1 0 84.48H317.013333c-10.496 0-20.736 0.768-30.72 2.901334-10.24 1.877333-20.053333 4.778667-29.610666 8.874666-9.642667 3.669333-18.773333 8.533333-27.306667 14.165334s-16.554667 12.032-23.893333 19.114666a171.52 171.52 0 0 0-19.626667 23.296 170.922667 170.922667 0 0 0-14.762667 26.709334 155.306667 155.306667 0 0 0-11.690666 58.88v258.218666c0 10.24 0.853333 20.138667 2.986666 30.378667 2.133333 9.813333 5.12 19.797333 9.130667 29.269333 7.936 19.2 19.712 36.437333 34.730667 50.773334 14.848 14.592 32.512 26.026667 51.882666 33.706666 9.557333 4.010667 19.797333 6.912 30.037334 9.130667 10.24 1.877333 20.48 2.901333 31.146666 2.901333a48.896 48.896 0 0 1 45.653334 29.696c2.56 5.802667 3.669333 11.946667 4.010666 18.176l0.341334 59.733334 108.970666-77.909334c27.904-19.797333 58.88-29.696 93.269334-29.696H706.56a165.546667 165.546667 0 0 0 60.330667-11.690666c9.472-4.010667 18.602667-8.789333 27.392-14.250667 8.533333-5.546667 16.554667-12.117333 23.893333-19.029333 7.253333-7.253333 13.824-14.933333 19.626667-23.381334 11.605333-16.725333 19.626667-35.584 23.466666-55.552 2.133333-9.984 2.901333-19.797333 2.901334-30.037333V473.173333a40.789333 40.789333 0 0 1 12.8-29.610666 44.544 44.544 0 0 1 47.189333-9.130667c5.12 1.877333 9.813333 5.12 13.909333 9.130667a40.874667 40.874667 0 0 1 12.8 29.610666V606.72c-0.085333 47.274667-14.506667 93.44-41.386666 132.437333a220.842667 220.842667 0 0 1-30.293334 36.096 262.144 262.144 0 0 1-36.949333 29.696 251.733333 251.733333 0 0 1-135.68 40.277334H599.722667c-33.962667 0-65.109333 9.813333-92.842667 29.610666L362.410667 977.92a49.92 49.92 0 0 1-29.269334 9.557333z" fill="currentColor"/><path d="M902.570667 188.16H665.173333a42.154667 42.154667 0 1 0 0 84.309333h237.397334a42.154667 42.154667 0 0 0 0-84.309333z" fill="currentColor"/><path d="M826.965333 116.309333a43.178667 43.178667 0 0 0-86.272 0v228.096a43.178667 43.178667 0 1 0 86.357334 0V116.309333z" fill="currentColor"/></svg>
          </span>
          <!-- 清空对话 -->
          <span class="ai-header-btn ai-header-btn-danger" @click="handleClearMessages" title="清空对话">
            <svg viewBox="0 0 1024 1024" width="16" height="16"><path d="M202.666667 256h-42.666667a32 32 0 0 1 0-64h704a32 32 0 0 1 0 64H266.666667v565.333333a53.333333 53.333333 0 0 0 53.333333 53.333334h384a53.333333 53.333333 0 0 0 53.333333-53.333334V352a32 32 0 0 1 64 0v469.333333c0 64.8-52.533333 117.333333-117.333333 117.333334H320c-64.8 0-117.333333-52.533333-117.333333-117.333334V256z m224-106.666667a32 32 0 0 1 0-64h170.666666a32 32 0 0 1 0 64H426.666667z m-32 288a32 32 0 0 1 64 0v256a32 32 0 0 1-64 0V437.333333z m170.666666 0a32 32 0 0 1 64 0v256a32 32 0 0 1-64 0V437.333333z" fill="currentColor"/></svg>
          </span>
          <!-- 设置（主题切换） -->
          <div ref="settingsRef" style="position: relative;">
            <span class="ai-header-btn" @click="showSettings = !showSettings" title="设置">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
            </span>
            <transition name="ai-dropdown">
              <div v-if="showSettings" class="ai-settings-dropdown">
                <div class="ai-settings-label">主题模式</div>
                <div class="ai-settings-item" :class="{ active: !darkMode }" @click="toggleTheme(false)">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
                  <span>浅色模式</span>
                  <svg v-if="!darkMode" class="ai-settings-check" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                </div>
                <div class="ai-settings-item" :class="{ active: darkMode }" @click="toggleTheme(true)">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
                  <span>深色模式</span>
                  <svg v-if="darkMode" class="ai-settings-check" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                </div>
              </div>
            </transition>
          </div>
          <!-- 收起面板 -->
          <span class="ai-header-btn" @click="showPanel = false" title="收起">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6"/></svg>
          </span>
        </div>
      </div>

      <!-- 模型选择 -->
      <div class="ai-model-bar" v-if="totalModelCount > 0" ref="modelBarRef">
        <div class="ai-model-selector" @click="totalModelCount > 1 && (showModelDropdown = !showModelDropdown)">
          <!-- AI模型切换图标 -->
          <svg class="ai-model-switch-icon" viewBox="0 0 1024 1024" width="13" height="13" fill="currentColor"><path d="M810.333408 946.516895h-37.867362a25.586056 25.586056 0 1 1 0-50.489816h37.867362a23.880319 23.880319 0 0 0 25.586056-25.586056V176.717769a127.930278 127.930278 0 0 0-126.224541-126.224541 124.945238 124.945238 0 0 0-126.224541 126.224541v100.979633H53.412597V176.717769A175.605628 175.605628 0 0 1 230.126954 0.003411h479.567969a179.102389 179.102389 0 0 1 176.714357 176.714358v694.064401a74.370135 74.370135 0 0 1-75.734724 75.734725zM103.646553 227.378159h429.078152v-50.489816a175.264481 175.264481 0 0 1 52.963135-126.224541H229.444659a124.945238 124.945238 0 0 0-126.224541 126.224541z"/></svg>
          <span class="ai-model-current">{{ currentModelName }}</span>
          <svg v-if="totalModelCount > 1" class="ai-model-arrow" :class="{ rotated: showModelDropdown }" viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6"/></svg>
        </div>
        <transition name="ai-dropdown">
          <div v-if="showModelDropdown" class="ai-model-dropdown">
            <template v-for="group in allGroupedModels" :key="group.group">
              <div class="ai-model-group-label">{{ group.groupName }}</div>
              <div v-for="m in group.models" :key="m.id" class="ai-model-item" :class="{ active: m.id === currentModelId }" @click="onSwitchModel(m.id)">
                <span class="ai-model-item-name">{{ m.name }}</span>
                <span v-if="m.tag" class="ai-model-item-tag" :class="'ai-tag-' + (m.tag_type || 'info')">{{ m.tag }}</span>
                <svg v-if="m.icon === 'auto'" class="ai-model-icon" viewBox="0 0 1024 1024" width="13" height="13" fill="currentColor"><path d="M512 64L576 448L960 512L576 576L512 960L448 576L64 512L448 448Z"/></svg><svg v-else-if="m.icon === 'brain'" class="ai-model-icon" viewBox="0 0 1024 1024" width="13" height="13" fill="currentColor"><path d="M298.666667 256c0 10.112 1.706667 19.712 4.906666 28.586667a42.666667 42.666667 0 0 1-42.666666 56.874666 85.333333 85.333333 0 0 0-68.565334 142.08 42.666667 42.666667 0 0 1 0.042667 56.917334 85.376 85.376 0 0 0 36.864 137.941333 42.666667 42.666667 0 0 1 28.586667 48.426667 106.666667 106.666667 0 0 0 210.432 35.157333c0.256-1.834667 0.597333-3.541333 1.066666-5.248V256a85.333333 85.333333 0 1 0-170.666666 0z m256 500.736c0.426667 1.706667 0.853333 3.413333 1.066666 5.205333a106.666667 106.666667 0 1 0 210.432-35.114666 42.666667 42.666667 0 0 1 28.586667-48.426667 85.376 85.376 0 0 0 36.864-137.941333 42.666667 42.666667 0 0 1 0-56.917334 85.333333 85.333333 0 0 0-68.565333-142.08 42.666667 42.666667 0 0 1-42.624-56.874666A85.333333 85.333333 0 1 0 554.666667 256v500.736zM384 85.333333a170.24 170.24 0 0 1 128 57.770667 170.666667 170.666667 0 0 1 298.581333 118.229333A170.752 170.752 0 0 1 915.84 512 170.538667 170.538667 0 0 1 853.333333 745.173333v1.493334a192 192 0 0 1-341.333333 120.661333A192 192 0 0 1 170.666667 746.666667v-1.493334A170.538667 170.538667 0 0 1 108.16 512a170.752 170.752 0 0 1 105.258667-250.624A170.666667 170.666667 0 0 1 384 85.333333z"/></svg><svg v-else-if="m.icon === 'vision'" class="ai-model-icon" viewBox="0 0 1117 1024" width="13" height="13" fill="currentColor"><path d="M1108.945455 470.109091C1099.636364 453.818182 915.781818 93.090909 559.709091 93.090909S19.781818 453.818182 10.472727 470.109091c-13.963636 25.6-13.963636 58.181818 0 83.781818 9.309091 16.290909 193.163636 377.018182 549.236364 377.018182s539.927273-360.727273 549.236364-377.018182c11.636364-25.6 11.636364-58.181818 0-83.781818zM559.709091 837.818182C257.163636 837.818182 94.254545 512 94.254545 512S257.163636 186.181818 559.709091 186.181818s465.454545 325.818182 465.454545 325.818182-162.909091 325.818182-465.454545 325.818182z"/><path d="M559.709091 325.818182c-102.4 0-186.181818 83.781818-186.181818 186.181818s83.781818 186.181818 186.181818 186.181818 186.181818-83.781818 186.181818-186.181818-83.781818-186.181818-186.181818-186.181818z m0 279.272727c-51.2 0-93.090909-41.890909-93.090909-93.090909s41.890909-93.090909 93.090909-93.090909 93.090909 41.890909 93.090909 93.090909-41.890909 93.090909-93.090909 93.090909z"/></svg><svg v-else-if="m.icon === 'image'" class="ai-model-icon" viewBox="0 0 1024 1024" width="13" height="13" fill="currentColor"><path d="M909.9 66.8H114.1C51.2 66.8 0 117.4 0 179.7v664.6c0 62.3 51.2 112.9 114.1 112.9h795.7c62.9 0 114.1-50.7 114.1-112.9V179.7c0.1-62.3-51.1-112.9-114-112.9zM44.5 179.7c0-37.7 31.2-68.4 69.6-68.4h795.7c38.4 0 69.6 30.7 69.6 68.4v403.8c-56.2-39.4-164.7-104.4-268-104.4-84.4 0-176.1 72.1-264.8 141.8-81.4 64-158.3 124.4-225.1 124.4-53.8 0-137.5-37.9-177-62.7V179.7z m935 664.5c0 37.8-31.2 68.5-69.6 68.5H114.1c-38.4 0-69.6-30.7-69.6-68.4V733.9c47.9 25.8 121.6 55.8 177 55.8 82.3 0 165-65 252.7-133.9 82.7-65 168.3-132.3 237.3-132.3 104.8 0 227.8 82.7 268 114.7v206z"/><path d="M475.2 384c0-85.9-69.9-155.8-155.8-155.8S163.6 298.1 163.6 384s69.9 155.8 155.8 155.8S475.2 469.9 475.2 384z m-267.1-0.1c0-61.3 50-111.3 111.3-111.3 61.4 0 111.3 49.9 111.3 111.3s-49.9 111.3-111.3 111.3-111.3-49.9-111.3-111.3z"/></svg>
              </div>
            </template>
          </div>
        </transition>
      </div>

      <!-- 消息区域 -->
      <div ref="messageArea" class="ai-messages" @click="handleContentClick">
        <div v-if="messages.length === 0" class="ai-welcome">
          <svg class="ai-welcome-icon ai-cat-sway" viewBox="0 0 800 800"><g transform="translate(143.608259,631.739000) scale(0.056999,-0.056999)" fill="currentColor" stroke="none"><path :d="catPath"/></g></svg>
          <p>你好！我是数字员工，有什么可以帮你的吗？</p>
          <p class="ai-welcome-hint">你可以问我关于线路定价、班次管理、订单退票规则等问题</p>
        </div>
        <div
          v-for="(msg, idx) in messages"
          :key="idx"
          class="ai-msg"
          :class="msg.role === 'user' ? 'ai-msg-user' : 'ai-msg-ai'"
        >
          <div class="ai-msg-bubble" v-if="msg.role === 'user'">
            <div v-if="msg.images && msg.images.length" class="ai-msg-images">
              <img v-for="(img, i) in msg.images" :key="i" :src="img" class="ai-msg-image" />
            </div>
            <span v-if="msg.content">{{ msg.content }}</span>
          </div>
          <div class="ai-msg-bubble ai-msg-ai-bubble" v-else>
            <!-- 图片生成中：相片框 + 水波纹 + 黑白主题白光 -->
            <div v-if="msg.isGeneratingImage" class="ai-photo-frame">
              <div class="ai-photo-frame-inner">
                <div class="ai-photo-glow"></div>
                <div class="ai-ripple"></div>
                <div class="ai-ripple ai-ripple-delay-1"></div>
                <div class="ai-ripple ai-ripple-delay-2"></div>
                <span class="ai-photo-frame-text">正在生成图片...</span>
              </div>
            </div>
            <!-- AI 生成的图片（点击放大、可下载） -->
            <div v-if="msg.generatedImage" class="ai-generated-image-wrap">
              <img :src="msg.generatedImage" class="ai-generated-image" @click="previewImage(msg.generatedImage!)" />
              <div class="ai-image-actions">
                <button class="ai-image-action-btn" @click="previewImage(msg.generatedImage!)" title="放大查看">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35M11 8v6M8 11h6"/></svg>
                  <span>放大</span>
                </button>
                <button class="ai-image-action-btn" @click="downloadImage(msg.generatedImage!)" title="下载图片">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
                  <span>下载</span>
                </button>
              </div>
            </div>
            <!-- 思考过程（DeepSeek-R1等推理模型返回） -->
            <div v-if="msg.reasoning" class="ai-reasoning">
              <div class="ai-reasoning-header" @click="toggleReasoning(idx)">
                <svg class="ai-reasoning-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18h6M10 22h4M12 2a7 7 0 0 0-4 12.7V17h8v-2.3A7 7 0 0 0 12 2z"/></svg>
                <span>思考过程</span>
                <span class="ai-reasoning-toggle">{{ expandedReasoning.includes(idx) ? '收起' : '展开' }}</span>
              </div>
              <div v-if="expandedReasoning.includes(idx)" class="ai-reasoning-body" v-html="formatContent(msg.reasoning)"></div>
            </div>
            <span v-if="msg.content === '' && !msg.reasoning && idx === messages.length - 1 && isThinking" class="ai-typing">
              <span class="ai-dot"></span>
              <span class="ai-dot"></span>
              <span class="ai-dot"></span>
            </span>
            <div v-else class="ai-markdown" v-html="formatContent(msg.content)"></div>
          </div>
        </div>
      </div>

      <!-- 输入区域 -->
      <div class="ai-input-area">
        <!-- 图片生成模式切换 -->
        <div class="ai-mode-bar">
          <button class="ai-mode-btn" :class="{ active: imageMode }" @click="imageMode = !imageMode" title="切换图片生成模式">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="9" cy="9" r="2"/><path d="M21 15l-5-5L5 21"/></svg>
            <span>图片生成</span>
          </button>
        </div>
        <!-- 待发送图片预览 -->
        <div v-if="pendingImages.length > 0" class="ai-image-preview-row">
          <div v-for="(img, idx) in pendingImages" :key="idx" class="ai-image-preview-item">
            <img :src="img" class="ai-preview-thumb" />
            <span class="ai-image-remove" @click="removeImage(idx)">×</span>
          </div>
        </div>
        <div class="ai-input-row">
          <!-- 图片上传按钮（始终显示，发送时按模型能力拦截） -->
          <label class="ai-image-btn" title="上传图片">
            <input ref="fileInput" type="file" accept="image/*" multiple @change="onImageSelect" style="display:none" />
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="M21 15l-5-5L5 21"/></svg>
          </label>
          <!-- AI回复时输入框蓝色行走光效 -->
          <div class="ai-input-wrap" :class="{ 'is-thinking': isThinking }">
            <textarea
              ref="inputBox"
              v-model="inputText"
              class="ai-input"
              :placeholder="isThinking ? '数字员工正在回复中...' : '输入消息，或点击左侧上传图片...'"
              @keydown.enter.exact.prevent="sendMessage"
              rows="2"
            ></textarea>
          </div>
          <!-- GPT风格方形图标发送按钮 -->
          <button
            class="ai-send-icon-btn"
            :class="{ 'is-stop': isThinking, 'is-disabled': !isThinking && !canSend }"
            :disabled="!isThinking && !canSend"
            @click="isThinking ? stopGeneration() : sendMessage()"
            :title="isThinking ? '停止生成' : '发送'"
          >
            <svg v-if="isThinking" viewBox="0 0 24 24" width="18" height="18" fill="currentColor"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>
            <svg v-else viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
          </button>
        </div>
      </div>
    </div>
  </transition>

  <!-- 图片预览遮罩：点击放大、关闭、下载 -->
  <transition name="ai-fade">
    <div v-if="previewImageSrc" class="ai-image-overlay" @click="previewImageSrc = ''">
      <button class="ai-overlay-close-btn" @click.stop="previewImageSrc = ''" title="关闭">
        <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18M6 6l12 12"/></svg>
      </button>
      <img :src="previewImageSrc" class="ai-overlay-image" @click.stop />
      <div class="ai-overlay-toolbar" @click.stop>
        <button class="ai-overlay-download-btn" @click="downloadImage(previewImageSrc)">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
          <span>下载图片</span>
        </button>
      </div>
    </div>
  </transition>

  <!-- 遮罩 -->
  <div v-if="showPanel" class="ai-mask" :class="{ 'ai-dark': darkMode }" @click="showPanel = false"></div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { aiApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { authFetch } from '@/utils/request'
// 狸猫Logo SVG path data
const catPath = 'M4004 7731 c-87 -53 -237 -402 -328 -765 -33 -129 -38 -140 -84 -194 -144 -168 -247 -377 -307 -622 -27 -112 -30 -118 -95 -197 -36 -46 -82 -116 -102 -155 -37 -76 -78 -196 -78 -229 0 -11 -6 -22 -14 -25 -8 -3 -140 4 -295 16 -207 16 -285 18 -300 10 -26 -13 -37 -54 -25 -89 12 -33 20 -34 354 -61 140 -11 257 -24 261 -28 11 -12 17 -102 7 -108 -6 -4 -116 -35 -244 -69 -129 -35 -246 -70 -259 -79 -17 -11 -25 -26 -25 -44 0 -33 8 -45 38 -61 23 -12 34 -10 429 95 50 13 97 24 105 24 7 -1 24 -23 36 -51 60 -126 204 -282 374 -404 l87 -62 -26 -24 c-49 -46 -185 -215 -252 -314 -190 -280 -304 -530 -379 -830 -70 -278 -89 -755 -38 -944 19 -70 58 -148 100 -201 41 -52 141 -123 207 -146 27 -10 49 -21 49 -24 0 -13 -110 -137 -149 -168 -60 -48 -156 -90 -238 -103 -61 -10 -99 -9 -228 6 -94 11 -179 15 -213 11 -250 -29 -344 -294 -173 -488 47 -53 158 -114 271 -150 67 -20 93 -23 245 -22 156 0 178 3 260 28 165 50 307 127 410 221 27 25 52 45 55 45 8 0 61 -509 70 -677 9 -171 26 -215 106 -276 179 -139 599 -230 844 -183 145 27 252 94 303 189 18 35 22 58 22 135 0 54 4 92 10 92 6 0 48 -7 94 -15 46 -8 150 -20 231 -27 226 -18 384 6 486 75 133 89 180 257 110 394 -44 87 -122 135 -264 163 -34 7 -62 16 -62 19 0 4 13 92 29 197 30 188 57 391 86 639 9 72 18 144 21 162 l6 31 56 -26 c79 -37 211 -38 290 -3 72 32 151 119 188 207 27 65 29 76 29 214 -1 155 -16 274 -65 491 -57 256 -140 482 -297 814 -51 110 -95 203 -96 207 -1 4 48 25 108 46 239 84 326 133 438 245 l89 87 207 -125 c131 -80 216 -125 234 -125 59 0 88 72 45 109 -13 11 -93 62 -178 114 -85 51 -174 106 -197 121 l-42 28 15 37 c8 20 19 40 24 45 6 6 90 -6 222 -32 117 -24 220 -42 229 -42 20 0 44 28 52 58 3 15 -2 31 -16 47 -18 20 -51 30 -197 60 -96 20 -191 40 -211 45 l-35 10 0 142 c0 130 -3 150 -28 223 -15 44 -38 99 -52 122 -24 41 -24 42 -13 172 21 232 -18 479 -101 646 l-40 81 19 115 c30 175 53 428 55 594 1 140 0 152 -21 181 -49 68 -125 69 -261 4 -123 -59 -267 -156 -462 -314 l-56 -44 -54 20 c-226 84 -408 119 -640 122 l-165 3 -111 155 c-293 409 -387 492 -490 429z m143 -2116 c16 -11 24 -33 33 -86 14 -83 7 -81 140 -25 94 39 117 36 131 -17 11 -40 -11 -60 -109 -101 -133 -55 -128 -48 -103 -138 12 -41 19 -83 15 -91 -14 -37 -110 -61 -132 -34 -6 7 -16 35 -22 62 -7 28 -14 56 -16 63 -4 15 -67 -14 -121 -56 -18 -13 -40 -22 -48 -18 -35 13 -47 103 -17 128 9 9 50 32 90 52 l73 37 -2 106 c-2 93 0 107 16 119 24 18 45 18 72 -1z'
import { marked } from 'marked'

const router = useRouter()
const authStore = useAuthStore()

// Markdown渲染 表格代码块列表都支持
marked.setOptions({
  breaks: true,
  gfm: true,
})

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  reasoning?: string // 思考过程（仅推理模型返回）
  images?: string[] // base64图片数组（多模态）
  generatedImage?: string // AI生成的图片（base64 data URI）
  isGeneratingImage?: boolean // 是否正在生成图片
  isImageGen?: boolean // 图片生成消息（不进入对话上下文，避免污染后续对话）
}

const showPanel = ref(false)
const inputText = ref('')
const isThinking = ref(false)
const imageMode = ref(false) // 图片生成模式
const isReasoning = ref(false) // 正在接收思考过程
const messages = ref<ChatMessage[]>([])
const messageArea = ref<HTMLElement>()
const inputBox = ref<HTMLTextAreaElement>()
const fileInput = ref<HTMLInputElement>()
const modelBarRef = ref<HTMLElement>()
const enabled = ref(false)
const abortController = ref<AbortController | null>(null) // 流式请求中断控制器
const pendingImages = ref<string[]>([]) // 待发送的base64图片
const previewImageSrc = ref('') // 图片预览遮罩的图片src

// 暗色模式 setup顶层加载 不然onMounted首次渲染会闪
const THEME_KEY = 'limaorizhi_ai_dark_mode'
const darkMode = ref(localStorage.getItem(THEME_KEY) === 'true')
const showSettings = ref(false)
const settingsRef = ref<HTMLElement>()

// ChatGPT风格侧边栏
const showSidebar = ref(false)
const conversations = ref<{ id: string; title: string; time: string; messages: ChatMessage[] }[]>([])
const CONV_STORAGE_KEY = 'limaorizhi_ai_conversations'
const currentConversationId = ref('') // 当前加载的对话ID，用于持续保存

// 模型选择
const providers = ref<any[]>([])
const currentProvider = ref('nvidia')
const currentModelId = ref('')
const showModelDropdown = ref(false)
// 按服务商分组 只显示有API Key的
// 图片生成模型单独一组，走独立接口，不走聊天模型切换
const allGroupedModels = computed(() => {
  const groups: Record<string, { group: string; groupName: string; models: any[] }> = {}
  for (const p of providers.value) {
    if (p.has_key && p.models && p.models.length > 0) {
      const g = p.group || 'other'
      const gname = p.group_name || p.name
      if (!groups[g]) {
        groups[g] = { group: g, groupName: gname, models: [] }
      }
      // 聊天模型和图片生成模型分开
      const chatModels = p.models.filter((m: any) => m.icon !== 'image')
      groups[g].models.push(...chatModels)
      // 图片生成模型放入独立分组
      const imageModels = p.models.filter((m: any) => m.icon === 'image')
      if (imageModels.length > 0) {
        if (!groups['image']) {
          groups['image'] = { group: 'image', groupName: '图片生成', models: [] }
        }
        groups['image'].models.push(...imageModels)
      }
    }
  }
  return Object.values(groups)
})
const totalModelCount = computed(() => allGroupedModels.value.reduce((sum: number, g: any) => sum + g.models.length, 0))
const currentModelName = computed(() => {
  for (const p of providers.value) {
    const m = p.models?.find((item: any) => item.id === currentModelId.value)
    if (m) return m.name
  }
  return currentModelId.value || ''
})
// 当前模型支不支持图片识别
const currentModelSupportsVision = computed(() => {
  for (const p of providers.value) {
    const m = p.models?.find((item: any) => item.id === currentModelId.value)
    if (m) return !!m.supports_vision
  }
  return false
})
// 当前模型图标类型 判断是不是图片生成模型
const currentModelIcon = computed(() => {
  for (const p of providers.value) {
    const m = p.models?.find((item: any) => item.id === currentModelId.value)
    if (m) return m.icon || ''
  }
  return ''
})
// 能不能发 有文字或有图片就行
const canSend = computed(() => inputText.value.trim() !== '' || pendingImages.value.length > 0)
const loadModels = async () => {
  try {
    const res: any = await aiApi.getModels()
    providers.value = res.data?.providers || []
    currentProvider.value = res.data?.current_provider || 'nvidia'
    currentModelId.value = res.data?.current_model || ''
  } catch {
    // ignore
  }
}
const onSwitchModel = async (modelId: string) => {
  showModelDropdown.value = false
  if (modelId === currentModelId.value) return

  let modelIcon = ''
  let modelName = ''
  for (const p of providers.value) {
    const m = p.models?.find((item: any) => item.id === modelId)
    if (m) { modelIcon = m.icon || ''; modelName = m.name; break }
  }

  // 图片生成模型走独立接口，不调后端 switchModel（后端会拒绝）
  if (modelIcon === 'image') {
    currentModelId.value = modelId
    imageMode.value = true
    ElMessage.success(`已切换到图片生成模型「${modelName}」，输入文字描述即可生成图片`)
    return
  }

  try {
    const res: any = await aiApi.switchModel(modelId)
    currentModelId.value = modelId
    if (res.data?.provider) {
      currentProvider.value = res.data.provider
    }
    // 切换到聊天模型时重置 imageMode，避免误触发图片生成
    imageMode.value = false
    if (modelId === 'auto') {
      ElMessage.success('已切换到狸猫员工，自动选择最优可用模型')
    } else if (res.data?.has_key === false) {
      ElMessage.warning('该服务商尚未配置API Key，请在系统配置中填写')
    } else {
      ElMessage.success('模型已切换')
    }
  } catch {
    // error handled by interceptor
  }
}

// 本地存 聊天记录放localStorage 省服务器资源
const STORAGE_KEY = 'limaorizhi_ai_chat_history'
const MAX_MESSAGES = 66 // 最多保留最近66条
const expandedReasoning = ref<number[]>([]) // 展开的思考过程索引

const saveMessages = () => {
  // 构造消息列表，withImages 控制是否携带图片(base64较大，配额超限时降级)
  const buildMessages = (count: number, withImages: boolean) => {
    return messages.value.slice(-count).map(m => {
      const item: any = {
        role: m.role,
        content: m.content,
        reasoning: m.reasoning || ''
      }
      if (withImages && m.images && m.images.length > 0) {
        item.images = m.images
      }
      if (withImages && m.generatedImage) {
        item.generatedImage = m.generatedImage
      }
      if (m.isImageGen) {
        item.isImageGen = true
      }
      return item
    })
  }

  const trySave = (msgs: any[]): boolean => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(msgs))
      return true
    } catch (e) {
      console.warn('[AI] 保存聊天记录失败:', e)
      return false
    }
  }

  // 1. 优先保存完整数据（含图片，最多66条）
  if (trySave(buildMessages(MAX_MESSAGES, true))) return
  // 2. 配额超限：减少到40条（含图片）
  if (trySave(buildMessages(40, true))) return
  // 3. 仍然超限：减少到20条（含图片）
  if (trySave(buildMessages(20, true))) return
  // 4. 最终降级：只保存纯文本（不含图片），确保文字记录不丢失
  if (!trySave(buildMessages(MAX_MESSAGES, false))) {
    ElMessage.warning('对话记录较多，本地存储空间不足，建议清空旧对话后继续')
  }

  // 同步更新侧边栏中的当前对话（支持返回继续聊）
  if (currentConversationId.value) {
    const conv = conversations.value.find(c => c.id === currentConversationId.value)
    if (conv) {
      conv.messages = JSON.parse(JSON.stringify(messages.value))
      const firstUserMsg = messages.value.find(m => m.role === 'user')
      if (firstUserMsg) conv.title = firstUserMsg.content.slice(0, 30)
      saveConversations()
    }
  }
}

const loadMessages = () => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const msgs = JSON.parse(saved) as ChatMessage[]
      // 清理未完成的图片生成状态（页面刷新后无法继续生成）
      messages.value = msgs.filter(m => !m.isGeneratingImage)
    }
  } catch {
    // 解析失败，忽略
  }
}

// 模型下拉菜单 / 设置下拉菜单外部点击关闭
const handleDocumentClick = (e: MouseEvent) => {
  if (showModelDropdown.value && modelBarRef.value) {
    if (!modelBarRef.value.contains(e.target as Node)) {
      showModelDropdown.value = false
    }
  }
  if (showSettings.value && settingsRef.value) {
    if (!settingsRef.value.contains(e.target as Node)) {
      showSettings.value = false
    }
  }
}

// 主题切换：持久化到 localStorage，刷新后保持
const toggleTheme = (dark: boolean) => {
  darkMode.value = dark
  localStorage.setItem(THEME_KEY, String(dark))
  showSettings.value = false
}


// 挂载时读取 AI 开关配置
onMounted(async () => {
  document.addEventListener('click', handleDocumentClick)
  loadMessages()
  loadConversations()
  loadModels()
  try {
    const res: any = await aiApi.getConfig()
    enabled.value = res.data?.ai_employee_enabled ?? false
  } catch {
    enabled.value = false
  }
})

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick)
  // 中断正在进行的 SSE 流式请求，避免组件卸载后继续更新响应式数据
  abortController.value?.abort()
  // 清理 rAF，避免卸载后操作已移除的 DOM
  if (scrollRafId !== null) {
    cancelAnimationFrame(scrollRafId)
    scrollRafId = null
  }
})

const togglePanel = () => {
  if (!enabled.value) {
    ElMessage.warning('AI数字员工未启用，请在系统配置中开启')
    return
  }
  showPanel.value = !showPanel.value
  if (showPanel.value) {
    // 展开时滚动到底部，确保历史消息显示在最新位置
    nextTick(() => {
      inputBox.value?.focus()
      scrollToBottom()
    })
    // 延迟再次滚动，确保面板展开动画完成后正确定位
    setTimeout(() => scrollToBottom(), 350)
  }
}

// scrollToBottom 节流：流式输出时高频调用会导致布局抖动，用 rAF 合并
let scrollRafId: number | null = null
const scrollToBottom = () => {
  if (scrollRafId !== null) return // 已有待执行帧，跳过
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = null
    nextTick(() => {
      if (messageArea.value) {
        messageArea.value.scrollTop = messageArea.value.scrollHeight
      }
    })
  })
}

const clearMessages = () => {
  messages.value = []
  expandedReasoning.value = []
  localStorage.removeItem(STORAGE_KEY)
}

// 删对话：点一下变红提示确认再清
const handleClearMessages = async () => {
  try {
    await ElMessageBox.confirm('确定要清空所有对话记录吗？', '清空对话', {
      confirmButtonText: '确定清空',
      cancelButtonText: '取消',
      type: 'warning',
    })
    // 流式输出中清空对话：先中断当前请求，避免aiIndex失效导致越界崩溃
    if (isThinking.value && abortController.value) {
      abortController.value.abort()
      abortController.value = null
      isThinking.value = false
      isReasoning.value = false
    }
    clearMessages()
    ElMessage.success('对话已清空')
  } catch {
    // 用户取消
  }
}

// 新对话：保存当前对话到历史记录，然后清空
const handleNewChat = async () => {
  if (messages.value.length === 0) {
    currentConversationId.value = ''
    showSidebar.value = true
    nextTick(() => inputBox.value?.focus())
    return
  }
  try {
    await ElMessageBox.confirm('确定要开始新对话吗？当前对话将被保存到历史记录。', '新对话', {
      confirmButtonText: '开始新对话',
      cancelButtonText: '取消',
      type: 'info',
    })
    // 流式输出中开始新对话：先中断当前请求，避免aiIndex失效导致越界崩溃
    if (isThinking.value && abortController.value) {
      abortController.value.abort()
      abortController.value = null
      isThinking.value = false
      isReasoning.value = false
    }
    saveCurrentConversation()
    clearMessages()
    currentConversationId.value = ''
    ElMessage.success('已开始新对话')
    showSidebar.value = true
    nextTick(() => inputBox.value?.focus())
  } catch {
    // 用户取消
  }
}

// 侧边栏开关
const toggleSidebar = () => {
  showSidebar.value = !showSidebar.value
}

// 存对话进历史（加载过的就更新，新的就新建）
const saveCurrentConversation = () => {
  if (messages.value.length === 0) return
  // 已有当前对话：更新现有对话
  if (currentConversationId.value) {
    const conv = conversations.value.find(c => c.id === currentConversationId.value)
    if (conv) {
      conv.messages = JSON.parse(JSON.stringify(messages.value))
      const firstUserMsg = messages.value.find(m => m.role === 'user')
      if (firstUserMsg) conv.title = firstUserMsg.content.slice(0, 30)
      saveConversations()
      return
    }
  }
  // 新对话：创建并保存
  const firstUserMsg = messages.value.find(m => m.role === 'user')
  const title = firstUserMsg ? firstUserMsg.content.slice(0, 30) : '新对话'
  const now = new Date()
  const time = `${now.getMonth() + 1}/${now.getDate()} ${now.getHours()}:${String(now.getMinutes()).padStart(2, '0')}`
  const newId = `conv_${Date.now()}`
  const conv = {
    id: newId,
    title,
    time,
    messages: JSON.parse(JSON.stringify(messages.value))
  }
  conversations.value.unshift(conv)
  currentConversationId.value = newId
  if (conversations.value.length > 20) {
    conversations.value = conversations.value.slice(0, 20)
  }
  saveConversations()
}

const saveConversations = () => {
  try {
    localStorage.setItem(CONV_STORAGE_KEY, JSON.stringify(conversations.value))
  } catch (e) {
    console.warn('[AI] 保存对话历史失败:', e)
    ElMessage.warning('对话历史存储空间不足，已自动清理部分旧对话')
    // 存储空间不足时自动清理最早的对话，保留最近的
    if (conversations.value.length > 5) {
      conversations.value = conversations.value.slice(0, Math.floor(conversations.value.length / 2))
      try {
        localStorage.setItem(CONV_STORAGE_KEY, JSON.stringify(conversations.value))
      } catch { /* ignore */ }
    }
  }
}

const loadConversations = () => {
  try {
    const saved = localStorage.getItem(CONV_STORAGE_KEY)
    if (saved) {
      conversations.value = JSON.parse(saved)
    }
  } catch {
    // ignore
  }
}

const loadConversation = (id: string) => {
  const conv = conversations.value.find(c => c.id === id)
  if (conv) {
    // 流式输出中切换对话：先中断当前请求，避免aiIndex失效导致越界崩溃
    if (isThinking.value && abortController.value) {
      abortController.value.abort()
      abortController.value = null
      isThinking.value = false
      isReasoning.value = false
    }
    currentConversationId.value = id
    messages.value = JSON.parse(JSON.stringify(conv.messages))
    expandedReasoning.value = []
    saveMessages()
    scrollToBottom()
    showSidebar.value = false
    ElMessage.success('已加载对话，可继续聊天')
  }
}

// 删侧边栏历史对话
const deleteConversation = async (id: string) => {
  try {
    await ElMessageBox.confirm('确定要删除这条对话记录吗？', '删除对话', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    conversations.value = conversations.value.filter(c => c.id !== id)
    // 如果删除的是当前正在使用的对话，清空消息区域
    if (currentConversationId.value === id) {
      currentConversationId.value = ''
      clearMessages()
    }
    saveConversations()
    ElMessage.success('对话已删除')
  } catch {
    // 用户取消
  }
}

const toggleReasoning = (idx: number) => {
  const i = expandedReasoning.value.indexOf(idx)
  if (i >= 0) {
    expandedReasoning.value.splice(i, 1)
  } else {
    expandedReasoning.value.push(idx)
  }
}

// 停止生成：用户点击“暂停”时中断流式请求，保留已生成内容
const stopGeneration = () => {
  if (abortController.value) {
    abortController.value.abort()
    abortController.value = null
  }
  isThinking.value = false
  isReasoning.value = false
  // 若AI尚无任何内容输出则移除空占位消息
  const lastIdx = messages.value.length - 1
  if (lastIdx >= 0 && messages.value[lastIdx].role === 'assistant' && messages.value[lastIdx].content === '' && !messages.value[lastIdx].reasoning) {
    messages.value.splice(lastIdx, 1)
  }
  saveMessages()
}

// XSS 防护：基于 DOM 白名单的 HTML 净化（比正则可靠，覆盖无引号事件、svg/object/embed、data:URI 等）
const sanitizeHtml = (html: string): string => {
  const doc = new DOMParser().parseFromString(html, 'text/html')
  const allowedTags = new Set(['p','br','strong','em','b','i','code','pre','a','ul','ol','li','table','thead','tbody','tr','td','th','h1','h2','h3','h4','h5','h6','blockquote','span','div','img','hr','sup','sub','del','s','mark','dl','dt','dd'])
  const globalAttrs = new Set(['class'])
  const allowedAttrs: Record<string, Set<string>> = {
    a: new Set(['href']),
    img: new Set(['src','alt','title','width','height']),
  }
  const walk = (node: Element) => {
    for (const el of Array.from(node.children)) {
      const tag = el.tagName.toLowerCase()
      if (!allowedTags.has(tag)) { el.remove(); continue }
      for (const attr of Array.from(el.attributes)) {
        const name = attr.name.toLowerCase()
        const allowed = (allowedAttrs[tag]?.has(name)) || globalAttrs.has(name)
        if (!allowed || name.startsWith('on')) { el.removeAttribute(attr.name); continue }
        if ((name === 'href' || name === 'src') && /^\s*(javascript|data):/i.test(attr.value)) {
          el.removeAttribute(attr.name)
        }
      }
      walk(el)
    }
  }
  walk(doc.body)
  return doc.body.innerHTML
}

// 转义 HTML 特殊字符，防止拼接出的 action 按钮属性/文本被注入
const escapeHtml = (str: string): string => {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

// 格式化内容：Markdown 渲染 + DOM 白名单净化防 XSS
const formatContent = (text: string) => {
  if (!text) return ''
  const html = marked.parse(text) as string
  const sanitized = sanitizeHtml(html)
  // 将 action: 链接转换为快捷操作按钮（净化后处理）
  // 安全：仅保留 data-route 属性，丢弃 $2 中的所有其他属性，避免 XSS 注入
  return sanitized.replace(/<a\s+href="action:([^"]+)"[^>]*>([^<]+)<\/a>/g, (_match, route, label) => {
    // 对 route 值做白名单校验，只允许字母/数字/斜杠/横杠
    const safeRoute = String(route).replace(/[^a-zA-Z0-9\/\-]/g, '')
    return `<a class="ai-action-btn" data-route="${escapeHtml(safeRoute)}">${escapeHtml(label)}</a>`
  })
}

// AI 快捷操作允许跳转的路由白名单（防止恶意 AI 返回任意路由导航到敏感页）
const allowedActionRoutes = new Set([
  'order/list','order/refunds','ticket/stations','ticket/routes','ticket/trips','ticket/vehicles','ticket/cargo',
  'marketing/coupons','marketing/point-rules','marketing/user-points','marketing/distribution',
  'user/list','user/passengers','user/idcard-verify',
  'setting/drivers','setting/admins','system/password','system/brand','system/config','system/agreement','system/logs',
  'design/banners','design/decorate','design/phone','design/coupon-display','track','dashboard'
])

// 超管专属路由：普通管理员点击这些快捷按钮时提示无权限而非静默跳转失败
const superAdminRoutes = new Set([
  'setting/drivers','setting/admins',
  'system/config','system/brand','system/agreement','system/logs'
])

// 处理 AI 回答中快捷操作按钮的点击（事件委托）
const handleContentClick = (e: MouseEvent) => {
  const target = (e.target as HTMLElement).closest('.ai-action-btn') as HTMLElement | null
  if (!target) return
  const route = target.dataset.route
  // 白名单校验：仅允许跳转到预定义的合法路由，防止恶意 AI 导航到任意页面
  if (route && allowedActionRoutes.has(route)) {
    // 超管权限检查：普通管理员点击超管专属页面时提示无权限
    if (superAdminRoutes.has(route) && !authStore.isSuperAdmin()) {
      ElMessage.warning('此页面需要超级管理员权限，您无权访问')
      return
    }
    showPanel.value = false // 关闭 AI 面板
    router.push('/' + route) // 跳转到对应管理页面
  }
}

// 图片上传限制
const MAX_IMAGE_SIZE = 2 * 1024 * 1024 // 单张最大2MB
const MAX_IMAGES = 3 // 最多3张

// 图片选择：转base64（含大小和数量校验）
// 单文件顺序读、多文件用Promise.all保证顺序 不然FileReader并发回调顺序不确定
const readFileAsDataURL = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    if (!file.type.startsWith('image/')) return resolve('')
    if (file.size > MAX_IMAGE_SIZE) { reject(new Error(`图片「${file.name}」超过2MB限制`)); return }
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(new Error('读取图片失败'))
    reader.readAsDataURL(file)
  })
}
const onImageSelect = async (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files) return
  const files = Array.from(target.files).filter(f => f.type.startsWith('image/'))
  if (files.length === 0) { target.value = ''; return }
  if (pendingImages.value.length + files.length > MAX_IMAGES) {
    ElMessage.warning(`最多上传${MAX_IMAGES}张图片`)
    target.value = ''
    return
  }
  for (const file of files) {
    try {
      const dataUrl = await readFileAsDataURL(file)
      if (dataUrl) pendingImages.value.push(dataUrl)
    } catch (err: any) {
      ElMessage.warning(err.message || '图片读取失败')
    }
  }
  target.value = '' // 重置，允许重复选同一文件
}

const removeImage = (idx: number) => {
  pendingImages.value.splice(idx, 1)
}

const sendMessage = async () => {
  const text = inputText.value.trim()
  const imgs = pendingImages.value
  if ((!text && imgs.length === 0) || isThinking.value) return

  // 图片生成检测：开启图片模式 / 当前为图片生成模型 / 消息含图片生成关键词
  // 支持自然语言，如“帮我画一张xxx图片”、“生成一张xxx”等，无需切换模型
  const imageKeywords = ['生成图片', '生成一张图', '生成图像', '画一张图', '画一幅',
    '帮我生成一张', '帮我画一张', '帮我画个', '帮我做一张', '做一张图', '做一张图片',
    '生成照片', 'create image', 'generate image', 'draw a picture']
  const isImageModel = currentModelIcon.value === 'image'
  const isImageRequest = (imageMode.value || isImageModel || imageKeywords.some(kw => text.includes(kw))) && text && imgs.length === 0

  if (isImageRequest) {
    inputText.value = ''
    imageMode.value = false
    // 自动选择图片生成模型：当前为图片模型则沿用，否则默认用“自然写实”
    const imgModel = isImageModel ? currentModelId.value : 'flux-realistic'
    await generateImage(text, imgModel)
    return
  }

  // 图片生成模型不支持图片识别：提示用户先切换到对话模型
  if (imgs.length > 0 && isImageModel) {
    ElMessage.warning('当前为图片生成模型，不支持图片识别。请先切换到对话模型后再上传图片')
    return
  }

  // 添加用户消息（含图片）
  messages.value.push({ role: 'user', content: text || '请描述这张图片', images: imgs.length ? imgs : undefined })
  inputText.value = ''
  pendingImages.value = []
  saveMessages()
  scrollToBottom()

  // 图片识别：当前模型不支持视觉时，自动切换到最快可用的视觉模型
  // 视觉模型按响应速度排序（8B→11B→12B→17B→119B...），谁先能识别让谁先上
  if (imgs.length > 0 && !currentModelSupportsVision.value) {
    // 收集所有已配置Key的视觉模型，按配置顺序（即速度排序）
    const visionModels: { model: any; provider: string }[] = []
    for (const p of providers.value) {
      if (!p.has_key) continue
      for (const m of (p.models || [])) {
        if (m.supports_vision && m.icon !== 'image') {
          visionModels.push({ model: m, provider: p.value })
        }
      }
    }
    if (visionModels.length === 0) {
      messages.value.push({
        role: 'assistant',
        content: `当前模型「${currentModelName.value}」不支持图片识别，且未找到已配置 API Key 的视觉模型。\n请在系统配置中为英伟达配置 API Key，即可使用以下视觉模型：\n• Nemotron Nano VL 8B（极速）\n• Cosmos Reason2 8B\n• Llama 3.2 11B Vision\n• Nemotron Nano 12B VL\n• Nemotron 3 Nano Omni 30B（推荐）\n• Kimi K2.6\n• Llama 3.2 90B Vision（最强）`
      })
      saveMessages()
      scrollToBottom()
      return
    }
    // 逐个尝试切换，失败则自动降级到下一个视觉模型
    let switched = false
    for (const { model: vm, provider: vp } of visionModels) {
      try {
        const res: any = await aiApi.switchModel(vm.id)
        currentModelId.value = vm.id
        if (res.data?.provider) currentProvider.value = res.data.provider
        ElMessage.success(`已自动切换到视觉模型「${vm.name}」识别图片`)
        switched = true
        break
      } catch {
        // 当前视觉模型切换失败，继续尝试下一个
        console.warn(`[AI] 视觉模型「${vm.name}」切换失败，尝试下一个...`)
      }
    }
    if (!switched) {
      ElMessage.error('所有视觉模型切换均失败，请手动在模型列表中选择一个视觉模型后重试')
      return
    }
  }

  // 添加 AI 占位消息（用于流式追加）
  messages.value.push({ role: 'assistant', content: '' })
  isThinking.value = true
  scrollToBottom()

  let reader: any = null
  try {
    // 构造消息历史（不含当前空的 AI 占位），支持多模态content
    // 截断历史消息：保留最近66条，防止超出模型 token 限制导致请求失败
    const MAX_CHAT_HISTORY = 66
    // 仅保留最近3条消息的图片base64，旧消息用文字占位符替代
    // 原因：base64图片体积大，全量发送会导致请求体超nginx限制(413)且API费用暴增
    const IMAGE_KEEP_COUNT = 3
    const historyMsgs = messages.value
      .filter((_, idx) => idx < messages.value.length - 1)
      // 过滤图片生成消息：图片生成的提示词和结果不进入对话上下文，避免污染后续对话
      .filter(m => !m.isImageGen)
      .slice(-MAX_CHAT_HISTORY)
    const chatMessages = historyMsgs.map((m, idx, arr) => {
      const hasImages = m.images && m.images.length > 0
      // 看是不是在最近N条里（带图的才用判断）
      const isRecent = idx >= arr.length - IMAGE_KEEP_COUNT
      if (hasImages && isRecent) {
        // 最近的消息：发送完整多模态内容（含base64图片）
        const content: any[] = []
        if (m.content) content.push({ type: 'text', text: m.content })
        for (const img of m.images!) {
          content.push({ type: 'image_url', image_url: { url: img } })
        }
        return { role: m.role, content }
      } else if (hasImages && !isRecent) {
        // 旧消息：用文字占位符替代图片，节省请求体积和API费用
        const imgPlaceholder = `[已省略${m.images!.length}张图片]`
        return { role: m.role, content: (m.content ? m.content + ' ' : '') + imgPlaceholder }
      }
      return { role: m.role, content: m.content }
    })

    // 创建可中断控制器，用于“暂停”按钮
    const controller = new AbortController()
    abortController.value = controller
    const response = await authFetch('/admin/ai/chat', {
      method: 'POST',
      body: JSON.stringify({ messages: chatMessages }),
      signal: controller.signal,
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    reader = response.body?.getReader()
    if (!reader) throw new Error('无法读取响应流')

    const decoder = new TextDecoder()
    let buffer = ''
    let aiIndex = messages.value.length - 1

    // SSE流读取主循环
    // 收到done事件后主动break退出，不依赖连接关闭
    // 避免后端未及时关闭连接时前端卡在"正在回答"状态导致无法继续聊天
    sseLoop: while (true) {
      // 带超时的read：网络断开/nginx超时断开时fetch可能不触发done
      // 超时后抛错并abort fetch，避免reader.read()永远pending导致fd泄漏
      const READ_TIMEOUT_MS = 60000 // 60秒无数据则判定超时
      let timeoutId: ReturnType<typeof setTimeout> | undefined
      const readResult = await Promise.race([
        reader.read(),
        new Promise<{ done: boolean; value?: Uint8Array }>((_, reject) => {
          timeoutId = setTimeout(() => {
            controller.abort() // 中止fetch，否则reader.read()永不resolve
            reject(new Error('读取超时：AI响应超过60秒未返回数据，请稍后重试'))
          }, READ_TIMEOUT_MS)
        }),
      ])
      // 收到数据后立即清除本次超时计时器，避免旧计时器在新迭代中误触abort
      if (timeoutId) clearTimeout(timeoutId)

      const { done, value } = readResult
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // 按行解析 SSE 数据
      const lines = buffer.split('\n')
      buffer = lines.pop() || '' // 保留最后不完整的行

      for (const line of lines) {
        if (!line.startsWith('data:')) continue
        const dataStr = line.slice(5).trim()
        if (!dataStr) continue

        try {
          const data = JSON.parse(dataStr)
          if (data.type === 'reasoning' && data.content) {
            // 追加思考过程
            messages.value[aiIndex].reasoning = (messages.value[aiIndex].reasoning || '') + data.content
            // 自动展开思考过程
            if (!expandedReasoning.value.includes(aiIndex)) {
              expandedReasoning.value.push(aiIndex)
            }
            isReasoning.value = true
            scrollToBottom()
          } else if (data.type === 'content' && data.content) {
            // 追加正文内容
            messages.value[aiIndex].content += data.content
            isReasoning.value = false
            scrollToBottom()
          } else if (data.type === 'done') {
            // 收到done事件后主动退出循环，不依赖连接关闭
            // 避免后端未及时关闭连接时前端卡在"正在回答"状态
            isReasoning.value = false
            break sseLoop
          }
        } catch {
          // 忽略无法解析的行
        }
      }
    }

    // 流结束，buffer 里可能还剩最后一条 data 没消费
    if (buffer.startsWith('data:')) {
      try {
        const dataStr = buffer.slice(5).trim()
        if (dataStr) {
          const data = JSON.parse(dataStr)
          if (data.type === 'reasoning' && data.content) {
            messages.value[aiIndex].reasoning = (messages.value[aiIndex].reasoning || '') + data.content
          } else if (data.type === 'content' && data.content) {
            messages.value[aiIndex].content += data.content
          }
        }
      } catch { /* ignore */ }
    }

    // 如果 AI 没有返回任何内容
    if (messages.value[aiIndex].content === '') {
      messages.value[aiIndex].content = '抱歉，我没有收到有效回复，请稍后再试。'
    }
  } catch (e) {
    const err = e as Error
    // 用户主动暂停：保留已生成内容，不报错
    if (err.name === 'AbortError') {
      const lastIdx = messages.value.length - 1
      if (lastIdx >= 0 && messages.value[lastIdx].role === 'assistant' && messages.value[lastIdx].content === '' && !messages.value[lastIdx].reasoning) {
        messages.value.splice(lastIdx, 1) // 暂停时若AI无任何内容则移除空占位
      }
    } else {
      const lastIdx = messages.value.length - 1
      if (lastIdx >= 0 && messages.value[lastIdx].content === '') {
        messages.value[lastIdx].content = `请求失败：${err.message}。请检查AI配置是否正确。`
      }
      ElMessage.error('AI回复失败，请稍后重试')
    }
  } finally {
    // 主动取消reader，避免SSE连接资源泄漏
    if (reader) reader.cancel().catch(() => {})
    abortController.value = null
    isThinking.value = false
    isReasoning.value = false
    saveMessages()
    scrollToBottom()
  }
}

// addWatermark 在图片右下角添加"AI生成.狸猫售票系统"水印
// 使用Canvas API，无需后端字体依赖，浏览器原生支持中文渲染
// 任何异常均返回原图，确保不阻断生成流程
const addWatermark = (dataURI: string): Promise<string> => {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => {
      try {
        const canvas = document.createElement('canvas')
        canvas.width = img.width
        canvas.height = img.height
        const ctx = canvas.getContext('2d')
        if (!ctx) { resolve(dataURI); return }

        // 绘制原图
        ctx.drawImage(img, 0, 0)

        // 水印文字
        const text = 'AI生成.狸猫售票系统'
        const fontSize = Math.max(14, Math.floor(img.width / 35))
        ctx.font = `600 ${fontSize}px "PingFang SC", "Microsoft YaHei", "Noto Sans CJK SC", sans-serif`

        // 测量文字宽度，计算背景框尺寸
        const metrics = ctx.measureText(text)
        const padX = fontSize * 0.6
        const padY = fontSize * 0.35
        const bgW = metrics.width + padX * 2
        const bgH = fontSize + padY * 2
        const margin = fontSize * 0.8
        const x = img.width - bgW - margin
        const y = img.height - bgH - margin

        // 半透明黑色圆角背景
        ctx.fillStyle = 'rgba(0, 0, 0, 0.42)'
        ctx.beginPath()
        const r = 6
        ctx.moveTo(x + r, y)
        ctx.arcTo(x + bgW, y, x + bgW, y + bgH, r)
        ctx.arcTo(x + bgW, y + bgH, x, y + bgH, r)
        ctx.arcTo(x, y + bgH, x, y, r)
        ctx.arcTo(x, y, x + bgW, y, r)
        ctx.closePath()
        ctx.fill()

        // 白色文字
        ctx.fillStyle = 'rgba(255, 255, 255, 0.88)'
        ctx.textBaseline = 'middle'
        ctx.textAlign = 'left'
        ctx.fillText(text, x + padX, y + bgH / 2)

        // 保留原图类型（JPEG用压缩质量，PNG无损）
        const isJPEG = dataURI.includes('image/jpeg') || dataURI.includes('image/jpg')
        const mimeType = isJPEG ? 'image/jpeg' : 'image/png'
        const quality = isJPEG ? 0.92 : undefined
        resolve(canvas.toDataURL(mimeType, quality))
      } catch {
        resolve(dataURI) // 任何异常都返回原图，确保不阻断流程
      }
    }
    img.onerror = () => resolve(dataURI) // 图片加载失败返回原图
    img.src = dataURI
  })
}

// generateImage 調用后端AI图片生成接口
// 用户输入文字描述 → 后端调用Pollinations.ai → 返回base64图片
// model: 图片生成模型ID（flux-realistic/flux-portrait/flux/turbo/sana），不传则用当前模型
const generateImage = async (prompt: string, model?: string) => {
  // 添加用户消息（标记为图片生成，不进入对话上下文）
  messages.value.push({ role: 'user', content: prompt, isImageGen: true })
  saveMessages()
  scrollToBottom()

  // 添加AI占位消息（生成中状态）— 不保存到localStorage，避免刷新后显示永久"生成中"
  messages.value.push({ role: 'assistant', content: '', isGeneratingImage: true, isImageGen: true })
  isThinking.value = true
  scrollToBottom()

  // 创建可中断控制器，与聊天流共用停止按钮
  const controller = new AbortController()
  abortController.value = controller

  try {
    const resp = await authFetch('/admin/ai/image', {
      method: 'POST',
      body: JSON.stringify({ prompt, model: model || currentModelId.value }),
      signal: controller.signal,
    })

    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)

    const data = await resp.json()
    if (data.code !== 0) throw new Error(data.message || '图片生成失败')

    // 更新AI消息：展示生成的图片（添加水印后），保持isImageGen标记
    const aiIdx = messages.value.length - 1
    messages.value[aiIdx].isGeneratingImage = false
    messages.value[aiIdx].generatedImage = await addWatermark(data.data.image)
    messages.value[aiIdx].content = '已为你生成图片'
    messages.value[aiIdx].isImageGen = true
    saveMessages()
    scrollToBottom()
  } catch (e) {
    const err = e as Error
    // 用户主动取消：移除占位消息，不报错
    if (err.name === 'AbortError') {
      const aiIdx = messages.value.length - 1
      if (aiIdx >= 0 && messages.value[aiIdx].isGeneratingImage) {
        messages.value.splice(aiIdx, 1)
      }
    } else {
      const aiIdx = messages.value.length - 1
      messages.value[aiIdx].isGeneratingImage = false
      messages.value[aiIdx].content = `图片生成失败：${err.message}`
      ElMessage.error('图片生成失败，请稍后重试')
    }
  } finally {
    abortController.value = null
    isThinking.value = false
    imageMode.value = false
    saveMessages()
    scrollToBottom()
  }
}

// previewImage 点击图片放大预览（应用内遮罩）
const previewImage = (src: string) => {
  previewImageSrc.value = src
}

// downloadImage 下载图片（从base64 data URI创建下载链接）
const downloadImage = (src: string) => {
  const link = document.createElement('a')
  link.href = src
  link.download = `ai-image-${Date.now()}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}
</script>

<style scoped>
/* 悬浮按钮 - position:fixed 脱离 topbar 的 backdrop-filter 层叠上下文，避免被页面元素遮挡 */
.ai-trigger {
  position: fixed;
  top: 14px;
  right: 28px;
  z-index: 3000;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px 2px 3px;
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08), 0 2px 8px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: box-shadow 0.2s;
  user-select: none;
  flex-shrink: 0;
  -webkit-tap-highlight-color: transparent;
  outline: none;
}
.ai-trigger:hover {
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.15), 0 4px 12px rgba(0, 0, 0, 0.1);
}
.ai-trigger:active {
  background: #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.12), 0 2px 6px rgba(0, 0, 0, 0.08);
}
.ai-trigger-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.ai-trigger-text {
  font-size: 12px;
  font-weight: 500;
  color: #000;
  white-space: nowrap;
}
/* 头部按钮选中态 */
.ai-header-btn.active {
  background: rgba(0, 0, 0, 0.06);
  color: #000;
}

/* 聊天面板 */
.ai-panel {
  position: fixed;
  top: 60px;
  right: 20px;
  width: 400px;
  height: calc(100vh - 80px);
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.06), 0 8px 32px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  z-index: 3000;
  overflow: hidden;
}

/* 面板头部：GPT风微毛玻璃，半透明泛光 */
/* z-index确保header及其子元素(设置下拉框)不被下方model-bar遮挡 */
.ai-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(8px) saturate(1.2);
  -webkit-backdrop-filter: blur(8px) saturate(1.2);
  position: relative;
  z-index: 20;
}
.ai-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.ai-header-info {
  display: flex;
  align-items: center;
  gap: 10px;
}
.ai-header-avatar {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
}
.ai-header-meta {
  display: flex;
  flex-direction: column;
}
.ai-header-name {
  font-size: 14px;
  font-weight: 600;
  color: #000;
}
.ai-header-status {
  font-size: 12px;
  color: #000;
  font-weight: 500;
}
.ai-header-actions {
  display: flex;
  gap: 8px;
}
.ai-header-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  color: #000;
  transition: all 0.15s;
}
.ai-header-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #000;
}
/* 删除按钮：默认黑色，悬停变红提示危险操作 */
.ai-header-btn-danger svg path {
  fill: #2c2c2c;
  transition: fill 0.2s;
}
.ai-header-btn-danger:hover svg path {
  fill: #d81e06;
}

/* 消息区域 */
.ai-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px 16px 28px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 欢迎信息 */
.ai-welcome {
  text-align: center;
  padding: 30px 16px;
  color: #000;
}
.ai-welcome-icon {
  width: 56px;
  height: 56px;
  margin-bottom: 12px;
}
.ai-welcome p {
  font-size: 14px;
  color: #000;
  margin: 4px 0;
}
.ai-welcome-hint {
  font-size: 12px !important;
  color: #999 !important;
}

/* 消息气泡 */
.ai-msg {
  display: flex;
  max-width: 100%;
}
.ai-msg-user {
  justify-content: flex-end;
}
.ai-msg-ai {
  justify-content: flex-start;
}
.ai-msg-bubble {
  max-width: 85%;
  padding: 10px 14px;
  border-radius: 14px;
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
}
.ai-msg-user .ai-msg-bubble {
  background: #000;
  color: #fff;
  border-bottom-right-radius: 4px;
}
.ai-msg-ai-bubble {
  background: #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08);
  color: #000;
  border-bottom-left-radius: 4px;
}

/* 打字动画 */
.ai-typing {
  display: inline-flex;
  gap: 4px;
  padding: 4px 0;
}
.ai-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #000;
  animation: ai-bounce 1.4s infinite ease-in-out;
}
.ai-dot:nth-child(2) {
  animation-delay: 0.2s;
}
.ai-dot:nth-child(3) {
  animation-delay: 0.4s;
}
@keyframes ai-bounce {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-6px); opacity: 1; }
}

/* 输入区域 */
.ai-input-area {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}
/* 待发送图片预览行 */
.ai-image-preview-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ai-image-preview-item {
  position: relative;
  width: 56px;
  height: 56px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.1);
}
.ai-preview-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.ai-image-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  font-size: 12px;
  line-height: 16px;
  text-align: center;
  cursor: pointer;
}
/* 输入行：图片按钮 + 输入框 + 发送按钮 */
.ai-input-row {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}
.ai-image-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #fff;
  color: #000;
  border: 1px solid rgba(0, 0, 0, 0.1);
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.2s;
}
.ai-image-btn:hover {
  background: #f5f5f5;
  box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.05);
}
.ai-input {
  flex: 1;
  border: 1.5px solid #1a1a1a;
  border-radius: 10px;
  padding: 8px 12px;
  font-size: 14px;
  resize: none;
  outline: none;
  font-family: inherit;
  background: #fff;
  color: #000;
  transition: border-color 0.2s, box-shadow 0.2s;
  max-height: 120px;
}
.ai-input:focus {
  border-color: #1e6fff;
  box-shadow: 0 0 0 3px rgba(30, 111, 255, 0.1);
}
/* AI回复中：输入框蓝色边框行走光效 */
/* @property 定义可动画的角度变量，驱动conic-gradient旋转 */
@property --ai-angle {
  syntax: '<angle>';
  initial-value: 0deg;
  inherits: false;
}
.ai-input-wrap {
  flex: 1;
  position: relative;
  border-radius: 10px;
}
.ai-input-wrap .ai-input {
  width: 100%;
  position: relative;
  z-index: 1;
}
/* 藍色光带沿输入框边框转圈行走 */
/* conic-gradient生成旋转光弧，mask仅显示边框区域 */
/* inset:-2px让伪元素超出输入框，z-index:0藏在输入框后面，输入框遮挡中心只露出边框光环 */
.ai-input-wrap.is-thinking::before {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: 12px;
  background: conic-gradient(
    from var(--ai-angle),
    transparent 0%,
    transparent 60%,
    rgba(30, 111, 255, 0.9) 72%,
    #1e6fff 84%,
    rgba(30, 111, 255, 0.9) 96%,
    transparent 100%
  );
  pointer-events: none;
  z-index: 0;
  animation: ai-border-walk 2s linear infinite;
}
.ai-input-wrap.is-thinking .ai-input {
  border-color: rgba(30, 111, 255, 0.4);
  box-shadow: 0 0 12px rgba(30, 111, 255, 0.25);
}
@keyframes ai-border-walk {
  to { --ai-angle: 360deg; }
}
/* GPT风格方形图标发送按钮 */
.ai-send-icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #000;
  color: #fff;
  border: none;
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.2s;
}
.ai-send-icon-btn:hover:not(.is-disabled):not(.is-stop) {
  background: rgba(0, 0, 0, 0.85);
}
.ai-send-icon-btn.is-disabled {
  background: rgba(0, 0, 0, 0.08);
  color: rgba(0, 0, 0, 0.3);
  cursor: not-allowed;
}
.ai-send-icon-btn.is-stop {
  background: #fff;
  color: #000;
  border: 1px solid rgba(0, 0, 0, 0.15);
}
.ai-send-icon-btn.is-stop:hover {
  background: #f5f5f5;
  box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.05);
}
.ai-send-icon-btn:active:not(.is-disabled) {
  transform: scale(0.95);
}
/* 消息内图片 */
.ai-msg-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
}
.ai-msg-image {
  max-width: 160px;
  max-height: 160px;
  border-radius: 8px;
  object-fit: cover;
}

/* 图片生成模式切换按钮 */
.ai-mode-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}
.ai-mode-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border: 1.5px solid rgba(0, 0, 0, 0.15);
  border-radius: 99px;
  background: transparent;
  color: rgba(0, 0, 0, 0.6);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.ai-mode-btn.active {
  background: #000;
  color: #fff;
  border-color: #000;
}
.ai-mode-btn:hover:not(.active) {
  border-color: rgba(0, 0, 0, 0.4);
  box-shadow: 0 0 8px rgba(0, 0, 0, 0.08);
}

/* 图片生成中：相片框 + 水波纹 + 黑白主题白光 */
.ai-photo-frame {
  width: 220px;
  padding: 10px;
  background: #1a1a1a;
  border-radius: 4px;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.08), 0 4px 24px rgba(0, 0, 0, 0.4);
  position: relative;
}
.ai-photo-frame-inner {
  width: 200px;
  height: 200px;
  background: #0a0a0a;
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;
}
/* 白光光晕 */
.ai-photo-glow {
  position: absolute;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.35), transparent 70%);
  animation: ai-photo-glow-anim 1.5s ease-in-out infinite alternate;
}
@keyframes ai-photo-glow-anim {
  0% { opacity: 0.3; transform: scale(0.7); }
  100% { opacity: 0.9; transform: scale(1.3); }
}
/* 水波纹扩散 */
.ai-ripple {
  position: absolute;
  width: 40px;
  height: 40px;
  border: 2px solid rgba(255, 255, 255, 0.7);
  border-radius: 50%;
  animation: ai-ripple-anim 2s ease-out infinite;
}
.ai-ripple-delay-1 { animation-delay: 0.6s; }
.ai-ripple-delay-2 { animation-delay: 1.2s; }
@keyframes ai-ripple-anim {
  0% { transform: scale(0.5); opacity: 1; }
  100% { transform: scale(4.5); opacity: 0; border-color: rgba(255, 255, 255, 0.1); }
}
.ai-photo-frame-text {
  position: relative;
  z-index: 1;
  color: rgba(255, 255, 255, 0.75);
  font-size: 12px;
  letter-spacing: 0.5px;
}

/* AI 生成的图片展示 */
.ai-generated-image-wrap {
  margin: 8px 0;
}
.ai-generated-image {
  max-width: 100%;
  border-radius: 8px;
  border: 1.5px solid rgba(0, 0, 0, 0.12);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: box-shadow 0.2s, transform 0.2s;
}
.ai-generated-image:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  transform: scale(1.01);
}
/* 图片操作按钮（放大、下载） */
.ai-image-actions {
  display: flex;
  gap: 8px;
  margin-top: 6px;
}
.ai-image-action-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1.5px solid rgba(0, 0, 0, 0.15);
  border-radius: 99px;
  background: transparent;
  color: rgba(0, 0, 0, 0.6);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.ai-image-action-btn:hover {
  background: #000;
  color: #fff;
  border-color: #000;
}

/* 图片预览遮罩 */
.ai-image-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.88);
  z-index: 9999;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
}
.ai-overlay-image {
  max-width: 90vw;
  max-height: 80vh;
  border-radius: 8px;
  box-shadow: 0 0 40px rgba(255, 255, 255, 0.1);
}
.ai-overlay-close-btn {
  position: absolute;
  top: 20px;
  right: 20px;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}
.ai-overlay-close-btn:hover {
  background: rgba(255, 255, 255, 0.25);
}
.ai-overlay-toolbar {
  display: flex;
  gap: 12px;
}
.ai-overlay-download-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 20px;
  border-radius: 99px;
  background: #fff;
  color: #000;
  border: none;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}
.ai-overlay-download-btn:hover {
  background: rgba(255, 255, 255, 0.9);
  transform: scale(1.05);
}
/* 遮罩淡入淡出 */
.ai-fade-enter-active, .ai-fade-leave-active {
  transition: opacity 0.2s;
}
.ai-fade-enter-from, .ai-fade-leave-to {
  opacity: 0;
}

/* Markdown 渲染样式（v-html 内容需用 :deep 穿透 scoped） */
.ai-markdown :deep(table),
.ai-reasoning-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 8px 0;
  font-size: 13px;
}
.ai-markdown :deep(th),
.ai-markdown :deep(td),
.ai-reasoning-body :deep(th),
.ai-reasoning-body :deep(td) {
  border: 1px solid rgba(0, 0, 0, 0.15);
  padding: 6px 10px;
  text-align: left;
}
.ai-markdown :deep(th),
.ai-reasoning-body :deep(th) {
  background: rgba(0, 0, 0, 0.04);
  font-weight: 600;
}
.ai-markdown :deep(code),
.ai-reasoning-body :deep(code) {
  background: rgba(0, 0, 0, 0.05);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'Courier New', Consolas, monospace;
}
.ai-markdown :deep(pre),
.ai-reasoning-body :deep(pre) {
  background: rgba(0, 0, 0, 0.04);
  border-radius: 8px;
  padding: 12px;
  overflow-x: auto;
  margin: 8px 0;
}
.ai-markdown :deep(pre code),
.ai-reasoning-body :deep(pre code) {
  background: none;
  padding: 0;
}
.ai-markdown :deep(ul),
.ai-markdown :deep(ol),
.ai-reasoning-body :deep(ul),
.ai-reasoning-body :deep(ol) {
  padding-left: 20px;
  margin: 8px 0;
}
.ai-markdown :deep(li),
.ai-reasoning-body :deep(li) {
  margin: 4px 0;
}
.ai-markdown :deep(blockquote),
.ai-reasoning-body :deep(blockquote) {
  border-left: 3px solid rgba(0, 0, 0, 0.2);
  padding-left: 12px;
  margin: 8px 0;
  color: #666;
}
.ai-markdown :deep(a),
.ai-reasoning-body :deep(a) {
  color: #1e6fff;
  text-decoration: underline;
}
.ai-markdown :deep(strong),
.ai-reasoning-body :deep(strong) {
  font-weight: 700;
}
.ai-markdown :deep(h1),
.ai-markdown :deep(h2),
.ai-markdown :deep(h3),
.ai-markdown :deep(h4),
.ai-reasoning-body :deep(h1),
.ai-reasoning-body :deep(h2),
.ai-reasoning-body :deep(h3),
.ai-reasoning-body :deep(h4) {
  margin: 12px 0 8px;
  font-weight: 600;
}
.ai-markdown :deep(h1),
.ai-reasoning-body :deep(h1) { font-size: 18px; }
.ai-markdown :deep(h2),
.ai-reasoning-body :deep(h2) { font-size: 16px; }
.ai-markdown :deep(h3),
.ai-reasoning-body :deep(h3) { font-size: 15px; }
.ai-markdown :deep(h4),
.ai-reasoning-body :deep(h4) { font-size: 14px; }

/* AI 快捷操作按钮（黑白主题药丸形） */
.ai-markdown :deep(.ai-action-btn) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 14px;
  margin: 4px 4px 0 0;
  border: 1.5px solid rgba(0, 0, 0, 0.15);
  border-radius: 99px;
  background: transparent;
  color: rgba(0, 0, 0, 0.7);
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.2s;
  -webkit-tap-highlight-color: transparent;
}
.ai-markdown :deep(.ai-action-btn:hover) {
  background: #000;
  color: #fff;
  border-color: #000;
  text-decoration: none;
}

/* 遮罩：透明，仅用于点击外部关闭面板 */
.ai-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2999;
  background: transparent;
}

/* 面板滑出动画 */
.ai-slide-enter-active,
.ai-slide-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}
.ai-slide-enter-from,
.ai-slide-leave-to {
  transform: translateY(20px);
  opacity: 0;
}

/* 思考过程 */
.ai-reasoning {
  margin-bottom: 8px;
  border-radius: 8px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}
.ai-reasoning-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 12px;
  color: #000;
  font-weight: 500;
  transition: all 0.15s;
}
.ai-reasoning-header:hover {
  background: rgba(0, 0, 0, 0.04);
}
.ai-reasoning-icon {
  flex-shrink: 0;
}
.ai-reasoning-toggle {
  margin-left: auto;
  color: #999;
}
.ai-reasoning-body {
  padding: 8px 10px;
  font-size: 13px;
  color: #666;
  line-height: 1.6;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  max-height: 200px;
  overflow-y: auto;
}

/* 模型选择栏 */
.ai-model-bar {
  position: relative;
  padding: 6px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}
.ai-model-selector {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  border-radius: 99px;
  background: transparent;
  border: 1px solid rgba(0, 0, 0, 0.12);
  cursor: pointer;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.7);
  transition: all 0.15s;
}
.ai-model-selector:hover {
  background: rgba(0, 0, 0, 0.04);
  border-color: rgba(0, 0, 0, 0.2);
}
.ai-model-arrow {
  transition: transform 0.2s;
}
.ai-model-arrow.rotated {
  transform: rotate(180deg);
}
.ai-model-dropdown {
  position: absolute;
  top: 100%;
  left: 16px;
  margin-top: 4px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  padding: 4px;
  z-index: 10;
  max-height: 300px;
  overflow-y: auto;
  min-width: 200px;
}
.ai-model-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.15s;
}
.ai-model-item:hover {
  background: rgba(0, 0, 0, 0.04);
}
/* 模型选中：黑色主题，字体变白 */
.ai-model-item.active {
  background: #000;
  color: #fff;
  font-weight: 600;
}
.ai-model-item.active .ai-model-icon {
  color: #fff;
}
.ai-model-item-name {
  flex: none;
}
.ai-model-icon {
  flex-shrink: 0;
  color: rgba(0, 0, 0, 0.5);
}
/* 模型标签（推荐/极速/不推荐等） */
.ai-model-item-tag {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  font-weight: 500;
  line-height: 1.4;
  flex-shrink: 0;
}
.ai-tag-primary { background: #409eff; color: #fff; }
.ai-tag-secondary { background: rgba(64,158,255,0.12); color: #409eff; }
.ai-tag-info { background: rgba(144,147,153,0.12); color: #909399; }
.ai-tag-warning { background: rgba(230,162,60,0.12); color: #e6a23c; }
.ai-model-item.active .ai-model-item-tag { opacity: 0.85; }
/* 模型选择器内的切换图标 */
.ai-model-switch-icon {
  flex-shrink: 0;
  opacity: 0.5;
}

/* ChatGPT风格侧边栏：紧挨主面板左侧 */
.ai-sidebar {
  position: fixed;
  top: 60px;
  right: 430px;
  width: 260px;
  height: calc(100vh - 80px);
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.06), 0 8px 32px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  z-index: 3001;
  overflow: hidden;
}
.ai-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(8px) saturate(1.2);
  -webkit-backdrop-filter: blur(8px) saturate(1.2);
}
.ai-sidebar-title {
  font-size: 13px;
  font-weight: 600;
  color: #000;
}
.ai-sidebar-close {
  cursor: pointer;
  color: #666;
  padding: 4px;
  border-radius: 6px;
  transition: all 0.15s;
  display: flex;
  align-items: center;
}
.ai-sidebar-close:hover {
  background: rgba(0, 0, 0, 0.06);
  color: #000;
}
.ai-sidebar-newchat {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: #000;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  transition: background 0.15s;
}
.ai-sidebar-newchat:hover {
  background: rgba(0, 0, 0, 0.04);
}
.ai-sidebar-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px;
}
.ai-sidebar-empty {
  padding: 24px 14px;
  text-align: center;
  font-size: 12px;
  color: #999;
}
.ai-sidebar-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
}
.ai-sidebar-item:hover {
  background: rgba(0, 0, 0, 0.04);
}
.ai-sidebar-item-icon {
  flex-shrink: 0;
  color: rgba(0, 0, 0, 0.4);
}
.ai-sidebar-item-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
  flex: 1;
}
.ai-sidebar-item-title {
  font-size: 13px;
  color: #000;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ai-sidebar-item-time {
  font-size: 11px;
  color: #999;
}
/* 删除按钮：默认隐藏，hover 对话项时显示 */
.ai-sidebar-item-delete {
  flex-shrink: 0;
  display: none;
  color: #999;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.15s;
}
.ai-sidebar-item:hover .ai-sidebar-item-delete {
  display: flex;
  align-items: center;
}
.ai-sidebar-item-delete:hover {
  background: rgba(229, 57, 53, 0.1);
  color: #e53935;
}

/* 侧边栏滑出动画 */
.ai-sidebar-slide-enter-active,
.ai-sidebar-slide-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}
.ai-sidebar-slide-enter-from,
.ai-sidebar-slide-leave-to {
  transform: translateX(-100%);
  opacity: 0;
}
.ai-model-group-label {
  padding: 6px 12px 2px;
  font-size: 10px;
  color: #999;
  font-weight: 600;
  letter-spacing: 0.5px;
}
/* 下拉动画 */
.ai-dropdown-enter-active,
.ai-dropdown-leave-active {
  transition: all 0.2s;
}
.ai-dropdown-enter-from,
.ai-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* 狸猫站立左右晃动动画 */
@keyframes ai-cat-sway {
  0%, 100% { transform: rotate(-4deg); }
  50% { transform: rotate(4deg); }
}
.ai-cat-sway {
  transform-origin: bottom center;
  animation: ai-cat-sway 2.5s ease-in-out infinite;
}

/* ========== 设置下拉菜单 ========== */
.ai-settings-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 4px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  padding: 4px;
  z-index: 10;
  min-width: 140px;
}
.ai-settings-label {
  padding: 6px 12px 2px;
  font-size: 10px;
  color: #999;
  font-weight: 600;
  letter-spacing: 0.5px;
}
.ai-settings-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.8);
  transition: background 0.15s;
}
.ai-settings-item:hover {
  background: rgba(0, 0, 0, 0.04);
}
.ai-settings-item.active {
  color: #000;
  font-weight: 600;
}
.ai-settings-check {
  margin-left: auto;
  color: #000;
}

/* ========== 暗色模式全覆盖 ========== */
/* 悬浮按钮 */
.ai-dark.ai-trigger {
  background: #1e1e1e;
  color: #e0e0e0;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.08), 0 2px 8px rgba(0, 0, 0, 0.3);
}
.ai-dark.ai-trigger:active {
  background: #252525;
}
.ai-dark .ai-trigger-text {
  color: #e0e0e0;
}

/* 面板 */
/* 根因修复：在容器上统一设color，所有子元素自动继承浅色文字，
   不再需要逐个元素覆盖。新增元素也不会忘记设color导致继承全局黑色 */
.ai-dark.ai-panel {
  background: #1a1a1a;
  color: #e0e0e0;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.06), 0 8px 32px rgba(0, 0, 0, 0.4);
}

/* 头部 */
.ai-dark .ai-header,
.ai-dark .ai-header-info {
  color: #e0e0e0;
}
.ai-dark .ai-header {
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(30, 30, 30, 0.8);
}
.ai-dark .ai-header-name,
.ai-dark .ai-header-status,
.ai-dark .ai-header-btn {
  color: #e0e0e0;
}
.ai-dark .ai-header-btn:hover,
.ai-dark .ai-header-btn.active {
  background: rgba(255, 255, 255, 0.08);
}
.ai-dark .ai-header-btn-danger svg path {
  fill: #ccc;
}
.ai-dark .ai-header-btn-danger:hover svg path {
  fill: #f44336;
}

/* 消息区域 */
.ai-dark .ai-welcome,
.ai-dark .ai-welcome p {
  color: #e0e0e0;
}
.ai-dark .ai-welcome-hint {
  color: #999 !important;
}

/* 用户气泡：深色模式反转为浅色 */
.ai-dark .ai-msg-user .ai-msg-bubble {
  background: #e0e0e0;
  color: #1a1a1a;
}
/* AI气泡 */
.ai-dark .ai-msg-ai-bubble {
  background: #252525;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.08);
  color: #e0e0e0;
}

/* 打字动画 */
.ai-dark .ai-dot {
  background: #e0e0e0;
}

/* 输入区域 */
.ai-dark .ai-input-area {
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}
.ai-dark .ai-input {
  background: #252525;
  color: #e0e0e0;
  border-color: #404040;
}
.ai-dark .ai-input:focus {
  border-color: #4a9eff;
  box-shadow: 0 0 0 3px rgba(74, 158, 255, 0.15);
}
.ai-dark .ai-input::placeholder {
  color: #888;
}
/* 暗色模式：输入框蓝色边框行走光效 */
.ai-dark .ai-input-wrap.is-thinking::before {
  background: conic-gradient(
    from var(--ai-angle),
    transparent 0%,
    transparent 60%,
    rgba(74, 158, 255, 0.9) 72%,
    #4a9eff 84%,
    rgba(74, 158, 255, 0.9) 96%,
    transparent 100%
  );
}
.ai-dark .ai-input-wrap.is-thinking .ai-input {
  border-color: rgba(74, 158, 255, 0.4) !important;
  box-shadow: 0 0 12px rgba(74, 158, 255, 0.25) !important;
}
.ai-dark .ai-image-btn {
  background: #252525;
  color: #e0e0e0;
  border-color: rgba(255, 255, 255, 0.12);
}
.ai-dark .ai-image-btn:hover {
  background: #2a2a2a;
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.05);
}
.ai-dark .ai-send-icon-btn {
  background: #1a1a1a;
  color: #e0e0e0;
}
.ai-dark .ai-send-icon-btn:hover:not(.is-disabled):not(.is-stop) {
  background: #2a2a2a;
}
.ai-dark .ai-send-icon-btn.is-disabled {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.4);
}
.ai-dark .ai-send-icon-btn.is-stop {
  background: #e0e0e0;
  color: #1a1a1a;
  border-color: rgba(255, 255, 255, 0.15);
}

/* 图片生成模式按钮 */
.ai-dark .ai-mode-btn {
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.9);
}
.ai-dark .ai-mode-btn.active {
  background: #e0e0e0;
  color: #1a1a1a;
  border-color: #e0e0e0;
}

/* 模型选择 */
.ai-dark .ai-model-bar {
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}
.ai-dark .ai-model-selector {
  background: #2a2a2a;
  border-color: rgba(255, 255, 255, 0.25);
  color: #fff;
}
.ai-dark .ai-model-selector:hover {
  background: #333;
  border-color: rgba(255, 255, 255, 0.25);
}
.ai-dark .ai-model-dropdown {
  background: #383838;
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.5);
}
/* 深色模式下未选中模型项文字用纯白 !important 防止被全局样式覆盖 */
.ai-dark .ai-model-item,
.ai-dark .ai-model-item-name {
  color: #fff !important;
}
.ai-dark .ai-model-item:hover {
  background: rgba(255, 255, 255, 0.1);
}
.ai-dark .ai-model-item.active {
  background: #e0e0e0;
  color: #1a1a1a;
}
.ai-dark .ai-model-item.active .ai-model-icon {
  color: #1a1a1a;
}
/* 深色模式下选中模型项名称也要变深色，否则子元素的color:#e0e0e0优先于父元素的color:#1a1a1a导致白字白底不可见 */
.ai-dark .ai-model-item.active .ai-model-item-name {
  color: #1a1a1a !important;
}
.ai-dark .ai-model-icon {
  color: #fff !important;
}
.ai-dark .ai-model-group-label {
  color: #ccc;
}
.ai-dark .ai-model-switch-icon {
  opacity: 0.85;
}

/* 侧边栏 */
.ai-dark.ai-sidebar {
  background: #1a1a1a;
  color: #e0e0e0;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.06), 0 8px 32px rgba(0, 0, 0, 0.4);
}
.ai-dark .ai-sidebar-header {
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(30, 30, 30, 0.8);
}
.ai-dark .ai-sidebar-title,
.ai-dark .ai-sidebar-newchat,
.ai-dark .ai-sidebar-item-title {
  color: #e0e0e0;
}
.ai-dark .ai-sidebar-close {
  color: #aaa;
}
.ai-dark .ai-sidebar-close:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #e0e0e0;
}
.ai-dark .ai-sidebar-newchat {
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}
.ai-dark .ai-sidebar-newchat:hover,
.ai-dark .ai-sidebar-item:hover {
  background: rgba(255, 255, 255, 0.04);
}
.ai-dark .ai-sidebar-empty {
  color: #888;
}
.ai-dark .ai-sidebar-item-icon {
  color: rgba(255, 255, 255, 0.6);
}
.ai-dark .ai-sidebar-item-time {
  color: #888;
}
.ai-dark .ai-sidebar-item-delete {
  color: #888;
}
.ai-dark .ai-sidebar-item-delete:hover {
  background: rgba(244, 67, 54, 0.15);
  color: #f44336;
}

/* 思考过程 */
.ai-dark .ai-reasoning {
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.08);
}
.ai-dark .ai-reasoning-header {
  color: #e0e0e0;
}
.ai-dark .ai-reasoning-header:hover {
  background: rgba(255, 255, 255, 0.04);
}
.ai-dark .ai-reasoning-toggle {
  color: #888;
}
.ai-dark .ai-reasoning-body {
  color: #aaa;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

/* Markdown 深色穿透 */
.ai-dark .ai-markdown :deep(th),
.ai-dark .ai-reasoning-body :deep(th) {
  background: rgba(255, 255, 255, 0.06);
}
.ai-dark .ai-markdown :deep(th),
.ai-dark .ai-markdown :deep(td),
.ai-dark .ai-reasoning-body :deep(th),
.ai-dark .ai-reasoning-body :deep(td) {
  border-color: rgba(255, 255, 255, 0.15);
}
.ai-dark .ai-markdown :deep(code),
.ai-dark .ai-reasoning-body :deep(code) {
  background: rgba(255, 255, 255, 0.06);
}
.ai-dark .ai-markdown :deep(pre),
.ai-dark .ai-reasoning-body :deep(pre) {
  background: rgba(0, 0, 0, 0.3);
}
.ai-dark .ai-markdown :deep(blockquote),
.ai-dark .ai-reasoning-body :deep(blockquote) {
  border-left-color: rgba(255, 255, 255, 0.2);
  color: #999;
}
.ai-dark .ai-markdown :deep(a),
.ai-dark .ai-reasoning-body :deep(a) {
  color: #4a9eff;
}
.ai-dark .ai-markdown :deep(.ai-action-btn) {
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.7);
}
.ai-dark .ai-markdown :deep(.ai-action-btn:hover) {
  background: #e0e0e0;
  color: #1a1a1a;
  border-color: #e0e0e0;
}

/* 生成的图片边框 */
.ai-dark .ai-generated-image {
  border-color: rgba(255, 255, 255, 0.12);
}
.ai-dark .ai-image-action-btn {
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.6);
}
.ai-dark .ai-image-action-btn:hover {
  background: #e0e0e0;
  color: #1a1a1a;
  border-color: #e0e0e0;
}

/* 设置下拉菜单深色 */
.ai-dark .ai-settings-dropdown {
  background: #383838;
  border: 1px solid rgba(255, 255, 255, 0.15);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.5);
}
.ai-dark .ai-settings-label {
  color: #aaa;
}
.ai-dark .ai-settings-item {
  color: #fff;
}
.ai-dark .ai-settings-item:hover {
  background: rgba(255, 255, 255, 0.1);
}
.ai-dark .ai-settings-item.active {
  color: #fff;
  font-weight: 600;
}
.ai-dark .ai-settings-check {
  color: #fff;
}

/* ====== 终极兜底：深色模式强制所有文字浅色 ====== */
/* 用 !important + 通配符一把梭，根治"每次改代码忘加 dark 覆盖 → 文字继承全局 #1a1a1a 变黑"的顽疾。
   下方更具体的规则靠更高 CSS 优先级自动覆盖本兜底，无需手动排除。 */
.ai-dark,
.ai-dark * {
  color: #e0e0e0 !important;
}

/* 浅色背景上的元素需要深色文字（选中项、用户气泡等白底元素） */
.ai-dark .ai-msg-user .ai-msg-bubble,
.ai-dark .ai-msg-user .ai-msg-bubble *,
.ai-dark .ai-mode-btn.active,
.ai-dark .ai-model-item.active,
.ai-dark .ai-model-item.active *,
.ai-dark .ai-image-action-btn:hover,
.ai-dark .ai-overlay-download-btn {
  color: #1a1a1a !important;
}

/* 禁用按钮半透明 */
.ai-dark .ai-send-icon-btn.is-disabled {
  color: rgba(255, 255, 255, 0.3) !important;
}

/* placeholder 略暗 */
.ai-dark .ai-input::placeholder {
  color: #888 !important;
}

/* 链接保持蓝色 */
.ai-dark .ai-markdown :deep(a),
.ai-dark .ai-reasoning-body :deep(a) {
  color: #4a9eff !important;
}

/* 二级辅助文字略暗 */
.ai-dark .ai-welcome-hint,
.ai-dark .ai-sidebar-item-time,
.ai-dark .ai-sidebar-item-delete,
.ai-dark .ai-sidebar-empty,
.ai-dark .ai-reasoning-toggle,
.ai-dark .ai-settings-label,
.ai-dark .ai-model-group-label,
.ai-dark .ai-reasoning-body {
  color: #999 !important;
}
</style>
