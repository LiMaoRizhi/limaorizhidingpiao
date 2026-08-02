// 管理后台 - 数字员工（AI 对话，SSE 流式回复）
// 接 /api/wx/admin/ai/chat(SSE流) + /api/wx/admin/ai/models(模型列表)
const { request, BASE_URL } = require('../../utils/request')

Page({
  data: {
    adminRole: 0,
    // 对话
    messages: [],          // [{role:'user'|'assistant', content:'', reasoning:'', loading:false, error:false}]
    input: '',             // 输入框文本
    sending: false,        // 是否正在发送/等待回复
    scrollIntoView: '',    // 滚动锚点 id
    // 模型
    modelPickerVisible: false,
    providers: [],
    currentModel: '',
    currentModelName: '',
    currentProvider: '',
    modelsLoadError: false,   // 模型列表加载失败标记（picker 里展示重试入口）
    // 诊断用：顶栏直接展示加载状态/数量，定位"无模型可选"是接口失败还是数据为空
    modelsLoading: false,
    modelsTotal: 0,
    // 错误提示
    fatalError: ''         // AI 未启用等不可恢复错误，置顶提示
  },

  // SSE 相关（非 data，不放 setData）
  _requestTask: null,
  _sseBuffer: '',          // SSE 文本缓冲（按 \n\n 切事件）
  _utf8Tail: '',           // 跨 chunk 的不完整 UTF-8 字符（理论上 TextDecoder 会处理，这里兜底）
  _decoder: null,
  _assistantIdx: -1,       // 当前正在写入的助手消息 index

  onLoad() {
    const userInfo = wx.getStorageSync('user_info') || {}
    const adminRole = userInfo.admin_role || 0
    // 前端权限拦截：非管理员直接退出，避免后端拒了请求还白渲染一屏
    if (!adminRole) {
      wx.showToast({ title: '无权访问', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 800)
      return
    }
    this.setData({ adminRole })
    if (typeof TextDecoder !== 'undefined') {
      this._decoder = new TextDecoder('utf-8')
    }
    this.loadModels()
  },

  onUnload() {
    this.abortStream()
  },

  onHide() {
    // 离开页面时中断流，避免后台继续接收造成卡顿
    this.abortStream()
  },

  // 拉取模型列表 + 当前模型
  loadModels() {
    this.setData({ modelsLoading: true })
    request({ url: '/api/wx/admin/ai/models', method: 'GET', silent: true }).then(res => {
      // 诊断：打印完整响应。res 形如 {code,message,data:{providers,current_provider,current_model}}
      // 若顶栏数量为 0，看这里即可判断是接口失败、数据为空、还是结构不匹配
      console.log('[admin-ai] loadModels 响应:', res)
      const d = res.data || {}
      const providers = (d.providers || []).map(p => ({
        value: p.value,
        name: p.name,
        group_name: p.group_name || '',
        has_key: !!p.has_key,
        // 不再过滤 image 模型，和 web 端保持一致（后端 SwitchModel 会对 image 模型返回明确错误）
        models: (p.models || []).map(m => ({
          id: m.id,
          name: m.name,
          description: m.description || '',
          tag: m.tag || '',
          tag_type: m.tag_type || '',
          supports_vision: !!m.supports_vision,
          icon: m.icon || ''
        }))
      }))
      // 标记当前选中
      const cur = d.current_model || ''
      providers.forEach(p => {
        p.models.forEach(m => {
          m.selected = (m.id === cur)
        })
      })
      const curProvider = d.current_provider || ''
      const curModelName = this.findModelName(providers, cur)
      const modelsTotal = providers.reduce((s, p) => s + (p.models ? p.models.length : 0), 0)
      this.setData({
        providers,
        currentModel: cur,
        currentModelName: curModelName,
        currentProvider: curProvider,
        modelsLoadError: false,
        modelsLoading: false,
        modelsTotal
      })
    }).catch((err) => {
      // silent 模式下 request.js 不弹 toast，这里必须自己提示，否则用户面对空列表不知发生了什么
      console.log('[admin-ai] loadModels 失败:', err)
      this.setData({ modelsLoadError: true, modelsLoading: false, modelsTotal: 0 })
      const msg = (err && err.message) || '模型列表加载失败'
      wx.showToast({ title: msg, icon: 'none' })
    })
  },

  findModelName(providers, modelId) {
    for (const p of providers) {
      for (const m of p.models) {
        if (m.id === modelId) return m.name
      }
    }
    return modelId || ''
  },

  // 切换模型选择面板
  toggleModelPicker() {
    this.setData({ modelPickerVisible: !this.data.modelPickerVisible })
  },

  closeModelPicker() {
    this.setData({ modelPickerVisible: false })
  },

  // 重新加载模型列表（picker 内空状态点重试用）
  reloadModels() {
    this.loadModels()
  },

  // 选中某个模型
  onModelSelect(e) {
    wx.vibrateShort({ type: 'light', fail: () => {} })
    // currentTarget 取不到时兜底用 target
    const ds = (e.currentTarget && e.currentTarget.dataset) || (e.target && e.target.dataset) || {}
    const model = ds.model
    const provider = ds.provider
    if (!model) {
      wx.showToast({ title: '未识别到模型，请重试', icon: 'none' })
      return
    }
    if (this.data.sending) {
      wx.showToast({ title: '请等待当前回复完成', icon: 'none' })
      return
    }
    // 仅允许有 key 的服务商
    const p = this.data.providers.find(x => x.value === provider)
    if (p && !p.has_key) {
      wx.showToast({ title: '该服务商未配置 API Key', icon: 'none' })
      return
    }
    if (model === this.data.currentModel) {
      this.setData({ modelPickerVisible: false })
      return
    }
    wx.showLoading({ title: '切换中...' })
    request({
      url: '/api/wx/admin/ai/model',
      method: 'PUT',
      data: { model }
    }).then(() => {
      const providers = this.data.providers.map(p => {
        return {
          ...p,
          models: p.models.map(m => ({ ...m, selected: (m.id === model) }))
        }
      })
      this.setData({
        providers,
        currentModel: model,
        currentModelName: this.findModelName(providers, model),
        currentProvider: provider,
        modelPickerVisible: false
      })
      wx.hideLoading()
      wx.showToast({ title: '已切换模型', icon: 'success' })
    }).catch((err) => {
      wx.hideLoading()
      // showLoading 会压制 request.js 内部的 toast，这里补一次确保错误可见
      const msg = (err && err.message) || '切换失败，请重试'
      setTimeout(() => {
        wx.showToast({ title: msg, icon: 'none' })
      }, 100)
    })
  },

  // 输入框
  onInput(e) {
    this.setData({ input: e.detail.value })
  },

  // 发送
  onSendTap() {
    this.sendMessage()
  },

  // 真正发送逻辑
  sendMessage() {
    if (this.data.sending) return
    const text = (this.data.input || '').trim()
    if (!text) return

    // 用户消息
    const userMsg = { role: 'user', content: text }
    // 助手占位消息（流式写入）
    const assistantMsg = { role: 'assistant', content: '', reasoning: '', loading: true, error: false }

    const messages = this.data.messages.concat([userMsg, assistantMsg])
    this.setData({
      messages,
      input: '',
      sending: true,
      fatalError: ''
    })
    this._assistantIdx = messages.length - 1
    this.scrollToBottom()

    // 构造给后端的 messages（不含占位助手消息）
    const payload = {
      messages: this.data.messages.slice(0, -1).map(m => ({
        role: m.role,
        content: m.content
      }))
    }

    this.startSSE(payload)
  },

  // 启动 SSE 流
  startSSE(payload) {
    const token = wx.getStorageSync('user_token') || ''
    this._sseBuffer = ''
    this._aborted = false  // 新请求开始前重置中止标志（abortStream 不再自己重置，避免 fail 回调异步触发时已被重置）
    this._utf8Tail = ''

    const that = this
    this._requestTask = wx.request({
      url: BASE_URL + '/api/wx/admin/ai/chat',
      method: 'POST',
      data: payload,
      enableChunked: true,
      timeout: 300000, // 5 分钟，与后端 client.Timeout 对齐
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? 'Bearer ' + token : '',
        'Accept': 'text/event-stream'
      },
      success(res) {
        // enableChunked 模式下，res.data 可能是空字符串/空对象
        // 实际数据通过 onChunkReceived 接收
        // 这里只处理"非流式错误"（如 AI 未启用、Token 失效、配置不全等）
        if (res.statusCode !== 200) {
          that.handleNonStreamError('服务器错误(HTTP ' + res.statusCode + ')')
          return
        }
        // 部分情况下后端直接返回 JSON 错误体（未走 SSE）
        const data = res.data
        if (data && typeof data === 'object' && data.code !== undefined) {
          if (data.code !== 0) {
            if (data.code === 1002) {
              that.handleNonStreamError('登录已失效，请重新登录')
              // 触发 request.js 的统一登录失效处理：直接 reLaunch
              wx.removeStorageSync('user_token')
              wx.removeStorageSync('user_info')
              setTimeout(() => wx.reLaunch({ url: '/pages/login/login' }), 800)
              return
            }
            that.handleNonStreamError(data.message || '请求失败')
            return
          }
        }
        // 流正常结束但没收到 done（或 done 已在 chunk 中处理），确保 loading 关闭
        that.finalizeAssistant()
      },
      fail(err) {
        // 网络错误/超时
        if (that._aborted) return
        that.handleNonStreamError('网络错误，请稍后重试')
      }
    })

    // 接收分块数据
    if (this._requestTask && this._requestTask.onChunkReceived) {
      this._requestTask.onChunkReceived(function (res) {
        if (!res || !res.data) return
        try {
          let text
          if (that._decoder) {
            text = that._decoder.decode(new Uint8Array(res.data), { stream: true })
          } else {
            // 兜底：用 wx.arrayBufferToBase64 + 解码（极少数老版本）
            text = that.arrayBufferToUtf8(res.data)
          }
          that.feedSSEText(text)
        } catch (e) {
          // 解码失败，忽略此 chunk
        }
      })
    }
  },

  // 把分块文本喂给 SSE 解析器
  feedSSEText(text) {
    if (!text) return
    this._sseBuffer += text
    // 标准化换行，兼容经代理后变成 \r\n\r\n 的事件分隔
    if (this._sseBuffer.indexOf('\r') !== -1) {
      this._sseBuffer = this._sseBuffer.replace(/\r\n/g, '\n').replace(/\r/g, '\n')
    }
    // SSE 事件以空行分隔
    let idx
    while ((idx = this._sseBuffer.indexOf('\n\n')) !== -1) {
      const block = this._sseBuffer.slice(0, idx)
      this._sseBuffer = this._sseBuffer.slice(idx + 2)
      this.handleSSEBlock(block)
    }
  },

  // 处理单个 SSE 事件块
  handleSSEBlock(block) {
    // 解析 data: 行（gin SSEvent 格式：event: message\ndata: {...}）
    const lines = block.split('\n')
    let dataLine = ''
    for (const line of lines) {
      const trimmed = line.trim()
      if (trimmed.indexOf('data:') === 0) {
        dataLine += trimmed.slice(5).trim()
      }
    }
    if (!dataLine) return

    let payload
    try {
      payload = JSON.parse(dataLine)
    } catch (e) {
      return
    }
    if (!payload || typeof payload !== 'object') return

    const type = payload.type || ''
    if (type === 'done') {
      this.finalizeAssistant()
      return
    }
    if (type === 'reasoning') {
      this.appendAssistantContent('reasoning', payload.content || '')
      return
    }
    if (type === 'content') {
      this.appendAssistantContent('content', payload.content || '')
      return
    }
  },

  // 写入助手消息（content 或 reasoning）
  appendAssistantContent(field, text) {
    if (!text) return
    const idx = this._assistantIdx
    if (idx < 0 || idx >= this.data.messages.length) return
    const msg = this.data.messages[idx]
    if (field === 'reasoning') {
      msg.reasoning = (msg.reasoning || '') + text
    } else {
      msg.content = (msg.content || '') + text
      msg.loading = false
    }
    // 直接修改数组引用，避免频繁全量 setData
    this.data.messages[idx] = msg
    // 仅更新该条 + 滚动
    this.setData({
      ['messages[' + idx + ']']: msg
    })
    this.scrollToBottom()
  },

  // 完成助手消息（关闭 loading）
  finalizeAssistant() {
    if (this._assistantIdx < 0) return
    const idx = this._assistantIdx
    if (idx >= this.data.messages.length) {
      this.resetSending()
      return
    }
    const msg = this.data.messages[idx]
    if (msg.loading) {
      msg.loading = false
      // 如果完全没有内容，给个兜底提示
      if (!msg.content) {
        msg.content = '(未收到回复)'
        msg.error = true
      }
      this.setData({
        ['messages[' + idx + ']']: msg,
        sending: false
      })
    } else {
      this.setData({ sending: false })
    }
    this._assistantIdx = -1
    this._requestTask = null
  },

  // 非 SSE 错误（请求层失败、AI 未启用、配置不全等）
  handleNonStreamError(message) {
    const idx = this._assistantIdx
    if (idx >= 0 && idx < this.data.messages.length) {
      const msg = this.data.messages[idx]
      msg.loading = false
      msg.error = true
      msg.content = message || '请求失败'
      this.setData({
        ['messages[' + idx + ']']: msg,
        sending: false
      })
    } else {
      this.setData({ sending: false })
    }
    this._assistantIdx = -1
    this._requestTask = null
  },

  // 中断当前流（_aborted 不在此重置，由下次 startSSE 重置，避免 fail 回调异步触发时已被重置成 false）
  abortStream() {
    if (this._aborted && !this._requestTask) return  // 已经中止过，避免重复处理
    this._aborted = true
    if (this._requestTask) {
      try { this._requestTask.abort() } catch (e) {}
      this._requestTask = null
    }
    // 手动收尾助手消息：用户主动停止应显示"已停止"，而非 finalizeAssistant 的"(未收到回复)"+error
    const idx = this._assistantIdx
    if (idx >= 0 && idx < this.data.messages.length) {
      const msg = this.data.messages[idx]
      if (msg.loading) {
        msg.loading = false
        if (!msg.content) {
          msg.content = '（已停止）'
        }
        this.setData({
          ['messages[' + idx + ']']: msg,
          sending: false
        })
      } else {
        this.setData({ sending: false })
      }
    } else {
      this.setData({ sending: false })
    }
    this._assistantIdx = -1
  },

  // 手动停止
  onStopTap() {
    this.abortStream()
  },

  // 重试上一条
  onRetryTap() {
    if (this.data.sending) return
    const msgs = this.data.messages
    if (msgs.length < 2) return
    // 找到最后一条用户消息
    let lastUserIdx = -1
    for (let i = msgs.length - 1; i >= 0; i--) {
      if (msgs[i].role === 'user') { lastUserIdx = i; break }
    }
    if (lastUserIdx < 0) return
    const text = msgs[lastUserIdx].content
    // 删除该用户消息之后的所有消息（含失败的助手回复）
    const kept = msgs.slice(0, lastUserIdx)
    this.setData({ messages: kept })
    this.setData({ input: text })
    this.sendMessage()
  },

  // 清空对话
  onClearTap() {
    if (this.data.sending) {
      wx.showToast({ title: '请等待当前回复完成', icon: 'none' })
      return
    }
    if (this.data.messages.length === 0) return
    wx.showModal({
      title: '清空对话',
      content: '确定要清空所有对话记录吗？',
      confirmColor: '#1a1a1a',
      success: (res) => {
        if (res.confirm) {
          this.setData({ messages: [], fatalError: '' })
        }
      }
    })
  },

  // 兜底 UTF-8 解码（无 TextDecoder 时）
  arrayBufferToUtf8(buf) {
    try {
      const base64 = wx.arrayBufferToBase64(buf)
      // base64 -> utf8
      const binary = atob(base64)
      // 处理 utf8 多字节
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
      return decodeURIComponent(escape(binary))
    } catch (e) {
      return ''
    }
  },

  // 滚动到底部
  scrollToBottom() {
    const idx = this._assistantIdx >= 0 ? this._assistantIdx : this.data.messages.length - 1
    if (idx < 0) return
    this.setData({
      scrollIntoView: 'msg-' + idx
    })
  },

  // 切换思考过程展开/收起
  toggleReasoning(e) {
    const idx = e.currentTarget.dataset.idx
    if (idx === undefined) return
    const msg = this.data.messages[idx]
    if (!msg) return
    msg.reasoningExpanded = !msg.reasoningExpanded
    this.setData({ ['messages[' + idx + ']']: msg })
  },

  // 防止冒泡（model picker 内点击不关闭）
  onPickerContentTap() {
    return
  }
})
