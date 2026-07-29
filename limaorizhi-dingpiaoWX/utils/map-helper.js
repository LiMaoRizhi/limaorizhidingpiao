// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
// 地图标记/路线构建公共逻辑（bus-progress / vehicle-track / driver-track 共用）
const { formatSpeed } = require('./format')

/**
 * 导航到车辆位置（拉起微信内置地图导航）
 * @param {object} location - { latitude, longitude }
 * @param {string} fromName - 出发站名
 * @param {string} toName - 到达站名
 */
function navigateToVehicle(location, fromName, toName) {
  if (!location) return
  wx.openLocation({
    latitude: location.latitude,
    longitude: location.longitude,
    scale: 16,
    name: '车辆当前位置',
    address: (fromName || '') + ' → ' + (toName || '')
  })
}

/**
 * 构建车辆位置标记（深色图标 + 速度信息）
 * @param {object} location - { latitude, longitude, speed }
 * @returns {object} marker对象
 */
function buildVehicleMarker(location) {
  var speed = (location && location.speed) || 0
  return {
    id: 1,
    latitude: location.latitude,
    longitude: location.longitude,
    iconPath: '/images/map-marker-vehicle.png',
    width: 36,
    height: 36,
    anchor: { x: 0.5, y: 0.5 },
    callout: {
      content: '车辆当前位置\n' + formatSpeed(speed),
      color: '#ffffff',
      fontSize: 13,
      bgColor: '#1a1a1a',
      borderRadius: 8,
      padding: 8,
      display: 'ALWAYS'
    }
  }
}

/**
 * 构建路线虚线（统一样式：深色虚线+箭头）
 * @param {Array} points - [{ latitude, longitude }, ...]
 * @returns {Array} polyline数组（点数<2时返回空数组）
 */
function buildPolyline(points) {
  if (!points || points.length < 2) return []
  return [{
    points: points,
    color: '#1a1a1aCC',
    width: 4,
    dottedLine: true,
    arrowLine: true
  }]
}

module.exports = {
  navigateToVehicle: navigateToVehicle,
  buildVehicleMarker: buildVehicleMarker,
  buildPolyline: buildPolyline
}
