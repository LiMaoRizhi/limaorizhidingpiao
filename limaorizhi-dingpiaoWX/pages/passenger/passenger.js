// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')
const { request } = require('../../utils/request')

Page({
  data: {
    passengers: [],
    selectMode: false,   // 是否勾选模式（从班次详情跳转过来）
    selectedIds: []       // 勾选的乘客 ID 列表
  },

  onLoad(options) {
    if (options.selectMode === '1') {
      this.setData({ selectMode: true })
      // 读取已选乘客 id 列表，用于回显勾选状态
      const preselectIds = wx.getStorageSync('trip_preselect_ids') || []
      this.setData({ selectedIds: preselectIds.map(String) })
    }
  },

  onShow() {
    this.loadPassengers()
  },

  loadPassengers() {
    request({ url: '/api/wx/passengers', method: 'GET' }).then(res => {
      const list = (res.data || []).map(p => ({
        ...p,
        checked: this.data.selectedIds.map(String).indexOf(String(p.id)) !== -1
      }))
      this.setData({ passengers: list })
    }).catch((e) => { log.error('加载乘客列表失败', e) })
  },

  // 跳转到新增页面
  showAddForm() {
    wx.navigateTo({ url: '/pages/passenger-form/passenger-form' })
  },

  // 跳转到编辑页面
  showEditForm(e) {
    if (this.data.selectMode) return // 勾选模式下不触发编辑
    const item = e.currentTarget.dataset.item
    wx.setStorageSync('edit_passenger', item)
    wx.navigateTo({ url: '/pages/passenger-form/passenger-form?mode=edit' })
  },

  // 删除乘客
  deletePassenger(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({
      title: '确认删除',
      content: '确认删除此乘客？',
      success: async (res) => {
        if (res.confirm) {
          try {
            await request({
              url: `/api/wx/passengers/${id}`,
              method: 'DELETE'
            })
            wx.showToast({ title: '删除成功', icon: 'success' })
            this.loadPassengers()
          } catch (e) { log.error('删除乘客失败', e) }
        }
      }
    })
  },

  // 勾选/取消勾选乘客
  toggleSelect(e) {
    const id = String(e.currentTarget.dataset.id)
    const ids = [...this.data.selectedIds]
    const idx = ids.indexOf(id)
    if (idx !== -1) {
      ids.splice(idx, 1)
    } else {
      ids.push(id)
    }
    // 更新每个乘客的 checked 状态
    const passengers = this.data.passengers.map(p => ({
      ...p,
      checked: ids.indexOf(String(p.id)) !== -1
    }))
    this.setData({ selectedIds: ids, passengers })
  },

  // 确认选择，将选中乘客写入 storage 并返回
  confirmSelection() {
    const ids = this.data.selectedIds
    if (!ids.length) {
      wx.showToast({ title: '请至少选择一位乘客', icon: 'none' })
      return
    }
    const selected = this.data.passengers
      .filter(p => p.checked)
      .map(p => ({ id: p.id, name: p.name, id_card_no: p.id_card_no }))
    wx.setStorageSync('trip_selected_passengers', selected)
    // 清除预选 id 缓存，避免下次误用
    wx.removeStorageSync('trip_preselect_ids')
    wx.navigateBack()
  }
})
