// 管理后台 - 车辆轨迹（运行中班次实时监控 + 历史轨迹回放），接 /api/wx/admin/trips/active + /api/wx/admin/trips/:id/track
const { request } = require('../../utils/request')

Page({
  data: {
    list: [],
    loading: true,
    viewMode: 'list',      // list / detail
    selectedTripId: 0,
    detail: null,          // 班次信息
    trackLoading: false,
    markers: [],
    polyline: [],
    includePoints: [],
    centerLng: 116.40,
    centerLat: 39.90,
    hasTrack: false,
    pointCount: 0,
    stationCount: 0
  },

  onShow() {
    // 每次进入页面刷新运行中班次列表
    if (this.data.viewMode === 'list') {
      this.loadActiveTrips()
    }
  },

  loadActiveTrips() {
    this.setData({ loading: true })
    request({ url: '/api/wx/admin/trips/active', method: 'GET', silent: true }).then(res => {
      const d = res.data || {}
      const raw = d.list || []
      const items = raw.map(t => ({
        trip_id: t.trip_id,
        trip_no: t.trip_no || '-',
        trip_date: t.trip_date || '',
        departure_time: t.departure_time || '',
        arrival_time: t.arrival_time || '',
        route_name: t.route_name || (t.from_station + ' → ' + t.to_station),
        from_station: t.from_station || '-',
        to_station: t.to_station || '-',
        driver_name: t.driver_name || '未分配',
        driver_phone: t.driver_phone || '',
        vehicle_plate_no: t.vehicle_plate_no || '-',
        passed_order: t.passed_order || 0,
        total_stations: t.total_stations || 0,
        has_gps: t.longitude != null && t.latitude != null,
        reported_at: t.reported_at || '',
        seconds_ago: t.seconds_ago != null ? t.seconds_ago : -1,
        freshness: this.freshnessText(t.seconds_ago)
      }))
      this.setData({ list: items, loading: false })
    }).catch(() => {
      this.setData({ loading: false, list: [] })
    })
  },

  freshnessText(secondsAgo) {
    if (secondsAgo == null || secondsAgo < 0) return '无定位'
    if (secondsAgo < 60) return secondsAgo + '秒前'
    if (secondsAgo < 300) return Math.floor(secondsAgo / 60) + '分钟前'
    return '已过期'
  },

  onTripTap(e) {
    const id = e.currentTarget.dataset.id
    this.setData({
      viewMode: 'detail',
      selectedTripId: id,
      detail: null,
      trackLoading: true,
      hasTrack: false,
      markers: [],
      polyline: [],
      includePoints: []
    })
    request({ url: `/api/wx/admin/trips/${id}/track`, method: 'GET', silent: true }).then(res => {
      const d = res.data || {}
      const trip = d.trip || {}
      const points = d.points || []
      const stations = d.stations || []

      const detail = {
        trip_no: trip.trip_no || '-',
        trip_date: trip.trip_date || '',
        departure_time: trip.departure_time || '',
        arrival_time: trip.arrival_time || '',
        route_name: trip.route_name || '',
        from_station: trip.from_station || '-',
        to_station: trip.to_station || '-',
        driver_name: trip.driver_name || '未分配',
        vehicle_plate: trip.vehicle_plate || '-',
        passed_order: trip.passed_order || 0
      }

      // 拼地图数据
      const markers = []
      const polyPoints = []
      const includePoints = []

      // GPS轨迹点
      points.forEach(p => {
        polyPoints.push({ latitude: p.latitude, longitude: p.longitude })
      })

      // 站点标记
      stations.forEach((s, idx) => {
        if (s.longitude && s.latitude) {
          markers.push({
            id: idx + 1,
            latitude: s.latitude,
            longitude: s.longitude,
            width: 24,
            height: 24,
            anchor: { x: 0.5, y: 0.5 },
            callout: {
              content: s.stop_order + '. ' + s.name,
              color: '#ffffff',
              fontSize: 11,
              bgColor: '#1a1a1a',
              borderRadius: 6,
              padding: 5,
              display: 'BYCLICK'
            }
          })
          includePoints.push({ latitude: s.latitude, longitude: s.longitude })
        }
      })

      // 车辆当前位置标记（最后一个轨迹点）
      let centerLng = 116.40, centerLat = 39.90
      if (polyPoints.length > 0) {
        const last = polyPoints[polyPoints.length - 1]
        markers.push({
          id: 0,
          latitude: last.latitude,
          longitude: last.longitude,
          iconPath: '/images/map-marker-vehicle.png',
          width: 36,
          height: 36,
          anchor: { x: 0.5, y: 0.5 },
          callout: {
            content: '车辆位置',
            color: '#ffffff',
            fontSize: 12,
            bgColor: '#1a1a1a',
            borderRadius: 8,
            padding: 6,
            display: 'ALWAYS'
          }
        })
        centerLng = last.longitude
        centerLat = last.latitude
        includePoints.push({ latitude: last.latitude, longitude: last.longitude })
      }

      const polyline = polyPoints.length >= 2 ? [{
        points: polyPoints,
        color: '#1a1a1aCC',
        width: 4,
        dottedLine: false,
        arrowLine: true
      }] : []

      this.setData({
        detail,
        trackLoading: false,
        markers,
        polyline,
        includePoints,
        centerLng,
        centerLat,
        hasTrack: polyPoints.length > 0,
        pointCount: polyPoints.length,
        stationCount: markers.filter(m => m.id > 0).length
      })
    }).catch(() => {
      this.setData({ trackLoading: false })
      wx.showToast({ title: '加载轨迹失败', icon: 'none' })
    })
  },

  backToList() {
    this.setData({ viewMode: 'list', detail: null, selectedTripId: 0 })
    this.loadActiveTrips()
  },

  onPullDownRefresh() {
    if (this.data.viewMode === 'list') this.loadActiveTrips()
    wx.stopPullDownRefresh()
  },

  onCallDriver(e) {
    const phone = e.currentTarget.dataset.phone
    if (phone) wx.makePhoneCall({ phoneNumber: phone })
  }
})
