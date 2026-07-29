/** 格式化ISO时间为 yyyy-MM-dd HH:mm:ss */
export const formatTime = (t?: string): string => {
  if (!t) return '-'
  return t.replace('T', ' ').substring(0, 19)
}
