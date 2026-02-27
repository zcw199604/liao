import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as mtphotoApi from '@/api/mtphoto'
import { useSystemConfigStore } from '@/stores/systemConfig'

const MTPHOTO_FAVORITES_ALBUM_ID = 1
const DEFAULT_MTPHOTO_TIMELINE_DEFER_SUBFOLDER_THRESHOLD = 10

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

type MtPhotoFavoriteFilterMode = 'any' | 'all'
type MtPhotoFavoriteSortBy = 'updatedAt' | 'name' | 'tagCount'
type MtPhotoFavoriteSortOrder = 'asc' | 'desc'
type MtPhotoFavoriteGroupBy = 'none' | 'tag'

type MtPhotoFavoriteGroup = {
  key: string
  items: MtPhotoFolderFavorite[]
}

type OpenExternalFolderOptions = {
  folderId?: number
  folderPath?: string
  folderName?: string
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

const toTimestamp = (value?: string): number => {
  const ts = Date.parse(String(value || '').trim())
  return Number.isFinite(ts) ? ts : 0
}

const firstCoverMD5 = (cover?: string, sCover?: string | null) => {
  const secondary = String(sCover ?? '').trim()
  if (secondary) return secondary
  const primary = String(cover ?? '').trim()
  if (!primary) return ''
  const first = primary.split(',')[0]
  return String(first ?? '').trim()
}

const normalizeFolderPathForLookup = (value?: string) => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  let normalized = raw.replace(/\\/g, '/')
  normalized = normalized.replace(/\/{2,}/g, '/')
  if (!normalized.startsWith('/')) normalized = '/' + normalized
  if (normalized.length > 1 && normalized.endsWith('/')) {
    normalized = normalized.slice(0, -1)
  }
  return normalized
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
  const systemConfigStore = useSystemConfigStore()

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
  const folderTimelineDeferred = ref(false)
  const folderTimelineThreshold = ref(DEFAULT_MTPHOTO_TIMELINE_DEFER_SUBFOLDER_THRESHOLD)
  const folderHistory = ref<MtPhotoFolderHistoryItem[]>([{ folderId: null, folderName: '根目录' }])

  const folderFavorites = ref<MtPhotoFolderFavorite[]>([])
  const folderFavoritesLoading = ref(false)
  const folderFavoriteSaving = ref(false)
  const favoriteFilterInputKeyword = ref('')
  const favoriteFilterKeyword = ref('')
  const favoriteFilterMode = ref<MtPhotoFavoriteFilterMode>('any')
  const favoriteSortBy = ref<MtPhotoFavoriteSortBy>('updatedAt')
  const favoriteSortOrder = ref<MtPhotoFavoriteSortOrder>('desc')
  const favoriteGroupBy = ref<MtPhotoFavoriteGroupBy>('none')
  const favoriteEditingFolderId = ref<number | null>(null)
  const favoriteDraftTags = ref('')
  const favoriteDraftNote = ref('')
  let favoriteFilterDebounceTimer: ReturnType<typeof setTimeout> | null = null

  const currentFolderFavorite = computed(() => {
    if (!folderCurrentId.value) return null
    return folderFavorites.value.find(item => item.folderId === folderCurrentId.value) || null
  })

  const allUniqueTags = computed(() => {
    const seen = new Set<string>()
    const out: string[] = []
    for (const item of folderFavorites.value) {
      for (const rawTag of item.tags) {
        const tag = String(rawTag || '').trim()
        if (!tag || seen.has(tag)) continue
        seen.add(tag)
        out.push(tag)
      }
    }
    return out
  })

  const filteredFolderFavorites = computed(() => {
    const keyword = favoriteFilterKeyword.value.trim().toLowerCase()
    if (!keyword) return folderFavorites.value

    const tokens = keyword
      .split(/[\s,，]+/)
      .map(v => v.trim())
      .filter(Boolean)
    if (tokens.length === 0) return folderFavorites.value

    return folderFavorites.value.filter(item => {
      const tags = item.tags.map(tag => String(tag || '').toLowerCase())
      if (favoriteFilterMode.value === 'all') {
        return tokens.every(token => tags.some(tag => tag.includes(token)))
      }
      return tokens.some(token => tags.some(tag => tag.includes(token)))
    })
  })

  const sortedFolderFavorites = computed(() => {
    const list = [...filteredFolderFavorites.value]
    const order = favoriteSortOrder.value === 'asc' ? 1 : -1
    list.sort((a, b) => {
      let compared = 0
      if (favoriteSortBy.value === 'name') {
        compared = a.folderName.localeCompare(b.folderName, 'zh-CN')
      } else if (favoriteSortBy.value === 'tagCount') {
        compared = a.tags.length - b.tags.length
      } else {
        compared = toTimestamp(a.updateTime || a.createTime) - toTimestamp(b.updateTime || b.createTime)
      }
      if (compared === 0) compared = a.folderId - b.folderId
      return compared * order
    })
    return list
  })

  const groupedFolderFavorites = computed<MtPhotoFavoriteGroup[]>(() => {
    if (favoriteGroupBy.value !== 'tag') {
      return [{ key: '全部', items: sortedFolderFavorites.value }]
    }

    const grouped = new Map<string, MtPhotoFolderFavorite[]>()
    for (const item of sortedFolderFavorites.value) {
      const tags = item.tags.length ? item.tags : ['未标记']
      for (const tag of tags) {
        if (!grouped.has(tag)) grouped.set(tag, [])
        grouped.get(tag)!.push(item)
      }
    }
    return Array.from(grouped.entries()).map(([key, items]) => ({ key, items }))
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
    folderTimelineDeferred.value = false
    folderHistory.value = [{ folderId: null, folderName: '根目录' }]
  }

  const resetFavoriteViewState = () => {
    if (favoriteFilterDebounceTimer) {
      clearTimeout(favoriteFilterDebounceTimer)
      favoriteFilterDebounceTimer = null
    }
    favoriteFilterInputKeyword.value = ''
    favoriteFilterKeyword.value = ''
    favoriteFilterMode.value = 'any'
    favoriteSortBy.value = 'updatedAt'
    favoriteSortOrder.value = 'desc'
    favoriteGroupBy.value = 'none'
    favoriteEditingFolderId.value = null
    favoriteDraftTags.value = ''
    favoriteDraftNote.value = ''
  }

  const open = async () => {
    showModal.value = true
    mode.value = 'albums'
    view.value = 'albums'
    lastError.value = ''
    resetAlbumState()
    resetFolderState()
    resetFavoriteViewState()
    await loadAlbums()
  }

  const close = () => {
    resetFavoriteViewState()
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

  const normalizeTimelineThreshold = (raw: unknown) => {
    const value = Number(raw)
    if (!Number.isFinite(value) || value <= 0) {
      return DEFAULT_MTPHOTO_TIMELINE_DEFER_SUBFOLDER_THRESHOLD
    }
    if (value > 500) return 500
    return Math.floor(value)
  }

  const syncFolderTimelineThreshold = async () => {
    if (!systemConfigStore.loaded && !systemConfigStore.loading) {
      await systemConfigStore.loadSystemConfig()
    }
    folderTimelineThreshold.value = normalizeTimelineThreshold(systemConfigStore.mtPhotoTimelineDeferSubfolderThreshold)
  }

  const shouldDeferFolderTimeline = (subFolderCount: number) => {
    return Number.isFinite(subFolderCount) && subFolderCount > folderTimelineThreshold.value
  }

  const applyDeferredFolderTimelinePlaceholder = () => {
    folderTimelineDeferred.value = true
    folderFiles.value = []
    folderPage.value = 1
    folderTotalPages.value = 0
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
      folderTimelineDeferred.value = false
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
      subFolderNum?: number
      fromHistory?: boolean
      resetHistory?: boolean
    }
  ) => {
    if (!Number.isFinite(folderID) || folderID <= 0) return false
    folderLoading.value = true
    try {
      await syncFolderTimelineThreshold()
      const hintedSubFolderNum = Number(options?.subFolderNum)
      const hasSubFolderHint = Number.isFinite(hintedSubFolderNum) && hintedSubFolderNum >= 0
      const shouldLoadTimelineInitially = hasSubFolderHint
        ? !shouldDeferFolderTimeline(hintedSubFolderNum)
        : false

      const res = await mtphotoApi.getMtPhotoFolderContent(folderID, 1, folderPageSize.value, shouldLoadTimelineInitially)
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
      const subFolderCount = Array.isArray(res?.folderList) ? res.folderList.length : 0
      if (shouldDeferFolderTimeline(subFolderCount)) {
        applyDeferredFolderTimelinePlaceholder()
      } else if (!shouldLoadTimelineInitially) {
        const timelineRes = await mtphotoApi.getMtPhotoFolderContent(folderID, 1, folderPageSize.value, true)
        applyFolderContent(timelineRes, false)
        folderTimelineDeferred.value = false
      } else {
        folderTimelineDeferred.value = false
      }

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
      coverMd5: coverMD5,
      subFolderNum: folder.subFolderNum
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
    if (folderTimelineDeferred.value) return
    if (folderTotalPages.value > 0 && folderPage.value >= folderTotalPages.value) return

    folderLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoFolderContent(
        folderCurrentId.value,
        folderPage.value + 1,
        folderPageSize.value,
        true
      )
      applyFolderContent(res, true)
      lastError.value = ''
    } catch (e: any) {
      console.error('加载更多目录图片失败:', e)
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
    } finally {
      folderLoading.value = false
    }
  }

  const loadFolderTimeline = async () => {
    if (folderLoading.value) return false
    if (!folderCurrentId.value) return false
    if (!folderTimelineDeferred.value) return false

    folderLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoFolderContent(folderCurrentId.value, 1, folderPageSize.value, true)
      applyFolderContent(res, false)
      folderTimelineDeferred.value = false
      lastError.value = ''
      return true
    } catch (e: any) {
      console.error('加载目录时间线图片失败:', e)
      lastError.value = e?.response?.data?.error || e?.message || '加载失败'
      return false
    } finally {
      folderLoading.value = false
    }
  }

  const loadFolderFavorites = async (query?: mtphotoApi.MtPhotoFolderFavoritesQuery) => {
    folderFavoritesLoading.value = true
    try {
      const res = await mtphotoApi.getMtPhotoFolderFavorites(query)
      const list = Array.isArray(res?.items) ? res.items : []
      folderFavorites.value = list
        .map((item: any) => mapFavoriteItem(item))
        .filter((item: MtPhotoFolderFavorite | null): item is MtPhotoFolderFavorite => item !== null)
      if (
        favoriteEditingFolderId.value &&
        !folderFavorites.value.some(item => item.folderId === favoriteEditingFolderId.value)
      ) {
        cancelEditFavorite()
      }
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

  const setFavoriteFilterKeyword = (value: string, options: { debounceMs?: number; immediate?: boolean } = {}) => {
    const nextKeyword = String(value || '')
    favoriteFilterInputKeyword.value = nextKeyword
    const debounceMs = Number(options.debounceMs ?? 200)
    if (favoriteFilterDebounceTimer) {
      clearTimeout(favoriteFilterDebounceTimer)
      favoriteFilterDebounceTimer = null
    }
    if (options.immediate || debounceMs <= 0) {
      favoriteFilterKeyword.value = nextKeyword.trim()
      return
    }
    favoriteFilterDebounceTimer = setTimeout(() => {
      favoriteFilterKeyword.value = nextKeyword.trim()
      favoriteFilterDebounceTimer = null
    }, debounceMs)
  }

  const resetFavoriteFilter = () => {
    setFavoriteFilterKeyword('', { immediate: true })
    favoriteFilterMode.value = 'any'
    favoriteSortBy.value = 'updatedAt'
    favoriteSortOrder.value = 'desc'
    favoriteGroupBy.value = 'none'
  }

  const startEditFavorite = (favorite: MtPhotoFolderFavorite) => {
    if (!favorite || !favorite.folderId) return
    favoriteEditingFolderId.value = favorite.folderId
    favoriteDraftTags.value = favorite.tags.join(', ')
    favoriteDraftNote.value = favorite.note || ''
  }

  const cancelEditFavorite = () => {
    favoriteEditingFolderId.value = null
    favoriteDraftTags.value = ''
    favoriteDraftNote.value = ''
  }

  const upsertFolderFavorite = async (payload: {
    folderId: number
    folderName: string
    folderPath: string
    coverMd5?: string
    tags?: string[]
    note?: string
  }) => {
    const folderID = Number(payload.folderId)
    if (!Number.isFinite(folderID) || folderID <= 0) {
      lastError.value = 'folderId 参数非法'
      return false
    }

    const folderName = String(payload.folderName || '').trim() || `目录 ${folderID}`
    const path = String(payload.folderPath || '').trim()
    if (!path) {
      lastError.value = 'folderPath 不能为空'
      return false
    }

    folderFavoriteSaving.value = true
    try {
      const tags = normalizeTags(payload.tags ?? [])
      const note = String(payload.note ?? '').trim()
      const coverMd5 = String(payload.coverMd5 || '').trim()

      const res = await mtphotoApi.upsertMtPhotoFolderFavorite({
        folderId: folderID,
        folderName,
        folderPath: path,
        coverMd5: coverMd5 || undefined,
        tags,
        note
      })

      const mapped = mapFavoriteItem(res?.item)
      if (res?.success && mapped) {
        const index = folderFavorites.value.findIndex(item => item.folderId === mapped.folderId)
        if (index >= 0) {
          const next = [...folderFavorites.value]
          next[index] = mapped
          folderFavorites.value = next
        } else {
          folderFavorites.value = [mapped, ...folderFavorites.value]
        }

        if (folderCurrentId.value === mapped.folderId) {
          folderCurrentCoverMd5.value = mapped.coverMd5 || folderCurrentCoverMd5.value
        }

        cancelEditFavorite()
        lastError.value = ''
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

  const upsertCurrentFolderFavorite = async (payload: { tags?: string[]; note?: string } = {}) => {
    if (!folderCurrentId.value) return false

    const folderName =
      String(folderCurrentName.value || '').trim() ||
      normalizeFolderName(folderPath.value, `目录 ${folderCurrentId.value}`)
    const coverMd5 = String(folderCurrentCoverMd5.value || '').trim()
    const ok = await upsertFolderFavorite({
      folderId: folderCurrentId.value,
      folderName,
      folderPath: String(folderPath.value || '').trim(),
      coverMd5: coverMd5 || undefined,
      tags: normalizeTags(payload.tags ?? []),
      note: String(payload.note ?? '').trim()
    })
    if (ok) {
      const current = folderFavorites.value.find(item => item.folderId === folderCurrentId.value)
      favoriteDraftTags.value = current?.tags.join(', ') || ''
      favoriteDraftNote.value = current?.note || ''
    }
    return ok
  }

  const removeFolderFavorite = async (folderID: number) => {
    if (!Number.isFinite(folderID) || folderID <= 0) return false
    folderFavoriteSaving.value = true
    try {
      const res = await mtphotoApi.removeMtPhotoFolderFavorite(folderID)
      if (res?.success) {
        folderFavorites.value = folderFavorites.value.filter(item => item.folderId !== folderID)
        if (favoriteEditingFolderId.value === folderID) {
          cancelEditFavorite()
        }
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
      cancelEditFavorite()
      return
    }

    view.value = 'folders'
    await syncFolderTimelineThreshold()
    await Promise.all([loadFolderRoot(), loadFolderFavorites()])
  }

  const resolveFolderNodeByPath = async (folderPath: string) => {
    const normalizedPath = normalizeFolderPathForLookup(folderPath)
    if (!normalizedPath || normalizedPath === '/') return null

    const segments = normalizedPath.split('/').filter(Boolean)
    if (segments.length === 0) return null

    const rootRes = await mtphotoApi.getMtPhotoFolderRoot()
    let folderNodes = mapFolderNodes(Array.isArray(rootRes?.folderList) ? rootRes.folderList : [])

    let currentNode: MtPhotoFolderNode | null = null
    for (let idx = 0; idx < segments.length; idx++) {
      const expectedPath = '/' + segments.slice(0, idx + 1).join('/')
      const segment = segments[idx]

      const byPath = folderNodes.find(node => normalizeFolderPathForLookup(node.path) === expectedPath)
      const byName =
        byPath ||
        folderNodes.find(node => {
          const nodeName = String(node.name || '').trim()
          return nodeName === segment
        })
      const matched = byName || null
      if (!matched || !matched.id) return null

      currentNode = matched
      if (idx < segments.length - 1) {
        const subRes = await mtphotoApi.getMtPhotoFolderContent(matched.id, 1, 1, false)
        folderNodes = mapFolderNodes(Array.isArray(subRes?.folderList) ? subRes.folderList : [])
      }
    }
    return currentNode
  }

  const openFromExternalFolder = async (options: OpenExternalFolderOptions = {}) => {
    showModal.value = true
    mode.value = 'folders'
    view.value = 'folders'
    lastError.value = ''

    await syncFolderTimelineThreshold()
    await loadFolderFavorites()

    const folderID = Number(options.folderId || 0)
    if (Number.isFinite(folderID) && folderID > 0) {
      const ok = await loadFolderByID(folderID, {
        folderName: String(options.folderName || '').trim(),
        resetHistory: true
      })
      if (ok) return true
    }

    const normalizedPath = normalizeFolderPathForLookup(options.folderPath)
    if (normalizedPath) {
      try {
        const resolvedNode = await resolveFolderNodeByPath(normalizedPath)
        if (resolvedNode?.id) {
          const ok = await loadFolderByID(resolvedNode.id, {
            folderName: String(options.folderName || '').trim() || resolvedNode.name,
            coverMd5: firstCoverMD5(resolvedNode.cover, resolvedNode.sCover),
            subFolderNum: resolvedNode.subFolderNum,
            resetHistory: true
          })
          if (ok) return true
        }
      } catch (e) {
        console.warn('按路径定位 mtPhoto 目录失败:', e)
      }
    }

    await loadFolderRoot()
    return false
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
    folderTimelineDeferred,
    folderTimelineThreshold,
    folderHistory,
    folderFavorites,
    folderFavoritesLoading,
    folderFavoriteSaving,
    favoriteFilterInputKeyword,
    favoriteFilterKeyword,
    favoriteFilterMode,
    favoriteSortBy,
    favoriteSortOrder,
    favoriteGroupBy,
    favoriteEditingFolderId,
    favoriteDraftTags,
    favoriteDraftNote,
    allUniqueTags,
    filteredFolderFavorites,
    sortedFolderFavorites,
    groupedFolderFavorites,
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
    loadFolderTimeline,
    loadFolderFavorites,
    openFavoriteFolder,
    setFavoriteFilterKeyword,
    resetFavoriteFilter,
    startEditFavorite,
    cancelEditFavorite,
    upsertFolderFavorite,
    upsertCurrentFolderFavorite,
    removeFolderFavorite,
    openFromExternalFolder
  }
})
