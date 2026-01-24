<template>
  <teleport to="body">
    <div
      v-if="douyinStore.showModal"
      class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center"
      @click="close"
    >
      <div
        class="w-[95%] max-w-3xl h-[85vh] bg-[#18181b] rounded-2xl shadow-2xl flex flex-col"
        @click.stop
      >
        <!-- 头部 -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
          <div class="flex items-center gap-2 min-w-0">
            <i class="fas fa-cloud-download-alt text-emerald-400 flex-shrink-0"></i>
            <h3 class="text-lg font-bold text-white truncate">抖音下载</h3>
          </div>

          <button
            @click="close"
            class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-6 no-scrollbar">
            <div class="space-y-3">
            <div class="flex items-center gap-2">
              <button
                class="px-4 py-2 rounded-xl border transition text-sm"
                :class="activeMode === 'detail' ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                :disabled="uiDisabled"
                @click="switchMode('detail')"
              >
                作品解析
              </button>
              <button
                class="px-4 py-2 rounded-xl border transition text-sm"
                :class="activeMode === 'account' ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                :disabled="uiDisabled"
                @click="switchMode('account')"
              >
                用户作品
              </button>
              <button
                class="px-4 py-2 rounded-xl border transition text-sm"
                :class="activeMode === 'favorites' ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                :disabled="uiDisabled"
                @click="switchMode('favorites')"
              >
                收藏
              </button>
            </div>

            <div class="text-xs text-gray-500">
              <template v-if="activeMode === 'detail'">
                支持直接粘贴整段分享文本/短链/URL/作品ID；无需手动提取链接。
              </template>
              <template v-else-if="activeMode === 'account'">
                支持粘贴用户主页链接/分享文本/sec_uid，拉取该用户发布作品列表（默认全量拉取）。
              </template>
              <template v-else>
                查看已收藏的抖音用户/作品，并支持一键再次解析。
              </template>
            </div>

            <div v-if="activeMode === 'detail'" class="relative">
              <textarea
                v-model="inputText"
                class="w-full min-h-[110px] bg-[#111113] border border-gray-700 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500"
                placeholder="粘贴抖音分享文本/短链/完整URL/作品ID"
              ></textarea>
              <button
                v-if="inputText"
                class="absolute top-2 right-2 w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
                @click="handleClear"
                title="清空输入"
              >
                <i class="fas fa-times"></i>
              </button>
            </div>
            <div v-else-if="activeMode === 'account'" class="relative">
              <textarea
                v-model="accountInput"
                class="w-full min-h-[110px] bg-[#111113] border border-gray-700 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500"
                placeholder="粘贴抖音用户主页链接/分享文本/sec_uid"
              ></textarea>
              <button
                v-if="accountInput"
                class="absolute top-2 right-2 w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
                @click="handleClear"
                title="清空输入"
              >
                <i class="fas fa-times"></i>
              </button>
            </div>

            <div class="flex flex-col sm:flex-row gap-2">
              <button
                class="px-4 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                @click="showAdvanced = !showAdvanced"
              >
                {{ showAdvanced ? '隐藏 Cookie' : '填写 Cookie' }}
              </button>
              <button
                v-if="showAdvanced && cookie"
                class="px-4 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                @click="clearCookie"
                title="清除本地保存的 Cookie"
              >
                清除 Cookie
              </button>
            </div>

            <div v-if="showAdvanced" class="space-y-2">
              <textarea
                v-model="cookie"
                :class="[
                  'w-full min-h-[90px] bg-[#111113] border rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500',
                  highlightConfig ? 'border-red-500' : 'border-gray-700'
                ]"
                placeholder="抖音 Cookie（可选；仅本地保存，不会写入服务端）"
              ></textarea>
              <div class="text-xs text-red-400">
                Cookie 属敏感信息：仅保存在本地浏览器（localStorage），请勿在公共设备使用；如怀疑泄露请立即失效/更新。
              </div>
            </div>

            <div v-if="activeMode !== 'favorites'" class="flex gap-2">
              <button
                class="flex-1 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition disabled:opacity-60 disabled:cursor-not-allowed"
                :disabled="uiDisabled"
                @click="handlePrimaryAction"
              >
                <template v-if="activeMode === 'detail'">
                  {{ loading ? '解析中…' : '解析' }}
                </template>
                <template v-else>
                  {{ accountLoading ? '获取中…' : '获取作品' }}
                </template>
              </button>
              <button
                class="px-5 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                :disabled="uiDisabled"
                @click="pasteFromClipboard"
              >
                粘贴
              </button>
              <button
                class="px-5 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                :disabled="uiDisabled"
                @click="handleClear"
              >
                清空
              </button>
            </div>

            <div
              v-if="activeMode !== 'favorites'"
              class="flex flex-col sm:flex-row gap-2 sm:items-center sm:justify-between text-xs text-gray-500"
            >
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input type="checkbox" v-model="autoClipboard" class="accent-emerald-500" />
                <span>打开时自动读取剪贴板</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input type="checkbox" v-model="autoResolveClipboard" class="accent-emerald-500" />
                <span>{{ activeMode === 'detail' ? '读取后自动解析' : '读取后自动获取' }}</span>
              </label>
            </div>

            <div v-if="cookieHint" class="text-xs text-amber-300">
              {{ cookieHint }}
            </div>

            <div v-if="activeMode === 'detail' && error" class="text-sm text-red-400">
              {{ error }}
            </div>

            <div v-if="activeMode === 'account' && accountError" class="text-sm text-red-400">
              {{ accountError }}
            </div>

            <div v-if="activeMode === 'account'" class="pt-2">
              <div v-if="accountSecUserId" class="flex items-center justify-between gap-3 text-xs text-gray-500">
                <div class="min-w-0">
                  sec_user_id: <span class="font-mono text-gray-300">{{ accountSecUserId }}</span>
                </div>
                <button
                  class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs flex items-center gap-2 flex-shrink-0"
                  :disabled="uiDisabled"
                  @click="toggleFavoriteCurrentUser"
                  title="收藏/取消收藏该用户"
                >
                  <i :class="isFavoriteUser(accountSecUserId) ? 'fas fa-star text-yellow-400' : 'far fa-star text-gray-400'"></i>
                  <span>{{ isFavoriteUser(accountSecUserId) ? '已收藏' : '收藏用户' }}</span>
                </button>
              </div>

              <div v-if="accountItems.length > 0" class="mt-3 space-y-3">
                <div class="flex items-center justify-between gap-3">
                  <div class="text-xs text-gray-400">
                    已加载 {{ accountItems.length }} 个作品
                  </div>
                  <button
                    v-if="accountHasMore"
                    class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs"
                    :disabled="accountLoading"
                    @click="handleFetchMoreAccount"
                  >
                    {{ accountLoading ? '加载中…' : '加载更多' }}
                  </button>
                </div>

                <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
                  <button
                    v-for="item in accountItems"
                    :key="`douyin-account-${item.detailId}`"
                    class="rounded-xl overflow-hidden border border-gray-700 hover:border-emerald-500 transition bg-black/20 text-left relative group"
                    @click="openAccountItem(item)"
                    :disabled="accountItemLoading.has(item.detailId)"
                    :title="item.desc || item.detailId"
	                  >
                      <div v-if="accountItemLoading.has(item.detailId)" class="absolute inset-0 z-20 bg-black/60 flex items-center justify-center backdrop-blur-sm">
                        <i class="fas fa-spinner fa-spin text-white text-xl"></i>
                      </div>
                      <button
                        class="absolute top-2 right-2 z-30 w-10 h-10 rounded-xl bg-black/50 hover:bg-black/70 backdrop-blur border border-white/10 flex items-center justify-center transition-transform active:scale-95"
                        :disabled="uiDisabled"
                        @click.stop="toggleFavoriteAwemeFromAccount(item)"
                        :title="isFavoriteAweme(item.detailId) ? '取消收藏' : '收藏作品'"
                      >
                        <i :class="isFavoriteAweme(item.detailId) ? 'fas fa-star text-yellow-400' : 'far fa-star text-gray-300'"></i>
                      </button>
	                    <div class="aspect-video bg-[#111113] overflow-hidden">
		                      <MediaTile
		                        v-if="item.coverDownloadUrl || item.coverUrl"
		                        :src="(item.coverDownloadUrl || item.coverUrl)!"
		                        type="image"
		                        class="w-full h-full group-hover:scale-105 transition-transform duration-500"
		                        :show-skeleton="false"
		                        img-referrer-policy="no-referrer"
		                      />
	                      <div v-else class="w-full h-full flex items-center justify-center text-gray-600 text-xs">
	                        无封面
	                      </div>
	                    </div>
                    <div class="p-3 space-y-1">
                      <div class="text-sm text-white line-clamp-2">
                        {{ item.desc || '（无描述）' }}
                      </div>
                      <div class="text-xs text-gray-500 font-mono truncate">
                        {{ item.detailId }}
                      </div>
                    </div>
                  </button>
                </div>
              </div>

              <div v-else-if="accountLoading" class="mt-4 text-sm text-gray-500">
                获取中…
              </div>
              <div v-else-if="accountQueried" class="mt-4 text-sm text-gray-500">
                暂无作品
              </div>
              <div v-else class="mt-4 text-sm text-gray-500">
                粘贴用户主页链接后点击“获取作品”
              </div>
            </div>

            <div v-if="activeMode === 'favorites'" class="pt-2">
              <div class="sticky top-0 z-10 bg-[#18181b] py-2">
                <div class="flex items-center justify-between gap-2">
                  <div class="flex items-center gap-2">
                    <button
                      class="px-3 py-2 rounded-xl border transition text-xs"
                      :class="favoritesTab === 'users' ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                      :disabled="favoritesLoading"
                      @click="favoritesTab = 'users'"
                    >
                      用户收藏
                    </button>
                    <button
                      class="px-3 py-2 rounded-xl border transition text-xs"
                      :class="favoritesTab === 'awemes' ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                      :disabled="favoritesLoading"
                      @click="favoritesTab = 'awemes'"
                    >
                      作品收藏
                    </button>
                  </div>

                  <button
                    class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs"
                    :disabled="favoritesLoading"
                    @click="refreshFavorites"
                  >
                    {{ favoritesLoading ? '刷新中…' : '刷新' }}
                  </button>
                </div>
              </div>

              <div v-if="favoritesError" class="mt-2 text-sm text-red-400">
                {{ favoritesError }}
              </div>

              <div v-if="favoritesLoading" class="mt-4 text-sm text-gray-500">
                加载中…
              </div>

              <template v-else-if="favoritesTab === 'users'">
                <div v-if="favoriteUsers.length === 0" class="mt-4 text-sm text-gray-500">
                  暂无收藏用户
                </div>
                <div v-else class="mt-4 space-y-3">
                  <div
                    v-for="u in favoriteUsers"
                    :key="u.secUserId"
                    class="rounded-xl border border-gray-700 bg-black/20 p-3"
                  >
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0">
                        <div class="text-white text-sm font-medium truncate">
                          {{ u.displayName || u.secUserId }}
                        </div>
                        <div class="text-xs text-gray-500 font-mono truncate">
                          {{ u.secUserId }}
                        </div>
                        <div v-if="u.lastParsedAt || u.lastParsedCount" class="text-xs text-gray-500 mt-1">
                          <span v-if="u.lastParsedAt">上次解析: {{ u.lastParsedAt }}</span>
                          <span v-if="u.lastParsedCount"> · 作品: {{ u.lastParsedCount }}</span>
                        </div>
                      </div>

                      <div class="flex flex-col gap-2 flex-shrink-0 sm:flex-row sm:items-center">
                        <button
                          class="px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="uiDisabled"
                          @click="reparseFavoriteUser(u)"
                        >
                          再次解析
                        </button>
                        <button
                          class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="uiDisabled"
                          @click="removeFavoriteUser(u.secUserId)"
                        >
                          取消
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </template>

              <template v-else>
                <div v-if="favoriteAwemes.length === 0" class="mt-4 text-sm text-gray-500">
                  暂无收藏作品
                </div>
                <div v-else class="mt-4 space-y-3">
                  <div
                    v-for="it in favoriteAwemes"
                    :key="it.awemeId"
                    class="rounded-xl border border-gray-700 bg-black/20 p-3 flex gap-3"
                  >
                    <div class="w-24 h-14 bg-[#111113] rounded-lg overflow-hidden flex-shrink-0">
                      <MediaTile
                        v-if="it.coverUrl"
                        :src="it.coverUrl"
                        type="image"
                        class="w-full h-full"
                        :show-skeleton="false"
                        img-referrer-policy="no-referrer"
                      />
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="text-white text-sm font-medium line-clamp-2">
                        {{ it.desc || '（无描述）' }}
                      </div>
                      <div class="text-xs text-gray-500 font-mono truncate mt-1">
                        {{ it.awemeId }}
                      </div>
                      <div class="flex flex-col gap-2 mt-2 sm:flex-row sm:items-center">
                        <button
                          class="px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="uiDisabled"
                          @click="reparseFavoriteAweme(it)"
                        >
                          解析
                        </button>
                        <button
                          class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="uiDisabled"
                          @click="removeFavoriteAweme(it.awemeId)"
                        >
                          取消
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </div>

            <div v-if="activeMode === 'detail' && detail" class="pt-2">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-white font-medium text-sm truncate">
                    {{ detail.title }}
                  </div>
                  <div class="text-xs text-gray-500 mt-1">
                    类型: {{ detail.type }} · ID: {{ detail.detailId }}
                    <template v-if="detail.duration"> · 时长: {{ detail.duration }}</template>
                    <template v-if="detail.width && detail.height && detail.width > 0 && detail.height > 0">
                      · 分辨率: {{ detail.width }}×{{ detail.height }}
                    </template>
                  </div>
                </div>

                <div class="flex items-center gap-2 flex-shrink-0">
                  <button
                    class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs flex items-center gap-2"
                    :disabled="uiDisabled"
                    @click="toggleFavoriteCurrentDetail"
                    title="收藏/取消收藏该作品"
                  >
                    <i :class="isFavoriteAweme(detail.detailId) ? 'fas fa-star text-yellow-400' : 'far fa-star text-gray-400'"></i>
                    <span>{{ isFavoriteAweme(detail.detailId) ? '已收藏' : '收藏' }}</span>
                  </button>

                  <button
                    v-if="userStore.currentUser"
                    class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs flex items-center gap-2"
                    @click="openUploadMenu"
                    title="打开聊天页上传菜单"
                  >
                    <i class="fas fa-paper-plane"></i>
                    <span>上传菜单</span>
                  </button>
                </div>
              </div>

              <div v-if="detail.items && detail.items.length > 0" class="mt-4 space-y-2">
                <div class="flex flex-col sm:flex-row gap-2 sm:items-center sm:justify-between">
                  <div class="flex flex-wrap items-center gap-2 text-xs text-gray-400">
                    <button
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                      @click="toggleSelectionMode"
                      :disabled="batchImport.running || batchDownload.running"
                    >
                      {{ selectionMode ? '退出多选' : '多选模式' }}
                    </button>
                    <span v-if="selectionMode">已选 {{ selectedCount }} / {{ detail.items.length }}</span>
                    <button
                      v-if="selectionMode"
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                      @click="selectAll"
                      :disabled="batchImport.running || batchDownload.running"
                    >
                      全选
                    </button>
                    <button
                      v-if="selectionMode"
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                      @click="clearSelection"
                      :disabled="batchImport.running || batchDownload.running"
                    >
                      清空选择
                    </button>
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <button
                      class="px-4 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                      :disabled="batchDownload.running || batchImport.running"
                      @click="handleBatchDownload"
                      title="浏览器可能会提示允许多个文件下载"
                    >
                      {{ selectionMode ? '下载选中' : '下载全部' }}
                    </button>
                    <button
                      class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                      :disabled="batchImport.running || batchDownload.running || !userStore.currentUser"
                      @click="handleBatchImport"
                    >
                      {{ selectionMode ? '导入选中' : '导入全部' }}
                    </button>
                  </div>
                </div>

                <div v-if="batchDownload.running" class="space-y-1">
                  <div class="text-xs text-gray-400">
                    下载中：{{ batchDownload.done }} / {{ batchDownload.total }}
                  </div>
                  <div class="h-2 bg-[#111113] rounded-full overflow-hidden border border-gray-800">
                    <div class="h-full bg-indigo-500/70" :style="{ width: `${batchDownloadPercent}%` }"></div>
                  </div>
                </div>

                <div v-if="batchImport.running" class="space-y-1">
                  <div class="text-xs text-gray-400">
                    导入中：{{ batchImport.done }} / {{ batchImport.total }}
                  </div>
                  <div class="h-2 bg-[#111113] rounded-full overflow-hidden border border-gray-800">
                    <div class="h-full bg-emerald-500/70" :style="{ width: `${batchImportPercent}%` }"></div>
                  </div>
                </div>
              </div>

              <div v-if="detail.items && detail.items.length > 0" class="mt-3 grid grid-cols-3 sm:grid-cols-4 gap-3">
                <button
                  v-for="item in detail.items"
                  :key="`douyin-item-${item.index}`"
                  class="aspect-square rounded-xl overflow-hidden border border-gray-700 transition relative bg-black/20"
	                  :class="[
	                    selectionMode && isSelected(item.index) ? 'border-emerald-500 ring-2 ring-emerald-500/30' : 'hover:border-emerald-500',
	                    getItemState(detail.key, item.index)?.status === 'error' ? 'border-red-500' : ''
	                  ]"
	                  @click="handleItemClick(item.index)"
	                  :title="item.originalFilename || ''"
		                  >
	                  <MediaTile
	                    :src="item.downloadUrl"
	                    :type="item.type"
	                    class="w-full h-full"
	                    :show-skeleton="false"
	                    :muted="true"
	                    :indicator-size="'lg'"
	                  >
	                    <template #top-left>
	                      <MediaTileSelectMark
	                        v-if="selectionMode"
	                        :checked="isSelected(item.index)"
	                        tone="emerald"
	                        size="sm"
	                      />
	                    </template>

		                    <template #top-right>
		                      <div v-if="getItemState(detail.key, item.index)?.status && getItemState(detail.key, item.index)?.status !== 'idle'">
		                        <MediaTileBadge
		                          v-if="getItemState(detail.key, item.index)?.status === 'importing'"
		                          variant="success"
		                        >
		                          导入中…
		                        </MediaTileBadge>
		                        <MediaTileBadge
		                          v-else-if="getItemState(detail.key, item.index)?.status === 'imported'"
		                          variant="success"
		                        >
		                          已导入
		                        </MediaTileBadge>
		                        <MediaTileBadge
		                          v-else-if="getItemState(detail.key, item.index)?.status === 'exists'"
		                          variant="info"
		                        >
		                          已存在
		                        </MediaTileBadge>
		                        <MediaTileBadge
		                          v-else-if="getItemState(detail.key, item.index)?.status === 'error'"
		                          variant="danger"
		                          :title="getItemState(detail.key, item.index)?.message || ''"
		                        >
		                          失败
		                        </MediaTileBadge>
		                      </div>
		                    </template>

		                    <template #bottom-left>
		                      <MediaTileBadge
		                        v-if="getItemMeta(detail.key, item.index)?.size"
		                        variant="neutral"
		                      >
		                        {{ formatBytes(getItemMeta(detail.key, item.index)?.size || 0) }}
		                      </MediaTileBadge>
		                    </template>
		                  </MediaTile>
		                </button>
              </div>

              <div v-else class="mt-4 text-sm text-gray-500">
                暂无可下载资源
              </div>
            </div>
          </div>
        </div>
      </div>

      <MediaPreview
        v-model:visible="showPreview"
        :url="previewUrl"
        :type="previewType"
        :can-upload="canUpload"
        :upload-disabled="previewUploadDisabled"
        :upload-loading="previewUploadLoading"
        :upload-text="previewUploadText"
        :media-list="previewMediaList"
        @upload="confirmImportUpload"
        @media-change="handlePreviewMediaChange"
      />
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useDouyinStore } from '@/stores/douyin'
import { useUserStore } from '@/stores/user'
	import { useMediaStore } from '@/stores/media'
	import { useSystemConfigStore } from '@/stores/systemConfig'
	import { useToast } from '@/composables/useToast'
	import MediaTile from '@/components/common/MediaTile.vue'
	import MediaTileBadge from '@/components/common/MediaTileBadge.vue'
	import MediaTileSelectMark from '@/components/common/MediaTileSelectMark.vue'
	import { generateCookie } from '@/utils/cookie'
	import * as douyinApi from '@/api/douyin'
	import MediaPreview from '@/components/media/MediaPreview.vue'
	import type { UploadedMedia } from '@/types'

interface DouyinDetailItem {
  index: number
  type: 'image' | 'video'
  url: string
  downloadUrl: string
  originalFilename?: string
}

interface DouyinDetailResponse {
  key: string
  detailId: string
  type: string
  title: string
  coverUrl?: string
  duration?: string
  width?: number
  height?: number
  items: DouyinDetailItem[]
}

interface DouyinAccountItem {
  detailId: string
  type?: 'image' | 'video'
  desc?: string
  coverUrl?: string
  coverDownloadUrl?: string
  key?: string
  items?: DouyinDetailItem[]
}

interface DouyinAccountResponse {
  secUserId: string
  tab: string
  cursor: number
  hasMore: boolean
  items: DouyinAccountItem[]
}

interface DouyinFavoriteUser {
  secUserId: string
  sourceInput?: string
  displayName?: string
  avatarUrl?: string
  profileUrl?: string
  lastParsedAt?: string
  lastParsedCount?: number
  createTime: string
  updateTime: string
}

interface DouyinFavoriteAweme {
  awemeId: string
  secUserId?: string
  type?: 'image' | 'video'
  desc?: string
  coverUrl?: string
  createTime: string
  updateTime: string
}

const douyinStore = useDouyinStore()
const userStore = useUserStore()
const mediaStore = useMediaStore()
const systemConfigStore = useSystemConfigStore()
const { show } = useToast()

const activeMode = ref<'detail' | 'account' | 'favorites'>('detail')

const inputText = ref('')
const accountInput = ref('')
const cookie = ref('')
const showAdvanced = ref(false)
const cookieHint = ref('')
const highlightConfig = ref(false)

const autoClipboard = ref(true)
const autoResolveClipboard = ref(false)

const selectionMode = ref(false)
const selectedIndices = ref<Set<number>>(new Set())

const loading = ref(false)
const error = ref('')
const detail = ref<DouyinDetailResponse | null>(null)

const accountLoading = ref(false)
const accountError = ref('')
const accountItems = ref<DouyinAccountItem[]>([])
const accountCursor = ref(0)
const accountHasMore = ref(false)
const accountSecUserId = ref('')
const accountQueried = ref(false)
const accountItemLoading = reactive<Set<string>>(new Set())

const favoritesTab = ref<'users' | 'awemes'>('users')
const favoritesLoading = ref(false)
const favoritesError = ref('')
const favoriteUsers = ref<DouyinFavoriteUser[]>([])
const favoriteAwemes = ref<DouyinFavoriteAweme[]>([])

const favoriteUserIdSet = computed(() => new Set(favoriteUsers.value.map((u) => String(u.secUserId || '').trim()).filter(Boolean)))
const favoriteAwemeIdSet = computed(() => new Set(favoriteAwemes.value.map((it) => String(it.awemeId || '').trim()).filter(Boolean)))

const isFavoriteUser = (secUserId: string) => favoriteUserIdSet.value.has(String(secUserId || '').trim())
const isFavoriteAweme = (awemeId: string) => favoriteAwemeIdSet.value.has(String(awemeId || '').trim())

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewMediaList = ref<UploadedMedia[]>([])
const previewIndex = ref(0)
const previewContextKey = ref('')
const previewContextItems = ref<DouyinDetailItem[]>([])

const canUpload = computed(() => !!userStore.currentUser)

	type ItemStatus = 'idle' | 'importing' | 'imported' | 'exists' | 'error'
	type ItemState = { status: ItemStatus; message?: string }

	const buildItemId = (key: string, idx: number) => `${String(key || '').trim()}:${Number(idx)}`
	const itemStateById = reactive<Record<string, ItemState>>({})
	const itemMetaById = reactive<Record<string, { size?: number; mime?: string }>>({})
	const getItemState = (key: string, idx: number) => itemStateById[buildItemId(key, idx)]
	const getItemMeta = (key: string, idx: number) => itemMetaById[buildItemId(key, idx)]

const batchImport = reactive({ running: false, total: 0, done: 0, success: 0, fail: 0 })
const batchDownload = reactive({ running: false, total: 0, done: 0, success: 0, fail: 0 })

const batchImportPercent = computed(() => (batchImport.total > 0 ? Math.round((batchImport.done / batchImport.total) * 100) : 0))
const batchDownloadPercent = computed(() => (batchDownload.total > 0 ? Math.round((batchDownload.done / batchDownload.total) * 100) : 0))

const restoreLocalConfig = () => {
  cookie.value = localStorage.getItem('douyin_cookie') || ''

  autoClipboard.value = localStorage.getItem('douyin_auto_clipboard') !== '0'
  autoResolveClipboard.value = localStorage.getItem('douyin_auto_resolve_clipboard') === '1'
}

const persistLocalConfig = () => {
  localStorage.setItem('douyin_cookie', cookie.value || '')
  localStorage.removeItem('douyin_proxy')
  localStorage.setItem('douyin_auto_clipboard', autoClipboard.value ? '1' : '0')
  localStorage.setItem('douyin_auto_resolve_clipboard', autoResolveClipboard.value ? '1' : '0')
}

const isLikelyDouyinText = (value: string) => {
  const v = String(value || '').trim()
  if (!v) return false
  if (v.includes('v.douyin.com') || v.includes('www.douyin.com') || v.includes('douyin.com')) return true
  if (v.includes('/user/') || v.includes('sec_uid') || v.includes('sec_user_id')) return true
  if (v.includes('modal_id=') || v.includes('aweme_id=')) return true
  if (/^\\d{6,}$/.test(v)) return true
  if (!v.includes(' ') && v.startsWith('MS4wLjAB')) return true
  return false
}

const isLikelyDouyinDetailInput = (value: string) => {
  const v = String(value || '').trim()
  if (!v) return false
  if (v.includes('/video/') || v.includes('/note/')) return true
  if (v.includes('modal_id=') || v.includes('aweme_id=')) return true
  if (/^\\d{6,}$/.test(v)) return true
  return false
}

const isLikelyDouyinUserInput = (value: string) => {
  const v = String(value || '').trim()
  if (!v) return false
  if (v.includes('/user/') || v.includes('sec_uid') || v.includes('sec_user_id')) return true
  if (!v.includes(' ') && v.startsWith('MS4wLjAB')) return true
  return false
}

const applyInputText = (txt: string) => {
  const v = String(txt || '').trim()
  if (!v) return

  const looksDetail = isLikelyDouyinDetailInput(v)
  const looksUser = isLikelyDouyinUserInput(v)

  if (looksUser && !looksDetail) {
    activeMode.value = 'account'
    accountInput.value = v
    return
  }
  if (looksDetail && !looksUser) {
    activeMode.value = 'detail'
    inputText.value = v
    return
  }

  if (activeMode.value === 'account') {
    accountInput.value = v
  } else {
    inputText.value = v
  }
}

const clearCookie = () => {
  cookie.value = ''
  localStorage.removeItem('douyin_cookie')
  show('已清除本地 Cookie')
}

const pasteFromClipboard = async () => {
  try {
    const txt = await navigator.clipboard?.readText?.()
    if (!String(txt || '').trim()) {
      show('剪贴板为空')
      return
    }
    applyInputText(txt)
    show('已从剪贴板填充')
    if (autoResolveClipboard.value) {
      await handlePrimaryAction()
    }
  } catch (e) {
    console.warn('read clipboard failed:', e)
    show('无法读取剪贴板（可能需要浏览器授权或 https）')
  }
}

const resetDetailStates = () => {
  Object.keys(itemStateById).forEach((k) => delete itemStateById[k])
  Object.keys(itemMetaById).forEach((k) => delete itemMetaById[k])
  previewContextKey.value = ''
  previewContextItems.value = []
  selectionMode.value = false
  selectedIndices.value = new Set()
  batchImport.running = false
  batchImport.total = 0
  batchImport.done = 0
  batchImport.success = 0
  batchImport.fail = 0
  batchDownload.running = false
  batchDownload.total = 0
  batchDownload.done = 0
  batchDownload.success = 0
  batchDownload.fail = 0
}

const resetAccountStates = () => {
  accountLoading.value = false
  accountError.value = ''
  accountItems.value = []
  accountCursor.value = 0
  accountHasMore.value = false
  accountSecUserId.value = ''
  accountQueried.value = false
}

const refreshFavorites = async () => {
  favoritesLoading.value = true
  favoritesError.value = ''
  try {
    const [usersRes, awemesRes] = await Promise.all([douyinApi.listDouyinFavoriteUsers(), douyinApi.listDouyinFavoriteAwemes()])

    favoriteUsers.value = Array.isArray(usersRes?.items) ? (usersRes.items as DouyinFavoriteUser[]) : []
    favoriteAwemes.value = Array.isArray(awemesRes?.items) ? (awemesRes.items as DouyinFavoriteAweme[]) : []
  } catch (e: any) {
    console.error('加载抖音收藏失败:', e)
    favoritesError.value = e?.response?.data?.error || e?.message || '加载失败'
  } finally {
    favoritesLoading.value = false
  }
}

watch(
  () => douyinStore.showModal,
  async (v) => {
    if (v) {
      restoreLocalConfig()
      error.value = ''
      detail.value = null
      accountError.value = ''
      resetAccountStates()
      cookieHint.value = ''
      highlightConfig.value = false
      resetDetailStates()
      showPreview.value = false
      previewUrl.value = ''
      previewMediaList.value = []
      previewIndex.value = 0
      favoritesTab.value = 'users'
      void refreshFavorites()

      // 优先使用调用方传入的预填内容；否则按设置尝试读取剪贴板
      const hasInput = () => !!String(inputText.value || '').trim() || !!String(accountInput.value || '').trim()
      if (!hasInput() && douyinStore.draftInput) {
        applyInputText(douyinStore.draftInput)
      } else if (!hasInput() && autoClipboard.value) {
        try {
          const txt = await navigator.clipboard?.readText?.()
          if (String(txt || '').trim()) {
            applyInputText(txt)
            show('已从剪贴板读取')
          }
        } catch {
          // ignore
        }
      }

      if (autoResolveClipboard.value && hasInput()) {
        await handlePrimaryAction()
      }
    }
  }
)

const close = () => {
  persistLocalConfig()
  douyinStore.close()
  showPreview.value = false
  previewUrl.value = ''
  previewMediaList.value = []
  previewIndex.value = 0
  activeMode.value = 'detail'
  error.value = ''
  accountError.value = ''
  cookieHint.value = ''
  highlightConfig.value = false
  detail.value = null
  resetDetailStates()
  resetAccountStates()
}

const handleClear = () => {
  cookieHint.value = ''
  highlightConfig.value = false

  if (activeMode.value === 'detail') {
    inputText.value = ''
    error.value = ''
    detail.value = null
    resetDetailStates()
    return
  }

  if (activeMode.value === 'favorites') {
    favoritesError.value = ''
    return
  }

  accountInput.value = ''
  accountError.value = ''
  resetAccountStates()
}

const uiDisabled = computed(() => loading.value || accountLoading.value || batchImport.running || batchDownload.running)

const switchMode = (mode: 'detail' | 'account' | 'favorites') => {
  if (uiDisabled.value) return
  activeMode.value = mode
  cookieHint.value = ''
  highlightConfig.value = false

  // 从“作品解析”切到“用户作品”时，避免预览悬浮造成干扰
  if (mode !== 'detail') {
    showPreview.value = false
    previewUrl.value = ''
    previewMediaList.value = []
    previewIndex.value = 0
  }

  if (mode === 'favorites') {
    if (favoriteUsers.value.length === 0 && favoriteAwemes.value.length === 0) {
      void refreshFavorites()
    }
  }
}

const handleFetchAccount = async (opts: { append?: boolean } = {}) => {
  const input = String(accountInput.value || '').trim()
  if (!input) {
    show('请输入抖音用户主页链接/分享文本/sec_uid')
    return
  }
  if (isLikelyDouyinDetailInput(input) && !isLikelyDouyinUserInput(input)) {
    show('识别到作品链接，请切换到“作品解析”')
    activeMode.value = 'detail'
    inputText.value = input
    return
  }
  if (opts.append && !accountHasMore.value) return

  const append = !!opts.append

  accountQueried.value = true
  accountLoading.value = true
  accountError.value = ''
  cookieHint.value = ''
  highlightConfig.value = false
  persistLocalConfig()

  if (!append) {
    accountItems.value = []
    accountCursor.value = 0
    accountHasMore.value = false
    accountSecUserId.value = ''
  }

  try {
    const countPerPage = 18
    const existing = new Set(accountItems.value.map((i) => String(i.detailId || '').trim()).filter(Boolean))

    const fetchPage = async (cursor: number) => {
      const res = await douyinApi.getDouyinAccount({
        input,
        cookie: String(cookie.value || '').trim(),
        tab: 'post',
        cursor,
        count: countPerPage
      })
      if (!res || !Array.isArray(res?.items)) {
        throw new Error(res?.error || '获取失败')
      }
      return res as DouyinAccountResponse
    }

    const appendItems = (incoming: DouyinAccountItem[]) => {
      for (const it of incoming || []) {
        const id = String(it?.detailId || '').trim()
        if (!id || existing.has(id)) continue
        existing.add(id)
        accountItems.value.push({
          detailId: id,
          type: it?.type,
          desc: String(it?.desc || '').trim(),
          coverUrl: String(it?.coverUrl || '').trim(),
          coverDownloadUrl: String(it?.coverDownloadUrl || '').trim(),
          key: String(it?.key || '').trim(),
          items: Array.isArray(it?.items) ? it.items : []
        })
      }
    }

    if (append) {
      const res = await fetchPage(accountCursor.value)
      accountSecUserId.value = String(res?.secUserId || accountSecUserId.value || '').trim()
      accountCursor.value = Number(res?.cursor || 0)
      accountHasMore.value = !!res?.hasMore
      appendItems(res.items || [])
      return
    }

    let cursor = 0
    let loops = 0
    while (true) {
      loops += 1
      const res = await fetchPage(cursor)
      accountSecUserId.value = String(res?.secUserId || accountSecUserId.value || '').trim()
      appendItems(res.items || [])

      const nextCursor = Number(res?.cursor || 0)
      const hasMore = !!res?.hasMore
      accountCursor.value = nextCursor
      accountHasMore.value = hasMore

      if (!hasMore) break
      if (nextCursor <= 0 || nextCursor === cursor) break
      cursor = nextCursor

      // best-effort：避免上游异常导致死循环（正常情况下 hasMore 会在有限次数内归零）
      if (loops >= 200) {
        show('已达到最大分页限制，可能还有未加载的作品')
        break
      }
    }

    // 若该用户已被收藏，则同步更新“最后解析时间/作品数”
    const secUserId = String(accountSecUserId.value || '').trim()
    if (secUserId && isFavoriteUser(secUserId)) {
      try {
        const updated = await douyinApi.addDouyinFavoriteUser({
          secUserId,
          sourceInput: input,
          lastParsedCount: accountItems.value.length
        })
        if (updated?.secUserId) {
          const next = [updated as DouyinFavoriteUser, ...favoriteUsers.value.filter((u) => u.secUserId !== updated.secUserId)]
          favoriteUsers.value = next
        }
      } catch (e) {
        console.warn('同步抖音用户收藏解析信息失败:', e)
      }
    }
  } catch (e: any) {
    console.error('获取抖音账号作品失败:', e)
    const msg = e?.response?.data?.error || e?.message || '获取失败'
    accountError.value = msg

    if (String(msg).includes('获取数据失败') || String(msg).toLowerCase().includes('cookie') || String(msg).includes('风控')) {
      showAdvanced.value = true
      highlightConfig.value = true
      cookieHint.value = '提示：可能需要更新 Cookie 后重试。'
    }
  } finally {
    accountLoading.value = false
  }
}

const handleFetchMoreAccount = async () => {
  if (accountLoading.value) return
  if (!accountHasMore.value) return
  await handleFetchAccount({ append: true })
}

const upsertFavoriteUser = async (payload: {
  secUserId: string
  sourceInput?: string
  displayName?: string
  avatarUrl?: string
  profileUrl?: string
  lastParsedCount?: number
  lastParsedRaw?: any
}) => {
  const secUserId = String(payload?.secUserId || '').trim()
  if (!secUserId) return

  try {
    const res = await douyinApi.addDouyinFavoriteUser(payload as any)
    if (res?.secUserId) {
      favoriteUsers.value = [res as DouyinFavoriteUser, ...favoriteUsers.value.filter((u) => u.secUserId !== res.secUserId)]
      show('已收藏用户')
    } else {
      throw new Error(res?.error || '收藏失败')
    }
  } catch (e: any) {
    console.error('收藏抖音用户失败:', e)
    show(e?.response?.data?.error || e?.message || '收藏失败')
  }
}

const removeFavoriteUser = async (secUserId: string) => {
  const id = String(secUserId || '').trim()
  if (!id) return

  try {
    await douyinApi.removeDouyinFavoriteUser({ secUserId: id })
    favoriteUsers.value = favoriteUsers.value.filter((u) => u.secUserId !== id)
    show('已取消收藏')
  } catch (e: any) {
    console.error('取消抖音用户收藏失败:', e)
    show(e?.response?.data?.error || e?.message || '取消失败')
  }
}

const toggleFavoriteCurrentUser = async () => {
  const secUserId = String(accountSecUserId.value || '').trim()
  if (!secUserId) {
    show('请先获取用户作品')
    return
  }
  if (isFavoriteUser(secUserId)) {
    await removeFavoriteUser(secUserId)
    return
  }

  await upsertFavoriteUser({
    secUserId,
    sourceInput: String(accountInput.value || '').trim(),
    lastParsedCount: accountItems.value.length
  })
}

const upsertFavoriteAweme = async (payload: {
  awemeId: string
  secUserId?: string
  type?: string
  desc?: string
  coverUrl?: string
  rawDetail?: any
}) => {
  const awemeId = String(payload?.awemeId || '').trim()
  if (!awemeId) return

  try {
    const res = await douyinApi.addDouyinFavoriteAweme(payload as any)
    if (res?.awemeId) {
      favoriteAwemes.value = [res as DouyinFavoriteAweme, ...favoriteAwemes.value.filter((it) => it.awemeId !== res.awemeId)]
      show('已收藏作品')
    } else {
      throw new Error(res?.error || '收藏失败')
    }
  } catch (e: any) {
    console.error('收藏抖音作品失败:', e)
    show(e?.response?.data?.error || e?.message || '收藏失败')
  }
}

const removeFavoriteAweme = async (awemeId: string) => {
  const id = String(awemeId || '').trim()
  if (!id) return

  try {
    await douyinApi.removeDouyinFavoriteAweme({ awemeId: id })
    favoriteAwemes.value = favoriteAwemes.value.filter((it) => it.awemeId !== id)
    show('已取消收藏')
  } catch (e: any) {
    console.error('取消抖音作品收藏失败:', e)
    show(e?.response?.data?.error || e?.message || '取消失败')
  }
}

const toggleFavoriteAwemeFromAccount = async (item: DouyinAccountItem) => {
  const awemeId = String(item?.detailId || '').trim()
  if (!awemeId) return

  if (isFavoriteAweme(awemeId)) {
    await removeFavoriteAweme(awemeId)
    return
  }

  await upsertFavoriteAweme({
    awemeId,
    secUserId: String(accountSecUserId.value || '').trim(),
    type: String(item?.type || '').trim(),
    desc: String(item?.desc || '').trim(),
    coverUrl: String(item?.coverUrl || '').trim()
  })
}

const toggleFavoriteCurrentDetail = async () => {
  const d = detail.value
  if (!d) return

  const awemeId = String(d.detailId || '').trim()
  if (!awemeId) return

  if (isFavoriteAweme(awemeId)) {
    await removeFavoriteAweme(awemeId)
    return
  }

  const typeValue = String(d.items?.[0]?.type || '').trim() || (d.type?.includes('图集') ? 'image' : 'video')
  await upsertFavoriteAweme({
    awemeId,
    type: typeValue,
    desc: String(d.title || '').trim(),
    coverUrl: String(d.coverUrl || '').trim(),
    rawDetail: d
  })
}

const reparseFavoriteUser = async (u: DouyinFavoriteUser) => {
  const secUserId = String(u?.secUserId || '').trim()
  if (!secUserId) return

  activeMode.value = 'account'
  accountInput.value = secUserId
  await handleFetchAccount()
}

const reparseFavoriteAweme = async (it: DouyinFavoriteAweme) => {
  const awemeId = String(it?.awemeId || '').trim()
  if (!awemeId) return

  activeMode.value = 'detail'
  inputText.value = awemeId
  await handleResolve()
}

const getAccountItemTitle = (item: DouyinAccountItem) => {
  const t = String(item?.desc || '').trim()
  if (t) return t
  return String(item?.detailId || '').trim()
}

const buildAccountPreviewMediaList = (items: DouyinAccountItem[]): UploadedMedia[] => {
  const list: UploadedMedia[] = []
  for (const it of items || []) {
    const key = String(it?.key || '').trim()
    const medias = Array.isArray(it?.items) ? it.items : []
    if (!key || medias.length === 0) continue
    list.push(...buildPreviewMediaList(medias, { key, title: getAccountItemTitle(it) }))
  }
  return list
}

const openPreviewFromAccount = (item: DouyinAccountItem) => {
  resetDetailStates()
  previewContextKey.value = String(item.key || '')
  previewContextItems.value = item.items || []

  const first = (item.items || []).slice().sort((a, b) => Number(a.index) - Number(b.index))[0]
  if (!first) return

  previewIndex.value = Number(first.index) || 0
  previewType.value = first.type
  previewUrl.value = String(first.downloadUrl || first.url || '').trim()
  const fullList = buildAccountPreviewMediaList(accountItems.value)
  previewMediaList.value = fullList.length > 0 ? fullList : buildPreviewMediaList(item.items || [], { key: String(item.key || '').trim(), title: getAccountItemTitle(item) })
  showPreview.value = true
}

const openAccountItem = async (item: DouyinAccountItem) => {
  const id = String(item?.detailId || '').trim()
  if (!id) return

  const key = String(item?.key || '').trim()
  const items = Array.isArray(item?.items) ? item.items : []
  if (key && items.length > 0) {
    openPreviewFromAccount(item)
    return
  }

  if (accountItemLoading.has(id)) return
  accountItemLoading.add(id)

  try {
    const res = await douyinApi.getDouyinDetail({
      input: id,
      cookie: String(cookie.value || '').trim()
    })

    if (res?.key && Array.isArray(res?.items)) {
      item.key = res.key
      item.items = res.items

      // Update in accountItems list if ref mismatch
      const found = accountItems.value.find((i) => i.detailId === id)
      if (found) {
        found.key = res.key
        found.items = res.items
      }

      openPreviewFromAccount(item)
    } else {
      throw new Error(res?.error || '解析未返回有效数据')
    }
  } catch (e: any) {
    console.error('获取作品详情失败:', e)
    show(e?.message || e?.response?.data?.error || '获取详情失败')
  } finally {
    accountItemLoading.delete(id)
  }
}

const handlePrimaryAction = async () => {
  if (activeMode.value === 'detail') {
    await handleResolve()
    return
  }
  if (activeMode.value === 'favorites') {
    await refreshFavorites()
    return
  }
  await handleFetchAccount()
}

const openUploadMenu = () => {
  mediaStore.requestOpenUploadMenu()
  show('已打开上传菜单（可在“已上传的文件”中发送）')
}

const toggleSelectionMode = () => {
  selectionMode.value = !selectionMode.value
  selectedIndices.value = new Set()
}

const isSelected = (idx: number) => selectedIndices.value.has(Number(idx))
const selectedCount = computed(() => selectedIndices.value.size)

const selectAll = () => {
  if (!detail.value?.items) return
  selectedIndices.value = new Set(detail.value.items.map((i) => Number(i.index)))
}

const clearSelection = () => {
  selectedIndices.value = new Set()
}

const toggleSelect = (idx: number) => {
  const next = new Set(selectedIndices.value)
  const v = Number(idx)
  if (next.has(v)) next.delete(v)
  else next.add(v)
  selectedIndices.value = next
}

const handleItemClick = (idx: number) => {
  if (selectionMode.value) {
    toggleSelect(idx)
    return
  }
  openPreview(idx)
}

const formatBytes = (bytes: number) => {
  const n = Number(bytes || 0)
  if (!Number.isFinite(n) || n <= 0) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let v = n
  let u = 0
  while (v >= 1024 && u < units.length - 1) {
    v /= 1024
    u += 1
  }
  return `${v.toFixed(u === 0 ? 0 : 1)}${units[u]}`
}

const getAuthToken = () => String(localStorage.getItem('authToken') || '').trim()

const prefetchMetas = async (key: string, items: DouyinDetailItem[]) => {
  const token = getAuthToken()
  if (!token) return

  const queue = items.map((i) => i.index)
  const concurrency = 4

  const worker = async () => {
    while (queue.length > 0) {
      const idx = Number(queue.shift())
      const item = items.find((i) => Number(i.index) === idx)
      if (!item) continue
      try {
        const resp = await fetch(item.downloadUrl, {
          method: 'HEAD',
          headers: { Authorization: `Bearer ${token}` }
        })
        if (!resp.ok) continue
        const len = Number(resp.headers.get('Content-Length') || 0)
        const mime = String(resp.headers.get('Content-Type') || '').trim()
        if (len > 0) {
          const id = buildItemId(key, idx)
          itemMetaById[id] = { ...(itemMetaById[id] || {}), size: len, mime }
        }
      } catch {
        // ignore
      }
    }
  }

  await Promise.all(Array.from({ length: concurrency }, () => worker()))
}

const handleResolve = async () => {
  const input = String(inputText.value || '').trim()
  if (!input) {
    show('请输入抖音分享文本/链接/作品ID')
    return
  }
  if (isLikelyDouyinUserInput(input) && !isLikelyDouyinDetailInput(input)) {
    show('识别到用户主页链接，请切换到“用户作品”')
    activeMode.value = 'account'
    accountInput.value = input
    return
  }

  loading.value = true
  error.value = ''
  detail.value = null
  cookieHint.value = ''
  highlightConfig.value = false
  resetDetailStates()
  persistLocalConfig()

  try {
    const res = await douyinApi.getDouyinDetail({
      input,
      cookie: String(cookie.value || '').trim()
    })

    if (!res?.key || !Array.isArray(res?.items)) {
      error.value = res?.error || '解析失败'
      return
    }

    detail.value = res as DouyinDetailResponse

    // 若该作品已被收藏，则同步更新“最后解析数据”（不弹 Toast）
    const awemeId = String(detail.value.detailId || '').trim()
    if (awemeId && isFavoriteAweme(awemeId)) {
      try {
        const typeValue = String(detail.value.items?.[0]?.type || '').trim() || (detail.value.type?.includes('图集') ? 'image' : 'video')
        const updated = await douyinApi.addDouyinFavoriteAweme({
          awemeId,
          type: typeValue,
          desc: String(detail.value.title || '').trim(),
          coverUrl: String(detail.value.coverUrl || '').trim(),
          rawDetail: detail.value
        })
        if (updated?.awemeId) {
          favoriteAwemes.value = [updated as DouyinFavoriteAweme, ...favoriteAwemes.value.filter((it) => it.awemeId !== updated.awemeId)]
        }
      } catch (e) {
        console.warn('同步抖音作品收藏解析信息失败:', e)
      }
    }

    // 预取文件大小（最佳努力）
    void prefetchMetas(detail.value.key, detail.value.items || [])
	  } catch (e: any) {
	    console.error('解析抖音失败:', e)
	    const msg = e?.response?.data?.error || e?.message || '解析失败'
	    error.value = msg

    // 经验判断：Cookie 问题更常见，解析失败时引导用户填写
    if (String(msg).includes('获取数据失败') || String(msg).toLowerCase().includes('cookie') || String(msg).includes('风控')) {
      showAdvanced.value = true
      highlightConfig.value = true
      cookieHint.value = '提示：可能需要更新 Cookie 后重试。'
    }
  } finally {
    loading.value = false
  }
}

const buildPreviewMediaList = (items: DouyinDetailItem[], opts: { key?: string; title?: string } = {}): UploadedMedia[] => {
  const key = String(opts.key || '').trim()
  const title = String(opts.title || '').trim()
  return items
    .slice()
    .sort((a, b) => Number(a.index) - Number(b.index))
    .map((it) => ({
      url: it.downloadUrl,
      type: it.type,
      downloadUrl: it.downloadUrl,
      originalFilename: it.originalFilename,
      title: title || undefined,
      context: key ? { provider: 'douyin', key, index: Number(it.index) } : undefined
    }))
}

const openPreview = (idx: number) => {
  if (!detail.value) return
  const item = detail.value.items.find((i) => Number(i.index) === Number(idx))
  if (!item) return

  previewIndex.value = Number(item.index) || 0
  previewType.value = item.type
  previewUrl.value = item.downloadUrl

	  previewContextKey.value = detail.value.key
	  previewContextItems.value = detail.value.items || []

	  if (item.type === 'image') {
	    const images = detail.value.items.filter((i) => i.type === 'image')
	    previewMediaList.value = buildPreviewMediaList(images, { key: detail.value.key, title: detail.value.title })
	  } else {
	    previewMediaList.value = buildPreviewMediaList([item], { key: detail.value.key, title: detail.value.title })
	  }

  showPreview.value = true
}

const handlePreviewMediaChange = (media: UploadedMedia) => {
  const url = String(media?.url || '').trim()
  if (!url) return

  const ctx = media?.context
  if (ctx?.provider === 'douyin') {
    const key = String(ctx.key || '').trim()
    const idx = Number(ctx.index)
    if (key) {
      previewContextKey.value = key
      if (detail.value?.key && String(detail.value.key) === key) {
        previewContextItems.value = detail.value.items || []
      } else {
        const found = accountItems.value.find((i) => String(i.key || '').trim() === key)
        previewContextItems.value = (found?.items || []) as DouyinDetailItem[]
      }
    }
    if (Number.isFinite(idx)) {
      previewIndex.value = idx
    }
  } else {
    const item = (previewContextItems.value || []).find((i) => i.downloadUrl === url || i.url === url)
    if (item) {
      previewIndex.value = Number(item.index) || previewIndex.value
    }
  }
  previewUrl.value = url
  previewType.value = media.type || previewType.value
}

const previewItemState = computed(() => {
  const key = String(previewContextKey.value || detail.value?.key || '').trim()
  if (!key) return undefined
  const idx = Number(previewIndex.value) || 0
  return itemStateById[buildItemId(key, idx)]
})

const previewUploadLoading = computed(() => previewItemState.value?.status === 'importing')
const previewUploadDisabled = computed(() => {
  if (!userStore.currentUser) return true
  const key = String(previewContextKey.value || detail.value?.key || '').trim()
  if (!key) return true
  const st = previewItemState.value?.status
  return st === 'importing' || st === 'imported' || st === 'exists'
})
const previewUploadText = computed(() => {
  if (!userStore.currentUser) return ''
  const st = previewItemState.value?.status
  if (st === 'importing') return '导入中…'
  if (st === 'imported') return '已导入'
  if (st === 'exists') return '已存在（去重）'
  return `导入此${previewType.value === 'image' ? '图片' : (previewType.value === 'video' ? '视频' : '文件')}`
})

const ensureImgServer = async () => {
  if (!mediaStore.imgServer) {
    await mediaStore.loadImgServer()
  }
  return !!mediaStore.imgServer
}

const importIndex = async (idx: number) => {
  if (!userStore.currentUser) {
    show('请先选择身份后再导入上传')
    return { ok: false, dedup: false, error: '未选择身份' }
  }
  const key = String(previewContextKey.value || detail.value?.key || '').trim()
  if (!key) return { ok: false, dedup: false, error: '解析信息缺失' }
  if (!await ensureImgServer()) return { ok: false, dedup: false, error: '图片服务器地址未获取' }

  const itemId = buildItemId(key, idx)
  const current = itemStateById[itemId]?.status
  if (current === 'imported' || current === 'exists') {
    return { ok: true, dedup: current === 'exists' }
  }
  if (current === 'importing') {
    return { ok: false, dedup: false, error: '导入进行中' }
  }

  itemStateById[itemId] = { status: 'importing' }

  const cookieData = generateCookie(userStore.currentUser.id, userStore.currentUser.name)
  const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
  const userAgent = navigator.userAgent

  try {
    const res = await douyinApi.importDouyinMedia({
      userid: userStore.currentUser.id,
      key,
      index: idx,
      cookieData,
      referer,
      userAgent
    })

    if (res?.state === 'OK' && res.msg) {
      const port = String(res.port || await systemConfigStore.resolveImagePort(res.msg, mediaStore.imgServer))
      const remoteUrl = `http://${mediaStore.imgServer}:${port}/img/Upload/${res.msg}`
      const inferredType =
        (previewContextItems.value || []).find((i) => Number(i.index) === Number(idx))?.type ||
        detail.value?.items?.find((i) => Number(i.index) === Number(idx))?.type ||
        previewType.value

      if (!mediaStore.uploadedMedia.some((m) => m.url === remoteUrl)) {
        mediaStore.addUploadedMedia({
          url: remoteUrl,
          type: inferredType,
          localFilename: res.localFilename
        })
      }

      const dedup = !!res.dedup
      itemStateById[itemId] = { status: dedup ? 'exists' : 'imported' }
      show(dedup ? '已存在（去重复用）' : '已导入上传（可在上传菜单发送）')
      return { ok: true, dedup }
    }

    itemStateById[itemId] = { status: 'error', message: String(res?.error || res?.msg || '导入失败') }
    return { ok: false, dedup: false, error: String(res?.error || res?.msg || '导入失败') }
  } catch (e: any) {
    console.error('导入失败:', e)
    const msg = e?.response?.data?.error || e?.message || '导入失败'
    itemStateById[itemId] = { status: 'error', message: String(msg) }
    return { ok: false, dedup: false, error: String(msg) }
  }
}

const handleBatchImport = async () => {
  if (!detail.value?.items?.length) return
  const targets = selectionMode.value && selectedIndices.value.size > 0
    ? Array.from(selectedIndices.value)
    : detail.value.items.map((i) => Number(i.index))

  if (!targets.length) return
  if (!userStore.currentUser) {
    show('请先选择身份后再导入上传')
    return
  }

  batchImport.running = true
  batchImport.total = targets.length
  batchImport.done = 0
  batchImport.success = 0
  batchImport.fail = 0

  try {
    for (const idx of targets.sort((a, b) => a - b)) {
      const r = await importIndex(idx)
      batchImport.done += 1
      if (r.ok) batchImport.success += 1
      else batchImport.fail += 1
    }
  } finally {
    batchImport.running = false
    show(`导入完成：成功 ${batchImport.success}，失败 ${batchImport.fail}`)
  }
}

const decodeMaybeEscapedFilename = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  try {
    return decodeURIComponent(raw)
  } catch {
    return raw
  }
}

const getFilenameFromContentDisposition = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  const m5987 = raw.match(/filename\\*\\s*=\\s*UTF-8''([^;]+)/i)
  if (m5987 && m5987[1]) return decodeMaybeEscapedFilename(m5987[1])
  const m = raw.match(/filename\\s*=\\s*\"([^\"]+)\"/i) || raw.match(/filename\\s*=\\s*([^;]+)/i)
  if (m && m[1]) return decodeMaybeEscapedFilename(m[1])
  return ''
}

const sanitizeFilename = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  return raw
    .replace(/[\\\\/:*?\"<>|]/g, '_')
    .replace(/[\\r\\n\\t]/g, ' ')
    .replace(/\\s+/g, ' ')
    .trim()
    .slice(0, 120)
}

const triggerBlobDownload = (blob: Blob, filename: string) => {
  const blobUrl = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = blobUrl
  link.download = filename || 'download'
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(blobUrl), 1000)
}

const downloadIndex = async (idx: number) => {
  if (!detail.value?.items) return { ok: false, error: '解析信息缺失' }
  const item = detail.value.items.find((i) => Number(i.index) === Number(idx))
  if (!item) return { ok: false, error: '资源不存在' }

  const token = getAuthToken()
  if (!token) return { ok: false, error: '未登录或Token缺失' }

  const resp = await fetch(item.downloadUrl, {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` }
  })
  if (!resp.ok) {
    let msg = `下载失败: ${resp.status}`
    try {
      const data = await resp.json()
      msg = data?.error || data?.msg || msg
    } catch {
      // ignore
    }
    return { ok: false, error: msg }
  }

  const blob = await resp.blob()
  const cd = resp.headers.get('Content-Disposition') || ''
  const filenameFromHeader = getFilenameFromContentDisposition(cd)
  const filename = filenameFromHeader || item.originalFilename || `download_${idx}`
  triggerBlobDownload(blob, sanitizeFilename(filename) || 'download')
  return { ok: true }
}

const handleBatchDownload = async () => {
  if (!detail.value?.items?.length) return
  const targets = selectionMode.value && selectedIndices.value.size > 0
    ? Array.from(selectedIndices.value)
    : detail.value.items.map((i) => Number(i.index))
  if (!targets.length) return

  if (targets.length >= 3) {
    const ok = window.confirm(`即将下载 ${targets.length} 个文件，浏览器可能会提示允许多个下载，是否继续？`)
    if (!ok) return
  }

  batchDownload.running = true
  batchDownload.total = targets.length
  batchDownload.done = 0
  batchDownload.success = 0
  batchDownload.fail = 0

  try {
    for (const idx of targets.sort((a, b) => a - b)) {
      const r = await downloadIndex(idx)
      batchDownload.done += 1
      if (r.ok) batchDownload.success += 1
      else batchDownload.fail += 1
    }
  } finally {
    batchDownload.running = false
    show(`下载完成：成功 ${batchDownload.success}，失败 ${batchDownload.fail}`)
  }
}

const confirmImportUpload = async () => {
  const idx = Number(previewIndex.value) || 0
  const r = await importIndex(idx)
  if (!r.ok) {
    show(`导入失败: ${r.error || '未知错误'}`)
  }
}
</script>
