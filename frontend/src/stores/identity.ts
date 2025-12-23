import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Identity } from '@/types'
import * as identityApi from '@/api/identity'

export const useIdentityStore = defineStore('identity', () => {
  const identityList = ref<Identity[]>([])
  const loading = ref(false)
  const showCreateForm = ref(false)
  const newIdentity = ref<Partial<Identity>>({
    name: '',
    sex: '男'
  })
  const deleteConfirmIdentity = ref<Identity | null>(null)

  const loadList = async () => {
    loading.value = true
    try {
      const res = await identityApi.getIdentityList()
      console.log('身份列表API响应:', res)
      if (res.code === 0 && res.data) {
        identityList.value = res.data
        console.log('身份列表已更新:', identityList.value)
      }
    } catch (error) {
      console.error('加载身份列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  const createIdentity = async (data: { name: string; sex: string }) => {
    const res = await identityApi.createIdentity(data)
    if (res.code === 0) {
      await loadList()
      return true
    }
    return false
  }

  const deleteIdentity = async (id: string) => {
    const res = await identityApi.deleteIdentity(id)
    if (res.code === 0) {
      await loadList()
      return true
    }
    return false
  }

  const selectIdentity = async (id: string) => {
    await identityApi.selectIdentity(id)
  }

  return {
    identityList,
    loading,
    showCreateForm,
    newIdentity,
    deleteConfirmIdentity,
    loadList,
    createIdentity,
    deleteIdentity,
    selectIdentity
  }
})
