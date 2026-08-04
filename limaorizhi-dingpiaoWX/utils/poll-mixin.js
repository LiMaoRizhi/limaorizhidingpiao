// 轮询退避机制 Mixin（bus-progress / vehicle-track 共用）
// 正常间隔8秒，请求失败时指数退避（8→16→32→60秒封顶），成功后恢复8秒

/**
 * 创建轮询退避机制，展开到 Page() 选项中使用
 * @param {string} loadFnName - 页面加载数据的方法名（如 'loadTrip' 或 'loadLocation'）
 * @returns {object} 属性和方法，可直接展开到 Page({ ...pollMixin, ... })
 *
 * 用法：
 *   const pollMixin = createPollMixin('loadTrip')
 *   Page({
 *     ...pollMixin,
 *     // 页面自有属性和方法
 *   })
 */
function createPollMixin(loadFnName) {
  return {
    // 轮询间隔（毫秒），动态调整
    pollInterval: 8000,
    // 最大轮询间隔（60秒封顶）
    maxPollInterval: 60000,
    // 连续失败计数
    consecutiveErrors: 0,
    // setTimeout 句柄
    timer: null,

    // 清除轮询定时器
    clearTimer() {
      if (this.timer) {
        clearTimeout(this.timer)
        this.timer = null
      }
    },

    // 调度下一次轮询
    scheduleNextPoll() {
      this.clearTimer()
      this.timer = setTimeout(() => {
        this[loadFnName]()
      }, this.pollInterval)
    },

    // 根据请求结果调整轮询间隔
    adjustPollInterval(success) {
      if (success) {
        this.consecutiveErrors = 0
        this.pollInterval = 8000
      } else {
        this.consecutiveErrors++
        this.pollInterval = Math.min(this.pollInterval * 2, this.maxPollInterval)
      }
      this.scheduleNextPoll()
    },

    // 重置轮询（onShow/重试的时候得调一下）
    resetPollState() {
      this.pollInterval = 8000
      this.consecutiveErrors = 0
    }
  }
}

module.exports = {
  createPollMixin: createPollMixin
}
