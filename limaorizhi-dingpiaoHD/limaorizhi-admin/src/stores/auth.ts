/* limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 */
import { defineStore } from 'pinia'
import { http } from '@/utils/request'

interface AdminInfo {
  id: number
  username: string
  real_name: string
  avatar_url: string
  role: number
  must_change_password?: boolean
}

export const useAuthStore = defineStore('auth', {
  state: () => {
    // F-1安全注释：token 存于 localStorage 便于持久登录，但存在 XSS 风险。
    // 已有防护措施：1. fetchProfile 从服务器验证角色（防篡改）
    // 2. 后端 token_invalid_before 机制实现登出失效
    // 3. 路由守卫在进入页面前调用 fetchProfile 验证 token 有效性
    // 待改进：后续可迁移为 httpOnly cookie + CSRF token 方案
    // 安全解析 localStorage，防止 JSON 解析异常导致应用崩溃
    let adminInfo: AdminInfo | null = null
    try {
      adminInfo = JSON.parse(localStorage.getItem('limaorizhi_admin_info') || 'null') as AdminInfo | null
    } catch {
      localStorage.removeItem('limaorizhi_admin_info')
    }
    return {
      token: localStorage.getItem('limaorizhi_admin_token') || '',
      adminInfo,
      // 默认口令强制改密标志：从 localStorage 恢复，登录后后端返回该标志时设置
      mustChangePassword: localStorage.getItem('limaorizhi_admin_must_change') === 'true',
      profileValidated: false, // 是否已从服务器验证过角色信息
    }
  },
  actions: {
    async login(username: string, password: string, captchaVerification: string) {
      const res = await http.post<{ token: string } & AdminInfo>('/admin/login', { username, password, captchaVerification })
      this.token = res.data.token
      this.mustChangePassword = !!res.data.must_change_password
      this.adminInfo = {
        id: res.data.id,
        username: res.data.username,
        real_name: res.data.real_name,
        avatar_url: res.data.avatar_url,
        role: res.data.role,
        must_change_password: this.mustChangePassword,
      }
      localStorage.setItem('limaorizhi_admin_token', this.token)
      localStorage.setItem('limaorizhi_admin_info', JSON.stringify(this.adminInfo))
      // 持久化强制改密标志，路由守卫据此拦截
      if (this.mustChangePassword) {
        localStorage.setItem('limaorizhi_admin_must_change', 'true')
      } else {
        localStorage.removeItem('limaorizhi_admin_must_change')
      }
      return res.data
    },
    async logout() {
      try {
        if (this.token) {
          await http.post('/admin/logout')
        }
      } catch {
        // 即使后端调用失败也继续清除本地登录状态
      } finally {
        this.token = ''
        this.adminInfo = null
        this.mustChangePassword = false
        localStorage.removeItem('limaorizhi_admin_token')
        localStorage.removeItem('limaorizhi_admin_info')
        localStorage.removeItem('limaorizhi_admin_must_change')
      }
    },
    isLogin() {
      return !!this.token
    },
    isSuperAdmin() {
      return this.adminInfo?.role === 1
    },
    // 从服务器获取最新管理员信息（防止localStorage角色被篡改）
    async fetchProfile() {
      if (!this.token) return
      try {
        const res = await http.get<AdminInfo>('/admin/profile')
        // 从服务器同步强制改密标志，路由守卫据此拦截（防localStorage被篡改绕过）
        this.mustChangePassword = !!res.data.must_change_password
        this.adminInfo = {
          id: res.data.id,
          username: res.data.username,
          real_name: res.data.real_name,
          avatar_url: res.data.avatar_url,
          role: res.data.role,
          must_change_password: this.mustChangePassword,
        }
        localStorage.setItem('limaorizhi_admin_info', JSON.stringify(this.adminInfo))
        if (this.mustChangePassword) {
          localStorage.setItem('limaorizhi_admin_must_change', 'true')
        } else {
          localStorage.removeItem('limaorizhi_admin_must_change')
        }
        this.profileValidated = true
      } catch {
        // Token无效，清除登录态
        this.token = ''
        this.adminInfo = null
        this.mustChangePassword = false
        this.profileValidated = false
        localStorage.removeItem('limaorizhi_admin_token')
        localStorage.removeItem('limaorizhi_admin_info')
        localStorage.removeItem('limaorizhi_admin_must_change')
      }
    },
  },
})

