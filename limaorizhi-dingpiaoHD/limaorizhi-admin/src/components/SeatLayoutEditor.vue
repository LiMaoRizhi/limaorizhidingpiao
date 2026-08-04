<!-- 座位布局可视化编辑器 - 灵活自定义 -->
<template>
  <div class="seat-editor">
    <div class="editor-toolbar">
      <span class="toolbar-label">布局编辑</span>
      <el-button size="small" @click="addRow">+行</el-button>
      <el-button size="small" @click="removeRow" :disabled="rows <= 2">-行</el-button>
      <el-button size="small" @click="addCol">+列</el-button>
      <el-button size="small" @click="removeCol" :disabled="cols <= 2">-列</el-button>
      <el-divider direction="vertical" />
      <el-button size="small" type="primary" @click="autoFit">匹配座位数</el-button>
      <el-button size="small" @click="renumber">重新编号</el-button>
    </div>

    <div class="editor-legend">
      <span class="legend-item"><span class="dot seat-dot"></span>座位</span>
      <span class="legend-item"><span class="dot aisle-dot"></span>过道</span>
      <span class="legend-item"><span class="dot empty-dot"></span>空白</span>
      <span class="legend-item"><span class="dot driver-dot"></span>驾驶区</span>
      <span class="legend-hint">点击格子切换类型 ｜ 编辑完点"匹配座位数"自动对齐</span>
    </div>

    <div class="bus-front">← 车头</div>

    <div class="bus-body" :style="{ gridTemplateColumns: `repeat(${cols}, 52px)` }">
      <template v-for="r in rows" :key="r">
        <div
          v-for="c in cols"
          :key="r + '-' + c"
          class="cell"
          :class="cellClass(r, c)"
          @click="cycleCell(r, c)"
        >
          <span class="cell-label" v-if="getCell(r,c).type === 'seat'">{{ getCell(r,c).seatNo }}</span>
        </div>
      </template>
    </div>

    <div class="bus-rear">后排 →</div>

    <div class="editor-summary">
      总座位数：<b>{{ seatCount }}</b>　|　行：{{ rows }}　列：{{ cols }}
      <span v-if="layoutValid" style="color:#67c23a">　✓ 匹配</span>
      <span v-else style="color:#f56c6c">　✗ 与车辆座位数({{vehicleSeatCount}})不一致，点"匹配座位数"自动调整</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface Cell {
  row: number; col: number; type: string; seatNo: number
}

const props = defineProps<{ modelValue: string; vehicleSeatCount: number }>()
const emit = defineEmits(['update:modelValue'])

const rows = ref(10)
const cols = ref(5)
const cells = ref<Cell[]>([])
const userEdited = ref(false) // 用户手动改过布局就不自动覆盖

function initFromJSON(json: string) {
  if (json) {
    try {
      const layout = JSON.parse(json)
      if (layout.rows && layout.cols && layout.cells && layout.cells.length > 0) {
        rows.value = layout.rows
        cols.value = layout.cols
        cells.value = layout.cells.map((c: any) => ({ row: c.row, col: c.col, type: c.type, seatNo: c.seat_no || 0 }))
        return
      }
    } catch (_) { /* fall through */ }
  }
  // 无已有布局：按车辆座位数自动生成，没有则给个空网格
  if (props.vehicleSeatCount > 0) {
    autoFit()
  } else {
    rows.value = 10; cols.value = 5; cells.value = []
    userEdited.value = false
  }
}

function getCell(r: number, c: number): Cell {
  const found = cells.value.find(cl => cl.row === r && cl.col === c)
  return found || { row: r, col: c, type: 'empty', seatNo: 0 }
}

function cycleCell(r: number, c: number) {
  userEdited.value = true
  const idx = cells.value.findIndex(cl => cl.row === r && cl.col === c)
  const current = idx >= 0 ? cells.value[idx].type : 'empty'
  let next = ''
  if (current === 'empty') next = 'seat'
  else if (current === 'seat') next = 'aisle'
  else if (current === 'aisle') next = 'driver'
  else next = 'empty'

  if (idx >= 0) {
    if (next === 'empty') { cells.value.splice(idx, 1) }
    else { cells.value[idx] = { row: r, col: c, type: next, seatNo: 0 } }
  } else if (next !== 'empty') {
    cells.value.push({ row: r, col: c, type: next, seatNo: 0 })
  }
  renumber()
}

function cellClass(r: number, c: number) {
  const cell = getCell(r, c)
  return {
    'cell-seat': cell.type === 'seat',
    'cell-aisle': cell.type === 'aisle',
    'cell-empty': cell.type === 'empty' || cell.type === '',
    'cell-driver': cell.type === 'driver'
  }
}

function addRow() { userEdited.value = true; rows.value++; syncCells() }
function removeRow() { if (rows.value > 2) { userEdited.value = true; rows.value--; syncCells() } }
function addCol() { userEdited.value = true; cols.value++; syncCells() }
function removeCol() { if (cols.value > 2) { userEdited.value = true; cols.value--; syncCells() } }

function syncCells() {
  cells.value = cells.value.filter(cl => cl.row <= rows.value && cl.col <= cols.value)
  renumber()
}

function renumber() {
  let num = 0
  for (let r = 1; r <= rows.value; r++) {
    for (let c = 1; c <= cols.value; c++) {
      const idx = cells.value.findIndex(cl => cl.row === r && cl.col === c)
      if (idx >= 0 && cells.value[idx].type === 'seat') {
        num++
        cells.value[idx] = { ...cells.value[idx], seatNo: num }
      }
    }
  }
  emitLayout()
}

// 按车辆座位数自动生成匹配布局
function autoFit() {
  const target = props.vehicleSeatCount
  if (target <= 0) return

  // 5列标准布局：驾驶区占1列，实际可用4列
  // 前两排预留驾驶区 后排连排
  cols.value = 5
  const rowSeats = 4 // 每行4座（2+过道+2 或 4连排）
  const lastRowFull = target % rowSeats === 0 ? rowSeats : target % rowSeats
  const middleTotal = target - lastRowFull
  const middleRows = Math.ceil(middleTotal / rowSeats)
  rows.value = middleRows + 1 // +1 是最后排

  cells.value = []

  // 驾驶区：左上角1格
  cells.value.push({ row: 1, col: 1, type: 'driver', seatNo: 0 })

  // 中间排：过道在第3列，左右各2座
  for (let r = 1; r <= middleRows; r++) {
    cells.value.push({ row: r, col: 2, type: 'seat', seatNo: 0 })
    cells.value.push({ row: r, col: 3, type: 'aisle', seatNo: 0 })
    cells.value.push({ row: r, col: 4, type: 'seat', seatNo: 0 })
    cells.value.push({ row: r, col: 5, type: 'seat', seatNo: 0 })
  }

  // 最后排：连排
  const lastRow = middleRows + 1
  for (let c = 2; c <= 5; c++) {
    cells.value.push({ row: lastRow, col: c, type: 'seat', seatNo: 0 })
  }

  renumber()
  userEdited.value = false
}

const seatCount = computed(() => cells.value.filter(c => c.type === 'seat').length)
const layoutValid = computed(() => seatCount.value === props.vehicleSeatCount)

function emitLayout() {
  const layout = {
    rows: rows.value,
    cols: cols.value,
    cells: cells.value.map(c => ({ row: c.row, col: c.col, type: c.type, seat_no: c.seatNo }))
  }
  emit('update:modelValue', JSON.stringify(layout))
}

watch(() => props.modelValue, (val) => { initFromJSON(val) }, { immediate: true })

// 座位数变化时自动匹配（用户没手动改过布局的话）
watch(() => props.vehicleSeatCount, (newCount) => {
  if (newCount > 0 && !userEdited.value && seatCount.value !== newCount) {
    autoFit()
  }
})

watch(seatCount, () => { emitLayout() })
</script>

<style scoped>
.seat-editor { border: 1px solid #dcdfe6; border-radius: 8px; padding: 16px; background: #fafafa; }

.editor-toolbar { display: flex; align-items: center; gap: 6px; margin-bottom: 12px; flex-wrap: wrap; }
.toolbar-label { font-weight: 600; color: #303133; margin-right: 8px; }

.editor-legend { display: flex; align-items: center; gap: 16px; margin-bottom: 12px; font-size: 12px; color: #909399; }
.legend-item { display: flex; align-items: center; gap: 4px; }
.legend-hint { margin-left: auto; }
.dot { width: 14px; height: 14px; border-radius: 3px; display: inline-block; }
.seat-dot { background: #303133; }
.aisle-dot { background: #c0c4cc; }
.empty-dot { background: #f5f5f5; border: 1px dashed #dcdfe6; }
.driver-dot { background: #909399; }

.bus-front, .bus-rear {
  text-align: center; font-size: 12px; color: #909399; padding: 4px 0;
  background: #f0f0f0; border-radius: 4px; margin: 4px 0;
}

.bus-body {
  display: grid;
  gap: 4px;
  justify-content: center;
  padding: 8px;
  background: #ffffff;
  border: 2px solid #dcdfe6;
  border-radius: 8px;
  min-height: 100px;
}

.cell {
  width: 48px; height: 48px;
  border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; user-select: none;
  font-size: 11px; font-weight: 600;
  transition: all 0.15s;
  border: 2px solid transparent;
}
.cell:hover { transform: scale(1.08); z-index: 1; }

.cell-seat { background: #303133; color: #fff; border-color: #1a1a1a; }
.cell-seat:hover { background: #555; }

.cell-aisle { background: #e8e8e8; color: #bbb; cursor: default; }
.cell-aisle:hover { transform: none; }

.cell-empty { background: #fafafa; border: 1px dashed #e4e7ed; color: #ccc; }
.cell-empty:hover { background: #f0f0f0; }

.cell-driver { background: #909399; color: #fff; border-color: #707378; }
.cell-driver:hover { background: #a8abb0; }

.editor-summary { margin-top: 10px; font-size: 13px; color: #606266; text-align: center; }
</style>
