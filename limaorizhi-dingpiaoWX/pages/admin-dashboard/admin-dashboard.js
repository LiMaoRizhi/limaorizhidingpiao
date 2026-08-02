// 管理后台 - 统计看板
const { request } = require('../../utils/request')

Page({
  data: {
    loading: true,
    stats: null,
    trend: [],          // 近14天订单趋势（柱状）
    trendMax: 1,
    routeRank: [],
    recentOrders: []
  },

  onLoad() {
    this.loadDashboard()
  },

  loadDashboard() {
    this.setData({ loading: true })
    request({ url: '/api/wx/admin/dashboard', method: 'GET' }).then(res => {
      const d = res.data || {}
      // 近14天趋势（移动端宽度有限，取最近14天）
      const trendAll = d.trend || []
      const trend = trendAll.slice(-14).map(t => ({
        date: (t.date || '').slice(5), // MM-DD
        orders: t.orders || 0,
        revenue: t.revenue || 0
      }))
      const trendMax = Math.max(1, ...trend.map(t => t.orders))
      // 热门线路
      const routeRank = (d.route_rank || []).map(r => ({
        name: r.route_name || '未知线路',
        count: r.order_count || 0,
        revenue: (r.revenue || 0).toFixed(0)
      }))
      // 最近订单
      const recentOrders = (d.recent_orders || []).map(o => ({
        order_no: o.order_no,
        type: o.order_type === 2 ? '托运' : '车票',
        from: o.from_station_name || (o.from_station && o.from_station.name) || '-',
        to: o.to_station_name || (o.to_station && o.to_station.name) || '-',
        price: (o.total_price || 0).toFixed(2),
        status: this.statusText(o.status),
        statusCls: this.statusCls(o.status)
      }))
      this.setData({
        loading: false,
        stats: {
          today_orders: d.today_orders || 0,
          today_revenue: (d.today_revenue || 0).toFixed(0),
          today_users: d.today_users || 0,
          seat_occupancy: (d.seat_occupancy || 0).toFixed(1),
          week_orders: d.week_orders || 0,
          week_revenue: (d.week_revenue || 0).toFixed(0),
          active_trips: d.active_trips || 0,
          refund_rate: (d.refund_rate || 0).toFixed(1),
          total_users: d.total_users || 0,
          total_orders: d.total_orders || 0,
          total_revenue: (d.total_revenue || 0).toFixed(0)
        },
        trend,
        trendMax,
        routeRank,
        recentOrders
      })
    }).catch(() => { this.setData({ loading: false }) })
  },

  statusText(s) {
    return { 0: '待支付', 1: '待出行', 2: '已完成', 3: '已退款', 4: '已取消', 5: '已取件' }[s] || '未知'
  },
  statusCls(s) {
    return { 0: 'pay', 1: 'travel', 2: 'done', 3: 'refund', 4: 'cancel', 5: 'done' }[s] || ''
  },

  onPullDownRefresh() {
    this.loadDashboard()
    wx.stopPullDownRefresh()
  }
})
