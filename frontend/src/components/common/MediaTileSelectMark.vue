<template>
  <component
    :is="interactive ? 'button' : 'div'"
    :type="interactive ? 'button' : undefined"
    class="touch-manipulation select-none flex items-center justify-center rounded-full"
    :class="[hitSizeClass, interactive ? '' : 'pointer-events-none']"
    @click="handleClick"
  >
    <span
      class="flex items-center justify-center rounded-full border shadow-sm backdrop-blur-md transition-colors"
      :class="[innerSizeClass, checked ? checkedClass : uncheckedClass]"
    >
      <i v-if="checked" class="fas fa-check text-white" :class="iconSizeClass"></i>
    </span>
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type Size = 'sm' | 'md' | 'lg'
type Tone = 'purple' | 'emerald' | 'indigo'

const props = withDefaults(defineProps<{
  checked: boolean
  interactive?: boolean
  size?: Size
  tone?: Tone
}>(), {
  interactive: false,
  size: 'md',
  tone: 'indigo'
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
      return 'w-6 h-6'
    case 'lg':
      return 'w-9 h-9'
    default:
      return 'w-7 h-7'
  }
})

const iconSizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'text-[10px]'
    case 'lg':
      return 'text-sm'
    default:
      return 'text-xs'
  }
})

const checkedClass = computed(() => {
  switch (props.tone) {
    case 'purple':
      return 'bg-purple-500/80 border-purple-300/50'
    case 'emerald':
      return 'bg-emerald-500/80 border-emerald-300/50'
    default:
      return 'bg-indigo-500/80 border-indigo-300/50'
  }
})

const uncheckedClass = computed(() => {
  return 'bg-black/30 border-white/40'
})

const handleClick = (e: MouseEvent) => {
  if (!props.interactive) return
  e.stopPropagation()
  emit('click', e)
}
</script>
