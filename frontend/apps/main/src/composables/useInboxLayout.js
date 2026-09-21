import { computed } from 'vue'
import { useStorage } from '@vueuse/core'

export const INBOX_LAYOUT = { CHAT: 'chat', TABLE: 'table' }

// Module scope: one instance shared by every component in the app.
const layout = useStorage('inboxLayout', INBOX_LAYOUT.CHAT)

export function useInboxLayout () {
  const isTableLayout = computed(() => layout.value === INBOX_LAYOUT.TABLE)
  const setLayout = (value) => {
    layout.value = value
  }
  const toggleLayout = () =>
    setLayout(isTableLayout.value ? INBOX_LAYOUT.CHAT : INBOX_LAYOUT.TABLE)
  return { layout, isTableLayout, setLayout, toggleLayout }
}
