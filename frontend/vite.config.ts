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
      // Keep json-summary for quick CI summary, and json for per-file branch analysis.
      reporter: ['text', 'json-summary', 'json'],
      reportsDirectory: 'coverage',
      // NOTE: V8 coverage for Vue SFC templates contains branch slots that are not coverable
      // (e.g. branch counts may be `null`). Excluding templates keeps branch coverage stable.
      exclude: ['src/**/*.vue'],
      thresholds: {
        branches: 99
      }
    }
  }
})
