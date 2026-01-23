<template>
  <button
    type="button"
    class="group touch-manipulation select-none rounded-full flex items-center justify-center disabled:opacity-60 disabled:cursor-not-allowed"
    :class="hitSizeClass"
    :disabled="disabled"
    v-bind="$attrs"
    @click.stop="handleClick"
  >
    <span
      class="flex items-center justify-center rounded-full transition-all duration-150 border shadow-sm"
      :class="[innerSizeClass, surfaceClass, toneClass]"
    >
      <slot />
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type Size = 'sm' | 'md' | 'lg'
type Tone = 'neutral' | 'danger' | 'primary'

const props = withDefaults(defineProps<{
  size?: Size
  tone?: Tone
  disabled?: boolean
}>(), {
  size: 'md',
  tone: 'neutral',
  disabled: false
})

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

const hitSizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'w-9 h-9'
    case 'lg':
      return 'w-12 h-12'
    default:
      return 'w-10 h-10'
  }
})

const innerSizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'w-7 h-7'
    case 'lg':
      return 'w-10 h-10'
    default:
      return 'w-8 h-8'
  }
})

const surfaceClass = computed(() => {
  return 'bg-black/40 backdrop-blur-md border-white/10 group-hover:bg-black/60 group-active:bg-black/70 group-active:scale-95'
})

const toneClass = computed(() => {
  switch (props.tone) {
    case 'danger':
      return 'text-red-300 group-hover:text-red-200'
    case 'primary':
      return 'text-indigo-200 group-hover:text-indigo-100'
    default:
      return 'text-white/90 group-hover:text-white'
  }
})

const handleClick = (e: MouseEvent) => {
  if (props.disabled) return
  emit('click', e)
}
</script>
