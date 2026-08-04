<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.plate_no" placeholder="车牌号" clearable style="width: 180px" @keyup.enter="handleSearch" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleAdd">新增车辆</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="plate_no" label="车牌号" width="120" />
        <el-table-column prop="vehicle_type" label="车型" width="100" />
        <el-table-column prop="seat_count" label="座位数" width="80" />
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'warning'" size="small">{{ row.status === 1 ? '可用' : '维修' }}</el-tag></template>
        </el-table-column>
        <el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }"><el-button size="small" @click="handleEdit(row)">编辑</el-button><el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button></template>
        </el-table-column>
      </el-table>
      <div class="pagination"><el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" /></div>
    </div>
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑车辆' : '新增车辆'" width="800px" top="5vh">
      <el-form :model="editing" label-width="80px">
        <el-form-item label="车牌号" required><el-input v-model="editing.plate_no" /></el-form-item>
        <el-form-item label="车型"><el-input v-model="editing.vehicle_type" placeholder="大巴/中巴/商务车" /></el-form-item>
        <el-form-item label="座位数"><el-input-number v-model="editing.seat_count" :min="1" /></el-form-item>
        <el-form-item label="状态"><el-switch v-model="editing.status" :active-value="1" :inactive-value="0" active-text="可用" inactive-text="维修" /></el-form-item>
        <el-form-item label="座位布局">
          <SeatLayoutEditor v-model="editing.seat_layout" :vehicle-seat-count="editing.seat_count" />
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" :loading="saving" @click="handleSave">保存</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { vehicleApi } from '@/api'
import { formatTime } from '@/utils/format'
import SeatLayoutEditor from '@/components/SeatLayoutEditor.vue'
const list = ref<any[]>([]), total = ref(0), loading = ref(false), dialogVisible = ref(false), saving = ref(false)
const query = reactive({ plate_no: '', page: 1, page_size: 20 })
const editing = reactive({ id: 0, plate_no: '', vehicle_type: '', seat_count: 0, seat_layout: '', status: 1 })
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await vehicleApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
const handleAdd = () => { Object.assign(editing, { id: 0, plate_no: '', vehicle_type: '', seat_count: 0, seat_layout: '', status: 1 }); dialogVisible.value = true }
const handleEdit = (row: any) => { Object.assign(editing, row); dialogVisible.value = true }
const handleSave = async () => {
  if (!editing.plate_no) { ElMessage.warning('请输入车牌号'); return }
  saving.value = true
  try {
    if (editing.id) await vehicleApi.update(editing.id, { ...editing })
    else await vehicleApi.create({ ...editing })
    ElMessage.success('保存成功')
    dialogVisible.value = false
    loadData()
  } finally {
    saving.value = false
  }
}
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确认删除该车辆？', '提示', { type: 'warning' })
    await vehicleApi.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    // 用户取消不处理；API 错误已由 request 拦截器提示
    void e
  }
}
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px }.pagination { display: flex; justify-content: flex-end; margin-top: 16px }</style>
