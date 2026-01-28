<template>
  <div class="h-full flex flex-col bg-surface">
     <!-- Header -->
     <div class="p-4 border-b border-line shrink-0">
        <h2 class="text-lg font-bold text-fg">全局收藏</h2>
        <p class="text-xs text-fg-muted">管理所有身份的收藏联系人</p>
     </div>

     <!-- List -->
     <PullToRefresh :on-refresh="handleRefresh" class="flex-1 min-h-0">
        <div class="h-full overflow-y-auto p-4 custom-scrollbar">
           <div v-if="favoriteStore.loading && favoriteStore.allFavorites.length === 0" class="space-y-3 py-2">
              <div v-for="i in 6" :key="'sk-fav-' + i" class="flex items-center justify-between p-3 bg-surface-active rounded-lg border border-line">
                 <div class="flex items-center gap-3 overflow-hidden">
                    <Skeleton class="w-10 h-10 rounded-full" />
                    <div class="min-w-0">
                       <Skeleton class="h-4 w-24 rounded" />
                       <Skeleton class="h-3 w-32 rounded mt-2" />
                    </div>
                 </div>
                 <Skeleton class="h-4 w-10 rounded" />
              </div>
           </div>
           
           <div v-else-if="Object.keys(favoriteStore.groupedFavorites).length === 0" class="flex flex-col items-center justify-center mt-20 text-fg-subtle">
              <i class="fas fa-star text-4xl mb-3 opacity-20"></i>
              <p>暂无收藏</p>
           </div>
   
           <div v-else class="space-y-6">
              <div v-for="(favs, identityId) in favoriteStore.groupedFavorites" :key="identityId" class="animate-in fade-in slide-in-from-bottom-2 duration-300">
                 <!-- Identity Header -->
                 <div class="flex items-center gap-2 mb-2 px-2 sticky top-0 bg-surface/95 backdrop-blur-sm py-2 z-10">
                    <div class="w-1 h-4 bg-indigo-500 rounded-full"></div>
                    <h3 class="text-sm font-bold text-fg">
                       {{ getIdentityName(identityId) }}
                    </h3>
                    <span class="text-xs text-fg-subtle font-mono hidden sm:inline">({{ identityId.slice(0, 6) }}...)</span>
                 </div>
   
                 <!-- Favorites Grid -->
                 <div class="grid grid-cols-1 gap-2">
                    <div v-for="fav in favs" :key="fav.id" @click="openPreview(fav)" class="group flex items-center justify-between p-3 bg-surface-3 hover:bg-surface-hover rounded-lg border border-line hover:border-indigo-500/30 transition-all cursor-pointer">
                       <div class="flex items-center gap-3 overflow-hidden">
                          <div class="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-sm shrink-0 shadow-lg">
                             {{ (fav.targetUserName || fav.targetUserId || '?').slice(0, 1).toUpperCase() }}
                          </div>
                          <div class="min-w-0">
                             <div class="text-sm text-fg font-medium truncate">{{ fav.targetUserName || '未知用户' }}</div>
                             <div class="text-xs text-fg-subtle truncate font-mono">{{ fav.targetUserId }}</div>
                          </div>
                       </div>
   
                       <div class="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button @click.stop="openPreview(fav)" class="w-8 h-8 flex items-center justify-center bg-surface/60 hover:bg-surface-hover rounded-lg text-fg-muted hover:text-indigo-500 transition border border-line" title="预览聊天">
                             <i class="fas fa-eye"></i>
                          </button>
                          <button @click.stop="directSwitch(fav)" class="w-8 h-8 flex items-center justify-center bg-surface/60 hover:bg-surface-hover rounded-lg text-fg-muted hover:text-green-500 transition border border-line" title="切换并聊天">
                             <i class="fas fa-comment-dots"></i>
                          </button>
                          <button @click.stop="confirmDelete(fav)" class="w-8 h-8 flex items-center justify-center bg-surface/60 hover:bg-surface-hover rounded-lg text-fg-muted hover:text-red-500 transition border border-line" title="取消收藏">
                             <i class="fas fa-trash-alt"></i>
                          </button>
                       </div>
                    </div>
                 </div>
              </div>
           </div>
        </div>
     </PullToRefresh>
    
     <!-- Preview Modal -->
     <ChatHistoryPreview 
        :visible="showPreview"
        :identity-id="previewIdentityId"
        :target-user-id="previewTargetId"
        :target-user-name="previewTargetName"
        @close="showPreview = false"
        @switch="handlePreviewSwitch"
     />

     <!-- Delete Confirm Dialog -->
     <div v-if="showDeleteDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
        <div class="bg-surface rounded-xl p-6 w-full max-w-sm shadow-xl border border-line">
           <h3 class="text-lg font-bold text-fg mb-2">确认删除</h3>
           <p class="text-fg-muted text-sm mb-6">确定要取消收藏该用户吗？</p>
           <div class="flex justify-end gap-3">
              <button @click="showDeleteDialog = false" class="px-4 py-2 text-fg-muted hover:text-fg text-sm">取消</button>
              <button @click="executeDelete" class="px-4 py-2 bg-red-500/10 text-red-400 hover:bg-red-500/20 rounded-lg text-sm">删除</button>
           </div>
        </div>
     </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useFavoriteStore } from '@/stores/favorite'
import { useIdentityStore } from '@/stores/identity'
import { useIdentity } from '@/composables/useIdentity'
import { useChat } from '@/composables/useChat'
import { useWebSocket } from '@/composables/useWebSocket'
import type { Favorite } from '@/types'
import ChatHistoryPreview from '@/components/chat/ChatHistoryPreview.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import PullToRefresh from '@/components/common/PullToRefresh.vue'
import { useToast } from '@/composables/useToast'
import { useRouter } from 'vue-router'

const favoriteStore = useFavoriteStore()
const identityStore = useIdentityStore()
const { select } = useIdentity()
const { enterChat } = useChat()
const { disconnect } = useWebSocket()
const { show } = useToast()
const router = useRouter()

const showPreview = ref(false)
const previewIdentityId = ref('')
const previewTargetId = ref('')
const previewTargetName = ref('')

const showDeleteDialog = ref(false)
const deleteTarget = ref<Favorite | null>(null)

const getIdentityName = (id: string) => {
   const identity = identityStore.identityList.find(i => i.id === id)
   return identity ? identity.name : '未知身份'
}

const openPreview = (fav: Favorite) => {
   previewIdentityId.value = fav.identityId
   previewTargetId.value = fav.targetUserId
   previewTargetName.value = fav.targetUserName || ''
   showPreview.value = true
}

const handlePreviewSwitch = () => {
   // Switch using the preview data
   const identity = identityStore.identityList.find(i => i.id === previewIdentityId.value)
   if (identity) {
      switchToIdentityAndChat(identity, previewTargetId.value, previewTargetName.value)
      showPreview.value = false
   } else {
      show('身份不存在，无法切换')
   }
}

const directSwitch = (fav: Favorite) => {
   const identity = identityStore.identityList.find(i => i.id === fav.identityId)
   if (identity) {
      switchToIdentityAndChat(identity, fav.targetUserId, fav.targetUserName)
   } else {
      show('身份不存在，无法切换')
   }
}

const switchToIdentityAndChat = async (identity: any, targetUserId: string, targetUserName?: string) => {
   try {
      // Disconnect current socket to force reconnection with new identity
      disconnect(true)
      
      await select(identity)
      // Wait a bit for router push and store updates
      setTimeout(() => {
         // Create a temporary user object for the target
         const name = targetUserName || '用户' + targetUserId.slice(0, 4)
         const targetUser = {
            id: targetUserId,
            name: name,
            nickname: name,
            sex: '未知',
            ip: '',
            isFavorite: true 
         }
         
         // Set the current chat user in the store
         enterChat(targetUser, true)
         
         // Navigate directly to the chat room
         router.push(`/chat/${targetUserId}`)
      }, 500)
   } catch (e) {
      console.error(e)
      show('切换失败')
   }
}

const confirmDelete = (fav: Favorite) => {
   deleteTarget.value = fav
   showDeleteDialog.value = true
}

const executeDelete = async () => {
   if (!deleteTarget.value) return
   
   const success = await favoriteStore.removeFavoriteById(deleteTarget.value.id)
   if (success) {
      show('已取消收藏')
   } else {
      show('操作失败')
   }
   showDeleteDialog.value = false
   deleteTarget.value = null
}

const handleRefresh = async () => {
  await favoriteStore.loadAllFavorites()
}

onMounted(() => {
   favoriteStore.loadAllFavorites()
   // Ensure identity list is loaded to show names
   if (identityStore.identityList.length === 0) {
      identityStore.loadList()
   }
})
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #3f3f46;
  border-radius: 2px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #52525b;
}
</style>
