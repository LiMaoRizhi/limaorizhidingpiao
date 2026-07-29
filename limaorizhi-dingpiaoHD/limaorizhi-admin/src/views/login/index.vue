<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="login-wrapper">
    <!-- 左侧 Logo + 角色动画区域 -->
    <div class="login-left">
      <div class="logo-area">
        <img :src="brandStore.logoUrl" :alt="brandStore.systemName" class="login-logo" />
        <h1 class="login-title">{{ brandStore.systemName }}</h1>
        <p class="login-subtitle">管理后台</p>
      </div>

      <!-- 四个角色动画 -->
      <div class="characters-area">
        <AnimatedCharacters
          :isTyping="isTyping"
          :showPassword="showPassword"
          :passwordLength="form.password.length"
        />
      </div>

      <div class="login-footer">
        <p>售后微信：lihao68681818</p>
      </div>
    </div>

    <!-- 右侧 表单区域 -->
    <div class="login-right">
      <div class="login-form-wrapper">
        <h2 class="form-title">欢迎登录</h2>
        <p class="form-desc">请输入管理员账号和密码</p>
        <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin" class="login-form">
          <el-form-item prop="username">
            <el-input
              v-model="form.username"
              placeholder="账号"
              size="large"
              @focus="isTyping = true"
              @blur="isTyping = false"
            >
              <template #prefix>
                <span class="input-icon" v-html="icons.account"></span>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="密码"
              size="large"
              @focus="isTyping = true"
              @blur="isTyping = false"
              @keyup.enter="handleLogin"
            >
              <template #prefix>
                <span class="input-icon" v-html="icons.password"></span>
              </template>
              <template #suffix>
                <span class="pwd-toggle" v-html="showPassword ? icons.eyeOpen : icons.eyeClose" @click="showPassword = !showPassword"></span>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              size="large"
              class="login-btn"
              @click="handleLogin"
            >
              登录
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useBrandStore } from '@/stores/brand'
import type { FormInstance, FormRules } from 'element-plus'
import AnimatedCharacters from '@/components/AnimatedCharacters.vue'
import accountIcon from '@/assets/icons/account.svg?raw'
import passwordIcon from '@/assets/icons/password.svg?raw'
import eyeOpenIcon from '@/assets/icons/eye-open.svg?raw'
import eyeCloseIcon from '@/assets/icons/eye-close.svg?raw'

const icons = {
  account: accountIcon,
  password: passwordIcon,
  eyeOpen: eyeOpenIcon,
  eyeClose: eyeCloseIcon,
}

const router = useRouter()
const brandStore = useBrandStore()
const formRef = ref<FormInstance>()

const showPassword = ref(false)
const isTyping = ref(false)

const form = reactive({
  username: '',
  password: '',
})

const rules: FormRules = {
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const handleLogin = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    // 表单校验通过，将账号密码临时存入 sessionStorage，跳转到独立验证页
    sessionStorage.setItem('limao_pending_login', JSON.stringify({
      username: form.username,
      password: form.password,
    }))
    router.push('/verify')
  })
}
</script>

<style scoped>
/* 整页渐变背景 */
.login-wrapper {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
}

/* 左侧毛玻璃区域 */
.login-left {
  width: 58%;
  background: rgba(255, 255, 255, 0.35);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  border-right: 1px solid rgba(255, 255, 255, 0.5);
}

.logo-area {
  text-align: center;
  margin-top: 6vh;
}
.login-logo {
  width: 100px;
  height: 100px;
  margin-bottom: 16px;
  border-radius: 12px;
  object-fit: contain;
}
.login-title {
  color: #1a1a1a;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 6px;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.6);
}
.login-subtitle {
  color: #595959;
  font-size: 15px;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.5);
}

/* 角色动画区域 */
.characters-area {
  flex: 1;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  width: 100%;
  overflow: hidden;
}

.login-footer {
  margin-bottom: 24px;
  color: #595959;
  font-size: 13px;
}
.login-footer p {
  color: #595959;
}

/* 右侧表单区域 - 实色 */
.login-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
}
.login-form-wrapper {
  width: 360px;
  padding: 40px;
}
.form-title {
  font-size: 24px;
  font-weight: 700;
  color: #1a1a1a;
  margin-bottom: 8px;
}
.form-desc {
  color: #999;
  font-size: 14px;
  margin-bottom: 32px;
}
.login-form {
  width: 100%;
}

/* 输入框图标 */
.input-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
}
.input-icon :deep(svg) {
  width: 20px;
  height: 20px;
}
.pwd-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  cursor: pointer;
}
.pwd-toggle :deep(svg) {
  width: 20px;
  height: 20px;
}
.login-btn {
  width: 100%;
  background: #1a1a1a;
  border-color: #1a1a1a;
}
.login-btn:hover {
  background: #2c2c2c;
  border-color: #2c2c2c;
}
@media (max-width: 768px) {
  .login-left {
    display: none;
  }
  .login-form-wrapper {
    width: 90%;
  }
}
</style>
