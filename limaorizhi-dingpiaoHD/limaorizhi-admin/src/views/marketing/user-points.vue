<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.phone" placeholder="用户手机号" clearable style="width: 180px" @keyup.enter="handleSearch" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column label="用户" min-width="160">
          <template #default="{ row }">{{ row.user?.nickname || '-' }} ({{ row.user?.phone || '-' }})</template>
        </el-table-column>
        <el-table-column prop="balance" label="当前余额" width="100" />
        <el-table-column prop="total_earned" label="累计获得" width="100" />
        <el-table-column prop="total_spent" label="累计消耗" width="100" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleRecords(row)">积分明细</el-button>
            <el-button size="small" type="primary" @click="handleAdjust(row)">调整积分</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>
    <!-- 积分明细弹窗 -->
    <el-dialog v-model="recordsVisible" title="积分明细" width="600px">
      <el-table :data="records" stripe v-loading="recordsLoading">
        <el-table-column label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="row.change_type === 1 ? 'success' : 'warning'" size="small">{{ row.change_type === 1 ? '获得' : '消耗' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="points" label="积分" width="80" />
        <el-table-column prop="source" label="来源" width="100"><template #default="{ row }">{{ sourceText(row.source) }}</template></el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" />
        <el-table-column label="操作人" width="100"><template #default="{ row }">{{ row.admin_name || '-' }}</template></el-table-column>
        <el-table-column label="时间" width="160"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="recordQuery.page" v-model:page-size="recordQuery.page_size" :total="recordTotal" layout="total, prev, pager, next" @current-change="loadRecords" />
      </div>
    </el-dialog>
    <!-- 调整积分弹窗 -->
    <el-dialog v-model="adjustVisible" title="调整积分" width="460px">
      <el-form label-width="80px">
        <el-form-item label="用户"><span>{{ adjustRow.user?.nickname }} ({{ adjustRow.user?.phone }})</span></el-form-item>
        <el-form-item label="当前余额"><span style="font-weight: 600">{{ adjustRow.balance }}</span></el-form-item>
        <el-form-item label="调整积分" required>
          <el-input-number v-model="adjustForm.points" :step="10" style="width: 100%" />
          <span style="color: #999; font-size: 12px; margin-left: 8px">正数增加，负数减少</span>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="adjustForm.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustVisible = false">取消</el-button>
        <el-button type="primary" :loading="adjustSubmitting" @click="confirmAdjust">确认调整</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { userPointsApi } from '@/api'
import { formatTime } from '@/utils/format'
const list = ref<any[]>([]), total = ref(0), loading = ref(false)
const query = reactive({ phone: '', page: 1, page_size: 20 })
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await userPointsApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }

// 积分明细
const recordsVisible = ref(false), recordsLoading = ref(false)
const records = ref<any[]>([]), recordTotal = ref(0)
const recordQuery = reactive({ page: 1, page_size: 20 })
const currentUserId = ref(0)
const sourceText = (s: string) => ({ order: '订单', manual: '手动调整', register: '注册' } as any)[s] || s
const handleRecords = async (row: any) => {
  currentUserId.value = row.user_id; recordsVisible.value = true; recordQuery.page = 1
  await loadRecords()
}
const loadRecords = async () => {
  recordsLoading.value = true
  // 传入recordQuery分页参数，使后端分页生效
  try { const res: any = await userPointsApi.records(currentUserId.value, recordQuery); records.value = res.data.list || []; recordTotal.value = res.data.total || 0 } finally { recordsLoading.value = false }
}

// 调整积分
const adjustVisible = ref(false)
const adjustSubmitting = ref(false)
const adjustRow = ref<any>({}), adjustForm = reactive({ points: 0, remark: '' })
const handleAdjust = (row: any) => { adjustRow.value = row; Object.assign(adjustForm, { points: 0, remark: '' }); adjustVisible.value = true }
const confirmAdjust = async () => {
  if (adjustForm.points === 0) { ElMessage.warning('调整积分不能为0'); return }
  adjustSubmitting.value = true
  try {
    await userPointsApi.adjust(adjustRow.value.user_id, { ...adjustForm })
    ElMessage.success('积分调整成功')
    adjustVisible.value = false
    loadData()
  } finally {
    adjustSubmitting.value = false
  }
}
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px }.pagination { display: flex; justify-content: flex-end; margin-top: 16px }</style>
