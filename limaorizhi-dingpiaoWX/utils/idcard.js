// limaorizhi-dingpiaoWX  狸猫日志售票系统  联系微信：lihao68681818  搬运或商用前麻烦先微信说一声

// 身份证号格式校验（本地校验，不调用外部API）
// 校验规则：18位、前17位数字、最后一位数字或X、校验码正确
// 返回：空字符串表示校验通过，非空字符串为错误提示
function validateIDCard(idCard) {
  if (!idCard) {
    return '请输入身份证号'
  }
  if (idCard.length !== 18) {
    return '身份证号必须是18位'
  }

  // 前17位必须为数字
  if (!/^\d{17}/.test(idCard)) {
    return '身份证号前17位必须为数字'
  }

  // 最后一位必须为数字或X
  if (!/^[\dXx]$/.test(idCard[17])) {
    return '身份证号最后一位必须为数字或X'
  }

  // 校验出生日期
  var year = parseInt(idCard.substr(6, 4))
  var month = parseInt(idCard.substr(10, 2))
  var day = parseInt(idCard.substr(12, 2))
  if (month < 1 || month > 12) {
    return '身份证号出生月份无效'
  }
  if (day < 1 || day > 31) {
    return '身份证号出生日期无效'
  }
  // 校验日期合理性
  var date = new Date(year, month - 1, day)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return '身份证号出生日期不合法'
  }
  if (date > new Date()) {
    return '身份证号出生日期不能为未来日期'
  }

  // 校验码验证（ISO 7064:1983.MOD 11-2）
  var weightFactors = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2]
  var checkCodes = ['1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2']
  var sum = 0
  for (var i = 0; i < 17; i++) {
    sum += parseInt(idCard[i]) * weightFactors[i]
  }
  var expectedCode = checkCodes[sum % 11]
  var lastChar = idCard[17].toUpperCase()
  if (lastChar !== expectedCode) {
    return '身份证号校验码错误，请检查输入'
  }

  return '' // 校验通过
}

module.exports = { validateIDCard }
