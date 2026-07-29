<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.keyword" placeholder="姓名/身份证/手机号" clearable style="width: 250px" @keyup.enter="handleSearch" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="id_card_no" label="身份证号" width="200" />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column prop="user_nickname" label="所属用户" width="150" />
        <el-table-column prop="usage_count" label="使用次数" width="80" />
        <el-table-column label="添加时间" width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="warning" link @click="handleInvalidateCache(row)">清除认证缓存</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { userApi, idCardVerifyApi } from '@/api'
import { formatTime } from '@/utils/format'
const list = ref<any[]>([]), total = ref(0), loading = ref(false)
const query = reactive({ keyword: '', page: 1, page_size: 20 })
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await userApi.passengers(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }

// 清除指定乘客的实名认证缓存
// 应用场景：管理端标记某乘客证件异常/需要重新走云市场 API 时调用
// 删除后，下次该乘客下单/编辑时会重新调用云市场 API（0.3 元/次）
const handleInvalidateCache = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `确认清除 ${row.name}（${row.id_card_no}）的实名认证缓存？\n清除后，下次该乘客认证将重新调用云市场 API（0.3 元/次）。`,
      '清除认证缓存',
      { type: 'warning', confirmButtonText: '确认清除', cancelButtonText: '取消' }
    )
  } catch {
    return // 用户取消
  }
  try {
    const res: any = await idCardVerifyApi.invalidateCache(row.id)
    ElMessage.success(`已清除 ${res.data.name} 的认证缓存`)
  } catch {
    // request 工具会自动 toast 错误消息，这里不需重复提示
  }
}
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px; flex-wrap: wrap }.pagination { display: flex; justify-content: flex-end; margin-top: 16px }</style>
