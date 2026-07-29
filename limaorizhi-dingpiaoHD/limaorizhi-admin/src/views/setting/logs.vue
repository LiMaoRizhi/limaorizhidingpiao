<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar">
        <el-input v-model="query.keyword" placeholder="操作人/内容" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-date-picker v-model="query.start_date" type="date" value-format="YYYY-MM-DD" placeholder="开始日期" style="width: 150px" />
        <el-date-picker v-model="query.end_date" type="date" value-format="YYYY-MM-DD" placeholder="结束日期" style="width: 150px" />
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button type="primary" @click="handleExport" :loading="exporting"><svg style="width:16px;height:16px;vertical-align:middle;margin-right:4px" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg"><path d="M589.6192 713.6256c0-23.04 20.1728-43.2128 43.2128-43.2128h273.6128c23.04 0 43.2128 20.1728 43.2128 43.2128s-20.1728 43.2128-43.2128 43.2128H632.7296c-23.04 0-43.1104-20.1728-43.1104-43.2128z" fill="currentColor"></path><path d="M907.9808 756.8384c-8.6016 0-20.1728-2.8672-31.6416-11.5712L746.7008 615.7312c-17.3056-17.3056-17.3056-43.2128 0-60.5184s43.2128-17.3056 60.5184 0l129.6384 129.6384c17.3056 17.3056 17.3056 43.2128 0 60.5184-5.8368 8.6016-17.3056 11.4688-28.8768 11.4688z" fill="currentColor"></path><path d="M746.1888 874.9056c-17.3056-17.3056-17.3056-43.2128 0-60.5184l129.6384-129.536c17.3056-17.3056 43.2128-17.3056 60.5184 0s17.3056 43.2128 0 60.5184L806.7072 874.9056c-5.7344 8.6016-17.3056 11.5712-28.7744 11.5712-11.5712-0.1024-23.04-5.8368-31.744-11.5712z" fill="currentColor"></path><path d="M851.8656 932.4544a126.65856 126.65856 0 0 1-89.2928 37.4784c-34.6112 0-63.3856-11.5712-89.2928-37.4784L546.5088 802.9184c-17.3056-17.3056-28.7744-40.3456-34.6112-63.3856-2.8672-8.6016-2.8672-17.3056-2.8672-25.9072s0-17.3056 2.8672-25.9072c5.7344-23.04 17.3056-46.08 34.6112-63.3856L676.0448 494.592c48.4352-49.3568 127.6928-49.9712 177.0496-1.536l1.536 1.536c17.6128 17.6128 52.0192 62.1568 67.2768 82.1248 1.4336 1.8432 4.096 2.2528 5.9392 0.8192 1.024-0.8192 1.6384-2.048 1.6384-3.3792V280.064c0-65.6384-53.76-119.3984-119.3984-119.3984H314.5728c-65.6384 0-119.3984 53.76-119.3984 119.3984v573.3376c0 65.6384 53.76 119.3984 119.3984 119.3984h492.6464c65.6384 0 119.3984-53.76 119.3984-119.3984 0-2.3552-1.8432-4.1984-4.1984-4.1984a3.9936 3.9936 0 0 0-3.3792 1.7408c-14.7456 20.1728-47.616 64.1024-67.1744 81.5104z" fill="currentColor"></path><path d="M131.7888 823.0912V212.48c0-66.2528 54.6816-120.9344 120.9344-120.9344h524.1856C756.8384 68.5056 725.0944 51.2 690.5856 51.2H189.44C126.0544 51.2 74.24 103.0144 74.24 166.4v584.6016c0 43.2128 23.04 80.5888 60.5184 100.7616-2.9696-11.4688-2.9696-20.0704-2.9696-28.672z" fill="currentColor"></path></svg>导出备份</el-button>
      </div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="admin_name" label="操作人" width="120" />
        <el-table-column prop="module" label="模块" width="100" />
        <el-table-column prop="action" label="操作类型" width="100" />
        <el-table-column prop="detail" label="操作内容" show-overflow-tooltip />
        <el-table-column prop="ip_address" label="IP地址" width="130" />
        <el-table-column label="操作时间" min-width="180"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
      </el-table>
      <div class="pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { adminApi } from '@/api'
import { formatTime } from '@/utils/format'
const list = ref<any[]>([]), total = ref(0), loading = ref(false), exporting = ref(false)
const query = reactive({ keyword: '', start_date: '', end_date: '', page: 1, page_size: 20 })
const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => { loading.value = true; try { const res: any = await adminApi.logs(query); list.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
const handleExport = async () => {
  exporting.value = true
  try {
    const blob = await adminApi.logsExport(query)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `操作日志_${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } finally { exporting.value = false }
}
onMounted(loadData)
</script>
<style scoped>.toolbar { display: flex; gap: 8px; flex-wrap: wrap }.pagination { display: flex; justify-content: flex-end; margin-top: 16px }</style>
