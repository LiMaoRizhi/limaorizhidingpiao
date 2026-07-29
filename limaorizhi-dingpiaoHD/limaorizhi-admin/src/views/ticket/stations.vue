<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.name" placeholder="站点名称" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleAdd">新增站点</el-button>
      </div>
    </div>

    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe style="width: 100%">
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="name" label="站点名称" />
        <el-table-column prop="pinyin" label="拼音" />
        <el-table-column prop="sort_order" label="排序" width="80" />
        <el-table-column label="经纬度" width="200">
          <template #default="{ row }">
            <span v-if="row.longitude">{{ row.longitude.toFixed(6) }}, {{ row.latitude.toFixed(6) }}</span>
            <span v-else style="color: #999">未设置</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleRoutes(row)">关联线路</el-button>
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.page_size"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadData"
        />
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑站点' : '新增站点'" width="450px">
      <el-form :model="editing" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="editing.name" placeholder="站点名称" />
        </el-form-item>
        <el-form-item label="拼音">
          <el-input v-model="editing.pinyin" placeholder="站点拼音(用于搜索)" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="editing.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="经纬度">
          <div style="display: flex; gap: 8px; width: 100%;">
            <el-input-number v-model="editing.longitude" :precision="6" :step="0.000001" :min="0" :max="180" controls-position="right" placeholder="经度" style="flex: 1" />
            <el-input-number v-model="editing.latitude" :precision="6" :step="0.000001" :min="0" :max="90" controls-position="right" placeholder="纬度" style="flex: 1" />
            <el-button type="primary" plain @click="openMapPicker">地图选点</el-button>
          </div>
          <div style="font-size: 12px; color: #999; margin-top: 4px;">
            坐标需为 GCJ02（火星坐标系），可点击「地图选点」在高德地图上拾取
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="editing.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 关联线路弹窗 -->
    <el-dialog v-model="routesDialogVisible" :title="`站点「${currentStationName}」关联线路`" width="700px">
      <div v-loading="routesLoading">
        <el-empty v-if="!routesLoading && stationRoutes.length === 0" description="该站点暂无关联线路" />
        <el-table v-else :data="stationRoutes" stripe style="width: 100%">
          <el-table-column prop="name" label="线路名称" min-width="140" />
          <el-table-column label="类型" width="100">
            <template #default="{ row }">
              <el-tag :type="routeTypeStyle(row.route_type)" size="small">{{ routeTypeText(row.route_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="起点 → 终点" min-width="160">
            <template #default="{ row }">{{ row.from_station }} → {{ row.to_station }}</template>
          </el-table-column>
          <el-table-column label="关系" width="80">
            <template #default="{ row }">
              <el-tag :type="row.relation === 'from' ? 'success' : row.relation === 'to' ? 'warning' : 'info'" size="small">
                {{ row.relation === 'from' ? '出发站' : row.relation === 'to' ? '到达站' : '途经站' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="routesDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { stationApi } from '@/api'
import { formatTime } from '@/utils/format'

const list = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const saving = ref(false)
const routesDialogVisible = ref(false)
const routesLoading = ref(false)
const currentStationName = ref('')
const stationRoutes = ref<any[]>([])

const query = reactive({ name: '', page: 1, page_size: 20 })
const editing = reactive({ id: 0, name: '', pinyin: '', sort_order: 0, longitude: 0, latitude: 0, status: 1 })

const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => {
  loading.value = true
  try {
    const res: any = await stationApi.list(query)
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const handleAdd = () => {
  Object.assign(editing, { id: 0, name: '', pinyin: '', sort_order: 0, longitude: 0, latitude: 0, status: 1 })
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  Object.assign(editing, row)
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!editing.name) { ElMessage.warning('请输入名称'); return }
  saving.value = true
  try {
    if (editing.id) {
      await stationApi.update(editing.id, { ...editing })
    } else {
      await stationApi.create({ ...editing })
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    loadData()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确认删除该站点？', '提示', { type: 'warning' })
  } catch { return }
  try {
    await stationApi.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e: any) {
    // 检查是否有引用详情，提供强制删除选项
    const detail = e?.response?.data?.data
    const total = Number(detail?.total_count ?? 0)
    if (total === 0) return
    const routeCount = Number(detail?.route_count ?? 0)
    const rsCount = Number(detail?.route_station_count ?? 0)
    const orderCount = Number(detail?.order_count ?? 0)
    const lines: string[] = []
    if (routeCount > 0) lines.push(`线路起终点 ${routeCount} 处`)
    if (rsCount > 0) lines.push(`线路站点序列 ${rsCount} 处`)
    if (orderCount > 0) lines.push(`订单 ${orderCount} 条`)
    const msg = `该站点被引用（共 ${total} 处：${lines.join('、')}）。\n\n强制删除将断开这些引用，相关线路和订单的站点信息会失效（订单记录仍保留）。\n\n是否继续？`
    try {
      await ElMessageBox.confirm(msg, '强制删除确认', { type: 'warning', confirmButtonText: '强制删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' })
    } catch { return }
    try {
      await stationApi.delete(row.id, true)
      ElMessage.success('删除成功')
      loadData()
    } catch (e) {
      // API 错误已由 request 拦截器提示
      void e
    }
  }
}

onMounted(loadData)

// 站点关联线路
const handleRoutes = async (row: any) => {
  currentStationName.value = row.name
  routesDialogVisible.value = true
  routesLoading.value = true
  stationRoutes.value = []
  try {
    const res: any = await stationApi.routes(row.id)
    stationRoutes.value = res.data?.routes || []
  } catch (e) {
    // API 错误已由 request 拦截器提示
    void e
  } finally {
    routesLoading.value = false
  }
}

// 地图选点：打开高德坐标拾取器，选中后手动填入
const openMapPicker = () => {
  const name = editing.name || ''
  const url = `https://lbs.amap.com/tools/picker${name ? '?keyword=' + encodeURIComponent(name) : ''}`
  window.open(url, '_blank')
  ElMessage.info('在高德地图上搜索站点位置，点击地图后复制坐标填入此处（高德坐标即为GCJ02）')
}

const routeTypeText = (t: number) => t === 2 ? '城际客运' : t === 3 ? '旅游专线' : '城乡公交'
const routeTypeStyle = (t: number) => t === 2 ? 'primary' : t === 3 ? 'warning' : 'success'
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
