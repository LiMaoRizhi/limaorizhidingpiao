// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818
// 站点选择弹窗公共逻辑（home / cargo-create 共用）
// 页面需自行实现 onStationTap（选中站点后的业务回调）
var { matchStation } = require('./pinyin')

module.exports = {
  data: {
    showStationPicker: false,
    stationPickerType: '',
    stationSearchValue: '',
    stationSearchFocused: false
  },

  // 出发站选择
  showFromPicker() {
    this.setData({ showStationPicker: true, stationPickerType: 'from', stationSearchValue: '', filteredStations: this.data.stations })
  },

  // 到达站选择
  showToPicker() {
    this.setData({ showStationPicker: true, stationPickerType: 'to', stationSearchValue: '', filteredStations: this.data.stations })
  },

  closeStationPicker() {
    wx.hideKeyboard()
    this.setData({ showStationPicker: false, stationSearchValue: '', filteredStations: this.data.stations, stationSearchFocused: false })
  },

  // 弹窗遮罩点击：仅点击遮罩空白处才关闭
  onStationMaskTap(e) {
    if (e.target.id === 'stationMask') this.closeStationPicker()
  },

  // 空方法，供 catchtap 阻止冒泡使用
  noop() {},

  // 站点弹窗搜索输入（中文 / 全拼 / 首字母联想）
  onStationSearch(e) {
    var value = e.detail.value
    var filtered = this.data.stations.filter(function (s) { return matchStation(s, value) })
    this.setData({ stationSearchValue: value, filteredStations: filtered })
  },

  // 站点搜索确认：收起键盘
  onStationSearchConfirm() {
    wx.hideKeyboard()
  },

  onStationSearchFocus() {
    this.setData({ stationSearchFocused: true })
  },

  onStationSearchBlur() {
    this.setData({ stationSearchFocused: false })
  },

  // 清空站点搜索
  clearStationSearch() {
    this.setData({ stationSearchValue: '', filteredStations: this.data.stations })
  }
}
