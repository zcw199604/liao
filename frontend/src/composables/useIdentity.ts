import { useIdentityStore } from '@/stores/identity'
import { useUserStore } from '@/stores/user'
import { useRouter } from 'vue-router'
import { generateRandomIP } from '@/utils/id'
import { generateCookie } from '@/utils/cookie'
import { getColorClass } from '@/constants/colors'

export const useIdentity = () => {
  const identityStore = useIdentityStore()
  const userStore = useUserStore()
  const router = useRouter()

  const loadList = async () => {
    await identityStore.loadList()
  }

  const create = async (name: string, sex: string) => {
    const success = await identityStore.createIdentity({ name, sex })
    return success
  }

  const quickCreate = async () => {
    const randomName = `用户${Math.floor(Math.random() * 10000)}`
    const randomSex = Math.random() > 0.5 ? '男' : '女'

    return await create(randomName, randomSex)
  }

  const deleteIdentity = async (id: string) => {
    return await identityStore.deleteIdentity(id)
  }

  const select = async (identity: any) => {
    console.log('选择身份:', identity)

    const name = identity.name || 'User'
    const cookie = generateCookie(identity.id, name)

    const currentUser = {
      id: identity.id,
      name: name,
      nickname: name,
      sex: identity.sex,
      color: getColorClass(identity.id),
      created_at: identity.created_at || identity.createdAt || '',
      cookie,
      ip: generateRandomIP(),
      area: '未知'
    }

    console.log('设置当前用户:', currentUser)
    userStore.setCurrentUser(currentUser)

    await identityStore.selectIdentity(identity.id)

    router.push('/list')
  }

  return {
    loadList,
    create,
    quickCreate,
    deleteIdentity,
    select
  }
}
