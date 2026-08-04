// 选座页面 - 大巴座位图（单网格精确定位）
const { request } = require('../../utils/request')
const log = require('../../utils/log')

Page({
  data: {
    tripId: 0,
    fromStationId: 0,
    toStationId: 0,
    passengerCount: 0,

    // 座位图数据
    rows: 0,
    cols: 0,
    cells: [],           // 所有格子 [{row, col, type, seatNo, colSpan}]
    seatStatus: {},      // {seatNo: 'free'|'occupied'|'selected'}
    selectedSeats: [],   // 已选座位号
    colWidths: '',       // grid-template-columns 字符串

    // UI
    loading: true,
    seatMapLoaded: false,
    totalSeats: 0,
    availSeats: 0,
    // TODO: 座位号超过2位数的车（50座以上）格子会挤，后面改成根据位数动态算 cellSize
    cellSize: 72,        // 座位格 rpx
    aisleWidth: 40,      // 过道宽 rpx

    // 无座站票（选座页内可选无座）
    standingAllowed: false,
    standingAvailable: 0,
    standingPriceText: '',
    buyStanding: false
  },

  onLoad(options) {
    if (!options.trip_id || !options.from_sid || !options.to_sid || !options.count) {
      wx.showToast({ title: '参数错误', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 1000)
      return
    }
    const count = parseInt(options.count)
    if (isNaN(count) || count <= 0) {
      wx.showToast({ title: '乘客数量无效', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 1000)
      return
    }
    this.setData({
      tripId: parseInt(options.trip_id),
      fromStationId: parseInt(options.from_sid),
      toStationId: parseInt(options.to_sid),
      passengerCount: Math.min(count, 10),
      standingAllowed: options.standing_allowed === '1',
      standingAvailable: parseInt(options.standing_available) || 0,
      standingPriceText: decodeURIComponent(options.standing_price || '')
    })
    this.loadSeatMap()
  },

  // 选座页内切换无座站票（黑白）
  toggleStanding() {
    if (!this.data.standingAllowed) return
    const newVal = !this.data.buyStanding
    if (newVal && this.data.standingAvailable === 0) {
      wx.showToast({ title: '无座票已售罄', icon: 'none' })
      return
    }
    if (newVal && this.data.passengerCount > this.data.standingAvailable) {
      wx.showToast({ title: `无座票仅剩${this.data.standingAvailable}张，请减少乘客数`, icon: 'none' })
      return
    }
    // 切到无座：清空已选座位（座位图本身由 wxml 的 !buyStanding 控制隐藏）
    this.setData({
      buyStanding: newVal,
      selectedSeats: newVal ? [] : this.data.selectedSeats
    })
  },

  async loadSeatMap() {
    this.setData({ loading: true, seatMapLoaded: false })
    try {
      const res = await request({
        url: `/api/wx/trips/${this.data.tripId}/seats?from_station_id=${this.data.fromStationId}&to_station_id=${this.data.toStationId}`,
        method: 'GET'
      })
      const data = res.data
      const layout = data.layout || {}
      const seats = data.seats || []

      let passengerCount = this.data.passengerCount
      if (data.avail !== undefined && passengerCount > data.avail) {
        passengerCount = Math.max(1, data.avail)
        wx.showToast({ title: `该区间仅剩${data.avail}座，已调整乘客数`, icon: 'none', duration: 2000 })
      }

      // 整座位状态表
      const seatStatus = {}
      const occupiedSet = new Set()
      seats.forEach(s => {
        if (s.occupied) occupiedSet.add(s.seat_no)
        seatStatus[s.seat_no] = s.occupied ? 'occupied' : 'free'
      })

      // 整理所有格子：过道/空白格也要保留（占位），并生成网格列宽
      const totalCols = layout.cols || 0
      const totalRows = layout.rows || 0
      const cells = (layout.cells || []).map(cell => {
        const key = cell.seat_no
        if (cell.type === 'seat' && seatStatus[key] === undefined) {
          seatStatus[key] = 'free'
        }
        return {
          row: cell.row,
          col: cell.col,
          type: cell.type,
          seatNo: cell.seat_no,
          _key: `${cell.row}_${cell.col}`
        }
      })

      cells.sort((a, b) => a.row !== b.row ? a.row - b.row : a.col - b.col)

      // 拼列宽字符串：所有列均分（过道格靠CSS自己缩）
      const colWidths = `repeat(${totalCols}, 1fr)`

      this.setData({
        rows: totalRows,
        cols: totalCols,
        cells: cells,
        colWidths: colWidths,
        seatStatus: seatStatus,
        selectedSeats: [],
        totalSeats: data.seats ? data.seats.length : 0,
        availSeats: data.avail || 0,
        loading: false,
        seatMapLoaded: true,
        passengerCount: passengerCount
      })
    } catch (e) {
      log.error('加载座位图失败', e)
      wx.showToast({ title: '加载座位图失败', icon: 'none' })
      this.setData({ loading: false })
    }
  },

  onSeatTap(e) {
    const { seatNo } = e.currentTarget.dataset
    if (!seatNo) return

    const status = this.data.seatStatus[seatNo]
    if (status === 'occupied') {
      wx.showToast({ title: '该座位已被占用', icon: 'none' })
      return
    }

    const selected = [...this.data.selectedSeats]
    const seatStatus = { ...this.data.seatStatus }

    if (status === 'selected') {
      const idx = selected.indexOf(seatNo)
      if (idx >= 0) selected.splice(idx, 1)
      seatStatus[seatNo] = 'free'
    } else {
      if (selected.length >= this.data.passengerCount) {
        wx.showToast({ title: `最多选${this.data.passengerCount}个座位`, icon: 'none' })
        return
      }
      selected.push(seatNo)
      seatStatus[seatNo] = 'selected'
    }

    this.setData({ selectedSeats: selected, seatStatus: seatStatus })
  },

  confirmSelection() {
    // 无座模式：无需选座，直接确认
    if (this.data.buyStanding) {
      this.goBackStanding()
      return
    }
    const { selectedSeats, passengerCount } = this.data
    if (selectedSeats.length === 0) {
      wx.showModal({
        title: '未选座位',
        content: '您还未选择座位，将自动分配。确定继续？',
        success: (res) => { if (res.confirm) this.goBack([]) }
      })
      return
    }
    if (selectedSeats.length < passengerCount) {
      wx.showModal({
        title: '座位不足',
        content: `您选了${selectedSeats.length}个座，但需要${passengerCount}个。未选座位将自动分配。`,
        success: (res) => { if (res.confirm) this.goBack(selectedSeats) }
      })
      return
    }
    this.goBack(selectedSeats)
  },

  // 无座模式返回：写入 'STANDING' 标记，下单页据此切换无座模式
  goBackStanding() {
    wx.setStorageSync('trip_selected_seats', 'STANDING')
    wx.navigateBack()
  },

  goBack(seatNos) {
    wx.setStorageSync('trip_selected_seats', seatNos)
    wx.navigateBack()
  },

  skipSelection() {
    if (this.data.buyStanding) {
      this.goBackStanding()
      return
    }
    wx.setStorageSync('trip_selected_seats', [])
    wx.navigateBack()
  }
})
