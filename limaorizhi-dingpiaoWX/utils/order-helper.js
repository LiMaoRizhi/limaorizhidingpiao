// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声

/**
 * 安全提取日期字符串的纯日期部分（前10位），兼容 "2026-07-21" 和 "2026-07-21T00:00:00+08:00" 两种格式
 */
function safeDate(str) {
  if (!str) return ''
  return str.length > 10 ? str.substring(0, 10) : str
}

/**
 * 格式化日期为 MM-DD 显示格式（去掉年份）
 */
function formatDate(str) {
  var d = safeDate(str)
  return d.length >= 10 ? d.substring(5) : ''
}

/**
 * 订单状态映射（车票）含状态颜色
 */
var TICKET_STATUS_MAP = {
  0: { key: '0', text: '待支付', color: '#ff3b30', action: '去支付', secondaryAction: '取消订单', secondaryActionType: 'cancel' },
  1: { key: '1', text: '待出行', color: '#34c759', action: '查看详情', secondaryAction: '申请退票', secondaryActionType: 'refund' },
  2: { key: '2', text: '已完成', color: '#8a8a8a', action: '查看详情' },
  3: { key: '3', text: '已退款', color: '#ff3b30', action: '查看详情' },
  4: { key: '4', text: '已取消', color: '#8a8a8a', action: '查看详情' }
}

/**
 * 订单状态映射（托运）含状态颜色
 */
var CARGO_STATUS_MAP = {
  0: { key: '0', text: '待支付', color: '#ff3b30', action: '去支付', secondaryAction: '取消订单', secondaryActionType: 'cancel' },
  1: { key: '1', text: '待运输', color: '#34c759', action: '查看详情', secondaryAction: '申请退款', secondaryActionType: 'refund' },
  2: { key: '2', text: '运输中', color: '#007aff', action: '查看详情' },
  3: { key: '3', text: '已到达', color: '#34c759', action: '查看详情' },
  4: { key: '4', text: '已取消', color: '#8a8a8a', action: '查看详情' },
  5: { key: '5', text: '已取件', color: '#8a8a8a', action: '查看详情' }
}

/**
 * 格式化单个订单为列表展示数据
 */
function formatOrder(item) {
  // 优先使用冗余站名（删除线路/站点后仍可显示），回退到关联站名
  var fromName = item.from_station_name || (item.from_station ? item.from_station.name : '')
  var toName = item.to_station_name || (item.to_station ? item.to_station.name : '')
  var isCargo = item.order_type === 2
  var statusMap = isCargo ? CARGO_STATUS_MAP : TICKET_STATUS_MAP
  var statusInfo = statusMap[item.status] || { key: 'all', text: '未知', action: '查看详情' }
  var dateStr = formatDate(item.trip_date)
  var arrivalText = formatArrivalTime(item.arrival_time, item.arrival_day_offset)
  return {
    id: item.id,
    orderNo: item.order_no,
    typeText: isCargo ? '托运' : '车票',
    isCargo: isCargo,
    status: statusInfo.key,
    statusRaw: item.status,
    statusText: statusInfo.text,
    from: fromName,
    to: toName,
    date: dateStr,
    time: item.departure_time,
    arrivalText: arrivalText,
    price: (parseFloat(item.total_price) || 0).toFixed(0),
    infoText: isCargo ? (item.cargo_type + ' ' + item.weight + 'kg') : ('×' + item.passenger_count + '张'),
    actionText: statusInfo.action,
    secondaryAction: statusInfo.secondaryAction || '',
    secondaryActionType: statusInfo.secondaryActionType || ''
  }
}

/**
 * 批量格式化订单列表
 */
function formatOrderList(list) {
  return (list || []).map(formatOrder)
}

/**
 * 按tab筛选订单
 */
function filterOrdersByTab(orderList, tabKey) {
  if (tabKey === 'all') return orderList
  return orderList.filter(function (item) {
    return item.status === tabKey
  })
}

/**
 * 跨天到达天数标签
 * offset=0 → ''，offset=1 → '次日'，offset=2 → '第2天'，依此类推
 */
function arrivalOffsetLabel(arrivalDayOffset) {
  var offset = arrivalDayOffset || 0
  if (offset === 0) return ''
  if (offset === 1) return '次日'
  var labels = ['', '次日', '第2天', '第3天', '第4天', '第5天', '第6天', '第7天']
  return labels[offset] || ('第' + offset + '天')
}

/**
 * 格式化到达时间（带跨天标签）
 * @param {string} arrivalTime - 到达时间 HH:MM
 * @param {number} arrivalDayOffset - 到达天数偏移(0=当天,1=次日...)
 * @returns {string} 如 "次日08:00" 或 "08:00"
 */
function formatArrivalTime(arrivalTime, arrivalDayOffset) {
  if (!arrivalTime) return ''
  var label = arrivalOffsetLabel(arrivalDayOffset)
  return label ? label + arrivalTime : arrivalTime
}

/**
 * 获取订单状态信息（含文本、颜色、操作按钮文案）
 * @param {number} status - 订单状态码
 * @param {boolean} isCargo - 是否托运订单
 * @returns {object} { key, text, color, action, secondaryAction, secondaryActionType }
 */
function getStatusInfo(status, isCargo) {
  var statusMap = isCargo ? CARGO_STATUS_MAP : TICKET_STATUS_MAP
  return statusMap[status] || { key: 'all', text: '未知', color: '#8a8a8a', action: '查看详情' }
}

module.exports = {
  formatOrderList: formatOrderList,
  filterOrdersByTab: filterOrdersByTab,
  formatArrivalTime: formatArrivalTime,
  getStatusInfo: getStatusInfo
}
