import { ref, type Ref } from 'vue'
import { useSwipe, type UseSwipeOptions } from '@vueuse/core'

export interface SwipeActionOptions {
  onSwipeEnd?: (direction: 'left' | 'right' | 'up' | 'down') => void
  /**
   * 手势结束时必触发（无论是否命中 threshold）。
   * - deltaX/deltaY: 结束时的最终位移（带符号）
   * - isTriggered: 是否命中阈值并触发了 onSwipeEnd
   */
  onSwipeFinish?: (deltaX: number, deltaY: number, isTriggered: boolean) => void
  threshold?: number
  passive?: boolean
  onSwipeProgress?: (deltaX: number, deltaY: number) => void
}

/**
 * 封装带方向判定和进度的 Swipe 交互
 * @param target 目标元素
 * @param options 配置项
 */
export function useSwipeAction(target: Ref<HTMLElement | null | undefined>, options: SwipeActionOptions = {}) {
  const { threshold = 50, onSwipeEnd, onSwipeFinish, onSwipeProgress, passive = true } = options

  const { lengthX, lengthY, direction, isSwiping, coordsStart, coordsEnd } = useSwipe(target, {
    threshold: 0, // 设置为0以便我们在 onSwipe 手动处理进度
    passive: passive, 
    onSwipe: () => {
      if (onSwipeProgress) {
        // useSwipe 的 lengthX 向左滑是正数，向右滑是负数 (文档说 abs value, 实际行为需验证)
        // 实际上 VueUse 文档：lengthX is always positive. direction indicates the direction.
        // 但为了通用 delta，我们需要带符号的值。
        // VueUse 的 lengthX/Y 是绝对值，direction 是方向。
        
        // 计算带符号的 delta
        const deltaX = coordsEnd.x - coordsStart.x
        const deltaY = coordsEnd.y - coordsStart.y
        onSwipeProgress(deltaX, deltaY)
      }
    },
    onSwipeEnd: (_e, _direction) => {
      // VueUse 的 direction 在极短滑动时可能不准，结合位移判定。
      const deltaX = coordsEnd.x - coordsStart.x
      const deltaY = coordsEnd.y - coordsStart.y

      let isTriggered = false
      let endDirection: 'left' | 'right' | 'up' | 'down' | null = null

      if (Math.abs(deltaX) > Math.abs(deltaY) && Math.abs(deltaX) > threshold) {
        isTriggered = true
        endDirection = deltaX > 0 ? 'right' : 'left'
      } else if (Math.abs(deltaY) > Math.abs(deltaX) && Math.abs(deltaY) > threshold) {
        isTriggered = true
        endDirection = deltaY > 0 ? 'down' : 'up'
      }

      // 先触发阈值命中的业务回调，再做必触发的收尾回调，避免调用方出现动画时序冲突。
      if (isTriggered && endDirection && onSwipeEnd) {
        onSwipeEnd(endDirection)
      }
      if (onSwipeFinish) {
        onSwipeFinish(deltaX, deltaY, isTriggered)
      }
    }
  })

  return {
    isSwiping,
    lengthX,
    lengthY,
    direction,
    coordsStart
  }
}
