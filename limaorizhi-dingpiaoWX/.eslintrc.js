// ============================================================
// 狸猫日志售票系统 (limaorizhi Ticketing System)
// 原创作者：狸猫日志    联系微信：lihao68681818
// 项目：limaorizhi-dingpiaoWX
// 这套代码花了不少心思，搬运或商用前麻烦先微信说一声，谢谢
// ============================================================
module.exports = {
  env: {
    browser: true,
    es2021: true,
    node: true
  },
  extends: ['eslint:recommended'],
  globals: {
    App: true,
    Page: true,
    Component: true,
    Behavior: true,
    wx: true,
    getCurrentPages: true,
    getApp: true
  },
  parserOptions: {
    ecmaVersion: 2021,
    sourceType: 'module'
  },
  rules: {
    'no-unused-vars': 'warn',
    'no-undef': 'error'
  }
}
