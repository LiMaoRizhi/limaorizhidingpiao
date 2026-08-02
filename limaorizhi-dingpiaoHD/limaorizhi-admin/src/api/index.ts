/* limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 */
import { http } from '@/utils/request'

// 通用类型
interface PageParams {
  page?: number
  page_size?: number
  keyword?: string
  [key: string]: unknown
}

interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 实体类型
interface Station {
  id: number
  name: string
  pinyin: string
  sort_order: number
  longitude: number
  latitude: number
  status: number
  created_at: string
  updated_at: string
}

interface RouteStation {
  id: number
  route_id: number
  station_id: number
  station?: Station
  stop_order: number
  distance_km: number
  price: number
  arrival_time: string
  arrival_day_offset: number
  created_at: string
}

interface Route {
  id: number
  name: string
  route_type: number // 1=城乡公交 2=城际客运 3=旅游专线
  from_station_id: number
  to_station_id: number
  from_station?: Station
  to_station?: Station
  route_stations?: RouteStation[]
  distance_km: number
  duration_minutes: number
  min_fare?: number // 起步价（最低票价），0=不启用
  status: number
  created_at: string
  updated_at: string
}

interface Vehicle {
  id: number
  plate_no: string
  vehicle_type: string
  seat_count: number
  status: number
  created_at: string
  updated_at: string
}

interface Driver {
  id: number
  employee_no: string // 工号（可选，用于区分重名司机）
  name: string
  phone: string
  license_no: string
  status: number
  last_login_at: string | null
  created_at: string
  updated_at: string
}

interface Trip {
  id: number
  route_id: number
  route?: Route
  vehicle_id: number
  vehicle?: Vehicle
  driver_id: number
  trip_no: string
  trip_date: string
  departure_time: string
  arrival_time: string
  arrival_day_offset: number
  total_seats: number
  available_seats: number
  base_price: number
  status: number
  current_passed_order: number
  created_at: string
  updated_at: string
}

interface Order {
  id: number
  order_no: string
  order_type: number
  user_id: number
  user?: { id: number; nickname: string; phone: string }
  trip_id: number
  trip?: Trip
  route_id: number
  from_station_id: number
  from_station?: Station
  to_station_id: number
  to_station?: Station
  trip_date: string
  departure_time: string
  passenger_count: number
  total_price: number
  status: number
  contact_name: string
  contact_phone: string
  sender_name: string
  sender_phone: string
  receiver_name: string
  receiver_phone: string
  cargo_type: string
  weight: number
  description: string
  pay_time: string | null
  pay_method: string
  created_at: string
  updated_at: string
}

interface OrderPassenger {
  id: number
  order_id: number
  name: string
  id_card_type: number
  id_card_no: string
  phone: string
  seat_no: string
  check_status: number
  checked_at: string | null
  checked_by: number
  created_at: string
}

interface Passenger {
  id: number
  user_id: number
  name: string
  id_card_type: number
  id_card_no: string
  phone: string
  created_at: string
  updated_at: string
}

interface Refund {
  id: number
  order_id: number
  refund_no: string
  amount: number
  reason: string
  status: number
  refund_time: string | null
  created_at: string
  updated_at: string
}

interface Banner {
  id: number
  title: string
  title_color: string
  title_effect: number
  image_url: string
  link_type: number
  link_value: string
  sort_order: number
  status: number
  created_at: string
  updated_at: string
}

interface OperationLog {
  id: number
  admin_id: number
  admin_name: string
  module: string
  action: string
  target: string
  detail: string
  ip_address: string
  created_at: string
}

interface Coupon {
  id: number
  name: string
  type: number
  discount_value: number
  min_spend: number
  valid_days: number
  total_count: number
  issued_count: number
  used_count: number
  status: number
  created_at: string
  updated_at: string
}

interface UserCoupon {
  id: number
  user_id: number
  user?: { id: number; nickname: string; phone: string }
  coupon_id: number
  coupon?: Coupon
  status: number
  issued_at: string
  expired_at: string
  used_at: string | null
  order_id: number
  created_at: string
}

interface PointRule {
  id: number
  rule_name: string
  rule_type: number
  points_per_yuan: number
  fixed_points: number
  description: string
  status: number
  created_at: string
  updated_at: string
}

interface UserPoints {
  id: number
  user_id: number
  user?: { id: number; nickname: string; phone: string }
  balance: number
  total_earned: number
  total_spent: number
  updated_at: string
}

interface PointRecord {
  id: number
  user_id: number
  change_type: number
  points: number
  source: string
  order_id: number
  remark: string
  admin_id: number
  admin_name: string
  created_at: string
}

interface DashboardStats {
  [key: string]: number | string
}

interface SystemConfigData {
  [key: string]: string
}

interface AdminUser {
  id: number
  username: string
  real_name: string
  avatar_url: string
  role: number
  status: number
  last_login_at: string | null
  created_at: string
  updated_at: string
}

// 创建/更新参数类型
interface StationCreateData {
  name: string
  pinyin?: string
  sort_order?: number
  longitude?: number
  latitude?: number
  status?: number
}

interface RouteStationInput {
  station_id: number
  distance_km?: number
  price?: number
  arrival_time?: string
  arrival_day_offset?: number
}

interface RouteCreateData {
  name: string
  route_type?: number // 1=城乡公交 2=城际客运 3=旅游专线
  from_station_id?: number
  to_station_id?: number
  distance_km?: number
  duration_minutes?: number
  min_fare?: number // 起步价（最低票价），0=不启用
  status?: number
  stations?: RouteStationInput[]
  force?: boolean // 强制更新（即使有活跃订单）
}

interface VehicleCreateData {
  plate_no: string
  vehicle_type: string
  seat_count: number
  status?: number
}

interface DriverCreateData {
  name: string
  phone: string
  password: string
  license_no?: string
  employee_no?: string
  status?: number
}

interface TripCreateData {
  route_id: number
  vehicle_id: number
  driver_id?: number
  trip_no?: string
  trip_date: string
  departure_time: string
  arrival_time: string
  arrival_day_offset?: number
  total_seats?: number
  base_price: number
  status?: number
  current_passed_order?: number
}

interface TripBatchDateItem {
  date: string
  departure_time: string
  arrival_time: string
  arrival_day_offset?: number
}

interface TripBatchData {
  route_id: number
  vehicle_id: number
  driver_id?: number
  // 日期范围模式（向后兼容）
  start_date?: string
  end_date?: string
  departure_time?: string
  arrival_time?: string
  arrival_day_offset?: number
  exclude_weekdays?: number[]
  // 自由日期模式（优先使用，各日期独立设置时间）
  trip_dates?: TripBatchDateItem[]
  base_price: number
  total_seats?: number
  force?: boolean
}

interface BannerCreateData {
  title?: string
  title_color?: string
  title_effect?: number
  image_url: string
  link_type?: number
  link_value?: string
  sort_order?: number
  status?: number
}

interface CouponCreateData {
  name: string
  type: number
  discount_value: number
  min_spend?: number
  valid_days?: number
  total_count?: number
  status?: number
}

interface PointRuleCreateData {
  rule_name: string
  rule_type: number
  points_per_yuan?: number
  fixed_points?: number
  description?: string
  status?: number
}

interface AdminCreateData {
  username: string
  password: string
  real_name?: string
  role: number
}

interface AssignDriverData {
  driver_id: number
  force?: boolean // 强制分配（已知冲突仍继续）
}

interface DriverConflictInfo {
  trip_id: number
  trip_no: string
  route_name: string
  from_station: string
  to_station: string
  departure_time: string
  arrival_time: string
  status: number
  status_text: string
  conflict_type: string // 'time_overlap' | 'location_gap'
  conflict_desc: string
}

interface DriverAvailabilityResult {
  driver: {
    id: number
    name: string
    employee_no: string
    phone: string
    status: number
    current_location: string
  }
  trips: {
    trip_id: number
    trip_no: string
    route_name: string
    from_station: string
    to_station: string
    departure_time: string
    arrival_time: string
    status: number
    status_text: string
    conflict_type: string
    conflict_desc: string
  }[]
  chain_desc: string
  has_conflict: boolean
}

interface OrderStatusData {
  status: number
  reason?: string
}

interface RefundData {
  reason: string
}

interface PointsAdjustData {
  points: number
  remark: string
}

// API 定义

// 仪表盘
export const dashboardApi = {
  stats: () => http.get<DashboardStats>('/admin/dashboard'),
}

// 站点
interface StationRouteInfo {
  id: number
  name: string
  route_type: number
  from_station: string
  to_station: string
  relation: string // 'from' | 'to' | 'via'
}

export const stationApi = {
  list: (params: PageParams) => http.get<PageResult<Station>>('/admin/stations', params),
  all: () => http.get<Station[]>('/admin/stations/all'),
  routes: (id: number) => http.get<{ station_id: string; station_name: string; routes: StationRouteInfo[]; total: number }>(`/admin/stations/${id}/routes`),
  create: (data: StationCreateData) => http.post<Station>('/admin/stations', data),
  update: (id: number, data: Partial<StationCreateData>) => http.put<Station>(`/admin/stations/${id}`, data),
  delete: (id: number, force?: boolean) => http.delete<void>(`/admin/stations/${id}`, force ? { force: 1 } : undefined),
}

// 线路
export const routeApi = {
  list: (params: PageParams) => http.get<PageResult<Route>>('/admin/routes', params),
  all: () => http.get<Route[]>('/admin/routes/all'),
  create: (data: RouteCreateData) => http.post<Route>('/admin/routes', data),
  update: (id: number, data: Partial<RouteCreateData>) => http.put<Route>(`/admin/routes/${id}`, data),
  delete: (id: number, force?: boolean) => http.delete<void>(`/admin/routes/${id}`, force ? { force: 1 } : undefined),
  stations: (id: number) => http.get<RouteStation[]>(`/admin/routes/${id}/stations`),
}

// 车辆
export const vehicleApi = {
  list: (params: PageParams) => http.get<PageResult<Vehicle>>('/admin/vehicles', params),
  all: () => http.get<Vehicle[]>('/admin/vehicles/all'),
  create: (data: VehicleCreateData) => http.post<Vehicle>('/admin/vehicles', data),
  update: (id: number, data: Partial<VehicleCreateData>) => http.put<Vehicle>(`/admin/vehicles/${id}`, data),
  delete: (id: number) => http.delete<void>(`/admin/vehicles/${id}`),
}

// 班次
export const tripApi = {
  list: (params: PageParams & { trip_date?: string; status?: number | string; route_id?: number | string; trip_no?: string }) => http.get<PageResult<Trip>>('/admin/trips', params),
  create: (data: TripCreateData) => http.post<Trip>('/admin/trips', data),
  update: (id: number, data: Partial<TripCreateData>) => http.put<Trip>(`/admin/trips/${id}`, data),
  delete: (id: number, force?: boolean) => http.delete<void>(`/admin/trips/${id}`, force ? { force: 1 } : undefined),
  batch: (data: TripBatchData) => http.post<void>('/admin/trips/batch', data),
  cleanup: (data: { before_date?: string; route_id?: number; force?: boolean }) => http.post<{ trip_count?: number; before_date?: string; total_orders?: number; pending_orders?: number; paid_orders?: number; history_orders?: number; has_active?: boolean; status_breakdown?: Record<string, number>; deleted_count?: number; refund_count?: number }>('/admin/trips/cleanup', data),
  passengers: (id: number) => http.get<OrderPassenger[]>(`/admin/trips/${id}/passengers`),
  assignDriver: (id: number, data: AssignDriverData) => http.put<void>(`/admin/trips/${id}/assign-driver`, data),
  verifyStats: (params?: PageParams) => http.get<{ [key: string]: number | string }>('/admin/drivers/verify-stats', params),
}

// 订单
export const orderApi = {
  list: (params: PageParams & { order_type?: number; status?: number }) => http.get<PageResult<Order>>('/admin/orders', params),
  detail: (id: number) => http.get<{ order: Order; passengers: OrderPassenger[] }>(`/admin/orders/${id}`),
  updateStatus: (id: number, data: OrderStatusData) => http.put<void>(`/admin/orders/${id}/status`, data),
  refund: (id: number, data: RefundData) => http.post<void>(`/admin/orders/${id}/refund`, data),
  // 统一走API层导出CSV，避免View直接操作底层request
  export: (params: PageParams) => http.download('/admin/orders/export', params),
}

// 退款
export const refundApi = {
  list: (params: PageParams) => http.get<PageResult<Refund>>('/admin/refunds', params),
}

// 小程序用户
export interface WxUser {
  id: number
  openid: string
  unionid: string
  nickname: string
  avatar_url: string
  phone: string
  status: number
  created_at: string
  updated_at: string
  order_count?: number
  total_amount?: number
}
export interface UserDetail {
  user: WxUser
  order_count: number
  total_amount: number
  refund_count: number
  cargo_count: number
  orders: Order[]
  passengers: Passenger[]
}
export const userApi = {
  list: (params: PageParams) => http.get<PageResult<WxUser>>('/admin/users', params),
  detail: (id: number) => http.get<UserDetail>(`/admin/users/${id}`),
  updateStatus: (id: number, data: { status: number }) => http.put<void>(`/admin/users/${id}/status`, data),
  passengers: (params: PageParams) => http.get<PageResult<Passenger>>('/admin/passengers', params),
}

// 实名认证缓存管理（命中率监控 + 主动失效）
// 后端路由：GET/POST /admin/idcard-verify/stats[/reset]、DELETE /admin/passengers/:id/idcard-cache
export interface IDCardVerifyStats {
  cache_hits: number      // 缓存命中次数（直接返回，未调用云市场 API）
  cache_misses: number    // 缓存未命中次数（需调用云市场 API）
  api_calls: number        // 实际调用云市场 API 的次数（含重试）
  api_errors: number       // API 调用失败次数
  cache_writes: number    // 缓存写入次数
  cache_deletes: number    // 主动删除次数
  hit_rate: number          // 缓存命中率 = cache_hits / (cache_hits + cache_misses)
  since: string             // 统计起始时间（进程启动或最近一次重置）
}
export interface IDCardVerifyStatsResponse {
  stats: IDCardVerifyStats
  saved_cost_yuan: number   // 累计节省成本（估算，元）
  spent_cost_yuan: number   // 累计花费成本（估算，元）
  price_per_call: number    // 单价 0.3
  cost_unit: string           // "CNY"
  note: string                 // 说明
}
export interface InvalidateCacheResult {
  passenger_id: number
  name: string
  id_card_mask: string       // 脱敏后的身份证号
}
export const idCardVerifyApi = {
  stats: () => http.get<IDCardVerifyStatsResponse>('/admin/idcard-verify/stats'),
  resetStats: () => http.post<void>('/admin/idcard-verify/stats/reset'),
  invalidateCache: (passengerId: number) => http.delete<InvalidateCacheResult>(`/admin/passengers/${passengerId}/idcard-cache`),
}

// 轮播图
export const bannerApi = {
  list: () => http.get<Banner[]>('/admin/banners'),
  create: (data: BannerCreateData) => http.post<Banner>('/admin/banners', data),
  update: (id: number, data: Partial<BannerCreateData>) => http.put<Banner>(`/admin/banners/${id}`, data),
  delete: (id: number) => http.delete<void>(`/admin/banners/${id}`),
}

// 系统配置
export const configApi = {
  get: () => http.get<SystemConfigData>('/admin/config'),
  update: (data: { configs?: SystemConfigData; [key: string]: unknown }) => http.put<void>('/admin/config', data),
}

// 保险公司配置（通用保险对接框架）
interface InsuranceProvider {
  id: number
  name: string
  api_url: string
  app_id: string
  app_secret_masked: string
  product_code: string
  fee: number
  is_active: boolean
  required: boolean
  remark: string
  created_at: string
  updated_at: string
}

interface InsuranceProviderCreateData {
  name: string
  api_url: string
  app_id: string
  app_secret?: string
  product_code?: string
  fee?: number
  required?: boolean
  remark?: string
  is_active?: boolean
}

export const insuranceProviderApi = {
  list: (params: PageParams) => http.get<PageResult<InsuranceProvider>>('/admin/insurance-providers', params),
  create: (data: InsuranceProviderCreateData) => http.post<InsuranceProvider>('/admin/insurance-providers', data),
  update: (id: number, data: Partial<InsuranceProviderCreateData>) =>
    http.put<InsuranceProvider>(`/admin/insurance-providers/${id}`, data),
  delete: (id: number) => http.delete<void>(`/admin/insurance-providers/${id}`),
  activate: (id: number) => http.put<void>(`/admin/insurance-providers/${id}/activate`),
}

// 管理员
export const adminApi = {
  list: (params: PageParams) => http.get<PageResult<AdminUser>>('/admin/admins', params),
  create: (data: AdminCreateData) => http.post<AdminUser>('/admin/admins', data),
  update: (id: number, data: Partial<AdminCreateData>) => http.put<AdminUser>(`/admin/admins/${id}`, data),
  resetPassword: (id: number, data: { password: string }) => http.put<void>(`/admin/admins/${id}/reset-password`, data),
  delete: (id: number) => http.delete<void>(`/admin/admins/${id}`),
  logs: (params: PageParams) => http.get<PageResult<OperationLog>>('/admin/logs', params),
  logsExport: (params: PageParams) => http.download('/admin/logs/export', params),
}

// 上传
export const uploadApi = {
  upload: (formData: FormData) => http.post<{ url: string }>('/admin/upload', formData),
}

// 司机
export const driverApi = {
  list: (params: PageParams) => http.get<PageResult<Driver>>('/admin/drivers', params),
  all: () => http.get<Driver[]>('/admin/drivers/all'),
  create: (data: DriverCreateData) => http.post<Driver>('/admin/drivers', data),
  update: (id: number, data: Partial<DriverCreateData>) => http.put<Driver>(`/admin/drivers/${id}`, data),
  delete: (id: number) => http.delete<void>(`/admin/drivers/${id}`),
  availability: (params: { driver_id: number; trip_date: string; departure_time?: string; arrival_time?: string; exclude_trip_id?: number }) => http.get<DriverAvailabilityResult>('/admin/drivers/availability', params),
}

// 优惠券
export const couponApi = {
  list: (params: PageParams) => http.get<PageResult<Coupon>>('/admin/coupons', params),
  create: (data: CouponCreateData) => http.post<Coupon>('/admin/coupons', data),
  update: (id: number, data: Partial<CouponCreateData>) => http.put<Coupon>(`/admin/coupons/${id}`, data),
  delete: (id: number) => http.delete<void>(`/admin/coupons/${id}`),
}

// 发放记录
export const userCouponApi = {
  list: (params: PageParams) => http.get<PageResult<UserCoupon>>('/admin/user-coupons', params),
}

// 积分规则
export const pointRuleApi = {
  list: (params: PageParams) => http.get<PageResult<PointRule>>('/admin/point-rules', params),
  create: (data: PointRuleCreateData) => http.post<PointRule>('/admin/point-rules', data),
  update: (id: number, data: Partial<PointRuleCreateData>) => http.put<PointRule>(`/admin/point-rules/${id}`, data),
  delete: (id: number) => http.delete<void>(`/admin/point-rules/${id}`),
}

// 用户积分
export const userPointsApi = {
  list: (params: PageParams) => http.get<PageResult<UserPoints>>('/admin/user-points', params),
  records: (id: number, params: PageParams) => http.get<PageResult<PointRecord>>(`/admin/user-points/${id}/records`, params),
  adjust: (id: number, data: PointsAdjustData) => http.post<void>(`/admin/user-points/${id}/adjust`, data),
}

// 系统
export const systemApi = {
  changePassword: (data: { old_password: string; new_password: string }) => http.put<void>('/admin/password', data),
}

// 装修布局组件
interface LayoutItem {
  type: string
  title: string
  visible: boolean
}

// 首页装修 + 多页面装修（订单页标签/我的页订单分类/我的页功能菜单）
export const designApi = {
  getLayout: () => http.get<LayoutItem[]>('/admin/homepage/layout'),
  updateLayout: (data: unknown) => http.put<void>('/admin/homepage/layout', data),
  getPageLayout: (page: string) => http.get<LayoutItem[]>('/admin/design/page-layout', { page }),
  updatePageLayout: (page: string, data: unknown) => http.put<void>(`/admin/design/page-layout?page=${encodeURIComponent(page)}`, data),
}

// 公开接口（与小程序使用相同接口，用于装修预览）
export const publicApi = {
  banners: () => http.get<Banner[]>('/api/banners'),
  coupons: () => http.get<Coupon[]>('/api/homepage/coupons'),
  config: () => http.get<SystemConfigData>('/api/wx/config'),
  stations: () => http.get<Station[]>('/api/wx/stations'),
  trips: (tripDate: string) => http.get<Trip[]>('/api/wx/trips', { trip_date: tripDate }),
  pageLayout: (page: string) => http.get<LayoutItem[]>('/api/design/page-layout', { page }),
}

// 协议政策
export const agreementApi = {
  get: () => http.get<SystemConfigData>('/admin/config'),
  update: (data: { configs: SystemConfigData }) => http.put<void>('/admin/config', data),
}

// 轨迹可视化
export interface ActiveTrip {
  trip_id: number
  trip_no: string
  trip_date: string
  departure_time: string
  arrival_time: string
  route_name: string
  from_station: string
  to_station: string
  driver_name: string
  driver_phone: string
  vehicle_plate_no: string
  vehicle_type: string
  passed_order: number
  total_stations: number
  longitude: number | null
  latitude: number | null
  speed: number | null
  heading: number | null
  reported_at: string | null
  seconds_ago: number | null
}
export interface TrackPoint {
  longitude: number
  latitude: number
  speed: number
  heading: number
  reported_at: string
}
export interface TrackStation {
  stop_order: number
  station_id: number
  name: string
  longitude: number
  latitude: number
}
export interface TripTrackData {
  trip: Record<string, unknown>
  points: TrackPoint[]
  stations: TrackStation[]
}
export const trackApi = {
  activeTrips: () => http.get<{ list: ActiveTrip[]; total: number }>('/admin/trips/active'),
  tripTrack: (id: number) => http.get<TripTrackData>(`/admin/trips/${id}/track`),
}

// AI 数字员工
export interface AIConfig {
  ai_employee_enabled: boolean
  ai_provider: string
  ai_base_url: string
  ai_model: string
  ai_system_prompt: string
  ai_api_key_configured: boolean
}
export interface ModelInfo {
  id: string
  name: string
  description: string
  tag: string
  tag_type: string
  supports_vision?: boolean
  icon?: string
}
export interface ProviderInfo {
  value: string
  name: string
  group: string
  group_name: string
  base_url: string
  needs_key: boolean
  tag: string
  tag_type: string
  hint: string
  has_key: boolean
  models: ModelInfo[]
}
export interface AIModelsResponse {
  providers: ProviderInfo[]
  current_provider: string
  current_model: string
}
export const aiApi = {
  getConfig: () => http.get<AIConfig>('/admin/ai/config'),
  updateConfig: (data: { configs: Record<string, string> }) => http.put<void>('/admin/ai/config', data),
  getModels: () => http.get<AIModelsResponse>('/admin/ai/models'),
  switchModel: (model: string) => http.put<void>('/admin/ai/model', { model }),
  generateImage: (prompt: string) => http.post<{ image: string; prompt: string }>('/admin/ai/image', { prompt }),
  // chat 使用原生 fetch 处理 SSE 流（需携带 JWT 头，不能用 EventSource）
}
