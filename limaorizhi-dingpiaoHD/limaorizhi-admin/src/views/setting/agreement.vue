<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box" v-loading="loading">
      <div class="page-title">协议政策</div>
      <div class="page-desc">管理小程序登录页展示的《用户协议》和《隐私政策》内容。修改后即时生效，小程序端将展示最新内容。</div>

      <el-tabs v-model="activeTab" class="agreement-tabs">
        <el-tab-pane label="用户协议" name="user_agreement">
          <el-input
            v-model="form.user_agreement"
            type="textarea"
            :rows="24"
            placeholder="请输入用户协议内容..."
            resize="vertical"
          />
          <div class="field-hint">
            <el-icon><InfoFilled /></el-icon>
            用户协议展示在小程序登录页《用户协议》链接中，用于告知用户购票服务条款。
          </div>
        </el-tab-pane>
        <el-tab-pane label="隐私政策" name="privacy_policy">
          <el-input
            v-model="form.privacy_policy"
            type="textarea"
            :rows="24"
            placeholder="请输入隐私政策内容..."
            resize="vertical"
          />
          <div class="field-hint">
            <el-icon><InfoFilled /></el-icon>
            隐私政策展示在小程序登录页《隐私政策》链接中，用于告知用户个人信息收集与保护规则。
          </div>
        </el-tab-pane>
      </el-tabs>

      <div class="action-bar">
        <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
        <el-button @click="loadData">重新加载</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import { agreementApi } from '@/api'

const loading = ref(false)
const saving = ref(false)
const activeTab = ref('user_agreement')
const form = reactive<any>({
  user_agreement: '',
  privacy_policy: '',
})

const loadData = async () => {
  loading.value = true
  try {
    const res: any = await agreementApi.get()
    const data = res.data || {}
    form.user_agreement = data.user_agreement || ''
    form.privacy_policy = data.privacy_policy || ''
  } catch (e) {
    // 错误已由 request 拦截器提示
    void e
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!form.user_agreement.trim()) {
    ElMessage.warning('用户协议内容不能为空')
    activeTab.value = 'user_agreement'
    return
  }
  if (!form.privacy_policy.trim()) {
    ElMessage.warning('隐私政策内容不能为空')
    activeTab.value = 'privacy_policy'
    return
  }
  saving.value = true
  try {
    await agreementApi.update({
      configs: {
        user_agreement: form.user_agreement,
        privacy_policy: form.privacy_policy,
      },
    })
    ElMessage.success('保存成功')
  } catch (e) {
    // 错误已由 request 拦截器提示
    void e
  } finally {
    saving.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.page-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 8px;
  color: #1a1a1a;
}
.page-desc {
  font-size: 13px;
  color: #909399;
  margin-bottom: 20px;
  line-height: 1.6;
}
.agreement-tabs {
  margin-bottom: 16px;
}
.field-hint {
  font-size: 12px;
  color: #909399;
  line-height: 1.6;
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 4px;
}
.action-bar {
  margin-top: 16px;
  display: flex;
  gap: 12px;
}
</style>
