// 管理后台 - 司机管理（CRUD，仅超管可写）
const { request } = require('../../utils/request')

const STATUS_TABS = [
  { key: '', text: '全部' },
  { key: '1', text: '启用' },
  { key: '0', text: '禁用' }
]

Page({
  data: {
    adminRole: 0,
    isSuperAdmin: false,
    statusTabs: STATUS_TABS,
    activeStatus: '',
    keyword: '',
    inputFocused: '',
    list: [],
    page: 1,
    pageSize: 20,
    total: 0,
    hasMore: true,
    loading: false,
    loadingMore: false,
    // 表单弹层
    formVisible: false,
    formMode: 'add', // 'add' | 'edit'
    editingId: null,
    submitting: false,
    form: {
      id: 0, name: '', phone: '', password: '',
      license_no: '', employee_no: '', status: 1
    }
  },

  onLoad() {
    const userInfo = wx.getStorageSync('user_info') || {}
    const role = userInfo.admin_role || 0
    this.setData({ adminRole: role, isSuperAdmin: role === 1 })
    if (role === 1) {
      this.loadList()
    }
  },

  // 输入框聚焦/失焦（黑框点击变蓝）
  onFieldFocus(e) {
    this.setData({ inputFocused: e.currentTarget.dataset.field || '' })
  },
  onFieldBlur() {
    this.setData({ inputFocused: '' })
  },

  loadList(append) {
    if (append) {
      if (this.data.loadingMore || !this.data.hasMore || this.data.loading) return
    } else {
      if (this.data.loading) return
    }
    const page = append ? this.data.page + 1 : 1
    if (append) this.setData({ loadingMore: true })
    else this.setData({ loading: true })

    const data = { page, page_size: this.data.pageSize }
    if (this.data.activeStatus) data.status = this.data.activeStatus
    if (this.data.keyword.trim()) data.keyword = this.data.keyword.trim()

    request({ url: '/api/wx/admin/drivers', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(d => ({
        id: d.id,
        name: d.name || '-',
        phone: d.phone || '-',
        employee_no: d.employee_no || '',
        license_no: d.license_no || '',
        status: d.status,
        statusText: d.status === 1 ? '启用' : '禁用',
        statusCls: d.status === 1 ? 'ok' : 'ban',
        last_login: d.last_login_at ? (d.last_login_at + '').replace('T', ' ').slice(0, 16) : '未登录'
      }))
      const list = append ? this.data.list.concat(items) : items
      this.setData({
        list, page, total,
        hasMore: list.length < total,
        loading: false, loadingMore: false
      })
    }).catch(() => {
      this.setData({ loading: false, loadingMore: false, list: append ? this.data.list : [] })
    })
  },

  onStatusTab(e) {
    const key = e.currentTarget.dataset.key
    if (key === this.data.activeStatus) return
    this.setData({ activeStatus: key })
    this.loadList(false)
  },
  onKeywordInput(e) { this.setData({ keyword: e.detail.value }) },
  onSearchConfirm() { this.loadList(false) },
  onClearKeyword() { this.setData({ keyword: '' }); this.loadList(false) },
  onPullDownRefresh() { this.loadList(false); wx.stopPullDownRefresh() },
  onLoadMore() { this.loadList(true) },

  onCallDriver(e) {
    const phone = e.currentTarget.dataset.phone
    if (!phone || phone === '-') return
    wx.makePhoneCall({ phoneNumber: phone })
  },

  // 新增 / 编辑 / 删除 / 启用禁用
  preventBubble() {},

  openAdd() {
    if (!this.data.isSuperAdmin) return
    this.setData({
      formVisible: true, formMode: 'add', editingId: null,
      form: { id: 0, name: '', phone: '', password: '', license_no: '', employee_no: '', status: 1 }
    })
  },

  openEdit(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(d => d.id === id)
    if (!item) return
    this.setData({
      formVisible: true, formMode: 'edit', editingId: id,
      form: {
        id: id,
        name: item.name === '-' ? '' : item.name,
        phone: item.phone === '-' ? '' : item.phone,
        password: '',
        license_no: item.license_no || '',
        employee_no: item.employee_no || '',
        status: item.status === 1 ? 1 : 0
      }
    })
  },

  closeForm() {
    if (this.data.submitting) return
    this.setData({ formVisible: false })
  },

  onFormInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ ['form.' + field]: e.detail.value })
  },

  onStatusChange(e) {
    this.setData({ 'form.status': e.detail.value ? 1 : 0 })
  },

  // 校验密码：≥8位且同时含字母与数字
  _checkPassword(pwd) {
    if (pwd.length < 8) return '密码至少8位'
    if (!/[a-zA-Z]/.test(pwd) || !/\d/.test(pwd)) return '密码需同时包含字母和数字'
    return ''
  },

  onSave() {
    if (this.data.submitting) return
    const f = this.data.form
    const name = (f.name || '').trim()
    if (!name) { wx.showToast({ title: '请输入姓名', icon: 'none' }); return }
    const phone = (f.phone || '').trim()
    if (!phone) { wx.showToast({ title: '请输入手机号', icon: 'none' }); return }
    if (!/^1[3-9]\d{9}$/.test(phone)) { wx.showToast({ title: '手机号格式不正确', icon: 'none' }); return }

    const isEdit = this.data.formMode === 'edit'
    const password = (f.password || '').trim()
    const payload = {
      name: name,
      phone: phone,
      license_no: (f.license_no || '').trim(),
      employee_no: (f.employee_no || '').trim()
    }
    if (isEdit) {
      payload.status = f.status === 1 ? 1 : 0
      if (password) {
        const err = this._checkPassword(password)
        if (err) { wx.showToast({ title: err, icon: 'none' }); return }
        payload.password = password
      }
    } else {
      if (!password) { wx.showToast({ title: '请输入密码', icon: 'none' }); return }
      const err = this._checkPassword(password)
      if (err) { wx.showToast({ title: err, icon: 'none' }); return }
      payload.password = password
    }

    const url = isEdit ? ('/api/wx/admin/drivers/' + this.data.editingId) : '/api/wx/admin/drivers'
    const method = isEdit ? 'PUT' : 'POST'
    this.setData({ submitting: true })
    wx.showLoading({ title: '保存中...', mask: true })
    request({ url, method, data: payload }).then(() => {
      wx.hideLoading()
      wx.showToast({ title: isEdit ? '已保存' : '已新增', icon: 'success' })
      this.setData({ submitting: false, formVisible: false })
      this.loadList(false)
    }).catch(() => {
      wx.hideLoading()
      this.setData({ submitting: false })
    })
  },

  // 快速启用/禁用（PUT 全字段 + 翻转 status）
  onToggleStatus(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(d => d.id === id)
    if (!item) return
    const newStatus = item.status === 1 ? 0 : 1
    const action = newStatus === 1 ? '启用' : '禁用'
    wx.showModal({
      title: action + '司机',
      content: '确定' + action + '司机「' + item.name + '」吗？',
      success: r => {
        if (!r.confirm) return
        wx.showLoading({ title: '处理中...', mask: true })
        request({
          url: '/api/wx/admin/drivers/' + id, method: 'PUT',
          data: {
            name: item.name === '-' ? '' : item.name,
            phone: item.phone === '-' ? '' : item.phone,
            license_no: item.license_no || '',
            employee_no: item.employee_no || '',
            status: newStatus
          }
        }).then(() => {
          wx.hideLoading()
          wx.showToast({ title: '已' + action, icon: 'success' })
          this.loadList(false)
        }).catch(() => { wx.hideLoading() })
      }
    })
  },

  onDelete(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.list.find(d => d.id === id)
    if (!item) return
    wx.showModal({
      title: '删除确认',
      content: '确定删除司机「' + item.name + '」吗？若该司机有未发车的活跃班次，需先取消班次司机分配。',
      confirmText: '删除', confirmColor: '#f5222d',
      success: r => {
        if (!r.confirm) return
        wx.showLoading({ title: '删除中...', mask: true })
        request({ url: '/api/wx/admin/drivers/' + id, method: 'DELETE' }).then(() => {
          wx.hideLoading()
          wx.showToast({ title: '删除成功', icon: 'success' })
          this.loadList(false)
        }).catch(() => { wx.hideLoading() })
      }
    })
  }
})
