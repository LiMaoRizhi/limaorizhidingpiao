const { request } = require('../../utils/request')

Page({
  data: {
    type: 'user_agreement', // user_agreement | privacy_policy
    title: '用户协议',
    content: '',
    loading: true
  },

  onLoad(options) {
    const type = options.type || 'user_agreement'
    const title = type === 'privacy_policy' ? '隐私政策' : '用户协议'
    this.setData({ type, title })
    this.loadAgreement(type)
  },

  loadAgreement(type) {
    // 从公开配置接口获取协议内容（无需登录）
    request({
      url: '/api/wx/config',
      method: 'GET',
      silent: true
    }).then(res => {
      const config = res.data || {}
      const content = config[type] || ''
      this.setData({ content, loading: false })
    }).catch(() => {
      this.setData({ content: '', loading: false })
    })
  }
})
