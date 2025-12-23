<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-[80] bg-black/60 flex items-center justify-center"
    @click="$emit('update:visible', false)"
  >
    <div class="w-80 bg-[#18181b] rounded-2xl p-6 shadow-2xl" @click.stop>
      <h3 class="text-lg font-bold text-white mb-4 text-center">创建新身份</h3>

      <div class="space-y-4">
        <!-- 名字输入 -->
        <div>
          <label class="text-xs text-gray-500 mb-1 block">名字</label>
          <input
            v-model="formData.name"
            type="text"
            placeholder="输入名字"
            class="w-full bg-[#27272a] text-white px-4 py-3 rounded-xl border border-gray-700 focus:border-indigo-500 focus:outline-none"
          />
        </div>

        <!-- 性别选择 -->
        <div>
          <label class="text-xs text-gray-500 mb-1 block">性别</label>
          <div class="flex gap-3">
            <button
              @click="formData.sex = '男'"
              :class="formData.sex === '男' ? 'bg-blue-600 border-blue-600' : 'bg-[#27272a] border-gray-700'"
              class="flex-1 py-3 rounded-xl border text-white font-medium"
            >
              <i class="fas fa-mars mr-2"></i>男
            </button>
            <button
              @click="formData.sex = '女'"
              :class="formData.sex === '女' ? 'bg-pink-600 border-pink-600' : 'bg-[#27272a] border-gray-700'"
              class="flex-1 py-3 rounded-xl border text-white font-medium"
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
          class="flex-1 py-3 bg-[#27272a] text-gray-400 rounded-xl border border-gray-700"
        >
          取消
        </button>
        <button
          @click="handleConfirm"
          :disabled="!formData.name || !formData.sex"
          class="flex-1 py-3 bg-indigo-600 text-white rounded-xl font-medium disabled:opacity-50 disabled:cursor-not-allowed"
        >
          创建
        </button>
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
