<template>
  <router-view />
  <AllUploadImageModal />
</template>

<script setup lang="ts">
// 根组件：所有路由页面的容器
import AllUploadImageModal from '@/components/media/AllUploadImageModal.vue'
</script>

<style>
/* 全局样式重置 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  width: 100%;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
}
</style>
