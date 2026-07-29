// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声
var log = require('../../utils/log')
const { request, upload } = require('../../utils/request')
const { normalizeUrl, maskPhone } = require('../../utils/format')
const { logout, deleteAccount } = require('../../utils/auth-mixin')

Page({
  data: {
    userInfo: {
      nickName: '未登录',
      phone: '点击登录',
      avatarUrl: '',
      isLogin: false
    },
    logoutPosition: 'mine_bottom',
    phoneMasked: true,
    maskedPhone: ''
  },

  onShow() {
    this.checkLoginStatus()
    this.loadConfig()
  },

  // 检查登录状态
  checkLoginStatus() {
    const token = wx.getStorageSync('user_token')
    const userInfo = wx.getStorageSync('user_info')
    if (token && userInfo) {
      const phone = userInfo.phone || '未绑定'
      this.setData({
        userInfo: {
          nickName: userInfo.nickname || '用户',
          phone: phone,
          avatarUrl: normalizeUrl(userInfo.avatar_url),
          isLogin: true
        },
        phoneMasked: true,
        maskedPhone: maskPhone(phone)
      })
    } else {
      wx.navigateBack()
    }
  },

  // 切换手机号脱敏/明文
  togglePhoneMask() {
    this.setData({ phoneMasked: !this.data.phoneMasked })
  },

  // 加载系统配置
  loadConfig() {
    request({ url: '/api/wx/config', method: 'GET' }).then(res => {
      const config = res.data || {}
      this.setData({
        logoutPosition: config.logout_position || 'mine_bottom'
      })
    }).catch((e) => { log.error('加载系统配置失败', e) })
  },

  // 选择微信头像
  onChooseAvatar(e) {
    const tempUrl = e.detail.avatarUrl
    if (!tempUrl) return
    // chooseAvatar 返回临时路径可能无扩展名，后端白名单会拒绝
    // 复制到带 .png 扩展名的路径再上传（chooseAvatar 始终返回 PNG）
    const hasImgExt = /\.(png|jpg|jpeg|gif|webp)$/i.test(tempUrl)
    if (hasImgExt) {
      this.uploadAvatar(tempUrl)
    } else {
      const fs = wx.getFileSystemManager()
      const newPath = wx.env.USER_DATA_PATH + '/avatar_' + Date.now() + '.png'
      fs.copyFile({
        srcPath: tempUrl,
        destPath: newPath,
        success: () => this.uploadAvatar(newPath),
        fail: (err) => {
          log.error('copyFile 失败，尝试直接上传', err)
          this.uploadAvatar(tempUrl)
        }
      })
    }
  },

  // 上传头像到服务器并更新用户信息
  // 注意：request.js 的 upload/request 失败时会自动调用 wx.showToast，
  // showToast 会自动关闭 showLoading，所以 .catch 里不需要再调用 hideLoading，
  // 否则会触发"showLoading 与 hideLoading 必须配对使用"的警告
  uploadAvatar(filePath) {
    wx.showLoading({ title: '上传中...', mask: true })
    upload({ url: '/api/wx/upload', filePath: filePath, name: 'file' }).then((upRes) => {
      const serverUrl = upRes.data.url
      return request({ url: '/api/wx/user', method: 'PUT', data: { avatar_url: serverUrl } })
    }).then((res) => {
      const user = res.data
      wx.setStorageSync('user_info', user)
      this.setData({
        'userInfo.avatarUrl': normalizeUrl(user.avatar_url),
        'userInfo.nickName': user.nickname || this.data.userInfo.nickName
      })
      wx.hideLoading()
      wx.showToast({ title: '头像更新成功', icon: 'success' })
    }).catch((err) => {
      log.error('头像上传失败', err)
      // request.js 失败时已调用 wx.showToast，showLoading 已被自动关闭，无需再调 hideLoading
    })
  },

  // 昵称输入失焦时保存
  onNicknameBlur(e) {
    const nickname = (e.detail.value || '').trim()
    if (!nickname || nickname === this.data.userInfo.nickName) return
    request({ url: '/api/wx/user', method: 'PUT', data: { nickname } }).then((res) => {
      const user = res.data
      wx.setStorageSync('user_info', user)
      this.setData({ 'userInfo.nickName': user.nickname })
      wx.showToast({ title: '昵称修改成功', icon: 'success' })
    }).catch((err) => {
      log.error('昵称修改失败', err)
    })
  },

  // 退出登录
  handleLogout() {
    wx.showModal({
      title: '提示',
      content: '确认退出登录？',
      success: async (res) => {
        if (res.confirm) {
          await logout()
          wx.showToast({ title: '已退出', icon: 'success' })
          setTimeout(() => {
            wx.navigateBack()
          }, 800)
        }
      }
    })
  },

  // 注销账号（清除所有个人数据）
  handleDeleteAccount() {
    wx.showModal({
      title: '注销账号',
      content: '注销后将清除您的全部个人信息（乘客信息、优惠券、积分等），且不可恢复。确定要注销吗？',
      confirmText: '确认注销',
      cancelText: '再想想',
      confirmColor: '#e64340',
      success: async (res) => {
        if (!res.confirm) return
        // 二次确认，防止误操作
        wx.showModal({
          title: '最后确认',
          content: '此操作不可逆！您的账号将被永久注销，无法再登录。确定继续？',
          confirmText: '确认注销',
          cancelText: '取消',
          confirmColor: '#e64340',
          success: async (res2) => {
            if (!res2.confirm) return
            wx.showLoading({ title: '注销中...', mask: true })
            try {
              await deleteAccount()
              wx.hideLoading()
              wx.showToast({ title: '账号已注销', icon: 'success' })
              setTimeout(() => {
                wx.navigateBack()
              }, 800)
            } catch (e) {
              wx.hideLoading()
            }
          }
        })
      }
    })
  }
})
