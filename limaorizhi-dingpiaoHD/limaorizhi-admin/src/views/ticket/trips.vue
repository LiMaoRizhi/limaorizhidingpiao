<!-- limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 -->
<template>
  <div class="page-container">
    <!-- 线路总览 -->
    <template v-if="viewMode === 'routes'">
      <div class="card-box">
        <div class="toolbar">
          <el-input v-model="routeSearch" placeholder="搜索线路（站点名称）" clearable style="width: 240px" />
          <el-button @click="openBatchDialog">批量生成</el-button>
        </div>
      </div>
      <div class="card-box" v-loading="loading">
        <div class="route-grid" v-if="filteredRoutePairs.length > 0">
          <div v-for="pair in filteredRoutePairs" :key="pair.key" class="route-card" @click="enterRoute(pair)">
            <div class="route-card-header">
              <span class="route-card-name">{{ pair.stationA?.name }} ↔ {{ pair.stationB?.name }}</span>
            </div>
            <div class="route-card-body">
              <div class="route-direction-row" v-if="pair.forward">
                <span class="dir-label">{{ pair.forward.from_station?.name }} → {{ pair.forward.to_station?.name }}</span>
                <el-tag size="small" :type="routeTypeTag(pair.forward.route_type).type">{{ routeTypeTag(pair.forward.route_type).label }}</el-tag>
              </div>
              <div class="route-direction-row" v-if="pair.reverse">
                <span class="dir-label">{{ pair.reverse.from_station?.name }} → {{ pair.reverse.to_station?.name }}</span>
                <el-tag size="small" :type="routeTypeTag(pair.reverse.route_type).type">{{ routeTypeTag(pair.reverse.route_type).label }}</el-tag>
              </div>
            </div>
          </div>
        </div>
        <el-empty v-else description="暂无线路，请先在「线路管理」中创建" />
      </div>
    </template>

    <!-- 班次详情 -->
    <template v-else>
      <div class="card-box">
        <div class="toolbar">
          <el-button @click="backToRoutes">← 返回线路</el-button>
          <el-date-picker v-model="query.trip_date" type="date" value-format="YYYY-MM-DD" placeholder="发车日期" style="width: 160px" clearable />
          <el-select v-model="query.status" placeholder="状态" clearable style="width: 100px">
            <el-option label="可售" :value="1" /><el-option label="已发车" :value="2" /><el-option label="已取消" :value="3" /><el-option label="已完成" :value="4" /><el-option label="下架" :value="0" />
          </el-select>
          <el-button type="primary" @click="handleSearch">查询</el-button>
          <el-button type="primary" @click="handleAdd">新增班次</el-button>
          <el-button @click="openBatchDialog">批量生成</el-button>
          <el-button type="danger" plain :disabled="selectedTrips.length === 0" @click="handleBatchDelete">批量删除({{ selectedTrips.length }})</el-button>
          <el-button type="warning" plain @click="openCleanupDialog">清理历史</el-button>
        </div>
      </div>
      <!-- 方向标签：正反两个方向 -->
      <div class="card-box" v-if="selectedPair && selectedPair.forward && selectedPair.reverse">
        <el-tabs v-model="activeDirection" @tab-change="onDirectionChange">
          <el-tab-pane name="forward">
            <template #label>{{ selectedPair.forward.from_station?.name }} → {{ selectedPair.forward.to_station?.name }}</template>
          </el-tab-pane>
          <el-tab-pane name="reverse">
            <template #label>{{ selectedPair.reverse.from_station?.name }} → {{ selectedPair.reverse.to_station?.name }}</template>
          </el-tab-pane>
        </el-tabs>
      </div>
      <div class="card-box">
        <el-table :data="list" v-loading="loading || batchDeleteLoading" stripe @selection-change="handleSelectionChange">
          <el-table-column type="selection" width="44" />
          <el-table-column prop="trip_no" label="班次号" width="160" />
          <el-table-column label="出发时间" width="170">
            <template #default="{ row }">
              <div class="time-cell">
                <div class="time-date">{{ row.trip_date }}</div>
                <div class="time-clock">{{ row.departure_time?.substring(0, 5) }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="到达时间" width="170">
            <template #default="{ row }">
              <div class="time-cell">
                <div class="time-date">{{ formatArrivalDate(row) }}</div>
                <div class="time-clock">{{ row.arrival_time?.substring(0, 5) }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="票价" width="80"><template #default="{ row }">¥{{ row.base_price }}</template></el-table-column>
          <el-table-column width="110"><template #header><span>可售/总座</span><el-tooltip content="可售座数是管理员设定的可销售上限，并非当前实时余票。实际可售座位由区间复用模型动态计算。" placement="top"><el-icon style="margin-left:4px;cursor:help"><QuestionFilled /></el-icon></el-tooltip></template><template #default="{ row }">{{ row.available_seats }}/{{ row.total_seats }}</template></el-table-column>
          <el-table-column label="司机" width="120">
            <template #default="{ row }">
              <el-select v-model="row.driver_id" size="small" placeholder="分配" clearable style="width: 110px" @focus="(val: any) => { row._oldDriverId = row.driver_id }" @change="(val: any) => handleAssignDriver(row, val)">
                <el-option v-for="d in drivers" :key="d.id" :label="d.employee_no ? `${d.name}(${d.employee_no})` : d.name" :value="d.id" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="80">
            <template #default="{ row }"><el-tag :type="tripStatusType(row.status)" size="small">{{ tripStatusText(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="handleShowPassengers(row)">乘客</el-button>
              <el-button size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination"><el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" :total="total" layout="total, prev, pager, next" @current-change="loadData" /></div>
      </div>
    </template>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑班次' : '新增班次'" width="520px">
      <el-form :model="editing" label-width="90px">
        <el-form-item label="线路" required>
          <el-select v-model="editing.route_id" placeholder="选择线路" filterable>
            <el-option v-for="r in routes" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="车辆" required>
          <el-select v-model="editing.vehicle_id" placeholder="选择车辆" filterable>
            <el-option v-for="v in vehicles" :key="v.id" :label="`${v.plate_no} (${v.vehicle_type})`" :value="v.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="发车日期" required><el-date-picker v-model="editing.trip_date" type="date" value-format="YYYY-MM-DD" @change="onTripDateChange" /></el-form-item>
        <el-form-item label="发车时间" required><el-time-select v-model="editing.departure_time" start="00:00" step="00:05" end="23:55" placeholder="选择时间" /></el-form-item>
        <el-form-item label="到达日期" required><el-date-picker v-model="editing.arrival_date" type="date" value-format="YYYY-MM-DD" placeholder="选择到达日期" style="width: 100%" /></el-form-item>
        <el-form-item label="到达时间" required><el-time-select v-model="editing.arrival_time" start="00:00" step="00:05" end="23:55" placeholder="选择时间" /></el-form-item>
        <el-form-item label="总座位数"><el-input-number v-model="editing.total_seats" :min="0" /><span style="margin-left:8px;color:#909399;font-size:12px">车辆总座位数，留0则自动取车辆座位数</span></el-form-item>
        <el-form-item label="可售座位"><el-input-number v-model="editing.available_seats" :min="0" /><span style="margin-left:8px;color:#909399;font-size:12px">可销售上限，留0则自动等于总座位数</span></el-form-item>
        <el-form-item label="参考票价"><el-input-number v-model="editing.base_price" :min="0" :precision="2" /><span style="margin-left:8px;color:#909399;font-size:12px">全程参考价，实际按站点区间价表算</span></el-form-item>
        <el-form-item label="开放无座票">
          <el-switch v-model="editing.allow_standing" active-text="开启" inactive-text="关闭" />
          <span style="margin-left:8px;color:#909399;font-size:12px">开启后余座不足时乘客可购买无座站票，座位不占用</span>
        </el-form-item>
        <el-form-item v-if="editing.allow_standing" label="无座配额">
          <el-input-number v-model="editing.standing_quota" :min="0" :precision="0" />
          <span style="margin-left:8px;color:#909399;font-size:12px">无座票可售上限（≤总座位数）</span>
        </el-form-item>
        <el-form-item v-if="editing.allow_standing" label="无座折扣">
          <el-input-number v-model="editing.standing_discount" :min="0" :max="1" :step="0.05" :precision="2" />
          <span style="margin-left:8px;color:#909399;font-size:12px">无座票价折扣，1=与座位同价（默认），0.9 表示 9 折，填 0 也与座位同价不强制打折</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="editing.status"><el-option label="可售" :value="1" /><el-option label="已发车" :value="2" /><el-option label="已取消" :value="3" /><el-option label="已完成" :value="4" /><el-option label="下架" :value="0" /></el-select>
        </el-form-item>
        <el-form-item label="到站标记">
          <el-input-number v-model="editing.current_passed_order" :min="0" :step="1" />
          <span style="margin-left:8px;color:#909399;font-size:12px">手动标记车已驶过到第几站序(0=未标记)，>0时覆盖时刻表/推算，用于发车后按站控制能否下单</span>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleSave">保存</el-button></template>
    </el-dialog>

    <!-- 批量生成对话框 -->
    <el-dialog v-model="batchDialogVisible" title="批量生成班次" width="780px">
      <el-form :model="batch" label-width="100px">
        <el-form-item label="线路" required>
          <el-select v-model="batch.route_id" filterable placeholder="选择线路" style="width: 100%">
            <el-option v-for="r in routes" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="车辆" required>
          <el-select v-model="batch.vehicle_id" filterable placeholder="选择车辆" style="width: 100%">
            <el-option v-for="v in vehicles" :key="v.id" :label="`${v.plate_no} (${v.vehicle_type})`" :value="v.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="司机">
          <el-select v-model="batch.driver_id" filterable clearable placeholder="选择司机（可选）" style="width: 100%">
            <el-option v-for="d in drivers" :key="d.id" :label="d.employee_no ? `${d.name}(${d.employee_no})` : d.name" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="参考票价">
          <el-input-number v-model="batch.base_price" :min="0" :precision="2" />
          <span style="margin-left:8px;color:#909399;font-size:12px">全程参考价，实际按站点区间价表算</span>
        </el-form-item>
        <el-form-item label="开放无座票">
          <el-switch v-model="batch.allow_standing" active-text="开启" inactive-text="关闭" />
          <span style="margin-left:8px;color:#909399;font-size:12px">批量生成的班次统一按此无座配置</span>
        </el-form-item>
        <el-form-item v-if="batch.allow_standing" label="无座配额">
          <el-input-number v-model="batch.standing_quota" :min="0" :precision="0" />
          <span style="margin-left:8px;color:#909399;font-size:12px">无座票可售上限（≤车辆座位数）</span>
        </el-form-item>
        <el-form-item v-if="batch.allow_standing" label="无座折扣">
          <el-input-number v-model="batch.standing_discount" :min="0" :max="1" :step="0.05" :precision="2" />
          <span style="margin-left:8px;color:#909399;font-size:12px">无座票价折扣，1=与座位同价（默认），0.9 表示 9 折，填 0 也与座位同价不强制打折</span>
        </el-form-item>
        <el-form-item label="快捷选择">
          <div class="quick-select-bar">
            <el-button size="small" @click="quickSelect('thisMonth')">本月全部</el-button>
            <el-button size="small" @click="quickSelect('thisWorkdays')">本月工作日</el-button>
            <el-button size="small" @click="quickSelect('thisWeekends')">本月周末</el-button>
            <el-divider direction="vertical" />
            <el-button size="small" @click="quickSelect('nextMonth')">下月全部</el-button>
            <el-button size="small" @click="quickSelect('nextWorkdays')">下月工作日</el-button>
            <el-button size="small" @click="quickSelect('nextWeekends')">下月周末</el-button>
            <el-divider direction="vertical" />
            <el-button size="small" type="danger" plain @click="batchDates = []">清空</el-button>
          </div>
        </el-form-item>
        <el-form-item label="日期范围">
          <div class="range-select-bar">
            <el-date-picker v-model="batchRangeStart" type="date" value-format="YYYY-MM-DD" placeholder="起始日期" style="width: 150px" />
            <span style="margin: 0 4px">至</span>
            <el-date-picker v-model="batchRangeEnd" type="date" value-format="YYYY-MM-DD" placeholder="结束日期" style="width: 150px" />
            <span style="margin: 0 8px 0 12px;white-space:nowrap;color:#909399;font-size:12px">排除：</span>
            <el-checkbox-group v-model="batchExcludeWeekdays" size="small">
              <el-checkbox :value="0">日</el-checkbox>
              <el-checkbox :value="1">一</el-checkbox>
              <el-checkbox :value="2">二</el-checkbox>
              <el-checkbox :value="3">三</el-checkbox>
              <el-checkbox :value="4">四</el-checkbox>
              <el-checkbox :value="5">五</el-checkbox>
              <el-checkbox :value="6">六</el-checkbox>
            </el-checkbox-group>
            <el-button size="small" type="primary" @click="applyDateRange">生成日期</el-button>
          </div>
        </el-form-item>
        <el-form-item label="手动微调" required>
          <el-date-picker v-model="batchDates" type="dates" value-format="YYYY-MM-DD" :disabled-date="disablePastDates" placeholder="可在此手动增减日期（过去日期不可选）" style="width: 100%" />
        </el-form-item>
        <el-form-item label="默认时间">
          <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
            <el-time-select v-model="batchDefaultDep" start="00:00" step="00:05" end="23:55" placeholder="发车" style="width: 120px" />
            <span>→</span>
            <el-time-select v-model="batchDefaultArr" start="00:00" step="00:05" end="23:55" placeholder="到达" style="width: 120px" />
            <el-input-number v-model="batchDefaultOffset" :min="0" :max="7" style="width: 80px" />
            <span style="color:#909399;font-size:12px">天后到达（0=当天,1=次日）</span>
            <el-button size="small" @click="applyDefaultTime" :disabled="batchDateItems.length === 0">应用到全部</el-button>
            <el-button size="small" type="warning" @click="applyDefaultTimeToSelected" :disabled="batchSelectedItems.length === 0">应用到选中({{ batchSelectedItems.length }})</el-button>
          </div>
        </el-form-item>
        <el-form-item label="日期列表" v-if="batchDateItems.length > 0">
          <el-alert type="info" :closable="false" show-icon style="margin-bottom: 8px">
            共 {{ batchDateItems.length }} 个日期。可勾选特定日期（如节假日、周末）单独设置不同时间，也可全选后统一修改。红色行为周末。
          </el-alert>
          <div style="margin-bottom:8px;display:flex;gap:6px;flex-wrap:wrap">
            <el-button size="small" @click="selectBatchRows(true, true)">选周末</el-button>
            <el-button size="small" @click="selectBatchRows(true, false)">选工作日</el-button>
            <el-button size="small" @click="selectAllBatchRows(true)">全选</el-button>
            <el-button size="small" @click="selectAllBatchRows(false)">取消选择</el-button>
          </div>
          <el-table ref="batchTableRef" :data="batchDateItems" row-key="date" max-height="280" size="small" border :row-class-name="batchRowClassName" @selection-change="handleBatchSelectionChange">
            <el-table-column type="selection" width="40" />
            <el-table-column label="日期" width="150">
              <template #default="{ row }">
                <span>{{ row.date }}</span>
                <el-tag size="small" :type="isWeekend(row.date) ? 'danger' : 'info'" style="margin-left:4px">{{ weekdayName(row.date) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="发车时间" width="150">
              <template #default="{ row }">
                <el-time-select v-model="row.departure_time" start="00:00" step="00:05" end="23:55" style="width: 120px" />
              </template>
            </el-table-column>
            <el-table-column label="到达时间" width="150">
              <template #default="{ row }">
                <el-time-select v-model="row.arrival_time" start="00:00" step="00:05" end="23:55" style="width: 120px" />
              </template>
            </el-table-column>
            <el-table-column label="到达日期" width="150">
              <template #default="{ row }">
                <el-date-picker v-model="row.arrival_date" type="date" value-format="YYYY-MM-DD" size="small" style="width: 130px" :disabled="!row.date" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="60">
              <template #default="{ $index }">
                <el-button size="small" link type="danger" @click="removeBatchDate($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="batchDialogVisible = false">取消</el-button><el-button type="primary" @click="handleBatchCreate">批量生成</el-button></template>
    </el-dialog>

    <!-- 乘客名单对话框 -->
    <el-dialog v-model="passengerDialogVisible" :title="`乘客名单 - ${currentTrip?.trip_no || ''}`" width="700px">
      <el-table :data="passengerList" stripe max-height="400">
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column label="上车" width="100">
          <template #default="{ row }">{{ row.order?.from_station_name || row.order?.from_station?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="下车" width="100">
          <template #default="{ row }">{{ row.order?.to_station_name || row.order?.to_station?.name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="id_card_no" label="身份证号" width="180" />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column prop="seat_no" label="座位号" width="80" />
        <el-table-column label="核销状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.check_status === 1 ? 'success' : 'info'" size="small">
              {{ row.check_status === 1 ? '已核销' : '未核销' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="checked_at" label="核销时间" width="170">
          <template #default="{ row }">{{ row.checked_at ? row.checked_at.replace('T',' ').substring(0,19) : '-' }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 清理历史班次对话框 -->
    <el-dialog v-model="cleanupDialogVisible" title="清理历史班次" width="560px">
      <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 16px">
        将删除指定日期之前的所有历史班次（不含当天）。已发车未结束的班次不会被清理。
        <br />关联订单的路线、站点、日期等信息会保留，活跃订单将被取消，已支付订单自动退款。
      </el-alert>
      <el-form label-width="100px">
        <el-form-item label="截止日期" required>
          <el-date-picker v-model="cleanupBeforeDate" type="date" value-format="YYYY-MM-DD" placeholder="选择截止日期" style="width: 200px" />
          <span style="margin-left:8px;color:#909399;font-size:12px">删除此日期之前的班次</span>
        </el-form-item>
        <el-form-item label="清理范围">
          <el-radio-group v-model="cleanupScope">
            <el-radio :value="'all'">全部线路</el-radio>
            <el-radio :value="'current'" :disabled="!currentRouteId">仅当前线路</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div v-if="cleanupPreview" style="margin-top: 12px">
        <el-alert :type="cleanupPreview.has_active ? 'error' : 'success'" :closable="false" show-icon>
          <div style="line-height: 1.8">
            <strong>预览结果（截止 {{ cleanupPreview.before_date }}）</strong><br />
            待清理班次：<strong>{{ cleanupPreview.trip_count }}</strong> 个<br />
            状态分布：下架 {{ cleanupPreview.status_breakdown?.offline || 0 }}、可售 {{ cleanupPreview.status_breakdown?.available || 0 }}、已取消 {{ cleanupPreview.status_breakdown?.cancelled || 0 }}、已完成 {{ cleanupPreview.status_breakdown?.completed || 0 }}
            <span v-if="cleanupPreview.status_breakdown?.departed > 0">
              <br />未清理（已发车未结束）：{{ cleanupPreview.status_breakdown?.departed }} 个，需先手动结束行程
            </span>
            <template v-if="cleanupPreview.total_orders > 0">
              <br />关联订单：共 {{ cleanupPreview.total_orders }} 条
              <span v-if="cleanupPreview.pending_orders > 0">，待支付 {{ cleanupPreview.pending_orders }}</span>
              <span v-if="cleanupPreview.paid_orders > 0">，已支付 {{ cleanupPreview.paid_orders }}（将自动退款）</span>
              <span v-if="cleanupPreview.history_orders > 0">，已完成/取消 {{ cleanupPreview.history_orders }}</span>
            </template>
            <div v-if="cleanupPreview.has_active" style="margin-top: 4px; color: #f56c6c">
              ⚠ 存在活跃订单，删除后将自动取消并退款已支付订单！
            </div>
          </div>
        </el-alert>
      </div>
      <template #footer>
        <el-button @click="cleanupDialogVisible = false">取消</el-button>
        <el-button @click="previewCleanup" :loading="cleanupLoading">预览</el-button>
        <el-button type="danger" @click="executeCleanup" :disabled="!cleanupPreview || cleanupPreview.trip_count === 0" :loading="cleanupExecuting">确认清理</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { QuestionFilled } from '@element-plus/icons-vue'
import { tripApi, routeApi, vehicleApi, driverApi } from '@/api'

// 状态
const list = ref<any[]>([]), total = ref(0), loading = ref(false), dialogVisible = ref(false), batchDialogVisible = ref(false)
const routes = ref<any[]>([]), vehicles = ref<any[]>([]), drivers = ref<any[]>([])
const passengerDialogVisible = ref(false), passengerList = ref<any[]>([]), currentTrip = ref<any>(null)
const query = reactive({ trip_date: '', route_id: '' as any, status: '' as any, page: 1, page_size: 20 })
const editing = reactive({ id: 0, route_id: 0, vehicle_id: 0, trip_date: '', departure_time: '', arrival_time: '', arrival_date: '', arrival_day_offset: 0, base_price: 0, total_seats: 0, available_seats: 0, status: 1, current_passed_order: 0, allow_standing: false, standing_quota: 0, standing_discount: 1 })
const batch = reactive({ route_id: 0, vehicle_id: 0, driver_id: 0, base_price: 0, allow_standing: false, standing_quota: 0, standing_discount: 1 })
const batchDates = ref<string[]>([])
const batchDefaultDep = ref('')
const batchDefaultArr = ref('')
const batchDefaultOffset = ref(0)
const batchDateItems = ref<{ date: string, departure_time: string, arrival_time: string, arrival_date: string }[]>([])
const batchRangeStart = ref('')
const batchRangeEnd = ref('')
const batchExcludeWeekdays = ref<number[]>([])
const batchSelectedItems = ref<any[]>([])
const batchTableRef = ref()

// 视图模式：'routes' = 线路总览，'trips' = 班次详情
const viewMode = ref<'routes' | 'trips'>('routes')
const routeSearch = ref('')
const selectedPair = ref<any>(null)
const activeDirection = ref<'forward' | 'reverse'>('forward')

// 清理历史班次
const cleanupDialogVisible = ref(false)
const cleanupBeforeDate = ref('')
const cleanupScope = ref<'all' | 'current'>('all')
const cleanupPreview = ref<any>(null)
const cleanupLoading = ref(false)
const cleanupExecuting = ref(false)

// 批量选择
const selectedTrips = ref<any[]>([])
const batchDeleteLoading = ref(false)

// 线路配对逻辑
// 将正反方向的线路分组成一对（如"上海→商丘" + "商丘→上海" = 一对）
const routePairs = computed(() => {
  const pairs = new Map<string, any>()
  routes.value.forEach(r => {
    const ids = [r.from_station_id, r.to_station_id].sort((a: number, b: number) => a - b)
    const baseKey = ids.join('-')
    if (!pairs.has(baseKey)) {
      pairs.set(baseKey, {
        key: baseKey,
        stationA: r.from_station_id === ids[0] ? r.from_station : r.to_station,
        stationB: r.from_station_id === ids[0] ? r.to_station : r.from_station,
        forward: null as any,
        reverse: null as any,
      })
    }
    const pair = pairs.get(baseKey)!
    if (r.from_station_id === ids[0]) {
      if (!pair.forward) {
        pair.forward = r
      } else {
        // 相同起终站的多条正向线路：拆分为独立卡片，避免静默覆盖
        const dupKey = `${baseKey}-f${r.id}`
        pairs.set(dupKey, { key: dupKey, stationA: pair.stationA, stationB: pair.stationB, forward: r, reverse: null as any })
      }
    } else {
      if (!pair.reverse) {
        pair.reverse = r
      } else {
        const dupKey = `${baseKey}-r${r.id}`
        pairs.set(dupKey, { key: dupKey, stationA: pair.stationA, stationB: pair.stationB, forward: null as any, reverse: r })
      }
    }
  })
  return Array.from(pairs.values())
})

const filteredRoutePairs = computed(() => {
  if (!routeSearch.value) return routePairs.value
  const search = routeSearch.value.toLowerCase()
  return routePairs.value.filter(p => {
    const nameA = (p.stationA?.name || '').toLowerCase()
    const nameB = (p.stationB?.name || '').toLowerCase()
    return nameA.includes(search) || nameB.includes(search)
  })
})

const routeTypeTag = (t: number) => {
  const map: Record<number, { label: string; type: string }> = {
    1: { label: '城乡公交', type: 'success' },
    2: { label: '城际客运', type: 'primary' },
    3: { label: '旅游专线', type: 'warning' },
  }
  return map[t] || map[1]
}

// 当前选中方向的线路ID
const currentRouteId = computed(() => {
  if (!selectedPair.value) return 0
  const route = activeDirection.value === 'forward' ? selectedPair.value.forward : selectedPair.value.reverse
  return route?.id || 0
})

// 进入某条线路的班次详情
const enterRoute = (pair: any) => {
  selectedPair.value = pair
  viewMode.value = 'trips'
  activeDirection.value = pair.forward ? 'forward' : 'reverse'
  query.route_id = currentRouteId.value
  query.page = 1
  loadData()
}

const backToRoutes = () => {
  viewMode.value = 'routes'
  selectedPair.value = null
  list.value = []
  query.trip_date = ''
  query.status = ''
  query.route_id = ''
}

const onDirectionChange = () => {
  query.route_id = currentRouteId.value
  query.page = 1
  loadData()
}

// 日期辅助函数
const formatDateLocal = (d: Date) => {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

// 表格到达日期：trip_date + offset
const formatArrivalDate = (row: any) => {
  const offset = row.arrival_day_offset || 0
  const d = new Date(row.trip_date + 'T00:00:00')
  d.setDate(d.getDate() + offset)
  return formatDateLocal(d)
}

// 发车日期变更时，自动将到达日期设为同一天
const onTripDateChange = () => {
  if (!editing.arrival_date || editing.arrival_date < editing.trip_date) {
    editing.arrival_date = editing.trip_date
  }
}

const weekdayNames = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const weekdayName = (dateStr: string) => {
  const d = new Date(dateStr + 'T00:00:00')
  return weekdayNames[d.getDay()]
}
const isWeekend = (dateStr: string) => {
  const d = new Date(dateStr + 'T00:00:00')
  const wd = d.getDay()
  return wd === 0 || wd === 6
}

const batchRowClassName = ({ row }: { row: any }) => {
  return isWeekend(row.date) ? 'weekend-row' : ''
}
const handleBatchSelectionChange = (selection: any[]) => {
  batchSelectedItems.value = selection
}
const selectAllBatchRows = (select: boolean) => {
  nextTick(() => {
    if (select) {
      batchDateItems.value.forEach(row => batchTableRef.value?.toggleRowSelection(row, true))
    } else {
      batchTableRef.value?.clearSelection()
    }
  })
}
const selectBatchRows = (select: boolean, weekend: boolean) => {
  nextTick(() => {
    batchTableRef.value?.clearSelection()
    batchDateItems.value.forEach(row => {
      if (isWeekend(row.date) === weekend) {
        batchTableRef.value?.toggleRowSelection(row, true)
      }
    })
  })
}

const applyDefaultTimeToSelected = () => {
  if (!batchDefaultDep.value || !batchDefaultArr.value) {
    ElMessage.warning('请先设置默认发车和到达时间')
    return
  }
  const offset = batchDefaultOffset.value || 0
  batchSelectedItems.value.forEach(item => {
    item.departure_time = batchDefaultDep.value
    item.arrival_time = batchDefaultArr.value
    const d = new Date(item.date + 'T00:00:00')
    d.setDate(d.getDate() + offset)
    item.arrival_date = formatDateLocal(d)
  })
  ElMessage.success(`已应用到 ${batchSelectedItems.value.length} 个选中日期`)
}

// 禁用过去日期（批量生成日期选择器）
const disablePastDates = (date: Date) => {
  const t = new Date()
  t.setHours(0, 0, 0, 0)
  return date.getTime() < t.getTime()
}

const quickSelect = (mode: string) => {
  const now = new Date()
  let year = now.getFullYear()
  let month = now.getMonth()
  if (mode.startsWith('next')) {
    month++
    if (month > 11) { month = 0; year++ }
  }
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  const dates: string[] = []
  const excludeSet = new Set(batchExcludeWeekdays.value)
  const todayStr = formatDateLocal(new Date())
  let pastCount = 0
  for (let d = new Date(firstDay); d <= lastDay; d.setDate(d.getDate() + 1)) {
    const ds = formatDateLocal(d)
    if (ds < todayStr) { pastCount++; continue } // 跳过已过去的日期，避免生成历史班次
    const wd = d.getDay()
    if (mode.includes('Workdays') && (wd === 0 || wd === 6)) continue
    if (mode.includes('Weekends') && wd !== 0 && wd !== 6) continue
    if (excludeSet.has(wd)) continue
    dates.push(ds)
  }
  const existing = new Set(batchDates.value)
  dates.forEach(d => existing.add(d))
  batchDates.value = Array.from(existing).sort((a, b) => a.localeCompare(b))
  ElMessage.success(`已添加 ${dates.length} 个日期${pastCount > 0 ? `（跳过 ${pastCount} 个过去日期）` : ''}（共 ${batchDates.value.length} 个）`)
  if (batchDates.value.length > 0 && !batchDefaultDep.value) {
    ElNotification({ title: '提示', message: '请在下方设置默认发车/到达时间，点击"应用到全部"统一设置。', type: 'info', duration: 6000 })
  }
}

const applyDateRange = () => {
  if (!batchRangeStart.value || !batchRangeEnd.value) {
    ElMessage.warning('请选择起始和结束日期')
    return
  }
  if (batchRangeStart.value > batchRangeEnd.value) {
    ElMessage.warning('起始日期不能晚于结束日期')
    return
  }
  const start = new Date(batchRangeStart.value)
  const end = new Date(batchRangeEnd.value)
  const excludeSet = new Set(batchExcludeWeekdays.value)
  const dates: string[] = []
  const todayStr = formatDateLocal(new Date())
  let pastCount = 0
  for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
    const ds = formatDateLocal(d)
    if (ds < todayStr) { pastCount++; continue } // 跳过已过去的日期
    if (excludeSet.has(d.getDay())) continue
    dates.push(ds)
  }
  if (dates.length === 0) { ElMessage.warning('该范围内没有可用日期（可能全部为过去日期或已被排除）'); return }
  if (dates.length > 90) { ElMessage.warning('日期数量不能超过90天，请缩小范围'); return }
  const existing = new Set(batchDates.value)
  dates.forEach(d => existing.add(d))
  batchDates.value = Array.from(existing).sort((a, b) => a.localeCompare(b))
  ElMessage.success(`已添加 ${dates.length} 个日期${pastCount > 0 ? `（跳过 ${pastCount} 个过去日期）` : ''}（共 ${batchDates.value.length} 个）`)
}

const applyDefaultTime = () => {
  if (!batchDefaultDep.value || !batchDefaultArr.value) {
    ElMessage.warning('请先设置默认发车和到达时间')
    return
  }
  const offset = batchDefaultOffset.value || 0
  batchDateItems.value.forEach(item => {
    item.departure_time = batchDefaultDep.value
    item.arrival_time = batchDefaultArr.value
    const d = new Date(item.date + 'T00:00:00')
    d.setDate(d.getDate() + offset)
    item.arrival_date = formatDateLocal(d)
  })
}

const removeBatchDate = (index: number) => {
  const removed = batchDateItems.value[index]
  batchDateItems.value.splice(index, 1)
  batchDates.value = batchDates.value.filter(d => d !== removed.date)
}

// 多选日期同步到列表
watch(batchDates, (newDates) => {
  const existingMap = new Map(batchDateItems.value.map(d => [d.date, d]))
  batchDateItems.value = newDates
    .slice()
    .sort((a, b) => a.localeCompare(b))
    .map(d => existingMap.get(d) || { date: d, departure_time: batchDefaultDep.value, arrival_time: batchDefaultArr.value, arrival_date: d })
})

const openBatchDialog = () => {
  Object.assign(batch, { route_id: viewMode.value === 'trips' ? currentRouteId.value : 0, vehicle_id: 0, driver_id: 0, base_price: 0, allow_standing: false, standing_quota: 0, standing_discount: 1 })
  batchDates.value = []
  batchDefaultDep.value = ''
  batchDefaultArr.value = ''
  batchDefaultOffset.value = 0
  batchDateItems.value = []
  batchRangeStart.value = ''
  batchRangeEnd.value = ''
  batchExcludeWeekdays.value = []
  batchSelectedItems.value = []
  batchDialogVisible.value = true
}

const tripStatusText = (s: number) => ({ 1: '可售', 2: '已发车', 3: '已取消', 4: '已完成', 0: '下架' } as any)[s] || '未知'
const tripStatusType = (s: number) => ({ 1: 'success', 2: 'info', 3: 'danger', 4: 'success', 0: 'warning' } as any)[s] || 'info'

const handleSearch = () => { query.page = 1; loadData() }
const loadData = async () => {
  loading.value = true
  try {
    const res: any = await tripApi.list(query)
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}
const loadDeps = async () => {
  const r: any = await routeApi.all()
  routes.value = r.data || []
  const v: any = await vehicleApi.all()
  vehicles.value = v.data || []
  const d: any = await driverApi.all()
  drivers.value = d.data || []
}

const handleAdd = () => {
  Object.assign(editing, {
    id: 0, route_id: currentRouteId.value || 0, vehicle_id: 0,
    trip_date: query.trip_date || '', departure_time: '', arrival_time: '',
    arrival_date: query.trip_date || '', arrival_day_offset: 0,
    base_price: 0, total_seats: 0, available_seats: 0, status: 1, current_passed_order: 0,
    allow_standing: false, standing_quota: 0, standing_discount: 1
  })
  dialogVisible.value = true
}
const handleEdit = (row: any) => {
  const offset = row.arrival_day_offset || 0
  const arrDate = new Date(row.trip_date + 'T00:00:00')
  arrDate.setDate(arrDate.getDate() + offset)
  Object.assign(editing, {
    id: row.id, route_id: row.route_id, vehicle_id: row.vehicle_id,
    trip_date: row.trip_date,
    departure_time: row.departure_time ? row.departure_time.substring(0, 5) : '',
    arrival_time: row.arrival_time ? row.arrival_time.substring(0, 5) : '',
    arrival_date: formatDateLocal(arrDate),
    arrival_day_offset: offset,
    base_price: row.base_price, total_seats: row.total_seats,
    available_seats: row.available_seats, status: row.status,
    current_passed_order: row.current_passed_order || 0,
    allow_standing: row.allow_standing === true,
    standing_quota: row.standing_quota || 0,
    standing_discount: row.standing_discount || 1,
  });
  dialogVisible.value = true
}
const padTime = (t: string) => t || ''
const handleSave = async () => {
  if (!editing.route_id || !editing.trip_date) {
    ElMessage.warning('请选择线路和发车日期')
    return
  }
  if (!editing.departure_time || !editing.arrival_time) {
    ElMessage.warning('请选择发车时间和到达时间')
    return
  }
  if (!editing.arrival_date) {
    ElMessage.warning('请选择到达日期')
    return
  }
  if (editing.arrival_date < editing.trip_date) {
    ElMessage.warning('到达日期不能早于发车日期')
    return
  }
  const offset = Math.round((new Date(editing.arrival_date).getTime() - new Date(editing.trip_date).getTime()) / 86400000)
  if (offset === 0 && editing.departure_time >= editing.arrival_time) {
    ElMessage.warning('当天到达时间必须晚于发车时间，如需次日到达请选择到达日期为次日')
    return
  }
  // 无座票配置校验（与后端一致）
  if (editing.allow_standing) {
    if (!editing.standing_quota || editing.standing_quota <= 0) {
      ElMessage.warning('开放无座票需填写无座配额（>0）')
      return
    }
    const seats = editing.total_seats > 0 ? editing.total_seats : 0
    if (seats > 0 && editing.standing_quota > seats) {
      ElMessage.warning('无座配额不能超过总座位数')
      return
    }
    // 无座折扣：填0也中，跟有座一个价，不硬叫人家打折（后端自动按1存）；
    // 想打九折就填0.9，0到1中间随便填，超过1那可不行
    if (editing.standing_discount < 0 || editing.standing_discount > 1) {
      ElMessage.warning('无座票价折扣需在0~1之间（0=与座位同价不强制打折）')
      return
    }
  }
  const payload = {
    ...editing,
    arrival_day_offset: offset,
    departure_time: padTime(editing.departure_time),
    arrival_time: padTime(editing.arrival_time)
  }
  try {
    if (editing.id) {
      await tripApi.update(editing.id, payload)
    } else {
      await tripApi.create(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    loadData()
  } catch (e) { /* error handled by interceptor */ }
}
const handleSelectionChange = (selection: any[]) => {
  selectedTrips.value = selection
}
const handleBatchDelete = async () => {
  if (selectedTrips.value.length === 0) return
  const count = selectedTrips.value.length
  try {
    await ElMessageBox.confirm(
      `确认批量删除选中的 ${count} 个班次？\n\n如果班次存在关联订单，将自动取消活跃订单并退款已支付订单，然后物理删除班次。此操作不可撤销！`,
      '批量删除确认',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' }
    )
  } catch { return }
  batchDeleteLoading.value = true
  const results = { success: 0, fail: 0, errors: [] as string[] }
  for (const trip of selectedTrips.value) {
    try {
      await tripApi.delete(trip.id, true)
      results.success++
    } catch (e: any) {
      results.fail++
      const errMsg = e?.response?.data?.message || e?.message || '未知错误'
      results.errors.push(`${trip.trip_no}: ${errMsg}`)
    }
  }
  batchDeleteLoading.value = false
  selectedTrips.value = []
  if (results.fail === 0) {
    ElMessage.success(`成功删除 ${results.success} 个班次`)
  } else {
    ElNotification({
      title: '批量删除完成',
      message: `成功 ${results.success} 个，失败 ${results.fail} 个\n${results.errors.join('\n')}`,
      type: 'warning',
      duration: 10000,
    })
  }
  loadData()
}
const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确认删除该班次？', '提示', { type: 'warning' })
  } catch { return }
  try {
    await tripApi.delete(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e: any) {
    const detail = e?.response?.data?.data
    const total = Number(detail?.total_count ?? 0)
    if (total === 0) return
    const pending = Number(detail?.pending_count ?? 0)
    const paid = Number(detail?.paid_count ?? 0)
    const history = Number(detail?.history_count ?? 0)
    const lines: string[] = []
    if (pending > 0) lines.push(`待支付 ${pending} 条`)
    if (paid > 0) lines.push(`已支付 ${paid} 条`)
    if (history > 0) lines.push(`已完成/取消 ${history} 条`)
    const msg = `该班次存在关联订单（共 ${total} 条：${lines.join('、')}）。\n\n强制删除将${pending > 0 || paid > 0 ? '取消所有活跃订单' : ''}${paid > 0 ? '并对已支付订单自动发起退款，' : ''}然后物理删除该班次。订单记录（路线、站点、日期等）会保留。\n\n是否继续？`
    try {
      await ElMessageBox.confirm(msg, '强制删除确认', { type: 'warning', confirmButtonText: '强制删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' })
    } catch { return }
    try {
      await tripApi.delete(row.id, true)
      ElMessage.success('删除成功')
      loadData()
    } catch { /* error handled by interceptor */ }
  }
}
const handleBatchCreate = async () => {
  if (!batch.route_id || !batch.vehicle_id) {
    ElMessage.warning('请选择线路和车辆')
    return
  }
  if (batchDateItems.value.length === 0) {
    ElMessage.warning('请选择至少一个日期')
    return
  }
  for (const item of batchDateItems.value) {
    if (!item.departure_time || !item.arrival_time) {
      ElMessage.warning(`日期 ${item.date} 未设置时间`)
      return
    }
    if (!item.arrival_date) {
      ElMessage.warning(`日期 ${item.date} 未设置到达日期`)
      return
    }
    if (item.arrival_date < item.date) {
      ElMessage.warning(`日期 ${item.date} 的到达日期不能早于发车日期`)
      return
    }
    const offset = Math.round((new Date(item.arrival_date).getTime() - new Date(item.date).getTime()) / 86400000)
    if (offset === 0 && item.departure_time >= item.arrival_time) {
      ElMessage.warning(`日期 ${item.date} 的到达时间必须晚于发车时间，如需次日到达请选择到达日期为次日`)
      return
    }
  }
  // TODO: 这套日期+时间校验和新增班次里重复了，后面抽个公共函数统一处理
  // 无座票配置校验（与后端一致）
  if (batch.allow_standing) {
    if (!batch.standing_quota || batch.standing_quota <= 0) {
      ElMessage.warning('开放无座票需填写无座配额（>0）')
      return
    }
    // 无座折扣：填0也中，跟有座一个价，不硬叫人家打折（后端自动按1存）；
    // 想打九折就填0.9，0到1中间随便填，超过1那可不行
    if (batch.standing_discount < 0 || batch.standing_discount > 1) {
      ElMessage.warning('无座票价折扣需在0~1之间（0=与座位同价不强制打折）')
      return
    }
  }
  const payload = {
    route_id: batch.route_id,
    vehicle_id: batch.vehicle_id,
    driver_id: batch.driver_id || 0,
    base_price: batch.base_price,
    allow_standing: batch.allow_standing,
    standing_quota: batch.allow_standing ? batch.standing_quota : 0,
    standing_discount: batch.allow_standing ? batch.standing_discount : 1,
    trip_dates: batchDateItems.value.map(d => {
      const offset = Math.round((new Date(d.arrival_date).getTime() - new Date(d.date).getTime()) / 86400000)
      return {
        date: d.date,
        departure_time: padTime(d.departure_time),
        arrival_time: padTime(d.arrival_time),
        arrival_day_offset: offset
      }
    })
  }
  try {
    const res: any = await tripApi.batch(payload)
    ElMessage.success(res.message || '批量生成成功')
    batchDialogVisible.value = false
    if (viewMode.value === 'trips') {
      loadData()
    } else {
      // 从线路总览页批量生成后，自动跳转到对应线路的班次详情
      const pair = routePairs.value.find(p => p.forward?.id === batch.route_id || p.reverse?.id === batch.route_id)
      if (pair) {
        selectedPair.value = pair
        viewMode.value = 'trips'
        activeDirection.value = pair.forward?.id === batch.route_id ? 'forward' : 'reverse'
        query.route_id = batch.route_id
        query.trip_date = ''
        query.status = ''
        query.page = 1
        loadData()
      }
    }
  } catch (e) { /* error handled by interceptor */ }
}
const handleAssignDriver = async (row: any, driverId: number) => {
  // 先存旧值，接口挂了能回滚，省得界面和后端对不上
  const oldDriverId = row._oldDriverId ?? 0
  if (!driverId) {
    try { await tripApi.assignDriver(row.id, { driver_id: 0 }); ElMessage.success('已取消司机分配'); loadData() } catch (e) { row.driver_id = oldDriverId; /* 错误由axios拦截器处理 */ }
    return
  }
  try {
    await tripApi.assignDriver(row.id, { driver_id: driverId })
    ElMessage.success('分配司机成功')
    loadData()
  } catch (e: any) {
    const detail = e?.response?.data?.data
    const conflicts = detail?.conflicts
    if (conflicts && Array.isArray(conflicts) && conflicts.length > 0) {
      const timeConflicts = conflicts.filter((c: any) => c.conflict_type === 'time_overlap')
      const vehicleConflicts = conflicts.filter((c: any) => c.conflict_type === 'vehicle_overlap')
      const softConflicts = conflicts.filter((c: any) => c.conflict_type === 'location_gap')
      const driverName = drivers.value.find((d: any) => d.id === driverId)
      const driverLabel = driverName ? (driverName.employee_no ? `${driverName.name}(${driverName.employee_no})` : driverName.name) : `司机#${driverId}`
      let conflictMsg = ''
      if (timeConflicts.length > 0) {
        conflictMsg += '【司机时间重叠 - 硬冲突】\n'
        timeConflicts.forEach((c: any) => {
          conflictMsg += `  ✶ ${c.route_name} | ${c.trip_no} | ${c.departure_time}~${c.arrival_time} | ${c.status_text}\n`
          conflictMsg += `    → ${c.conflict_desc}\n`
        })
      }
      if (vehicleConflicts.length > 0) {
        conflictMsg += '\n【车辆冲突 - 硬冲突】\n'
        vehicleConflicts.forEach((c: any) => {
          conflictMsg += `  ✶ ${c.route_name} | ${c.trip_no} | ${c.departure_time}~${c.arrival_time} | ${c.status_text}\n`
          conflictMsg += `    → ${c.conflict_desc}\n`
        })
      }
      if (softConflicts.length > 0) {
        conflictMsg += '\n【位置断层 - 软警告】\n'
        softConflicts.forEach((c: any) => {
          conflictMsg += `  ◇ ${c.route_name} | ${c.trip_no} | ${c.departure_time}~${c.arrival_time} | ${c.status_text}\n`
          conflictMsg += `    → ${c.conflict_desc}\n`
        })
      }
      let chainMsg = ''
      const allTrips = [...conflicts].sort((a: any, b: any) => a.departure_time.localeCompare(b.departure_time))
      if (allTrips.length > 0) {
        const stations = allTrips.map((c: any) => `${c.from_station}→${c.to_station}`).join(' | ')
        const newRowRoute = row.route_name || ''
        chainMsg = `\n司机当日行程链：${stations} | 新班次：${newRowRoute}`
      }
      const confirmMsg = `司机「${driverLabel}」在此日期有冲突：\n\n${conflictMsg}${chainMsg}\n\n员工不足时可根据情况决定是否继续分配。是否强制分配？`
      try {
        await ElMessageBox.confirm(confirmMsg, '司机调度冲突提示', { type: 'warning', confirmButtonText: '强制分配', cancelButtonText: '取消', confirmButtonClass: 'el-button--warning' })
      } catch { row.driver_id = oldDriverId; return } // 用户取消时回滚
      try {
        await tripApi.assignDriver(row.id, { driver_id: driverId, force: true })
        ElMessage.success('已强制分配司机')
        loadData()
      } catch (e) { row.driver_id = oldDriverId; /* error handled by interceptor */ }
    } else {
      // 非冲突类错误，回滚 UI
      row.driver_id = oldDriverId
    }
  }
}
const handleShowPassengers = async (row: any) => {
  currentTrip.value = row
  const res: any = await tripApi.passengers(row.id)
  passengerList.value = res.data || []
  passengerDialogVisible.value = true
}
// 清理历史班次
const openCleanupDialog = () => {
  cleanupBeforeDate.value = formatDateLocal(new Date())
  cleanupScope.value = currentRouteId.value ? 'current' : 'all'
  cleanupPreview.value = null
  cleanupDialogVisible.value = true
}
const previewCleanup = async () => {
  if (!cleanupBeforeDate.value) {
    ElMessage.warning('请选择截止日期')
    return
  }
  cleanupLoading.value = true
  try {
    const res: any = await tripApi.cleanup({
      before_date: cleanupBeforeDate.value,
      route_id: cleanupScope.value === 'current' ? currentRouteId.value : 0,
      force: false
    })
    cleanupPreview.value = res.data
    if (res.data?.trip_count === 0) {
      ElMessage.info('没有需要清理的历史班次')
    }
  } catch { /* handled by interceptor */ } finally {
    cleanupLoading.value = false
  }
}
const executeCleanup = async () => {
  if (!cleanupPreview.value || cleanupPreview.value.trip_count === 0) return
  const hasActive = cleanupPreview.value.has_active
  const msg = hasActive
    ? `确认清理 ${cleanupPreview.value.trip_count} 个历史班次？\n\n存在活跃订单，删除后将自动取消并退款已支付订单。此操作不可撤销！`
    : `确认清理 ${cleanupPreview.value.trip_count} 个历史班次？此操作不可撤销！`
  try {
    await ElMessageBox.confirm(msg, '清理确认', { type: 'warning', confirmButtonText: '确认清理', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' })
  } catch { return }
  cleanupExecuting.value = true
  try {
    const res: any = await tripApi.cleanup({
      before_date: cleanupBeforeDate.value,
      route_id: cleanupScope.value === 'current' ? currentRouteId.value : 0,
      force: true
    })
    ElMessage.success(res.message || '清理成功')
    cleanupDialogVisible.value = false
    cleanupPreview.value = null
    loadData()
  } catch { /* handled by interceptor */ } finally {
    cleanupExecuting.value = false
  }
}

onMounted(() => { loadDeps() })
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; flex-wrap: wrap; align-items: center }
.pagination { display: flex; justify-content: flex-end; margin-top: 16px }
.quick-select-bar { display: flex; gap: 6px; flex-wrap: wrap; align-items: center }
.range-select-bar { display: flex; gap: 4px; flex-wrap: wrap; align-items: center }
:deep(.weekend-row) { background-color: #fef0f0 !important; }
:deep(.weekend-row:hover > td) { background-color: #fde2e2 !important; }

/* 线路卡片网格 */
.route-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}
.route-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s;
  background: #fff;
  user-select: none;
}
.route-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.15);
  transform: translateY(-2px);
}
.route-card-header {
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}
.route-card-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}
.route-card-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.route-direction-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.dir-label {
  font-size: 14px;
  color: #606266;
}

/* 时间单元格 */
.time-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.time-date {
  font-size: 13px;
  color: #303133;
  font-weight: 500;
}
.time-clock {
  font-size: 12px;
  color: #909399;
}
</style>
