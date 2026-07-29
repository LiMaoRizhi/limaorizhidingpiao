<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar"><el-button type="primary" @click="handleAdd">新增积分规则</el-button></div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="rule_name" label="规则名称" min-width="140" />
        <el-table-column label="规则类型" width="120">
          <template #default="{ row }">
            <el-tag :type="ruleTypeTag(row.rule_type)" size="small">{{ ruleTypeText(row.rule_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="积分配置" width="160">
          <template #default="{ row }">
            <span v-if="row.rule_type === 1">每消费1元得 {{ row.points_per_yuan }} 积分</span>
            <span v-else-if="row.rule_type === 2">赠送 {{ row.fixed_points }} 积分</span>
            <span v-else>手动调整</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="说明" min-width="200" />
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
    </div>
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑积分规则' : '新增积分规则'" width="520px">
      <el-form :model="editing" label-width="100px">
        <el-form-item label="规则名称" required><el-input v-model="editing.rule_name" placeholder="如：消费赠送" /></el-form-item>
        <el-form-item label="规则类型" required>
          <el-select v-model="editing.rule_type" style="width: 100%">
            <el-option label="消费赠送" :value="1" />
            <el-option label="注册赠送" :value="2" />
            <el-option label="手动调整" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editing.rule_type === 1" label="每元积分">
          <el-input-number v-model="editing.points_per_yuan" :min="0" :precision="2" :step="0.5" style="width: 100%" />
          <span style="color: #999; font-size: 12px; margin-left: 8px">每消费1元获得的积分</span>
        </el-form-item>
        <el-form-item v-if="editing.rule_type === 2" label="固定积分">
          <el-input-number v-model="editing.fixed_points" :min="0" :step="10" style="width: 100%" />
          <span style="color: #999; font-size: 12px; margin-left: 8px">赠送的积分数量</span>
        </el-form-item>
        <el-form-item label="规则说明"><el-input v-model="editing.description" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="状态"><el-switch v-model="editing.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { pointRuleApi } from '@/api'
const list = ref<any[]>([]), loading = ref(false), dialogVisible = ref(false)
const editing = reactive({ id: 0, rule_name: '', rule_type: 1, points_per_yuan: 1, fixed_points: 0, description: '', status: 1 })
const ruleTypeText = (t: number) => ({ 1: '消费赠送', 2: '注册赠送', 3: '手动调整' } as any)[t] || '未知'
const ruleTypeTag = (t: number) => ({ 1: 'success', 2: 'primary', 3: 'warning' } as any)[t] || 'info'
const loadData = async () => { loading.value = true; try { const res: any = await pointRuleApi.list({}); list.value = Array.isArray(res.data) ? res.data : (res.data?.list || []) } finally { loading.value = false } }
const handleAdd = () => { Object.assign(editing, { id: 0, rule_name: '', rule_type: 1, points_per_yuan: 1, fixed_points: 0, description: '', status: 1 }); dialogVisible.value = true }
const handleEdit = (row: any) => { Object.assign(editing, row); dialogVisible.value = true }
const handleSave = async () => {
  if (!editing.rule_name) { ElMessage.warning('请输入规则名称'); return }
  try { if (editing.id) await pointRuleApi.update(editing.id, { ...editing }); else await pointRuleApi.create({ ...editing }); ElMessage.success('保存成功'); dialogVisible.value = false; loadData() } catch (e) { /* 错误由axios拦截器处理 */ }
}
const handleDelete = async (row: any) => { try { await ElMessageBox.confirm('确认删除该积分规则？', '提示', { type: 'warning' }); await pointRuleApi.delete(row.id); ElMessage.success('删除成功'); loadData() } catch (e) { /* 用户取消或错误由拦截器处理 */ } }
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px }</style>
