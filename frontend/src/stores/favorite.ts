import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Favorite } from '@/types'
import * as favoriteApi from '@/api/favorite'

export const useFavoriteStore = defineStore('favorite', () => {
  const allFavorites = ref<Favorite[]>([])
  const loading = ref(false)

  // 按身份分组的收藏列表
  const groupedFavorites = computed(() => {
    const groups: Record<string, Favorite[]> = {}
    allFavorites.value.forEach(fav => {
      if (!groups[fav.identityId]) {
        groups[fav.identityId] = []
      }
      groups[fav.identityId]?.push(fav)
    })
    return groups
  })

  const loadAllFavorites = async () => {
    loading.value = true
    try {
      const res = await favoriteApi.listAllFavorites()
      if (res.code === 0 && res.data) {
        allFavorites.value = res.data
      }
    } catch (error) {
      console.error('加载全局收藏失败:', error)
    } finally {
      loading.value = false
    }
  }

  const addFavorite = async (identityId: string, targetUserId: string, targetUserName?: string) => {
    try {
      const res = await favoriteApi.addFavorite(identityId, targetUserId, targetUserName)
      if (res.code === 0) {
        await loadAllFavorites()
        return true
      }
    } catch (error) {
      console.error('添加收藏失败:', error)
    }
    return false
  }

  const removeFavorite = async (identityId: string, targetUserId: string) => {
    try {
      const res = await favoriteApi.removeFavorite(identityId, targetUserId)
      if (res.code === 0) {
        // Optimistic update
        allFavorites.value = allFavorites.value.filter(
          f => !(f.identityId === identityId && f.targetUserId === targetUserId)
        )
        return true
      }
    } catch (error) {
      console.error('移除收藏失败:', error)
    }
    return false
  }
  
  const removeFavoriteById = async (id: number) => {
      try {
          const res = await favoriteApi.removeFavoriteById(id)
          if (res.code === 0) {
              allFavorites.value = allFavorites.value.filter(f => f.id !== id)
              return true
          }
      } catch (error) {
          console.error('移除收藏失败:', error)
      }
      return false
  }

  const isFavorite = (identityId: string, targetUserId: string) => {
    return allFavorites.value.some(
      f => f.identityId === identityId && f.targetUserId === targetUserId
    )
  }

  return {
    allFavorites,
    groupedFavorites,
    loading,
    loadAllFavorites,
    addFavorite,
    removeFavorite,
    removeFavoriteById,
    isFavorite
  }
})
