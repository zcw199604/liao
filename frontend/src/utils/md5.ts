const toUtf8Bytes = (input: string): Uint8Array => {
  if (typeof TextEncoder !== 'undefined') {
    return new TextEncoder().encode(input)
  }

  const bytes: number[] = []
  for (let i = 0; i < input.length; i++) {
    let codePoint = input.charCodeAt(i)

    // surrogate pair
    if (codePoint >= 0xd800 && codePoint <= 0xdbff && i + 1 < input.length) {
      const next = input.charCodeAt(i + 1)
      if (next >= 0xdc00 && next <= 0xdfff) {
        codePoint = 0x10000 + ((codePoint - 0xd800) << 10) + (next - 0xdc00)
        i++
      }
    }

    if (codePoint <= 0x7f) {
      bytes.push(codePoint)
    } else if (codePoint <= 0x7ff) {
      bytes.push(0xc0 | (codePoint >> 6))
      bytes.push(0x80 | (codePoint & 0x3f))
    } else if (codePoint <= 0xffff) {
      bytes.push(0xe0 | (codePoint >> 12))
      bytes.push(0x80 | ((codePoint >> 6) & 0x3f))
      bytes.push(0x80 | (codePoint & 0x3f))
    } else {
      bytes.push(0xf0 | (codePoint >> 18))
      bytes.push(0x80 | ((codePoint >> 12) & 0x3f))
      bytes.push(0x80 | ((codePoint >> 6) & 0x3f))
      bytes.push(0x80 | (codePoint & 0x3f))
    }
  }

  return new Uint8Array(bytes)
}

const leftRotate = (value: number, shift: number) => {
  return ((value << shift) | (value >>> (32 - shift))) >>> 0
}

const S = [
  7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22,
  5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20,
  4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23,
  6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21
]

const K = (() => {
  const out = new Uint32Array(64)
  for (let i = 0; i < 64; i++) {
    out[i] = Math.floor(Math.abs(Math.sin(i + 1)) * 2 ** 32) >>> 0
  }
  return out
})()

const wordToHexLE = (word: number) => {
  const b0 = word & 0xff
  const b1 = (word >>> 8) & 0xff
  const b2 = (word >>> 16) & 0xff
  const b3 = (word >>> 24) & 0xff
  return (
    b0.toString(16).padStart(2, '0') +
    b1.toString(16).padStart(2, '0') +
    b2.toString(16).padStart(2, '0') +
    b3.toString(16).padStart(2, '0')
  )
}

export const md5Hex = (input: string): string => {
  const bytes = toUtf8Bytes(String(input ?? ''))

  const bitLen = BigInt(bytes.length) * 8n
  const lowBits = Number(bitLen & 0xffffffffn) >>> 0
  const highBits = Number((bitLen >> 32n) & 0xffffffffn) >>> 0

  const withOne = bytes.length + 1
  const padLen = withOne % 64 <= 56 ? 56 - (withOne % 64) : 56 + 64 - (withOne % 64)
  const totalLen = bytes.length + 1 + padLen + 8

  const buffer = new Uint8Array(totalLen)
  buffer.set(bytes, 0)
  buffer[bytes.length] = 0x80

  const lenOffset = totalLen - 8
  buffer[lenOffset + 0] = lowBits & 0xff
  buffer[lenOffset + 1] = (lowBits >>> 8) & 0xff
  buffer[lenOffset + 2] = (lowBits >>> 16) & 0xff
  buffer[lenOffset + 3] = (lowBits >>> 24) & 0xff
  buffer[lenOffset + 4] = highBits & 0xff
  buffer[lenOffset + 5] = (highBits >>> 8) & 0xff
  buffer[lenOffset + 6] = (highBits >>> 16) & 0xff
  buffer[lenOffset + 7] = (highBits >>> 24) & 0xff

  let a0 = 0x67452301
  let b0 = 0xefcdab89
  let c0 = 0x98badcfe
  let d0 = 0x10325476

  const M = new Uint32Array(16)

  for (let offset = 0; offset < buffer.length; offset += 64) {
    for (let i = 0; i < 16; i++) {
      const base = offset + i * 4
      M[i] = (
        (buffer[base] ?? 0) |
        ((buffer[base + 1] ?? 0) << 8) |
        ((buffer[base + 2] ?? 0) << 16) |
        ((buffer[base + 3] ?? 0) << 24)
      ) >>> 0
    }

    let A = a0
    let B = b0
    let C = c0
    let D = d0

    for (let i = 0; i < 64; i++) {
      let F = 0
      let g = 0

      if (i < 16) {
        F = ((B & C) | (~B & D)) >>> 0
        g = i
      } else if (i < 32) {
        F = ((D & B) | (~D & C)) >>> 0
        g = (5 * i + 1) % 16
      } else if (i < 48) {
        F = (B ^ C ^ D) >>> 0
        g = (3 * i + 5) % 16
      } else {
        F = (C ^ (B | ~D)) >>> 0
        g = (7 * i) % 16
      }

      const temp = D
      D = C
      C = B

      const sum = (A + F + K[i]! + M[g]!) >>> 0
      B = (B + leftRotate(sum, S[i]!)) >>> 0
      A = temp
    }

    a0 = (a0 + A) >>> 0
    b0 = (b0 + B) >>> 0
    c0 = (c0 + C) >>> 0
    d0 = (d0 + D) >>> 0
  }

  return (wordToHexLE(a0) + wordToHexLE(b0) + wordToHexLE(c0) + wordToHexLE(d0)).toLowerCase()
}

