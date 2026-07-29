<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container" v-loading="loading">
    <!-- 工具栏 -->
    <div class="card-box">
      <div class="toolbar">
        <el-button type="primary" @click="loadData">刷新</el-button>
        <el-button @click="handleReset">重置计数器</el-button>
        <span class="toolbar-hint">数据为进程级累计，重启后端服务会清零</span>
      </div>
    </div>

    <!-- 缓存概览 -->
    <div class="card-box">
      <div class="card-title">缓存概览</div>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="缓存命中率">
          <span class="metric-key">{{ hitRatePercent }}%</span>
        </el-descriptions-item>
        <el-descriptions-item label="统计起始时间">{{ formatTime(stats.stats?.since) }}</el-descriptions-item>
        <el-descriptions-item label="缓存命中次数">{{ stats.stats?.cache_hits || 0 }}</el-descriptions-item>
        <el-descriptions-item label="缓存未命中次数">{{ stats.stats?.cache_misses || 0 }}</el-descriptions-item>
      </el-descriptions>
    </div>

    <!-- API 调用统计 -->
    <div class="card-box">
      <div class="card-title">API 调用统计</div>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="API 调用次数（含重试）">{{ stats.stats?.api_calls || 0 }}</el-descriptions-item>
        <el-descriptions-item label="API 失败次数">
          <span :class="{ 'metric-warn': (stats.stats?.api_errors || 0) > 0 }">
            {{ stats.stats?.api_errors || 0 }}
          </span>
        </el-descriptions-item>
        <el-descriptions-item label="缓存写入次数">{{ stats.stats?.cache_writes || 0 }}</el-descriptions-item>
        <el-descriptions-item label="缓存主动删除次数">{{ stats.stats?.cache_deletes || 0 }}</el-descriptions-item>
      </el-descriptions>
    </div>

    <!-- 成本估算 -->
    <div class="card-box">
      <div class="card-title">成本估算</div>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="累计节省成本（估算）">
          ¥ {{ (stats.saved_cost_yuan || 0).toFixed(2) }}
        </el-descriptions-item>
        <el-descriptions-item label="累计花费成本（估算）">
          ¥ {{ (stats.spent_cost_yuan || 0).toFixed(2) }}
        </el-descriptions-item>
        <el-descriptions-item label="单价">{{ stats.price_per_call || 0.3 }} 元/次</el-descriptions-item>
        <el-descriptions-item label="说明">{{ stats.note }}</el-descriptions-item>
      </el-descriptions>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { idCardVerifyApi, type IDCardVerifyStatsResponse } from '@/api'

const stats = ref<Partial<IDCardVerifyStatsResponse>>({})
const loading = ref(false)

const hitRatePercent = computed(() => {
  const rate = stats.value.stats?.hit_rate
  if (rate === undefined || rate === null) return '0.00'
  return (rate * 100).toFixed(2)
})

const formatTime = (iso?: string) => {
  if (!iso) return '-'
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}

const loadData = async () => {
  loading.value = true
  try {
    const res: any = await idCardVerifyApi.stats()
    stats.value = res.data || {}
  } finally {
    loading.value = false
  }
}

const handleReset = async () => {
  try {
    await ElMessageBox.confirm(
      '确认重置所有计数器为 0？\n注意：重置只清零计数器，不影响 Redis 缓存本身。统计起始时间会更新为当前时间。',
      '重置统计',
      { type: 'warning', confirmButtonText: '确认重置', cancelButtonText: '取消' }
    )
  } catch {
    return // 用户取消
  }
  try {
    await idCardVerifyApi.resetStats()
    ElMessage.success('计数器已重置')
    await loadData()
  } catch {
    // request 工具会自动 toast 错误消息
  }
}

onMounted(loadData)
</script>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap }
.toolbar-hint { color: #909399; font-size: 12px }
.card-title { font-size: 16px; font-weight: 600; color: #303133; margin-bottom: 16px }
.metric-key { font-size: 18px; font-weight: 600; color: #303133 }
.metric-warn { color: #f56c6c; font-weight: 600 }
</style>
