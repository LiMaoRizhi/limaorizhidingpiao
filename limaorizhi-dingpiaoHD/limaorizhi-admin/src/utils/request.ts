/* limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 */
import axios, { type AxiosInstance, type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import { REQUEST_TIMEOUT_MS } from '@/utils/constants'

const request: AxiosInstance = axios.create({
  // 开发环境通过 Vite proxy 转发，生产环境通过 VITE_API_BASE_URL 配置
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: REQUEST_TIMEOUT_MS,
})

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('limaorizhi_admin_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 防止Token失效时多个请求同时触发跳转
let isRedirecting = false

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== undefined && res.code !== 0) {
      ElMessage.error(res.message || '请求失败')
      if (res.code === 1002 && !isRedirecting) {
        // Token失效，清除登录状态并跳转登录页
        isRedirecting = true
        localStorage.removeItem('limaorizhi_admin_token')
        localStorage.removeItem('limaorizhi_admin_info')
        window.location.replace('/login')
        // 兜底：如果跳转失败（如被拦截），1秒后重置标志位
        setTimeout(() => { isRedirecting = false }, 1000)
      }
      // 将完整 res 附加到 error 对象，调用方可选读取 error.response.data 做后续决策（如强制删除）
      const err = new Error(res.message || '请求失败') as Error & { response?: { data: typeof res } }
      err.response = { data: res }
      return Promise.reject(err)
    }
    return res
  },
  (error) => {
    ElMessage.error(error.message || '网络错误')
    return Promise.reject(error)
  }
)

// 后端统一响应结构（默认 unknown 而非 any，强制调用方显式指定返回类型）
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

// 封装GET/POST/PUT/DELETE（收窄 any，防止类型逃逸）
export const http = {
  get<T = unknown>(url: string, params?: Record<string, unknown>): Promise<ApiResponse<T>> {
    return request.get(url, { params }) as unknown as Promise<ApiResponse<T>>
  },
  post<T = unknown>(url: string, data?: unknown): Promise<ApiResponse<T>> {
    return request.post(url, data) as unknown as Promise<ApiResponse<T>>
  },
  put<T = unknown>(url: string, data?: unknown): Promise<ApiResponse<T>> {
    return request.put(url, data) as unknown as Promise<ApiResponse<T>>
  },
  delete<T = unknown>(url: string, params?: Record<string, unknown>): Promise<ApiResponse<T>> {
    return request.delete(url, { params }) as unknown as Promise<ApiResponse<T>>
  },
  // 支持Blob下载（CSV导出等场景），绕过code解析直接返回Blob
  download(url: string, params?: Record<string, unknown>): Promise<Blob> {
    return request.get(url, { params, responseType: 'blob' }) as unknown as Promise<Blob>
  },
}

// 封装带认证头的原生 fetch，用于 SSE 等 axios 不便处理的场景
export const authFetch = (url: string, options?: RequestInit): Promise<Response> => {
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''
  const token = localStorage.getItem('limaorizhi_admin_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options?.headers as Record<string, string> || {}),
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  return fetch(`${baseURL}${url}`, { ...options, headers })
}

export default request
