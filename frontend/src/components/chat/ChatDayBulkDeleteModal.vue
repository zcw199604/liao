<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-[90] bg-black/40 backdrop-blur-sm flex items-center justify-center p-4"
    @click="handleClose"
  >
    <div class="w-full max-w-md bg-surface ring-1 ring-line rounded-2xl shadow-2xl overflow-hidden" @click.stop>
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-line">
        <div class="flex items-center gap-2">
          <i class="fas fa-calendar-alt text-blue-400"></i>
          <h3 class="text-base font-bold text-fg">按天选择会话</h3>
        </div>
        <button
          @click="handleClose"
          class="w-8 h-8 flex items-center justify-center text-fg-muted hover:text-fg transition rounded-lg hover:bg-surface-3"
          aria-label="close"
        >
          <i class="fas fa-times"></i>
        </button>
      </div>

      <!-- Body -->
      <div class="p-4 max-h-[60vh] overflow-y-auto no-scrollbar">
        <div v-if="items.length === 0" class="text-center text-fg-subtle py-10">
          <i class="fas fa-calendar-times text-4xl mb-3 opacity-30"></i>
          <p class="text-sm">暂无可选择的日期</p>
        </div>

        <div v-else class="space-y-2">
          <button
            v-for="item in items"
            :key="item.key"
            @click="toggle(item.key)"
            class="w-full flex items-center gap-3 px-3 py-3 rounded-xl border border-line bg-surface-2 hover:bg-surface-hover transition text-left"
          >
            <div
              class="w-6 h-6 rounded-full border border-line-strong flex items-center justify-center shrink-0"
              :class="isSelected(item.key) ? 'bg-blue-600 border-blue-500 text-white' : 'bg-surface-3 text-fg-muted'"
            >
              <i v-if="isSelected(item.key)" class="fas fa-check text-xs"></i>
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium text-fg truncate">{{ item.label }}</div>
              <div class="text-xs text-fg-subtle">{{ item.count }} 个会话</div>
            </div>
            <div class="text-xs text-fg-muted shrink-0">{{ item.count }}</div>
          </button>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-4 py-4 border-t border-line bg-surface-2 flex items-center justify-between gap-2">
        <button
          @click="toggleAll"
          class="px-3 py-2 bg-surface-3 text-fg rounded-lg hover:bg-surface-hover transition text-sm border border-line"
          :disabled="items.length === 0"
        >
          {{ isAllSelected ? '取消全选' : '全选' }}
        </button>

        <div class="flex items-center gap-2">
          <button
            @click="handleClose"
            class="px-3 py-2 bg-surface-3 text-fg rounded-lg hover:bg-surface-hover transition text-sm border border-line"
          >
            取消
          </button>
          <button
            @click="handleConfirm"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition text-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            :disabled="selectedKeys.length === 0"
          >
            <i class="fas fa-check"></i>
            <span>选中会话 ({{ selectedTotalCount }})</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

type Item = {
  key: string
  label: string
  count: number
}

const props = withDefaults(
  defineProps<{
    visible: boolean
    items: Item[]
    preselectKey?: string
  }>(),
  {
    items: () => []
  }
)

const emit = defineEmits<{
  'update:visible': [value: boolean]
  confirm: [keys: string[]]
}>()

const selectedKeys = ref<string[]>([])

watch(
  () => props.visible,
  visible => {
    if (!visible) return
    if (props.preselectKey) {
      selectedKeys.value = [props.preselectKey]
    } else {
      selectedKeys.value = []
    }
  }
)

const isSelected = (key: string) => selectedKeys.value.includes(key)

const isAllSelected = computed(() => {
  if (props.items.length === 0) return false
  return selectedKeys.value.length === props.items.length
})

const selectedTotalCount = computed(() => {
  const selected = new Set(selectedKeys.value)
  return props.items.reduce((sum, item) => {
    if (!selected.has(item.key)) return sum
    return sum + (item.count || 0)
  }, 0)
})

const toggle = (key: string) => {
  if (isSelected(key)) {
    selectedKeys.value = selectedKeys.value.filter(k => k !== key)
    return
  }
  selectedKeys.value = [...selectedKeys.value, key]
}

const toggleAll = () => {
  if (isAllSelected.value) {
    selectedKeys.value = []
    return
  }
  selectedKeys.value = props.items.map(i => i.key)
}

const handleClose = () => {
  emit('update:visible', false)
}

const handleConfirm = () => {
  if (selectedKeys.value.length === 0) return
  emit('confirm', [...selectedKeys.value])
  emit('update:visible', false)
}
</script>
