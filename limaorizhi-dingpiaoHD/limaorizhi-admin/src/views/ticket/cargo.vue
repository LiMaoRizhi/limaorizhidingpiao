<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box" v-loading="loading">
      <div class="page-title">托运管理</div>

      <!-- 运费规则说明 -->
      <el-alert type="info" :closable="false" style="margin-bottom: 20px">
        <template #title>运费计算规则</template>
        基础运费 = max(最低运费, 线路距离 × 每公里运费)；免费重量以内不额外收费，超出部分按超重费/kg计算；总运费向上取整。
      </el-alert>

      <!-- 运费配置表单 -->
      <el-form :model="form" label-width="160px" style="max-width: 600px">
        <el-divider content-position="left">运费配置</el-divider>
        <el-form-item label="每公里运费(元)">
          <el-input-number v-model="form.cargo_price_per_km" :min="0" :precision="2" :step="0.1" />
          <span class="form-hint">线路距离每公里的运费单价</span>
        </el-form-item>
        <el-form-item label="最低运费(元)">
          <el-input-number v-model="form.cargo_min_fee" :min="0" :precision="0" />
          <span class="form-hint">无论距离多短，最低收取的运费</span>
        </el-form-item>
        <el-form-item label="免费重量(kg)">
          <el-input-number v-model="form.cargo_free_weight" :min="0" :precision="1" />
          <span class="form-hint">此重量内不收超重费</span>
        </el-form-item>
        <el-form-item label="超重费(元/kg)">
          <el-input-number v-model="form.cargo_extra_weight_fee" :min="0" :precision="0" />
          <span class="form-hint">超出免费重量部分每公斤加收</span>
        </el-form-item>
        <el-form-item label="最大重量限制(kg)">
          <el-input-number v-model="form.cargo_max_weight" :min="1" :max="200" :precision="0" />
          <span class="form-hint">单笔托运最大重量</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { configApi } from '@/api'
const loading = ref(false)
const saving = ref(false)
const form = reactive<any>({
  cargo_price_per_km: 0.5,
  cargo_min_fee: 10,
  cargo_free_weight: 5,
  cargo_extra_weight_fee: 3,
  cargo_max_weight: 50,
})
const loadData = async () => {
  loading.value = true
  try {
    const res: any = await configApi.get()
    const data = res.data || {}
    form.cargo_price_per_km = Number(data.cargo_price_per_km) || 0.5
    form.cargo_min_fee = Number(data.cargo_min_fee) || 10
    form.cargo_free_weight = Number(data.cargo_free_weight) || 5
    form.cargo_extra_weight_fee = Number(data.cargo_extra_weight_fee) || 3
    form.cargo_max_weight = Number(data.cargo_max_weight) || 50
  } finally { loading.value = false }
}
const handleSave = async () => {
  if (saving.value) return
  saving.value = true
  try {
    const configs: Record<string, string> = {
      cargo_price_per_km: String(form.cargo_price_per_km),
      cargo_min_fee: String(form.cargo_min_fee),
      cargo_free_weight: String(form.cargo_free_weight),
      cargo_extra_weight_fee: String(form.cargo_extra_weight_fee),
      cargo_max_weight: String(form.cargo_max_weight),
    }
    await configApi.update({ configs }); ElMessage.success('保存成功')
  } catch (e) {
    ElMessage.error('保存失败，请重试')
  } finally { saving.value = false }
}
onMounted(loadData)
</script>
<style scoped>
.page-title { font-size: 18px; font-weight: 600; margin-bottom: 20px; color: #1a1a1a }
.form-hint { margin-left: 12px; color: #999; font-size: 13px }
</style>
