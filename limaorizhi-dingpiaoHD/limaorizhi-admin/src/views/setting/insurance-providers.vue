<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.name" placeholder="保险公司名称" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleAdd">新增保险公司</el-button>
        <span class="tips">同一时刻仅一家保险公司可启用，启用新公司会自动停用其他公司</span>
      </div>
    </div>

    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="name" label="名称" width="160" />
        <el-table-column prop="api_url" label="出单API地址" min-width="240" show-overflow-tooltip />
        <el-table-column prop="app_id" label="商户号" width="140" />
        <el-table-column label="密钥" width="140">
          <template #default="{ row }">
            {{ row.app_secret_masked || '****' }}
          </template>
        </el-table-column>
        <el-table-column prop="product_code" label="产品代码" width="120">
          <template #default="{ row }">{{ row.product_code || '-' }}</template>
        </el-table-column>
        <el-table-column prop="fee" label="保费(元/人)" width="100" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
              {{ row.is_active ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="必购" width="80">
          <template #default="{ row }">
            <el-tag :type="row.required ? 'danger' : 'info'" size="small">
              {{ row.required ? '必购' : '可选' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="!row.is_active"
              title="确定启用该保险公司吗？启用后会自动停用其他保险公司。"
              @confirm="handleActivate(row.id)"
            >
              <template #reference>
                <el-button size="small" type="success">启用</el-button>
              </template>
            </el-popconfirm>
            <el-popconfirm title="确定删除该保险公司配置吗？" @confirm="handleDelete(row.id)">
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
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑保险公司' : '新增保险公司'" width="640px">
      <el-form :model="editing" label-width="120px">
        <el-form-item label="保险公司名称" required>
          <el-input v-model="editing.name" placeholder="如：中国人寿/平安产险" />
        </el-form-item>
        <el-form-item label="出单API地址" required>
          <el-input v-model="editing.api_url" placeholder="https://api.xxx.com/issue-policy" />
        </el-form-item>
        <el-form-item label="商户号/AppID" required>
          <el-input v-model="editing.app_id" placeholder="保险公司分配的商户号或应用ID" />
        </el-form-item>
        <el-form-item label="商户密钥" :required="!editing.id">
          <el-input
            v-model="editing.app_secret"
            type="password"
            show-password
            :placeholder="editing.id ? '留空不修改（当前：' + (editing.app_secret_masked || '****') + '）' : '商户密钥（HMAC-SHA256签名用）'"
          />
        </el-form-item>
        <el-form-item label="产品代码">
          <el-input v-model="editing.product_code" placeholder="保险产品代码（选填）" />
        </el-form-item>
        <el-form-item label="保费(元/人)" required>
          <el-input-number v-model="editing.fee" :min="0" :max="1000" :precision="2" :step="0.5" />
        </el-form-item>
        <el-form-item label="是否必购">
          <el-switch v-model="editing.required" active-text="必购" inactive-text="可选" />
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="editing.is_active" active-text="启用" inactive-text="停用" />
          <span style="margin-left:8px;color:#909399;font-size:12px">勾选启用会自动停用其他保险公司</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editing.remark" type="textarea" :rows="2" placeholder="对接联系人/合同号等备注信息（选填）" />
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
import { insuranceProviderApi } from '@/api'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const submitting = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const dialogVisible = ref(false)

const query = reactive({
  name: '',
  page: 1,
  page_size: 20,
})

const editing = reactive<any>({
  id: 0,
  name: '',
  api_url: '',
  app_id: '',
  app_secret: '',
  app_secret_masked: '',
  product_code: '',
  fee: 0,
  required: false,
  is_active: false,
  remark: '',
})

const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => {
  loading.value = true
  try {
    const res = await insuranceProviderApi.list(query)
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const resetEditing = () => {
  Object.assign(editing, {
    id: 0,
    name: '',
    api_url: '',
    app_id: '',
    app_secret: '',
    app_secret_masked: '',
    product_code: '',
    fee: 0,
    required: false,
    is_active: false,
    remark: '',
  })
}

const handleAdd = () => {
  resetEditing()
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  Object.assign(editing, { ...row, app_secret: '' })
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!editing.name) {
    ElMessage.warning('请填写保险公司名称')
    return
  }
  if (!editing.api_url) {
    ElMessage.warning('请填写出单API地址')
    return
  }
  if (!editing.app_id) {
    ElMessage.warning('请填写商户号/AppID')
    return
  }
  if (!editing.id && !editing.app_secret) {
    ElMessage.warning('请填写商户密钥')
    return
  }
  if (editing.fee < 0 || editing.fee > 1000) {
    ElMessage.warning('保费必须在0-1000之间')
    return
  }
  submitting.value = true
  try {
    const payload: any = {
      name: editing.name,
      api_url: editing.api_url,
      app_id: editing.app_id,
      product_code: editing.product_code || '',
      fee: editing.fee,
      required: editing.required,
      is_active: editing.is_active,
      remark: editing.remark || '',
    }
    if (editing.app_secret) {
      payload.app_secret = editing.app_secret
    }
    if (editing.id) {
      await insuranceProviderApi.update(editing.id, payload)
      ElMessage.success('更新成功')
    } else {
      await insuranceProviderApi.create(payload)
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
    await insuranceProviderApi.delete(id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    void e
  }
}

const handleActivate = async (id: number) => {
  try {
    await insuranceProviderApi.activate(id)
    ElMessage.success('启用成功')
    loadData()
  } catch (e) {
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
.toolbar .tips {
  color: #909399;
  font-size: 12px;
  margin-left: 8px;
}
.pagination-box {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
