<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <h3 style="margin-bottom: 24px">修改密码</h3>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
        style="max-width: 480px"
      >
        <el-form-item label="当前账号">
          <el-input :model-value="adminName" disabled />
        </el-form-item>
        <el-form-item label="原密码" prop="old_password">
          <el-input
            v-model="form.old_password"
            type="password"
            placeholder="请输入原密码"
            show-password
          />
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="form.new_password"
            type="password"
            placeholder="请输入新密码（至少8位，含字母和数字）"
            show-password
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm_password">
          <el-input
            v-model="form.confirm_password"
            type="password"
            placeholder="请再次输入新密码"
            show-password
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSubmit" :loading="loading">
            确认修改
          </el-button>
          <el-button @click="handleReset">重置</el-button>
          <el-link type="primary" @click="showForgotDialog = true" style="margin-left: 16px">
            忘记密码？
          </el-link>
        </el-form-item>
      </el-form>
    </div>

    <div class="card-box">
      <h3 style="margin-bottom: 16px">账号信息</h3>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="用户名">{{ adminInfo?.username || '-' }}</el-descriptions-item>
        <el-descriptions-item label="姓名">{{ adminInfo?.real_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="角色">{{ adminInfo?.role === 1 ? '超级管理员' : '管理员' }}</el-descriptions-item>
      </el-descriptions>
    </div>

    <!-- 忘记密码弹窗 -->
    <el-dialog v-model="showForgotDialog" title="忘记密码" width="460px">
      <!-- 超级管理员：可以重置任意管理员密码 -->
      <div v-if="isSuperAdmin">
        <el-alert
          title="您是超级管理员，可以重置任意管理员账号的密码"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        />
        <el-form label-width="90px">
          <el-form-item label="选择账号">
            <el-select v-model="resetForm.adminId" placeholder="请选择管理员" style="width: 100%">
              <el-option
                v-for="item in adminList"
                :key="item.id"
                :label="`${item.username}（${item.real_name || '未命名'}）`"
                :value="item.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="新密码">
            <el-input
              v-model="resetForm.password"
              type="password"
              placeholder="请输入新密码（至少8位，含字母和数字）"
              show-password
            />
          </el-form-item>
          <el-form-item label="确认密码">
            <el-input
              v-model="resetForm.confirmPassword"
              type="password"
              placeholder="请再次输入新密码"
              show-password
            />
          </el-form-item>
        </el-form>
      </div>

      <!-- 普通管理员：提示联系超级管理员 -->
      <div v-else>
        <el-alert
          title="忘记密码"
          type="warning"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        />
        <p style="line-height: 1.8; color: #595959; font-size: 14px">
          如果您忘记了密码，请联系<b>超级管理员</b>为您重置密码。
        </p>
        <p style="line-height: 1.8; color: #595959; font-size: 14px; margin-top: 8px">
          超级管理员可以在「管理 → 管理员管理」页面或此页面直接重置您的密码。
        </p>
      </div>

      <template #footer>
        <el-button @click="showForgotDialog = false">取消</el-button>
        <el-button v-if="isSuperAdmin" type="primary" @click="handleResetPassword" :loading="resetLoading">
          确认重置
        </el-button>
        <el-button v-else @click="showForgotDialog = false">知道了</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { systemApi, adminApi } from '@/api'

const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

const adminInfo = computed(() => authStore.adminInfo)
const adminName = computed(() => authStore.adminInfo?.real_name || authStore.adminInfo?.username || '管理员')
const isSuperAdmin = computed(() => authStore.adminInfo?.role === 1)

const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

const rules: FormRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '密码至少8位', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        if (!/(?=.*[a-zA-Z])(?=.*\d)/.test(value)) {
          callback(new Error('密码必须包含字母和数字'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        if (value !== form.new_password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      await systemApi.changePassword({
        old_password: form.old_password,
        new_password: form.new_password,
      })
      ElMessage.success('密码修改成功，请重新登录')
      authStore.logout()
      setTimeout(() => {
        window.location.href = '/login'
      }, 1500)
    } catch (e: any) {
      ElMessage.error(e.message || '修改失败')
    } finally {
      loading.value = false
    }
  })
}

const handleReset = () => {
  formRef.value?.resetFields()
}

// 忘记密码功能
const showForgotDialog = ref(false)
const resetLoading = ref(false)
const adminList = ref<any[]>([])
const resetForm = reactive({
  adminId: null as number | null,
  password: '',
  confirmPassword: '',
})

// 开弹窗时把管理员列表拉出来
const loadAdminList = async () => {
  try {
    const res = await adminApi.list({ page: 1, page_size: 100 })
    adminList.value = res.data.list || []
  } catch (e) {
    // ignore
  }
}

const onDialogOpen = () => {
  if (isSuperAdmin.value) {
    loadAdminList()
  }
}

watch(showForgotDialog, (val) => {
  if (val) onDialogOpen()
})

const handleResetPassword = async () => {
  if (!resetForm.adminId) {
    ElMessage.warning('请选择管理员')
    return
  }
  if (resetForm.password.length < 8) {
    ElMessage.warning('密码至少8位')
    return
  }
  if (!/(?=.*[a-zA-Z])(?=.*\d)/.test(resetForm.password)) {
    ElMessage.warning('密码必须包含字母和数字')
    return
  }
  if (resetForm.password !== resetForm.confirmPassword) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }
  resetLoading.value = true
  try {
    await adminApi.resetPassword(resetForm.adminId, { password: resetForm.password })
    ElMessage.success('密码重置成功')
    showForgotDialog.value = false
    resetForm.adminId = null
    resetForm.password = ''
    resetForm.confirmPassword = ''
  } catch (e: any) {
    ElMessage.error(e.message || '重置失败')
  } finally {
    resetLoading.value = false
  }
}
</script>

<style scoped>
</style>
