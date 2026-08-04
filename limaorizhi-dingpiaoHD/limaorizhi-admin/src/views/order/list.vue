<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.order_no" placeholder="订单号" clearable style="width: 160px" @keyup.enter="handleSearch" />
        <el-select v-model="query.order_type" placeholder="类型" clearable style="width: 90px">
          <el-option label="车票" :value="1" />
          <el-option label="托运" :value="2" />
        </el-select>
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 110px">
          <el-option label="待支付" :value="0" />
          <el-option label="待出行/运输" :value="1" />
          <el-option label="已完成/运输中" :value="2" />
          <el-option label="已退款/到达" :value="3" />
          <el-option label="已取消" :value="4" />
          <el-option label="已取件" :value="5" />
        </el-select>
        <el-input v-model="query.contact_phone" placeholder="联系电话" clearable style="width: 130px" @keyup.enter="handleSearch" />
        <el-date-picker v-model="query.start_date" type="date" value-format="YYYY-MM-DD" placeholder="开始" style="width: 130px" />
        <el-date-picker v-model="query.end_date" type="date" value-format="YYYY-MM-DD" placeholder="结束" style="width: 130px" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button @click="handleExport">导出CSV</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe size="small">
        <el-table-column prop="order_no" label="订单号" width="160" show-overflow-tooltip />
        <el-table-column label="类型" width="50">
          <template #default="{ row }">
            <el-tag :type="row.order_type === 2 ? 'warning' : 'primary'" size="small" effect="plain">{{ row.order_type === 2 ? '运' : '票' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="线路" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">{{ row.from_station_name || row.from_station?.name }} → {{ row.to_station_name || row.to_station?.name }}</template>
        </el-table-column>
        <el-table-column label="日期/时间" width="140">
          <template #default="{ row }">
            <div>{{ formatDate(row.trip_date) }}</div>
            <div class="row-time">{{ row.departure_time }}</div>
          </template>
        </el-table-column>
        <el-table-column label="人/重" width="60">
          <template #default="{ row }">{{ row.order_type === 2 ? row.weight + 'kg' : row.passenger_count + '人' }}</template>
        </el-table-column>
        <el-table-column label="金额" width="70"><template #default="{ row }">¥{{ row.total_price }}</template></el-table-column>
                <el-table-column label="下单时间" width="140">
                  <template #default="{ row }"><span class="row-time">{{ formatDateTime(row.created_at) }}</span></template>
                </el-table-column>
        <el-table-column label="联系人/电话" width="140">
          <template #default="{ row }">
            <div>{{ row.order_type === 2 ? row.sender_name : row.contact_name }}</div>
            <div class="row-time">{{ row.order_type === 2 ? row.sender_phone : row.contact_phone }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="70">
          <template #default="{ row }">
            <el-tag :type="orderStatusType(row.status)" size="small">{{ orderStatusText(row.status, row.order_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link @click="handleDetail(row)">详情</el-button>
            <!-- 待支付：管理员手动确认收款（线下付款场景） -->
            <el-button size="small" link type="success" v-if="row.status === 0" @click="handleUpdateStatus(row, 1, '确认该订单已收到付款？\n（仅用于线下付款，微信支付会自动确认）')">确认收款</el-button>
            <!-- 车票订单：退款（status=0也可能是微信已扣款但回调没到，点退款后端会先查单对账，所以叫"查单退款"） -->
            <el-button size="small" link type="warning" v-if="row.order_type === 1 && row.status <= 2" @click="handleRefund(row)">{{ row.status === 0 ? '查单退款' : '退款' }}</el-button>
            <!-- 托运订单：状态流转 -->
            <el-button size="small" link type="primary" v-if="row.order_type === 2 && row.status === 1" @click="handleUpdateStatus(row, 2, '确认开始运输该托运货物？')">开始运输</el-button>
            <el-button size="small" link type="success" v-if="row.order_type === 2 && row.status === 2" @click="handleUpdateStatus(row, 3, '确认货物已到达目的地？')">确认到达</el-button>
            <el-button size="small" link type="info" v-if="row.order_type === 2 && row.status === 3" @click="handleUpdateStatus(row, 5, '确认收件人已取件？')">确认取件</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" title="订单详情" width="600px">
      <el-descriptions :column="2" border v-if="detail">
        <el-descriptions-item label="订单号">{{ detail.order?.order_no }}</el-descriptions-item>
        <el-descriptions-item label="类型">
          <el-tag :type="detail.order?.order_type === 2 ? 'warning' : 'primary'" size="small">{{ detail.order?.order_type === 2 ? '托运' : '车票' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="orderStatusType(detail.order?.status)" size="small">{{ orderStatusText(detail.order?.status, detail.order?.order_type) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="线路">{{ detail.order?.from_station_name || detail.order?.from_station?.name }} → {{ detail.order?.to_station_name || detail.order?.to_station?.name }}</el-descriptions-item>
        <el-descriptions-item label="日期">{{ formatDate(detail.order?.trip_date) }} {{ detail.order?.departure_time }}</el-descriptions-item>
        <!-- 车票字段 -->
        <template v-if="detail.order?.order_type !== 2">
          <el-descriptions-item label="联系人">{{ detail.order?.contact_name }}</el-descriptions-item>
          <el-descriptions-item label="联系电话">{{ detail.order?.contact_phone }}</el-descriptions-item>
          <el-descriptions-item label="人数">{{ detail.order?.passenger_count }}</el-descriptions-item>
        </template>
        <!-- 托运字段 -->
        <template v-else>
          <el-descriptions-item label="寄件人">{{ detail.order?.sender_name }}</el-descriptions-item>
          <el-descriptions-item label="寄件电话">{{ detail.order?.sender_phone }}</el-descriptions-item>
          <el-descriptions-item label="收件人">{{ detail.order?.receiver_name }}</el-descriptions-item>
          <el-descriptions-item label="收件电话">{{ detail.order?.receiver_phone }}</el-descriptions-item>
          <el-descriptions-item label="货物类型">{{ detail.order?.cargo_type }}</el-descriptions-item>
          <el-descriptions-item label="重量">{{ detail.order?.weight }}kg</el-descriptions-item>
          <el-descriptions-item label="备注" :span="2">{{ detail.order?.description || '-' }}</el-descriptions-item>
        </template>
        <el-descriptions-item label="金额">¥{{ detail.order?.total_price }}</el-descriptions-item>
        <el-descriptions-item label="下单时间">{{ formatDateTime(detail.order?.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="支付时间" v-if="detail.order?.pay_time">{{ formatDateTime(detail.order?.pay_time) }}</el-descriptions-item>
      </el-descriptions>
      <!-- 乘客信息（仅车票订单） -->
      <el-table :data="detail?.passengers || []" style="margin-top: 16px" stripe v-if="detail.order?.order_type !== 2">
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column prop="id_card_no" label="身份证号" />
        <el-table-column prop="phone" label="电话" width="120" />
        <el-table-column prop="seat_no" label="座位号" width="80" />
      </el-table>
      <!-- 支付流水：回调没送达时订单状态可能是假的（待支付），这里能看出微信真实扣款 -->
      <el-table :data="detail?.payments || []" style="margin-top: 16px" stripe v-if="(detail?.payments || []).length">
        <el-table-column prop="payment_no" label="支付流水号" width="170" />
        <el-table-column prop="transaction_id" label="微信交易单号" min-width="190" />
        <el-table-column prop="amount" label="金额" width="80" />
        <el-table-column prop="method" label="方式" width="90" />
        <el-table-column label="支付时间" width="160">
          <template #default="{ row }">{{ formatDateTime(row.pay_time) }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 退款弹窗 -->
    <el-dialog v-model="refundVisible" title="退款处理" width="400px">
      <el-form label-width="80px">
        <el-form-item label="订单号"><span>{{ refundRow.order_no }}</span></el-form-item>
        <el-form-item label="退款金额"><span>¥{{ refundRow.total_price }}</span></el-form-item>
        <el-form-item label="退款原因"><el-input v-model="refundReason" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="refundVisible = false">取消</el-button>
        <el-button type="primary" :loading="refundSubmitting" @click="confirmRefund">确认退款</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { orderApi } from '@/api'
const list = ref<any[]>([]), total = ref(0), loading = ref(false)
const detailVisible = ref(false), detail = ref<any>(null)
const refundVisible = ref(false), refundRow = ref<any>({}), refundReason = ref(''), refundSubmitting = ref(false)
const query = reactive({ order_no: '', order_type: '' as any, status: '' as any, contact_phone: '', start_date: '', end_date: '', page: 1, page_size: 20 })
// 日期取前10位就中了，兼容带不带时间
const formatDate = (dt: string) => {
  if (!dt) return '-'
  return dt.slice(0, 10)
}
// 时间显示成 "YYYY-MM-DD HH:mm"
const formatDateTime = (dt: string) => {
  if (!dt) return '-'
  // 后端 JSONTime 输出 "2006-01-02 15:04:05"，取前16位即可
  return dt.length > 16 ? dt.slice(0, 16) : dt
}
const orderStatusText = (s: number, t?: number) => {
  if (t === 2) return ({ 0: '待支付', 1: '待运输', 2: '运输中', 3: '已到达', 4: '已取消', 5: '已取件' } as any)[s] || '未知'
  return ({ 0: '待支付', 1: '待出行', 2: '已完成', 3: '已退款', 4: '已取消', 5: '已核销' } as any)[s] || '未知'
}
const orderStatusType = (s: number) => ({ 0: 'warning', 1: 'success', 2: 'info', 3: 'danger', 4: 'info', 5: 'info' } as any)[s] || 'info'
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await orderApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
const handleDetail = async (row: any) => { const res: any = await orderApi.detail(row.id); detail.value = res.data; detailVisible.value = true }
const handleUpdateStatus = async (row: any, status: number, msg?: string) => {
  try {
    await ElMessageBox.confirm(msg || '确认更改订单状态？', '提示', { type: 'warning' })
    await orderApi.updateStatus(row.id, { status })
    ElMessage.success('状态更新成功')
    loadData()
  } catch (e) {
    // 用户取消不处理；API 错误已由 request 拦截器提示
    void e
  }
}
const handleRefund = (row: any) => { refundRow.value = row; refundReason.value = ''; refundVisible.value = true }
const confirmRefund = async () => {
  if (!refundReason.value.trim()) { ElMessage.warning('请填写退款原因'); return }
  refundSubmitting.value = true
  try {
    await orderApi.refund(refundRow.value.id, { reason: refundReason.value })
    ElMessage.success('退款成功')
    refundVisible.value = false
    loadData()
  } finally {
    refundSubmitting.value = false
  }
}
// 统一走API层导出CSV
const handleExport = async () => {
  try {
    const blob: any = await orderApi.export(query)
    const url = window.URL.createObjectURL(new Blob([blob], { type: 'text/csv;charset=utf-8' }))
    const link = document.createElement('a')
    link.href = url
    link.download = `orders_${new Date().toISOString().slice(0, 10)}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch (e) { ElMessage.error('导出失败') }
}
onMounted(loadData)
</script>
<style scoped>
.toolbar { display: flex; gap: 8px; flex-wrap: wrap }
.pagination { display: flex; justify-content: flex-end; margin-top: 16px }
.row-time { font-size: 12px; color: #999; }
</style>
