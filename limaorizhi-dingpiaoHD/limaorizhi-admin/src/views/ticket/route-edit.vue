<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="route-edit-page">
    <!-- 顶部导航 -->
    <div class="page-header">
      <el-button :icon="ArrowLeft" @click="goBack">返回线路列表</el-button>
      <span class="page-title">{{ isEdit ? '编辑线路' : '新增线路' }}</span>
    </div>

    <!-- 基本信息区 -->
    <div class="card-box">
      <div class="section-header">
        <span class="section-title">基本信息</span>
      </div>
      <el-form :model="editing" label-width="100px" class="route-form">
        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="线路名称" required>
              <el-input v-model="editing.name" placeholder="如：商丘中心站→界沟" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="线路类型">
              <el-radio-group v-model="editing.route_type">
                <el-radio :value="1">城乡公交</el-radio>
                <el-radio :value="2">城际客运</el-radio>
                <el-radio :value="3">旅游专线</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="24">
          <el-col :span="6">
            <el-form-item label="时长(分钟)">
              <el-input-number v-model="editing.duration_minutes" :min="0" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="总里程(km)">
              <el-input-number v-model="editing.distance_km" :min="0" :precision="1" />
              <span class="form-tip">留0自动算</span>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="起步价">
              <el-input-number v-model="editing.min_fare" :min="0" :precision="2" :step="1" />
              <span class="form-tip">0=不启用</span>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="状态">
              <el-switch v-model="editing.status" :active-value="1" :inactive-value="0" active-text="运营" inactive-text="停运" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <!-- 站点序列区 -->
    <div class="card-box">
      <div class="section-header">
        <span class="section-title">站点序列与累计票价</span>
        <span class="station-count">共 {{ editing.stations.length }} 站</span>
      </div>
      <div class="tip-banner">
        <span class="tip-label">定价规则</span>
        每个站点的"累计票价"= 从起点到该站的总票价。起点站必须填 0。
        任意两站之间票价 = |下车站累计价 - 上车站累计价|。
        例如：站3累计价4元、站9累计价13元，则站3→站9 = 13-4 = 9元。
        <br/>
        <span class="tip-label" style="background:#409eff">起步价</span>
        设定起步价后，实际票价 = max(起步价, 累计差额)。
        多站同价（如前4站都填0）时，站间票价不低于起步价，避免"零元购"。
      </div>
      <div class="station-list">
        <div v-for="(s, idx) in editing.stations" :key="idx" class="station-row">
          <div class="station-row-main">
            <div class="station-index" :class="{ 'is-start': idx === 0, 'is-end': idx === editing.stations.length - 1 }">
              {{ idx + 1 }}
            </div>
            <el-select v-model="s.station_id" placeholder="选择站点" filterable style="flex: 1; min-width: 160px" @change="onStationChange">
              <el-option v-for="st in stations" :key="st.id" :label="st.name" :value="st.id" :disabled="isStationUsed(st.id, idx)" />
            </el-select>
            <div class="station-actions">
              <el-button :icon="ArrowUp" :disabled="idx === 0" circle size="small" @click="moveStation(idx, -1)" />
              <el-button :icon="ArrowDown" :disabled="idx === editing.stations.length - 1" circle size="small" @click="moveStation(idx, 1)" />
              <el-button type="danger" :icon="Delete" circle size="small" :disabled="editing.stations.length <= 2" @click="removeStation(idx)" />
            </div>
          </div>
          <div class="station-row-fields">
            <div class="field-group">
              <span class="field-label">累计票价</span>
              <el-input-number v-model="s.price" :min="0" :precision="2" :step="1" style="width: 110px" :class="{ 'price-zero': s.price === 0 && idx === 0 }" />
              <span class="field-unit">元</span>
            </div>
            <div class="field-group">
              <span class="field-label">累计里程</span>
              <el-input-number v-model="s.distance_km" :min="0" :precision="1" :step="1" style="width: 110px" />
              <span class="field-unit">km</span>
            </div>
            <div class="field-group">
              <span class="field-label">到达时刻</span>
              <el-time-select v-model="s.arrival_time" start="00:00" step="00:05" end="23:55" placeholder="时刻" style="width: 100px" />
            </div>
            <div class="field-group">
              <span class="field-label">到达日</span>
              <el-select v-model="s.arrival_day_offset" style="width: 90px" placeholder="天数">
                <el-option label="当天" :value="0" />
                <el-option label="次日" :value="1" />
                <el-option label="第2天" :value="2" />
                <el-option label="第3天" :value="3" />
              </el-select>
            </div>
          </div>
        </div>
      </div>
      <div class="add-station-bar">
        <el-button type="primary" :icon="Plus" @click="addStation">添加站点</el-button>
      </div>
    </div>

    <!-- 可视化票价面板 -->
    <div class="card-box" v-if="validStations.length >= 2">
      <div class="section-header">
        <span class="section-title">区间票价可视化面板</span>
      </div>

      <!-- 路线示意图 -->
      <div class="route-diagram">
        <div class="diagram-stations">
          <template v-for="(s, idx) in validStations" :key="idx">
            <div class="diagram-node" :class="{ 'node-start': idx === 0, 'node-end': idx === validStations.length - 1 }">
              <div class="node-circle">{{ idx + 1 }}</div>
              <div class="node-name">{{ getStationName(s.station_id) }}</div>
              <div class="node-price">¥{{ s.price }}</div>
            </div>
            <div v-if="idx < validStations.length - 1" class="diagram-segment">
              <div class="segment-price">
                ¥{{ (() => { let f = Math.abs(validStations[idx + 1].price - s.price); return (editing.min_fare > 0 && f < editing.min_fare ? editing.min_fare : f).toFixed(2) })() }}
              </div>
              <div class="segment-line"></div>
            </div>
          </template>
        </div>
      </div>

      <!-- 票价矩阵表 -->
      <div class="matrix-wrapper">
        <table class="price-matrix">
          <thead>
            <tr>
              <th class="matrix-corner">上车 →<br/>↓ 下车</th>
              <th v-for="(s, idx) in validStations" :key="'h-' + idx" class="matrix-header" :class="{ 'header-start': idx === 0, 'header-end': idx === validStations.length - 1 }">
                <el-tooltip :content="getStationName(s.station_id)" placement="top" :show-after="300">
                  <div class="header-content">
                    <span class="header-idx">{{ idx + 1 }}</span>
                    <span class="header-name">{{ getStationName(s.station_id) }}</span>
                  </div>
                </el-tooltip>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(from, i) in validStations" :key="'r-' + i">
              <td class="matrix-row-header" :class="{ 'row-start': i === 0, 'row-end': i === validStations.length - 1 }">
                <el-tooltip :content="getStationName(from.station_id)" placement="left" :show-after="300">
                  <div class="header-content">
                    <span class="header-idx">{{ i + 1 }}</span>
                    <span class="header-name">{{ getStationName(from.station_id) }}</span>
                  </div>
                </el-tooltip>
              </td>
              <td v-for="(to, j) in validStations" :key="'c-' + i + '-' + j" class="matrix-cell" :class="cellClass(i, j)">
                <span v-if="i === j" class="cell-dash">—</span>
                <span v-else class="cell-price">¥{{ priceBetween(i, j) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 统计信息 -->
      <div class="matrix-stats">
        <div class="stat-item">
          <span class="stat-label">站点数</span>
          <span class="stat-value">{{ validStations.length }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">全程票价</span>
          <span class="stat-value">¥{{ (validStations[validStations.length - 1].price - validStations[0].price).toFixed(2) }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">全程里程</span>
          <span class="stat-value">{{ (validStations[validStations.length - 1].distance_km - validStations[0].distance_km).toFixed(1) }}km</span>
        </div>
        <div class="stat-item" v-if="editing.duration_minutes > 0">
          <span class="stat-label">运行时长</span>
          <span class="stat-value">{{ editing.duration_minutes }}分钟</span>
        </div>
      </div>
    </div>

    <!-- 底部操作栏 -->
    <div class="page-footer">
      <el-button @click="goBack">取消</el-button>
      <el-button v-if="isEdit && validStations.length >= 2" @click="openReverseDialog">生成反向线路</el-button>
      <el-button type="primary" @click="handleSave">保存线路</el-button>
    </div>

    <!-- 反向线路生成弹窗 -->
    <el-dialog v-model="showReverseDialog" title="生成反向线路" width="560px" :close-on-click-modal="false">
      <el-form :model="reverseConfig" label-width="100px">
        <el-form-item label="线路名称" required>
          <el-input v-model="reverseConfig.name" placeholder="如：界沟→商丘中心站" />
        </el-form-item>
        <el-form-item label="线路类型">
          <el-radio-group v-model="reverseConfig.route_type">
            <el-radio :value="1">城乡公交</el-radio>
            <el-radio :value="2">城际客运</el-radio>
            <el-radio :value="3">旅游专线</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="运行时长">
              <el-input-number v-model="reverseConfig.duration_minutes" :min="0" />
              <span class="form-tip">分钟</span>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="发车时间">
              <el-time-select v-model="reverseConfig.departure_time" start="00:00" step="00:05" end="23:55" placeholder="选择发车时间" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="状态">
          <el-switch v-model="reverseConfig.status" :active-value="1" :inactive-value="0" active-text="运营" inactive-text="停运" />
        </el-form-item>
      </el-form>
      <!-- 预览面板 -->
      <div class="reverse-preview" v-if="reversePreview.length >= 2">
        <div class="reverse-preview-header">
          <span class="rp-title">预览：站点反转 + 票价自动反算</span>
          <span class="rp-count">共 {{ reversePreview.length }} 站</span>
        </div>
        <div class="rp-stations">
          <div v-for="(s, idx) in reversePreview" :key="idx" class="rp-station" :class="{ 'rp-is-start': idx === 0, 'rp-is-end': idx === reversePreview.length - 1 }">
            <span class="rp-idx">{{ idx + 1 }}</span>
            <span class="rp-name">{{ getStationName(s.station_id) }}</span>
            <span class="rp-price">¥{{ s.price.toFixed(2) }}</span>
            <span class="rp-dist">{{ s.distance_km.toFixed(1) }}km</span>
            <span class="rp-time">{{ s.arrival_day_offset > 0 ? '+' + s.arrival_day_offset + '天 ' : '' }}{{ s.arrival_time || '--:--' }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showReverseDialog = false">取消</el-button>
        <el-button type="primary" @click="generateReverse">生成反向线路</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, ArrowUp, ArrowDown, Delete, Plus } from '@element-plus/icons-vue'
import { routeApi, stationApi } from '@/api'

const router = useRouter()
const route = useRoute()

const isEdit = computed(() => !!route.query.id)
const stations = ref<any[]>([])

const emptyEditing = () => ({
  id: 0 as number,
  name: '' as string,
  route_type: 1 as number,
  duration_minutes: 0 as number,
  distance_km: 0 as number,
  min_fare: 0 as number,
  status: 1 as number,
  force: false as boolean,
  stations: [
    { station_id: 0, price: 0, distance_km: 0, arrival_time: '', arrival_day_offset: 0 },
    { station_id: 0, price: 0, distance_km: 0, arrival_time: '', arrival_day_offset: 0 },
  ] as { station_id: number; price: number; distance_km: number; arrival_time: string; arrival_day_offset: number }[],
})

const editing = reactive(emptyEditing())

// 有效站点（已选择站点ID的）
const validStations = computed(() => editing.stations.filter(s => s.station_id > 0))

const goBack = () => router.push('/ticket/routes')

const getStationName = (id: number) => {
  const st = stations.value.find((s: any) => s.id === id)
  return st ? st.name : `站点#${id}`
}

// 两站之间的票价（含起步价逻辑）
const priceBetween = (i: number, j: number) => {
  const from = validStations.value[i]
  const to = validStations.value[j]
  if (!from || !to) return '0.00'
  let fare = Math.abs(to.price - from.price)
  if (editing.min_fare > 0 && fare < editing.min_fare) {
    fare = editing.min_fare
  }
  return fare.toFixed(2)
}

// 矩阵单元格样式
const cellClass = (i: number, j: number) => {
  if (i === j) return 'cell-diagonal'
  if (i < j) return 'cell-forward' // 正向
  return 'cell-backward' // 反向
}

const isStationUsed = (stationId: number, currentIndex: number) => {
  return editing.stations.some((s, idx) => idx !== currentIndex && s.station_id === stationId)
}

const onStationChange = () => { /* placeholder */ }

const addStation = () => {
  editing.stations.push({ station_id: 0, price: 0, distance_km: 0, arrival_time: '', arrival_day_offset: 0 })
}

const removeStation = (idx: number) => {
  editing.stations.splice(idx, 1)
}

const moveStation = (idx: number, dir: number) => {
  const newIdx = idx + dir
  if (newIdx < 0 || newIdx >= editing.stations.length) return
  const tmp = editing.stations[idx]
  editing.stations[idx] = editing.stations[newIdx]
  editing.stations[newIdx] = tmp
}

const loadData = async () => {
  const stRes: any = await stationApi.all()
  stations.value = stRes.data || []

  if (route.query.id) {
    const id = Number(route.query.id)
    const res: any = await routeApi.all()
    const routeData = (res.data || []).find((r: any) => r.id === id)
    if (routeData) {
      Object.assign(editing, {
        id: routeData.id,
        name: routeData.name,
        route_type: routeData.route_type || 1,
        duration_minutes: routeData.duration_minutes || 0,
        distance_km: routeData.distance_km || 0,
        min_fare: routeData.min_fare || 0,
        status: routeData.status,
      })
      try {
        const stRes2: any = await routeApi.stations(id)
        if (stRes2.data && stRes2.data.length > 0) {
          editing.stations = stRes2.data.map((rs: any) => ({
            station_id: rs.station_id,
            price: rs.price || 0,
            distance_km: rs.distance_km || 0,
            arrival_time: rs.arrival_time ? rs.arrival_time.substring(0, 5) : '',
            arrival_day_offset: rs.arrival_day_offset || 0,
          }))
        }
      } catch (e) { /* 旧线路可能无站点序列 */ }
    }
  }
}

const handleSave = async () => {
  if (!editing.name) { ElMessage.warning('请填写线路名称'); return }
  if (editing.stations.length < 2) { ElMessage.warning('至少需要2个站点'); return }
  for (let i = 0; i < editing.stations.length; i++) {
    if (!editing.stations[i].station_id) { ElMessage.warning(`第${i + 1}个站点未选择`); return }
  }
  if (editing.stations[0].price !== 0) {
    ElMessage.warning('起点站累计票价必须为0'); return
  }
  // 票价得一路涨上去，不能回头
  for (let i = 1; i < editing.stations.length; i++) {
    if (editing.stations[i].price < editing.stations[i - 1].price) {
      ElMessage.warning(`第${i + 1}站累计票价不应低于前一站（当前${editing.stations[i].price} < ${editing.stations[i - 1].price}）`)
      return
    }
  }
  await doSave(false)
}

const doSave = async (force: boolean) => {
  try {
    const payload = { ...editing, force }
    if (editing.id) {
      await routeApi.update(editing.id, payload)
    } else {
      await routeApi.create(payload)
    }
    ElMessage.success('保存成功')
    router.push('/ticket/routes')
  } catch (e: any) {
    const detail = e?.response?.data?.data
    if (detail?.active_order_count > 0 && !force) {
      try {
        await ElMessageBox.confirm(
          `该线路有 ${detail.active_order_count} 个活跃订单。修改站点序列可能影响已有订单的票价计算。\n\n是否强制保存？`,
          '活跃订单提示',
          { type: 'warning', confirmButtonText: '强制保存', cancelButtonText: '取消', confirmButtonClass: 'el-button--warning' }
        )
        await doSave(true)
      } catch { /* 用户取消 */ }
    }
  }
}

// 反向线路生成
const showReverseDialog = ref(false)
const reverseConfig = reactive({
  name: '',
  route_type: 1,
  duration_minutes: 0,
  departure_time: '',
  status: 1,
})

const openReverseDialog = () => {
  const valid = editing.stations.filter(s => s.station_id > 0)
  if (valid.length < 2) {
    ElMessage.warning('至少需要2个有效站点才能生成反向线路')
    return
  }
  // 自动生成反向名称：原终点→原起点
  const firstName = getStationName(valid[0].station_id)
  const lastName = getStationName(valid[valid.length - 1].station_id)
  reverseConfig.name = `${lastName}→${firstName}`
  reverseConfig.route_type = editing.route_type
  reverseConfig.duration_minutes = editing.duration_minutes
  reverseConfig.departure_time = ''
  reverseConfig.status = editing.status
  showReverseDialog.value = true
}

// 反向预览数据（站点反转 + 票价/里程反算 + 时刻按里程比例分配）
const reversePreview = computed(() => {
  const valid = editing.stations.filter(s => s.station_id > 0)
  if (valid.length < 2) return []

  const startPrice = valid[0].price
  const fullPrice = valid[valid.length - 1].price - startPrice
  const startDist = valid[0].distance_km
  const fullDist = valid[valid.length - 1].distance_km - startDist

  // 反转站点，反算累计票价和里程
  // 公式：反向第i站Price = 全程价 - (原站Price - 起点Price)
  const reversed = [...valid].reverse().map((s, idx) => ({
    station_id: s.station_id,
    price: idx === 0 ? 0 : Math.max(0, fullPrice - (s.price - startPrice)),
    distance_km: idx === 0 ? 0 : Math.max(0, fullDist - (s.distance_km - startDist)),
    arrival_time: '',
    arrival_day_offset: 0,
  }))

  // 按发车时间 + 里程比例计算各站到达时刻
  if (reverseConfig.departure_time && reverseConfig.duration_minutes > 0 && fullDist > 0) {
    const [h, m] = reverseConfig.departure_time.split(':').map(Number)
    const startMin = h * 60 + m
    const totalDur = reverseConfig.duration_minutes

    reversed.forEach((s, idx) => {
      if (idx === 0) {
        s.arrival_time = reverseConfig.departure_time
        s.arrival_day_offset = 0
      } else {
        const ratio = s.distance_km / fullDist
        const arriveMin = Math.round(startMin + totalDur * ratio)
        const days = Math.floor(arriveMin / (60 * 24))
        const hh = Math.floor(arriveMin / 60) % 24
        const mm = arriveMin % 60
        s.arrival_time = `${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}`
        s.arrival_day_offset = days
      }
    })
  }

  return reversed
})

const generateReverse = async () => {
  if (!reverseConfig.name) {
    ElMessage.warning('请填写线路名称')
    return
  }
  const preview = reversePreview.value
  if (preview.length < 2) {
    ElMessage.warning('有效站点不足')
    return
  }

  // 用反向数据填充editing，切换为新增模式
  const payload = {
    ...editing,
    id: 0,
    name: reverseConfig.name,
    route_type: reverseConfig.route_type,
    duration_minutes: reverseConfig.duration_minutes,
    distance_km: preview[preview.length - 1].distance_km,
    status: reverseConfig.status,
    force: false,
    stations: preview.map(s => ({ ...s })),
  }

  showReverseDialog.value = false
  try {
    ElMessage.info('正在创建反向线路...')
    await routeApi.create(payload)
    ElMessage.success('反向线路创建成功')
    router.push('/ticket/routes')
  } catch (e: any) {
    const detail = e?.response?.data?.data
    if (detail?.active_order_count > 0) {
      try {
        await ElMessageBox.confirm(
          `该线路有 ${detail.active_order_count} 个活跃订单。是否强制创建？`,
          '活跃订单提示',
          { type: 'warning', confirmButtonText: '强制创建', cancelButtonText: '取消' }
        )
        await routeApi.create({ ...payload, force: true })
        ElMessage.success('反向线路创建成功')
        router.push('/ticket/routes')
      } catch { /* 用户取消 */ }
    } else {
      ElMessage.error(e?.response?.data?.message || '创建失败，请检查数据')
    }
  }
}

onMounted(loadData)
</script>

<style scoped>
.route-edit-page {
  padding: 16px;
  padding-bottom: 80px;
}

.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  position: relative;
  padding-left: 12px;
}

.section-title::before {
  content: '';
  position: absolute;
  left: 0;
  top: 2px;
  width: 4px;
  height: 16px;
  background: #409eff;
  border-radius: 2px;
}

.form-tip {
  margin-left: 8px;
  color: #999;
  font-size: 12px;
}

/* 站点序列 */
.tip-banner {
  background: #f7f8fa;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  padding: 10px 14px;
  margin-bottom: 16px;
  font-size: 13px;
  color: #595959;
  line-height: 1.7;
}

.tip-label {
  display: inline-block;
  background: #1a1a1a;
  color: #fff;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 3px;
  margin-right: 8px;
  font-weight: 600;
}

.station-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.station-count {
  font-size: 13px;
  color: #999;
}

.add-station-bar {
  position: sticky;
  bottom: 70px;
  display: flex;
  justify-content: flex-start;
  padding: 8px 0;
  z-index: 50;
}

.add-station-bar .el-button {
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.3);
}

.station-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 16px;
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  transition: all 0.2s;
}

.station-row-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.station-row-fields {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-left: 44px;
  flex-wrap: wrap;
}

.station-row:hover {
  border-color: #d0d0d0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.station-index {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #1a1a1a;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
}

.station-index.is-start {
  background: #409eff;
}

.station-index.is-end {
  background: #1a1a1a;
  border: 2px solid #409eff;
}

/* station-fields 已合并到 station-row-main 和 station-row-fields */

.field-group {
  display: flex;
  align-items: center;
  gap: 6px;
}

.field-label {
  font-size: 12px;
  color: #999;
  white-space: nowrap;
}

.field-unit {
  font-size: 13px;
  color: #999;
}

.station-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

/* 路线示意图 */
.route-diagram {
  overflow-x: auto;
  padding: 24px 16px 16px;
  margin-bottom: 24px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #f0f0f0;
}

.diagram-stations {
  display: flex;
  align-items: flex-start;
  min-width: max-content;
}

.diagram-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  width: 100px;
}

.node-circle {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  border: 2px solid #d0d0d0;
  color: #595959;
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 6px;
}

.node-start .node-circle {
  background: #409eff;
  border-color: #409eff;
  color: #fff;
}

.node-end .node-circle {
  background: #1a1a1a;
  border-color: #1a1a1a;
  color: #fff;
}

.node-name {
  font-size: 12px;
  color: #1a1a1a;
  text-align: center;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 2px;
  word-break: break-all;
}

.node-price {
  font-size: 11px;
  color: #409eff;
  font-weight: 600;
}

.diagram-segment {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  padding-top: 8px;
  min-width: 60px;
}

.segment-price {
  font-size: 11px;
  color: #1a1a1a;
  font-weight: 600;
  background: #fff;
  padding: 2px 8px;
  border-radius: 10px;
  border: 1px solid #e8e8e8;
  margin-bottom: 4px;
  white-space: nowrap;
}

.segment-line {
  width: 100%;
  min-width: 40px;
  height: 2px;
  background: linear-gradient(90deg, #d0d0d0, #409eff, #d0d0d0);
  position: relative;
}

.segment-line::after {
  content: '';
  position: absolute;
  right: -4px;
  top: -3px;
  width: 0;
  height: 0;
  border-left: 6px solid #409eff;
  border-top: 4px solid transparent;
  border-bottom: 4px solid transparent;
}

/* 票价矩阵 */
.matrix-wrapper {
  overflow-x: auto;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
}

.price-matrix {
  border-collapse: collapse;
  font-size: 13px;
  white-space: nowrap;
}

.price-matrix th,
.price-matrix td {
  border: 1px solid #f0f0f0;
  padding: 8px 12px;
  text-align: center;
}

.matrix-corner {
  background: #1a1a1a !important;
  color: #fff !important;
  font-size: 11px !important;
  font-weight: 500 !important;
  position: sticky;
  left: 0;
  z-index: 2;
}

.matrix-header {
  background: #fafafa;
  font-weight: normal;
  min-width: 80px;
  max-width: 100px;
}

.header-idx {
  display: inline-block;
  width: 18px;
  height: 18px;
  line-height: 18px;
  border-radius: 50%;
  background: #1a1a1a;
  color: #fff;
  font-size: 11px;
  margin-right: 4px;
}

.header-start .header-idx {
  background: #409eff;
}

.header-end .header-idx {
  background: #1a1a1a;
  border: 1px solid #409eff;
}

.header-content {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  overflow: hidden;
}

.header-name {
  font-size: 12px;
  color: #1a1a1a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 70px;
}

.matrix-row-header {
  background: #fafafa;
  text-align: left;
  position: sticky;
  left: 0;
  z-index: 1;
  min-width: 80px;
  max-width: 100px;
}

.row-start .header-idx {
  background: #409eff;
}

.matrix-cell {
  min-width: 60px;
}

.cell-diagonal {
  background: #f5f5f5;
}

.cell-dash {
  color: #ccc;
  font-size: 14px;
}

.cell-price {
  font-weight: 600;
  color: #1a1a1a;
}

.cell-forward {
  background: #ecf5ff;
}

.cell-forward .cell-price {
  color: #409eff;
}

.cell-backward {
  background: #fff;
}

/* 统计信息 */
.matrix-stats {
  display: flex;
  gap: 24px;
  margin-top: 16px;
  padding: 12px 16px;
  background: #fafafa;
  border-radius: 6px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.stat-label {
  font-size: 12px;
  color: #999;
}

.stat-value {
  font-size: 16px;
  font-weight: 700;
  color: #1a1a1a;
}

/* 底部操作栏 */
.page-footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 12px 24px;
  background: #fff;
  border-top: 1px solid #e8e8e8;
  z-index: 100;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
}

/* 价格输入框值为0时高亮起点 */
:deep(.price-zero .el-input__wrapper) {
  background: #ecf5ff;
}

/* 反向线路预览 */
.reverse-preview {
  margin-top: 16px;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  overflow: hidden;
}

.reverse-preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #1a1a1a;
  color: #fff;
}

.rp-title {
  font-size: 13px;
  font-weight: 600;
}

.rp-count {
  font-size: 12px;
  color: #999;
}

.rp-stations {
  max-height: 280px;
  overflow-y: auto;
}

.rp-station {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid #f0f0f0;
  font-size: 13px;
}

.rp-station:last-child {
  border-bottom: none;
}

.rp-station.rp-is-start {
  background: #ecf5ff;
}

.rp-station.rp-is-end {
  background: #fafafa;
}

.rp-idx {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #1a1a1a;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
}

.rp-is-start .rp-idx {
  background: #409eff;
}

.rp-name {
  flex: 1;
  color: #1a1a1a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rp-price {
  color: #409eff;
  font-weight: 600;
  min-width: 50px;
  text-align: right;
}

.rp-dist {
  color: #999;
  min-width: 55px;
  text-align: right;
}

.rp-time {
  color: #595959;
  min-width: 45px;
  text-align: right;
}
</style>
