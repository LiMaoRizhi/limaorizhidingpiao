<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.keyword" placeholder="用户名/姓名" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleAdd">新增管理员</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="username" label="用户名" width="150" />
        <el-table-column prop="real_name" label="姓名" width="120" />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column prop="role_name" label="角色" width="100" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后登录" width="180"><template #default="{ row }">{{ row.last_login_at ? formatTime(row.last_login_at) : '未登录' }}</template></el-table-column>
        <el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" v-if="row.role !== 1" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>

    <el-dialog v-model="formVisible" :title="formTitle" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="用户名" prop="username"><el-input v-model="form.username" :disabled="!!form.id" /></el-form-item>
        <el-form-item label="姓名" prop="real_name"><el-input v-model="form.real_name" /></el-form-item>
        <el-form-item label="手机号" prop="phone"><el-input v-model="form.phone" maxlength="11" /></el-form-item>
        <el-form-item label="密码" prop="password" v-if="!form.id"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="form.role" style="width: 100%">
            <el-option label="超级管理员" :value="1" v-if="isSuperAdmin" />
            <el-option label="管理员" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { adminApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { formatTime } from '@/utils/format'
const authStore = useAuthStore()
const isSuperAdmin = computed(() => authStore.isSuperAdmin())
const list = ref<any[]>([]), total = ref(0), loading = ref(false)
const formVisible = ref(false), formRef = ref<FormInstance>()
const form = reactive<any>({ id: null, username: '', real_name: '', phone: '', password: '', role: 2, status: 1 })
const query = reactive({ keyword: '', page: 1, page_size: 20 })
const formTitle = computed(() => form.id ? '编辑管理员' : '新增管理员')
// 表单校验规则
const phoneValidator = (rule: any, value: string, callback: any) => {
  if (!value) { callback(new Error('请输入手机号')); return }
  if (!/^1\d{10}$/.test(value)) { callback(new Error('手机号格式不正确')); return }
  callback()
}
const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度3-20位', trigger: 'blur' }
  ],
  real_name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
  phone: [{ required: true, validator: phoneValidator, trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少8位', trigger: 'blur' }
  ],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }]
}
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await adminApi.list(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
const resetForm = () => { Object.assign(form, { id: null, username: '', real_name: '', phone: '', password: '', role: 2, status: 1 }) }
const handleAdd = () => { resetForm(); formVisible.value = true }
const handleEdit = (row: any) => { Object.assign(form, { ...row, password: '' }); formVisible.value = true }
const handleSave = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    // 非超管不能创建/编辑超管角色
    if (form.role === 1 && !isSuperAdmin.value) {
      ElMessage.error('无权操作超级管理员角色')
      return
    }
    try { if (form.id) { await adminApi.update(form.id, form) } else { await adminApi.create(form) }; ElMessage.success('保存成功'); formVisible.value = false; loadData() } catch (e) {
      // 错误由axios拦截器统一处理，无需重复提示
    }
  })
}
const handleDelete = async (row: any) => {
  try { await ElMessageBox.confirm('确认删除该管理员？', '提示', { type: 'warning' }); await adminApi.delete(row.id); ElMessage.success('删除成功'); loadData() } catch (e) {
    // 用户取消或错误由拦截器统一处理
  }
}
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px; flex-wrap: wrap }.pagination { display: flex; justify-content: flex-end; margin-top: 16px }</style>
