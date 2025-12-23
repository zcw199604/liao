<template>
  <div class="page-container bg-[#0f0f13]">
    <!-- 顶部标题 -->
    <div class="pt-8 pb-4 px-4 text-center">
      <h1 class="text-2xl font-bold text-white mb-2">选择身份</h1>
      <p class="text-sm text-gray-400">选择一个身份开始聊天，或创建新身份</p>
    </div>

    <!-- 操作按钮区 -->
    <div class="flex gap-3 px-4 mb-4">
      <button
        @click="handleQuickCreate"
        :disabled="loading"
        class="flex-1 py-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl font-medium shadow-lg disabled:opacity-50"
      >
        <i class="fas fa-bolt mr-2"></i>快速进入
      </button>
      <button
        @click="showCreateDialog = true"
        class="flex-1 py-3 bg-[#27272a] text-white rounded-xl font-medium border border-gray-700"
      >
        <i class="fas fa-plus mr-2"></i>创建身份
      </button>
    </div>

    <!-- 身份列表 -->
    <div class="list-area no-scrollbar px-4">
      <!-- 加载中 -->
      <div v-if="loading" class="flex flex-col items-center justify-center mt-20">
        <div class="radar-spinner"></div>
        <p class="text-gray-500 text-sm mt-4">加载中...</p>
      </div>

      <!-- 身份卡片列表 -->
      <IdentityList
        v-else
        :identities="identityStore.identityList"
        @select="handleSelect"
        @delete="handleDeleteConfirm"
      />
    </div>

    <!-- 创建身份对话框 -->
    <CreateDialog
      v-model:visible="showCreateDialog"
      @created="handleCreate"
    />

    <!-- 删除确认对话框 -->
    <Dialog
      v-model:visible="showDeleteDialog"
      title="确认删除"
      :content="`确定要删除身份 ${deleteTarget?.name} 吗？`"
      show-warning
      @confirm="handleDelete"
    />

    <Toast />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useIdentityStore } from '@/stores/identity'
import { useIdentity } from '@/composables/useIdentity'
import { useToast } from '@/composables/useToast'
import IdentityList from '@/components/identity/IdentityList.vue'
import CreateDialog from '@/components/identity/CreateDialog.vue'
import Dialog from '@/components/common/Dialog.vue'
import Toast from '@/components/common/Toast.vue'
import type { Identity } from '@/types'

const identityStore = useIdentityStore()
const { loadList, select, create, quickCreate, deleteIdentity } = useIdentity()
const { show } = useToast()

const loading = ref(false)
const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const deleteTarget = ref<Identity | null>(null)

const handleSelect = async (identity: Identity) => {
  loading.value = true
  try {
    await select(identity)
  } catch (error) {
    console.error('选择身份失败:', error)
    show('选择身份失败')
  } finally {
    loading.value = false
  }
}

const handleQuickCreate = async () => {
  loading.value = true
  try {
    const success = await quickCreate()
    if (success) {
      show('创建成功')
      await loadList()
    }
  } catch (error) {
    console.error('快速创建失败:', error)
    show('创建失败')
  } finally {
    loading.value = false
  }
}

const handleCreate = async (data: { name: string; sex: string }) => {
  loading.value = true
  try {
    const success = await create(data.name, data.sex)
    if (success) {
      show('创建成功')
      await loadList()
    } else {
      show('创建失败')
    }
  } finally {
    loading.value = false
  }
}

const handleDeleteConfirm = (identity: Identity) => {
  deleteTarget.value = identity
  showDeleteDialog.value = true
}

const handleDelete = async () => {
  if (!deleteTarget.value) return

  loading.value = true
  try {
    const success = await deleteIdentity(deleteTarget.value.id)
    if (success) {
      show('删除成功')
    } else {
      show('删除失败')
    }
  } finally {
    loading.value = false
    deleteTarget.value = null
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await loadList()
  } finally {
    loading.value = false
  }
})
</script>
