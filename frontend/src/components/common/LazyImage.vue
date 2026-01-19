<template>
  <div class="relative overflow-hidden bg-gray-800" :class="containerClass">
    <img
      v-bind="$attrs"
      :src="src"
      :alt="alt"
      loading="lazy"
      class="w-full h-full object-cover transition-opacity duration-500"
      :class="[loaded ? 'opacity-100' : 'opacity-0', imgClass]"
      @load="onLoad"
      @error="onError"
    />
    <div
      v-if="!loaded && !error"
      class="absolute inset-0 flex items-center justify-center bg-[#27272a]"
    >
      <i class="fas fa-image text-gray-700 text-2xl animate-pulse"></i>
    </div>
    <div
        v-if="error"
        class="absolute inset-0 flex items-center justify-center bg-[#27272a] text-gray-500"
    >
        <i class="fas fa-exclamation-triangle"></i>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  src: string
  alt?: string
  containerClass?: string
  imgClass?: string
}>()

const loaded = ref(false)
const error = ref(false)

const onLoad = () => {
  loaded.value = true
}

const onError = () => {
  error.value = true
  loaded.value = true // Show placeholder/error state
}

watch(() => props.src, () => {
  loaded.value = false
  error.value = false
})
</script>
