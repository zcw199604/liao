<template>
  <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4" @click.self="close">
    <div class="bg-[#1e1e24] rounded-xl shadow-2xl w-full max-w-2xl h-[80vh] flex flex-col overflow-hidden border border-gray-800 animate-in fade-in zoom-in duration-200">
      
      <!-- Header -->
      <div class="px-6 py-4 border-b border-gray-800 bg-[#2d2d33] flex items-center justify-between shrink-0">
        <div>
           <h3 class="font-bold text-lg text-white flex items-center gap-2">
             <i class="fas fa-history text-indigo-400"></i>
             历史消息预览
           </h3>
           <p class="text-xs text-gray-400 mt-1">
             与 <span class="text-indigo-400 font-mono">{{ targetUserName || targetUserId }}</span> 的聊天记录
           </p>
        </div>
        <button @click="close" class="text-gray-400 hover:text-white transition w-8 h-8 rounded-full flex items-center justify-center hover:bg-white/10">
          <i class="fas fa-times text-lg"></i>
        </button>
      </div>

      <!-- Messages -->
      <div class="flex-1 overflow-y-auto p-4" ref="msgContainer">
         <div v-if="loading" class="flex flex-col items-center justify-center h-full gap-3">
            <div class="w-8 h-8 border-4 border-indigo-500 border-t-transparent rounded-full animate-spin"></div>
            <p class="text-xs text-gray-500">正在加载历史记录...</p>
         </div>
         
         <div v-else-if="error" class="flex flex-col items-center justify-center h-full text-red-400 gap-2">
            <i class="fas fa-exclamation-circle text-3xl"></i>
            <p>{{ error }}</p>
            <p class="text-xs text-gray-500">请尝试重新登录该身份以刷新凭证</p>
         </div>

         <div v-else-if="messages.length === 0" class="flex flex-col items-center justify-center h-full text-gray-500">
            <i class="fas fa-comment-slash text-4xl mb-3 opacity-50"></i>
            <p>暂无聊天记录</p>
         </div>

         <template v-else>
            <div v-for="msg in messages" :key="msg.tid || msg.time" class="flex flex-col mb-4 w-full" :class="msg.isSelf ? 'items-end' : 'items-start'">
               <div class="text-[10px] text-gray-500 mb-1 px-1 flex items-center gap-2">
                 <span v-if="!msg.isSelf" class="font-medium text-gray-400">{{ msg.fromuser?.nickname }}</span>
                 {{ formatTime(msg.time) }}
               </div>
               
               <div class="msg-bubble shadow-sm text-sm max-w-[85%]" :class="msg.isSelf ? 'msg-right' : 'msg-left'">
                  <!-- 文本 -->
                  <div v-if="!msg.isImage && !msg.isVideo && !msg.isFile" v-html="parseEmoji(msg.content, emojiMap)"></div>
                  
                  <!-- 图片 -->
                  <div v-if="msg.isImage" class="mt-1">
                    <img :src="getMediaUrl(msg.imageUrl || msg.content || '')" class="rounded-lg max-w-full max-h-[300px] object-cover bg-gray-900/50 block" />
                  </div>

                  <!-- 视频 -->
                  <div v-if="msg.isVideo" class="mt-1">
                     <video :src="getMediaUrl(msg.videoUrl || msg.content || '')" controls class="rounded-lg max-w-full max-h-[300px] bg-black block"></video>
                  </div>
               </div>
            </div>
         </template>
      </div>

      <!-- Footer -->
      <div class="p-4 border-t border-gray-800 bg-[#2d2d33] flex justify-between items-center shrink-0">
        <span class="text-xs text-gray-500">
           <i class="fas fa-info-circle mr-1"></i>
           当前为预览模式，无法发送消息
        </span>
        <div class="flex gap-3">
          <button @click="close" class="px-4 py-2 text-gray-400 hover:text-white text-sm transition">
            关闭
          </button>
          <button @click="switchToChat" class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg text-sm font-medium transition shadow-lg shadow-indigo-600/20 flex items-center gap-2">
            <i class="fas fa-exchange-alt"></i>
            切换身份并聊天
          </button>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useIdentityStore } from '@/stores/identity'
import * as chatApi from '@/api/chat'
import type { ChatMessage } from '@/types'
import { formatTime } from '@/utils/time'
import { parseEmoji } from '@/utils/string'
import { emojiMap } from '@/constants/emoji'
import { useUpload } from '@/composables/useUpload'

const props = defineProps<{
  visible: boolean
  identityId: string
  targetUserId: string
  targetUserName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'switch'): void
}>()

const identityStore = useIdentityStore()
const { getMediaUrl } = useUpload()

const messages = ref<ChatMessage[]>([])
const loading = ref(false)
const error = ref('')
const msgContainer = ref<HTMLElement | null>(null)

const loadHistory = async () => {
  if (!props.identityId || !props.targetUserId) return
  
  loading.value = true
  error.value = ''
  messages.value = []

  const cookie = identityStore.getIdentityCookie(props.identityId)
  if (!cookie) {
    error.value = '未找到该身份的登录凭证，请先登录一次该身份'
    loading.value = false
    return
  }

  try {
    // 构造请求，模拟 UserAgent 和 Referer
    const referer = `http://v1.chat2019.cn/chat/index?u=${props.targetUserId}`
    const userAgent = navigator.userAgent
    
    // 注意：这里我们通过前端代理或后端代理转发，需要确保 cookieData 格式正确
    // getMessageHistory 需要 cookieData 字符串
    const res = await chatApi.getMessageHistory(
      props.identityId,
      props.targetUserId,
      '1', // isFirst
      '0', // firstTid
      cookie,
      referer,
      userAgent
    )

    // Handle JSON response (new format)
    if (res && res.code === 0 && Array.isArray(res.contents_list)) {
        const mapped: ChatMessage[] = res.contents_list.reverse().map((msg: any) => {
          const rawContent = String(msg?.content || '')
          const msgTid = String(msg?.Tid || msg?.tid || '')
          const msgTime = String(msg?.time || '')
          const isSelf = String(msg?.id || '') !== props.targetUserId 
          
          let content = rawContent
          let type = 'text'
          
          if (rawContent.startsWith('[') && rawContent.endsWith(']')) {
             if (emojiMap[rawContent]) {
               type = 'text'
             } else {
               const path = rawContent.substring(1, rawContent.length - 1)
               const isVideo = path.match(/\.(mp4|mov|avi)$/i)
               const isImage = path.match(/\.(jpg|jpeg|png|gif|webp)$/i)
               
               if (isVideo) type = 'video'
               else if (isImage) type = 'image'
               else type = 'file'
             }
          }
          
          return {
             code: 0,
             tid: msgTid,
             fromuser: {
                id: String(msg?.id || ''),
                name: isSelf ? '我' : (props.targetUserName || '对方'),
                nickname: isSelf ? '我' : (props.targetUserName || '对方'),
                sex: '未知',
                ip: ''
             },
             content,
             time: msgTime,
             isSelf: String(msg?.id || '') !== props.targetUserId,
             type,
             isImage: type === 'image',
             isVideo: type === 'video',
             isFile: type === 'file',
             imageUrl: type === 'image' ? content : '',
             videoUrl: type === 'video' ? content : '',
             fileUrl: type === 'file' ? content : ''
          }
        })
        messages.value = mapped
    }
    // Handle XML response (legacy format)
    else if (res && (typeof res === 'string' || typeof res.data === 'string')) {
       const xmlStr = typeof res === 'string' ? res : res.data
       if (xmlStr.includes('<ArrayOfMsg')) {
           const parser = new DOMParser()
           const xmlDoc = parser.parseFromString(xmlStr, 'text/xml')
           const msgs = xmlDoc.getElementsByTagName('Msg')
           
           const parsed: ChatMessage[] = []
           for(let i=0; i<msgs.length; i++) {
              const msg = msgs[i]
              if (!msg) continue
              
              const from = msg.querySelector('From')?.textContent || ''
              const body = msg.querySelector('Body')?.textContent || ''
              const time = msg.querySelector('Time')?.textContent || ''
              const tid = time 
              
              const isImage = body.startsWith('[img]')
              const isVideo = body.startsWith('[video]')
              
              let content = body
              let imageUrl = ''
              let videoUrl = ''
              
              if (isImage) {
                 imageUrl = body.replace('[img]', '').replace('[/img]', '')
                 content = imageUrl
              }
              if (isVideo) {
                 videoUrl = body.replace('[video]', '').replace('[/video]', '')
                 content = videoUrl
              }
              
              parsed.push({
                 code: 0,
                 tid,
                 fromuser: {
                    id: from,
                    name: from === props.identityId ? '我' : '对方',
                    nickname: from === props.identityId ? '我' : '对方',
                    sex: '未知',
                    ip: ''
                 },
                 content,
                 time,
                 isSelf: from === props.identityId,
                 type: isImage ? 'image' : isVideo ? 'video' : 'text',
                 isImage,
                 isVideo,
                 imageUrl,
                 videoUrl,
                 isFile: false,
                 fileUrl: ''
              })
           }
           messages.value = parsed
       } else {
           messages.value = Array.isArray(res.data) ? res.data : []
       }
    } else {
        error.value = '获取消息失败'
    }
  } catch (e) {
    console.error(e)
    error.value = '请求历史记录出错'
  } finally {
    loading.value = false
    nextTick(() => {
        scrollToBottom()
    })
  }
}

const scrollToBottom = () => {
    if (msgContainer.value) {
        msgContainer.value.scrollTop = msgContainer.value.scrollHeight
    }
}

const close = () => {
  emit('close')
}

const switchToChat = () => {
  emit('switch')
}

watch(() => props.visible, (val) => {
  if (val) {
    loadHistory()
  }
})
</script>

<style scoped>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
