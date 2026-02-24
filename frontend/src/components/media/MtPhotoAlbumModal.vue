<template>
  <teleport to="body">
    <div
      v-if="mtPhotoStore.showModal"
      class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center"
      @click="close"
    >
      <div
        :class="[
          'bg-surface flex flex-col min-h-0 transition-all duration-200 ease-out',
          isFullscreen
            ? 'w-full max-w-none h-full h-[100dvh] rounded-none shadow-none pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] pl-[env(safe-area-inset-left)] pr-[env(safe-area-inset-right)]'
            : 'w-[95%] max-w-[1680px] h-[90vh] h-[90dvh] rounded-2xl shadow-2xl'
        ]"
        @click.stop
      >
        <!-- 头部 -->
        <div class="flex items-center justify-between px-4 py-2.5 border-b border-line">
          <div class="flex items-center gap-2 min-w-0">
            <button
              v-if="showBackButton"
              class="w-8 h-8 flex items-center justify-center text-fg-muted hover:text-fg transition rounded-lg hover:bg-surface-3 flex-shrink-0"
              @click="handleBack"
              :title="mtPhotoStore.mode === 'albums' ? '返回相册列表' : '返回上级目录'"
            >
              <i class="fas fa-arrow-left"></i>
            </button>

            <i class="fas fa-photo-video text-pink-400 flex-shrink-0"></i>
            <h3 class="text-base font-semibold text-fg truncate">{{ titleText }}</h3>
            <span v-if="subTitleText" class="text-xs text-fg-subtle ml-2 flex-shrink-0">
              {{ subTitleText }}
            </span>
          </div>

          <div class="flex items-center gap-2">
            <div class="flex items-center bg-surface-3 rounded-lg p-1">
              <button
                class="px-3 py-1.5 text-xs rounded-md transition"
                :class="mtPhotoStore.mode === 'albums' ? 'bg-pink-500/20 text-pink-300' : 'text-fg-subtle hover:text-fg'"
                @click="switchMode('albums')"
              >
                相册
              </button>
              <button
                class="px-3 py-1.5 text-xs rounded-md transition"
                :class="mtPhotoStore.mode === 'folders' ? 'bg-pink-500/20 text-pink-300' : 'text-fg-subtle hover:text-fg'"
                @click="switchMode('folders')"
              >
                文件夹
              </button>
            </div>

            <button
              v-if="showLayoutSwitcher"
              @click="toggleLayout"
              class="w-8 h-8 flex items-center justify-center text-fg-muted hover:text-fg transition rounded-lg hover:bg-surface-3"
              :title="layoutMode === 'masonry' ? '切换到网格视图' : '切换到瀑布流视图'"
            >
              <i :class="layoutMode === 'masonry' ? 'fas fa-th' : 'fas fa-stream'"></i>
            </button>

            <button
              @click="toggleFullscreen"
              class="w-8 h-8 flex items-center justify-center text-fg-muted hover:text-fg transition rounded-lg hover:bg-surface-3"
              :title="isFullscreen ? '退出全屏' : '全屏'"
            >
              <i :class="isFullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
            </button>

            <button
              @click="close"
              class="w-8 h-8 flex items-center justify-center text-fg-muted hover:text-fg transition rounded-lg hover:bg-surface-3"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>
        </div>

        <!-- 错误提示 -->
        <div v-if="mtPhotoStore.lastError" class="px-6 py-3 text-xs text-red-400 border-b border-line">
          {{ mtPhotoStore.lastError }}
        </div>

        <!-- 相册模式 -->
        <template v-if="mtPhotoStore.mode === 'albums'">
          <!-- 相册列表 -->
          <div v-if="mtPhotoStore.view === 'albums'" class="flex-1 overflow-y-auto p-2 no-scrollbar">
            <div v-if="mtPhotoStore.albumsLoading" class="flex-1 flex items-center justify-center text-fg-subtle text-sm">
              加载中...
            </div>

            <div v-else-if="mtPhotoStore.albums.length > 0" class="grid grid-cols-2 sm:grid-cols-3 gap-2">
              <button
                v-for="album in mtPhotoStore.albums"
                :key="album.id"
                class="text-left rounded-xl overflow-hidden border border-line-strong hover:border-pink-500 transition-colors bg-surface-deep"
                @click="mtPhotoStore.openAlbum(album)"
              >
                <div class="aspect-square bg-black/30 overflow-hidden">
                  <MediaTile
                    v-if="album.cover"
                    :src="getThumbUrl('s260', album.cover)"
                    type="image"
                    class="w-full h-full"
                    :show-skeleton="false"
                  />
                  <div v-else class="w-full h-full flex items-center justify-center text-fg-subtle">
                    <i class="fas fa-images text-3xl opacity-40"></i>
                  </div>
                </div>
                <div class="p-3">
                  <div class="text-fg font-medium text-sm truncate">{{ album.name }}</div>
                  <div class="text-xs text-fg-subtle mt-1">{{ album.count ?? 0 }} 个</div>
                </div>
              </button>
            </div>

            <div v-else class="flex-1 flex items-center justify-center text-fg-subtle text-sm">
              暂无相册
            </div>
          </div>

          <!-- 相册媒体 -->
          <InfiniteMediaGrid
            v-else
            :items="mtPhotoStore.mediaItems"
            :loading="mtPhotoStore.mediaLoading"
            :finished="mtPhotoStore.mediaTotalPages > 0 && mtPhotoStore.mediaPage >= mtPhotoStore.mediaTotalPages"
            :total="mtPhotoStore.mediaTotal"
            :layout-mode="layoutMode"
            :item-key="(item, idx) => item.md5 + '-' + idx"
            @load-more="mtPhotoStore.loadMore"
          >
            <template #default="{ item }">
              <MediaTile
                :src="getThumbUrl('h220', item.md5)"
                type="image"
                class="w-full rounded-xl overflow-hidden cursor-pointer border border-line-strong hover:border-pink-500 transition-colors bg-surface-3"
                :class="layoutMode === 'grid' ? 'h-full' : ''"
                :aspect-ratio="
                  layoutMode === 'masonry' && item.width && item.height
                    ? Number(item.width) / Number(item.height)
                    : undefined
                "
                :style="layoutMode === 'masonry' ? { contain: 'paint' } : {}"
                :show-skeleton="false"
                @click="handleMediaClick(item)"
              >
                <template v-if="item.type === 'video'" #center>
                  <div class="absolute inset-0 flex items-center justify-center bg-black/30">
                    <i class="fas fa-play-circle text-white text-3xl"></i>
                  </div>
                </template>
              </MediaTile>
            </template>

            <template #empty>
              <div class="flex items-center justify-center text-fg-subtle text-sm h-full">
                暂无媒体
              </div>
            </template>

            <template #finished-text> 已加载全部 </template>
          </InfiniteMediaGrid>
        </template>

        <!-- 文件夹模式 -->
        <div v-else class="flex-1 min-h-0 flex relative">
          <button
            v-if="isMobileFavoritesOpen"
            class="lg:hidden absolute inset-0 z-10 bg-black/45"
            type="button"
            aria-label="关闭收藏目录面板"
            @click="isMobileFavoritesOpen = false"
          />

          <aside
            :class="[
              'min-h-0 flex-col',
              isMobileFavoritesOpen
                ? 'flex absolute inset-y-0 left-0 z-20 w-[88%] max-w-[360px] bg-surface border-r border-line'
                : 'hidden lg:flex lg:w-72 lg:border-r lg:border-line'
            ]"
          >
            <div class="px-4 py-3 border-b border-line flex items-center justify-between">
              <div class="text-sm font-semibold text-fg">目录收藏</div>
              <div class="flex items-center gap-2">
                <button
                  class="text-xs text-fg-subtle hover:text-fg"
                  @click="mtPhotoStore.loadFolderFavorites()"
                  :disabled="mtPhotoStore.folderFavoritesLoading"
                >
                  刷新
                </button>
                <button
                  v-if="isMobileFavoritesOpen"
                  class="lg:hidden text-xs text-fg-subtle hover:text-fg"
                  @click="isMobileFavoritesOpen = false"
                >
                  关闭
                </button>
              </div>
            </div>

            <div class="px-3 py-2 border-b border-line space-y-2">
              <input
                v-model="favoriteFilterInputProxy"
                class="w-full rounded-md border border-line-strong bg-surface-3 px-2.5 py-1.5 text-xs text-fg placeholder:text-fg-subtle focus:outline-none focus:border-pink-500"
                placeholder="按标签筛选（逗号/空格分隔）"
              />
              <div class="flex items-center gap-1.5">
                <button
                  class="px-2 py-1 text-[11px] rounded border"
                  :class="
                    mtPhotoStore.favoriteFilterMode === 'any'
                      ? 'border-pink-500/60 text-pink-300 bg-pink-500/10'
                      : 'border-line-strong text-fg-subtle hover:text-fg'
                  "
                  @click="mtPhotoStore.favoriteFilterMode = 'any'"
                >
                  任一标签
                </button>
                <button
                  class="px-2 py-1 text-[11px] rounded border"
                  :class="
                    mtPhotoStore.favoriteFilterMode === 'all'
                      ? 'border-pink-500/60 text-pink-300 bg-pink-500/10'
                      : 'border-line-strong text-fg-subtle hover:text-fg'
                  "
                  @click="mtPhotoStore.favoriteFilterMode = 'all'"
                >
                  全部匹配
                </button>
              </div>
              <div class="flex items-center gap-1.5">
                <select
                  v-model="mtPhotoStore.favoriteSortBy"
                  class="flex-1 rounded-md border border-line-strong bg-surface-3 px-2 py-1 text-[11px] text-fg focus:outline-none focus:border-pink-500"
                >
                  <option value="updatedAt">按更新时间</option>
                  <option value="name">按目录名</option>
                  <option value="tagCount">按标签数</option>
                </select>
                <button
                  class="px-2 py-1 text-[11px] rounded border border-line-strong text-fg-subtle hover:text-fg"
                  @click="toggleFavoriteSortOrder"
                >
                  {{ mtPhotoStore.favoriteSortOrder === 'asc' ? '升序' : '降序' }}
                </button>
              </div>
              <div v-if="mtPhotoStore.allUniqueTags.length" class="flex flex-wrap gap-1">
                <button
                  v-for="tag in mtPhotoStore.allUniqueTags.slice(0, 20)"
                  :key="`favorite-tag-${tag}`"
                  class="px-2 py-0.5 rounded bg-pink-500/15 text-pink-300 text-[11px] hover:bg-pink-500/25"
                  @click="appendFavoriteFilterTag(tag)"
                >
                  #{{ tag }}
                </button>
              </div>
            </div>

            <div v-if="mtPhotoStore.folderFavoritesLoading" class="p-4 text-xs text-fg-subtle">加载中...</div>
            <div v-else-if="mtPhotoStore.folderFavorites.length === 0" class="p-4 text-xs text-fg-subtle">
              暂无收藏目录
            </div>
            <div v-else-if="mtPhotoStore.sortedFolderFavorites.length === 0" class="p-4 text-xs text-fg-subtle">
              无匹配收藏目录
            </div>
            <div v-else class="flex-1 overflow-y-auto no-scrollbar p-2 space-y-2">
              <div
                v-for="item in mtPhotoStore.sortedFolderFavorites"
                :key="item.folderId"
                class="rounded-lg border border-line-strong bg-surface-3 hover:border-pink-500 transition-colors p-3"
              >
                <button class="w-full text-left" @click="handleOpenFavorite(item)">
                  <div class="text-sm font-medium text-fg truncate">{{ item.folderName }}</div>
                  <div class="text-[11px] text-fg-subtle truncate mt-1">{{ item.folderPath || '/' }}</div>
                  <div v-if="item.tags.length" class="mt-2 flex flex-wrap gap-1">
                    <span
                      v-for="tag in item.tags.slice(0, 3)"
                      :key="`${item.folderId}-${tag}`"
                      class="px-2 py-0.5 rounded bg-pink-500/15 text-pink-300 text-[11px]"
                    >
                      #{{ tag }}
                    </span>
                    <span v-if="item.tags.length > 3" class="text-[11px] text-fg-subtle">+{{ item.tags.length - 3 }}</span>
                  </div>
                  <div v-if="item.note" class="mt-2 text-[11px] text-fg-subtle line-clamp-2">{{ item.note }}</div>
                  <div v-if="item.updateTime" class="mt-2 text-[11px] text-fg-subtle">更新：{{ item.updateTime }}</div>
                </button>
                <div class="mt-2 flex items-center justify-end gap-3">
                  <button class="text-[11px] text-pink-300 hover:text-pink-200" @click="openFavoriteEditor(item)">
                    编辑
                  </button>
                  <button class="text-[11px] text-red-300 hover:text-red-200" @click="removeFavorite(item.folderId)">
                    移除
                  </button>
                </div>
              </div>
            </div>
          </aside>

          <div class="flex-1 min-h-0 flex flex-col">
            <div class="px-4 py-2 border-b border-line bg-surface-2 shrink-0">
              <div class="flex items-center gap-2">
                <span class="text-xs text-fg-subtle shrink-0">路径：</span>
                <span class="text-sm text-fg truncate flex-1 min-w-0">{{ mtPhotoStore.folderPath || '/' }}</span>
                <button
                  class="lg:hidden shrink-0 px-2.5 py-1 text-xs rounded-lg border border-line-strong text-fg-subtle hover:text-fg hover:border-pink-500 transition"
                  @click="isMobileFavoritesOpen = true"
                >
                  收藏夹
                </button>
                <button
                  class="shrink-0 px-2.5 py-1 text-xs rounded-lg border border-line-strong hover:border-pink-500 transition"
                  :class="mtPhotoStore.currentFolderFavorite ? 'text-pink-300 border-pink-500/50' : 'text-fg-subtle'"
                  :disabled="!mtPhotoStore.folderCurrentId"
                  @click="isFavoriteEditOpen = !isFavoriteEditOpen"
                >
                  {{ mtPhotoStore.currentFolderFavorite ? '★ 已收藏' : '☆ 收藏' }}
                </button>
              </div>
            </div>

            <div v-show="isFavoriteEditOpen" class="px-4 py-3 border-b border-line bg-surface-2 shrink-0">
              <div class="grid grid-cols-1 xl:grid-cols-[1fr_auto] gap-3 items-start">
                <div class="space-y-2">
                  <input
                    v-model="favoriteTagsInput"
                    class="w-full rounded-lg border border-line-strong bg-surface-3 px-3 py-2 text-sm text-fg placeholder:text-fg-subtle focus:outline-none focus:border-pink-500"
                    placeholder="标签（逗号分隔，例如：旅行, 常用）"
                    :disabled="!mtPhotoStore.folderCurrentId || mtPhotoStore.folderFavoriteSaving"
                  />
                  <textarea
                    v-model="favoriteNoteInput"
                    rows="2"
                    class="w-full rounded-lg border border-line-strong bg-surface-3 px-3 py-2 text-sm text-fg placeholder:text-fg-subtle focus:outline-none focus:border-pink-500 resize-y"
                    placeholder="备注（可选）"
                    :disabled="!mtPhotoStore.folderCurrentId || mtPhotoStore.folderFavoriteSaving"
                  />
                </div>

                <div class="flex items-center gap-2 justify-start xl:justify-end">
                  <button
                    class="px-3 py-2 rounded-lg bg-pink-500 text-white text-sm hover:bg-pink-600 disabled:opacity-50"
                    :disabled="!mtPhotoStore.folderCurrentId || mtPhotoStore.folderFavoriteSaving"
                    @click="saveCurrentFolderFavorite"
                  >
                    {{ mtPhotoStore.currentFolderFavorite ? '更新收藏' : '收藏当前目录' }}
                  </button>
                  <button
                    class="px-3 py-2 rounded-lg border border-line-strong text-sm text-fg-subtle hover:text-fg hover:border-red-400 disabled:opacity-50"
                    :disabled="!mtPhotoStore.currentFolderFavorite || mtPhotoStore.folderFavoriteSaving"
                    @click="removeCurrentFolderFavorite"
                  >
                    取消收藏
                  </button>
                </div>
              </div>
            </div>

            <div class="px-2 pt-2 pb-1 shrink-0">
              <div class="flex items-center justify-between gap-2 mb-1.5">
                <div class="text-xs text-fg-subtle">子文件夹</div>
                <input
                  v-if="mtPhotoStore.folderList.length > 1"
                  v-model="folderFilter"
                  type="text"
                  class="h-7 px-2 text-xs rounded-md border border-line-strong bg-surface-3 focus:outline-none focus:border-pink-500 w-40"
                  placeholder="搜索目录..."
                />
              </div>

              <div class="max-h-[28vh] lg:max-h-[34vh] overflow-y-auto pr-1">
                <div v-if="mtPhotoStore.folderLoading && mtPhotoStore.folderList.length === 0" class="text-xs text-fg-subtle px-1 py-2">
                  加载中...
                </div>
                <div v-else-if="folderFilter.trim() && filteredFolderList.length === 0" class="text-xs text-fg-subtle px-1 py-2">
                  无匹配目录
                </div>
                <div v-else-if="mtPhotoStore.folderList.length === 0" class="text-xs text-fg-subtle px-1 py-2">
                  当前目录无子文件夹
                </div>
                <div v-else class="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-5 gap-2">
                  <button
                    v-for="folder in filteredFolderList"
                    :key="folder.id"
                    class="text-left rounded-lg border border-line-strong bg-surface-3 hover:border-pink-500 transition-colors p-2"
                    @click="mtPhotoStore.openFolder(folder)"
                  >
                    <div class="text-sm text-fg truncate">{{ folder.name }}</div>
                    <div class="text-[11px] text-fg-subtle mt-1">
                      {{ folder.subFileNum ?? 0 }} 图 · {{ folder.subFolderNum ?? 0 }} 目录
                    </div>
                  </button>
                </div>
              </div>
            </div>

            <div
              v-if="mtPhotoStore.folderTimelineDeferred"
              class="px-4 py-2 border-t border-line bg-surface-2/60 shrink-0 flex flex-wrap items-center gap-2"
            >
              <button
                class="px-3 py-1.5 text-xs rounded-lg border border-line-strong text-fg-subtle hover:text-fg hover:border-pink-500 transition disabled:opacity-50"
                :disabled="mtPhotoStore.folderLoading"
                @click="loadFolderTimeline"
              >
                {{ mtPhotoStore.folderLoading ? '加载中...' : '加载时间线图片' }}
              </button>
              <span class="text-xs text-fg-subtle">
                当前目录子文件夹超过 {{ mtPhotoStore.folderTimelineThreshold }} 个，已暂停自动加载时间线预览
              </span>
            </div>

            <InfiniteMediaGrid
              :items="mtPhotoStore.folderFiles"
              :loading="mtPhotoStore.folderLoading"
              :finished="
                mtPhotoStore.folderTimelineDeferred ||
                (mtPhotoStore.folderTotalPages > 0 && mtPhotoStore.folderPage >= mtPhotoStore.folderTotalPages)
              "
              :total="mtPhotoStore.folderTotal"
              :layout-mode="layoutMode"
              :item-key="(item, idx) => item.md5 + '-' + idx"
              @load-more="mtPhotoStore.loadFolderMore"
            >
              <template #default="{ item }">
                <MediaTile
                  :src="getThumbUrl('h220', item.md5)"
                  type="image"
                  class="w-full rounded-xl overflow-hidden cursor-pointer border border-line-strong hover:border-pink-500 transition-colors bg-surface-3"
                  :class="layoutMode === 'grid' ? 'h-full' : ''"
                  :aspect-ratio="
                    layoutMode === 'masonry' && item.width && item.height
                      ? Number(item.width) / Number(item.height)
                      : undefined
                  "
                  :style="layoutMode === 'masonry' ? { contain: 'paint' } : {}"
                  :show-skeleton="false"
                  @click="handleMediaClick(item)"
                >
                  <template v-if="item.type === 'video'" #center>
                    <div class="absolute inset-0 flex items-center justify-center bg-black/30">
                      <i class="fas fa-play-circle text-white text-3xl"></i>
                    </div>
                  </template>
                </MediaTile>
              </template>

              <template #empty>
                <div class="flex items-center justify-center text-fg-subtle text-sm h-full">
                  {{
                    mtPhotoStore.folderTimelineDeferred
                      ? '已暂停自动加载时间线，点击上方按钮后查看'
                      : '当前目录暂无图片'
                  }}
                </div>
              </template>

              <template #finished-text> 已加载全部 </template>
            </InfiniteMediaGrid>
          </div>
        </div>
      </div>

      <div
        v-if="isFavoriteEditorOpen && favoriteEditorTarget"
        class="fixed inset-0 z-[95] bg-black/60 flex items-center justify-center p-4"
        @click="closeFavoriteEditor"
      >
        <div
          class="w-full max-w-lg rounded-xl border border-line-strong bg-surface p-4 lg:p-5 shadow-2xl"
          @click.stop
        >
          <div class="flex items-center justify-between gap-2">
            <div class="min-w-0">
              <div class="text-sm font-semibold text-fg truncate">编辑收藏目录</div>
              <div class="text-xs text-fg-subtle truncate mt-1">{{ favoriteEditorTarget.folderName }}</div>
            </div>
            <button class="w-7 h-7 rounded-md text-fg-subtle hover:text-fg hover:bg-surface-3" @click="closeFavoriteEditor">
              <i class="fas fa-times"></i>
            </button>
          </div>

          <div class="mt-4 space-y-3">
            <div>
              <div class="text-xs text-fg-subtle mb-1">标签（逗号分隔）</div>
              <input
                v-model="mtPhotoStore.favoriteDraftTags"
                class="w-full rounded-lg border border-line-strong bg-surface-3 px-3 py-2 text-sm text-fg placeholder:text-fg-subtle focus:outline-none focus:border-pink-500"
                placeholder="例如：旅行, 常用, 人像"
              />
            </div>

            <div>
              <div class="text-xs text-fg-subtle mb-1">备注</div>
              <textarea
                v-model="mtPhotoStore.favoriteDraftNote"
                rows="3"
                class="w-full rounded-lg border border-line-strong bg-surface-3 px-3 py-2 text-sm text-fg placeholder:text-fg-subtle focus:outline-none focus:border-pink-500 resize-y"
                placeholder="可选备注"
              />
            </div>
          </div>

          <div class="mt-4 flex items-center justify-end gap-2">
            <button
              class="px-3 py-2 rounded-lg border border-line-strong text-sm text-fg-subtle hover:text-fg"
              :disabled="mtPhotoStore.folderFavoriteSaving"
              @click="closeFavoriteEditor"
            >
              取消
            </button>
            <button
              class="px-3 py-2 rounded-lg bg-pink-500 text-white text-sm hover:bg-pink-600 disabled:opacity-60"
              :disabled="mtPhotoStore.folderFavoriteSaving"
              @click="saveFavoriteEditor"
            >
              保存
            </button>
          </div>
        </div>
      </div>

      <MediaPreview
        v-model:visible="showPreview"
        :url="previewUrl"
        :type="previewType"
        :can-upload="previewCanUpload"
        :media-list="previewMediaList"
        :resolve-original-filename="resolveMtPhotoOriginalFilename"
        @upload="confirmImportUpload"
        @media-change="handlePreviewMediaChange"
      />
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useMtPhotoStore, type MtPhotoFolderFavorite, type MtPhotoMediaItem } from '@/stores/mtphoto'
import { useUserStore } from '@/stores/user'
import { useToast } from '@/composables/useToast'
import { useModalFullscreen } from '@/composables/useModalFullscreen'
import * as mtphotoApi from '@/api/mtphoto'
import MediaPreview from '@/components/media/MediaPreview.vue'
import InfiniteMediaGrid from '@/components/common/InfiniteMediaGrid.vue'
import MediaTile from '@/components/common/MediaTile.vue'
import type { UploadedMedia } from '@/types'

const mtPhotoStore = useMtPhotoStore()
const userStore = useUserStore()
const { show } = useToast()

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewCanUpload = ref(true)
const previewMediaList = ref<UploadedMedia[]>([])
const previewMD5 = ref('')

const favoriteTagsInput = ref('')
const favoriteNoteInput = ref('')
const folderFilter = ref('')
const isMobileFavoritesOpen = ref(false)
const isFavoriteEditOpen = ref(false)

const filteredFolderList = computed(() => {
  const keyword = folderFilter.value.trim().toLowerCase()
  if (!keyword) return mtPhotoStore.folderList
  return mtPhotoStore.folderList.filter(folder => String(folder.name || '').toLowerCase().includes(keyword))
})

const favoriteFilterInputProxy = computed({
  get: () => mtPhotoStore.favoriteFilterInputKeyword,
  set: (value: string) => mtPhotoStore.setFavoriteFilterKeyword(value, { debounceMs: 200 })
})

const isFavoriteEditorOpen = computed(() => mtPhotoStore.favoriteEditingFolderId !== null)

const favoriteEditorTarget = computed(() => {
  if (!mtPhotoStore.favoriteEditingFolderId) return null
  return mtPhotoStore.folderFavorites.find(item => item.folderId === mtPhotoStore.favoriteEditingFolderId) || null
})

// 真实文件名解析缓存：md5 -> basename(filename)
const mtPhotoOriginalFilenameCache = new Map<string, string>()

const extractBasename = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  const normalized = raw.replace(/\\/g, '/')
  const withoutQuery = normalized.split('?')[0] || ''
  const withoutHash = withoutQuery.split('#')[0] || ''
  const parts = withoutHash.split('/').filter(Boolean)
  return parts[parts.length - 1] || ''
}

const resolveMtPhotoOriginalFilename = async (media: UploadedMedia): Promise<string> => {
  const md5Value = String(media.md5 || '').trim()
  if (!md5Value) return ''
  const cached = mtPhotoOriginalFilenameCache.get(md5Value)
  if (cached) return cached

  try {
    const res = await mtphotoApi.resolveMtPhotoFilePath(md5Value)
    const filename = extractBasename(String(res?.filePath || ''))
    if (filename) {
      mtPhotoOriginalFilenameCache.set(md5Value, filename)
      return filename
    }
  } catch (e) {
    console.warn('解析 mtPhoto 文件名失败:', e)
  }

  return ''
}

// 布局模式：'masonry' | 'grid'（与“全站图片库”保持一致）
const layoutMode = ref<'masonry' | 'grid'>(
  (localStorage.getItem('media_layout_mode') as 'masonry' | 'grid') || 'masonry'
)

const toggleLayout = () => {
  layoutMode.value = layoutMode.value === 'masonry' ? 'grid' : 'masonry'
  localStorage.setItem('media_layout_mode', layoutMode.value)
}

const showBackButton = computed(() => {
  if (mtPhotoStore.mode === 'albums') return mtPhotoStore.view === 'album'
  return mtPhotoStore.folderCurrentId !== null
})

const showLayoutSwitcher = computed(() => {
  if (mtPhotoStore.mode === 'albums') return mtPhotoStore.view === 'album'
  return true
})

const titleText = computed(() => {
  if (mtPhotoStore.mode === 'albums') {
    if (mtPhotoStore.view === 'albums') return 'mtPhoto 相册'
    return mtPhotoStore.selectedAlbum?.name || 'mtPhoto 相册'
  }
  return mtPhotoStore.folderCurrentName || 'mtPhoto 文件夹'
})

const subTitleText = computed(() => {
  if (mtPhotoStore.mode === 'albums') {
    if (mtPhotoStore.view === 'albums') return mtPhotoStore.albums.length ? `(共 ${mtPhotoStore.albums.length} 个)` : ''
    if (mtPhotoStore.selectedAlbum) return `(共 ${mtPhotoStore.selectedAlbum.count ?? 0} 个)`
    return ''
  }
  return `(共 ${mtPhotoStore.folderTotal || mtPhotoStore.folderFiles.length} 个)`
})

const currentMediaItems = computed(() =>
  mtPhotoStore.mode === 'folders' ? mtPhotoStore.folderFiles : mtPhotoStore.mediaItems
)

const syncFavoriteDraft = () => {
  const current = mtPhotoStore.currentFolderFavorite
  favoriteTagsInput.value = current?.tags.join(', ') || ''
  favoriteNoteInput.value = current?.note || ''
}

watch(
  () => mtPhotoStore.currentFolderFavorite,
  () => {
    if (mtPhotoStore.mode === 'folders') {
      syncFavoriteDraft()
    }
  },
  { immediate: true }
)

watch(
  () => mtPhotoStore.folderCurrentId,
  () => {
    folderFilter.value = ''
    isMobileFavoritesOpen.value = false
    isFavoriteEditOpen.value = false
    mtPhotoStore.cancelEditFavorite()
    if (!mtPhotoStore.currentFolderFavorite) {
      favoriteTagsInput.value = ''
      favoriteNoteInput.value = ''
    }
  }
)

const splitTags = (raw: string) => {
  const seen = new Set<string>()
  const out: string[] = []
  raw
    .split(/[,，\n]/)
    .map(v => v.trim())
    .forEach(tag => {
      if (!tag || seen.has(tag)) return
      seen.add(tag)
      out.push(tag)
    })
  return out
}

const toggleFavoriteSortOrder = () => {
  mtPhotoStore.favoriteSortOrder = mtPhotoStore.favoriteSortOrder === 'asc' ? 'desc' : 'asc'
}

const appendFavoriteFilterTag = (tag: string) => {
  const current = String(mtPhotoStore.favoriteFilterInputKeyword || '')
    .split(/[\s,，]+/)
    .map(v => v.trim())
    .filter(Boolean)
  if (current.includes(tag)) return
  const next = [...current, tag].join(', ')
  mtPhotoStore.setFavoriteFilterKeyword(next, { immediate: true })
}

const openFavoriteEditor = (favorite: MtPhotoFolderFavorite) => {
  mtPhotoStore.startEditFavorite(favorite)
}

const closeFavoriteEditor = () => {
  mtPhotoStore.cancelEditFavorite()
}

const saveFavoriteEditor = async () => {
  const target = favoriteEditorTarget.value
  if (!target) return

  const ok = await mtPhotoStore.upsertFolderFavorite({
    folderId: target.folderId,
    folderName: target.folderName,
    folderPath: target.folderPath,
    coverMd5: target.coverMd5,
    tags: splitTags(mtPhotoStore.favoriteDraftTags),
    note: mtPhotoStore.favoriteDraftNote
  })
  if (ok) {
    if (mtPhotoStore.folderCurrentId === target.folderId) {
      syncFavoriteDraft()
    }
    show('收藏标签已更新')
  } else {
    show(mtPhotoStore.lastError || '更新失败')
  }
}

const saveCurrentFolderFavorite = async () => {
  if (!mtPhotoStore.folderCurrentId) {
    show('请先进入一个具体目录后再收藏')
    return
  }

  const ok = await mtPhotoStore.upsertCurrentFolderFavorite({
    tags: splitTags(favoriteTagsInput.value),
    note: favoriteNoteInput.value
  })

  if (ok) {
    show('目录收藏已保存')
  } else {
    show(mtPhotoStore.lastError || '保存目录收藏失败')
  }
}

const removeCurrentFolderFavorite = async () => {
  if (!mtPhotoStore.currentFolderFavorite) return
  const ok = await mtPhotoStore.removeFolderFavorite(mtPhotoStore.currentFolderFavorite.folderId)
  if (ok) {
    favoriteTagsInput.value = ''
    favoriteNoteInput.value = ''
    show('已取消目录收藏')
  } else {
    show(mtPhotoStore.lastError || '取消收藏失败')
  }
}

const removeFavorite = async (folderId: number) => {
  const ok = await mtPhotoStore.removeFolderFavorite(folderId)
  if (ok) {
    show('已移除收藏目录')
  } else {
    show(mtPhotoStore.lastError || '移除失败')
  }
}

const handleOpenFavorite = async (favorite: MtPhotoFolderFavorite) => {
  const ok = await mtPhotoStore.openFavoriteFolder(favorite)
  if (ok) {
    isMobileFavoritesOpen.value = false
    closeFavoriteEditor()
  } else {
    show('该收藏目录可能已失效（目录被删除或无权限），可直接移除该收藏')
  }
}

const switchMode = async (mode: 'albums' | 'folders') => {
  if (mode !== 'folders') {
    isMobileFavoritesOpen.value = false
    isFavoriteEditOpen.value = false
    closeFavoriteEditor()
  }
  await mtPhotoStore.switchMode(mode)
}

const loadFolderTimeline = async () => {
  const ok = await mtPhotoStore.loadFolderTimeline()
  if (!ok && mtPhotoStore.lastError) {
    show(mtPhotoStore.lastError)
  }
}

const getThumbUrl = (size: 's260' | 'h220', md5: string) => {
  const safeMD5 = encodeURIComponent(md5 || '')
  return `/api/getMtPhotoThumb?size=${size}&md5=${safeMD5}`
}

const getOriginalDownloadUrl = (id: number, md5: string) => {
  const safeID = encodeURIComponent(String(id || ''))
  const safeMD5 = encodeURIComponent(md5 || '')
  return `/api/downloadMtPhotoOriginal?id=${safeID}&md5=${safeMD5}`
}

const close = () => {
  mtPhotoStore.close()
  showPreview.value = false
  previewUrl.value = ''
  previewMediaList.value = []
  previewMD5.value = ''
  isMobileFavoritesOpen.value = false
  isFavoriteEditOpen.value = false
  closeFavoriteEditor()
}

const { isFullscreen, toggleFullscreen } = useModalFullscreen({
  isModalOpen: () => mtPhotoStore.showModal,
  isBlocked: () => showPreview.value,
  onRequestClose: close
})

const handleBack = async () => {
  if (mtPhotoStore.mode === 'albums') {
    mtPhotoStore.backToAlbums()
    return
  }
  await mtPhotoStore.backFolder()
}

const handleMediaClick = async (item: MtPhotoMediaItem) => {
  previewMD5.value = item.md5
  previewType.value = item.type
  previewCanUpload.value = !!userStore.currentUser

  // 图片直接用网关缩略图预览；视频则解析本地路径以便播放
  previewUrl.value = getThumbUrl('h220', item.md5)
  previewMediaList.value = []
  if (item.type === 'image') {
    // 仅在“点图片”时启用画廊模式：左右切换浏览当前已加载的图片列表。
    const list: UploadedMedia[] = currentMediaItems.value
      .filter(m => m.type === 'image')
      .map(m => ({
        url: getThumbUrl('h220', m.md5),
        type: 'image',
        downloadUrl: getOriginalDownloadUrl(m.id, m.md5),
        md5: m.md5,
        originalFilename: mtPhotoOriginalFilenameCache.get(m.md5) || m.fileName,
        fileExtension: m.fileType ? String(m.fileType).trim().toLowerCase() : undefined,
        width: m.width,
        height: m.height,
        duration: m.duration ?? undefined,
        day: m.day
      }))
    previewMediaList.value = list
  }
  if (item.type === 'video') {
    try {
      const res = await mtphotoApi.resolveMtPhotoFilePath(item.md5)
      if (res?.filePath) {
        previewUrl.value = res.filePath
        const filename = extractBasename(res.filePath)
        if (filename) {
          mtPhotoOriginalFilenameCache.set(item.md5, filename)
        }
      }
    } catch {
      // ignore
    }
    previewMediaList.value = [
      {
        url: previewUrl.value,
        type: 'video',
        md5: item.md5,
        originalFilename: mtPhotoOriginalFilenameCache.get(item.md5) || item.fileName,
        fileExtension: item.fileType ? String(item.fileType).trim().toLowerCase() : undefined,
        width: item.width,
        height: item.height,
        duration: item.duration ?? undefined,
        day: item.day
      }
    ]
  }

  showPreview.value = true
}

const handlePreviewMediaChange = (media: UploadedMedia) => {
  // 预览内部切换后，同步当前媒体，确保“上传此图片”导入的是当前所见内容。
  previewUrl.value = media.url || previewUrl.value
  previewType.value = media.type || previewType.value
  if (media.md5) {
    previewMD5.value = media.md5
  }
}

const confirmImportUpload = async () => {
  if (!userStore.currentUser) {
    show('请先选择身份后再导入上传')
    return
  }
  if (!previewMD5.value) return

  try {
    const res = await mtphotoApi.importMtPhotoMedia({
      userid: userStore.currentUser.id,
      md5: previewMD5.value
    })

    if (res?.state === 'OK' && res.localPath) {
      const dedup = !!res.dedup
      show(dedup ? '已存在（去重复用）' : '已导入到本地（去“所有图片”里手动上传后发送）')
      showPreview.value = false
      mtPhotoStore.close()
      return
    }

    show(`导入失败: ${res?.error || res?.msg || '未知错误'}`)
  } catch (e: any) {
    console.error('导入失败:', e)
    show('导入失败')
  }
}
</script>
