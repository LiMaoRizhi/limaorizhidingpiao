<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<!-- 品牌设置：修改系统名称和 Logo，保存后侧边栏/登录页即时生效 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <h3 style="margin-bottom: 24px">品牌设置</h3>
      <el-alert
        title="修改系统名称和 Logo 后，侧边栏、登录页等位置将即时显示新品牌信息。"
        type="info" :closable="false" show-icon
        style="margin-bottom: 20px; max-width: 560px"
      />
      <el-form label-width="100px" style="max-width: 560px">
        <!-- Logo 上传 -->
        <el-form-item label="系统Logo">
          <div class="logo-upload-area">
            <div class="logo-preview-box">
              <img :src="logoPreview" class="logo-preview-img" />
            </div>
            <el-upload
              :show-file-list="false"
              :before-upload="handleLogoUpload"
              accept="image/png,image/jpeg,image/gif,image/webp"
            >
              <el-button type="primary" :loading="uploading">
                {{ uploading ? '上传中...' : '上传Logo' }}
              </el-button>
            </el-upload>
            <span class="logo-tip">建议尺寸 128×128，支持 PNG/JPG/GIF/WebP</span>
          </div>
        </el-form-item>

        <!-- 系统名称 -->
        <el-form-item label="系统名称">
          <el-input
            v-model="form.name"
            placeholder="如：狸猫日志售票"
            maxlength="20"
            show-word-limit
          />
        </el-form-item>

        <!-- 操作按钮 -->
        <el-form-item>
          <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
          <el-button @click="handleReset">重置默认</el-button>
        </el-form-item>
      </el-form>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useBrandStore } from '@/stores/brand'
import { uploadApi } from '@/api'

const brandStore = useBrandStore()

const form = reactive({
  name: brandStore.systemName,
  logo: brandStore.logoUrl,
})
const logoPreview = ref(brandStore.logoUrl)
const uploading = ref(false)
const saving = ref(false)

// Logo 上传：调用后端 /admin/upload 接口，返回 URL
const handleLogoUpload = async (file: File) => {
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await uploadApi.upload(formData)
    form.logo = res.data.url
    logoPreview.value = res.data.url
    ElMessage.success('Logo 上传成功，记得点保存')
  } catch (e: any) {
    ElMessage.error(e.message || 'Logo 上传失败')
  } finally {
    uploading.value = false
  }
  return false  // 阻止 el-upload 默认上传行为
}

// 保存品牌信息到后端 + store + localStorage
const handleSave = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请输入系统名称')
    return
  }
  saving.value = true
  try {
    await brandStore.updateBrand(form.name.trim(), form.logo)
    ElMessage.success('品牌信息保存成功')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

// 重置为默认值（同步到后端）
const handleReset = async () => {
  saving.value = true
  try {
    await brandStore.resetBrand()
    form.name = brandStore.systemName
    form.logo = brandStore.logoUrl
    logoPreview.value = brandStore.logoUrl
    ElMessage.success('已重置为默认品牌信息')
  } catch (e: any) {
    ElMessage.error(e.message || '重置失败')
  } finally {
    saving.value = false
  }
}

// 页面加载时从后端同步品牌配置
onMounted(async () => {
  await brandStore.syncFromBackend()
  form.name = brandStore.systemName
  form.logo = brandStore.logoUrl
  logoPreview.value = brandStore.logoUrl
})

</script>

<style scoped>
.logo-upload-area {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.logo-preview-box {
  width: 80px;
  height: 80px;
  border-radius: 12px;
  border: 1px solid #e0e0e0;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f9f9f9;
  flex-shrink: 0;
}
.logo-preview-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.logo-tip {
  font-size: 12px;
  color: #999;
}

</style>
