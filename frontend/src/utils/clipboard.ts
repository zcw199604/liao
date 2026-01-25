export const copyToClipboard = async (text: string): Promise<boolean> => {
  const value = String(text ?? '').trim()
  if (!value) return false

  const writeText = navigator.clipboard?.writeText
  if (typeof writeText === 'function') {
    try {
      await writeText.call(navigator.clipboard, value)
      return true
    } catch {
      // fallback to execCommand
    }
  }

  try {
    if (!document?.body) return false

    const textarea = document.createElement('textarea')
    textarea.value = value
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.top = '0'
    textarea.style.left = '0'
    textarea.style.width = '1px'
    textarea.style.height = '1px'
    textarea.style.padding = '0'
    textarea.style.border = 'none'
    textarea.style.outline = 'none'
    textarea.style.boxShadow = 'none'
    textarea.style.background = 'transparent'
    textarea.style.opacity = '0'

    document.body.appendChild(textarea)
    textarea.focus()
    textarea.select()
    textarea.setSelectionRange(0, textarea.value.length)

    const ok = typeof document.execCommand === 'function' && document.execCommand('copy')
    document.body.removeChild(textarea)
    return !!ok
  } catch {
    return false
  }
}

