// 身份卡片背景颜色类
export const colorClasses: string[] = [
  'bg-color-1',
  'bg-color-2',
  'bg-color-3',
  'bg-color-4',
  'bg-color-5'
]

// 根据ID获取颜色类
export const getColorClass = (id: string): string => {
  if (!id || id.length < 2) return 'bg-color-1'
  const hash = id.charCodeAt(0) + id.charCodeAt(1)
  const index = hash % colorClasses.length
  return colorClasses[index] || 'bg-color-1'
}
