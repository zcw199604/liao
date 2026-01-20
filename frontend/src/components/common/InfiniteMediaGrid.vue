<template>
  <div
    ref="scrollContainer"
    class="flex-1 overflow-y-auto p-2 no-scrollbar"
    @scroll="handleScroll"
  >
    <!-- Loading Initial -->
    <div
      v-if="loading && items.length === 0"
      class="flex h-full items-center justify-center"
    >
      <div class="text-center">
        <div class="radar-spinner mx-auto mb-3"></div>
        <p class="text-gray-500 text-sm">加载中...</p>
      </div>
    </div>

    <!-- Content -->
    <template v-else-if="items.length > 0">
      <!-- Masonry Layout (JS Calculated) -->
      <div v-if="layoutMode === 'masonry'" class="flex gap-2 items-start">
        <div
          v-for="(colItems, colIndex) in masonryColumns"
          :key="colIndex"
          class="flex-1 flex flex-col gap-2"
        >
          <div
            v-for="{ data: item, originalIndex } in colItems"
            :key="getItemKey(item, originalIndex)"
            class="relative group w-full will-change-transform"
          >
            <slot :item="item" :index="originalIndex"></slot>
          </div>
        </div>
      </div>

      <!-- Grid Layout (CSS Grid) -->
      <div v-else :class="gridClass">
        <div
          v-for="(item, index) in items"
          :key="getItemKey(item, index)"
          class="relative group"
          :class="itemClass"
        >
          <slot :item="item" :index="index"></slot>
        </div>
      </div>

      <!-- Load More / Finished -->
      <div v-if="loading" class="flex justify-center py-6 text-gray-500 text-sm w-full">
        <div class="flex items-center gap-2 bg-[#27272a] px-4 py-2 rounded-full shadow-lg">
          <span class="w-4 h-4 border-2 border-purple-500 border-t-transparent rounded-full animate-spin"></span>
          <span>加载更多...</span>
        </div>
      </div>

      <div
        v-else-if="finished"
        class="flex justify-center py-8 text-gray-600 text-xs w-full"
      >
        <span class="px-3 py-1 bg-[#27272a]/50 rounded-full">
           <slot name="finished-text">已加载全部 {{ total ? total + ' 个' : '' }}</slot>
        </span>
      </div>
    </template>

    <!-- Empty -->
    <div v-else class="flex h-full items-center justify-center">
       <slot name="empty">
          <div class="text-center text-gray-500">
            <i class="fas fa-image text-5xl mb-4 opacity-30"></i>
            <p>暂无数据</p>
          </div>
       </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useBreakpoints, breakpointsTailwind } from '@vueuse/core'

const props = withDefaults(defineProps<{
  items: any[]
  loading: boolean
  finished: boolean
  layoutMode?: 'grid' | 'masonry'
  itemKey?: string | ((item: any, index: number) => string | number)
  total?: number
}>(), {
  layoutMode: 'grid',
  items: () => []
})

const emit = defineEmits<{
  (e: 'loadMore'): void
}>()

const scrollContainer = ref<HTMLElement | null>(null)

// 响应式断点
const breakpoints = useBreakpoints(breakpointsTailwind)

// 根据屏幕宽度自动计算列数
const columnCount = computed(() => {
  if (breakpoints.xl.value) return 4
  if (breakpoints.lg.value) return 3
  if (breakpoints.md.value) return 2
  return 2 // 移动端默认 2 列
})

// JS 计算的瀑布流列数据
const masonryColumns = computed(() => {
  if (props.layoutMode !== 'masonry') return []
  
  const count = columnCount.value
  // 存储包装对象 { data: item, originalIndex: number }
  const result: { data: any, originalIndex: number }[][] = Array.from({ length: count }, () => [])
  const heights = Array(count).fill(0)
  
  props.items.forEach((item, index) => {
    // 找到当前最矮的列
    let minHeightIndex = 0
    let minHeight = heights[0]
    
    for (let i = 1; i < count; i++) {
      if (heights[i] < minHeight) {
        minHeight = heights[i]
        minHeightIndex = i
      }
    }
    
    // 分配项目
    const targetCol = result[minHeightIndex]
    if (targetCol) {
      targetCol.push({ data: item, originalIndex: index })
    }
    
    // 累加高度：使用宽高比计算占位高度
    // 如果没有宽高，默认给个 1 (正方形)
    // 注意：item.width 和 item.height 可能是字符串或数字
    const w = Number(item.width)
    const h = Number(item.height)
    const ratio = (w && h) ? h / w : 1
    
    // 累加高度 (加上间距因子，这里简化为高度比)
    heights[minHeightIndex] += ratio
  })
  
  return result
})

const handleScroll = () => {
  const el = scrollContainer.value
  if (!el) return

  const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 120
  if (!nearBottom) return

  if (props.loading || props.finished) return
  
  emit('loadMore')
}

const gridClass = computed(() => {
  // 仅在 Grid 模式下使用
  return 'grid grid-cols-3 sm:grid-cols-4 gap-2'
})

const itemClass = computed(() => {
   // 仅在 Grid 模式下使用
   return 'aspect-square'
})

const getItemKey = (item: any, index: number) => {
  if (typeof props.itemKey === 'function') {
    return props.itemKey(item, index)
  }
  if (props.itemKey && item[props.itemKey]) {
    return item[props.itemKey]
  }
  // 尝试使用常见的唯一标识字段，最后使用索引
  return item.id ?? item.md5 ?? index
}
</script>
