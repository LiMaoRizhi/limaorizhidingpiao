// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818

// 订阅消息工具
// 微信订阅消息机制：用户每次通过 wx.requestSubscribeMessage 授权一个模板，后端获得1次发送配额
// 用完后需用户下次操作时再次授权，所以每次支付前都应请求订阅

var log = require('./log')
const { request } = require('./request')

// 缓存后端已配置的订阅消息模板（避免每次支付都请求后端）
var cachedTemplates = null

// getEnabledTemplates 获取后端已配置的订阅消息模板列表
// 返回 Promise<[{key, template_id}]>，未配置则返回空数组
function getEnabledTemplates() {
  if (cachedTemplates !== null) {
    return Promise.resolve(cachedTemplates)
  }
  return request({
    url: '/api/wx/subscribe/templates',
    method: 'GET'
  }).then(function(res) {
    if (res.data && res.data.enabled && res.data.templates) {
      cachedTemplates = res.data.templates
      return cachedTemplates
    }
    cachedTemplates = []
    return cachedTemplates
  }).catch(function(e) {
    log.error('获取订阅消息模板失败', e)
    return []
  })
}

// requestSubscribe 请求用户订阅消息授权并上报到后端
// 在支付前调用，用户授权后后端获得发送配额
// 无论用户是否同意或获取模板失败，都会调用 callback 继续后续流程（不阻塞支付）
//
// 微信API限制：wx.requestSubscribeMessage 每次最多3个模板ID
// 当后端配置了4个模板时，分批请求：先发前3个，再发剩余的
function requestSubscribe(callback) {
  getEnabledTemplates().then(function(templates) {
    if (!templates || templates.length === 0) {
      callback()
      return
    }

    // 分批：每批最多3个模板
    var allAcceptedKeys = []
    var batches = []
    for (var i = 0; i < templates.length; i += 3) {
      batches.push(templates.slice(i, i + 3))
    }

    function processBatch(index) {
      if (index >= batches.length) {
        // 所有批次处理完，统一上报
        if (allAcceptedKeys.length > 0) {
          request({
            url: '/api/wx/subscribe/report',
            method: 'POST',
            data: { template_keys: allAcceptedKeys }
          }).catch(function(e) {
            log.error('上报订阅消息授权失败', e)
          })
        }
        callback()
        return
      }

      var batch = batches[index]
      var tmplIds = batch.map(function(t) { return t.template_id })
      wx.requestSubscribeMessage({
        tmplIds: tmplIds,
        success: function(res) {
          batch.forEach(function(t) {
            if (res[t.template_id] === 'accept') {
              allAcceptedKeys.push(t.key)
            }
          })
          processBatch(index + 1)
        },
        fail: function() {
          // 某批失败不影响其他批，继续处理下一批
          processBatch(index + 1)
        }
      })
    }

    processBatch(0)
  }).catch(function() {
    callback()
  })
}

module.exports = {
  requestSubscribe: requestSubscribe
}
