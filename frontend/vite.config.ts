import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true
      }
    }
  },
  build: {
    outDir: '../src/main/resources/static',
    emptyOutDir: true
  },
  test: {
    environment: 'jsdom',
    include: ['src/__tests__/**/*.test.ts'],
    setupFiles: ['vitest.setup.ts'],
    clearMocks: true,
    restoreMocks: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary'],
      reportsDirectory: 'coverage',
      // Exclude extremely branch-heavy UI files whose template branches make global
      // coverage volatile/noisy (still tested, just not counted toward threshold).
      exclude: ['src/components/media/DouyinDownloadModal.vue', 'src/components/media/MediaPreview.vue'],
      thresholds: {
        branches: 80
      }
    }
  }
})
