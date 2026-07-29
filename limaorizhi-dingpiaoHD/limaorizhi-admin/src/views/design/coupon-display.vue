<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box" v-loading="loading">
      <div class="page-title">优惠券展示</div>
      <el-alert title="勾选要在小程序首页展示的优惠券，用户在首页可看到优惠券入口并领取。未勾选的优惠券不会在首页展示。" type="info" :closable="false" show-icon style="margin-bottom: 20px; max-width: 700px" />
      <div style="max-width: 700px">
        <div v-if="couponList.length === 0" style="text-align: center; padding: 40px 0; color: #999">
          暂无可选优惠券，请先在「营销 - 优惠券管理」中创建
        </div>
        <el-checkbox-group v-model="selectedIds" v-else>
          <div v-for="item in couponList" :key="item.id" class="coupon-select-item">
            <el-checkbox :value="item.id">
              <div class="coupon-info">
                <el-tag :type="couponTypeTag(item.type)" size="small">{{ couponTypeText(item.type) }}</el-tag>
                <span class="coupon-name">{{ item.name }}</span>
                <span class="coupon-value">
                  <template v-if="item.type === 1 || item.type === 3">¥{{ item.discount_value }}</template>
                  <template v-else>{{ item.discount_value }}折</template>
                </span>
                <el-tag :type="item.status === 1 ? 'success' : 'info'" size="small">{{ item.status === 1 ? '启用' : '停用' }}</el-tag>
              </div>
            </el-checkbox>
          </div>
        </el-checkbox-group>
        <div style="margin-top: 20px" v-if="couponList.length > 0">
          <el-button type="primary" @click="handleSave" :loading="saving">保存配置</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { couponApi, configApi } from '@/api'

const loading = ref(false), saving = ref(false)
const couponList = ref<any[]>([])
const selectedIds = ref<number[]>([])

const couponTypeText = (t: number) => ({ 1: '满减券', 2: '折扣券', 3: '固定金额' } as any)[t] || '未知'
const couponTypeTag = (t: number) => ({ 1: 'danger', 2: 'warning', 3: 'primary' } as any)[t] || 'info'

const loadData = async () => {
  loading.value = true
  try {
    // 并行加载优惠券列表和当前配置
    const [couponRes, configRes]: any[] = await Promise.all([
      couponApi.list({ page: 1, page_size: 100 }),
      configApi.get()
    ])
    couponList.value = couponRes.data.list || []
    const ids = (configRes.data.homepage_coupon_ids || '').split(',').filter((s: string) => s.trim())
    selectedIds.value = ids.map((s: string) => parseInt(s.trim())).filter((n: number) => !isNaN(n) && n > 0)
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    const idStr = selectedIds.value.join(',')
    await configApi.update({ configs: { homepage_coupon_ids: idStr } })
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.page-title { font-size: 18px; font-weight: 600; margin-bottom: 20px; color: #1a1a1a }
.coupon-select-item {
  padding: 12px 16px;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  margin-bottom: 10px;
  transition: border-color 0.2s;
}
.coupon-select-item:hover { border-color: #409eff }
.coupon-info {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin-left: 8px;
}
.coupon-name { font-size: 14px; font-weight: 500; min-width: 120px }
.coupon-value { color: #f56c6c; font-weight: 600 }
</style>
