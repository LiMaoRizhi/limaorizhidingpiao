var log = require('../../utils/log')
const { request } = require('../../utils/request')
const { validateIDCard } = require('../../utils/idcard')

Page({
  data: {
    editingId: 0,
    formName: '',
    formIdCardNo: '',
    focusedField: '',
    isEditing: false
  },

  onLoad(options) {
    if (options.mode === 'edit') {
      const item = wx.getStorageSync('edit_passenger')
      if (item) {
        this.setData({
          editingId: item.id,
          formName: item.name,
          // 回显脱敏身份证号（只读展示，后端识别脱敏值含*不覆盖原明文）
          formIdCardNo: item.id_card_no || '',
          isEditing: true
        })
        wx.removeStorageSync('edit_passenger')
        wx.showToast({ title: '证件号已加密保存，如需修改请删除重新添加', icon: 'none', duration: 2500 })
      }
    }
  },

  onNameInput(e) {
    this.setData({ formName: e.detail.value })
  },

  onIdCardInput(e) {
    this.setData({ formIdCardNo: e.detail.value })
  },

  onFieldFocus(e) {
    this.setData({ focusedField: e.currentTarget.dataset.field })
  },

  onFieldBlur() {
    this.setData({ focusedField: '' })
  },

  async savePassenger() {
    const name = this.data.formName
    const idCardNo = this.data.formIdCardNo
    if (!name) {
      wx.showToast({ title: '请输入姓名', icon: 'none' })
      return
    }
    if (!idCardNo) {
      wx.showToast({ title: '请输入身份证号', icon: 'none' })
      return
    }
    // 编辑时身份证号可能是脱敏值（含*，未修改），跳过格式校验；新增或已修改则校验
    const isMasked = idCardNo.indexOf('*') !== -1
    if (!isMasked) {
      const cardErr = validateIDCard(idCardNo)
      if (cardErr) {
        wx.showToast({ title: cardErr, icon: 'none' })
        return
      }
    }

    const data = {
      name: name
    }
    // 编辑模式下身份证号为脱敏值（含*）时不下发，避免后端误覆盖
    if (!(this.data.isEditing && isMasked)) {
      data.id_card_no = idCardNo
    }

    try {
      if (this.data.editingId) {
        await request({
          url: `/api/wx/passengers/${this.data.editingId}`,
          method: 'PUT',
          data
        })
        wx.showToast({ title: '更新成功', icon: 'success' })
      } else {
        await request({
          url: '/api/wx/passengers',
          method: 'POST',
          data
        })
        wx.showToast({ title: '添加成功', icon: 'success' })
      }
      setTimeout(() => {
        wx.navigateBack()
      }, 1000)
    } catch (e) { log.error('保存乘客失败', e) }
  }
})
