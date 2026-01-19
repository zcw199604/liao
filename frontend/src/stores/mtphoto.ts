import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as mtphotoApi from '@/api/mtphoto'

const MTPHOTO_FAVORITES_ALBUM_ID = 1

export interface MtPhotoAlbum {
  // 本地唯一ID（用于 v-for key / 选中态），避免与上游保留相册（如收藏夹）ID 冲突
  id: number
  // 上游 mtPhoto 的相册ID（用于请求 filesV2/{id}）
  mtPhotoAlbumId: number
  name: string
  cover: string
  count: number
  isFavorites?: boolean
  startTime?: string
  endTime?: string
}

export interface MtPhotoMediaItem {
  id: number
  md5: string
  type: 'image' | 'video'
  fileType?: string
  width?: number
  height?: number
  duration?: number
  day?: string
}

export const useMtPhotoStore = defineStore('mtphoto', () => {
  const showModal = ref(false)
  const view = ref<'albums' | 'album'>('albums')

  const albums = ref<MtPhotoAlbum[]>([])
  const albumsLoading = ref(false)
  const lastError = ref('')

  const selectedAlbum = ref<MtPhotoAlbum | null>(null)

  const mediaItems = ref<MtPhotoMediaItem[]>([])
  const mediaLoading = ref(false)
  const mediaPage = ref(1)
  const mediaPageSize = ref(60)
  const mediaTotal = ref(0)
  const mediaTotalPages = ref(0)

  const open = async () => {
    showModal.value = true
    view.value = 'albums'
    selectedAlbum.value = null
    mediaItems.value = []
    mediaPage.value = 1
    mediaTotal.value = 0
    mediaTotalPages.value = 0
    lastError.value = ''
    await loadAlbums()
  }

  const close = () => {
    showModal.value = false
  }

  const loadAlbums = async () => {
    albumsLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoAlbums()
      const data = Array.isArray(res?.data) ? res.data : []

      // 统一映射为本地模型，并置顶插入“收藏夹”（封面为空，后续进入时再从 total 同步数量）
      const mapped: MtPhotoAlbum[] = data
        .filter((a: any) => a && typeof a === 'object')
        .map((a: any) => ({
          id: Number(a.id),
          mtPhotoAlbumId: Number(a.id),
          name: String(a.name ?? ''),
          cover: String(a.cover ?? ''),
          count: Number(a.count ?? 0),
          startTime: a.startTime ? String(a.startTime) : undefined,
          endTime: a.endTime ? String(a.endTime) : undefined
        }))
        // 上游可能存在保留相册（如收藏夹），避免与本地置顶入口重复
        .filter((a: MtPhotoAlbum) => a.mtPhotoAlbumId > 0 && a.mtPhotoAlbumId !== MTPHOTO_FAVORITES_ALBUM_ID)

      const favorites: MtPhotoAlbum = {
        id: -MTPHOTO_FAVORITES_ALBUM_ID,
        mtPhotoAlbumId: MTPHOTO_FAVORITES_ALBUM_ID,
        name: '收藏夹',
        cover: '',
        count: 0,
        isFavorites: true
      }

      albums.value = [favorites, ...mapped]
      lastError.value = ''
    } catch (e: any) {
      console.error('加载 mtPhoto 相册失败:', e)
      albums.value = []
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
    } finally {
      albumsLoading.value = false
    }
  }

  const openAlbum = async (album: MtPhotoAlbum) => {
    selectedAlbum.value = album
    view.value = 'album'
    mediaItems.value = []
    mediaPage.value = 1
    mediaTotal.value = 0
    mediaTotalPages.value = 0
    await loadAlbumPage(1)
  }

  const backToAlbums = () => {
    view.value = 'albums'
    selectedAlbum.value = null
    mediaItems.value = []
    mediaPage.value = 1
    mediaTotal.value = 0
    mediaTotalPages.value = 0
  }

  const loadAlbumPage = async (page: number) => {
    if (!selectedAlbum.value) return
    if (mediaLoading.value) return

    mediaLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoAlbumFiles(selectedAlbum.value.mtPhotoAlbumId, page, mediaPageSize.value)
      const data = Array.isArray(res?.data) ? res.data : []

      if (page === 1) {
        mediaItems.value = data
      } else {
        mediaItems.value.push(...data)
      }

      mediaTotal.value = Number(res?.total || 0)
      mediaPage.value = Number(res?.page || page)
      mediaPageSize.value = Number(res?.pageSize || mediaPageSize.value)
      mediaTotalPages.value = Number(res?.totalPages || 0)
      // 进入相册/收藏夹后以返回 total 同步数量，避免在列表页额外拉取
      selectedAlbum.value.count = mediaTotal.value
      lastError.value = ''
    } catch (e: any) {
      console.error('加载 mtPhoto 相册媒体失败:', e)
      if (page === 1) mediaItems.value = []
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
    } finally {
      mediaLoading.value = false
    }
  }

  const loadMore = async () => {
    if (!selectedAlbum.value) return
    if (mediaLoading.value) return
    if (mediaTotalPages.value > 0 && mediaPage.value >= mediaTotalPages.value) return
    await loadAlbumPage(mediaPage.value + 1)
  }

  return {
    showModal,
    view,
    albums,
    albumsLoading,
    lastError,
    selectedAlbum,
    mediaItems,
    mediaLoading,
    mediaPage,
    mediaPageSize,
    mediaTotal,
    mediaTotalPages,
    open,
    close,
    loadAlbums,
    openAlbum,
    backToAlbums,
    loadMore
  }
})
