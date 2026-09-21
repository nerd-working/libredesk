// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref, reactive } from 'vue'

const { push, toggleSelect, store } = vi.hoisted(() => {
  const conversations = [
    {
      uuid: 'a',
      reference_number: 101,
      subject: 'Printer on fire',
      status: 'Open',
      priority: 'High',
      unread_message_count: 2,
      last_message_at: new Date().toISOString(),
      contact: { first_name: 'Ada', last_name: 'Lovelace', avatar_url: '' }
    },
    {
      uuid: 'b',
      reference_number: 102,
      subject: 'Invoice question',
      status: 'Resolved',
      priority: 'Low',
      unread_message_count: 0,
      last_message_at: new Date().toISOString(),
      contact: { first_name: 'Grace', last_name: 'Hopper', avatar_url: '' }
    }
  ]
  const selected = new Set()
  return {
    push: vi.fn(),
    toggleSelect: vi.fn((uuid) => (selected.has(uuid) ? selected.delete(uuid) : selected.add(uuid))),
    store: {
      conversations: { loading: false, initialized: true, errorMessage: '', hasMore: false, fetching: false, total: 2 },
      conversationsList: conversations,
      current: {},
      selectedCount: 0,
      getContactFullName: (uuid) => {
        const c = conversations.find((x) => x.uuid === uuid)
        return `${c.contact.first_name} ${c.contact.last_name}`
      },
      isSelected: (uuid) => selected.has(uuid)
    }
  }
})

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
  useRoute: () => reactive({ name: 'inbox', params: { type: 'assigned' }, meta: {} })
}))
vi.mock('@/stores/conversation', () => ({ useConversationStore: () => ({ ...store, toggleSelect }) }))
vi.mock('@/composables/useBulkActionPermissions', () => ({
  useBulkActionPermissions: () => ({ canBulkAct: ref(true) })
}))
// The toolbar pulls in the sidebar / dropdown primitives; it's covered by the chat list already.
vi.mock('@/features/conversation/list/ConversationListToolbar.vue', () => ({
  default: { setup: () => () => h('div', { 'data-testid': 'toolbar' }) }
}))
vi.mock('@shared-ui/components/ui/checkbox', () => ({
  Checkbox: {
    props: ['checked'],
    setup: (props) => () => h('input', { type: 'checkbox', checked: props.checked })
  }
}))
vi.mock('@shared-ui/components/ui/avatar', () => ({
  Avatar: { setup: (_, { slots }) => () => h('span', slots.default?.()) },
  AvatarImage: { setup: () => () => null },
  AvatarFallback: { setup: (_, { slots }) => () => h('span', slots.default?.()) }
}))
vi.mock('@/features/conversation/PriorityMarker.vue', () => ({
  default: { props: ['priority'], setup: () => () => null }
}))

import ConversationTable from './ConversationTable.vue'

let app
let root
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.clearAllMocks()
})

const mount = async () => {
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(ConversationTable)
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
  await nextTick()
}

describe('ConversationTable', () => {
  it('renders one row per conversation with its columns', async () => {
    await mount()
    const rows = root.querySelectorAll('tbody tr')
    expect(rows).toHaveLength(2)

    const first = rows[0].textContent
    expect(first).toContain('101')
    expect(first).toContain('Printer on fire')
    expect(first).toContain('Ada Lovelace')
    expect(first).toContain('Open')
    expect(first).toContain('High')
  })

  it('bolds the subject of unread conversations only', async () => {
    await mount()
    const [unread, read] = root.querySelectorAll('tbody tr')
    expect(unread.querySelector('.font-semibold')).not.toBeNull()
    expect(read.querySelector('.font-semibold')).toBeNull()
  })

  it('navigates to the conversation when a row is clicked', async () => {
    await mount()
    root.querySelectorAll('tbody tr')[1].click()
    expect(push).toHaveBeenCalledWith({
      name: 'inbox-conversation',
      params: { uuid: 'b' },
      query: {}
    })
  })

  it('toggles selection from the checkbox cell without navigating', async () => {
    await mount()
    const cell = root.querySelector('tbody tr td')
    cell.click()
    expect(toggleSelect).toHaveBeenCalledWith('a', false)
    expect(push).not.toHaveBeenCalled()
  })

  it('opens the conversation on Enter', async () => {
    await mount()
    const row = root.querySelector('tbody tr')
    row.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    expect(push).toHaveBeenCalledWith(expect.objectContaining({ params: { uuid: 'a' } }))
  })
})
