<template>
  <div class="space-y-6">
    <!-- 外观 -->
    <div class="bg-surface-3 rounded-xl p-4">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-fg font-medium flex items-center gap-2">
          <i class="fas fa-palette text-indigo-500"></i>
          <span>外观</span>
        </h3>
      </div>

      <div class="bg-surface rounded-lg p-3">
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div class="min-w-0">
            <div class="text-sm text-fg font-medium">主题</div>
            <div class="text-xs text-fg-muted mt-0.5">
              {{ themePreferenceLabel }}
              <span class="mx-1 text-fg/25">·</span>
              当前：{{ themeResolvedLabel }}
            </div>
          </div>

          <div class="w-full sm:w-auto grid grid-cols-3 gap-1 bg-surface-2 p-1 rounded-xl border border-line sm:flex sm:items-center">
            <button
              type="button"
              class="w-full sm:w-auto px-2.5 py-1.5 text-xs rounded-lg transition flex items-center justify-center gap-1 whitespace-nowrap"
              :class="themeStore.preference === 'auto' ? 'bg-indigo-600 text-white shadow-sm' : 'text-fg-muted hover:text-fg hover:bg-surface-hover'"
              :aria-pressed="themeStore.preference === 'auto'"
              @click="themeStore.setPreference('auto')"
              title="跟随系统"
            >
              <i class="fas fa-desktop text-[11px]"></i>
              <span>系统</span>
            </button>
            <button
              type="button"
              class="w-full sm:w-auto px-2.5 py-1.5 text-xs rounded-lg transition flex items-center justify-center gap-1 whitespace-nowrap"
              :class="themeStore.preference === 'light' ? 'bg-indigo-600 text-white shadow-sm' : 'text-fg-muted hover:text-fg hover:bg-surface-hover'"
              :aria-pressed="themeStore.preference === 'light'"
              @click="themeStore.setPreference('light')"
              title="浅色"
            >
              <i class="fas fa-sun text-[11px]"></i>
              <span>浅色</span>
            </button>
            <button
              type="button"
              class="w-full sm:w-auto px-2.5 py-1.5 text-xs rounded-lg transition flex items-center justify-center gap-1 whitespace-nowrap"
              :class="themeStore.preference === 'dark' ? 'bg-indigo-600 text-white shadow-sm' : 'text-fg-muted hover:text-fg hover:bg-surface-hover'"
              :aria-pressed="themeStore.preference === 'dark'"
              @click="themeStore.setPreference('dark')"
              title="深色"
            >
              <i class="fas fa-moon text-[11px]"></i>
              <span>深色</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="bg-surface-3 rounded-xl p-4">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-fg font-medium flex items-center gap-2">
          <i class="fas fa-network-wired text-blue-400"></i>
          <span>连接管理</span>
        </h3>
        <button
          @click="$emit('refresh')"
          class="px-3 py-1 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition flex items-center gap-1"
        >
          <i class="fas fa-sync-alt"></i>
          <span>刷新</span>
        </button>
      </div>

      <div class="space-y-3">
        <!-- 连接统计 -->
        <div class="bg-surface rounded-lg p-3 space-y-2">
          <div class="flex justify-between items-center text-sm">
            <span class="text-fg-muted">总连接数</span>
            <span class="text-fg font-mono">{{ stats.active || 0 }}</span>
          </div>
          <div class="flex justify-between items-center text-sm">
            <span class="text-fg-muted">上游连接</span>
            <span class="text-green-400 font-mono">{{ stats.upstream || 0 }}</span>
          </div>
          <div class="flex justify-between items-center text-sm">
            <span class="text-fg-muted">下游连接</span>
            <span class="text-blue-400 font-mono">{{ stats.downstream || 0 }}</span>
          </div>
        </div>

        <!-- 断开所有连接按钮 -->
        <button
          @click="$emit('disconnectAll')"
          :disabled="loading"
          class="w-full py-3 bg-red-600 hover:bg-red-700 disabled:bg-gray-600 text-white rounded-xl font-medium transition flex items-center justify-center gap-2"
        >
          <i class="fas fa-power-off"></i>
          <span>{{ loading ? '断开中...' : '断开所有连接' }}</span>
        </button>
        <p class="text-xs text-fg-subtle text-center">
          此操作会断开所有客户端和上游服务器的连接
        </p>

        <!-- Forceout用户管理 -->
        <div v-if="forceoutCount > 0" class="bg-amber-500/10 border border-amber-500/30 rounded-lg p-3">
          <div class="flex items-center gap-2 mb-2 text-amber-500">
            <i class="fas fa-exclamation-triangle"></i>
            <span class="text-sm font-medium">当前有 {{ forceoutCount }} 个用户被禁止连接（5分钟限制）</span>
          </div>
          <button
            @click="$emit('clearForceout')"
            class="w-full py-2 bg-amber-600 hover:bg-amber-700 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
          >
            <i class="fas fa-unlock"></i>
            <span>清除所有禁止连接的用户</span>
          </button>
        </div>
        <div v-else class="text-center text-green-400 text-sm flex items-center justify-center gap-2">
          <i class="fas fa-check-circle"></i>
          <span>当前没有被禁止的用户</span>
        </div>
      </div>
    </div>

    <div class="bg-surface-3 rounded-xl p-4">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-fg font-medium flex items-center gap-2">
          <i class="fas fa-image text-purple-400"></i>
          <span>图片端口策略</span>
        </h3>
        <button
          @click="saveImagePortConfig"
          :disabled="savingConfig"
          class="px-3 py-1 text-sm bg-emerald-600 hover:bg-emerald-700 disabled:bg-gray-600 text-white rounded-lg transition flex items-center gap-1"
        >
          <i class="fas fa-save"></i>
          <span>{{ savingConfig ? '保存中...' : '保存' }}</span>
        </button>
      </div>

      <div class="space-y-3">
        <div class="bg-surface rounded-lg p-3 space-y-3">
          <div class="flex items-center justify-between text-sm">
            <span class="text-fg-muted">模式（全局共用）</span>
            <select
              v-model="draftMode"
              class="bg-surface-3 text-fg text-sm rounded-lg px-2 py-1 border border-line-strong focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              <option value="fixed">固定</option>
              <option value="probe">可用端口探测</option>
              <option value="real">真实图片请求</option>
            </select>
          </div>

          <div class="flex items-center justify-between text-sm">
            <span class="text-fg-muted">固定图片端口</span>
            <input
              v-model="draftFixedPort"
              type="number"
              inputmode="numeric"
              class="w-24 bg-surface-3 text-fg text-sm rounded-lg px-2 py-1 border border-line-strong focus:outline-none focus:ring-2 focus:ring-purple-500 text-right"
              placeholder="9006"
            />
          </div>

          <div v-if="draftMode === 'real'" class="flex items-center justify-between text-sm">
            <span class="text-fg-muted">最小字节阈值</span>
            <input
              v-model.number="draftRealMinBytes"
              type="number"
              inputmode="numeric"
              class="w-24 bg-surface-3 text-fg text-sm rounded-lg px-2 py-1 border border-line-strong focus:outline-none focus:ring-2 focus:ring-purple-500 text-right"
              placeholder="2048"
              min="256"
              max="65536"
            />
          </div>
        </div>

        <p class="text-xs text-fg-subtle leading-relaxed">
          视频端口仍保持现有固定逻辑；仅图片按上述策略解析。真实图片请求会对候选端口发起小范围读取并按阈值判定，首次可能稍慢。
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { ConnectionStats } from '@/types'
import type { ImagePortMode } from '@/types'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useToast } from '@/composables/useToast'
import { useThemeStore } from '@/stores/theme'

interface Props {
  stats: ConnectionStats
  forceoutCount: number
  loading: boolean
}

defineProps<Props>()
defineEmits<{
  'refresh': []
  'disconnectAll': []
  'clearForceout': []
}>()

const systemConfigStore = useSystemConfigStore()
const { show } = useToast()
const themeStore = useThemeStore()

const draftMode = ref<ImagePortMode>('fixed')
const draftFixedPort = ref('9006')
const draftRealMinBytes = ref(2048)

const savingConfig = computed(() => systemConfigStore.saving)
const themePreferenceLabel = computed(() => {
  if (themeStore.preference === 'auto') return '跟随系统'
  if (themeStore.preference === 'light') return '浅色'
  return '深色'
})
const themeResolvedLabel = computed(() => (themeStore.resolved === 'dark' ? '深色' : '浅色'))

onMounted(async () => {
  await systemConfigStore.loadSystemConfig()
  draftMode.value = systemConfigStore.imagePortMode
  draftFixedPort.value = String(systemConfigStore.imagePortFixed || '9006')
  draftRealMinBytes.value = Number(systemConfigStore.imagePortRealMinBytes || 2048)
})

const saveImagePortConfig = async () => {
  const ok = await systemConfigStore.saveSystemConfig({
    imagePortMode: draftMode.value,
    imagePortFixed: String(draftFixedPort.value || '9006'),
    imagePortRealMinBytes: Number(draftRealMinBytes.value || 2048)
  })
  show(ok ? '已保存' : '保存失败')
}
</script>
