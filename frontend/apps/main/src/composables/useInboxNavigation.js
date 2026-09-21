import { useRouter } from 'vue-router'
import { useIsMobile } from '@shared-ui/composables'
import { useConversationStore } from '@main/stores/conversation'
import { useInboxLayout } from '@main/composables/useInboxLayout'

export function useInboxNavigation() {
  const router = useRouter()
  const isMobile = useIsMobile()
  const { isTableLayout } = useInboxLayout()
  const conversationStore = useConversationStore()

  // Keep the open conversation only in the split layout; full-screen layouts land on the list.
  const openConversationUUID = () =>
    !isMobile.value && !isTableLayout.value && conversationStore.isConversationOpen
      ? conversationStore.conversation.data?.uuid
      : null

  const navigate = (listName, params) => {
    const uuid = openConversationUUID()
    if (uuid) {
      return router.push({ name: `${listName}-conversation`, params: { ...params, uuid } })
    }
    return router.push({ name: listName, params })
  }

  const navigateToInbox = (type) => navigate('inbox', { type })
  const navigateToTeamInbox = (teamID) => navigate('team-inbox', { teamID })
  const navigateToViewInbox = (viewID) => navigate('view-inbox', { viewID })

  return { navigateToInbox, navigateToTeamInbox, navigateToViewInbox }
}
