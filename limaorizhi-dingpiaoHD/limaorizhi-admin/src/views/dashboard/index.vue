<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container dashboard-layout">
    <!-- 左侧手机展示列（入场旋转后停留） -->
    <aside class="phone-column">
      <PhoneShowcase :show-entrance-spin="shouldSpin" />
    </aside>
    <!-- 右侧统计流 -->
    <div class="stats-column">
    <!-- 数据统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card" v-for="card in statCards" :key="card.label">
        <div class="stat-value">{{ card.value }}</div>
        <div class="stat-label">{{ card.label }}</div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="chart-row">
      <div class="card-box chart-card">
        <h3 class="chart-title">近30天订单趋势</h3>
        <div ref="trendChartRef" class="chart-container"></div>
      </div>
      <div class="card-box chart-card">
        <h3 class="chart-title">热门线路TOP10</h3>
        <div ref="rankChartRef" class="chart-container"></div>
      </div>
    </div>

    <!-- 今日按小时分布 -->
    <div class="card-box">
      <h3 class="chart-title">今日按小时订单分布</h3>
      <div ref="hourChartRef" class="chart-container-wide"></div>
    </div>

    <!-- 最近订单 -->
    <div class="card-box">
      <h3 class="chart-title">最近10笔订单</h3>
      <el-table :data="recentOrders" stripe style="width: 100%">
        <el-table-column prop="order_no" label="订单号" width="160" />
        <el-table-column prop="contact_name" label="联系人" width="100" />
        <el-table-column prop="contact_phone" label="联系电话" width="130" />
        <el-table-column prop="total_price" label="金额" width="100">
          <template #default="{ row }">¥{{ row.total_price }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
      </el-table>
    </div>
    </div><!-- stats-column -->
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import { dashboardApi } from '@/api'
import PhoneShowcase from '@/components/PhoneShowcase.vue'
import { formatTime } from '@/utils/format'
import { PHONE_SPIN_INTERVAL_MS } from '@/utils/constants'

// 手机模型入场旋转：每 6 小时首次进入首页才转，避免每次刷新都转圈
const shouldSpin = (() => {
  try {
    const last = Number(localStorage.getItem('limao_phone_spin_at') || '0')
    if (Date.now() - last >= PHONE_SPIN_INTERVAL_MS) {
      localStorage.setItem('limao_phone_spin_at', String(Date.now()))
      return true
    }
  } catch (e) {
    // localStorage 不可用时默认不转
  }
  return false
})()

const statCards = ref([
  { label: '今日订单', value: '0' },
  { label: '今日营收', value: '¥0' },
  { label: '今日新增用户', value: '0' },
  { label: '本周订单', value: '0' },
  { label: '本周营收', value: '¥0' },
  { label: '总用户数', value: '0' },
  { label: '退票率', value: '0%' },
  { label: '今日上座率', value: '0%' },
  { label: '今日活跃班次', value: '0' },
])

const recentOrders = ref<any[]>([])

const trendChartRef = ref<HTMLElement>()
const rankChartRef = ref<HTMLElement>()
const hourChartRef = ref<HTMLElement>()
let trendChart: echarts.ECharts | null = null
let rankChart: echarts.ECharts | null = null
let hourChart: echarts.ECharts | null = null

const statusText = (status: number) => {
  const map: Record<number, string> = { 0: '待支付', 1: '待出行', 2: '已完成', 3: '已退款', 4: '已取消', 5: '已核销' }
  return map[status] || '未知'
}
const statusTagType = (status: number) => {
  const map: Record<number, string> = { 0: 'danger', 1: 'success', 2: 'info', 3: 'danger', 4: 'info' }
  return map[status] || 'info'
}

const loadData = async () => {
  try {
    const res: any = await dashboardApi.stats()
    const s = res.data
    statCards.value = [
      { label: '今日订单', value: String(s.today_orders || 0) },
      { label: '今日营收', value: `¥${(s.today_revenue || 0).toFixed(2)}` },
      { label: '今日新增用户', value: String(s.today_users || 0) },
      { label: '本周订单', value: String(s.week_orders || 0) },
      { label: '本周营收', value: `¥${(s.week_revenue || 0).toFixed(2)}` },
      { label: '总用户数', value: String(s.total_users || 0) },
      { label: '退票率', value: `${(s.refund_rate || 0).toFixed(1)}%` },
      { label: '今日上座率', value: `${(s.seat_occupancy || 0).toFixed(1)}%` },
      { label: '今日活跃班次', value: String(s.active_trips || 0) },
    ]
    recentOrders.value = s.recent_orders || []

    await nextTick()
    // 订单趋势折线图
    const trendData = s.trend || []
    if (trendChartRef.value && trendData.length) {
      if (trendChart) trendChart.dispose()
      trendChart = echarts.init(trendChartRef.value)
      trendChart.setOption({
        tooltip: { trigger: 'axis' },
        grid: { left: 40, right: 40, top: 30, bottom: 30 },
        xAxis: {
          type: 'category',
          data: trendData.map((i: any) => i.date),
          axisLine: { lineStyle: { color: '#e0e0e0' } },
          axisLabel: { color: '#999' },
        },
        yAxis: {
          type: 'value',
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: { lineStyle: { color: '#f0f0f0', type: 'dashed' } },
          axisLabel: { color: '#999' },
        },
        series: [
          {
            name: '订单数',
            type: 'line',
            smooth: true,
            symbol: 'circle',
            symbolSize: 6,
            data: trendData.map((i: any) => i.orders),
            itemStyle: { color: '#409eff' },
            lineStyle: { width: 3, color: '#409eff' },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: 'rgba(64, 158, 255, 0.35)' },
                { offset: 1, color: 'rgba(64, 158, 255, 0.02)' },
              ]),
            },
          },
        ],
      })
    }
    // 热门线路柱状图
    const rankData = (s.route_rank || []).slice(0, 10)
    if (rankChartRef.value && rankData.length) {
      if (rankChart) rankChart.dispose()
      rankChart = echarts.init(rankChartRef.value)
      rankChart.setOption({
        tooltip: { trigger: 'axis' },
        grid: { left: 100, right: 40, top: 20, bottom: 30 },
        xAxis: {
          type: 'value',
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: { lineStyle: { color: '#f0f0f0', type: 'dashed' } },
          axisLabel: { color: '#999' },
        },
        yAxis: {
          type: 'category',
          data: rankData.map((i: any) => i.route_name).reverse(),
          axisLine: { lineStyle: { color: '#e0e0e0' } },
          axisTick: { show: false },
          axisLabel: { color: '#666' },
        },
        series: [
          {
            type: 'bar',
            barWidth: 16,
            data: rankData.map((i: any) => i.order_count).reverse(),
            itemStyle: {
              borderRadius: [0, 4, 4, 0],
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: '#6ba4ff' },
                { offset: 1, color: '#409eff' },
              ]),
            },
          },
        ],
      })
    }
    // 按小时分布
    const hourData = s.hourly || []
    if (hourChartRef.value && hourData.length) {
      if (hourChart) hourChart.dispose()
      hourChart = echarts.init(hourChartRef.value)
      hourChart.setOption({
        tooltip: { trigger: 'axis' },
        legend: { right: 20, top: 0, textStyle: { color: '#666' } },
        grid: { left: 50, right: 60, top: 40, bottom: 30 },
        xAxis: {
          type: 'category',
          data: hourData.map((i: any) => i.hour),
          axisLine: { lineStyle: { color: '#e0e0e0' } },
          axisLabel: { color: '#999' },
        },
        yAxis: [
          {
            type: 'value', name: '订单数',
            nameTextStyle: { color: '#999' },
            axisLine: { show: false },
            axisTick: { show: false },
            splitLine: { lineStyle: { color: '#f0f0f0', type: 'dashed' } },
            axisLabel: { color: '#999' },
          },
          {
            type: 'value', name: '营收', position: 'right',
            nameTextStyle: { color: '#999' },
            axisLine: { show: false },
            axisTick: { show: false },
            splitLine: { show: false },
            axisLabel: { color: '#999' },
          },
        ],
        series: [
          {
            name: '订单数', type: 'bar', barWidth: 14,
            data: hourData.map((i: any) => i.orders),
            itemStyle: { borderRadius: [4, 4, 0, 0], color: 'rgba(64, 158, 255, 0.45)' },
          },
          {
            name: '营收', type: 'line', smooth: true, symbol: 'circle', symbolSize: 6,
            yAxisIndex: 1,
            data: hourData.map((i: any) => i.revenue),
            itemStyle: { color: '#2c7de9' },
            lineStyle: { width: 3, color: '#2c7de9' },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: 'rgba(44, 125, 233, 0.3)' },
                { offset: 1, color: 'rgba(44, 125, 233, 0.02)' },
              ]),
            },
          },
        ],
      })
    }
  } catch (e) {
    console.error('加载仪表盘数据失败', e)
  }
}

const handleResize = () => {
  trendChart?.resize()
  rankChart?.resize()
  hourChart?.resize()
}

onMounted(() => {
  loadData()
  window.addEventListener('resize', handleResize)
})
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  rankChart?.dispose()
  hourChart?.dispose()
})
</script>

<style scoped>
/* 首页布局：左手机固定 + 右统计流 */
.dashboard-layout {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.phone-column {
  flex: 0 0 360px;
  position: sticky;
  top: 16px;
  z-index: 1;
}
.stats-column {
  flex: 1;
  min-width: 0;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}
.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  text-align: center;
}
.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1a1a1a;
  margin-bottom: 4px;
}
.stat-label {
  font-size: 13px;
  color: #999;
}

.chart-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}
.chart-card {
  flex: 1;
}
.chart-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 12px;
}
.chart-container {
  height: 280px;
}
.chart-container-wide {
  height: 220px;
}
</style>
