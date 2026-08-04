// 管理端AI对话(SSE流式) + 生图
const { request, BASE_URL } = require('../../utils/request')

Page({
  data: {
    adminRole: 0,
    // 对话
    messages: [],
    input: '',
    keyboardHeight: 0,
    sending: false,
    scrollIntoView: '',
    // 模型
    modelPickerVisible: false,
    providers: [],
    currentModel: '',
    currentModelName: '',
    currentProvider: '',
    modelsLoadError: false,
    modelsLoading: false,
    modelsTotal: 0,
    fatalError: '',
    // 图片识别 / 生成
    pendingImages: [],
    imageMode: false
  },

  // SSE 相关（不放 setData,省得每次渲染都带着）
  _requestTask: null,
  _sseBuffer: '',
  _utf8Tail: '',
  _decoder: null,
  _assistantIdx: -1,
  _imageTask: null,

  onLoad() {
    const userInfo = wx.getStorageSync('user_info') || {}
    const adminRole = userInfo.admin_role || 0
    // 不是管理员直接请走,省得后端拒了还白渲染一屏
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

  // 键盘高度变了就垫高输入栏
  onKeyboardHeightChange(e) {
    this.setData({ keyboardHeight: e.detail.height || 0 })
  },

  onUnload() {
    this.abortStream()
  },

  onHide() {
    // 离开页面把流断了,省得后台一直收着卡顿
    this.abortStream()
  },

  // 拉模型列表 + 当前模型
  loadModels() {
    this.setData({ modelsLoading: true })
    request({ url: '/api/wx/admin/ai/models', method: 'GET', silent: true }).then(res => {
      console.log('[admin-ai] loadModels 响应:', res)
      const d = res.data || {}
      const providers = (d.providers || []).map(p => ({
        value: p.value,
        name: p.name,
        group_name: p.group_name || '',
        has_key: !!p.has_key,
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
      // silent 模式下这里自己提示,不然用户面对空列表不知咋回事
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

  toggleModelPicker() {
    this.setData({ modelPickerVisible: !this.data.modelPickerVisible })
  },

  closeModelPicker() {
    this.setData({ modelPickerVisible: false })
  },

  reloadModels() {
    this.loadModels()
  },

  onModelSelect(e) {
    wx.vibrateShort({ type: 'light', fail: () => {} })
    // currentTarget 取不到就退而求其次用 target
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
    // 没配 key 的服务商不让选
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
      // showLoading 把 toast 压住了,这里补一个
      const msg = (err && err.message) || '切换失败，请重试'
      setTimeout(() => {
        wx.showToast({ title: msg, icon: 'none' })
      }, 100)
    })
  },

  onInput(e) {
    this.setData({ input: e.detail.value })
  },

  // 选图:相册/拍照 → 压缩 → base64,随消息一块发
  onPickImageTap() {
    if (this.data.sending) {
      wx.showToast({ title: '请等待当前回复完成', icon: 'none' })
      return
    }
    // 画图模式下点识图:先退出画图模式再选图
    if (this.data.imageMode) this.setData({ imageMode: false })
    const remain = 3 - this.data.pendingImages.length
    if (remain <= 0) {
      wx.showToast({ title: '最多同时发送3张图片', icon: 'none' })
      return
    }
    wx.chooseMedia({
      count: remain,
      mediaType: ['image'],
      sourceType: ['album', 'camera'],
      sizeType: ['compressed'],
      success: (res) => {
        this.compressImagesToBase64(res.tempFiles || [])
      }
    })
  },

  // quality 40 控体积,不然请求体太大发不出去
  compressImagesToBase64(files) {
    if (!files || files.length === 0) return
    wx.showLoading({ title: '处理图片...', mask: true })
    const that = this
    const pending = this.data.pendingImages.slice()
    const failList = []
    let remain = files.length
    const finish = () => {
      remain--
      if (remain > 0) return
      wx.hideLoading()
      if (pending.length > 0) {
        that.setData({ pendingImages: pending })
        if (failList.length > 0) {
          wx.showToast({ title: failList.length + '张图片处理失败', icon: 'none' })
        }
      } else {
        wx.showToast({ title: '图片处理失败，请重试', icon: 'none' })
      }
    }
    files.forEach(f => {
      wx.compressImage({
        src: f.tempFilePath,
        quality: 40,
        success: (c) => {
          wx.getFileSystemManager().readFile({
            filePath: c.tempFilePath,
            encoding: 'base64',
            success: (r) => {
              const mime = /\.png$/i.test(c.tempFilePath) ? 'image/png' : 'image/jpeg'
              pending.push('data:' + mime + ';base64,' + r.data)
              finish()
            },
            fail: () => { failList.push(f.tempFilePath); finish() }
          })
        },
        fail: () => { failList.push(f.tempFilePath); finish() }
      })
    })
  },

  onRemovePendingImage(e) {
    const idx = e.currentTarget.dataset.index
    const pendingImages = this.data.pendingImages.filter((_, i) => i !== idx)
    this.setData({ pendingImages })
  },

  // 画图模式:输入描述,后端生图
  onToggleImageMode() {
    if (this.data.sending) {
      wx.showToast({ title: '请等待当前回复完成', icon: 'none' })
      return
    }
    const imageMode = !this.data.imageMode
    // 进画图模式先把图清掉,免得两种输入搅一块儿
    this.setData({ imageMode, pendingImages: imageMode ? [] : this.data.pendingImages })
  },

  onPreviewImage(e) {
    const src = e.currentTarget.dataset.src
    if (!src) return
    wx.previewImage({ urls: [src], current: src })
  },

  // 长按生成图：保存到相册
  onSaveImageTap(e) {
    const src = e.currentTarget.dataset.src
    if (!src) return
    wx.showActionSheet({
      itemList: ['保存图片到相册'],
      success: () => {
        const fs = wx.getFileSystemManager()
        const mime = (src.split(';')[0] || '').split(':')[1] || 'image/png'
        const ext = mime.indexOf('jpeg') >= 0 ? 'jpg' : 'png'
        const filePath = wx.env.USER_DATA_PATH + '/ai-img-' + Date.now() + '.' + ext
        fs.writeFile({
          filePath,
          data: src.split(',')[1] || '',
          encoding: 'base64',
          success: () => {
            wx.saveImageToPhotosAlbum({
              filePath,
              success: () => wx.showToast({ title: '已保存到相册', icon: 'success' }),
              fail: () => wx.showToast({ title: '保存失败', icon: 'none' })
            })
          },
          fail: () => wx.showToast({ title: '保存失败', icon: 'none' })
        })
      }
    })
  },

  // 发送/暂停:闲时发,回复中点同一个按钮就暂停
  onSendTap() {
    if (this.data.sending) {
      this.abortStream()
      return
    }
    this.sendMessage()
  },

  // 画图走生图接口,否则走聊天接口(可带图识别)
  sendMessage() {
    if (this.data.sending) {
      console.log('[admin-ai] sending=true，忽略发送请求')
      return
    }
    const text = (this.data.input || '').trim()
    const images = this.data.pendingImages || []
    console.log('[admin-ai] 发送 文字=%s 图片数=%d imageMode=%s', text, images.length, this.data.imageMode)
    if (!text && images.length === 0) {
      // 之前是闷声 return,用户都不知道咋回事,改成明说
      wx.showToast({ title: '请先输入内容或选择图片', icon: 'none' })
      return
    }
    // 画图模式:描述直接生成图片
    if (this.data.imageMode) {
      if (!text) {
        wx.showToast({ title: '请输入图片描述', icon: 'none' })
        return
      }
      this.startGenerateImage(text)
      return
    }
    // 聊天里带生图关键词就直接生图,跟 web 端一致
    if (text && images.length === 0) {
      const imageKeywords = ['生成图片', '生成一张图', '生成图像', '画一张图', '画一幅',
        '帮我生成一张', '帮我画一张', '帮我画个', '帮我做一张', '做一张图', '做一张图片',
        '生成照片', 'create image', 'generate image', 'draw a picture']
      if (imageKeywords.some(kw => text.indexOf(kw) >= 0)) {
        this.startGenerateImage(text)
        return
      }
    }
    this.sendVisionMessage(text, images)
  },

  // 聊天消息发送(带图时 images 转多模态 content)
  sendVisionMessage(text, images) {
    // 只发图不打字时,自动补一句默认描述,模型才知道要弄啥
    const contentText = text || '请描述这张图片'
    const userMsg = { role: 'user', content: contentText, images: images }
    // 助手占位消息,流式写入
    const assistantMsg = { role: 'assistant', content: '', reasoning: '', loading: true, error: false }

    const messages = this.data.messages.concat([userMsg, assistantMsg])
    this.setData({
      messages,
      input: '',
      pendingImages: [],
      sending: true,
      fatalError: ''
    })
    this._assistantIdx = messages.length - 1
    this.scrollToBottom()

    // 给后端的 messages(去掉占位助手消息)
    const payload = {
      messages: this.data.messages.slice(0, -1).map(m => {
        const imgs = m.images || []
        if (imgs.length > 0) {
          // OpenAI 兼容多模态:文本 + 图片数组,后端自动切视觉模型
          const content = [{ type: 'text', text: m.content || '' }]
          imgs.forEach(img => content.push({ type: 'image_url', image_url: { url: img } }))
          return { role: m.role, content }
        }
        return { role: m.role, content: m.content }
      })
    }

    this.startSSE(payload)
  },

  // 生图:最长等120s,支持暂停
  startGenerateImage(prompt) {
    const userMsg = { role: 'user', content: prompt, genPrompt: true }
    const assistantMsg = { role: 'assistant', content: '', genImage: true, generating: true, error: false }

    const messages = this.data.messages.concat([userMsg, assistantMsg])
    this.setData({
      messages,
      input: '',
      imageMode: false,
      sending: true,
      fatalError: ''
    })
    this._assistantIdx = messages.length - 1
    this.scrollToBottom()

    const token = wx.getStorageSync('user_token') || ''
    const that = this
    this._imageTask = wx.request({
      url: BASE_URL + '/api/wx/admin/ai/image',
      method: 'POST',
      data: { prompt: prompt, model: this.data.currentModel || '' },
      timeout: 150000,
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? 'Bearer ' + token : ''
      },
      success(res) {
        that._imageTask = null
        const data = res.data || {}
        if (res.statusCode === 200 && data.code === 0) {
          that.finishGenerateImage(messages.length - 1, (data.data && data.data.image) || '', '')
        } else if (data.code === 1002) {
          that.finishGenerateImage(messages.length - 1, '', '登录已失效，请重新登录')
          wx.removeStorageSync('user_token')
          wx.removeStorageSync('user_info')
          setTimeout(() => wx.reLaunch({ url: '/pages/login/login' }), 800)
        } else {
          that.finishGenerateImage(messages.length - 1, '', data.message || '图片生成失败')
        }
      },
      fail(err) {
        that._imageTask = null
        // 主动暂停(abort)在 abortStream 里处理了,这儿只管网络层失败
        if (that._aborted) return
        that.finishGenerateImage(messages.length - 1, '', '网络错误，请稍后重试')
      }
    })
  },

  // 生图收尾:成功塞图 / 失败提示
  finishGenerateImage(idx, image, errMsg) {
    if (idx < 0 || idx >= this.data.messages.length) {
      this.setData({ sending: false })
      return
    }
    const msg = this.data.messages[idx]
    msg.generating = false
    if (image) {
      msg.image = image
      msg.content = '已为你生成图片'
    } else {
      msg.error = true
      msg.content = errMsg || '图片生成失败'
    }
    this.setData({ ['messages[' + idx + ']']: msg, sending: false })
    this._assistantIdx = -1
    this.scrollToBottom()
  },

  // 启动 SSE 流
  startSSE(payload) {
    const token = wx.getStorageSync('user_token') || ''
    this._sseBuffer = ''
    this._aborted = false  // 新请求开始前重置中止标志(abortStream 不自己重置,免得 fail 回调异步触发时已经被重置成 false)
    this._utf8Tail = ''

    const that = this
    this._requestTask = wx.request({
      url: BASE_URL + '/api/wx/admin/ai/chat',
      method: 'POST',
      data: payload,
      enableChunked: true,
      timeout: 300000, // 5 分钟,跟后端 client.Timeout 对齐
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? 'Bearer ' + token : '',
        'Accept': 'text/event-stream'
      },
      success(res) {
        // enableChunked 模式下数据走 onChunkReceived,这儿只处理非流式错误
        if (res.statusCode !== 200) {
          that.handleNonStreamError('服务器错误(HTTP ' + res.statusCode + ')')
          return
        }
        // 个别情况下后端直接返回 JSON 错误体(没走 SSE)
        const data = res.data
        if (data && typeof data === 'object' && data.code !== undefined) {
          if (data.code !== 0) {
            if (data.code === 1002) {
              that.handleNonStreamError('登录已失效，请重新登录')
              wx.removeStorageSync('user_token')
              wx.removeStorageSync('user_info')
              setTimeout(() => wx.reLaunch({ url: '/pages/login/login' }), 800)
              return
            }
            that.handleNonStreamError(data.message || '请求失败')
            return
          }
        }
        // 流正常走完但没等来 done,把 loading 关干净
        that.finalizeAssistant()
      },
      fail(err) {
        if (that._aborted) return
        that.handleNonStreamError('网络错误，请稍后重试')
      }
    })

    if (this._requestTask && this._requestTask.onChunkReceived) {
      this._requestTask.onChunkReceived(function (res) {
        if (!res || !res.data) return
        try {
          let text
          if (that._decoder) {
            text = that._decoder.decode(new Uint8Array(res.data), { stream: true })
          } else {
            // 兜底:用 wx.arrayBufferToBase64 + 解码(极少见的老版本)
            text = that.arrayBufferToUtf8(res.data)
          }
          that.feedSSEText(text)
        } catch (e) {
          // 这 chunk 解不开就扔了
        }
      })
    }
  },

  feedSSEText(text) {
    if (!text) return
    this._sseBuffer += text
    // 换行标准化,兼容经代理后变成 \r\n\r\n 的事件分隔
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

  // 收一个 SSE 事件块
  handleSSEBlock(block) {
    // 解析 data: 行(gin 格式:event: message\ndata: {...})
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
    // 直接改数组引用,省得频繁全量 setData
    this.data.messages[idx] = msg
    this.setData({
      ['messages[' + idx + ']']: msg
    })
    this.scrollToBottom()
  },

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
      // 要是啥都没回,给个兜底提示
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

  // 非 SSE 错误(请求层失败、AI 没启用、配置不全等)
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

  // 中断当前流(_aborted 不在这儿重置,由下次 startSSE 重置,免得 fail 回调异步触发时已被重置成 false)
  abortStream() {
    if (this._aborted && !this._requestTask && !this._imageTask) return
    this._aborted = true
    if (this._requestTask) {
      try { this._requestTask.abort() } catch (e) {}
      this._requestTask = null
    }
    if (this._imageTask) {
      try { this._imageTask.abort() } catch (e) {}
      this._imageTask = null
      // 生图占位消息:直接标停止(fail 回调因为 _aborted 不再处理)
      const idx = this._assistantIdx
      if (idx >= 0 && idx < this.data.messages.length) {
        const msg = this.data.messages[idx]
        if (msg && msg.generating) {
          msg.generating = false
          msg.error = true
          msg.content = '（已停止生成）'
          this.setData({ ['messages[' + idx + ']']: msg, sending: false })
          this._assistantIdx = -1
          return
        }
      }
    }
    // 用户主动停显示"已停止",跟"(未收到回复)"区分开
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

  // 重发上一条
  onRetryTap() {
    if (this.data.sending) return
    const msgs = this.data.messages
    if (msgs.length < 2) return
    let lastUserIdx = -1
    for (let i = msgs.length - 1; i >= 0; i--) {
      if (msgs[i].role === 'user') { lastUserIdx = i; break }
    }
    if (lastUserIdx < 0) return
    const last = msgs[lastUserIdx]
    const text = last.content || ''
    const images = last.images || []
    if (!text && images.length === 0) return
    // 把这条用户消息后头的都删了(含失败的回复)
    const kept = msgs.slice(0, lastUserIdx)
    this.setData({ messages: kept })
    this.setData({ input: text, pendingImages: images })
    // 生图失败重试:恢复画图模式重新生成
    if (last.genPrompt) {
      this.setData({ imageMode: true })
    }
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

  // 没 TextDecoder 时的 UTF-8 兜底解码
  arrayBufferToUtf8(buf) {
    try {
      const base64 = wx.arrayBufferToBase64(buf)
      const binary = atob(base64)
      // utf8 多字节字符,挨个拼回来
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
      return decodeURIComponent(escape(binary))
    } catch (e) {
      return ''
    }
  },

  scrollToBottom() {
    const idx = this._assistantIdx >= 0 ? this._assistantIdx : this.data.messages.length - 1
    if (idx < 0) return
    this.setData({
      scrollIntoView: 'msg-' + idx
    })
  },

  toggleReasoning(e) {
    const idx = e.currentTarget.dataset.idx
    if (idx === undefined) return
    const msg = this.data.messages[idx]
    if (!msg) return
    msg.reasoningExpanded = !msg.reasoningExpanded
    this.setData({ ['messages[' + idx + ']']: msg })
  },

  // 点了模型弹层别往外冒,不然一下把弹层关了
  onPickerContentTap() {
    return
  }
})
