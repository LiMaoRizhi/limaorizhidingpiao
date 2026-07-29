<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.keyword" placeholder="姓名/手机号" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleAdd">新增司机</el-button>
      </div>
    </div>

    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="employee_no" label="工号" width="100">
          <template #default="{ row }">
            {{ row.employee_no || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="phone" label="手机号" width="140" />
        <el-table-column prop="license_no" label="驾驶证号" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_login_at" label="最后登录" width="170">
          <template #default="{ row }">
            {{ row.last_login_at ? formatTime(row.last_login_at) : '未登录' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除该司机吗？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-box">
        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.page_size"
          :total="total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑司机' : '新增司机'" width="500px">
      <el-form :model="editing" label-width="100px">
        <el-form-item label="工号">
          <el-input v-model="editing.employee_no" placeholder="工号（选填，用于区分重名司机）" />
          <span style="margin-left:8px;color:#909399;font-size:12px">多个司机重名时，填写工号方便区分</span>
        </el-form-item>
        <el-form-item label="姓名" required>
          <el-input v-model="editing.name" placeholder="司机姓名" />
        </el-form-item>
        <el-form-item label="手机号" required>
          <el-input v-model="editing.phone" placeholder="手机号（登录用）" />
        </el-form-item>
        <el-form-item label="密码" :required="!editing.id">
          <el-input v-model="editing.password" type="password" show-password
            :placeholder="editing.id ? '留空不修改' : '登录密码'" />
        </el-form-item>
        <el-form-item label="驾驶证号">
          <el-input v-model="editing.license_no" placeholder="驾驶证号（选填）" />
        </el-form-item>
        <el-form-item label="状态" v-if="editing.id">
          <el-switch v-model="editing.status" :active-value="1" :inactive-value="0"
            active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { driverApi } from '@/api'
import { ElMessage } from 'element-plus'
import { formatTime } from '@/utils/format'

const loading = ref(false)
const submitting = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const dialogVisible = ref(false)

const query = reactive({
  keyword: '',
  status: '' as string | number,
  page: 1,
  page_size: 20,
})

const editing = reactive<any>({
  id: 0,
  employee_no: '',
  name: '',
  phone: '',
  password: '',
  license_no: '',
  status: 1,
})

const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => {
  loading.value = true
  try {
    const res = await driverApi.list(query)
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const resetEditing = () => {
  Object.assign(editing, { id: 0, employee_no: '', name: '', phone: '', password: '', license_no: '', status: 1 })
}

const handleAdd = () => {
  resetEditing()
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  Object.assign(editing, { ...row, password: '' })
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!editing.name || !editing.phone) {
    ElMessage.warning('请填写姓名和手机号')
    return
  }
  // 编辑/新增司机时校验手机号格式（11位、1开头）
  if (!/^1[0-9]{10}$/.test(editing.phone)) {
    ElMessage.warning('手机号格式不正确（需11位、1开头）')
    return
  }
  if (!editing.id && !editing.password) {
    ElMessage.warning('请设置登录密码')
    return
  }
  submitting.value = true
  try {
    if (editing.id) {
      await driverApi.update(editing.id, editing)
      ElMessage.success('更新成功')
    } else {
      await driverApi.create(editing)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await driverApi.delete(id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    // API 错误已由 request 拦截器提示
    void e
  }
}

onMounted(() => loadData())
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}
.pagination-box {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
