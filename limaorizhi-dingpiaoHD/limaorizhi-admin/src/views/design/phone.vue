<template>
  <div class="page-container">
    <div class="card-box" v-loading="loading">
      <div class="page-title">电话设置</div>
      <el-form :model="form" label-width="120px" style="max-width: 560px">
        <el-form-item label="客服热线电话">
          <el-input v-model="form.customer_service_phone" placeholder="请输入客服电话号码" clearable>
            <template #prepend>电话</template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
          <el-button @click="handleCopyPhone">复制号码</el-button>
        </el-form-item>
      </el-form>

      <el-alert title="小程序「我的」页面中的「电话热线」菜单将使用此号码，用户点击后直接拨打电话。" type="info" :closable="false" show-icon style="max-width: 560px; margin-top: 8px" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { configApi } from '@/api'

const loading = ref(false)
const saving = ref(false)
const form = reactive({
  customer_service_phone: '',
})

const loadData = async () => {
  loading.value = true
  try {
    const res: any = await configApi.get()
    const data = res.data || {}
    form.customer_service_phone = data.customer_service_phone || ''
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    await configApi.update({ configs: { customer_service_phone: form.customer_service_phone.trim() } })
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleCopyPhone = async () => {
  const phone = form.customer_service_phone.trim()
  if (!phone) {
    ElMessage.warning('请先输入电话号码')
    return
  }
  // 桌面浏览器不支持tel:协议，改为复制到剪贴板
  try {
    await navigator.clipboard.writeText(phone)
    ElMessage.success(`电话号码已复制：${phone}`)
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

onMounted(loadData)
</script>

<style scoped>
.page-title { font-size: 18px; font-weight: 600; margin-bottom: 20px; color: #1a1a1a }
</style>
