import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as mtphotoApi from '@/api/mtphoto'

const MTPHOTO_FAVORITES_ALBUM_ID = 1

export interface MtPhotoAlbum {
  // 本地唯一 ID（用于 v-for key / 选中态），避免与上游保留相册（如收藏夹）ID 冲突
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

export interface MtPhotoFolderNode {
  id: number
  name: string
  path?: string
  cover?: string
  sCover?: string | null
  subFolderNum?: number
  subFileNum?: number
  fileType?: string
  trashNum?: number
}

export interface MtPhotoFolderFavorite {
  id: number
  folderId: number
  folderName: string
  folderPath: string
  coverMd5?: string
  tags: string[]
  note?: string
  createTime?: string
  updateTime?: string
}

export interface MtPhotoMediaItem {
  id: number
  md5: string
  type: 'image' | 'video'
  fileType?: string
  fileName?: string
  size?: string
  tokenAt?: string
  width?: number
  height?: number
  duration?: number | null
  day?: string
  status?: number
}

type MtPhotoFolderHistoryItem = {
  folderId: number | null
  folderName: string
  coverMd5?: string
}

const inferMediaType = (fileType: unknown): 'image' | 'video' => {
  const normalized = String(fileType ?? '')
    .trim()
    .toUpperCase()
  if (normalized === 'MP4' || normalized === 'MOV' || normalized === 'M4V' || normalized === 'AVI') {
    return 'video'
  }
  return 'image'
}

const normalizeFolderName = (path: string, fallback: string) => {
  const trimmedPath = String(path || '').trim()
  if (!trimmedPath) return fallback
  const normalized = trimmedPath.replace(/\\/g, '/')
  const parts = normalized.split('/').filter(Boolean)
  return parts[parts.length - 1] || fallback
}

const normalizeTags = (tags: unknown): string[] => {
  if (!Array.isArray(tags)) return []
  const seen = new Set<string>()
  const output: string[] = []
  for (const raw of tags) {
    const value = String(raw ?? '').trim()
    if (!value || seen.has(value)) continue
    seen.add(value)
    output.push(value)
  }
  return output
}

const firstCoverMD5 = (cover?: string, sCover?: string | null) => {
  const secondary = String(sCover ?? '').trim()
  if (secondary) return secondary
  const primary = String(cover ?? '').trim()
  if (!primary) return ''
  const first = primary.split(',')[0]
  return String(first ?? '').trim()
}

const mapFolderNode = (raw: any): MtPhotoFolderNode | null => {
  if (!raw || typeof raw !== 'object') return null
  const id = Number(raw.id)
  if (!Number.isFinite(id) || id <= 0) return null
  return {
    id,
    name: String(raw.name ?? '').trim() || `目录 ${id}`,
    path: raw.path ? String(raw.path) : undefined,
    cover: raw.cover ? String(raw.cover) : '',
    sCover: raw.s_cover === null || raw.s_cover === undefined ? null : String(raw.s_cover),
    subFolderNum: Number.isFinite(Number(raw.subFolderNum)) ? Number(raw.subFolderNum) : undefined,
    subFileNum: Number.isFinite(Number(raw.subFileNum)) ? Number(raw.subFileNum) : undefined,
    fileType: raw.fileType ? String(raw.fileType) : undefined,
    trashNum: Number.isFinite(Number(raw.trashNum)) ? Number(raw.trashNum) : undefined
  }
}

const mapFolderNodes = (rawList: any[]): MtPhotoFolderNode[] => {
  if (!Array.isArray(rawList)) return []
  const out: MtPhotoFolderNode[] = []
  for (const raw of rawList) {
    const item = mapFolderNode(raw)
    if (item) out.push(item)
  }
  return out
}

const mapMediaItem = (raw: any): MtPhotoMediaItem | null => {
  if (!raw || typeof raw !== 'object') return null

  const md5 = String(raw.md5 ?? raw.MD5 ?? '').trim()
  if (!md5) return null

  const id = Number(raw.id)
  if (!Number.isFinite(id) || id <= 0) return null

  const fileType = raw.fileType ? String(raw.fileType) : undefined
  const type = raw.type ? (String(raw.type) === 'video' ? 'video' : 'image') : inferMediaType(fileType)

  const widthValue = Number(raw.width)
  const heightValue = Number(raw.height)
  const durationValue = Number(raw.duration)
  const statusValue = Number(raw.status)

  return {
    id,
    md5,
    type,
    fileType,
    fileName: raw.fileName ? String(raw.fileName) : undefined,
    size: raw.size !== undefined && raw.size !== null ? String(raw.size) : undefined,
    tokenAt: raw.tokenAt ? String(raw.tokenAt) : undefined,
    width: Number.isFinite(widthValue) && widthValue > 0 ? widthValue : undefined,
    height: Number.isFinite(heightValue) && heightValue > 0 ? heightValue : undefined,
    duration: Number.isFinite(durationValue) ? durationValue : null,
    day: raw.day ? String(raw.day) : undefined,
    status: Number.isFinite(statusValue) ? statusValue : undefined
  }
}

const mapMediaItems = (rawList: any[]): MtPhotoMediaItem[] => {
  if (!Array.isArray(rawList)) return []
  const out: MtPhotoMediaItem[] = []
  for (const raw of rawList) {
    const item = mapMediaItem(raw)
    if (item) out.push(item)
  }
  return out
}

const mapFavoriteItem = (raw: any): MtPhotoFolderFavorite | null => {
  if (!raw || typeof raw !== 'object') return null
  const folderId = Number(raw.folderId)
  if (!Number.isFinite(folderId) || folderId <= 0) return null
  return {
    id: Number(raw.id || 0),
    folderId,
    folderName: String(raw.folderName ?? '').trim() || `目录 ${folderId}`,
    folderPath: String(raw.folderPath ?? '').trim(),
    coverMd5: raw.coverMd5 ? String(raw.coverMd5).trim() : '',
    tags: normalizeTags(raw.tags),
    note: raw.note ? String(raw.note).trim() : '',
    createTime: raw.createTime ? String(raw.createTime) : undefined,
    updateTime: raw.updateTime ? String(raw.updateTime) : undefined
  }
}

export const useMtPhotoStore = defineStore('mtphoto', () => {
  const showModal = ref(false)
  const mode = ref<'albums' | 'folders'>('albums')
  const view = ref<'albums' | 'album' | 'folders'>('albums')

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

  const folderList = ref<MtPhotoFolderNode[]>([])
  const folderFiles = ref<MtPhotoMediaItem[]>([])
  const folderLoading = ref(false)
  const folderPath = ref('')
  const folderCurrentId = ref<number | null>(null)
  const folderCurrentName = ref('根目录')
  const folderCurrentCoverMd5 = ref('')
  const folderPage = ref(1)
  const folderPageSize = ref(60)
  const folderTotal = ref(0)
  const folderTotalPages = ref(0)
  const folderHistory = ref<MtPhotoFolderHistoryItem[]>([{ folderId: null, folderName: '根目录' }])

  const folderFavorites = ref<MtPhotoFolderFavorite[]>([])
  const folderFavoritesLoading = ref(false)
  const folderFavoriteSaving = ref(false)

  const currentFolderFavorite = computed(() => {
    if (!folderCurrentId.value) return null
    return folderFavorites.value.find(item => item.folderId === folderCurrentId.value) || null
  })

  const resetAlbumState = () => {
    selectedAlbum.value = null
    mediaItems.value = []
    mediaPage.value = 1
    mediaPageSize.value = 60
    mediaTotal.value = 0
    mediaTotalPages.value = 0
  }

  const resetFolderState = () => {
    folderList.value = []
    folderFiles.value = []
    folderPath.value = ''
    folderCurrentId.value = null
    folderCurrentName.value = '根目录'
    folderCurrentCoverMd5.value = ''
    folderPage.value = 1
    folderPageSize.value = 60
    folderTotal.value = 0
    folderTotalPages.value = 0
    folderHistory.value = [{ folderId: null, folderName: '根目录' }]
  }

  const open = async () => {
    showModal.value = true
    mode.value = 'albums'
    view.value = 'albums'
    lastError.value = ''
    resetAlbumState()
    resetFolderState()
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

      void (async () => {
        try {
          const favRes = await mtphotoApi.getMtPhotoAlbumFiles(MTPHOTO_FAVORITES_ALBUM_ID, 1, 1)
          favorites.count = Number(favRes?.total || 0)
        } catch (e) {
          console.warn('加载 mtPhoto 收藏夹数量失败:', e)
        }
      })()
    } catch (e: any) {
      console.error('加载 mtPhoto 相册失败:', e)
      albums.value = []
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
    } finally {
      albumsLoading.value = false
    }
  }

  const openAlbum = async (album: MtPhotoAlbum) => {
    mode.value = 'albums'
    selectedAlbum.value = album
    view.value = 'album'
    mediaItems.value = []
    mediaPage.value = 1
    mediaTotal.value = 0
    mediaTotalPages.value = 0
    await loadAlbumPage(1)
  }

  const backToAlbums = () => {
    mode.value = 'albums'
    view.value = 'albums'
    resetAlbumState()
  }

  const loadAlbumPage = async (page: number) => {
    if (!selectedAlbum.value) return
    mediaLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoAlbumFiles(selectedAlbum.value.mtPhotoAlbumId, page, mediaPageSize.value)
      const data = mapMediaItems(Array.isArray(res?.data) ? res.data : [])

      if (page === 1) {
        mediaItems.value = data
      } else {
        mediaItems.value.push(...data)
      }

      mediaTotal.value = Number(res?.total || 0)
      mediaPage.value = Number(res?.page || page)
      mediaPageSize.value = Number(res?.pageSize || mediaPageSize.value)
      mediaTotalPages.value = Number(res?.totalPages || 0)
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

  const applyFolderContent = (res: any, append = false) => {
    const nodes = mapFolderNodes(Array.isArray(res?.folderList) ? res.folderList : [])
    const files = mapMediaItems(Array.isArray(res?.fileList) ? res.fileList : [])

    folderList.value = nodes
    if (append) {
      folderFiles.value.push(...files)
    } else {
      folderFiles.value = files
    }

    folderPath.value = String(res?.path ?? '')

    const page = Number(res?.page || (append ? folderPage.value + 1 : 1))
    const pageSize = Number(res?.pageSize || folderPageSize.value || 60)
    const totalFromAPI = Number(res?.total)
    const total = Number.isFinite(totalFromAPI) && totalFromAPI >= 0 ? totalFromAPI : (append ? folderTotal.value : files.length)

    folderPage.value = Number.isFinite(page) && page > 0 ? page : 1
    folderPageSize.value = Number.isFinite(pageSize) && pageSize > 0 ? pageSize : 60
    folderTotal.value = total

    const totalPagesFromAPI = Number(res?.totalPages)
    if (Number.isFinite(totalPagesFromAPI) && totalPagesFromAPI >= 0) {
      folderTotalPages.value = totalPagesFromAPI
    } else {
      const ps = Math.max(folderPageSize.value, 1)
      folderTotalPages.value = folderTotal.value > 0 ? Math.ceil(folderTotal.value / ps) : 0
    }
  }

  const loadFolderRoot = async () => {
    folderLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoFolderRoot()
      folderCurrentId.value = null
      folderCurrentName.value = '根目录'
      folderCurrentCoverMd5.value = ''
      folderHistory.value = [{ folderId: null, folderName: '根目录' }]
      applyFolderContent(res, false)
      lastError.value = ''
    } catch (e: any) {
      console.error('加载 mtPhoto 根目录失败:', e)
      resetFolderState()
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
    } finally {
      folderLoading.value = false
    }
  }

  const loadFolderByID = async (
    folderID: number,
    options?: {
      folderName?: string
      coverMd5?: string
      fromHistory?: boolean
      resetHistory?: boolean
    }
  ) => {
    if (!Number.isFinite(folderID) || folderID <= 0) return false
    folderLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoFolderContent(folderID, 1, folderPageSize.value)
      folderCurrentId.value = folderID
      folderCurrentName.value =
        String(options?.folderName || '').trim() || normalizeFolderName(String(res?.path || ''), `目录 ${folderID}`)
      folderCurrentCoverMd5.value =
        String(options?.coverMd5 || '').trim() ||
        currentFolderFavorite.value?.coverMd5 ||
        firstCoverMD5('', null)

      if (options?.resetHistory) {
        folderHistory.value = [
          { folderId: null, folderName: '根目录' },
          { folderId: folderID, folderName: folderCurrentName.value, coverMd5: folderCurrentCoverMd5.value }
        ]
      } else if (!options?.fromHistory) {
        const idx = folderHistory.value.findIndex(item => item.folderId === folderID)
        if (idx >= 0) {
          folderHistory.value = folderHistory.value.slice(0, idx + 1)
        } else {
          folderHistory.value.push({
            folderId: folderID,
            folderName: folderCurrentName.value,
            coverMd5: folderCurrentCoverMd5.value
          })
        }
      }

      applyFolderContent(res, false)
      lastError.value = ''
      return true
    } catch (e: any) {
      console.error('加载 mtPhoto 目录失败:', e)
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
      return false
    } finally {
      folderLoading.value = false
    }
  }

  const openFolder = async (folder: MtPhotoFolderNode) => {
    const coverMD5 = firstCoverMD5(folder.cover, folder.sCover)
    await loadFolderByID(folder.id, {
      folderName: folder.name,
      coverMd5: coverMD5
    })
  }

  const backFolder = async () => {
    if (folderLoading.value) return
    if (folderHistory.value.length <= 1) {
      await loadFolderRoot()
      return
    }
    const history = [...folderHistory.value]
    history.pop()
    const target = history[history.length - 1]
    if (!target || target.folderId === null) {
      await loadFolderRoot()
      return
    }
    folderHistory.value = history
    await loadFolderByID(target.folderId, {
      folderName: target.folderName,
      coverMd5: target.coverMd5,
      fromHistory: true
    })
  }

  const loadFolderMore = async () => {
    if (folderLoading.value) return
    if (!folderCurrentId.value) return
    if (folderTotalPages.value > 0 && folderPage.value >= folderTotalPages.value) return

    folderLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoFolderContent(folderCurrentId.value, folderPage.value + 1, folderPageSize.value)
      applyFolderContent(res, true)
      lastError.value = ''
    } catch (e: any) {
      console.error('加载更多目录图片失败:', e)
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
    } finally {
      folderLoading.value = false
    }
  }

  const loadFolderFavorites = async () => {
    folderFavoritesLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoFolderFavorites()
      const list = Array.isArray(res?.items) ? res.items : []
      folderFavorites.value = list
        .map((item: any) => mapFavoriteItem(item))
        .filter((item: MtPhotoFolderFavorite | null): item is MtPhotoFolderFavorite => item !== null)
      lastError.value = ''
    } catch (e: any) {
      console.error('加载目录收藏失败:', e)
      folderFavorites.value = []
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
    } finally {
      folderFavoritesLoading.value = false
    }
  }

  const openFavoriteFolder = async (favorite: MtPhotoFolderFavorite) => {
    if (!favorite || !favorite.folderId) return false
    const ok = await loadFolderByID(favorite.folderId, {
      folderName: favorite.folderName,
      coverMd5: favorite.coverMd5,
      resetHistory: true
    })
    if (!ok) return false
    return true
  }

  const upsertCurrentFolderFavorite = async (payload: { tags?: string[]; note?: string } = {}) => {
    if (!folderCurrentId.value) return false

    folderFavoriteSaving.value = true
    try {
      const tags = normalizeTags(payload.tags ?? [])
      const note = String(payload.note ?? '').trim()
      const folderName =
        String(folderCurrentName.value || '').trim() ||
        normalizeFolderName(folderPath.value, `目录 ${folderCurrentId.value}`)
      const coverMd5 = String(folderCurrentCoverMd5.value || '').trim()

      const res = await mtphotoApi.upsertMtPhotoFolderFavorite({
        folderId: folderCurrentId.value,
        folderName,
        folderPath: String(folderPath.value || '').trim(),
        coverMd5: coverMd5 || undefined,
        tags,
        note
      })

      const mapped = mapFavoriteItem(res?.item)
      if (res?.success && mapped) {
        const next = folderFavorites.value.filter(item => item.folderId !== mapped.folderId)
        next.unshift(mapped)
        folderFavorites.value = next
        folderCurrentCoverMd5.value = mapped.coverMd5 || folderCurrentCoverMd5.value
        return true
      }

      await loadFolderFavorites()
      return !!res?.success
    } catch (e: any) {
      console.error('保存目录收藏失败:', e)
      lastError.value = e?.response?.data?.error || e?.message || '保存失败'
      return false
    } finally {
      folderFavoriteSaving.value = false
    }
  }

  const removeFolderFavorite = async (folderID: number) => {
    if (!Number.isFinite(folderID) || folderID <= 0) return false
    folderFavoriteSaving.value = true
    try {
      const res = await mtphotoApi.removeMtPhotoFolderFavorite(folderID)
      if (res?.success) {
        folderFavorites.value = folderFavorites.value.filter(item => item.folderId !== folderID)
        return true
      }
      return false
    } catch (e: any) {
      console.error('移除目录收藏失败:', e)
      lastError.value = e?.response?.data?.error || e?.message || '移除失败'
      return false
    } finally {
      folderFavoriteSaving.value = false
    }
  }

  const switchMode = async (nextMode: 'albums' | 'folders') => {
    if (nextMode === mode.value) return
    mode.value = nextMode
    lastError.value = ''

    if (nextMode === 'albums') {
      view.value = 'albums'
      resetFolderState()
      return
    }

    view.value = 'folders'
    await Promise.all([loadFolderRoot(), loadFolderFavorites()])
  }

  const isCurrentFolderFavorited = computed(() => !!currentFolderFavorite.value)

  return {
    showModal,
    mode,
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
    folderList,
    folderFiles,
    folderLoading,
    folderPath,
    folderCurrentId,
    folderCurrentName,
    folderPage,
    folderPageSize,
    folderTotal,
    folderTotalPages,
    folderHistory,
    folderFavorites,
    folderFavoritesLoading,
    folderFavoriteSaving,
    currentFolderFavorite,
    isCurrentFolderFavorited,
    open,
    close,
    switchMode,
    loadAlbums,
    openAlbum,
    backToAlbums,
    loadMore,
    loadFolderRoot,
    openFolder,
    backFolder,
    loadFolderMore,
    loadFolderFavorites,
    openFavoriteFolder,
    upsertCurrentFolderFavorite,
    removeFolderFavorite
  }
})
