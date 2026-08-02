<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box" v-loading="loading">
      <div class="page-title">系统配置</div>
      <el-form :model="form" label-width="140px" style="max-width: 600px">
        <el-divider content-position="left">联系方式</el-divider>
        <el-form-item label="客服电话">
          <el-input v-model="form.customer_service_phone" placeholder="客服热线电话" />
          <div class="field-hint">小程序「我的」页「电话热线」点击拨打的号码</div>
        </el-form-item>
        <el-form-item label="售后微信">
          <el-input v-model="form.after_sales_wechat" placeholder="售后微信号" />
          <div class="field-hint">售后联系微信号，用户可添加好友咨询售后问题</div>
        </el-form-item>

        <!-- AI 数字员工配置（仅超管可见） -->
        <el-divider content-position="left">AI 数字员工</el-divider>
        <template v-if="authStore.isSuperAdmin()">
          <el-form-item label="启用AI数字员工">
            <el-switch v-model="aiForm.ai_employee_enabled" />
            <div class="field-hint">开启后管理后台右上角显示数字员工按钮，可随时咨询业务问题</div>
          </el-form-item>
          <!-- 英伟达 -->
          <el-form-item v-for="p in nvidiaProviders" :key="p.value" :label="p.name">
            <div class="ai-key-row">
              <el-input v-model="aiApiKeys[p.value]" type="password" show-password :placeholder="p.has_key ? '已配置，留空不修改' : '请输入API Key'" />
              <span v-if="p.tag" class="ai-pill">{{ p.tag }}</span>
            </div>
          </el-form-item>
          <!-- 国产 -->
          <el-form-item v-for="p in domesticProviders" :key="p.value" :label="p.name">
            <div class="ai-key-row">
              <el-input v-model="aiApiKeys[p.value]" type="password" show-password :placeholder="p.has_key ? '已配置，留空不修改' : '请输入API Key'" />
              <span v-if="p.tag" class="ai-pill">{{ p.tag }}</span>
            </div>
          </el-form-item>
          <el-form-item label="系统提示词">
            <el-input v-model="aiForm.ai_system_prompt" type="textarea" :rows="4" placeholder="留空使用默认提示词" />
            <div class="field-hint">自定义AI行为指令，留空使用内置默认提示词</div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSaveAll">保存全部配置</el-button>
          </el-form-item>
        </template>
        <el-form-item v-else label="AI数字员工">
          <div class="field-hint">仅超级管理员可配置</div>
        </el-form-item>

        <el-divider content-position="left">订单配置</el-divider>
        <el-form-item label="支付超时(分钟)">
          <el-input-number v-model="form.order_expire_minutes" :min="1" :max="120" />
          <div class="field-hint">用户下单后未支付的订单自动取消的时间</div>
        </el-form-item>
        <el-form-item label="退票截止时间(小时)">
          <el-input-number v-model="form.refund_before_departure_hours" :min="0" :max="48" />
          <div class="field-hint">发车前多少小时内不允许退票，设为 0 表示始终可退</div>
        </el-form-item>
        <el-form-item label="退票手续费率(%)">
          <el-input-number v-model="form.refund_fee_rate" :min="0" :max="100" :precision="1" />
          <div class="field-hint">退票时扣除的手续费比例，0 表示全额退款</div>
        </el-form-item>
        <el-divider content-position="left">首页公告</el-divider>
        <el-form-item label="公告内容">
          <el-input v-model="form.notice" type="textarea" :rows="4" placeholder="首页滚动公告，留空则不显示" />
          <div class="field-hint">小程序首页顶部滚动显示的通知文案</div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSaveAll">保存全部配置</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { configApi, aiApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
const loading = ref(false)
const authStore = useAuthStore()
const form = reactive<any>({
  customer_service_phone: '',
  after_sales_wechat: '',
  order_expire_minutes: 15,
  refund_before_departure_hours: 2,
  refund_fee_rate: 10,
  notice: '',
})
// AI 数字员工配置
const aiForm = reactive<any>({
  ai_employee_enabled: false,
  ai_system_prompt: '',
})
const providers = ref<any[]>([])
const aiApiKeys = reactive<Record<string, string>>({})
const nvidiaProviders = computed(() => providers.value.filter((p: any) => p.group === 'nvidia'))
const domesticProviders = computed(() => providers.value.filter((p: any) => p.group === 'domestic'))
const loadData = async () => {
  loading.value = true
  try {
    const res: any = await configApi.get()
    const data = res.data || {}
    form.customer_service_phone = data.customer_service_phone || ''
    form.after_sales_wechat = data.after_sales_wechat || ''
    form.order_expire_minutes = Number(data.order_expire_minutes) || 15
    form.refund_before_departure_hours = Number(data.refund_before_departure_hours) || 2
    form.refund_fee_rate = Number(data.refund_fee_rate) || 0
    form.notice = data.notice || ''
    // 加载 AI 配置
    if (authStore.isSuperAdmin()) {
      try {
        const [aiRes, modelsRes] = await Promise.all([aiApi.getConfig(), aiApi.getModels()])
        const aiData = (aiRes as any).data || {}
        aiForm.ai_employee_enabled = aiData.ai_employee_enabled ?? false
        aiForm.ai_system_prompt = aiData.ai_system_prompt || ''
        providers.value = (modelsRes as any).data?.providers || []
        for (const p of providers.value) {
          aiApiKeys[p.value] = ''
        }
      } catch { /* AI配置加载失败不阻塞 */ }
    }
  } finally { loading.value = false }
}
// 统一保存全部配置（普通配置 + AI配置），避免管理员遗漏点击不同保存按钮
const handleSaveAll = async () => {
  loading.value = true
  try {
    // 1. 保存普通配置
    const configs: Record<string, string> = {
      customer_service_phone: String(form.customer_service_phone),
      after_sales_wechat: String(form.after_sales_wechat),
      order_expire_minutes: String(form.order_expire_minutes),
      refund_before_departure_hours: String(form.refund_before_departure_hours),
      refund_fee_rate: String(form.refund_fee_rate),
      notice: String(form.notice),
    }
    const promises: Promise<any>[] = [configApi.update({ configs })]

    // 2. 同时保存AI配置（仅超管）
    if (authStore.isSuperAdmin()) {
      const aiConfigs: Record<string, string> = {
        ai_employee_enabled: aiForm.ai_employee_enabled ? 'true' : 'false',
        ai_system_prompt: aiForm.ai_system_prompt,
      }
      for (const p of providers.value) {
        if (aiApiKeys[p.value]) {
          aiConfigs[`ai_api_key_${p.value}`] = aiApiKeys[p.value]
        }
      }
      promises.push(aiApi.updateConfig({ configs: aiConfigs }))
    }

    await Promise.all(promises)

    // 3. 清空API Key输入框并刷新服务商列表
    if (authStore.isSuperAdmin()) {
      for (const p of providers.value) {
        aiApiKeys[p.value] = ''
      }
      const modelsRes: any = await aiApi.getModels()
      providers.value = modelsRes.data?.providers || []
    }

    ElMessage.success('全部配置保存成功')
  } catch (e) {
    /* 错误由axios拦截器处理 */
  } finally {
    loading.value = false
  }
}
onMounted(loadData)
</script>
<style scoped>
.page-title { font-size: 18px; font-weight: 600; margin-bottom: 20px; color: #1a1a1a }
.field-hint { font-size: 12px; color: #909399; line-height: 1.6; margin-top: 4px }
/* AI 药丸标签：黑色药丸白色字体，放在输入框右方 */
.ai-key-row { display: flex; align-items: center; gap: 8px; width: 100% }
.ai-pill { display: inline-block; padding: 2px 10px; border-radius: 99px; font-size: 11px; font-weight: 500; line-height: 1.6; white-space: nowrap; background: #000; color: #fff; flex-shrink: 0 }
</style>
