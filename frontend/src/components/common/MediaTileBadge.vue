<template>
  <span
    class="px-1.5 py-0.5 text-[10px] leading-tight font-medium tracking-wide rounded border backdrop-blur-sm select-none"
    :class="[variantClass, interactive ? 'pointer-events-auto' : 'pointer-events-none']"
    v-bind="$attrs"
  >
    <slot />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type Variant = 'neutral' | 'success' | 'info' | 'warn' | 'danger'

const props = withDefaults(defineProps<{
  variant?: Variant
  interactive?: boolean
}>(), {
  variant: 'neutral',
  interactive: false
})

const variantClass = computed(() => {
  switch (props.variant) {
    case 'success':
      return 'bg-emerald-600/80 text-white border-emerald-400/30'
    case 'info':
      return 'bg-indigo-600/80 text-white border-indigo-400/30'
    case 'warn':
      return 'bg-amber-600/80 text-white border-amber-400/30'
    case 'danger':
      return 'bg-red-600/80 text-white border-red-400/30'
    default:
      return 'bg-black/50 text-white/90 border-white/10'
  }
})
</script>
