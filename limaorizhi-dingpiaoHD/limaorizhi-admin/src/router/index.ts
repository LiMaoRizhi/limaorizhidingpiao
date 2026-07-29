/* limaorizhi-admin  狸猫日志售票系统  联系微信：lihao68681818 */
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBrandStore } from '@/stores/brand'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录' },
  },
  {
    path: '/verify',
    name: 'Verify',
    component: () => import('@/views/verify/index.vue'),
    meta: { title: '安全验证' },
  },
  {
    path: '/',
    component: () => import('@/layout/index.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '首页', icon: 'dashboard' },
      },
      {
        path: 'track',
        name: 'Track',
        component: () => import('@/views/track/index.vue'),
        meta: { title: '轨迹' },
      },
      {
        path: 'ticket/stations',
        name: 'Stations',
        component: () => import('@/views/ticket/stations.vue'),
        meta: { title: '站点管理' },
      },
      {
        path: 'ticket/routes',
        name: 'Routes',
        component: () => import('@/views/ticket/routes.vue'),
        meta: { title: '线路管理' },
      },
      {
        path: 'ticket/routes/edit',
        name: 'RouteEdit',
        component: () => import('@/views/ticket/route-edit.vue'),
        meta: { title: '线路编辑' },
      },
      {
        path: 'ticket/trips',
        name: 'Trips',
        component: () => import('@/views/ticket/trips.vue'),
        meta: { title: '班次管理' },
      },
      {
        path: 'ticket/vehicles',
        name: 'Vehicles',
        component: () => import('@/views/ticket/vehicles.vue'),
        meta: { title: '车辆管理' },
      },
      {
        path: 'design/banners',
        name: 'Banners',
        component: () => import('@/views/ticket/banners.vue'),
        meta: { title: '轮播图装修' },
      },
      {
        path: 'design/decorate',
        name: 'Decorate',
        component: () => import('@/views/design/decorate.vue'),
        meta: { title: '小程序装修' },
      },
      {
        path: 'design/phone',
        name: 'PhoneSettings',
        component: () => import('@/views/design/phone.vue'),
        meta: { title: '电话设置' },
      },
      {
        path: 'design/coupon-display',
        name: 'CouponDisplay',
        component: () => import('@/views/design/coupon-display.vue'),
        meta: { title: '优惠券展示' },
      },
      {
        path: 'ticket/cargo',
        name: 'Cargo',
        component: () => import('@/views/ticket/cargo.vue'),
        meta: { title: '托运管理' },
      },
      {
        path: 'marketing/coupons',
        name: 'Coupons',
        component: () => import('@/views/marketing/coupons.vue'),
        meta: { title: '优惠券管理' },
      },
      {
        path: 'marketing/point-rules',
        name: 'PointRules',
        component: () => import('@/views/marketing/point-rules.vue'),
        meta: { title: '积分规则' },
      },
      {
        path: 'marketing/user-points',
        name: 'UserPoints',
        component: () => import('@/views/marketing/user-points.vue'),
        meta: { title: '用户积分明细' },
      },
      {
        path: 'marketing/distribution',
        name: 'Distribution',
        component: () => import('@/views/marketing/distribution.vue'),
        meta: { title: '发放记录' },
      },
      {
        path: 'order/list',
        name: 'OrderList',
        component: () => import('@/views/order/list.vue'),
        meta: { title: '订单列表' },
      },
      {
        path: 'order/refunds',
        name: 'Refunds',
        component: () => import('@/views/order/refunds.vue'),
        meta: { title: '退款记录' },
      },
      {
        path: 'user/list',
        name: 'UserList',
        component: () => import('@/views/user/list.vue'),
        meta: { title: '小程序用户' },
      },
      {
        path: 'user/passengers',
        name: 'Passengers',
        component: () => import('@/views/user/passengers.vue'),
        meta: { title: '常用乘客' },
      },
      {
        path: 'user/idcard-verify',
        name: 'IDCardVerify',
        component: () => import('@/views/user/idcard-verify.vue'),
        meta: { title: '实名认证缓存' },
      },
      {
        path: 'setting/drivers',
        name: 'Drivers',
        component: () => import('@/views/setting/drivers.vue'),
        meta: { title: '司机管理', requireSuperAdmin: true },
      },
      {
        path: 'setting/admins',
        name: 'Admins',
        component: () => import('@/views/setting/admins.vue'),
        meta: { title: '管理员管理', requireSuperAdmin: true },
      },
      {
        path: 'system/password',
        name: 'Password',
        component: () => import('@/views/system/password.vue'),
        meta: { title: '修改密码' },
      },
      {
        path: 'system/brand',
        name: 'Brand',
        component: () => import('@/views/system/brand.vue'),
        meta: { title: '品牌设置', requireSuperAdmin: true },
      },
      {
        path: 'system/config',
        name: 'Config',
        component: () => import('@/views/setting/config.vue'),
        meta: { title: '系统配置', requireSuperAdmin: true },
      },
      {
        path: 'system/agreement',
        name: 'Agreement',
        component: () => import('@/views/setting/agreement.vue'),
        meta: { title: '协议政策', requireSuperAdmin: true },
      },
      {
        path: 'system/logs',
        name: 'Logs',
        component: () => import('@/views/setting/logs.vue'),
        meta: { title: '操作日志', requireSuperAdmin: true },
      },
    ],
  },
  // 兜底：未知路径重定向到首页
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫
router.beforeEach(async (to, _from, next) => {
  const brandStore = useBrandStore()
  document.title = (to.meta.title as string) ? `${to.meta.title} - ${brandStore.systemName}` : brandStore.systemName
  const authStore = useAuthStore()
  if (to.path !== '/login' && to.path !== '/verify' && !authStore.isLogin()) {
    next('/login')
  } else if ((to.path === '/login' || to.path === '/verify') && authStore.isLogin()) {
    next('/')
  } else {
    // 首次导航或进入超管页面时从服务器验证token有效性并获取最新角色信息
    // 超管页面每次都重新校验，防止localStorage角色被篡改后绕过前端守卫
    if (authStore.isLogin() && (!authStore.profileValidated || to.meta.requireSuperAdmin)) {
      await authStore.fetchProfile()
      // 验证后若token失效，跳转登录
      if (!authStore.isLogin()) {
        next('/login')
        return
      }
    }
    // 默认口令强制改密：已登录但未改密时，除改密页外强制跳转改密页
    if (authStore.isLogin() && authStore.mustChangePassword && to.path !== '/system/password') {
      next('/system/password')
      return
    }
    if (to.meta.requireSuperAdmin && !authStore.isSuperAdmin()) {
      next('/')
    } else {
      next()
    }
  }
})

export default router
