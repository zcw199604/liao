import { describe, expect, it } from 'vitest'

import { colorClasses, getColorClass } from '@/constants/colors'

describe('constants/colors', () => {
  it('falls back to bg-color-1 when id is empty or too short', () => {
    expect(getColorClass('')).toBe('bg-color-1')
    expect(getColorClass('a')).toBe('bg-color-1')
  })

  it('returns a known color class for normal ids', () => {
    expect(getColorClass('ab')).toMatch(/^bg-color-/)
  })

  it('falls back when computed color class is missing', () => {
    const original = colorClasses[0]
    colorClasses[0] = '' as any
    try {
      expect(getColorClass('ab')).toBe('bg-color-1')
    } finally {
      colorClasses[0] = original
    }
  })
})
