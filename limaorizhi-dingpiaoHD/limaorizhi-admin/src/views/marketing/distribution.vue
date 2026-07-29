<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.user_id" placeholder="用户ID" clearable style="width: 120px" @keyup.enter="handleSearch" />
        <el-input v-model="query.coupon_id" placeholder="优惠券ID" clearable style="width: 120px" @keyup.enter="handleSearch" />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="未使用" :value="0" />
          <el-option label="已使用" :value="1" />
          <el-option label="已过期" :value="2" />
        </el-select>
        <el-button type="primary" @click="handleSearch">查询</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column label="用户" min-width="140">
          <template #default="{ row }">{{ row.user?.nickname || '-' }}（{{ row.user?.phone || '-' }}）</template>
        </el-table-column>
        <el-table-column label="优惠券" min-width="140">
          <template #default="{ row }">{{ row.coupon?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="发放时间" width="160"><template #default="{ row }">{{ formatTime(row.issued_at) }}</template></el-table-column>
        <el-table-column label="过期时间" width="160"><template #default="{ row }">{{ formatTime(row.expired_at) }}</template></el-table-column>
        <el-table-column label="使用时间" width="160"><template #default="{ row }">{{ row.used_at ? formatTime(row.used_at) : '-' }}</template></el-table-column>
        <el-table-column label="关联订单" width="100"><template #default="{ row }">{{ row.order_id ? row.order_id : '-' }}</template></el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { userCouponApi } from '@/api'
import { formatTime } from '@/utils/format'
const list = ref<any[]>([]), total = ref(0), loading = ref(false)
const query = reactive({ user_id: '', coupon_id: '', status: '' as any, page: 1, page_size: 20 })
const statusText = (s: number) => ({ 0: '未使用', 1: '已使用', 2: '已过期' } as any)[s] || '未知'
const statusTag = (s: number) => ({ 0: 'success', 1: 'info', 2: 'warning' } as any)[s] || 'info'
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await userCouponApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px; flex-wrap: wrap }.pagination { display: flex; justify-content: flex-end; margin-top: 16px }</style>
