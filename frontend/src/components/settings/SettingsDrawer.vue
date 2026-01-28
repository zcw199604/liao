<template>
  <teleport to="body">
    <div v-if="visible" class="fixed inset-0 z-[70] bg-black/50" @click="close">
      <div class="absolute right-0 top-0 bottom-0 w-80 bg-surface shadow-2xl overflow-y-auto" @click.stop>
        <!-- 头部 -->
        <div class="h-14 flex items-center justify-between px-4 border-b border-line shrink-0">
          <h2 class="text-lg font-bold text-fg">
            {{ mode === 'identity' ? '身份信息' : mode === 'system' ? '系统设置' : mode === 'media' ? '图片管理' : '全局收藏' }}
          </h2>
          <div class="flex items-center gap-2">
            <button
              v-if="mode === 'identity' && !editMode"
              @click="startEdit"
              class="px-3 py-1 bg-indigo-600 text-white text-sm rounded-lg"
            >
              编辑
            </button>
            <button
              v-else-if="mode === 'identity' && editMode"
              @click="saveUserInfo"
              :disabled="saving"
              class="px-3 py-1 bg-green-600 text-white text-sm rounded-lg disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              <span v-if="saving" class="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin"></span>
              <span>{{ saving ? '保存中...' : '保存' }}</span>
            </button>
            <button
              @click="close"
              class="w-10 h-10 flex items-center justify-center text-fg/40 hover:text-fg transition-colors rounded-lg hover:bg-surface-2"
            >
              <i class="fas fa-times text-xl"></i>
            </button>
          </div>
        </div>

        <!-- 身份信息 -->
        <div v-if="mode === 'identity'" class="p-6 space-y-6">
          <div class="flex justify-center">
            <div class="w-24 h-24 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white text-3xl font-bold shadow-lg">
              {{ currentUser?.name?.charAt(0) || '?' }}
            </div>
          </div>

          <div class="space-y-4">
            <div class="bg-surface-3 rounded-xl p-4">
              <div class="text-xs text-fg-subtle mb-2">用户名</div>
              <input
                v-if="editMode"
                v-model="edit.name"
                type="text"
                class="w-full bg-surface text-fg px-3 py-2 rounded-lg border border-line-strong focus:border-indigo-500 focus:outline-none placeholder-fg-subtle"
                placeholder="输入用户名"
              />
              <div v-else class="text-base text-fg font-medium">{{ currentUser?.name }}</div>
            </div>

            <div class="bg-surface-3 rounded-xl p-4">
              <div class="text-xs text-fg-subtle mb-2">用户ID</div>
              <input
                v-if="editMode"
                v-model="edit.id"
                type="text"
                class="w-full bg-surface text-fg px-3 py-2 rounded-lg border border-line-strong focus:border-indigo-500 focus:outline-none font-mono text-sm placeholder-fg-subtle"
                placeholder="输入用户ID"
              />
              <div v-else class="text-base text-fg font-mono text-sm break-all">{{ currentUser?.id }}</div>
            </div>

            <div class="bg-surface-3 rounded-xl p-4">
              <div class="text-xs text-fg-subtle mb-2">性别</div>
              <select
                v-if="editMode"
                v-model="edit.sex"
                class="w-full bg-surface text-fg px-3 py-2 rounded-lg border border-line-strong focus:border-indigo-500 focus:outline-none"
              >
                <option value="男">男</option>
                <option value="女">女</option>
              </select>
              <div v-else class="text-base text-fg">{{ currentUser?.sex }}</div>
            </div>

            <div class="bg-surface-3 rounded-xl p-4">
              <div class="text-xs text-fg-subtle mb-1">IP地址</div>
              <div class="text-base text-fg">{{ currentUser?.ip }}</div>
            </div>

            <div class="bg-surface-3 rounded-xl p-4">
              <div class="text-xs text-fg-subtle mb-1">地区</div>
              <div class="text-base text-fg">{{ currentUser?.area }}</div>
            </div>

            <div class="bg-surface-3 rounded-xl p-4">
              <div class="text-xs text-fg-subtle mb-1">Cookie（自动生成）</div>
              <div class="text-xs text-fg-muted break-all font-mono leading-relaxed">{{ currentUser?.cookie }}</div>
            </div>

            <div class="bg-surface-3 rounded-xl p-4">
              <div class="flex justify-between items-center">
                <span class="text-sm text-fg-muted">WebSocket 状态</span>
                <div class="flex items-center gap-2">
                  <span class="w-2 h-2 rounded-full" :class="chatStore.wsConnected ? 'bg-green-500' : 'bg-red-500'"></span>
                  <span class="text-sm" :class="chatStore.wsConnected ? 'text-green-500' : 'text-red-500'">
                    {{ chatStore.wsConnected ? '已连接' : '未连接' }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 系统设置 -->
        <div v-else-if="mode === 'system'" class="p-6 space-y-6">
          <SystemSettings
            :stats="connectionStats"
            :forceout-count="forceoutUserCount"
            :loading="disconnectAllLoading"
            @refresh="refreshSystem"
            @disconnect-all="confirmDisconnectAll"
            @clear-forceout="confirmClearForceout"
          />
        </div>

        <!-- 图片管理 -->
        <div v-else-if="mode === 'media'" class="p-6 space-y-6">
          <!-- 媒体库 -->
          <div class="bg-surface-3 rounded-xl p-4">
             <h3 class="text-fg font-medium mb-3 flex items-center gap-2">
                <i class="fas fa-photo-video text-purple-400"></i>
                <span>媒体库</span>
             </h3>
             
             <div 
               @click="openMediaManagement"
               class="group relative overflow-hidden rounded-xl bg-gradient-to-br from-purple-900/50 to-indigo-900/50 border border-white/5 cursor-pointer hover:border-purple-500/50 transition-all"
             >
                <div class="p-4 flex items-center justify-between z-10 relative">
                   <div>
                      <div class="text-2xl font-bold text-white mb-1">{{ mediaStore.allUploadTotal || 0 }}</div>
                      <div class="text-xs text-gray-400 group-hover:text-purple-300 transition-colors">所有上传图片</div>
                   </div>
                   <div class="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center group-hover:bg-purple-500 group-hover:text-white transition-all">
                      <i class="fas fa-chevron-right"></i>
                   </div>
                </div>
                
                <!-- 装饰性背景图标 -->
                <i class="fas fa-images absolute -bottom-4 -right-2 text-6xl text-white/5 group-hover:text-white/10 transition-colors rotate-12"></i>
             </div>
             
             <p class="text-xs text-gray-500 mt-3 flex items-center gap-1.5">
               <i class="fas fa-info-circle"></i>
               <span>点击卡片进入管理或清理历史图片</span>
             </p>

             <div
               @click="openMtPhotoAlbums"
               class="group relative overflow-hidden rounded-xl bg-gradient-to-br from-pink-900/40 to-rose-900/40 border border-white/5 cursor-pointer hover:border-pink-500/50 transition-all mt-4"
             >
                <div class="p-4 flex items-center justify-between z-10 relative">
                   <div>
                      <div class="text-base font-bold text-white mb-1">mtPhoto 相册</div>
                      <div class="text-xs text-gray-400 group-hover:text-pink-300 transition-colors">按相册浏览并导入上传</div>
                   </div>
                   <div class="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center group-hover:bg-pink-500 group-hover:text-white transition-all">
                      <i class="fas fa-chevron-right"></i>
                   </div>
                </div>
                <i class="fas fa-photo-video absolute -bottom-4 -right-2 text-6xl text-white/5 group-hover:text-white/10 transition-colors rotate-12"></i>
             </div>

             <div
               @click="openDouyinDownloader"
               class="group relative overflow-hidden rounded-xl bg-gradient-to-br from-emerald-900/40 to-teal-900/40 border border-white/5 cursor-pointer hover:border-emerald-500/50 transition-all mt-4"
             >
                <div class="p-4 flex items-center justify-between z-10 relative">
                   <div>
                      <div class="text-base font-bold text-white mb-1">抖音下载</div>
                      <div class="text-xs text-gray-400 group-hover:text-emerald-300 transition-colors">解析→预览→下载→导入上传</div>
                   </div>
                   <div class="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center group-hover:bg-emerald-500 group-hover:text-white transition-all">
                      <i class="fas fa-chevron-right"></i>
                   </div>
                </div>
                <i class="fas fa-cloud-download-alt absolute -bottom-4 -right-2 text-6xl text-white/5 group-hover:text-white/10 transition-colors rotate-12"></i>
             </div>

             <div
               @click="openDuplicateCheck"
               class="group relative overflow-hidden rounded-xl bg-gradient-to-br from-blue-900/40 to-cyan-900/40 border border-white/5 cursor-pointer hover:border-blue-500/50 transition-all mt-4"
             >
                <div class="p-4 flex items-center justify-between z-10 relative">
                   <div>
                      <div class="text-base font-bold text-white mb-1">图片查重</div>
                      <div class="text-xs text-gray-400 group-hover:text-blue-300 transition-colors">按相似度检测重复图片</div>
                   </div>
                   <div class="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center group-hover:bg-blue-500 group-hover:text-white transition-all">
                      <i class="fas fa-chevron-right"></i>
                   </div>
                </div>
                <i class="fas fa-search absolute -bottom-4 -right-2 text-6xl text-white/5 group-hover:text-white/10 transition-colors rotate-12"></i>
             </div>

             <div
               @click="openVideoExtractTasks"
               class="group relative overflow-hidden rounded-xl bg-gradient-to-br from-purple-900/40 to-indigo-900/40 border border-white/5 cursor-pointer hover:border-purple-500/50 transition-all mt-4"
             >
                <div class="p-4 flex items-center justify-between z-10 relative">
                   <div>
                      <div class="text-base font-bold text-white mb-1">抽帧任务中心</div>
                      <div class="text-xs text-gray-400 group-hover:text-purple-300 transition-colors">查看/终止/继续抽帧任务</div>
                   </div>
                   <div class="w-10 h-10 rounded-full bg-white/10 flex items-center justify-center group-hover:bg-purple-500 group-hover:text-white transition-all">
                      <i class="fas fa-chevron-right"></i>
                   </div>
                </div>
                <i class="fas fa-tasks absolute -bottom-4 -right-2 text-6xl text-white/5 group-hover:text-white/10 transition-colors rotate-12"></i>
             </div>
          </div>
        </div>

        <!-- 全局收藏 -->
        <GlobalFavorites v-else-if="mode === 'favorites'" />

        <Dialog
          v-model:visible="showDisconnectAllDialog"
          title="确认断开所有连接"
          content="此操作会断开所有客户端和上游服务器的连接"
          show-warning
          @confirm="doDisconnectAll"
        />

        <Dialog
          v-model:visible="showClearForceoutDialog"
          title="确认清除禁止连接用户"
          :content="`当前有 ${forceoutUserCount} 个用户被禁止连接（5分钟限制），确定清除？`"
          show-warning
          confirm-button-class="bg-amber-600"
          @confirm="doClearForceout"
        />

        <DuplicateCheckModal v-model:visible="showDuplicateCheck" />
      </div>
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useMediaStore } from '@/stores/media'
import { useMtPhotoStore } from '@/stores/mtphoto'
import { useDouyinStore } from '@/stores/douyin'
import { useVideoExtractStore } from '@/stores/videoExtract'
import { useWebSocket } from '@/composables/useWebSocket'
import { useSettings } from '@/composables/useSettings'
import { useToast } from '@/composables/useToast'
import { generateCookie } from '@/utils/cookie'
import * as identityApi from '@/api/identity'
import SystemSettings from '@/components/settings/SystemSettings.vue'
import GlobalFavorites from '@/components/settings/GlobalFavorites.vue'
import Dialog from '@/components/common/Dialog.vue'
import DuplicateCheckModal from '@/components/media/DuplicateCheckModal.vue'

type Mode = 'identity' | 'system' | 'media' | 'favorites'

interface Props {
  visible: boolean
  mode: Mode
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const userStore = useUserStore()
const chatStore = useChatStore()
const messageStore = useMessageStore()
const mediaStore = useMediaStore()
const mtPhotoStore = useMtPhotoStore()
const douyinStore = useDouyinStore()
const videoExtractStore = useVideoExtractStore()
const { disconnect, connect, send } = useWebSocket()
const { show } = useToast()

const { connectionStats, forceoutUserCount, disconnectAllLoading, loadConnectionStats, loadForceoutUserCount, disconnectAll, clearForceout } = useSettings()

const currentUser = computed(() => userStore.currentUser)

const editMode = ref(false)
const saving = ref(false)
const edit = ref({ id: '', name: '', sex: '男' })

const showDisconnectAllDialog = ref(false)
const showClearForceoutDialog = ref(false)
const showDuplicateCheck = ref(false)

const close = () => {
  editMode.value = false
  saving.value = false
  emit('update:visible', false)
}

const startEdit = () => {
  if (!currentUser.value) return
  edit.value = { id: currentUser.value.id, name: currentUser.value.name, sex: currentUser.value.sex }
  editMode.value = true
}

const saveUserInfo = async () => {
  if (!currentUser.value) return

  const idChanged = edit.value.id !== currentUser.value.id
  const nameChanged = edit.value.name !== currentUser.value.name
  const sexChanged = edit.value.sex !== currentUser.value.sex

  if (!idChanged && !nameChanged && !sexChanged) {
    show('没有任何修改')
    editMode.value = false
    return
  }

  saving.value = true
  try {
    // ID 改变：更新数据库 + 断开连接 + 重新连接
    if (idChanged) {
      const res = await identityApi.updateIdentityId({
        oldId: currentUser.value.id,
        newId: edit.value.id,
        name: edit.value.name,
        sex: edit.value.sex
      })

      if (res.code !== 0) {
        show(`更新失败: ${res.msg || '未知错误'}`)
        return
      }

      userStore.setCurrentUser({
        ...currentUser.value,
        id: edit.value.id,
        name: edit.value.name,
        nickname: edit.value.name,
        sex: edit.value.sex,
        cookie: generateCookie(edit.value.id, edit.value.name)
      })

      disconnect(true)
      messageStore.resetAll()
      chatStore.clearAllUsers()
      chatStore.activeTab = 'history'

      connect()
      await Promise.all([
        chatStore.loadHistoryUsers(edit.value.id, edit.value.name),
        chatStore.loadFavoriteUsers(edit.value.id, edit.value.name)
      ])
      show('用户ID已更新，正在重新连接...')
      editMode.value = false
      emit('update:visible', false)
      return
    }

    // 名字/性别改变：更新数据库 + 同步上游 WS（如果已连接）
    if (nameChanged || sexChanged) {
      const res = await identityApi.updateIdentity({
        id: currentUser.value.id,
        name: edit.value.name,
        sex: edit.value.sex
      })

      if (res.code !== 0) {
        show(`更新失败: ${res.msg || '未知错误'}`)
        return
      }

      if (!chatStore.wsConnected) {
        show('数据库已更新，但WebSocket未连接，无法同步聊天状态')
      } else {
        if (sexChanged) {
          send({
            act: 'modinfo',
            id: currentUser.value.id,
            userSex: edit.value.sex,
            address_show: 'false',
            randomhealthmode: '0',
            randomvipsex: '0',
            randomvipaddress: '0'
          })
        }
        if (nameChanged) {
          send({
            act: 'chgname',
            id: currentUser.value.id,
            msg: edit.value.name
          })
        }
      }

      userStore.updateUserInfo({
        name: edit.value.name,
        nickname: edit.value.name,
        sex: edit.value.sex,
        cookie: generateCookie(currentUser.value.id, edit.value.name)
      })

      show('信息已保存')
      editMode.value = false
    }
  } catch (e: any) {
    console.error('保存用户信息失败:', e)
    show(`保存失败: ${e?.message || '未知错误'}`)
  } finally {
    saving.value = false
  }
}

const refreshSystem = async () => {
  await Promise.all([loadConnectionStats(), loadForceoutUserCount()])
}

const confirmDisconnectAll = () => {
  showDisconnectAllDialog.value = true
}

const doDisconnectAll = async () => {
  const ok = await disconnectAll()
  show(ok ? '已断开所有连接' : '操作失败')
}

const confirmClearForceout = () => {
  showClearForceoutDialog.value = true
}

const doClearForceout = async () => {
  const res = await clearForceout()
  show(res.success ? res.message : res.message)
}

const openMediaManagement = async () => {
  mediaStore.managementMode = true
  mediaStore.selectionMode = false
  mediaStore.selectedImages = []
  mediaStore.showAllUploadImageModal = true
  await mediaStore.loadAllUploadImages(1)
}

const openMtPhotoAlbums = async () => {
  await mtPhotoStore.open()
}

const openDouyinDownloader = () => {
  close()
  douyinStore.open()
}

const openDuplicateCheck = () => {
  showDuplicateCheck.value = true
}

const openVideoExtractTasks = async () => {
  await videoExtractStore.openTaskCenter()
}

watch(
  () => props.visible,
  async (v) => {
    if (!v) return

    if (props.mode === 'system') {
      await refreshSystem()
    }

    if (props.mode === 'media') {
      await mediaStore.loadAllUploadImages(1)
    }
  }
)
</script>
