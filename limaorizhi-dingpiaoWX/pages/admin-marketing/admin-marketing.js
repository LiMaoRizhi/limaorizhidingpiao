// 管理后台 - 营销管理（优惠券/发放记录/积分规则/用户积分）
// 超管可新增/编辑/删除优惠券和积分规则
const { request } = require('../../utils/request')

const TABS = [
  { key: 'coupons', text: '优惠券' },
  { key: 'records', text: '发放记录' },
  { key: 'rules', text: '积分规则' },
  { key: 'points', text: '用户积分' }
]

// 优惠券类型：1=满减 2=折扣 3=固定金额
const COUPON_TYPE_TEXT = { 1: '满减券', 2: '折扣券', 3: '抵扣券' }
const COUPON_TYPE_OPTIONS = ['满减券', '折扣券', '抵扣券']
const COUPON_TYPE_VALUES = [1, 2, 3]
// 发放记录状态：0=未使用 1=已使用 2=已过期
const RECORD_STATUS_TEXT = { 0: '未使用', 1: '已使用', 2: '已过期' }
const RECORD_STATUS_CLS = { 0: 'ok', 1: 'used', 2: 'cancel' }
// 积分规则类型：1=消费赠送 2=注册赠送 3=手动调整
const RULE_TYPE_TEXT = { 1: '消费赠送', 2: '注册赠送', 3: '手动调整' }
const RULE_TYPE_OPTIONS = ['消费赠送', '注册赠送', '手动调整']
const RULE_TYPE_VALUES = [1, 2, 3]

const DEFAULT_COUPON_FORM = {
  name: '', type: 1, typeIndex: 0, discount_value: '',
  min_spend: '', valid_days: 30, total_count: 0, status: 1
}
const DEFAULT_RULE_FORM = {
  rule_name: '', rule_type: 1, rule_typeIndex: 0,
  points_per_yuan: '', fixed_points: '', description: '', status: 1
}

Page({
  data: {
    tabs: TABS,
    activeTab: 'coupons',
    keyword: '',
    inputFocused: '',
    isSuperAdmin: false,

    // 优惠券
    couponList: [],
    couponPage: 1,
    couponTotal: 0,
    couponHasMore: true,

    // 发放记录
    recordList: [],
    recordPage: 1,
    recordTotal: 0,
    recordHasMore: true,

    // 积分规则（不分页）
    ruleList: [],

    // 用户积分
    pointsList: [],
    pointsPage: 1,
    pointsTotal: 0,
    pointsHasMore: true,

    pageSize: 20,
    loading: false,
    loadingMore: false,

    // 表单弹层
    couponTypeOptions: COUPON_TYPE_OPTIONS,
    ruleTypeOptions: RULE_TYPE_OPTIONS,
    formVisible: false,
    editingType: '', // 'coupon' | 'rule'
    editingId: null, // null=新增
    submitting: false,
    couponForm: Object.assign({}, DEFAULT_COUPON_FORM),
    ruleForm: Object.assign({}, DEFAULT_RULE_FORM)
  },

  onLoad() {
    const userInfo = wx.getStorageSync('user_info') || {}
    this.setData({ isSuperAdmin: userInfo.admin_role === 1 })
    this.loadCoupons(false)
  },

  // 输入框聚焦/失焦（黑线框点击变蓝）
  onFieldFocus(e) {
    this.setData({ inputFocused: e.currentTarget.dataset.field || '' })
  },
  onFieldBlur() {
    this.setData({ inputFocused: '' })
  },

  onTabTap(e) {
    const key = e.currentTarget.dataset.key
    if (key === this.data.activeTab) return
    this.setData({ activeTab: key, keyword: '' })
    if (key === 'coupons' && this.data.couponList.length === 0) this.loadCoupons(false)
    else if (key === 'records' && this.data.recordList.length === 0) this.loadRecords(false)
    else if (key === 'rules' && this.data.ruleList.length === 0) this.loadRules()
    else if (key === 'points' && this.data.pointsList.length === 0) this.loadPoints(false)
  },

  onKeywordInput(e) { this.setData({ keyword: e.detail.value }) },
  onSearchConfirm() {
    const t = this.data.activeTab
    if (t === 'coupons') this.loadCoupons(false)
    else if (t === 'records') this.loadRecords(false)
    else if (t === 'points') this.loadPoints(false)
  },
  onClearKeyword() {
    this.setData({ keyword: '' })
    this.onSearchConfirm()
  },

  // 优惠券
  loadCoupons(append) {
    if (this.data.loading) return
    const page = append ? this.data.couponPage + 1 : 1
    if (append && (this.data.loadingMore || !this.data.couponHasMore)) return
    this.setData({ loading: !append, loadingMore: append })
    const data = { page, page_size: this.data.pageSize }
    if (this.data.keyword.trim()) data.name = this.data.keyword.trim()
    request({ url: '/api/wx/admin/coupons', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(c => this.formatCoupon(c))
      const list = append ? this.data.couponList.concat(items) : items
      this.setData({
        couponList: list, couponPage: page, couponTotal: total,
        couponHasMore: list.length < total, loading: false, loadingMore: false
      })
    }).catch(() => { this.setData({ loading: false, loadingMore: false, couponList: append ? this.data.couponList : [] }) })
  },

  formatCoupon(c) {
    let valueText = ''
    if (c.type === 1) valueText = '满' + (c.min_spend || 0).toFixed(0) + '减' + c.discount_value.toFixed(0)
    else if (c.type === 2) valueText = (c.discount_value) + '折'
    else valueText = '抵' + c.discount_value.toFixed(0) + '元'
    return {
      id: c.id,
      name: c.name || '-',
      type_text: COUPON_TYPE_TEXT[c.type] || '其他',
      value_text: valueText,
      min_spend: (c.min_spend || 0).toFixed(2),
      valid_days: c.valid_days || 0,
      total_count: c.total_count || 0,
      issued_count: c.issued_count || 0,
      used_count: c.used_count || 0,
      status: c.status,
      statusText: c.status === 1 ? '启用' : '停用',
      statusCls: c.status === 1 ? 'ok' : 'ban',
      // 原始字段，编辑时回填
      raw_type: c.type,
      raw_discount_value: c.discount_value,
      raw_min_spend: c.min_spend || 0,
      raw_valid_days: c.valid_days || 30,
      raw_total_count: c.total_count || 0
    }
  },

  // 发放记录
  loadRecords(append) {
    if (this.data.loading) return
    const page = append ? this.data.recordPage + 1 : 1
    if (append && (this.data.loadingMore || !this.data.recordHasMore)) return
    this.setData({ loading: !append, loadingMore: append })
    const data = { page, page_size: this.data.pageSize }
    const kw = this.data.keyword.trim()
    if (kw && /^\d+$/.test(kw)) data.user_id = kw
    request({ url: '/api/wx/admin/user-coupons', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(uc => {
        const coupon = uc.coupon || {}
        const user = uc.user || {}
        return {
          id: uc.id,
          coupon_name: coupon.name || '-',
          coupon_type: COUPON_TYPE_TEXT[coupon.type] || '其他',
          user_nickname: user.nickname || ('用户#' + uc.user_id),
          user_phone: user.phone || '-',
          status: uc.status,
          statusText: RECORD_STATUS_TEXT[uc.status] || '未知',
          statusCls: RECORD_STATUS_CLS[uc.status] || '',
          issued_at: (uc.issued_at || uc.created_at || '').toString().replace('T', ' ').slice(0, 16),
          expired_at: (uc.expired_at || '').toString().replace('T', ' ').slice(0, 16)
        }
      })
      const list = append ? this.data.recordList.concat(items) : items
      this.setData({
        recordList: list, recordPage: page, recordTotal: total,
        recordHasMore: list.length < total, loading: false, loadingMore: false
      })
    }).catch(() => { this.setData({ loading: false, loadingMore: false, recordList: append ? this.data.recordList : [] }) })
  },

  // 积分规则（不分页）
  loadRules() {
    this.setData({ loading: true })
    request({ url: '/api/wx/admin/point-rules', method: 'GET', silent: true }).then(res => {
      const raw = res.data || []
      const items = raw.map(r => ({
        id: r.id,
        rule_name: r.rule_name || '-',
        rule_type_text: RULE_TYPE_TEXT[r.rule_type] || '其他',
        points_per_yuan: (r.points_per_yuan || 0).toFixed(2),
        fixed_points: r.fixed_points || 0,
        description: r.description || '',
        status: r.status,
        statusText: r.status === 1 ? '启用' : '停用',
        statusCls: r.status === 1 ? 'ok' : 'ban',
        // 原始字段，编辑时回填
        raw_rule_type: r.rule_type,
        raw_points_per_yuan: r.points_per_yuan || 0,
        raw_fixed_points: r.fixed_points || 0
      }))
      this.setData({ ruleList: items, loading: false })
    }).catch(() => { this.setData({ loading: false, ruleList: [] }) })
  },

  // 用户积分
  loadPoints(append) {
    if (this.data.loading) return
    const page = append ? this.data.pointsPage + 1 : 1
    if (append && (this.data.loadingMore || !this.data.pointsHasMore)) return
    this.setData({ loading: !append, loadingMore: append })
    const data = { page, page_size: this.data.pageSize }
    if (this.data.keyword.trim()) data.phone = this.data.keyword.trim()
    request({ url: '/api/wx/admin/user-points', method: 'GET', data, silent: true }).then(res => {
      const raw = (res.data && res.data.list) || []
      const total = (res.data && res.data.total) || 0
      const items = raw.map(up => {
        const user = up.user || {}
        return {
          id: up.id,
          user_id: up.user_id,
          nickname: user.nickname || ('用户#' + up.user_id),
          phone: user.phone || '-',
          balance: up.balance || 0,
          total_earned: up.total_earned || 0,
          total_spent: up.total_spent || 0
        }
      })
      const list = append ? this.data.pointsList.concat(items) : items
      this.setData({
        pointsList: list, pointsPage: page, pointsTotal: total,
        pointsHasMore: list.length < total, loading: false, loadingMore: false
      })
    }).catch(() => { this.setData({ loading: false, loadingMore: false, pointsList: append ? this.data.pointsList : [] }) })
  },

  onPullDownRefresh() {
    const t = this.data.activeTab
    if (t === 'coupons') this.loadCoupons(false)
    else if (t === 'records') this.loadRecords(false)
    else if (t === 'rules') this.loadRules()
    else if (t === 'points') this.loadPoints(false)
    wx.stopPullDownRefresh()
  },
  onLoadMore() {
    const t = this.data.activeTab
    if (t === 'coupons') this.loadCoupons(true)
    else if (t === 'records') this.loadRecords(true)
    else if (t === 'points') this.loadPoints(true)
  },

  // 新增/编辑/删除
  onAdd() {
    if (!this.data.isSuperAdmin) return
    if (this.data.activeTab === 'coupons') this.openCouponForm(null)
    else if (this.data.activeTab === 'rules') this.openRuleForm(null)
  },

  openCouponForm(id) {
    if (!this.data.isSuperAdmin) return
    const item = id ? this.data.couponList.find(c => c.id === id) : null
    if (id && !item) return
    let typeIndex = item ? COUPON_TYPE_VALUES.indexOf(item.raw_type) : 0
    if (typeIndex < 0) typeIndex = 0
    const form = item ? {
      name: item.name === '-' ? '' : item.name,
      type: item.raw_type,
      typeIndex: typeIndex,
      discount_value: String(item.raw_discount_value),
      min_spend: String(item.raw_min_spend),
      valid_days: item.raw_valid_days,
      total_count: item.raw_total_count,
      status: item.status === 1 ? 1 : 0
    } : Object.assign({}, DEFAULT_COUPON_FORM)
    this.setData({
      editingType: 'coupon',
      editingId: id || null,
      couponForm: form,
      formVisible: true
    })
  },

  openRuleForm(id) {
    if (!this.data.isSuperAdmin) return
    const item = id ? this.data.ruleList.find(r => r.id === id) : null
    if (id && !item) return
    let typeIndex = item ? RULE_TYPE_VALUES.indexOf(item.raw_rule_type) : 0
    if (typeIndex < 0) typeIndex = 0
    const form = item ? {
      rule_name: item.rule_name === '-' ? '' : item.rule_name,
      rule_type: item.raw_rule_type,
      rule_typeIndex: typeIndex,
      points_per_yuan: String(item.raw_points_per_yuan),
      fixed_points: String(item.raw_fixed_points),
      description: item.description || '',
      status: item.status === 1 ? 1 : 0
    } : Object.assign({}, DEFAULT_RULE_FORM)
    this.setData({
      editingType: 'rule',
      editingId: id || null,
      ruleForm: form,
      formVisible: true
    })
  },

  onEditCoupon(e) { this.openCouponForm(e.currentTarget.dataset.id) },
  onEditRule(e) { this.openRuleForm(e.currentTarget.dataset.id) },

  onDeleteCoupon(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.couponList.find(c => c.id === id)
    if (!item) return
    wx.showModal({
      title: '删除优惠券',
      content: '确定删除「' + item.name + '」吗？删除后不可恢复。',
      confirmText: '删除',
      confirmColor: '#fa5151',
      success: r => {
        if (!r.confirm) return
        wx.showLoading({ title: '删除中', mask: true })
        request({ url: '/api/wx/admin/coupons/' + id, method: 'DELETE' }).then(() => {
          wx.hideLoading()
          wx.showToast({ title: '已删除', icon: 'success' })
          this.loadCoupons(false)
        }).catch(() => {
          wx.hideLoading()
        })
      }
    })
  },

  onDeleteRule(e) {
    if (!this.data.isSuperAdmin) return
    const id = e.currentTarget.dataset.id
    const item = this.data.ruleList.find(r => r.id === id)
    if (!item) return
    wx.showModal({
      title: '删除积分规则',
      content: '确定删除「' + item.rule_name + '」吗？',
      confirmText: '删除',
      confirmColor: '#fa5151',
      success: r => {
        if (!r.confirm) return
        wx.showLoading({ title: '删除中', mask: true })
        request({ url: '/api/wx/admin/point-rules/' + id, method: 'DELETE' }).then(() => {
          wx.hideLoading()
          wx.showToast({ title: '已删除', icon: 'success' })
          this.loadRules()
        }).catch(() => {
          wx.hideLoading()
        })
      }
    })
  },

  onCloseForm() {
    if (this.data.submitting) return
    this.setData({ formVisible: false })
  },
  // 阻止弹层内容点击冒泡到遮罩
  preventBubble() {},

  // 优惠券表单输入
  onCouponNameInput(e) { this.setData({ 'couponForm.name': e.detail.value }) },
  onCouponTypeChange(e) {
    const idx = Number(e.detail.value)
    this.setData({ 'couponForm.typeIndex': idx, 'couponForm.type': COUPON_TYPE_VALUES[idx] })
  },
  onCouponDiscountInput(e) { this.setData({ 'couponForm.discount_value': e.detail.value }) },
  onCouponMinSpendInput(e) { this.setData({ 'couponForm.min_spend': e.detail.value }) },
  onCouponValidDaysInput(e) { this.setData({ 'couponForm.valid_days': e.detail.value }) },
  onCouponTotalCountInput(e) { this.setData({ 'couponForm.total_count': e.detail.value }) },
  onCouponStatusChange(e) { this.setData({ 'couponForm.status': e.detail.value ? 1 : 0 }) },

  // 积分规则表单输入
  onRuleNameInput(e) { this.setData({ 'ruleForm.rule_name': e.detail.value }) },
  onRuleTypeChange(e) {
    const idx = Number(e.detail.value)
    this.setData({ 'ruleForm.rule_typeIndex': idx, 'ruleForm.rule_type': RULE_TYPE_VALUES[idx] })
  },
  onRulePointsPerYuanInput(e) { this.setData({ 'ruleForm.points_per_yuan': e.detail.value }) },
  onRuleFixedPointsInput(e) { this.setData({ 'ruleForm.fixed_points': e.detail.value }) },
  onRuleDescInput(e) { this.setData({ 'ruleForm.description': e.detail.value }) },
  onRuleStatusChange(e) { this.setData({ 'ruleForm.status': e.detail.value ? 1 : 0 }) },

  onSubmitForm() {
    if (this.data.submitting) return
    if (this.data.editingType === 'coupon') this.submitCoupon()
    else if (this.data.editingType === 'rule') this.submitRule()
  },

  submitCoupon() {
    const f = this.data.couponForm
    const name = (f.name || '').trim()
    if (!name) {
      wx.showToast({ title: '请输入优惠券名称', icon: 'none' })
      return
    }
    const dv = parseFloat(f.discount_value)
    if (isNaN(dv) || dv <= 0) {
      wx.showToast({ title: '请输入有效的面值', icon: 'none' })
      return
    }
    if (f.type === 2 && dv >= 10) {
      wx.showToast({ title: '折扣值需在0~10之间', icon: 'none' })
      return
    }
    const payload = {
      name: name,
      type: f.type,
      discount_value: dv,
      min_spend: parseFloat(f.min_spend) || 0,
      valid_days: parseInt(f.valid_days, 10) || 30,
      total_count: parseInt(f.total_count, 10) || 0,
      status: f.status === 1 ? 1 : 0
    }
    const isEdit = !!this.data.editingId
    const url = isEdit ? '/api/wx/admin/coupons/' + this.data.editingId : '/api/wx/admin/coupons'
    const method = isEdit ? 'PUT' : 'POST'
    this.setData({ submitting: true })
    wx.showLoading({ title: '保存中', mask: true })
    request({ url, method, data: payload }).then(() => {
      wx.hideLoading()
      wx.showToast({ title: '保存成功', icon: 'success' })
      this.setData({ submitting: false, formVisible: false })
      this.loadCoupons(false)
    }).catch(() => {
      wx.hideLoading()
      this.setData({ submitting: false })
    })
  },

  submitRule() {
    const f = this.data.ruleForm
    const ruleName = (f.rule_name || '').trim()
    if (!ruleName) {
      wx.showToast({ title: '请输入规则名称', icon: 'none' })
      return
    }
    let ppy = 0
    let fp = 0
    if (f.rule_type === 1) {
      ppy = parseFloat(f.points_per_yuan)
      if (isNaN(ppy) || ppy <= 0) {
        wx.showToast({ title: '请输入有效的每元积分', icon: 'none' })
        return
      }
    } else {
      fp = parseInt(f.fixed_points, 10)
      if (isNaN(fp) || fp < 0) {
        wx.showToast({ title: '请输入有效的固定积分', icon: 'none' })
        return
      }
    }
    const payload = {
      rule_name: ruleName,
      rule_type: f.rule_type,
      points_per_yuan: ppy,
      fixed_points: fp,
      description: (f.description || '').trim(),
      status: f.status === 1 ? 1 : 0
    }
    const isEdit = !!this.data.editingId
    const url = isEdit ? '/api/wx/admin/point-rules/' + this.data.editingId : '/api/wx/admin/point-rules'
    const method = isEdit ? 'PUT' : 'POST'
    this.setData({ submitting: true })
    wx.showLoading({ title: '保存中', mask: true })
    request({ url, method, data: payload }).then(() => {
      wx.hideLoading()
      wx.showToast({ title: '保存成功', icon: 'success' })
      this.setData({ submitting: false, formVisible: false })
      this.loadRules()
    }).catch(() => {
      wx.hideLoading()
      this.setData({ submitting: false })
    })
  }
})
