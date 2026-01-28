<template>
  <div class="space-y-6">
    <div class="bg-[#27272a] rounded-xl p-4">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-white font-medium flex items-center gap-2">
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
        <div class="bg-[#18181b] rounded-lg p-3 space-y-2">
          <div class="flex justify-between items-center text-sm">
            <span class="text-gray-400">总连接数</span>
            <span class="text-white font-mono">{{ stats.active || 0 }}</span>
          </div>
          <div class="flex justify-between items-center text-sm">
            <span class="text-gray-400">上游连接</span>
            <span class="text-green-400 font-mono">{{ stats.upstream || 0 }}</span>
          </div>
          <div class="flex justify-between items-center text-sm">
            <span class="text-gray-400">下游连接</span>
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
        <p class="text-xs text-gray-500 text-center">
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

    <div class="bg-[#27272a] rounded-xl p-4">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-white font-medium flex items-center gap-2">
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
        <div class="bg-[#18181b] rounded-lg p-3 space-y-3">
          <div class="flex items-center justify-between text-sm">
            <span class="text-gray-400">模式（全局共用）</span>
            <select
              v-model="draftMode"
              class="bg-[#27272a] text-white text-sm rounded-lg px-2 py-1 border border-white/10 focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              <option value="fixed">固定</option>
              <option value="probe">可用端口探测</option>
              <option value="real">真实图片请求</option>
            </select>
          </div>

          <div class="flex items-center justify-between text-sm">
            <span class="text-gray-400">固定图片端口</span>
            <input
              v-model="draftFixedPort"
              type="number"
              inputmode="numeric"
              class="w-24 bg-[#27272a] text-white text-sm rounded-lg px-2 py-1 border border-white/10 focus:outline-none focus:ring-2 focus:ring-purple-500 text-right"
              placeholder="9006"
            />
          </div>

          <div v-if="draftMode === 'real'" class="flex items-center justify-between text-sm">
            <span class="text-gray-400">最小字节阈值</span>
            <input
              v-model.number="draftRealMinBytes"
              type="number"
              inputmode="numeric"
              class="w-24 bg-[#27272a] text-white text-sm rounded-lg px-2 py-1 border border-white/10 focus:outline-none focus:ring-2 focus:ring-purple-500 text-right"
              placeholder="2048"
              min="256"
              max="65536"
            />
          </div>
        </div>

        <p class="text-xs text-gray-500 leading-relaxed">
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

const draftMode = ref<ImagePortMode>('fixed')
const draftFixedPort = ref('9006')
const draftRealMinBytes = ref(2048)

const savingConfig = computed(() => systemConfigStore.saving)

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
