<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-[80] bg-black/40 backdrop-blur-sm"
    @click="$emit('update:visible', false)"
  >
    <div class="page-container justify-center items-center overflow-y-auto p-4">
      <div class="w-full max-w-xs bg-surface/90 ring-1 ring-line rounded-2xl p-6 shadow-2xl" @click.stop>
        <h3 class="text-lg font-bold text-fg mb-4 text-center">创建新身份</h3>

        <div class="space-y-4">
          <!-- 名字输入 -->
          <div>
            <label class="text-xs text-fg-subtle mb-1 block">名字</label>
            <input
              v-model="formData.name"
              type="text"
              placeholder="输入名字"
              class="w-full bg-surface-2 text-fg px-4 py-3 rounded-xl border border-line-strong focus:border-line-strong focus:ring-1 focus:ring-indigo-500/50 focus:bg-surface-hover focus:outline-none transition placeholder-fg-subtle"
            />
          </div>

          <!-- 性别选择 -->
          <div>
            <label class="text-xs text-fg-subtle mb-1 block">性别</label>
            <div class="flex gap-3">
              <button
                @click="formData.sex = '男'"
                :class="formData.sex === '男' ? 'bg-blue-600 border-blue-600 text-white' : 'bg-surface-2 border-line text-fg hover:bg-surface-hover'"
                class="flex-1 py-3 rounded-xl border font-medium transition-colors"
              >
                <i class="fas fa-mars mr-2"></i>男
              </button>
              <button
                @click="formData.sex = '女'"
                :class="formData.sex === '女' ? 'bg-pink-600 border-pink-600 text-white' : 'bg-surface-2 border-line text-fg hover:bg-surface-hover'"
                class="flex-1 py-3 rounded-xl border font-medium transition-colors"
              >
                <i class="fas fa-venus mr-2"></i>女
              </button>
            </div>
          </div>
        </div>

        <!-- 按钮 -->
        <div class="flex gap-3 mt-6">
          <button
            @click="$emit('update:visible', false)"
            class="flex-1 py-3 bg-surface-2 hover:bg-surface-hover text-fg rounded-xl border border-line transition-colors"
          >
            取消
          </button>
          <button
            @click="handleConfirm"
            :disabled="!formData.name || !formData.sex"
            class="flex-1 py-3 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-[inset_0_1px_0_rgba(255,255,255,0.2)]"
          >
            创建
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface Props {
  visible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'created': [data: { name: string; sex: string }]
}>()

const formData = ref({
  name: '',
  sex: '男'
})

const handleConfirm = () => {
  if (formData.value.name.trim()) {
    emit('created', formData.value)
    emit('update:visible', false)
    formData.value = { name: '', sex: '男' }
  }
}

watch(() => props.visible, (newVal) => {
  if (!newVal) {
    formData.value = { name: '', sex: '男' }
  }
})
</script>
