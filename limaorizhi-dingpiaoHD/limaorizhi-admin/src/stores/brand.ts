/* limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 */
import { defineStore } from 'pinia'
import { configApi } from '@/api'

const STORAGE_KEY_NAME = 'limao_brand_name'
const STORAGE_KEY_LOGO = 'limao_brand_logo'
const DEFAULT_NAME = '狸猫日志售票'
const DEFAULT_LOGO = '/logo.png'

export const useBrandStore = defineStore('brand', {
  state: () => ({
    // 系统名称：默认"狸猫日志售票"，可被品牌设置页修改
    systemName: localStorage.getItem(STORAGE_KEY_NAME) || '狸猫日志售票',
    // Logo URL：默认 public/logo.png，上传后为 /uploads/xxx.jpg
    logoUrl: localStorage.getItem(STORAGE_KEY_LOGO) || '/logo.png',
  }),
  actions: {
    // 从后端加载品牌配置，同步到 store + localStorage
    async syncFromBackend() {
      try {
        const res: any = await configApi.get()
        const data = res.data || {}
        if (data.brand_name) {
          this.systemName = data.brand_name
          localStorage.setItem(STORAGE_KEY_NAME, data.brand_name)
        }
        if (data.brand_logo) {
          this.logoUrl = data.brand_logo
          localStorage.setItem(STORAGE_KEY_LOGO, data.brand_logo)
        }
      } catch {
        // 后端不可用时保持 localStorage 缓存值
      }
    },
    // 更新品牌信息，同步写入后端配置 + localStorage
    async updateBrand(name: string, logo: string) {
      await configApi.update({ configs: { brand_name: name, brand_logo: logo } })
      this.systemName = name
      this.logoUrl = logo
      localStorage.setItem(STORAGE_KEY_NAME, name)
      localStorage.setItem(STORAGE_KEY_LOGO, logo)
    },
    // 重置为默认值，同步写入后端 + 清除 localStorage
    async resetBrand() {
      await configApi.update({ configs: { brand_name: DEFAULT_NAME, brand_logo: DEFAULT_LOGO } })
      this.systemName = DEFAULT_NAME
      this.logoUrl = DEFAULT_LOGO
      localStorage.removeItem(STORAGE_KEY_NAME)
      localStorage.removeItem(STORAGE_KEY_LOGO)
    },
  },
})
