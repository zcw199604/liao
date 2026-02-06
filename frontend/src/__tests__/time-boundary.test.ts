import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { formatFullTime, formatTime } from '@/utils/time'

describe('utils/time boundary', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns empty or raw input for empty/invalid date strings', () => {
    expect(formatTime('')).toBe('')
    expect(formatTime('not-a-date')).toBe('not-a-date')
    expect(formatFullTime('not-a-date')).toBe('not-a-date')
  })

  it('supports zero and negative epoch boundaries', () => {
    vi.setSystemTime(new Date('2026-01-05T12:00:00.000Z'))

    expect(formatTime(new Date(0).toISOString())).toBe('70/01/01')
    expect(formatTime(new Date(-1000).toISOString())).toBe('69/12/31')
  })

  it('returns raw input for extremely large timestamp strings', () => {
    expect(formatTime('9999999999999')).toBe('9999999999999')
    expect(formatFullTime('9999999999999')).toBe('9999999999999')
  })

  it('keeps timezone offset parsing and day classification stable', () => {
    vi.setSystemTime(new Date('2026-01-05T12:00:00.000Z'))

    expect(formatTime('2026-01-04T23:30:00-08:00')).toBe('07:30')
    expect(formatFullTime('2026-01-04T23:30:00-08:00')).toBe('2026-01-05 07:30:00')
  })
})
