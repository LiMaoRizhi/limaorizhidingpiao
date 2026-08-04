<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.keyword" placeholder="昵称/手机号" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="正常" :value="1" />
          <el-option label="已封禁" :value="0" />
        </el-select>
        <el-button type="primary" @click="handleSearch">查询</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column label="头像" width="70">
          <template #default="{ row }"><el-avatar :size="36" :src="row.avatar_url" /></template>
        </el-table-column>
        <el-table-column prop="nickname" label="昵称" min-width="120" />
        <el-table-column prop="phone" label="手机号" width="140" />
        <el-table-column prop="order_count" label="订单数" width="80" align="center" />
        <el-table-column label="消费金额" width="110" align="center">
          <template #default="{ row }">¥{{ row.total_amount?.toFixed(2) || '0.00' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '正常' : '已封禁' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="注册时间" width="170"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleDetail(row)">详情</el-button>
            <el-button size="small" :type="row.status === 1 ? 'danger' : 'success'" @click="handleToggleStatus(row)">
              {{ row.status === 1 ? '封禁' : '解禁' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>

    <!-- 用户详情弹窗 -->
    <el-dialog v-model="detailVisible" title="用户详情" width="800px" top="5vh">
      <div v-loading="detailLoading">
        <!-- 用户基本信息 + 统计 -->
        <div v-if="detail" class="detail-header">
          <div class="detail-avatar-wrap">
            <el-avatar :size="64" :src="detail.user.avatar_url" />
          </div>
          <div class="detail-info">
            <div class="detail-name">{{ detail.user.nickname || '未设置' }}</div>
            <div class="detail-meta">
              <span>手机号：{{ detail.user.phone || '-' }}</span>
              <el-tag :type="detail.user.status === 1 ? 'success' : 'danger'" size="small" style="margin-left: 8px">{{ detail.user.status === 1 ? '正常' : '已封禁' }}</el-tag>
            </div>
            <div class="detail-meta">
              <span>OpenID：{{ detail.user.openid ? detail.user.openid.substring(0, 16) + '...' : '-' }}</span>
            </div>
            <div class="detail-meta">
              <span>注册时间：{{ formatTime(detail.user.created_at) }}</span>
            </div>
          </div>
          <div class="detail-stats">
            <div class="stat-card">
              <div class="stat-num">{{ detail.order_count }}</div>
              <div class="stat-label">总订单</div>
            </div>
            <div class="stat-card">
              <div class="stat-num">¥{{ detail.total_amount?.toFixed(2) }}</div>
              <div class="stat-label">消费金额</div>
            </div>
            <div class="stat-card">
              <div class="stat-num">{{ detail.refund_count }}</div>
              <div class="stat-label">退票数</div>
            </div>
            <div class="stat-card">
              <div class="stat-num">{{ detail.cargo_count }}</div>
              <div class="stat-label">托运单</div>
            </div>
          </div>
        </div>

        <!-- 最近订单 -->
        <div v-if="detail && detail.orders.length > 0" style="margin-top: 20px">
          <div class="section-title">最近订单（{{ detail.orders.length }}条）</div>
          <el-table :data="detail.orders" size="small" stripe max-height="300">
            <el-table-column label="订单号" width="170" prop="order_no" show-overflow-tooltip />
            <el-table-column label="类型" width="70">
              <template #default="{ row }">
                <el-tag :type="row.order_type === 1 ? '' : 'warning'" size="small">{{ row.order_type === 1 ? '车票' : '托运' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="路线" min-width="160">
              <template #default="{ row }">
                {{ row.from_station?.name || row.from_station_name }} → {{ row.to_station?.name || row.to_station_name }}
              </template>
            </el-table-column>
            <el-table-column label="班次日期" width="120">
              <template #default="{ row }">{{ row.trip_date }} {{ row.departure_time }}</template>
            </el-table-column>
            <el-table-column label="金额" width="90" align="center">
              <template #default="{ row }">¥{{ row.total_price?.toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="orderStatusType(row.status)" size="small">{{ orderStatusText(row.status, row.order_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="下单时间" width="150">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 常用乘客 -->
        <div v-if="detail && detail.passengers.length > 0" style="margin-top: 20px">
          <div class="section-title">常用乘客（{{ detail.passengers.length }}人）</div>
          <el-table :data="detail.passengers" size="small" stripe>
            <el-table-column label="姓名" prop="name" width="120" />
            <el-table-column label="身份证号" prop="id_card_no" min-width="200" show-overflow-tooltip />
            <el-table-column label="手机号" prop="phone" width="140" />
            <el-table-column label="添加时间" width="150">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 空数据提示 -->
        <el-empty v-if="detail && detail.orders.length === 0 && detail.passengers.length === 0" description="该用户暂无订单和乘客记录" />
      </div>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi, type WxUser, type UserDetail } from '@/api'
import { formatTime } from '@/utils/format'
const list = ref<WxUser[]>([])
const total = ref(0)
const loading = ref(false)
const detailVisible = ref(false)
const detail = ref<UserDetail | null>(null)
const detailLoading = ref(false)
const query = reactive({ keyword: '', status: '' as string | number, page: 1, page_size: 20 })
const orderStatusText = (s: number, t?: number) => {
  if (t === 2) return ({ 0: '待支付', 1: '待运输', 2: '运输中', 3: '已到达', 4: '已取消', 5: '已取件' } as any)[s] || '未知'
  return ({ 0: '待支付', 1: '待出行', 2: '已完成', 3: '已退款', 4: '已取消', 5: '已核销' } as any)[s] || '未知'
}
const orderStatusType = (s: number) => ({ 0: 'warning', 1: 'success', 2: 'info', 3: 'danger', 4: 'info', 5: 'info' } as any)[s] || 'info'
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => {
  loading.value = true
  try {
    const res: any = await userApi.list(query)
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally { loading.value = false }
}
const handleDetail = async (row: WxUser) => {
  detailVisible.value = true
  detail.value = null
  detailLoading.value = true
  try {
    const res: any = await userApi.detail(row.id)
    detail.value = res.data
  } finally { detailLoading.value = false }
}
const handleToggleStatus = async (row: WxUser) => {
  const action = row.status === 1 ? '封禁' : '解禁'
  try { await ElMessageBox.confirm(`确认${action}该用户？`, '提示', { type: 'warning' }); await userApi.updateStatus(row.id, { status: row.status === 1 ? 0 : 1 }); ElMessage.success(`${action}成功`); loadData() } catch (e) {
    if (e !== 'cancel' && e?.toString() !== 'cancel') ElMessage.error(`${action}失败，请重试`)
  }
}
onMounted(loadData)
</script>
<style scoped>
.toolbar { display: flex; gap: 8px; flex-wrap: wrap }
.pagination { display: flex; justify-content: flex-end; margin-top: 16px }
.detail-header { display: flex; align-items: flex-start; gap: 20px }
.detail-avatar-wrap { flex-shrink: 0 }
.detail-info { flex: 1; min-width: 0 }
.detail-name { font-size: 18px; font-weight: 600; margin-bottom: 8px }
.detail-meta { color: #666; font-size: 13px; margin-bottom: 4px }
.detail-stats { display: flex; gap: 12px; flex-shrink: 0 }
.stat-card { text-align: center; padding: 8px 16px; background: #f5f7fa; border-radius: 8px; min-width: 80px }
.stat-num { font-size: 20px; font-weight: 700; color: #409eff }
.stat-label { font-size: 12px; color: #909399; margin-top: 2px }
.section-title { font-size: 14px; font-weight: 600; margin-bottom: 10px; color: #303133; border-left: 3px solid #409eff; padding-left: 8px }
</style>
