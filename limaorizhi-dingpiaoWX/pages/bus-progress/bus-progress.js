// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')

const { request } = require('../../utils/request')
const { formatArrivalTime } = require('../../utils/order-helper')
const { formatSpeed, formatReportTime, formatCurrentTime } = require('../../utils/format')
const mapHelper = require('../../utils/map-helper')
const { createPollMixin } = require('../../utils/poll-mixin')

const pollMixin = createPollMixin('loadTrip')

Page({
  data: {
    tripId: 0,
    trip: null,
    fromName: '',
    toName: '',
    stations: [],        // 站点列表（含圆圈类型）
    effectivePassedOrder: 0,
    totalStations: 0,
    loading: true,
    loadError: false,
    errorText: '',
    lastUpdateText: '',
    // 地图相关
    location: null,
    markers: [],
    polyline: [],
    includePoints: [],
    hasLocation: false,
    speedText: '',
    centerLng: 115.6562,
    centerLat: 34.0528
  },

  ...pollMixin,
  useFallbackApi: false,  // 无权限时回退到公开API

  onLoad(options) {
    const tripId = options.id
    if (tripId) {
      this.setData({ tripId })
    }
  },

  onShow() {
    this.resetPollState()
    this.useFallbackApi = false
    this.loadTrip()
  },

  onHide() {
    this.clearTimer()
  },

  onUnload() {
    this.clearTimer()
  },

  // 加载班次位置数据（优先使用位置API获取GPS，无权限时回退到公开API）
  loadTrip() {
    const { tripId } = this.data
    if (!tripId) return

    // 无权限时使用回退API
    if (this.useFallbackApi) {
      this.loadTripDetail()
      return
    }

    // 优先尝试位置API（含GPS数据，需有订单权限）
    request({
      url: `/api/wx/trips/${tripId}/location`,
      method: 'GET',
      silent: true
    }).then(res => {
      this.processTripData(res.data.trip, res.data.location, res.data.effective_passed_order)
      // 班次不在运行中时停止轮询
      if (res.data.trip && res.data.trip.status !== 2) {
        this.clearTimer()
        return
      }
      this.adjustPollInterval(true)
    }).catch(err => {
      // 1002 = 未登录、1003 = 无订单权限、3001 = 班次未发车
      // 三种情况都回退到公开API（不需要登录，仍可显示站点时间线，只是无GPS）
      if (err && (err.code === 1002 || err.code === 1003 || err.code === 3001)) {
        this.useFallbackApi = true
        this.loadTripDetail()
        return
      }
      // 网络错误 → 连续失败超过3次显示错误，否则退避重试
      // 注意：adjustPollInterval(false) 内部会递增 consecutiveErrors
      if (this.consecutiveErrors >= 3) {
        this.clearTimer()
        this.setData({ loading: false, loadError: true, errorText: '网络连接失败，请检查网络后重试' })
      } else {
        this.adjustPollInterval(false)
      }
    })
  },

  // 回退：公开班次详情API（无GPS数据，无订单也可访问）
  loadTripDetail() {
    const { tripId } = this.data
    if (!tripId) return

    request({
      url: `/api/wx/trips/${tripId}`,
      method: 'GET',
      silent: true
    }).then(res => {
      const trip = res.data.trip || res.data
      const epo = res.data.effective_passed_order || (trip ? trip.current_passed_order : 0) || 0
      this.processTripData(trip, null, epo)
      // 回退API无实时数据，不轮询
      this.clearTimer()
    }).catch(err => {
      log.error('加载班次详情失败', err)
      this.setData({ loading: false, loadError: true, errorText: '班次信息加载失败，请重试' })
    })
  },

  // 重新加载（用户点击重试按钮）
  retryLoad() {
    this.setData({ loading: true, loadError: false, errorText: '' })
    this.resetPollState()
    this.useFallbackApi = false
    this.loadTrip()
  },

  // 处理班次数据（位置API和回退API共用）
  processTripData(trip, location, effectivePassedOrder) {
    if (!trip) return

    const epo = effectivePassedOrder || trip.current_passed_order || 0
    const fromName = trip.route && trip.route.from_station ? trip.route.from_station.name : ''
    const toName = trip.route && trip.route.to_station ? trip.route.to_station.name : ''
    const routeStations = (trip.route && trip.route.route_stations) ? trip.route.route_stations : []

    // 构建站点列表，计算每个站点的圆圈类型
    // passed=已过(黑色), current=当前位置(蓝色), future=未到达(白色)
    const stations = routeStations.map(function (rs, index) {
      const stopOrder = rs.stop_order || (index + 1)
      let circleType = 'future'
      if (stopOrder < epo) circleType = 'passed'
      else if (stopOrder === epo) circleType = 'current'

      return {
        station_id: rs.station_id,
        stopOrder: stopOrder,
        circleType: circleType,
        stationName: rs.station ? rs.station.name : '',
        arrivalText: formatArrivalTime(rs.arrival_time, rs.arrival_day_offset),
        isFirst: index === 0,
        isLast: index === routeStations.length - 1
      }
    })

    const updateTime = formatCurrentTime()

    const updateData = {
      trip: {
        ...trip,
        arrivalText: formatArrivalTime(trip.arrival_time, trip.arrival_day_offset)
      },
      fromName,
      toName,
      stations,
      effectivePassedOrder: epo,
      totalStations: routeStations.length,
      loading: false,
      lastUpdateText: updateTime + ' 更新'
    }

    // 始终用站点坐标构建地图标记（有GPS时叠加车辆位置）
    const markers = this.buildMarkers(trip, location, epo)
    const polyline = this.buildPolyline(trip, location)
    const includePoints = markers.map(function (mk) {
      return { latitude: mk.latitude, longitude: mk.longitude }
    })
    updateData.markers = markers
    updateData.polyline = polyline
    updateData.includePoints = includePoints

    if (location) {
      // 有GPS位置
      const speed = location.speed || 0
      updateData.location = location
      updateData.hasLocation = true
      updateData.centerLng = location.longitude
      updateData.centerLat = location.latitude
      updateData.speedText = formatSpeed(speed)
      updateData.lastUpdateText = '更新于 ' + formatReportTime(location.reported_at)
    } else {
      // 无GPS位置：若站点有坐标则仍显示站点地图，否则显示占位
      const hasStationCoords = includePoints.length > 0
      updateData.hasLocation = hasStationCoords
      if (includePoints.length > 0) {
        const mid = Math.floor(includePoints.length / 2)
        updateData.centerLng = includePoints[mid].longitude
        updateData.centerLat = includePoints[mid].latitude
      }
      if (trip.status === 1) {
        updateData.lastUpdateText = '班次尚未发车'
      } else if (trip.status === 2) {
        updateData.lastUpdateText = '司机尚未上报位置'
      } else {
        updateData.lastUpdateText = '行程已结束'
      }
    }

    this.setData(updateData)
  },

  // 构建地图标记点（所有站点 + 车辆位置）
  buildMarkers(trip, location, epo) {
    const markers = []
    const passedOrder = epo || this.data.effectivePassedOrder || 0

    // 1. 所有站点标记（从 route_stations 遍历）
    const routeStations = (trip.route && trip.route.route_stations) ? trip.route.route_stations : []
    routeStations.forEach(function (rs, index) {
      const station = rs.station
      if (!station || !station.latitude || !station.longitude) return
      const stopOrder = rs.stop_order || (index + 1)
      const isFirst = index === 0
      const isLast = index === routeStations.length - 1

      let iconPath = '/images/map-marker-start.png'
      let bgColor = '#999999'
      let prefix = ''
      if (isFirst) {
        iconPath = '/images/map-marker-start.png'
        bgColor = '#34c759'
        prefix = '起 '
      } else if (isLast) {
        iconPath = '/images/map-marker-end.png'
        bgColor = '#ff3b30'
        prefix = '终 '
      } else {
        iconPath = '/images/map-marker-start.png'
        if (stopOrder < passedOrder) bgColor = '#999999'      // 已过站：灰色
        else if (stopOrder === passedOrder) bgColor = '#1296db' // 当前站：蓝色
        else bgColor = '#666666'                        // 未到站：深灰
        prefix = '第' + stopOrder + '站 '
      }

      markers.push({
        id: 100 + stopOrder,
        latitude: station.latitude,
        longitude: station.longitude,
        iconPath: iconPath,
        width: isFirst || isLast ? 28 : 22,
        height: isFirst || isLast ? 28 : 22,
        anchor: { x: 0.5, y: 0.5 },
        callout: {
          content: prefix + station.name,
          color: '#ffffff', fontSize: 11,
          bgColor: bgColor, borderRadius: 6, padding: 5,
          display: 'ALWAYS'
        }
      })
    })

    // 2. 车辆位置标记（叠加在站点之上）
    if (location && location.latitude && location.longitude) {
      markers.push(mapHelper.buildVehicleMarker(location))
    }

    return markers
  },

  // 构建路线虚线（所有站点依次连线 → 车辆位置）
  buildPolyline(trip, location) {
    const points = []
    const routeStations = (trip.route && trip.route.route_stations) ? trip.route.route_stations : []

    // 按顺序连接所有有坐标的站点
    routeStations.forEach(function (rs) {
      const station = rs.station
      if (station && station.latitude && station.longitude) {
        points.push({ latitude: station.latitude, longitude: station.longitude })
      }
    })

    // 末尾追加车辆实时位置
    if (location && location.latitude && location.longitude) {
      points.push({ latitude: location.latitude, longitude: location.longitude })
    }

    return mapHelper.buildPolyline(points)
  },

  // 导航到车辆位置
  navigateToVehicle() {
    mapHelper.navigateToVehicle(this.data.location, this.data.fromName, this.data.toName)
  },

  // 分享给好友（分享班次到站进度，好友点开可实时查看车辆位置）
  onShareAppMessage() {
    var trip = this.data.trip
    var title = '狸猫日志售票 · 到站进度'
    if (trip) {
      title = (this.data.fromName || '出发') +
        '→' + (this.data.toName || '到达') +
        ' 实时到站进度 · 狸猫日志售票'
    }
    return {
      title: title,
      path: '/pages/bus-progress/bus-progress?id=' + this.data.tripId
    }
  }
})
