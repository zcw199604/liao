<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-[90] bg-black/40 backdrop-blur-sm flex items-center justify-center"
    @click="$emit('update:visible', false)"
  >
    <div class="w-80 bg-surface/90 ring-1 ring-line rounded-2xl p-6 shadow-2xl" @click.stop>
      <div class="text-center mb-4">
        <i class="fas fa-exclamation-triangle text-4xl text-yellow-500 mb-3"></i>
        <h3 class="text-lg font-bold text-fg mb-2">{{ title }}</h3>
        <p class="text-sm text-fg-muted">
          <slot>{{ content }}</slot>
        </p>
        <p v-if="showWarning" class="text-xs text-fg-subtle mt-1">此操作无法撤销</p>
      </div>

      <!-- 按钮 -->
      <div class="flex gap-3">
        <button
          v-if="showCancel"
          @click="handleCancel"
          class="flex-1 py-3 bg-surface-2 hover:bg-surface-hover text-fg rounded-xl border border-line transition-colors"
        >
          {{ cancelText }}
        </button>
        <button
          @click="handleConfirm"
          class="flex-1 py-3 rounded-xl font-medium text-white"
          :class="confirmButtonClass"
        >
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
  title?: string
  content?: string
  confirmText?: string
  cancelText?: string
  showCancel?: boolean
  showWarning?: boolean
  confirmButtonClass?: string
}

withDefaults(defineProps<Props>(), {
  confirmText: '确定',
  cancelText: '取消',
  showCancel: true,
  showWarning: false,
  confirmButtonClass: 'bg-red-600'
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': []
  'cancel': []
}>()

const handleConfirm = () => {
  emit('confirm')
  emit('update:visible', false)
}

const handleCancel = () => {
  emit('cancel')
  emit('update:visible', false)
}
</script>
