import { beforeEach, describe, expect, it } from 'vitest'

import {
  MEDIA_PORT_CANDIDATES,
  applyLearnedMediaPort,
  learnedMediaPort,
  learnMediaPort,
  nextMediaUrlCandidate,
  parseUpstreamMediaUrl,
  resetLearnedMediaPort,
  withMediaPort
} from '@/utils/mediaPort'

describe('utils/mediaPort', () => {
  beforeEach(() => {
    resetLearnedMediaPort()
  })

  describe('parseUpstreamMediaUrl', () => {
    it('parses host/port/rest of an upstream /img/Upload url', () => {
      const parsed = parseUpstreamMediaUrl('http://156.233.225.80:9006/img/Upload/2026/01/a.png')
      expect(parsed).toEqual({
        origin: 'http://156.233.225.80:9006',
        protocol: 'http://',
        host: '156.233.225.80',
        port: '9006',
        rest: '/img/Upload/2026/01/a.png'
      })
    })

    it('supports https, query strings and missing port', () => {
      const parsed = parseUpstreamMediaUrl('https://img.example.com/img/Upload/a.jpg?x=1')
      expect(parsed?.protocol).toBe('https://')
      expect(parsed?.host).toBe('img.example.com')
      expect(parsed?.port).toBe('')
      expect(parsed?.rest).toBe('/img/Upload/a.jpg?x=1')
    })

    it('handles IPv6 literals with port', () => {
      const parsed = parseUpstreamMediaUrl('http://[::1]:7001/img/Upload/a.png')
      expect(parsed?.host).toBe('[::1]')
      expect(parsed?.port).toBe('7001')
      expect(parsed?.rest).toBe('/img/Upload/a.png')
    })

    it('handles IPv6 literals without port', () => {
      const parsed = parseUpstreamMediaUrl('http://[::1]/img/Upload/a.png')
      expect(parsed?.host).toBe('[::1]')
      expect(parsed?.port).toBe('')
    })

    it('returns null for empty or non-upstream urls', () => {
      expect(parseUpstreamMediaUrl('')).toBeNull()
      expect(parseUpstreamMediaUrl('   ')).toBeNull()
      expect(parseUpstreamMediaUrl('http://localhost:8080/upload/images/a.png')).toBeNull()
      expect(parseUpstreamMediaUrl('/api/douyin/download?key=k')).toBeNull()
      expect(parseUpstreamMediaUrl('https://img.t.sinajs.cn/x.gif')).toBeNull()
      // /img/Upload 必须在路径起始处
      expect(parseUpstreamMediaUrl('http://x/a/img/Upload/b.png')).toBeNull()
    })

    it('returns null when host is empty or IPv6 bracket is unterminated', () => {
      expect(parseUpstreamMediaUrl('http://:9006/img/Upload/a.png')).toBeNull()
      expect(parseUpstreamMediaUrl('http://[::1/img/Upload/a.png')).toBeNull()
    })
  })

  describe('withMediaPort', () => {
    it('replaces the port of an upstream url', () => {
      const url = 'http://1.2.3.4:9006/img/Upload/a.png'
      expect(withMediaPort(url, '7001')).toBe('http://1.2.3.4:7001/img/Upload/a.png')
    })

    it('keeps the url unchanged when port is identical or invalid', () => {
      const url = 'http://1.2.3.4:9006/img/Upload/a.png'
      expect(withMediaPort(url, '9006')).toBe(url)
      expect(withMediaPort(url, '')).toBe(url)
      expect(withMediaPort(url, '   ')).toBe(url)
    })

    it('keeps non-upstream urls unchanged', () => {
      const url = 'http://localhost:8080/upload/images/a.png'
      expect(withMediaPort(url, '7001')).toBe(url)
    })

    it('preserves protocol and rest of the url', () => {
      const url = 'https://h:9006/img/Upload/2026/01/a.png?x=1'
      expect(withMediaPort(url, '7004')).toBe('https://h:7004/img/Upload/2026/01/a.png?x=1')
    })
  })

  describe('learned port', () => {
    it('starts empty and can be learned/reset', () => {
      expect(learnedMediaPort.value).toBe('')
      learnMediaPort('7004')
      expect(learnedMediaPort.value).toBe('7004')
      resetLearnedMediaPort()
      expect(learnedMediaPort.value).toBe('')
    })

    it('ignores empty port values', () => {
      learnMediaPort('')
      learnMediaPort('   ')
      expect(learnedMediaPort.value).toBe('')
    })

    it('applyLearnedMediaPort rewrites only when a port was learned', () => {
      const url = 'http://1.2.3.4:9006/img/Upload/a.png'
      expect(applyLearnedMediaPort(url)).toBe(url)

      learnMediaPort('7001', '1.2.3.4')
      expect(applyLearnedMediaPort(url)).toBe('http://1.2.3.4:7001/img/Upload/a.png')
    })

    it('applyLearnedMediaPort does not apply a port learned for another host', () => {
      learnMediaPort('7001', 'other.host')
      const url = 'http://1.2.3.4:9006/img/Upload/a.png'
      expect(applyLearnedMediaPort(url)).toBe(url)
    })

    it('applyLearnedMediaPort still applies when no host was recorded', () => {
      learnMediaPort('7001')
      const url = 'http://1.2.3.4:9006/img/Upload/a.png'
      expect(applyLearnedMediaPort(url)).toBe('http://1.2.3.4:7001/img/Upload/a.png')
    })

    it('applyLearnedMediaPort keeps non-upstream urls unchanged even after learning', () => {
      learnMediaPort('7001')
      const url = '/upload/images/a.png'
      expect(applyLearnedMediaPort(url)).toBe(url)
    })
  })

  describe('nextMediaUrlCandidate', () => {
    it('returns null for non-upstream urls', () => {
      expect(nextMediaUrlCandidate('http://localhost:8080/upload/a.png', new Set())).toBeNull()
      expect(nextMediaUrlCandidate('', new Set())).toBeNull()
    })

    it('walks through candidates in order and records tried ports', () => {
      // 从 9006 出发：跳过已试端口，落到候选表里第一个未试端口
      const tried = new Set<string>()
      const first = nextMediaUrlCandidate('http://h:9006/img/Upload/a.png', tried)
      expect(first).toBe(`http://h:${MEDIA_PORT_CANDIDATES[0]}/img/Upload/a.png`)
      expect(tried.has('9006')).toBe(true)
      expect(tried.has(MEDIA_PORT_CANDIDATES[0]!)).toBe(true)

      const second = nextMediaUrlCandidate(first!, tried)
      expect(second).toBe(`http://h:${MEDIA_PORT_CANDIDATES[1]}/img/Upload/a.png`)

      // 继续直到候选耗尽
      let current = second
      let guard = 0
      while (current && guard < 50) {
        current = nextMediaUrlCandidate(current, tried)
        guard += 1
      }
      expect(current).toBeNull()
      // 9006 本身也在候选表里，所以集合大小就是候选数量
      expect(tried.size).toBe(MEDIA_PORT_CANDIDATES.length)
    })

    it('prefers the learned port before the candidate list', () => {
      learnMediaPort('7004')
      const tried = new Set<string>()
      const next = nextMediaUrlCandidate('http://h:9006/img/Upload/a.png', tried)
      expect(next).toBe('http://h:7004/img/Upload/a.png')
    })

    it('stops retrying once the port of that host has been verified', () => {
      learnMediaPort('7004', 'h')
      const tried = new Set<string>()
      // 该主机已确认使用 7004；请求 7004 仍失败说明资源本身有问题，不再换端口
      expect(nextMediaUrlCandidate('http://h:7004/img/Upload/a.png', tried)).toBeNull()
    })

    it('does not apply a port learned for another host', () => {
      learnMediaPort('7004', 'other.host')
      const tried = new Set<string>()
      const next = nextMediaUrlCandidate('http://h:9006/img/Upload/a.png', tried)
      expect(next).toBe(`http://h:${MEDIA_PORT_CANDIDATES[0]}/img/Upload/a.png`)
    })

    it('falls back to the candidate list when the learned port was already tried', () => {
      learnMediaPort('7004')
      const tried = new Set<string>(['7004'])
      const next = nextMediaUrlCandidate('http://h:9006/img/Upload/a.png', tried)
      expect(next).toBe(`http://h:${MEDIA_PORT_CANDIDATES[0]}/img/Upload/a.png`)
    })

    it('returns null once every candidate has been tried', () => {
      const tried = new Set<string>(MEDIA_PORT_CANDIDATES as unknown as string[])
      tried.add('9006')
      expect(nextMediaUrlCandidate('http://h:9006/img/Upload/a.png', tried)).toBeNull()
    })

    it('handles an upstream url without an explicit port', () => {
      const tried = new Set<string>()
      const next = nextMediaUrlCandidate('http://h/img/Upload/a.png', tried)
      expect(next).toBe(`http://h:${MEDIA_PORT_CANDIDATES[0]}/img/Upload/a.png`)
      // 没有端口时不该往 tried 里塞空字符串
      expect(tried.has('')).toBe(false)
    })

    it('keeps the url path/query intact when switching ports', () => {
      const next = nextMediaUrlCandidate('http://h:9006/img/Upload/2026/01/a.png?x=1', new Set(['7001']))
      expect(next).toBe('http://h:7004/img/Upload/2026/01/a.png?x=1')
    })
  })
})
