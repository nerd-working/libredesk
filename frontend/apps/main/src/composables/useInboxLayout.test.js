// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'

const load = async () => {
  const mod = await import('./useInboxLayout')
  return mod
}

describe('useInboxLayout', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('defaults to the chat layout', async () => {
    const { useInboxLayout, INBOX_LAYOUT } = await load()
    const { layout, isTableLayout } = useInboxLayout()
    expect(layout.value).toBe(INBOX_LAYOUT.CHAT)
    expect(isTableLayout.value).toBe(false)
  })

  it('toggles between chat and table and persists the choice', async () => {
    const { useInboxLayout, INBOX_LAYOUT } = await load()
    const { isTableLayout, toggleLayout, setLayout } = useInboxLayout()

    toggleLayout()
    await nextTick()
    expect(isTableLayout.value).toBe(true)
    expect(localStorage.getItem('inboxLayout')).toBe(INBOX_LAYOUT.TABLE)

    toggleLayout()
    await nextTick()
    expect(isTableLayout.value).toBe(false)
    expect(localStorage.getItem('inboxLayout')).toBe(INBOX_LAYOUT.CHAT)

    setLayout(INBOX_LAYOUT.TABLE)
    await nextTick()
    expect(isTableLayout.value).toBe(true)
  })

  it('shares one state between callers', async () => {
    const { useInboxLayout, INBOX_LAYOUT } = await load()
    const a = useInboxLayout()
    const b = useInboxLayout()
    a.setLayout(INBOX_LAYOUT.TABLE)
    expect(b.isTableLayout.value).toBe(true)
  })
})
