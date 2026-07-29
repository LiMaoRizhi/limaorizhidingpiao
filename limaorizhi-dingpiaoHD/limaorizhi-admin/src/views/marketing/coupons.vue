<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.name" placeholder="优惠券名称" clearable style="width: 180px" @keyup.enter="handleSearch" />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 100px">
          <el-option label="启用" :value="1" />
          <el-option label="停用" :value="0" />
        </el-select>
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleAdd">新增优惠券</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="couponTypeTag(row.type)" size="small">{{ couponTypeText(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="面值/折扣" width="100">
          <template #default="{ row }">
            <span v-if="row.type === 1 || row.type === 3">¥{{ row.discount_value }}</span>
            <span v-else>{{ row.discount_value }}折</span>
          </template>
        </el-table-column>
        <el-table-column label="最低消费" width="90"><template #default="{ row }">¥{{ row.min_spend }}</template></el-table-column>
        <el-table-column prop="valid_days" label="有效天数" width="90" />
        <el-table-column label="发放/已用" width="100">
          <template #default="{ row }">
            <span>{{ row.issued_count }} / {{ row.used_count }}</span>
            <span v-if="row.total_count > 0" style="color: #999; font-size: 12px"> / {{ row.total_count }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑优惠券' : '新增优惠券'" width="520px">
      <el-form :model="editing" label-width="100px">
        <el-form-item label="优惠券名称" required><el-input v-model="editing.name" placeholder="如：满50减10" /></el-form-item>
        <el-form-item label="优惠券类型" required>
          <el-select v-model="editing.type" style="width: 100%">
            <el-option label="满减券（满X减Y元）" :value="1" />
            <el-option label="折扣券（打X折）" :value="2" />
            <el-option label="固定金额券（抵扣X元）" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item label="面值/折扣" required>
          <el-input-number v-model="editing.discount_value" :min="0.01" :precision="2" :step="0.5" style="width: 100%" />
          <span style="color: #999; font-size: 12px; margin-left: 8px">{{ editing.type === 2 ? '折扣值，如8.5表示85折' : '单位：元' }}</span>
        </el-form-item>
        <el-form-item label="最低消费"><el-input-number v-model="editing.min_spend" :min="0" :precision="2" :step="0.5" style="width: 100%" /><span style="color: #999; font-size: 12px; margin-left: 8px">元</span></el-form-item>
        <el-form-item label="有效天数"><el-input-number v-model="editing.valid_days" :min="1" :step="1" style="width: 100%" /><span style="color: #999; font-size: 12px; margin-left: 8px">领取后有效天数</span></el-form-item>
        <el-form-item label="发放总量"><el-input-number v-model="editing.total_count" :min="0" :step="1" style="width: 100%" /><span style="color: #999; font-size: 12px; margin-left: 8px">0=不限量</span></el-form-item>
        <el-form-item label="状态"><el-switch v-model="editing.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { couponApi } from '@/api'
const list = ref<any[]>([]), total = ref(0), loading = ref(false), dialogVisible = ref(false), saving = ref(false)
const query = reactive({ name: '', status: '' as any, page: 1, page_size: 20 })
const editing = reactive({ id: 0, name: '', type: 1, discount_value: 10, min_spend: 0, valid_days: 30, total_count: 0, status: 1 })
const couponTypeText = (t: number) => ({ 1: '满减券', 2: '折扣券', 3: '固定金额' } as any)[t] || '未知'
const couponTypeTag = (t: number) => ({ 1: 'danger', 2: 'warning', 3: 'primary' } as any)[t] || 'info'
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await couponApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
const handleAdd = () => { Object.assign(editing, { id: 0, name: '', type: 1, discount_value: 10, min_spend: 0, valid_days: 30, total_count: 0, status: 1 }); dialogVisible.value = true }
const handleEdit = (row: any) => { Object.assign(editing, row); dialogVisible.value = true }
const handleSave = async () => {
  if (!editing.name) { ElMessage.warning('请输入优惠券名称'); return }
  if (editing.discount_value <= 0) { ElMessage.warning('面值/折扣必须大于0'); return }
  // 折扣券校验提示修正
  if (editing.type === 2 && editing.discount_value >= 10) { ElMessage.warning('折扣值必须大于0且小于10'); return }
  // 满减券和固定金额券的最低消费必须 ≥ 面值，防止负金额支付
  if ((editing.type === 1 || editing.type === 3) && editing.min_spend < editing.discount_value) {
    ElMessage.warning('最低消费不能小于面值，否则会导致负金额支付'); return
  }
  saving.value = true
  try {
    if (editing.id) await couponApi.update(editing.id, { ...editing })
    else await couponApi.create({ ...editing })
    ElMessage.success('保存成功')
    dialogVisible.value = false
    loadData()
  } finally {
    saving.value = false
  }
}
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确认删除该优惠券？', '提示', { type: 'warning' })
    await couponApi.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    // 用户取消不处理；API 错误已由 request 拦截器提示
    void e
  }
}
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px; flex-wrap: wrap }.pagination { display: flex; justify-content: flex-end; margin-top: 16px }</style>
