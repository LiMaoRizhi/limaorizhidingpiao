<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.name" placeholder="线路名称" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleAdd">新增线路</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="name" label="线路名称" min-width="120" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="routeTypeStyle(row.route_type).type" size="small">{{ routeTypeStyle(row.route_type).label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="起止站">
          <template #default="{ row }">{{ row.from_station?.name }} → {{ row.to_station?.name }}</template>
        </el-table-column>
        <el-table-column label="站点数" width="80">
          <template #default="{ row }">{{ row.route_stations?.length || 2 }}</template>
        </el-table-column>
        <el-table-column prop="distance_km" label="里程(km)" width="100" />
        <el-table-column label="时长" width="100">
          <template #default="{ row }">{{ row.duration_minutes }}分钟</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '运营' : '停运' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" @click="handleEdit(row)">票价面板</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination"><el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" /></div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { routeApi } from '@/api'
const router = useRouter()
const list = ref<any[]>([]), total = ref(0), loading = ref(false)
const query = reactive({ name: '', page: 1, page_size: 20 })
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await routeApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
const handleAdd = () => { router.push('/ticket/routes/edit') }
const handleEdit = (row: any) => { router.push(`/ticket/routes/edit?id=${row.id}`) }
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确认删除该线路？', '提示', { type: 'warning' })
  } catch { return }
  try {
    await routeApi.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e: any) {
    const detail = e?.response?.data?.data
    const total = Number(detail?.total_count ?? 0)
    if (total === 0) return
    const tripCount = Number(detail?.trip_count ?? 0)
    const orderCount = Number(detail?.order_count ?? 0)
    const lines: string[] = []
    if (tripCount > 0) lines.push(`班次 ${tripCount} 条`)
    if (orderCount > 0) lines.push(`订单 ${orderCount} 条`)
    const msg = `该线路被引用（共 ${total} 处：${lines.join('、')}）。\n\n强制删除将断开引用并删除线路站点序列。订单记录仍保留。\n\n是否继续？`
    try {
      await ElMessageBox.confirm(msg, '强制删除确认', { type: 'warning', confirmButtonText: '强制删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' })
    } catch { return }
    try {
      await routeApi.delete(row.id, true)
      ElMessage.success('删除成功')
      loadData()
    } catch { /* error handled by interceptor */ }
  }
}
const routeTypeStyle = (t: number) => {
  const map: Record<number, { label: string; type: string }> = {
    1: { label: '城乡公交', type: 'success' },
    2: { label: '城际客运', type: 'primary' },
    3: { label: '旅游专线', type: 'warning' },
  }
  return map[t] || map[1]
}
onMounted(() => { loadData() })
</script>
<style scoped>
.toolbar { display: flex; gap: 8px }
.pagination { display: flex; justify-content: flex-end; margin-top: 16px }

</style>
