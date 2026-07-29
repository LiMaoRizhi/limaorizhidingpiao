// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')

const { request } = require('../../utils/request')
const { formatArrivalTime } = require('../../utils/order-helper')
const { formatSpeed, formatReportTime } = require('../../utils/format')
const mapHelper = require('../../utils/map-helper')
const { createPollMixin } = require('../../utils/poll-mixin')

const pollMixin = createPollMixin('loadLocation')

Page({
  data: {
    tripId: 0,
    trip: null,
    location: null,
    markers: [],
    polyline: [],
    includePoints: [],
    centerLng: 116.40,
    centerLat: 39.90,
    hasLocation: false,
    lastUpdateText: '',
    fromName: '',
    toName: '',
    speedText: '',
    routeStations: [], // 站点序列
    currentPassedOrder: 0, // 司机手动标记的已过站序号
    effectivePassedOrder: 0, // 有效过站序号（GPS优先+手动标记取max，驱动进度条）
    currentStationName: '' // 当前所在站名（进度条上显示）
  },

  ...pollMixin,

  onLoad(options) {
    const tripId = options.id
    if (tripId) {
      this.setData({ tripId })
    }
  },

  onShow() {
    // 页面显示时开始轮询车辆位置
    // 使用递归setTimeout替代setInterval，支持动态退避间隔
    // 仅调用loadLocation，其完成后通过adjustPollInterval自动调度下一次
    this.resetPollState()
    this.loadLocation()
  },

  onHide() {
    this.clearTimer()
  },

  onUnload() {
    this.clearTimer()
  },

  // 加载车辆实时位置
  loadLocation() {
    const { tripId } = this.data
    if (!tripId) return

    request({
      url: `/api/wx/trips/${tripId}/location`,
      method: 'GET'
    }).then(res => {
      const trip = res.data.trip
      const location = res.data.location

      if (!trip) {
        // 服务器返回成功但trip为空（边缘情况），继续轮询等待数据
        this.adjustPollInterval(true)
        return
      }

      const fromName = trip.route && trip.route.from_station ? trip.route.from_station.name : ''
      const toName = trip.route && trip.route.to_station ? trip.route.to_station.name : ''
      const routeStations = (trip.route && trip.route.route_stations) ? trip.route.route_stations : []
      const currentPassedOrder = trip.current_passed_order || 0
      // 后端统一计算的有效过站序号（GPS优先+手动标记取max），未返回时回退到手动标记
      const effectivePassedOrder = res.data.effective_passed_order || currentPassedOrder
      // 计算当前站名：取已过站序号对应的站点名
      const currentStation = effectivePassedOrder > 0 && effectivePassedOrder <= routeStations.length
        ? routeStations[effectivePassedOrder - 1]
        : null
      const currentStationName = currentStation && currentStation.station ? currentStation.station.name : ''

      if (location) {
        const markers = this.buildMarkers(trip, location)
        const polyline = this.buildPolyline(trip, location)
        const includePoints = markers.map(m => ({ latitude: m.latitude, longitude: m.longitude }))
        const speed = location.speed || 0
        const arrivalText = formatArrivalTime(trip.arrival_time, trip.arrival_day_offset)
        this.setData({
          trip: { ...trip, arrivalText },
          location,
          markers,
          polyline,
          includePoints,
          centerLng: location.longitude,
          centerLat: location.latitude,
          fromName,
          toName,
          routeStations,
          currentPassedOrder,
          effectivePassedOrder,
          currentStationName,
          hasLocation: true,
          lastUpdateText: '更新于 ' + formatReportTime(location.reported_at),
          speedText: formatSpeed(speed)
        })
      } else {
        const arrivalText = formatArrivalTime(trip.arrival_time, trip.arrival_day_offset)
        this.setData({
          trip: { ...trip, arrivalText },
          fromName,
          toName,
          routeStations,
          currentPassedOrder,
          effectivePassedOrder,
          currentStationName,
          hasLocation: false,
          lastUpdateText: '司机尚未上报位置或行程已结束',
          // 清空旧地图标记，避免显示上一次的车辆位置
          markers: [],
          polyline: [],
          vehiclePolyline: [],
          includePoints: [],
          location: null
        })
      }
      // 请求成功，重置退避间隔
      this.adjustPollInterval(true)
    }).catch((e) => {
      log.error('加载车辆位置失败', e)
      // 1002=Token失效（request.js已自动跳转登录）、1003=权限不足、3001=班次不存在
      // 这些是永久性错误，继续轮询只会重复弹toast，应停止
      if (e && (e.code === 1002 || e.code === 1003 || e.code === 3001)) {
        this.clearTimer()
        if (e.code === 1003) {
          this.setData({ hasLocation: false, lastUpdateText: e.message || '无权查看此班次位置' })
        } else if (e.code === 3001) {
          this.setData({ hasLocation: false, lastUpdateText: '班次不存在或已结束' })
        }
        return
      }
      // 网络错误等临时性错误，指数退避重试
      this.adjustPollInterval(false)
      if (this.consecutiveErrors >= 3 && this.data.lastUpdateText.indexOf('网络') === -1) {
        this.setData({ lastUpdateText: '网络异常，正在重试...' })
      }
    })
  },

  // 构建地图标记点（带自定义图标）
  buildMarkers(trip, location) {
    const markers = []

    // 车辆位置marker（深色图标 + 速度信息）
    markers.push(mapHelper.buildVehicleMarker(location))

    // 出发站marker（绿色图标）
    const fromStation = trip.route && trip.route.from_station
    if (fromStation && fromStation.latitude && fromStation.longitude) {
      markers.push({
        id: 2,
        latitude: fromStation.latitude,
        longitude: fromStation.longitude,
        iconPath: '/images/map-marker-start.png',
        width: 30,
        height: 30,
        anchor: { x: 0.5, y: 0.5 },
        callout: {
          content: '出发：' + fromStation.name,
          color: '#ffffff',
          fontSize: 12,
          bgColor: '#34c759',
          borderRadius: 8,
          padding: 6,
          display: 'ALWAYS'
        }
      })
    }

    // 到达站marker（红色图标）
    const toStation = trip.route && trip.route.to_station
    if (toStation && toStation.latitude && toStation.longitude) {
      markers.push({
        id: 3,
        latitude: toStation.latitude,
        longitude: toStation.longitude,
        iconPath: '/images/map-marker-end.png',
        width: 30,
        height: 30,
        anchor: { x: 0.5, y: 0.5 },
        callout: {
          content: '到达：' + toStation.name,
          color: '#ffffff',
          fontSize: 12,
          bgColor: '#ff3b30',
          borderRadius: 8,
          padding: 6,
          display: 'ALWAYS'
        }
      })
    }

    return markers
  },

  // 构建路线连线（出发站→车辆→到达站）
  buildPolyline(trip, location) {
    const points = []
    const fromStation = trip.route && trip.route.from_station
    const toStation = trip.route && trip.route.to_station

    if (fromStation && fromStation.latitude && fromStation.longitude) {
      points.push({ latitude: fromStation.latitude, longitude: fromStation.longitude })
    }
    points.push({ latitude: location.latitude, longitude: location.longitude })
    if (toStation && toStation.latitude && toStation.longitude) {
      points.push({ latitude: toStation.latitude, longitude: toStation.longitude })
    }

    return mapHelper.buildPolyline(points)
  },

  // 导航到车辆当前位置（拉起微信内置地图导航）
  navigateToVehicle() {
    mapHelper.navigateToVehicle(this.data.location, this.data.fromName, this.data.toName)
  },

  // 点击进度条跳转到站进度页面（查看完整站点时间线）
  goToProgressDetail() {
    const { tripId } = this.data
    if (!tripId) return
    wx.navigateTo({
      url: `/pages/bus-progress/bus-progress?id=${tripId}`
    })
  }
})
