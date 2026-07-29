<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="track-page">
    <div ref="mapContainer" class="map-container"></div>

    <!-- 礼品袋U形提手 + 吊牌（顶部悬浮） -->
    <div class="gift-cluster">

      <!-- U形提手环（礼品袋手柄） -->
      <div class="gift-arch" :class="{ lit: !!activePanel }">
        <span class="arch-knot left" :class="{ active: activePanel === 'active' }"></span>
        <span class="arch-knot right" :class="{ active: activePanel === 'history' }"></span>
      </div>

      <!-- 两个吊牌 -->
      <div class="tag-row">
        <div class="hang-tag" :class="{ pulled: activePanel === 'active' }" @click="togglePanel('active')">
          <span class="tag-hole"></span>
          <span class="tag-text">运行中</span>
          <span class="tag-count" v-if="activeTrips.length > 0">{{ activeTrips.length }}</span>
        </div>
        <div class="hang-tag" :class="{ pulled: activePanel === 'history' }" @click="togglePanel('history')">
          <span class="tag-hole"></span>
          <span class="tag-text">历史</span>
        </div>
      </div>

      <!-- 从上方抽出的磨砂玻璃面板 -->
      <transition name="pullout">
        <div v-if="activePanel" class="gift-panel">
          <!-- 运行中 -->
          <div v-if="activePanel === 'active'" class="panel-body" v-loading="activeLoading">
            <div v-if="refreshing" class="refresh-bar"><span class="refresh-text">刷新中...</span></div>
            <div
              v-for="trip in activeTrips"
              :key="trip.trip_id"
              class="trip-card"
              :class="{ selected: selectedTripId === trip.trip_id }"
              @click="loadTrack(trip.trip_id)"
            >
              <div class="trip-head">
                <span class="trip-no">{{ trip.trip_no }}</span>
                <span class="status-tag" :class="trip.seconds_ago !== null && trip.seconds_ago < 300 ? 'on' : 'off'">
                  {{ trip.seconds_ago !== null && trip.seconds_ago < 300 ? '在线' : '离线' }}
                </span>
              </div>
              <div class="trip-route">{{ trip.from_station }} → {{ trip.to_station }}</div>
              <div class="trip-meta">
                <span>{{ trip.driver_name }}</span>
                <span v-if="trip.vehicle_plate_no">{{ trip.vehicle_plate_no }}</span>
              </div>
              <div class="trip-meta" v-if="trip.seconds_ago !== null">
                <span v-if="trip.speed !== null">{{ (trip.speed * 3.6).toFixed(0) }} km/h</span>
                <span>{{ formatAgo(trip.seconds_ago) }}</span>
              </div>
              <div class="trip-meta">
                <span>{{ trip.trip_date }} {{ trip.departure_time }}发车</span>
              </div>
            </div>
            <div v-if="!activeLoading && activeTrips.length === 0" class="empty">暂无运行中班次</div>
          </div>
          <!-- 历史 -->
          <div v-if="activePanel === 'history'" class="panel-body">
            <div class="history-box">
              <el-input v-model="searchTripNo" placeholder="输入班次号" clearable @keyup.enter="searchHistory" />
              <button class="go-btn" @click="searchHistory" :disabled="historyLoading">
                {{ historyLoading ? '...' : '查询' }}
              </button>
            </div>
          </div>
        </div>
      </transition>

      <!-- 刷新提示 -->
      <div class="refresh-hint" v-if="!activePanel">每 {{ TRACK_REFRESH_INTERVAL_SECONDS }} 秒自动刷新</div>
    </div>

    <!-- 班次信息（顶部居中） -->
    <transition name="fade">
      <div v-if="tripInfo" class="map-overlay">
        <div class="overlay-header">
          <span class="overlay-trip-no">{{ tripInfo.trip_no }}</span>
          <span class="overlay-route">{{ tripInfo.from_station }} → {{ tripInfo.to_station }}</span>
        </div>
        <div class="overlay-meta">
          <span>司机：{{ tripInfo.driver_name || '-' }}</span>
          <span>车牌：{{ tripInfo.vehicle_plate || '-' }}</span>
        </div>
        <div class="overlay-meta">
          <span>{{ tripInfo.trip_date }} {{ tripInfo.departure_time }}发车</span>
          <span v-if="trackPoints.length > 0">轨迹点：{{ trackPoints.length }}</span>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { trackApi, tripApi, type ActiveTrip, type TrackPoint, type TrackStation } from '@/api'
import { TRACK_REFRESH_INTERVAL_SECONDS } from '@/utils/constants'

const mapContainer = ref<HTMLElement>()
let map: L.Map | null = null
let trackLayer: L.Polyline | null = null
let stationMarkers: L.Marker[] = []
let vehicleMarker: L.Marker | null = null
let routeLayer: L.Polyline | null = null

const activePanel = ref<'active' | 'history' | null>(null)
const activeTrips = ref<ActiveTrip[]>([])
const activeLoading = ref(false)
const selectedTripId = ref<number | null>(null)
const searchTripNo = ref('')
const historyLoading = ref(false)
const tripInfo = ref<Record<string, any> | null>(null)
const trackPoints = ref<TrackPoint[]>([])
const trackStations = ref<TrackStation[]>([])
const refreshing = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const amapTileUrl = 'https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}'
const amapSubdomains = ['1', '2', '3', '4']

const stationIcon = L.divIcon({ className: 'station-marker', html: '<div class="station-dot"></div>', iconSize: [14, 14], iconAnchor: [7, 7] })
const vehicleIcon = L.divIcon({ className: 'vehicle-marker', html: '<div class="vehicle-dot"></div>', iconSize: [24, 24], iconAnchor: [12, 12] })

const formatAgo = (s: number) => s < 60 ? `${s}秒前` : s < 3600 ? `${Math.floor(s / 60)}分钟前` : `${Math.floor(s / 3600)}小时前`

const togglePanel = (p: 'active' | 'history') => {
  if (activePanel.value === p) { activePanel.value = null } else { activePanel.value = p; if (p === 'active') loadActiveTrips() }
}

const initMap = () => {
  if (!mapContainer.value) return
  map = L.map(mapContainer.value, { center: [34.42, 115.65], zoom: 12, zoomControl: true, attributionControl: false })
  L.tileLayer(amapTileUrl, { subdomains: amapSubdomains, maxZoom: 18 }).addTo(map!)
}

const clearMapLayers = () => {
  if (trackLayer) { map?.removeLayer(trackLayer); trackLayer = null }
  if (routeLayer) { map?.removeLayer(routeLayer); routeLayer = null }
  if (vehicleMarker) { map?.removeLayer(vehicleMarker); vehicleMarker = null }
  stationMarkers.forEach(m => map?.removeLayer(m)); stationMarkers = []
}

const addToMap = (l: L.Layer) => { if (map) l.addTo(map) }

const renderTrack = () => {
  clearMapLayers(); if (!map) return
  const vs = trackStations.value.filter(s => s.longitude && s.latitude)
  if (vs.length >= 2) { routeLayer = L.polyline(vs.map(s => [s.latitude, s.longitude] as [number, number]), { color: '#999', weight: 3, opacity: 0.5, dashArray: '8,6' }); addToMap(routeLayer) }
  vs.forEach(s => { const m = L.marker([s.latitude, s.longitude], { icon: stationIcon }).bindTooltip(`${s.stop_order}. ${s.name}`, { direction: 'top', offset: [0, -8] }); addToMap(m); stationMarkers.push(m) })
  const vp = trackPoints.value.filter(p => p.longitude && p.latitude)
  if (vp.length >= 2) {
    trackLayer = L.polyline(vp.map(p => [p.latitude, p.longitude] as [number, number]), { color: '#1296db', weight: 4, opacity: 0.85 }); addToMap(trackLayer)
    const last = vp[vp.length - 1]
    vehicleMarker = L.marker([last.latitude, last.longitude], { icon: vehicleIcon }).bindTooltip(`速度: ${(last.speed * 3.6).toFixed(0)} km/h`, { direction: 'top', offset: [0, -12] }); addToMap(vehicleMarker)
  }
  const b = L.latLngBounds([])
  vs.forEach(s => b.extend([s.latitude, s.longitude])); vp.forEach(p => b.extend([p.latitude, p.longitude]))
  if (b.isValid()) map.fitBounds(b, { padding: [50, 50], maxZoom: 15 })
}

const loadTrack = async (id: number) => {
  selectedTripId.value = id
  try { const r: any = await trackApi.tripTrack(id); tripInfo.value = r.data.trip; trackPoints.value = r.data.points || []; trackStations.value = r.data.stations || []; renderTrack() }
  catch { ElMessage.error('加载轨迹失败') }
}

const loadActiveTrips = async () => {
  refreshing.value = true
  try {
    const r: any = await trackApi.activeTrips(); activeTrips.value = r.data.list || []
    if (selectedTripId.value) { const u = activeTrips.value.find(t => t.trip_id === selectedTripId.value); if (u && u.longitude && u.latitude && map && vehicleMarker) { vehicleMarker.setLatLng(L.latLng(u.latitude, u.longitude)); vehicleMarker.setTooltipContent(`速度: ${(u.speed ? u.speed * 3.6 : 0).toFixed(0)} km/h`) } }
  } finally { refreshing.value = false }
}

const searchHistory = async () => {
  if (!searchTripNo.value.trim()) { ElMessage.warning('请输入班次号'); return }
  historyLoading.value = true
  try { const r: any = await tripApi.list({ page: 1, page_size: 1, trip_no: searchTripNo.value.trim() }); if (r.data.list?.length > 0) { await loadTrack(r.data.list[0].id) } else { ElMessage.warning('未找到该班次号') } }
  finally { historyLoading.value = false }
}

watch(activePanel, async () => { await nextTick(); if (map) map.invalidateSize() })
onMounted(async () => { await nextTick(); initMap(); loadActiveTrips(); refreshTimer = setInterval(loadActiveTrips, TRACK_REFRESH_INTERVAL_SECONDS * 1000) })
onUnmounted(() => { if (refreshTimer) clearInterval(refreshTimer); if (map) { map.remove(); map = null } })
</script>

<style scoped>
.track-page { position: relative; width: 100%; height: 100%; overflow: hidden; }
.map-container { position: absolute; top: 0; left: 0; width: 100%; height: 100%; z-index: 1; }

/* 礼品袋控制区（顶部右侧悬浮） */
.gift-cluster {
  position: absolute;
  top: 40px;
  right: 140px;
  z-index: 500;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

/* U形提手环（礼品袋手柄 ∩） */
.gift-arch {
  width: 128px;
  height: 22px;
  border: 2px solid rgba(100, 120, 150, 0.28);
  border-bottom: none;
  border-radius: 22px 22px 0 0;
  position: relative;
  transition: border-color 0.4s ease;
}
.gift-arch.lit {
  border-color: rgba(18, 150, 219, 0.4);
}
/* 提手环底部两端绳结 */
.arch-knot {
  position: absolute;
  bottom: -5px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: rgba(100, 120, 150, 0.22);
  border: 1.5px solid rgba(255, 255, 255, 0.45);
  transition: all 0.3s ease;
}
.arch-knot.left { left: -4px; }
.arch-knot.right { right: -4px; }
.arch-knot.active {
  background: rgba(18, 150, 219, 0.55);
  border-color: rgba(18, 150, 219, 0.65);
  box-shadow: 0 0 10px rgba(18, 150, 219, 0.35);
}

/* 吊牌（悬挂在U形提手下方） */
.tag-row {
  display: flex;
  justify-content: space-between;
  width: 128px;
  margin-top: 3px;
}
.hang-tag {
  position: relative;
  padding: 7px 16px 8px;
  background: rgba(255, 255, 255, 0.5);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.5);
  border-top: none;
  border-radius: 0 0 11px 11px;
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.07), inset 0 -1px 0 rgba(255, 255, 255, 0.3);
  color: #303133;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
  user-select: none;
}
/* 吊牌穿绳孔 */
.tag-hole {
  position: absolute;
  top: 2px;
  left: 50%;
  transform: translateX(-50%);
  width: 7px;
  height: 4px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.35);
  z-index: 1;
}
.hang-tag:hover {
  background: rgba(255, 255, 255, 0.72);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12), inset 0 -1px 0 rgba(255, 255, 255, 0.4);
}
.hang-tag.pulled {
  background: rgba(18, 150, 219, 0.1);
  border-color: rgba(18, 150, 219, 0.3);
  color: #1296db;
  box-shadow: 0 4px 16px rgba(18, 150, 219, 0.15);
}
.tag-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: #1296db;
  color: #fff;
  font-size: 10px;
  margin-left: 2px;
}

/* 抽出式磨砂玻璃面板 */
.gift-panel {
  width: 290px;
  max-height: 400px;
  margin-top: 6px;
  background: rgba(255, 255, 255, 0.55);
  backdrop-filter: blur(28px) saturate(180%);
  -webkit-backdrop-filter: blur(28px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.5);
  border-radius: 16px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.12), inset 0 1px 0 rgba(255, 255, 255, 0.4);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  position: relative;
}
/* 玻璃顶部光泽 */
.gift-panel::after {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 35%;
  background: linear-gradient(to bottom, rgba(255, 255, 255, 0.16), transparent);
  border-radius: 16px 16px 0 0;
  pointer-events: none;
  z-index: 0;
}
.panel-body { overflow-y: auto; padding: 8px 10px 10px; flex: 1; position: relative; z-index: 1; }
.panel-body::-webkit-scrollbar { width: 4px; }
.panel-body::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.1); border-radius: 2px; }

/* 刷新条 */
.refresh-bar { text-align: center; padding: 2px 0 4px; }
.refresh-text { font-size: 11px; color: #1296db; }

/* 班次卡片 */
.trip-card {
  background: rgba(255, 255, 255, 0.45);
  border: 1px solid rgba(0, 0, 0, 0.04);
  border-radius: 10px;
  padding: 10px 12px;
  margin-bottom: 6px;
  cursor: pointer;
  transition: all 0.2s;
}
.trip-card:hover { background: rgba(18, 150, 219, 0.08); border-color: rgba(18, 150, 219, 0.3); }
.trip-card.selected { background: rgba(18, 150, 219, 0.1); border-color: rgba(18, 150, 219, 0.4); }
.trip-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.trip-no { font-weight: 600; color: #303133; font-size: 13px; }
.status-tag { font-size: 11px; padding: 1px 7px; border-radius: 8px; }
.status-tag.on { background: rgba(52, 199, 89, 0.15); color: #34c759; }
.status-tag.off { background: rgba(144, 147, 153, 0.12); color: #909399; }
.trip-route { font-size: 13px; color: #303133; margin-bottom: 4px; font-weight: 500; }
.trip-meta { display: flex; justify-content: space-between; font-size: 12px; color: #909399; margin-bottom: 2px; }
.empty { text-align: center; padding: 28px 0; color: #909399; font-size: 13px; }

/* 历史搜索 */
.history-box { display: flex; gap: 8px; padding: 4px 0; }
.history-box :deep(.el-input__wrapper) { background: rgba(255, 255, 255, 0.6); box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08) inset; }
.history-box :deep(.el-input__wrapper.is-focus) { box-shadow: 0 0 0 1px rgba(18, 150, 219, 0.4) inset; }
.go-btn {
  padding: 0 16px; border: none; border-radius: 8px;
  background: #1296db; color: #fff; font-size: 13px;
  cursor: pointer; transition: all 0.2s; white-space: nowrap; flex-shrink: 0;
}
.go-btn:hover { background: #0d7ab0; }
.go-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* 刷新提示 */
.refresh-hint {
  margin-top: 6px;
  font-size: 11px;
  color: rgba(100, 110, 130, 0.55);
  text-align: center;
}

/* 班次信息（顶部居中） */
.map-overlay {
  position: absolute; top: 14px; left: 50%; transform: translateX(-50%);
  background: rgba(255, 255, 255, 0.55);
  backdrop-filter: blur(28px) saturate(180%);
  -webkit-backdrop-filter: blur(28px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.5);
  border-radius: 14px; padding: 10px 18px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1), inset 0 1px 0 rgba(255, 255, 255, 0.4);
  z-index: 400; min-width: 260px; max-width: 420px;
}
.overlay-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.overlay-trip-no { font-weight: 700; color: #303133; font-size: 14px; }
.overlay-route { color: #1296db; font-size: 13px; }
.overlay-meta { display: flex; justify-content: space-between; font-size: 12px; color: #666; margin-bottom: 2px; }

/* 抽出动画（从上方往下展开） */
.pullout-enter-active {
  animation: pullout 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
  transform-origin: top center;
}
.pullout-leave-active {
  animation: pullout 0.3s ease-in reverse;
  transform-origin: top center;
}
@keyframes pullout {
  from {
    opacity: 0;
    clip-path: inset(0 0 100% 0);
    transform: translateY(-8px) scaleY(0.3);
  }
  to {
    opacity: 1;
    clip-path: inset(0 0 0 0);
    transform: translateY(0) scaleY(1);
  }
}
.fade-enter-active, .fade-leave-active { transition: opacity 0.25s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* Leaflet 标记 */
:deep(.station-dot) { width: 12px; height: 12px; background: #1296db; border: 2px solid #fff; border-radius: 50%; box-shadow: 0 1px 4px rgba(0,0,0,0.3); }
:deep(.vehicle-dot) { width: 20px; height: 20px; background: #34c759; border: 3px solid #fff; border-radius: 50%; box-shadow: 0 2px 8px rgba(52,199,89,0.4); }
</style>
