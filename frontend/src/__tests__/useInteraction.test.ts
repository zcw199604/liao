import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

type SwipeDirection = 'left' | 'right' | 'up' | 'down'

type UseSwipeOptions = {
  onSwipe?: () => void
  onSwipeEnd?: (e: unknown, direction: unknown) => void
}

let lastUseSwipeOptions: UseSwipeOptions | null = null
let lastCoordsStart: { x: number; y: number } | null = null
let lastCoordsEnd: { x: number; y: number } | null = null

vi.mock('@vueuse/core', () => ({
  useSwipe: (_target: unknown, options: UseSwipeOptions) => {
    lastUseSwipeOptions = options
    lastCoordsStart = { x: 0, y: 0 }
    lastCoordsEnd = { x: 0, y: 0 }

    // useSwipeAction 只会读取 coordsStart/coordsEnd；其余字段在本测试中不关注
    return {
      lengthX: { value: 0 },
      lengthY: { value: 0 },
      direction: { value: '' },
      isSwiping: { value: false },
      coordsStart: lastCoordsStart,
      coordsEnd: lastCoordsEnd
    }
  }
}))

import { useSwipeAction } from '@/composables/useInteraction'

function setDelta(deltaX: number, deltaY: number) {
  if (!lastCoordsStart || !lastCoordsEnd) throw new Error('mock useSwipe was not initialized')
  lastCoordsStart.x = 0
  lastCoordsStart.y = 0
  lastCoordsEnd.x = deltaX
  lastCoordsEnd.y = deltaY
}

function triggerSwipeEnd() {
  if (!lastUseSwipeOptions?.onSwipeEnd) throw new Error('mock onSwipeEnd was not initialized')
  lastUseSwipeOptions.onSwipeEnd({}, '')
}

describe('composables/useInteraction - useSwipeAction', () => {
  it('calls onSwipeFinish when below threshold and does not call onSwipeEnd', () => {
    const calls: Array<{ kind: 'end' | 'finish'; dir?: SwipeDirection; dx?: number; dy?: number; triggered?: boolean }> = []

    useSwipeAction(ref<HTMLElement | null>(null), {
      threshold: 50,
      onSwipeEnd: (dir) => calls.push({ kind: 'end', dir }),
      onSwipeFinish: (dx, dy, isTriggered) => calls.push({ kind: 'finish', dx, dy, triggered: isTriggered })
    })

    setDelta(30, 0)
    triggerSwipeEnd()

    expect(calls).toEqual([{ kind: 'finish', dx: 30, dy: 0, triggered: false }])
  })

  it('calls onSwipeEnd then onSwipeFinish when exceeding threshold', () => {
    const calls: string[] = []

    useSwipeAction(ref<HTMLElement | null>(null), {
      threshold: 50,
      onSwipeEnd: (dir) => calls.push(`end:${dir}`),
      onSwipeFinish: (_dx, _dy, isTriggered) => calls.push(`finish:${isTriggered}`)
    })

    setDelta(60, 0)
    triggerSwipeEnd()

    expect(calls).toEqual(['end:right', 'finish:true'])
  })

  it('triggers vertical direction and reports isTriggered=true when exceeding threshold', () => {
    const calls: Array<{ dir: SwipeDirection } | { triggered: boolean }> = []

    useSwipeAction(ref<HTMLElement | null>(null), {
      threshold: 50,
      onSwipeEnd: (dir) => calls.push({ dir }),
      onSwipeFinish: (_dx, _dy, isTriggered) => calls.push({ triggered: isTriggered })
    })

    setDelta(0, 70)
    triggerSwipeEnd()

    expect(calls).toEqual([{ dir: 'down' }, { triggered: true }])
  })
})

