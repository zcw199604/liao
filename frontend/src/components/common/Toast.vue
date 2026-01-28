<template>
  <teleport to="body">
    <transition name="fade">
      <div
        v-if="visible"
        class="fixed top-16 left-1/2 -translate-x-1/2 z-[300] px-4 py-2 bg-zinc-800/90 backdrop-blur-md border border-white/10 text-white/90 text-sm rounded-full shadow-lg shadow-black/50"
      >
        {{ message }}
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { message, visible } = useToast()
</script>

<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
