<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <div class="card-box">
      <div class="toolbar"><el-button type="primary" @click="handleAdd">新增轮播图</el-button></div>
    </div>
    <div class="card-box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column type="index" label="序号" width="70" />
        <el-table-column prop="title" label="标题" />
        <el-table-column label="图片" width="120"><template #default="{ row }"><img v-if="row.image_url" :src="row.image_url" style="width: 80px; height: 40px; object-fit: cover; border-radius: 4px" /></template></el-table-column>
        <el-table-column prop="sort_order" label="排序" width="80" />
        <el-table-column prop="status" label="状态" width="80"><template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="160" fixed="right"><template #default="{ row }"><el-button size="small" @click="handleEdit(row)">编辑</el-button><el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button></template></el-table-column>
      </el-table>
    </div>
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑轮播图' : '新增轮播图'" width="560px">
      <el-form :model="editing" label-width="90px">
        <el-form-item label="标题"><el-input v-model="editing.title" placeholder="请输入标题（选填）" /></el-form-item>
        <el-form-item label="标题颜色" v-if="editing.title">
          <div class="color-picker-row">
            <div
              v-for="c in colorOptions" :key="c.value"
              class="color-swatch"
              :class="{ active: editing.title_color === c.value }"
              :style="{ background: c.hex }"
              @click="editing.title_color = c.value"
            >
              <span class="color-label">{{ c.label }}</span>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="标题特效" v-if="editing.title">
          <el-radio-group v-model="editing.title_effect">
            <el-radio :value="0">无</el-radio>
            <el-radio :value="1">阴影</el-radio>
            <el-radio :value="2">磨砂玻璃</el-radio>
            <el-radio :value="3">液态玻璃</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="图片" required>
          <el-upload
            class="banner-uploader"
            :show-file-list="false"
            :http-request="handleUpload"
            :before-upload="beforeUpload"
            accept="image/*"
          >
            <img v-if="editing.image_url" :src="editing.image_url" class="banner-preview" />
            <div v-else class="banner-uploader-placeholder">
              <el-icon class="banner-uploader-icon"><Plus /></el-icon>
              <span>点击上传图片</span>
            </div>
          </el-upload>
        </el-form-item>
        <el-form-item label="效果预览" v-if="editing.image_url">
          <div class="preview-box">
            <img :src="editing.image_url" class="preview-img" />
            <div v-if="editing.title" class="preview-title-wrap" :class="getPreviewEffectClass()">
              <span class="preview-title" :style="{ color: getPreviewColor() }">{{ editing.title }}</span>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="排序"><el-input-number v-model="editing.sort_order" :min="0" /></el-form-item>
        <el-form-item label="状态"><el-switch v-model="editing.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleSave">保存</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { bannerApi, uploadApi } from '@/api'
const list = ref<any[]>([]), loading = ref(false), dialogVisible = ref(false)
const editing = reactive({ id: 0, title: '', title_color: '', title_effect: 0, image_url: '', link_type: 0, link_value: '', sort_order: 0, status: 1 })
const colorOptions = [
  { label: '白', value: 'white', hex: '#ffffff' },
  { label: '黑', value: 'black', hex: '#000000' },
  { label: '黄', value: 'yellow', hex: '#ffcc00' },
  { label: '红', value: 'red', hex: '#ff3333' },
  { label: '蓝', value: 'blue', hex: '#2689ff' },
  { label: '绿', value: 'green', hex: '#33cc33' },
  { label: '青', value: 'cyan', hex: '#00cccc' },
  { label: '紫', value: 'purple', hex: '#9933ff' },
]
const titleColorMap: Record<string, string> = { white: '#ffffff', black: '#000000', yellow: '#ffcc00', red: '#ff3333', blue: '#2689ff', green: '#33cc33', cyan: '#00cccc', purple: '#9933ff' }
const getPreviewColor = () => titleColorMap[editing.title_color] || '#ffffff'
const getPreviewEffectClass = () => {
  if (editing.title_effect === 1) return 'pv-effect-shadow'
  if (editing.title_effect === 2) return 'pv-effect-glass'
  if (editing.title_effect === 3) return 'pv-effect-liquid'
  return ''
}
const loadData = async () => { loading.value = true; try { const res: any = await bannerApi.list(); list.value = res.data || [] } finally { loading.value = false } }
const handleAdd = () => { Object.assign(editing, { id: 0, title: '', title_color: '', title_effect: 0, image_url: '', link_type: 0, link_value: '', sort_order: 0, status: 1 }); dialogVisible.value = true }
const handleEdit = (row: any) => { Object.assign(editing, { title_color: '', title_effect: 0, ...row }); dialogVisible.value = true }
const beforeUpload = (file: File) => {
  const isImage = file.type.startsWith('image/')
  const isLt5M = file.size / 1024 / 1024 < 5
  if (!isImage) { ElMessage.error('只能上传图片文件'); return false }
  if (!isLt5M) { ElMessage.error('图片大小不能超过5MB'); return false }
  return true
}
const handleUpload = async (options: any) => {
  const formData = new FormData()
  formData.append('file', options.file)
  try {
    const res: any = await uploadApi.upload(formData)
    editing.image_url = res.data.url
    ElMessage.success('图片上传成功')
  } catch (e) { /* 错误由axios拦截器处理 */ }
}
const handleSave = async () => {
  if (!editing.image_url) { ElMessage.warning('请上传图片'); return }
  try { if (editing.id) await bannerApi.update(editing.id, { ...editing }); else await bannerApi.create({ ...editing }); ElMessage.success('保存成功'); dialogVisible.value = false; loadData() } catch (e) { /* 错误由axios拦截器处理 */ }
}
const handleDelete = async (row: any) => { try { await ElMessageBox.confirm('确认删除该轮播图？', '提示', { type: 'warning' }); await bannerApi.delete(row.id); ElMessage.success('删除成功'); loadData() } catch (e) { /* 用户取消或错误由拦截器处理 */ } }
onMounted(loadData)
</script>
<style scoped>
.toolbar { display: flex; gap: 8px }
.banner-uploader :deep(.el-upload) {
  width: 100%;
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: border-color 0.2s;
}
.banner-uploader :deep(.el-upload:hover) {
  border-color: #409eff;
}
.banner-uploader-placeholder {
  width: 100%;
  height: 140px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #8c8c8c;
}
.banner-uploader-icon {
  font-size: 28px;
  color: #8c8c8c;
}
.banner-preview {
  width: 100%;
  height: 140px;
  object-fit: cover;
  display: block;
}
.color-picker-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.color-swatch {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  cursor: pointer;
  border: 2px solid transparent;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
}
.color-swatch:hover {
  transform: scale(1.08);
}
.color-swatch.active {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64,158,255,0.3);
}
.color-label {
  font-size: 10px;
  font-weight: 700;
  color: #fff;
  text-shadow: 0 1px 2px rgba(0,0,0,0.5);
  padding-bottom: 2px;
}

/* 实时预览区 */
.preview-box {
  width: 100%;
  height: 160px;
  border-radius: 10px;
  overflow: hidden;
  position: relative;
  background: #f0f2f5;
}
.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.preview-title-wrap {
  position: absolute;
  top: 14px;
  left: 16px;
  z-index: 2;
  max-width: calc(100% - 32px);
}
.preview-title {
  font-size: 18px;
  font-weight: 800;
  line-height: 1.4;
}

/* 阴影特效 */
.pv-effect-shadow .preview-title {
  text-shadow: 0 1px 6px rgba(0,0,0,0.45);
}

/* 磨砂玻璃特效 */
.pv-effect-glass {
  background: rgba(255, 255, 255, 0.18);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border-radius: 6px;
  padding: 4px 10px;
  border: 1px solid rgba(255, 255, 255, 0.2);
}
.pv-effect-glass .preview-title {
  text-shadow: none;
}

/* 液态玻璃特效 - 更透明更通透 */
.pv-effect-liquid {
  background: rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(2px) saturate(180%);
  -webkit-backdrop-filter: blur(2px) saturate(180%);
  border-radius: 6px;
  padding: 4px 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: 0 1px 3px rgba(0,0,0,0.06), inset 0 1px 0 rgba(255,255,255,0.12);
}
.pv-effect-liquid .preview-title {
  text-shadow: none;
}
</style>
