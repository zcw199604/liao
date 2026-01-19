<template>
  <div
    ref="scrollContainer"
    class="flex-1 overflow-y-auto p-6 no-scrollbar"
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
      <div :class="gridClass">
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

const handleScroll = () => {
  const el = scrollContainer.value
  if (!el) return

  const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 120
  if (!nearBottom) return

  if (props.loading || props.finished) return
  
  emit('loadMore')
}

const gridClass = computed(() => {
  if (props.layoutMode === 'masonry') {
    // 使用 CSS 多列布局实现真正的瀑布流：columns-* 设置列数，gap-4 设置列间距
    return 'columns-2 md:columns-3 lg:columns-4 gap-4'
  }
  return 'grid grid-cols-3 sm:grid-cols-4 gap-4'
})

const itemClass = computed(() => {
   if (props.layoutMode === 'masonry') {
     // break-inside-avoid: 防止图片被切成两半
     // mb-4: 定义垂直间距（因为 gap 只管列间距）
     // w-full: 确保占满列宽
     // will-change-transform: 防止渲染闪烁
     return 'break-inside-avoid mb-4 w-full will-change-transform'
   }
   return 'aspect-square'
})

const getItemKey = (item: any, index: number) => {
  if (typeof props.itemKey === 'function') {
    return props.itemKey(item, index)
  }
  if (props.itemKey && item[props.itemKey]) {
    return item[props.itemKey]
  }
  return index
}
</script>
