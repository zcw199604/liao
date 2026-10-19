<template>
  <div class="page-container bg-canvas">
    <!-- 顶部标题 -->
    <div class="relative pt-8 pb-4 px-4 text-center">
      <button
        data-testid="identity-logout"
        @click="showLogoutDialog = true"
        class="absolute right-4 top-6 ui-icon-btn ui-icon-btn-ghost text-fg-muted"
        aria-label="退出登录"
        title="退出登录"
      >
        <i class="fas fa-sign-out-alt"></i>
      </button>
      <h1 class="text-2xl font-bold text-fg mb-2">选择身份</h1>
      <p class="text-sm text-fg-muted">选择一个身份开始聊天，或创建新身份</p>
    </div>

    <!-- 操作按钮区 -->
    <div class="flex gap-3 px-4 mb-4">
      <button
        @click="handleQuickCreate"
        :disabled="actionLoading"
        class="flex-1 py-3 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl font-medium shadow-lg shadow-indigo-500/20 shadow-[inset_0_1px_0_rgba(255,255,255,0.2)] disabled:opacity-50 transition-colors"
      >
        <i class="fas fa-bolt mr-2"></i>快速进入
      </button>
      <button
        @click="showCreateDialog = true"
        :disabled="actionLoading"
        class="flex-1 py-3 bg-surface-2 hover:bg-surface-hover text-fg rounded-xl font-medium border border-line disabled:opacity-50 transition-colors"
      >
        <i class="fas fa-plus mr-2"></i>创建身份
      </button>
    </div>

    <!-- 图片管理 -->
    <div class="px-4 mb-4">
      <button
        @click="showMediaDrawer = true"
        class="w-full py-3 bg-surface-2 hover:bg-surface-hover text-fg rounded-xl font-medium border border-line transition-colors flex items-center justify-center gap-2"
      >
        <i class="fas fa-images text-purple-400"></i>
        <span>图片管理</span>
      </button>
    </div>

    <!-- 身份列表 -->
    <div class="list-area no-scrollbar px-4">
      <!-- 首次加载中（后续操作不再替换整个列表） -->
      <div v-if="listLoading && identityStore.identityList.length === 0" class="flex flex-col items-center justify-center mt-20">
        <div class="radar-spinner"></div>
        <p class="text-fg-subtle text-sm mt-4">加载中...</p>
      </div>

      <!-- 身份卡片列表 -->
      <IdentityList
        v-else
        :identities="identityStore.identityList"
        :busy-id="busyIdentityId"
        :disabled="actionLoading"
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

    <!-- 退出登录确认 -->
    <Dialog
      v-model:visible="showLogoutDialog"
      title="退出登录"
      content="确定要退出登录吗？"
      confirm-text="退出"
      @confirm="handleLogout"
    />

    <Toast />

    <SettingsDrawer v-model:visible="showMediaDrawer" mode="media" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useIdentityStore } from '@/stores/identity'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
import { useIdentity } from '@/composables/useIdentity'
import { useWebSocket } from '@/composables/useWebSocket'
import { useToast } from '@/composables/useToast'
import IdentityList from '@/components/identity/IdentityList.vue'
import CreateDialog from '@/components/identity/CreateDialog.vue'
import Dialog from '@/components/common/Dialog.vue'
import Toast from '@/components/common/Toast.vue'
import SettingsDrawer from '@/components/settings/SettingsDrawer.vue'
import type { Identity } from '@/types'

const router = useRouter()
const identityStore = useIdentityStore()
const authStore = useAuthStore()
const userStore = useUserStore()
const { loadList, select, create, quickCreate, deleteIdentity } = useIdentity()
const { show } = useToast()

// 列表首次加载与各类操作分开管理，操作期间列表保持可见
const listLoading = ref(false)
const actionLoading = ref(false)
const busyIdentityId = ref('')
const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const showLogoutDialog = ref(false)
const deleteTarget = ref<Identity | null>(null)
const showMediaDrawer = ref(false)

// 统一包装操作：防重复触发，并记录正在处理的身份
const runAction = async (fn: () => Promise<void>, identityId = '') => {
  if (actionLoading.value) return
  actionLoading.value = true
  busyIdentityId.value = identityId
  try {
    await fn()
  } finally {
    actionLoading.value = false
    busyIdentityId.value = ''
  }
}

const handleSelect = (identity: Identity) => runAction(async () => {
  try {
    await select(identity)
  } catch (error) {
    console.error('选择身份失败:', error)
    show('选择身份失败')
  }
}, identity.id)

// 创建成功后 store 内部已刷新列表，这里无需重复加载
const handleQuickCreate = () => runAction(async () => {
  try {
    const success = await quickCreate()
    show(success ? '创建成功' : '创建失败')
  } catch (error) {
    console.error('快速创建失败:', error)
    show('创建失败')
  }
})

const handleCreate = (data: { name: string; sex: string }) => runAction(async () => {
  try {
    const success = await create(data.name, data.sex)
    show(success ? '创建成功' : '创建失败')
  } catch (error) {
    console.error('创建身份失败:', error)
    show('创建失败')
  }
})

const handleDeleteConfirm = (identity: Identity) => {
  deleteTarget.value = identity
  showDeleteDialog.value = true
}

const handleDelete = async () => {
  const target = deleteTarget.value
  if (!target) return

  await runAction(async () => {
    try {
      const success = await deleteIdentity(target.id)
      show(success ? '删除成功' : '删除失败')
    } catch (error) {
      console.error('删除身份失败:', error)
      show('删除失败')
    } finally {
      deleteTarget.value = null
    }
  }, target.id)
}

const handleLogout = () => {
  useWebSocket().disconnect(true)
  userStore.clearCurrentUser()
  authStore.logout()
  router.replace('/login')
}

onMounted(async () => {
  listLoading.value = true
  try {
    await loadList()
  } finally {
    listLoading.value = false
  }
})
</script>
