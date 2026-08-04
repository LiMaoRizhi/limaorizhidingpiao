// 拼音切分与首字母提取工具
// 用途：站点搜索联想时，支持"中文 / 全拼 / 首字母"三种输入匹配
// 例：输入"青田"、"qingtian"、"qtnz" 均可匹配到站点"青田镇"

// 汉语拼音有效音节表（无声调，小写），用于对全拼连写做贪心最长匹配切分
const PINYIN_SYLLABLES = [
  'a','ai','an','ang','ao','e','ei','en','eng','er','o','ou',
  'ba','bai','ban','bang','bao','bei','ben','beng','bi','bian','biao','bie','bin','bing','bo','bu',
  'pa','pai','pan','pang','pao','pei','pen','peng','pi','pian','piao','pie','pin','ping','po','pu',
  'ma','mai','man','mang','mao','me','mei','men','meng','mi','mian','miao','mie','min','ming','mo','mou','mu',
  'fa','fan','fang','fei','fen','feng','fo','fou','fu',
  'da','dai','dan','dang','dao','de','dei','den','deng','di','dia','dian','diao','die','ding','diu','dong','dou','du','dui','dun','duo',
  'ta','tai','tan','tang','tao','te','teng','ti','tian','tiao','tie','ting','tong','tou','tu','tui','tun','tuo',
  'na','nai','nan','nang','nao','ne','nei','nen','neng','ni','nian','niao','nie','nin','ning','niu','nong','nou','nu','nuan','nuo','nv','nve',
  'la','lai','lan','lang','lao','le','lei','leng','li','lia','lian','liao','lie','lin','ling','liu','long','lou','lu','luan','lun','luo','lv','lve',
  'ga','gai','gan','gang','gao','ge','gei','gen','geng','gong','gou','gu','gua','guai','guan','guang','gui','gun','guo',
  'ka','kai','kan','kang','kao','ke','ken','keng','kong','kou','ku','kua','kuai','kuan','kuang','kui','kun','kuo',
  'ha','hai','han','hang','hao','he','hei','hen','heng','hong','hou','hu','hua','huai','huan','huang','hui','hun','huo',
  'ji','jia','jian','jiang','jiao','jie','jin','jing','jiong','jiu','ju','juan','jue','jun',
  'qi','qia','qian','qiang','qiao','qie','qin','qing','qiong','qiu','qu','quan','que','qun',
  'xi','xia','xian','xiang','xiao','xie','xin','xing','xiong','xiu','xu','xuan','xue','xun',
  'ya','yan','yang','yao','ye','yi','yin','ying','yo','yong','you','yu','yuan','yue','yun',
  'wa','wai','wan','wang','wei','wen','weng','wo','wu',
  'za','zai','zan','zang','zao','ze','zei','zen','zeng','zi','zong','zou','zu','zuan','zui','zun','zuo',
  'ca','cai','can','cang','cao','ce','cen','ceng','ci','cong','cou','cu','cuan','cui','cun','cuo',
  'sa','sai','san','sang','sao','se','sen','seng','si','song','sou','su','suan','sui','sun','suo',
  'zha','zhai','zhan','zhang','zhao','zhe','zhei','zhen','zheng','zhi','zhong','zhou','zhu','zhua','zhuai','zhuan','zhuang','zhui','zhun','zhuo',
  'cha','chai','chan','chang','chao','che','chen','cheng','chi','chong','chou','chu','chua','chuai','chuan','chuang','chui','chun','chuo',
  'sha','shai','shan','shang','shao','she','shei','shen','sheng','shi','shou','shu','shua','shuai','shuan','shuang','shui','shun','shuo',
  'n','m','ng','hm','hng','huh'
]

const SYLLABLE_SET = {}
for (let i = 0; i < PINYIN_SYLLABLES.length; i++) {
  SYLLABLE_SET[PINYIN_SYLLABLES[i]] = true
}

// 音节最大长度（如 zhuang/chuang/shuang 为 6）
const MAX_SYLLABLE_LEN = 6

// 将全拼连写切分为音节数组（贪心最长匹配）
function splitPinyin(pinyinStr) {
  if (!pinyinStr) return []
  const s = pinyinStr.toLowerCase()
  const result = []
  let i = 0
  while (i < s.length) {
    let matched = false
    for (let len = MAX_SYLLABLE_LEN; len >= 1; len--) {
      if (i + len > s.length) continue
      const sub = s.slice(i, i + len)
      if (SYLLABLE_SET[sub]) {
        result.push(sub)
        i += len
        matched = true
        break
      }
    }
    if (!matched) {
      // 无法匹配的字符（录入异常），单独保留原字符，避免卡死
      result.push(s[i])
      i++
    }
  }
  return result
}

// 提取首字母序列（如 "qingtianzhen" -> "qtz"）
function getInitials(pinyinStr) {
  const syllables = splitPinyin(pinyinStr)
  let initials = ''
  for (let i = 0; i < syllables.length; i++) {
    initials += syllables[i][0]
  }
  return initials
}

// 站点匹配：中文包含 / 全拼包含 / 首字母序列包含
function matchStation(station, keyword) {
  if (!keyword) return true
  const kw = keyword.trim().toLowerCase()
  if (!kw) return true
  const name = (station.name || '').toLowerCase()
  if (name.indexOf(kw) >= 0) return true
  const pinyin = (station.pinyin || '').toLowerCase()
  if (pinyin.indexOf(kw) >= 0) return true
  const initials = getInitials(pinyin)
  if (initials.indexOf(kw) >= 0) return true
  return false
}

module.exports = {
  matchStation: matchStation
}
