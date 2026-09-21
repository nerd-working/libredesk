import { useRoute } from 'vue-router'

// Resolves the detail route for a conversation from the list route the user is on:
// team and view inboxes have their own detail routes carrying the teamID / viewID param.
export function useConversationRoute () {
  const route = useRoute()

  const conversationRouteFor = (conversation) => {
    const baseRoute = route.params.teamID
      ? 'team-inbox-conversation'
      : route.params.viewID
        ? 'view-inbox-conversation'
        : 'inbox-conversation'
    return {
      name: baseRoute,
      params: {
        uuid: conversation.uuid,
        ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
        ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID })
      },
      query: conversation.mentioned_message_uuid
        ? { scrollTo: conversation.mentioned_message_uuid }
        : {}
    }
  }

  return { conversationRouteFor }
}
