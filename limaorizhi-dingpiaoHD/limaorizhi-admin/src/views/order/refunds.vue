<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.order_no" placeholder="订单号" clearable style="width: 180px" @keyup.enter="handleSearch" />
        <el-date-picker v-model="query.start_date" type="date" value-format="YYYY-MM-DD" placeholder="开始日期" style="width: 150px" />
        <el-date-picker v-model="query.end_date" type="date" value-format="YYYY-MM-DD" placeholder="结束日期" style="width: 150px" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe size="small">
        <el-table-column prop="refund_no" label="退款单号" width="170" show-overflow-tooltip />
        <el-table-column label="订单号" width="160" show-overflow-tooltip><template #default="{ row }">{{ row.order?.order_no }}</template></el-table-column>
        <el-table-column label="联系人/电话" width="130">
          <template #default="{ row }">
            <div>{{ row.order?.order_type === 2 ? row.order?.sender_name : row.order?.contact_name }}</div>
            <div class="row-sub">{{ row.order?.order_type === 2 ? row.order?.sender_phone : row.order?.contact_phone }}</div>
          </template>
        </el-table-column>
        <el-table-column label="退款金额" width="80"><template #default="{ row }">¥{{ row.amount }}</template></el-table-column>
        <el-table-column label="状态" width="70">
          <template #default="{ row }">
            <el-tag :type="refundStatusType(row.status)" size="small">{{ refundStatusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="退款原因" min-width="120" show-overflow-tooltip />
        <el-table-column label="退款时间" width="140"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { refundApi } from '@/api'
const list = ref<any[]>([]), total = ref(0), loading = ref(false)
const query = reactive({ order_no: '', start_date: '', end_date: '', page: 1, page_size: 20 })
// 格式化日期时间：后端输出 "2006-01-02 15:04:05"，取前16位显示 "YYYY-MM-DD HH:mm"
const formatDateTime = (dt: string) => {
  if (!dt) return '-'
  return dt.slice(0, 16).replace('T', ' ')
}
const refundStatusText = (s: number) => ({ 0: '处理中', 1: '已退款', 2: '已拒绝' } as any)[s] || '未知'
const refundStatusType = (s: number) => ({ 0: 'warning', 1: 'success', 2: 'danger' } as any)[s] || 'info'
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await refundApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
onMounted(loadData)
</script>
<style scoped>
.toolbar { display: flex; gap: 8px; flex-wrap: wrap }
.pagination { display: flex; justify-content: flex-end; margin-top: 16px }
.row-sub { font-size: 12px; color: #999; }
</style>
