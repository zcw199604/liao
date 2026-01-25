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
                ref="detailInputRef"
                v-model="inputText"
                @paste="handleInputPaste"
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
                ref="accountInputRef"
                v-model="accountInput"
                @paste="handleInputPaste"
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
              <div class="sticky top-0 z-10 bg-[#18181b] py-2 space-y-2">
                <div class="flex items-center justify-between gap-2">
                  <div class="flex items-center gap-2">
                    <button
                      class="px-3 py-2 rounded-xl border transition text-xs"
                      :class="favoritesTab === 'users' ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                      :disabled="favoritesLoading"
                      @click="setFavoritesTab('users')"
                    >
                      用户收藏
                    </button>
                    <button
                      class="px-3 py-2 rounded-xl border transition text-xs"
                      :class="favoritesTab === 'awemes' ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                      :disabled="favoritesLoading"
                      @click="setFavoritesTab('awemes')"
                    >
                      作品收藏
                    </button>
                  </div>

                  <div class="flex items-center gap-2">
                    <button
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs"
                      :disabled="favoritesLoading || uiDisabled"
                      @click="toggleFavoritesSelectionMode"
                      title="批量选择"
                    >
                      {{ favoritesSelectionMode ? '完成' : '批量' }}
                    </button>
                    <button
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs"
                      :disabled="favoritesLoading"
                      @click="refreshFavorites"
                    >
                      {{ favoritesLoading ? '刷新中…' : '刷新' }}
                    </button>
                  </div>
                </div>

                <div class="flex items-center gap-2">
                  <div class="flex-1 flex items-center gap-2 overflow-x-auto no-scrollbar">
                    <button
                      class="px-3 py-1.5 rounded-full border transition text-xs whitespace-nowrap"
                      :class="currentFavoriteTagFilter === null ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                      :disabled="favoritesLoading"
                      @click="setCurrentFavoriteTagFilterValue(null)"
                    >
                      全部
                    </button>
                    <button
                      class="px-3 py-1.5 rounded-full border transition text-xs whitespace-nowrap"
                      :class="currentFavoriteTagFilter === -1 ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                      :disabled="favoritesLoading"
                      @click="setCurrentFavoriteTagFilterValue(-1)"
                    >
                      未分类
                    </button>
                    <button
                      v-for="t in currentFavoriteTags"
                      :key="`douyin-fav-tag-${favoritesTab}-${t.id}`"
                      class="px-3 py-1.5 rounded-full border transition text-xs max-w-[140px] truncate"
                      :class="currentFavoriteTagFilter === t.id ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                      :disabled="favoritesLoading"
                      @click="setCurrentFavoriteTagFilterValue(t.id)"
                      :title="t.name"
                    >
                      {{ t.name }}
                    </button>
                  </div>

                  <button
                    class="w-10 h-10 flex items-center justify-center bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition disabled:opacity-60 disabled:cursor-not-allowed"
                    :disabled="favoritesLoading || uiDisabled"
                    @click="openTagManager(favoritesTab)"
                    title="管理标签"
                  >
                    <i class="fas fa-tags text-sm"></i>
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
                <div v-if="filteredFavoriteUsers.length === 0" class="mt-4 text-sm text-gray-500">
                  {{ favoriteUsers.length === 0 ? '暂无收藏用户' : '暂无匹配结果' }}
                </div>
	                <div v-else class="mt-4 space-y-3">
	                  <div
	                    v-for="u in filteredFavoriteUsers"
	                    :key="u.secUserId"
	                    class="rounded-xl border bg-black/20 p-3 active:bg-white/5 transition"
                      :class="favoritesSelectionMode && selectedFavoriteUserIds.has(String(u.secUserId || '').trim()) ? 'border-emerald-500 ring-2 ring-emerald-500/30' : 'border-gray-700'"
	                    role="button"
	                    @click="favoritesSelectionMode ? toggleSelectFavoriteUser(u.secUserId) : openFavoriteUserDetail(u)"
	                  >
	                    <div class="flex items-start gap-3">
                        <div v-if="favoritesSelectionMode" class="pt-1">
                          <div
                            class="w-5 h-5 rounded-full border flex items-center justify-center"
                            :class="selectedFavoriteUserIds.has(String(u.secUserId || '').trim()) ? 'border-emerald-400 bg-emerald-600/10' : 'border-gray-600'"
                          >
                            <i
                              v-if="selectedFavoriteUserIds.has(String(u.secUserId || '').trim())"
                              class="fas fa-check text-emerald-400 text-[10px]"
                            ></i>
                          </div>
                        </div>
	                      <div class="w-12 h-12 rounded-full overflow-hidden bg-[#111113] border border-gray-700 flex-shrink-0">
	                        <img
	                          v-if="u.avatarUrl && !favoriteUserAvatarError.has(u.secUserId)"
	                          :src="u.avatarUrl"
	                          class="w-full h-full object-cover"
	                          referrerpolicy="no-referrer"
	                          alt=""
	                          @error="markFavoriteUserAvatarError(u.secUserId)"
	                        />
	                        <div v-else class="w-full h-full flex items-center justify-center text-gray-600 text-sm">
	                          <i class="fas fa-user"></i>
	                        </div>
	                      </div>
	
	                      <div class="min-w-0 flex-1">
	                        <div class="flex items-start justify-between gap-2">
	                            <div class="min-w-0">
	                            <div class="text-white text-sm font-medium whitespace-normal break-words">
	                              {{ u.displayName || '（未命名用户）' }}
	                            </div>
	                            <div class="text-xs text-gray-500 font-mono truncate">
	                              {{ u.secUserId }}
	                            </div>
	                            <div v-if="u.signature" class="text-xs text-gray-400 mt-1 line-clamp-1">
	                              {{ u.signature }}
	                            </div>
	
	                            <div
	                              v-if="u.followerCount || u.followingCount || u.awemeCount || u.totalFavorited"
	                              class="mt-1 flex flex-wrap gap-2 text-[10px] text-gray-400"
	                            >
	                              <span v-if="u.followerCount" class="bg-white/5 px-1.5 py-0.5 rounded">粉丝 {{ formatDouyinCount(u.followerCount) }}</span>
	                              <span v-if="u.followingCount" class="bg-white/5 px-1.5 py-0.5 rounded">关注 {{ formatDouyinCount(u.followingCount) }}</span>
	                              <span v-if="u.awemeCount" class="bg-white/5 px-1.5 py-0.5 rounded">作品 {{ formatDouyinCount(u.awemeCount) }}</span>
	                              <span v-if="u.totalFavorited" class="bg-white/5 px-1.5 py-0.5 rounded">获赞 {{ formatDouyinCount(u.totalFavorited) }}</span>
	                            </div>
	
	                            <div v-if="favoriteUserAvatarError.has(u.secUserId)" class="text-[10px] text-amber-300 mt-1">
	                              头像可能已更换，点开可刷新
	                            </div>
	
	                            <div v-if="u.lastParsedAt || u.lastParsedCount" class="text-xs text-gray-500 mt-1">
	                              <span v-if="u.lastParsedAt">上次解析: {{ u.lastParsedAt }}</span>
	                              <span v-if="u.lastParsedCount"> · 作品: {{ u.lastParsedCount }}</span>
	                            </div>

                              <div class="mt-2 flex flex-wrap gap-1">
                                <template v-if="Array.isArray(u.tagIds) && u.tagIds.length > 0">
                                  <span
                                    v-for="id in u.tagIds.slice(0, 2)"
                                    :key="`user-tag-${u.secUserId}-${id}`"
                                    class="text-[10px] bg-white/5 text-gray-300 px-2 py-0.5 rounded-full border border-white/10 max-w-[140px] truncate"
                                    :title="getFavoriteTagName('users', id)"
                                  >
                                    {{ getFavoriteTagName('users', id) }}
                                  </span>
                                  <span v-if="u.tagIds.length > 2" class="text-[10px] text-gray-500">+{{ u.tagIds.length - 2 }}</span>
                                </template>
                                <span v-else class="text-[10px] bg-white/5 text-gray-500 px-2 py-0.5 rounded-full border border-white/10">
                                  未分类
                                </span>
                              </div>
	                          </div>
	
	                          <div class="flex flex-col gap-2 flex-shrink-0 sm:flex-row sm:items-center">
	                            <template v-if="!favoritesSelectionMode">
	                              <button
	                                class="px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
	                                :disabled="uiDisabled"
	                                @click.stop="reparseFavoriteUser(u)"
	                              >
	                                再次解析
	                              </button>
	                              <button
	                                class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
	                                :disabled="uiDisabled"
	                                @click.stop="removeFavoriteUser(u.secUserId)"
	                              >
	                                取消
	                              </button>
	                            </template>
	                          </div>
	                        </div>
	                      </div>
	                    </div>
	                  </div>
	                </div>
              </template>

              <template v-else>
                <div v-if="filteredFavoriteAwemes.length === 0" class="mt-4 text-sm text-gray-500">
                  {{ favoriteAwemes.length === 0 ? '暂无收藏作品' : '暂无匹配结果' }}
                </div>
                <div v-else class="mt-4 space-y-3">
                  <div
                    v-for="it in filteredFavoriteAwemes"
                    :key="it.awemeId"
                    class="rounded-xl border bg-black/20 p-3 flex gap-3 active:bg-white/5 transition"
                    :class="favoritesSelectionMode && selectedFavoriteAwemeIds.has(String(it.awemeId || '').trim()) ? 'border-emerald-500 ring-2 ring-emerald-500/30' : 'border-gray-700'"
                    role="button"
                    @click="favoritesSelectionMode ? toggleSelectFavoriteAweme(it.awemeId) : null"
                  >
                    <div v-if="favoritesSelectionMode" class="pt-1">
                      <div
                        class="w-5 h-5 rounded-full border flex items-center justify-center"
                        :class="selectedFavoriteAwemeIds.has(String(it.awemeId || '').trim()) ? 'border-emerald-400 bg-emerald-600/10' : 'border-gray-600'"
                      >
                        <i
                          v-if="selectedFavoriteAwemeIds.has(String(it.awemeId || '').trim())"
                          class="fas fa-check text-emerald-400 text-[10px]"
                        ></i>
                      </div>
                    </div>
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
                      <div class="text-white text-sm font-medium whitespace-normal break-words">
                        {{ it.desc || '（无描述）' }}
                      </div>
                      <div class="text-xs text-gray-500 font-mono truncate mt-1">
                        {{ it.awemeId }}
                      </div>

                      <div class="mt-2 flex flex-wrap gap-1">
                        <template v-if="Array.isArray(it.tagIds) && it.tagIds.length > 0">
                          <span
                            v-for="id in it.tagIds.slice(0, 2)"
                            :key="`aweme-tag-${it.awemeId}-${id}`"
                            class="text-[10px] bg-white/5 text-gray-300 px-2 py-0.5 rounded-full border border-white/10 max-w-[140px] truncate"
                            :title="getFavoriteTagName('awemes', id)"
                          >
                            {{ getFavoriteTagName('awemes', id) }}
                          </span>
                          <span v-if="it.tagIds.length > 2" class="text-[10px] text-gray-500">+{{ it.tagIds.length - 2 }}</span>
                        </template>
                        <span v-else class="text-[10px] bg-white/5 text-gray-500 px-2 py-0.5 rounded-full border border-white/10">
                          未分类
                        </span>
                      </div>

                      <div v-if="!favoritesSelectionMode" class="flex flex-col gap-2 mt-2 sm:flex-row sm:items-center">
                        <button
                          class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="uiDisabled"
                          @click.stop="openTagSheet({ kind: 'awemes', mode: 'single', targetIds: [it.awemeId], presetTagIds: it.tagIds || [] })"
                        >
                          标签
                        </button>
                        <button
                          class="px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="uiDisabled"
                          @click.stop="reparseFavoriteAweme(it)"
                        >
                          解析
                        </button>
                        <button
                          class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="uiDisabled"
                          @click.stop="removeFavoriteAweme(it.awemeId)"
                        >
                          取消
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </template>

              <div
                v-if="favoritesSelectionMode"
                class="sticky bottom-0 z-10 mt-4 pt-3 pb-2 bg-[#18181b] border-t border-gray-800"
              >
                <div class="flex items-center justify-between gap-3">
                  <div class="text-xs text-gray-400">
                    已选 {{ selectedFavoritesCount }} 项
                  </div>
                  <div class="flex items-center gap-2">
                    <button
                      class="px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                      :disabled="uiDisabled || selectedFavoritesCount === 0"
                      @click="openBatchTagSheet"
                    >
                      打标签
                    </button>
                    <button
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                      :disabled="uiDisabled"
                      @click="toggleFavoritesSelectionMode"
                    >
                      取消选择
                    </button>
                  </div>
                </div>
              </div>
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
                      :disabled="batchImport.running || batchDownload.running"
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

        <!-- 收藏标签编辑（底部抽屉） -->
        <div
          v-if="tagSheetOpen"
          class="fixed inset-0 z-[87] bg-black/60 backdrop-blur-sm"
          @click.stop="closeTagSheet"
        >
          <div class="fixed bottom-0 left-0 right-0 z-[88] bg-[#18181b] rounded-t-2xl shadow-2xl border-t border-white/10" @click.stop>
            <div class="w-full h-6 flex items-center justify-center">
              <div class="w-10 h-1 bg-white/20 rounded-full"></div>
            </div>

            <div class="px-5 pb-5 max-h-[75vh] overflow-y-auto no-scrollbar">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-white text-base font-semibold">
                    {{ tagSheetMode === 'batch' ? '批量打标签' : '编辑标签' }}
                  </div>
                  <div class="text-xs text-gray-500 mt-1">
                    <template v-if="tagSheetMode === 'batch'">
                      将把所选标签添加到 {{ tagSheetTargetIds.length }} 个条目（不会移除已有标签）
                    </template>
                    <template v-else>
                      可选择多个标签；不选则为未分类
                    </template>
                  </div>
                </div>

                <button
                  class="w-9 h-9 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a] flex-shrink-0"
                  type="button"
                  @click="closeTagSheet"
                  title="关闭"
                >
                  <i class="fas fa-times"></i>
                </button>
              </div>

              <div v-if="tagSheetError" class="mt-3 text-sm text-red-400">
                {{ tagSheetError }}
              </div>

              <div
                v-if="(tagSheetKind === 'users' ? favoriteUserTags.length : favoriteAwemeTags.length) === 0"
                class="mt-4 text-sm text-gray-500"
              >
                暂无标签，可先去“管理标签”创建。
              </div>

              <div class="mt-4 flex flex-wrap gap-2">
                <button
                  v-for="t in (tagSheetKind === 'users' ? favoriteUserTags : favoriteAwemeTags)"
                  :key="`tag-sheet-${tagSheetKind}-${t.id}`"
                  class="px-3 py-2 rounded-full border transition text-xs max-w-[160px] truncate"
                  :class="tagSheetSelectedTagIds.has(t.id) ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-[#27272a] border-gray-700 text-gray-200 hover:bg-gray-700'"
                  @click="toggleTagInSheet(t.id)"
                  :title="t.name"
                >
                  {{ t.name }}
                </button>
              </div>

              <div class="mt-5 flex items-center justify-between gap-3">
                <button
                  class="px-4 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tagSheetApplying"
                  @click="openTagManager(tagSheetKind); closeTagSheet()"
                >
                  管理标签
                </button>
                <button
                  class="flex-1 px-4 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tagSheetApplying || (tagSheetMode === 'batch' && tagSheetSelectedTagIds.size === 0)"
                  @click="applyTagSheet"
                >
                  <i v-if="tagSheetApplying" class="fas fa-spinner fa-spin mr-2"></i>
                  <span>{{ tagSheetMode === 'batch' ? '添加' : '保存' }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 收藏标签管理（独立页面） -->
        <div
          v-if="tagManagerOpen"
          class="fixed inset-0 z-[90] bg-[#18181b] flex flex-col"
          @click.stop
        >
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
            <button
              class="w-9 h-9 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
              type="button"
              @click="closeTagManager"
              title="返回"
            >
              <i class="fas fa-arrow-left"></i>
            </button>

            <div class="text-white font-bold text-base">
              {{ tagManagerKind === 'users' ? '管理用户标签' : '管理作品标签' }}
            </div>

            <button
              class="w-9 h-9 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
              type="button"
              @click="closeTagManager"
              title="关闭"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>

          <div class="flex-1 overflow-y-auto p-6 no-scrollbar space-y-3">
            <div class="text-xs text-gray-500">
              标签全局共享；删除标签会从所有收藏条目移除（条目保留）。
            </div>
            <div class="text-xs text-gray-500">
              可拖拽左侧按钮调整展示顺序（自动保存）。
            </div>

            <div class="flex items-center gap-2">
              <input
                v-model="tagManagerNameInput"
                class="flex-1 bg-[#111113] border border-gray-700 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500"
                placeholder="新建标签名称（仅名称）"
                :disabled="tagManagerSaving"
                @keyup.enter="handleCreateTag"
              />
              <button
                class="px-4 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                :disabled="tagManagerSaving || !String(tagManagerNameInput || '').trim()"
                @click="handleCreateTag"
              >
                创建
              </button>
            </div>

            <div v-if="tagManagerError" class="text-sm text-red-400">
              {{ tagManagerError }}
            </div>

            <div v-if="(tagManagerKind === 'users' ? favoriteUserTags.length : favoriteAwemeTags.length) === 0" class="text-sm text-gray-500">
              暂无标签
            </div>

            <draggable
              v-else
              v-model="tagManagerTags"
              item-key="id"
              handle=".tag-drag-handle"
              :disabled="tagManagerSaving || editingTagId !== null"
              class="space-y-2"
              @end="saveTagManagerOrder"
            >
              <template #item="{ element: t }">
                <div class="rounded-xl border border-gray-700 bg-black/20 p-3">
                  <div class="flex items-start gap-3">
                    <button
                      class="tag-drag-handle w-9 h-9 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a] flex-shrink-0 cursor-grab active:cursor-grabbing disabled:opacity-60 disabled:cursor-not-allowed"
                      type="button"
                      :disabled="tagManagerSaving || editingTagId !== null"
                      title="拖拽排序"
                    >
                      <i class="fas fa-bars"></i>
                    </button>

                    <div class="min-w-0 flex-1">
                      <template v-if="editingTagId === t.id">
                        <input
                          v-model="editingTagName"
                          class="w-full bg-[#111113] border border-gray-700 rounded-xl px-3 py-2 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500 text-sm"
                          placeholder="标签名称"
                          :disabled="tagManagerSaving"
                          @keyup.enter="saveEditTag"
                        />
                      </template>
                      <template v-else>
                        <div class="text-white text-sm font-medium truncate">
                          {{ t.name }}
                        </div>
                      </template>

                      <div class="text-xs text-gray-500 mt-1">
                        {{ t.count }} 个{{ tagManagerKind === 'users' ? '用户' : '作品' }}
                      </div>
                    </div>

                    <div class="flex items-center gap-2 flex-shrink-0">
                      <template v-if="editingTagId === t.id">
                        <button
                          class="px-3 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="tagManagerSaving || !String(editingTagName || '').trim()"
                          @click="saveEditTag"
                        >
                          保存
                        </button>
                        <button
                          class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="tagManagerSaving"
                          @click="cancelEditTag"
                        >
                          取消
                        </button>
                      </template>
                      <template v-else>
                        <button
                          class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="tagManagerSaving"
                          @click="startEditTag(t)"
                        >
                          重命名
                        </button>
                        <button
                          class="px-3 py-2 bg-red-600/20 hover:bg-red-600/30 text-red-300 rounded-xl border border-red-500/30 transition text-xs disabled:opacity-60 disabled:cursor-not-allowed"
                          :disabled="tagManagerSaving"
                          @click="askDeleteTag(t)"
                        >
                          删除
                        </button>
                      </template>
                    </div>
                  </div>
                </div>
              </template>
            </draggable>
          </div>

          <div
            v-if="confirmDeleteTagOpen && confirmDeleteTag"
            class="fixed inset-0 z-[91] bg-black/70 flex items-center justify-center px-6"
            @click.stop="cancelDeleteTag"
          >
            <div class="w-full max-w-sm bg-[#18181b] rounded-2xl shadow-2xl border border-white/10 p-5" @click.stop>
              <div class="text-white font-semibold text-base">
                删除标签“{{ confirmDeleteTag.name }}”？
              </div>
              <div class="text-sm text-gray-400 mt-2 leading-relaxed">
                删除后，该标签将从 {{ confirmDeleteTag.count }} 个收藏条目中移除，收藏条目本身不会被删除。
              </div>
              <div class="mt-5 flex items-center gap-2">
                <button
                  class="flex-1 px-4 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tagManagerSaving"
                  @click="cancelDeleteTag"
                >
                  取消
                </button>
                <button
                  class="flex-1 px-4 py-3 bg-red-600 hover:bg-red-700 text-white rounded-xl transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tagManagerSaving"
                  @click="confirmDeleteTagNow"
                >
                  确认删除
                </button>
              </div>
            </div>
          </div>
        </div>

	      <!-- 收藏用户详情（移动端底部抽屉） -->
	      <div
	        v-if="favoriteUserDetailOpen"
	        class="fixed inset-0 z-[85] bg-black/60 backdrop-blur-sm"
	        @click.stop="closeFavoriteUserDetail"
	      >
	        <div class="fixed bottom-0 left-0 right-0 z-[86] bg-[#18181b] rounded-t-2xl shadow-2xl border-t border-white/10" @click.stop>
	          <div class="w-full h-6 flex items-center justify-center">
	            <div class="w-10 h-1 bg-white/20 rounded-full"></div>
	          </div>

		          <div
		            v-if="selectedFavoriteUser"
		            class="px-5 pb-5 max-h-[75vh] overflow-y-auto no-scrollbar"
		            @scroll.passive="handleFavoriteUserDetailScroll"
		          >
	            <div class="flex items-start justify-between gap-3">
	              <div class="min-w-0">
	                <div class="text-white text-base font-semibold whitespace-normal break-words">
	                  {{ selectedFavoriteUser.displayName || '（未命名用户）' }}
	                </div>
	                <button
	                  class="mt-1 text-xs text-gray-400 font-mono truncate flex items-center gap-2"
	                  type="button"
	                  @click="copyText(selectedFavoriteUser.secUserId)"
	                  title="复制 sec_user_id"
	                >
	                  <span class="truncate">{{ selectedFavoriteUser.secUserId }}</span>
	                  <i class="far fa-copy text-gray-500"></i>
	                </button>
	              </div>

	              <button
	                class="w-9 h-9 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a] flex-shrink-0"
	                type="button"
	                @click="closeFavoriteUserDetail"
	                title="关闭"
	              >
	                <i class="fas fa-times"></i>
	              </button>
	            </div>

	            <div class="mt-4 flex items-start gap-4">
	              <div class="relative w-20 h-20 rounded-full overflow-hidden bg-[#111113] border border-gray-700 flex-shrink-0">
	                <img
	                  v-if="selectedFavoriteUser.avatarUrl && !favoriteUserAvatarError.has(selectedFavoriteUser.secUserId)"
	                  :src="selectedFavoriteUser.avatarUrl"
	                  class="w-full h-full object-cover"
	                  referrerpolicy="no-referrer"
	                  alt=""
	                  @error="markFavoriteUserAvatarError(selectedFavoriteUser.secUserId)"
	                />
	                <div v-else class="w-full h-full flex items-center justify-center text-gray-600 text-xl">
	                  <i class="fas fa-user"></i>
	                </div>

	                <div
	                  v-if="favoriteUserAvatarError.has(selectedFavoriteUser.secUserId)"
	                  class="absolute inset-0 bg-black/50 flex items-center justify-center text-[10px] text-white backdrop-blur-[2px]"
	                >
	                  头像可能已更换
	                </div>
	              </div>

	              <div class="min-w-0 flex-1">
	                <div v-if="selectedFavoriteUser.signature" class="text-xs text-gray-300 leading-relaxed whitespace-pre-line">
	                  {{ selectedFavoriteUser.signature }}
	                </div>
	                <div v-else class="text-xs text-gray-500">
	                  （暂无简介）
	                </div>

	                <a
	                  v-if="selectedFavoriteUser.profileUrl"
	                  class="inline-flex items-center gap-2 mt-3 text-xs text-emerald-400 hover:text-emerald-300"
	                  :href="selectedFavoriteUser.profileUrl"
	                  target="_blank"
	                  rel="noopener"
	                >
	                  <i class="fas fa-external-link-alt"></i>
	                  打开主页
	                </a>
	              </div>
	            </div>

	            <button
	              class="mt-4 w-full flex items-center justify-center gap-2 px-4 py-3 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl transition disabled:opacity-60 disabled:cursor-not-allowed"
	              type="button"
	              :disabled="favoriteUserDetailLoading"
	              @click="refreshSelectedFavoriteUser"
	            >
	              <i v-if="favoriteUserDetailLoading" class="fas fa-spinner fa-spin"></i>
	              <span>{{ favoriteUserDetailLoading ? '更新中…' : '刷新信息' }}</span>
	            </button>

              <div class="mt-4">
                <div class="flex items-center justify-between gap-3">
                  <div class="text-xs text-gray-500">
                    分类标签
                  </div>
                  <button
                    class="text-xs text-emerald-400 hover:text-emerald-300 disabled:opacity-60 disabled:cursor-not-allowed"
                    type="button"
                    :disabled="uiDisabled"
                    @click="openTagSheet({ kind: 'users', mode: 'single', targetIds: [selectedFavoriteUser.secUserId], presetTagIds: selectedFavoriteUser.tagIds || [] })"
                  >
                    编辑
                  </button>
                </div>

                <div class="mt-2 flex flex-wrap gap-2">
                  <template v-if="Array.isArray(selectedFavoriteUser.tagIds) && selectedFavoriteUser.tagIds.length > 0">
                    <span
                      v-for="id in selectedFavoriteUser.tagIds"
                      :key="`user-detail-tag-${selectedFavoriteUser.secUserId}-${id}`"
                      class="text-xs bg-white/5 text-gray-200 px-3 py-1 rounded-full border border-white/10 max-w-[220px] truncate"
                      :title="getFavoriteTagName('users', id)"
                    >
                      {{ getFavoriteTagName('users', id) }}
                    </span>
                  </template>
                  <span v-else class="text-xs bg-white/5 text-gray-500 px-3 py-1 rounded-full border border-white/10">
                    未分类
                  </span>
                </div>
              </div>

		            <div v-if="selectedFavoriteUser.lastParsedAt || selectedFavoriteUser.lastParsedCount" class="mt-4 text-xs text-gray-500">
		              <span v-if="selectedFavoriteUser.lastParsedAt">上次解析: {{ selectedFavoriteUser.lastParsedAt }}</span>
		              <span v-if="selectedFavoriteUser.lastParsedCount"> · 作品: {{ selectedFavoriteUser.lastParsedCount }}</span>
		            </div>

	            <div class="mt-4 flex flex-col gap-2 sm:flex-row sm:items-center">
	              <button
	                class="flex-1 px-4 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
	                type="button"
	                :disabled="uiDisabled"
	                @click="reparseFavoriteUser(selectedFavoriteUser)"
	              >
	                再次解析作品
	              </button>
		              <button
		                class="flex-1 px-4 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
		                type="button"
		                :disabled="uiDisabled"
		                @click="removeFavoriteUser(selectedFavoriteUser.secUserId)"
		              >
		                取消收藏
		              </button>
		            </div>

		            <div class="mt-6 border-t border-white/10 pt-4">
		              <div class="flex items-center justify-between gap-3">
		                <div class="text-xs text-gray-400">
		                  已加载 {{ favoriteUserWorks.length }} 个作品
		                </div>
		                <button
		                  class="px-3 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl transition text-xs flex items-center gap-2 disabled:opacity-60 disabled:cursor-not-allowed"
		                  type="button"
		                  :disabled="uiDisabled || favoriteUserWorksPullingLatest"
		                  @click="pullLatestFavoriteUserWorks"
		                >
		                  <i v-if="favoriteUserWorksPullingLatest" class="fas fa-spinner fa-spin"></i>
		                  <span>{{ favoriteUserWorksPullingLatest ? '获取中…' : '获取最新作品' }}</span>
		                </button>
		              </div>

		              <div v-if="favoriteUserWorksError" class="mt-2 text-xs text-red-400">
		                {{ favoriteUserWorksError }}
		              </div>

		              <div v-if="favoriteUserWorks.length > 0" class="mt-3 grid grid-cols-2 sm:grid-cols-3 gap-3">
		                <button
		                  v-for="item in favoriteUserWorks"
		                  :key="`douyin-favorite-user-aweme-${favoriteUserDetailId}-${item.detailId}`"
		                  class="rounded-xl overflow-hidden border border-gray-700 hover:border-emerald-500 transition bg-black/20 text-left relative group"
		                  type="button"
		                  :disabled="favoriteUserWorkLoading.has(item.detailId)"
		                  @click="openFavoriteUserWork(item)"
		                  :title="item.desc || item.detailId"
		                >
		                  <div
		                    v-if="favoriteUserWorkLoading.has(item.detailId)"
		                    class="absolute inset-0 z-20 bg-black/60 flex items-center justify-center backdrop-blur-sm"
		                  >
		                    <i class="fas fa-spinner fa-spin text-white text-xl"></i>
		                  </div>
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
		              <div v-else-if="favoriteUserWorksLoading" class="mt-3 text-sm text-gray-500">
		                获取中…
		              </div>
		              <div v-else class="mt-3 text-sm text-gray-500">
		                暂无入库作品
		              </div>

		              <div v-if="favoriteUserWorksLoading && favoriteUserWorks.length > 0" class="mt-3 text-xs text-gray-500">
		                加载中…
		              </div>
		              <div v-else-if="favoriteUserWorksHasMore && favoriteUserWorks.length > 0" class="mt-3 text-xs text-gray-500">
		                下拉加载更多
		              </div>
		              <div v-else-if="!favoriteUserWorksHasMore && favoriteUserWorks.length > 0" class="mt-3 text-xs text-gray-500">
		                已加载全部
		              </div>
		            </div>
		          </div>

	          <div v-else class="px-5 pb-6 text-sm text-gray-400">
	            加载中…
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
import { computed, nextTick, reactive, ref, watch } from 'vue'
import draggable from 'vuedraggable'
import { useDouyinStore } from '@/stores/douyin'
import { useUserStore } from '@/stores/user'
import { useMediaStore } from '@/stores/media'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useToast } from '@/composables/useToast'
import MediaTile from '@/components/common/MediaTile.vue'
import MediaTileBadge from '@/components/common/MediaTileBadge.vue'
import MediaTileSelectMark from '@/components/common/MediaTileSelectMark.vue'
import { generateCookie } from '@/utils/cookie'
import { copyToClipboard } from '@/utils/clipboard'
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
  displayName?: string
  signature?: string
  avatarUrl?: string
  profileUrl?: string
  followerCount?: number
  followingCount?: number
  awemeCount?: number
  totalFavorited?: number
  tab: string
  cursor: number
  hasMore: boolean
  items: DouyinAccountItem[]
}

interface DouyinFavoriteUser {
  secUserId: string
  sourceInput?: string
  displayName?: string
  signature?: string
  avatarUrl?: string
  profileUrl?: string
  followerCount?: number
  followingCount?: number
  awemeCount?: number
  totalFavorited?: number
  lastParsedAt?: string
  lastParsedCount?: number
  createTime: string
  updateTime: string
  tagIds?: number[]
}

interface DouyinFavoriteAweme {
  awemeId: string
  secUserId?: string
  type?: 'image' | 'video'
  desc?: string
  coverUrl?: string
  createTime: string
  updateTime: string
  tagIds?: number[]
}

interface DouyinFavoriteTag {
  id: number
  name: string
  sortOrder?: number
  count: number
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
const detailInputRef = ref<HTMLTextAreaElement | null>(null)
const accountInputRef = ref<HTMLTextAreaElement | null>(null)
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
const accountDisplayName = ref('')
const accountSignature = ref('')
const accountAvatarUrl = ref('')
const accountProfileUrl = ref('')
const accountFollowerCount = ref<number | null>(null)
const accountFollowingCount = ref<number | null>(null)
const accountAwemeCount = ref<number | null>(null)
const accountTotalFavorited = ref<number | null>(null)
const accountQueried = ref(false)
const accountItemLoading = reactive<Set<string>>(new Set())

const favoritesTab = ref<'users' | 'awemes'>('users')
const favoritesLoading = ref(false)
const favoritesError = ref('')
const favoriteUsers = ref<DouyinFavoriteUser[]>([])
const favoriteAwemes = ref<DouyinFavoriteAweme[]>([])
const favoriteUserTags = ref<DouyinFavoriteTag[]>([])
const favoriteAwemeTags = ref<DouyinFavoriteTag[]>([])
const favoriteUserTagFilter = ref<number | null>(null) // null=全部, -1=未分类, 其他=tagId
const favoriteAwemeTagFilter = ref<number | null>(null) // null=全部, -1=未分类, 其他=tagId

const favoritesSelectionMode = ref(false)
const selectedFavoriteUserIds = reactive<Set<string>>(new Set())
const selectedFavoriteAwemeIds = reactive<Set<string>>(new Set())

const tagManagerOpen = ref(false)
const tagManagerKind = ref<'users' | 'awemes'>('users')
const tagManagerTags = computed<DouyinFavoriteTag[]>({
  get: () => (tagManagerKind.value === 'users' ? favoriteUserTags.value : favoriteAwemeTags.value),
  set: (v) => {
    if (tagManagerKind.value === 'users') favoriteUserTags.value = v
    else favoriteAwemeTags.value = v
  }
})
const tagManagerNameInput = ref('')
const tagManagerSaving = ref(false)
const tagManagerError = ref('')
const editingTagId = ref<number | null>(null)
const editingTagName = ref('')
const confirmDeleteTagOpen = ref(false)
const confirmDeleteTag = ref<DouyinFavoriteTag | null>(null)

const tagSheetOpen = ref(false)
const tagSheetKind = ref<'users' | 'awemes'>('users')
const tagSheetMode = ref<'single' | 'batch'>('single')
const tagSheetTargetIds = ref<string[]>([])
const tagSheetSelectedTagIds = reactive<Set<number>>(new Set())
const tagSheetApplying = ref(false)
const tagSheetError = ref('')

	const favoriteUserDetailOpen = ref(false)
	const favoriteUserDetailId = ref('')
	const favoriteUserDetailLoading = ref(false)
	const favoriteUserAvatarError = reactive<Set<string>>(new Set())
	const selectedFavoriteUser = computed(() =>
	  favoriteUsers.value.find((u) => String(u.secUserId || '').trim() === String(favoriteUserDetailId.value || '').trim()) || null
	)

	const favoriteUserWorksLoading = ref(false)
	const favoriteUserWorksPullingLatest = ref(false)
	const favoriteUserWorksError = ref('')
	const favoriteUserWorks = ref<DouyinAccountItem[]>([])
	const favoriteUserWorksCursor = ref(0)
	const favoriteUserWorksHasMore = ref(false)
	const favoriteUserWorkLoading = reactive<Set<string>>(new Set())

const favoriteUserIdSet = computed(() => new Set(favoriteUsers.value.map((u) => String(u.secUserId || '').trim()).filter(Boolean)))
const favoriteAwemeIdSet = computed(() => new Set(favoriteAwemes.value.map((it) => String(it.awemeId || '').trim()).filter(Boolean)))

const isFavoriteUser = (secUserId: string) => favoriteUserIdSet.value.has(String(secUserId || '').trim())
const isFavoriteAweme = (awemeId: string) => favoriteAwemeIdSet.value.has(String(awemeId || '').trim())

const favoriteUserTagNameById = computed(() => {
  const m = new Map<number, string>()
  for (const t of favoriteUserTags.value) {
    m.set(Number(t.id), String(t.name || '').trim())
  }
  return m
})

const favoriteAwemeTagNameById = computed(() => {
  const m = new Map<number, string>()
  for (const t of favoriteAwemeTags.value) {
    m.set(Number(t.id), String(t.name || '').trim())
  }
  return m
})

const getFavoriteTagName = (kind: 'users' | 'awemes', id: number) => {
  const map = kind === 'users' ? favoriteUserTagNameById.value : favoriteAwemeTagNameById.value
  return map.get(Number(id)) || `#${id}`
}

const filteredFavoriteUsers = computed(() => {
  const filter = favoriteUserTagFilter.value
  if (filter === null) return favoriteUsers.value
  if (filter === -1) return favoriteUsers.value.filter((u) => (Array.isArray(u.tagIds) ? u.tagIds : []).length === 0)
  return favoriteUsers.value.filter((u) => (Array.isArray(u.tagIds) ? u.tagIds : []).includes(Number(filter)))
})

const filteredFavoriteAwemes = computed(() => {
  const filter = favoriteAwemeTagFilter.value
  if (filter === null) return favoriteAwemes.value
  if (filter === -1) return favoriteAwemes.value.filter((it) => (Array.isArray(it.tagIds) ? it.tagIds : []).length === 0)
  return favoriteAwemes.value.filter((it) => (Array.isArray(it.tagIds) ? it.tagIds : []).includes(Number(filter)))
})

const currentFavoriteTags = computed(() => (favoritesTab.value === 'users' ? favoriteUserTags.value : favoriteAwemeTags.value))
const currentFavoriteTagFilter = computed(() => (favoritesTab.value === 'users' ? favoriteUserTagFilter.value : favoriteAwemeTagFilter.value))
const selectedFavoritesCount = computed(() => (favoritesTab.value === 'users' ? selectedFavoriteUserIds.size : selectedFavoriteAwemeIds.size))

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewMediaList = ref<UploadedMedia[]>([])
const previewIndex = ref(0)
const previewContextKey = ref('')
const previewContextItems = ref<DouyinDetailItem[]>([])

const canUpload = computed(() => true)

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

const focusActiveInput = async () => {
  await nextTick()
  const el = activeMode.value === 'account' ? accountInputRef.value : detailInputRef.value
  el?.focus()
}

const handleInputPaste = async (e: ClipboardEvent) => {
  const txt =
    e.clipboardData?.getData('text/plain') ||
    e.clipboardData?.getData('text') ||
    ''
  if (!String(txt || '').trim()) return

  e.preventDefault()
  applyInputText(txt)

  if (autoResolveClipboard.value && !uiDisabled.value) {
    await handlePrimaryAction()
  }
}

const pasteFromClipboard = async () => {
  const readText = navigator.clipboard?.readText
  if (typeof readText !== 'function') {
    await focusActiveInput()
    show('当前浏览器不支持一键读取剪贴板，请长按输入框手动粘贴')
    return
  }

  try {
    const txt = await readText.call(navigator.clipboard)
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
    await focusActiveInput()
    show('无法读取剪贴板（可能需要浏览器授权或 https），请长按输入框手动粘贴')
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
	accountDisplayName.value = ''
	accountSignature.value = ''
	accountAvatarUrl.value = ''
	accountProfileUrl.value = ''
	accountFollowerCount.value = null
	accountFollowingCount.value = null
	accountAwemeCount.value = null
	accountTotalFavorited.value = null
	accountQueried.value = false
}

const refreshFavorites = async () => {
  favoritesLoading.value = true
  favoritesError.value = ''
	try {
		const [usersRes, awemesRes, userTagsRes, awemeTagsRes] = await Promise.all([
      douyinApi.listDouyinFavoriteUsers(),
      douyinApi.listDouyinFavoriteAwemes(),
      douyinApi.listDouyinFavoriteUserTags(),
      douyinApi.listDouyinFavoriteAwemeTags()
    ])

    const normalizeTagIds = (raw: any) =>
      Array.isArray(raw)
        ? (raw as any[])
            .map((v) => Number(v))
            .filter((n) => Number.isFinite(n) && n > 0)
        : ([] as number[])

		favoriteUsers.value = Array.isArray(usersRes?.items)
      ? (usersRes.items as any[]).map((u) => ({ ...(u as any), tagIds: normalizeTagIds((u as any).tagIds) })) as DouyinFavoriteUser[]
      : []
		favoriteAwemes.value = Array.isArray(awemesRes?.items)
      ? (awemesRes.items as any[]).map((it) => ({ ...(it as any), tagIds: normalizeTagIds((it as any).tagIds) })) as DouyinFavoriteAweme[]
      : []

    favoriteUserTags.value = Array.isArray(userTagsRes?.items) ? (userTagsRes.items as DouyinFavoriteTag[]) : []
    favoriteAwemeTags.value = Array.isArray(awemeTagsRes?.items) ? (awemeTagsRes.items as DouyinFavoriteTag[]) : []

    const userTagIDSet = new Set(favoriteUserTags.value.map((t) => Number(t.id)).filter((n) => Number.isFinite(n) && n > 0))
    if (favoriteUserTagFilter.value !== null && favoriteUserTagFilter.value !== -1 && !userTagIDSet.has(favoriteUserTagFilter.value)) {
      favoriteUserTagFilter.value = null
    }
    const awemeTagIDSet = new Set(favoriteAwemeTags.value.map((t) => Number(t.id)).filter((n) => Number.isFinite(n) && n > 0))
    if (favoriteAwemeTagFilter.value !== null && favoriteAwemeTagFilter.value !== -1 && !awemeTagIDSet.has(favoriteAwemeTagFilter.value)) {
      favoriteAwemeTagFilter.value = null
    }

    const userIDSet = new Set(favoriteUsers.value.map((u) => String(u.secUserId || '').trim()).filter(Boolean))
    for (const id of Array.from(selectedFavoriteUserIds)) {
      if (!userIDSet.has(String(id || '').trim())) {
        selectedFavoriteUserIds.delete(id)
      }
    }
    const awemeIDSet = new Set(favoriteAwemes.value.map((it) => String(it.awemeId || '').trim()).filter(Boolean))
    for (const id of Array.from(selectedFavoriteAwemeIds)) {
      if (!awemeIDSet.has(String(id || '').trim())) {
        selectedFavoriteAwemeIds.delete(id)
      }
    }

		if (favoriteUserDetailOpen.value && String(favoriteUserDetailId.value || '').trim()) {
			const id = String(favoriteUserDetailId.value || '').trim()
			if (!favoriteUsers.value.some((u) => String(u.secUserId || '').trim() === id)) {
				closeFavoriteUserDetail()
			}
		}
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
        favoriteUserTags.value = []
        favoriteAwemeTags.value = []
        favoriteUserTagFilter.value = null
        favoriteAwemeTagFilter.value = null
        favoritesSelectionMode.value = false
        selectedFavoriteUserIds.clear()
        selectedFavoriteAwemeIds.clear()
        tagManagerOpen.value = false
        tagManagerKind.value = 'users'
        tagManagerNameInput.value = ''
        tagManagerSaving.value = false
        tagManagerError.value = ''
        editingTagId.value = null
        editingTagName.value = ''
        confirmDeleteTagOpen.value = false
        confirmDeleteTag.value = null
        tagSheetOpen.value = false
        tagSheetKind.value = 'users'
        tagSheetMode.value = 'single'
        tagSheetTargetIds.value = []
        tagSheetSelectedTagIds.clear()
        tagSheetApplying.value = false
        tagSheetError.value = ''
	      favoriteUserDetailOpen.value = false
	      favoriteUserDetailId.value = ''
	      favoriteUserDetailLoading.value = false
	      favoriteUserAvatarError.clear()
	      void refreshFavorites()

      // 优先使用调用方传入的预填内容；否则按设置尝试读取剪贴板
      const hasInput = () => !!String(inputText.value || '').trim() || !!String(accountInput.value || '').trim()
      if (!hasInput() && douyinStore.draftInput) {
        applyInputText(douyinStore.draftInput)
      } else if (!hasInput() && autoClipboard.value && typeof navigator.clipboard?.readText === 'function') {
        try {
          const txt = await navigator.clipboard.readText()
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
	favoriteUserDetailOpen.value = false
	favoriteUserDetailId.value = ''
	favoriteUserDetailLoading.value = false
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
  if (favoriteUserDetailOpen.value) {
    closeFavoriteUserDetail()
  }
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
		accountDisplayName.value = ''
		accountSignature.value = ''
		accountAvatarUrl.value = ''
		accountProfileUrl.value = ''
		accountFollowerCount.value = null
		accountFollowingCount.value = null
		accountAwemeCount.value = null
		accountTotalFavorited.value = null
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
		accountDisplayName.value = String(res?.displayName || accountDisplayName.value || '').trim()
		accountSignature.value = String(res?.signature || accountSignature.value || '').trim()
		accountAvatarUrl.value = String(res?.avatarUrl || accountAvatarUrl.value || '').trim()
		accountProfileUrl.value = String(res?.profileUrl || accountProfileUrl.value || '').trim()
		if (typeof res?.followerCount === 'number' && res.followerCount > 0) accountFollowerCount.value = res.followerCount
		if (typeof res?.followingCount === 'number' && res.followingCount > 0) accountFollowingCount.value = res.followingCount
		if (typeof res?.awemeCount === 'number' && res.awemeCount > 0) accountAwemeCount.value = res.awemeCount
		if (typeof res?.totalFavorited === 'number' && res.totalFavorited > 0) accountTotalFavorited.value = res.totalFavorited
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
		accountDisplayName.value = String(res?.displayName || accountDisplayName.value || '').trim()
		accountSignature.value = String(res?.signature || accountSignature.value || '').trim()
		accountAvatarUrl.value = String(res?.avatarUrl || accountAvatarUrl.value || '').trim()
		accountProfileUrl.value = String(res?.profileUrl || accountProfileUrl.value || '').trim()
		if (typeof res?.followerCount === 'number' && res.followerCount > 0) accountFollowerCount.value = res.followerCount
		if (typeof res?.followingCount === 'number' && res.followingCount > 0) accountFollowingCount.value = res.followingCount
		if (typeof res?.awemeCount === 'number' && res.awemeCount > 0) accountAwemeCount.value = res.awemeCount
		if (typeof res?.totalFavorited === 'number' && res.totalFavorited > 0) accountTotalFavorited.value = res.totalFavorited
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
	          displayName: accountDisplayName.value || undefined,
	          avatarUrl: accountAvatarUrl.value || undefined,
	          profileUrl: accountProfileUrl.value || undefined,
	          lastParsedCount: accountItems.value.length,
	          lastParsedRaw: buildFavoriteUserLastParsedRaw({
	            signature: accountSignature.value,
	            followerCount: accountFollowerCount.value,
	            followingCount: accountFollowingCount.value,
	            awemeCount: accountAwemeCount.value,
	            totalFavorited: accountTotalFavorited.value
	          })
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

const buildFavoriteUserLastParsedRaw = (meta: {
  signature?: string
  followerCount?: number | null
  followingCount?: number | null
  awemeCount?: number | null
  totalFavorited?: number | null
}) => {
  const out: Record<string, any> = {}
  const signature = String(meta?.signature || '').trim()
  if (signature) out.signature = signature

  if (typeof meta?.followerCount === 'number' && meta.followerCount > 0) out.followerCount = meta.followerCount
  if (typeof meta?.followingCount === 'number' && meta.followingCount > 0) out.followingCount = meta.followingCount
  if (typeof meta?.awemeCount === 'number' && meta.awemeCount > 0) out.awemeCount = meta.awemeCount
  if (typeof meta?.totalFavorited === 'number' && meta.totalFavorited > 0) out.totalFavorited = meta.totalFavorited

  return Object.keys(out).length > 0 ? out : undefined
}

const formatDouyinCount = (value?: number) => {
  const n = Number(value || 0)
  if (!Number.isFinite(n) || n <= 0) return '-'
  if (n >= 100000000) return `${(n / 100000000).toFixed(1).replace(/\\.0$/, '')}亿`
  if (n >= 10000) return `${(n / 10000).toFixed(1).replace(/\\.0$/, '')}万`
  return String(Math.round(n))
}

const setCurrentFavoriteTagFilterValue = (value: number | null) => {
  if (favoritesTab.value === 'users') {
    favoriteUserTagFilter.value = value
  } else {
    favoriteAwemeTagFilter.value = value
  }
  if (favoritesSelectionMode.value) {
    selectedFavoriteUserIds.clear()
    selectedFavoriteAwemeIds.clear()
  }
}

const toggleFavoritesSelectionMode = () => {
  favoritesSelectionMode.value = !favoritesSelectionMode.value
  selectedFavoriteUserIds.clear()
  selectedFavoriteAwemeIds.clear()
}

const setFavoritesTab = (tab: 'users' | 'awemes') => {
  if (favoritesLoading.value) return
  favoritesTab.value = tab
  if (favoritesSelectionMode.value) {
    favoritesSelectionMode.value = false
    selectedFavoriteUserIds.clear()
    selectedFavoriteAwemeIds.clear()
  }
}

const toggleSelectFavoriteUser = (secUserId: string) => {
  const id = String(secUserId || '').trim()
  if (!id) return
  if (selectedFavoriteUserIds.has(id)) {
    selectedFavoriteUserIds.delete(id)
  } else {
    selectedFavoriteUserIds.add(id)
  }
}

const toggleSelectFavoriteAweme = (awemeId: string) => {
  const id = String(awemeId || '').trim()
  if (!id) return
  if (selectedFavoriteAwemeIds.has(id)) {
    selectedFavoriteAwemeIds.delete(id)
  } else {
    selectedFavoriteAwemeIds.add(id)
  }
}

const openTagManager = (kind: 'users' | 'awemes') => {
  if (uiDisabled.value) return
  tagManagerKind.value = kind
  tagManagerOpen.value = true
  tagManagerNameInput.value = ''
  tagManagerError.value = ''
  editingTagId.value = null
  editingTagName.value = ''
  confirmDeleteTagOpen.value = false
  confirmDeleteTag.value = null
}

const closeTagManager = () => {
  tagManagerOpen.value = false
  tagManagerNameInput.value = ''
  tagManagerSaving.value = false
  tagManagerError.value = ''
  editingTagId.value = null
  editingTagName.value = ''
  confirmDeleteTagOpen.value = false
  confirmDeleteTag.value = null
}

const saveTagManagerOrder = async () => {
  if (tagManagerSaving.value) return
  const tagIds = tagManagerTags.value
    .map((t) => Number(t?.id))
    .filter((n) => Number.isFinite(n) && n > 0)
  if (tagIds.length === 0) return

  tagManagerSaving.value = true
  tagManagerError.value = ''
  try {
    if (tagManagerKind.value === 'users') {
      await douyinApi.reorderDouyinFavoriteUserTags({ tagIds })
    } else {
      await douyinApi.reorderDouyinFavoriteAwemeTags({ tagIds })
    }
    show('已更新顺序')
  } catch (e: any) {
    console.error('保存标签顺序失败:', e)
    tagManagerError.value = e?.response?.data?.error || e?.message || '保存失败'
  } finally {
    tagManagerSaving.value = false
  }
}

const handleCreateTag = async () => {
  const name = String(tagManagerNameInput.value || '').trim()
  if (!name || tagManagerSaving.value) return

  tagManagerSaving.value = true
  tagManagerError.value = ''
  try {
    if (tagManagerKind.value === 'users') {
      await douyinApi.addDouyinFavoriteUserTag({ name })
    } else {
      await douyinApi.addDouyinFavoriteAwemeTag({ name })
    }
    tagManagerNameInput.value = ''
    show('已创建标签')
    await refreshFavorites()
  } catch (e: any) {
    console.error('创建标签失败:', e)
    tagManagerError.value = e?.response?.data?.error || e?.message || '创建失败'
  } finally {
    tagManagerSaving.value = false
  }
}

const startEditTag = (t: DouyinFavoriteTag) => {
  editingTagId.value = Number(t?.id || 0) || null
  editingTagName.value = String(t?.name || '').trim()
}

const cancelEditTag = () => {
  editingTagId.value = null
  editingTagName.value = ''
}

const saveEditTag = async () => {
  const id = Number(editingTagId.value || 0)
  const name = String(editingTagName.value || '').trim()
  if (!id || !name || tagManagerSaving.value) return

  tagManagerSaving.value = true
  tagManagerError.value = ''
  try {
    if (tagManagerKind.value === 'users') {
      await douyinApi.updateDouyinFavoriteUserTag({ id, name })
    } else {
      await douyinApi.updateDouyinFavoriteAwemeTag({ id, name })
    }
    cancelEditTag()
    show('已更新标签')
    await refreshFavorites()
  } catch (e: any) {
    console.error('更新标签失败:', e)
    tagManagerError.value = e?.response?.data?.error || e?.message || '更新失败'
  } finally {
    tagManagerSaving.value = false
  }
}

const askDeleteTag = (t: DouyinFavoriteTag) => {
  confirmDeleteTag.value = t
  confirmDeleteTagOpen.value = true
}

const cancelDeleteTag = () => {
  confirmDeleteTagOpen.value = false
  confirmDeleteTag.value = null
}

const confirmDeleteTagNow = async () => {
  const t = confirmDeleteTag.value
  const id = Number(t?.id || 0)
  if (!id || tagManagerSaving.value) return

  tagManagerSaving.value = true
  tagManagerError.value = ''
  try {
    if (tagManagerKind.value === 'users') {
      await douyinApi.removeDouyinFavoriteUserTag({ id })
    } else {
      await douyinApi.removeDouyinFavoriteAwemeTag({ id })
    }
    cancelDeleteTag()
    show('已删除标签')
    await refreshFavorites()
  } catch (e: any) {
    console.error('删除标签失败:', e)
    tagManagerError.value = e?.response?.data?.error || e?.message || '删除失败'
  } finally {
    tagManagerSaving.value = false
  }
}

const openTagSheet = (opts: { kind: 'users' | 'awemes'; mode: 'single' | 'batch'; targetIds: string[]; presetTagIds?: number[] }) => {
  if (uiDisabled.value) return
  tagSheetKind.value = opts.kind
  tagSheetMode.value = opts.mode
  tagSheetTargetIds.value = Array.isArray(opts.targetIds) ? opts.targetIds.map((v) => String(v || '').trim()).filter(Boolean) : []
  tagSheetSelectedTagIds.clear()
  for (const id of Array.isArray(opts.presetTagIds) ? opts.presetTagIds : []) {
    const n = Number(id)
    if (Number.isFinite(n) && n > 0) {
      tagSheetSelectedTagIds.add(n)
    }
  }
  tagSheetError.value = ''
  tagSheetApplying.value = false
  tagSheetOpen.value = true
}

const closeTagSheet = () => {
  tagSheetOpen.value = false
  tagSheetTargetIds.value = []
  tagSheetSelectedTagIds.clear()
  tagSheetApplying.value = false
  tagSheetError.value = ''
}

const toggleTagInSheet = (id: number) => {
  const n = Number(id)
  if (!Number.isFinite(n) || n <= 0) return
  if (tagSheetSelectedTagIds.has(n)) {
    tagSheetSelectedTagIds.delete(n)
  } else {
    tagSheetSelectedTagIds.add(n)
  }
}

const applyTagSheet = async () => {
  if (tagSheetApplying.value) return

  const kind = tagSheetKind.value
  const targetIds = Array.isArray(tagSheetTargetIds.value) ? tagSheetTargetIds.value.map((v) => String(v || '').trim()).filter(Boolean) : []
  const tagIds = Array.from(tagSheetSelectedTagIds).map((v) => Number(v)).filter((n) => Number.isFinite(n) && n > 0)
  const mode = tagSheetMode.value === 'batch' ? 'add' : 'set'

  if (targetIds.length === 0) {
    closeTagSheet()
    return
  }
  if (tagSheetMode.value === 'batch' && tagIds.length === 0) {
    show('请选择标签')
    return
  }

  tagSheetApplying.value = true
  tagSheetError.value = ''
  try {
    if (kind === 'users') {
      await douyinApi.applyDouyinFavoriteUserTags({ secUserIds: targetIds, tagIds, mode })
    } else {
      await douyinApi.applyDouyinFavoriteAwemeTags({ awemeIds: targetIds, tagIds, mode })
    }
    show(tagSheetMode.value === 'batch' ? '已批量添加标签' : '已更新标签')
    closeTagSheet()
    await refreshFavorites()
  } catch (e: any) {
    console.error('更新标签失败:', e)
    tagSheetError.value = e?.response?.data?.error || e?.message || '保存失败'
    show(tagSheetError.value)
  } finally {
    tagSheetApplying.value = false
  }
}

const openBatchTagSheet = () => {
  if (favoritesTab.value === 'users') {
    openTagSheet({ kind: 'users', mode: 'batch', targetIds: Array.from(selectedFavoriteUserIds) })
    return
  }
  openTagSheet({ kind: 'awemes', mode: 'batch', targetIds: Array.from(selectedFavoriteAwemeIds) })
}

	const openFavoriteUserDetail = (u: DouyinFavoriteUser) => {
	  if (uiDisabled.value) return
	  const id = String(u?.secUserId || '').trim()
	  if (!id) return
	  favoriteUserDetailId.value = id
	  favoriteUserDetailOpen.value = true
	  resetFavoriteUserWorks()
	  void loadFavoriteUserWorks()
	}

	const closeFavoriteUserDetail = () => {
	  favoriteUserDetailOpen.value = false
	  favoriteUserDetailId.value = ''
	  favoriteUserDetailLoading.value = false
	  resetFavoriteUserWorks()
	}

	const resetFavoriteUserWorks = (opts: { keepPullingLatest?: boolean } = {}) => {
	  favoriteUserWorksLoading.value = false
	  if (!opts.keepPullingLatest) favoriteUserWorksPullingLatest.value = false
	  favoriteUserWorksError.value = ''
	  favoriteUserWorks.value = []
	  favoriteUserWorksCursor.value = 0
	  favoriteUserWorksHasMore.value = false
	}

	const loadFavoriteUserWorks = async (opts: { append?: boolean } = {}) => {
	  if (favoriteUserWorksLoading.value) return
	  const secUserId = String(favoriteUserDetailId.value || '').trim()
	  if (!secUserId) return

	  favoriteUserWorksLoading.value = true
	  favoriteUserWorksError.value = ''
	  try {
	    const cursor = opts.append ? Number(favoriteUserWorksCursor.value || 0) : 0
	    const res = await douyinApi.listDouyinFavoriteUserAwemes({ secUserId, cursor, count: 20 })
	    if (!res || !Array.isArray(res?.items)) {
	      throw new Error(res?.error || '获取失败')
	    }

	    const existing = new Set((opts.append ? favoriteUserWorks.value : []).map((i) => String(i.detailId || '').trim()).filter(Boolean))
	    const next = opts.append ? favoriteUserWorks.value.slice() : []
	    for (const it of res.items || []) {
	      const id = String(it?.detailId || '').trim()
	      if (!id || existing.has(id)) continue
	      existing.add(id)
	      next.push({
	        detailId: id,
	        type: it?.type,
	        desc: String(it?.desc || '').trim(),
	        coverUrl: String(it?.coverUrl || '').trim(),
	        coverDownloadUrl: String(it?.coverDownloadUrl || '').trim(),
	        key: String(it?.key || '').trim(),
	        items: Array.isArray(it?.items) ? it.items : []
	      })
	    }
	    favoriteUserWorks.value = next
	    favoriteUserWorksCursor.value = Number(res?.cursor || 0)
	    favoriteUserWorksHasMore.value = !!res?.hasMore
	  } catch (e: any) {
	    console.error('获取收藏用户作品失败:', e)
	    favoriteUserWorksError.value = e?.response?.data?.error || e?.message || '获取失败'
	  } finally {
	    favoriteUserWorksLoading.value = false
	  }
	}

	const pullLatestFavoriteUserWorks = async () => {
	  if (favoriteUserWorksPullingLatest.value) return
	  const secUserId = String(favoriteUserDetailId.value || '').trim()
	  if (!secUserId) return

	  favoriteUserWorksPullingLatest.value = true
	  try {
	    const res = await douyinApi.pullLatestDouyinFavoriteUserAwemes({
	      secUserId,
	      cookie: String(cookie.value || '').trim() || undefined,
	      count: 50
	    })
	    const added = Number(res?.added || 0)
	    show(`已获取 ${Number.isFinite(added) && added > 0 ? added : 0} 个新作品`)
	    resetFavoriteUserWorks({ keepPullingLatest: true })
	    await loadFavoriteUserWorks()
	  } catch (e: any) {
	    console.error('拉取最新作品失败:', e)
	    show(e?.response?.data?.error || e?.message || '拉取失败')
	  } finally {
	    favoriteUserWorksPullingLatest.value = false
	  }
	}

	const handleFavoriteUserDetailScroll = (e: Event) => {
	  if (favoriteUserWorksLoading.value) return
	  if (!favoriteUserWorksHasMore.value) return
	  const el = e.target as HTMLElement | null
	  if (!el) return
	  const remaining = el.scrollHeight - el.scrollTop - el.clientHeight
	  if (remaining <= 240) {
	    void loadFavoriteUserWorks({ append: true })
	  }
	}

	const openPreviewFromFavoriteUserWork = (item: DouyinAccountItem) => {
	  resetDetailStates()
	  previewContextKey.value = String(item.key || '')
	  previewContextItems.value = item.items || []

	  const first = (item.items || []).slice().sort((a, b) => Number(a.index) - Number(b.index))[0]
	  if (!first) return

	  previewIndex.value = Number(first.index) || 0
	  previewType.value = first.type
	  previewUrl.value = String(first.downloadUrl || first.url || '').trim()

	  const fullList = buildAccountPreviewMediaList(favoriteUserWorks.value)
	  previewMediaList.value =
	    fullList.length > 0 ? fullList : buildPreviewMediaList(item.items || [], { key: String(item.key || '').trim(), title: getAccountItemTitle(item) })
	  showPreview.value = true
	}

	const openFavoriteUserWork = async (item: DouyinAccountItem) => {
	  const id = String(item?.detailId || '').trim()
	  if (!id) return

	  const key = String(item?.key || '').trim()
	  const medias = Array.isArray(item?.items) ? item.items : []
	  if (key && medias.length > 0) {
	    openPreviewFromFavoriteUserWork(item)
	    return
	  }

	  if (favoriteUserWorkLoading.has(id)) return
	  favoriteUserWorkLoading.add(id)
	  try {
	    const res = await douyinApi.getDouyinDetail({
	      input: id,
	      cookie: String(cookie.value || '').trim()
	    })

	    if (res?.key && Array.isArray(res?.items)) {
	      item.key = res.key
	      item.items = res.items
	      if (res?.coverUrl) item.coverUrl = String(res.coverUrl || '').trim()
	      if (res?.key && String(res?.coverUrl || '').trim()) {
	        item.coverDownloadUrl = `/api/douyin/cover?key=${encodeURIComponent(String(res.key || '').trim())}`
	      }
	      openPreviewFromFavoriteUserWork(item)
	    } else {
	      throw new Error(res?.error || '解析未返回有效数据')
	    }
	  } catch (e: any) {
	    console.error('获取作品详情失败:', e)
	    show(e?.message || e?.response?.data?.error || '获取详情失败')
	  } finally {
	    favoriteUserWorkLoading.delete(id)
	  }
	}

	const markFavoriteUserAvatarError = (secUserId: string) => {
	  const id = String(secUserId || '').trim()
	  if (!id) return
	  favoriteUserAvatarError.add(id)
	}

const copyText = async (value: string, okMsg = '已复制') => {
  const v = String(value || '').trim()
  if (!v) return
  const ok = await copyToClipboard(v)
  show(ok ? okMsg : '复制失败')
}

	const refreshSelectedFavoriteUser = async () => {
	  const u = selectedFavoriteUser.value
	  if (!u) return
	  const secUserId = String(u.secUserId || '').trim()
	  if (!secUserId) return

  favoriteUserDetailLoading.value = true
  try {
    const res = (await douyinApi.getDouyinAccount({
      input: secUserId,
      cookie: String(cookie.value || '').trim(),
      tab: 'post',
      cursor: 0,
      count: 1
    })) as DouyinAccountResponse

    const updated = await douyinApi.addDouyinFavoriteUser({
      secUserId,
      sourceInput: String(u.sourceInput || '').trim() || undefined,
      displayName: String(res?.displayName || '').trim() || undefined,
      avatarUrl: String(res?.avatarUrl || '').trim() || undefined,
      profileUrl: String(res?.profileUrl || '').trim() || undefined,
      lastParsedRaw: buildFavoriteUserLastParsedRaw({
        signature: String(res?.signature || '').trim(),
        followerCount: typeof res?.followerCount === 'number' ? res.followerCount : null,
        followingCount: typeof res?.followingCount === 'number' ? res.followingCount : null,
        awemeCount: typeof res?.awemeCount === 'number' ? res.awemeCount : null,
        totalFavorited: typeof res?.totalFavorited === 'number' ? res.totalFavorited : null
      })
    } as any)

    if (updated?.secUserId) {
      const normalized = { ...(updated as any), tagIds: Array.isArray((updated as any).tagIds) ? (updated as any).tagIds : [] } as DouyinFavoriteUser
      favoriteUsers.value = [normalized, ...favoriteUsers.value.filter((it) => it.secUserId !== normalized.secUserId)]
      favoriteUserAvatarError.delete(normalized.secUserId)
      show('已更新用户信息')
    } else {
      throw new Error((updated as any)?.error || '更新失败')
    }
  } catch (e: any) {
    console.error('刷新抖音用户信息失败:', e)
    show(e?.response?.data?.error || e?.message || '更新失败')
  } finally {
    favoriteUserDetailLoading.value = false
	  }
	}

	const syncFavoriteUserWorksFromAccount = async (secUserId: string) => {
	  const id = String(secUserId || '').trim()
	  if (!id) return

	  if (String(accountSecUserId.value || '').trim() !== id) return

	  const items = accountItems.value || []
	  if (items.length === 0) return

	  const upserts = items
	    .map((it) => {
	      const awemeId = String(it?.detailId || '').trim()
	      if (!awemeId) return null
	      const downloads = Array.isArray(it?.items)
	        ? it.items.map((m) => String((m as any)?.url || '').trim()).filter(Boolean)
	        : []
	      return {
	        awemeId,
	        type: String(it?.type || '').trim() || undefined,
	        desc: String(it?.desc || '').trim() || undefined,
	        coverUrl: String(it?.coverUrl || '').trim() || undefined,
	        downloads: downloads.length > 0 ? downloads : undefined
	      }
	    })
	    .filter(Boolean) as { awemeId: string; type?: string; desc?: string; coverUrl?: string; downloads?: string[] }[]

	  if (upserts.length === 0) return

	  try {
	    await douyinApi.upsertDouyinFavoriteUserAwemes({ secUserId: id, items: upserts })
	  } catch (e) {
	    console.warn('同步收藏用户作品入库失败:', e)
	  }
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
	      const normalized = { ...(res as any), tagIds: Array.isArray((res as any).tagIds) ? (res as any).tagIds : [] } as DouyinFavoriteUser
	      favoriteUsers.value = [normalized, ...favoriteUsers.value.filter((u) => u.secUserId !== normalized.secUserId)]
	      show('已收藏用户')
	      await syncFavoriteUserWorksFromAccount(secUserId)
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
    selectedFavoriteUserIds.delete(id)
		if (favoriteUserDetailOpen.value && String(favoriteUserDetailId.value || '').trim() === id) {
			closeFavoriteUserDetail()
		}
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
		displayName: accountDisplayName.value || undefined,
		avatarUrl: accountAvatarUrl.value || undefined,
		profileUrl: accountProfileUrl.value || undefined,
		lastParsedCount: accountItems.value.length,
		lastParsedRaw: buildFavoriteUserLastParsedRaw({
			signature: accountSignature.value,
			followerCount: accountFollowerCount.value,
			followingCount: accountFollowingCount.value,
			awemeCount: accountAwemeCount.value,
			totalFavorited: accountTotalFavorited.value
		})
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
      const normalized = { ...(res as any), tagIds: Array.isArray((res as any).tagIds) ? (res as any).tagIds : [] } as DouyinFavoriteAweme
      favoriteAwemes.value = [normalized, ...favoriteAwemes.value.filter((it) => it.awemeId !== normalized.awemeId)]
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
    selectedFavoriteAwemeIds.delete(id)
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

  if (favoriteUserDetailOpen.value && String(favoriteUserDetailId.value || '').trim() === secUserId) {
    closeFavoriteUserDetail()
  }
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

  const sorted = (items || []).slice().sort((a, b) => Number(a.index) - Number(b.index))
  const images = sorted.filter((it) => it.type === 'image')
  const videos = sorted.filter((it) => it.type === 'video')
  const liveVideoIndexByImageIndex = new Map<number, number>()
  if (images.length > 0 && videos.length > 0) {
    if (videos.length === 1) {
      const onlyVideo = videos[0]
      if (onlyVideo) {
      for (const img of images) {
          liveVideoIndexByImageIndex.set(Number(img.index), Number(onlyVideo.index))
      }
      }
    } else {
      for (const img of images) {
        const imgIdx = Number(img.index)
        const nextVid = videos.find((v) => Number(v.index) > imgIdx)
        if (nextVid) liveVideoIndexByImageIndex.set(imgIdx, Number(nextVid.index))
      }
    }
  }

  return items
    .slice()
    .sort((a, b) => Number(a.index) - Number(b.index))
    .map((it) => ({
      url: it.downloadUrl,
      type: it.type,
      downloadUrl: it.downloadUrl,
      originalFilename: it.originalFilename,
      title: title || undefined,
      context: key
        ? {
            provider: 'douyin',
            key,
            index: Number(it.index),
            liveVideoIndex: it.type === 'image' ? liveVideoIndexByImageIndex.get(Number(it.index)) : undefined
          }
        : undefined
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

	  const hasImage = detail.value.items.some((i) => i.type === 'image')
	  const hasVideo = detail.value.items.some((i) => i.type === 'video')
	  const mixed = hasImage && hasVideo

	  if (mixed) {
	    previewMediaList.value = buildPreviewMediaList(detail.value.items, { key: detail.value.key, title: detail.value.title })
	  } else if (item.type === 'image') {
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
	        const found =
	          accountItems.value.find((i) => String(i.key || '').trim() === key) ||
	          favoriteUserWorks.value.find((i) => String(i.key || '').trim() === key)
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
  const key = String(previewContextKey.value || detail.value?.key || '').trim()
  if (!key) return true
  const st = previewItemState.value?.status
  return st === 'importing' || st === 'imported' || st === 'exists'
})
const previewUploadText = computed(() => {
  const st = previewItemState.value?.status
  if (st === 'importing') return '导入中…'
  if (st === 'imported') return '已导入'
  if (st === 'exists') return '已存在（去重）'
  return `导入此${previewType.value === 'image' ? '图片' : (previewType.value === 'video' ? '视频' : '文件')}`
})

const importIndex = async (idx: number) => {
  const key = String(previewContextKey.value || detail.value?.key || '').trim()
  if (!key) return { ok: false, dedup: false, error: '解析信息缺失' }

  const itemId = buildItemId(key, idx)
  const current = itemStateById[itemId]?.status
  if (current === 'imported' || current === 'exists') {
    return { ok: true, dedup: current === 'exists' }
  }
  if (current === 'importing') {
    return { ok: false, dedup: false, error: '导入进行中' }
  }

  itemStateById[itemId] = { status: 'importing' }

  const userId = userStore.currentUser?.id || 'pre_identity'
  const nickname = userStore.currentUser?.name || 'pre_identity'
  const cookieData = generateCookie(userId, nickname)
  const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
  const userAgent = navigator.userAgent

  try {
    const res = await douyinApi.importDouyinMedia({
      userid: userId,
      key,
      index: idx,
      cookieData,
      referer,
      userAgent
    })

    if (res?.state === 'OK' && res.localPath) {
      const dedup = !!res.dedup
      itemStateById[itemId] = { status: dedup ? 'exists' : 'imported' }
      show(dedup ? '已存在（去重复用）' : '已导入到本地（去“所有图片”里手动上传后发送）')
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
