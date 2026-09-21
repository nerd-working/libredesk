<template>
  <div class="h-full flex flex-col">
    <ConversationListToolbar />

    <!-- Content -->
    <div class="flex-grow overflow-y-auto overscroll-contain">
      <EmptyList
        v-if="showEmpty"
        key="empty"
        class="px-4 py-8"
        :title="t('conversation.noConversationsFound')"
        :message="t('conversation.tryAdjustingFilters')"
        :icon="Inbox"
      />

      <EmptyList
        v-if="hasErrored"
        key="error"
        class="px-4 py-8"
        :title="t('conversation.couldNotFetch')"
        :message="conversationStore.conversations.errorMessage"
        :icon="MessageCircleWarning"
      />

      <!-- Plain <table>, not the Table primitive: its overflow-auto wrapper would become the
           sticky header's scroll container, and the header would scroll away with the list. -->
      <table
        v-if="!hasErrored && (isLoading || hasConversations)"
        class="w-full caption-bottom text-sm"
      >
        <TableHeader class="sticky top-0 z-10 bg-background">
          <TableRow class="hover:bg-transparent">
            <TableHead v-if="canBulkAct" class="w-10">
              <span class="sr-only">{{ t('conversation.bulkActions.selectConversation') }}</span>
            </TableHead>
            <TableHead class="w-24">{{ t('globals.terms.referenceNumber') }}</TableHead>
            <TableHead>{{ t('globals.terms.subject', 1) }}</TableHead>
            <TableHead class="w-56">{{ t('globals.terms.contact', 1) }}</TableHead>
            <TableHead class="w-32">{{ t('globals.terms.status', 1) }}</TableHead>
            <TableHead class="w-32">{{ t('globals.terms.priority', 1) }}</TableHead>
            <TableHead class="w-36">{{ t('conversation.table.lastActivity') }}</TableHead>
          </TableRow>
        </TableHeader>

        <TableBody>
          <!-- Loading skeleton -->
          <template v-if="isLoading">
            <TableRow v-for="i in 12" :key="`skeleton-${i}`" class="hover:bg-transparent">
              <TableCell v-if="canBulkAct"><Skeleton class="h-4 w-4 rounded-sm" /></TableCell>
              <TableCell><Skeleton class="h-3.5 w-12" /></TableCell>
              <TableCell><Skeleton class="h-3.5" :style="{ width: `${45 + ((i * 17) % 40)}%` }" /></TableCell>
              <TableCell>
                <div class="flex items-center gap-2">
                  <Skeleton class="h-6 w-6 rounded-full" />
                  <Skeleton class="h-3.5 w-28" />
                </div>
              </TableCell>
              <TableCell><Skeleton class="h-5 w-16 rounded-md" /></TableCell>
              <TableCell><Skeleton class="h-3.5 w-14" /></TableCell>
              <TableCell><Skeleton class="h-3.5 w-20" /></TableCell>
            </TableRow>
          </template>

          <template v-else>
            <TableRow
              v-for="conversation in conversationStore.conversationsList"
              :key="conversation.uuid"
              tabindex="0"
              class="cursor-pointer hover:bg-muted/30 focus-visible:outline-none focus-visible:bg-muted/30"
              :class="{
                'bg-accent': isCurrent(conversation),
                'bg-primary/5 hover:bg-primary/10': isSelected(conversation) && !isCurrent(conversation)
              }"
              :aria-selected="isSelected(conversation)"
              @click="openConversation(conversation)"
              @keydown.enter.self="openConversation(conversation)"
            >
              <!-- Selection -->
              <TableCell
                v-if="canBulkAct"
                @click.stop.prevent="handleCheckboxClick(conversation, $event)"
              >
                <Checkbox
                  :checked="isSelected(conversation)"
                  :aria-label="t('conversation.bulkActions.selectConversation')"
                />
              </TableCell>

              <!-- Reference number -->
              <TableCell class="text-muted-foreground tabular-nums whitespace-nowrap">
                {{ conversation.reference_number }}
              </TableCell>

              <!-- Subject -->
              <TableCell class="max-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <span
                    class="truncate text-foreground"
                    :class="{ 'font-semibold': isUnread(conversation) }"
                    :title="conversation.subject"
                  >
                    {{ conversation.subject }}
                  </span>
                  <span
                    v-if="isUnread(conversation)"
                    class="flex items-center justify-center min-w-5 h-5 px-1 bg-primary text-primary-foreground text-xs font-medium rounded-full flex-shrink-0"
                  >
                    {{ conversation.unread_message_count > 9 ? '9+' : conversation.unread_message_count }}
                  </span>
                </div>
              </TableCell>

              <!-- Contact -->
              <TableCell class="max-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <Avatar class="w-6 h-6 rounded-full flex-shrink-0">
                    <AvatarImage :src="conversation.contact?.avatar_url || ''" class="object-cover" />
                    <AvatarFallback class="text-[10px]">
                      {{ (conversation.contact?.first_name || '').substring(0, 2).toUpperCase() }}
                    </AvatarFallback>
                  </Avatar>
                  <span
                    class="truncate"
                    :title="conversationStore.getContactFullName(conversation.uuid)"
                  >
                    {{ conversationStore.getContactFullName(conversation.uuid) }}
                  </span>
                </div>
              </TableCell>

              <!-- Status -->
              <TableCell>
                <Badge variant="secondary" class="whitespace-nowrap font-medium">
                  {{ conversation.status }}
                </Badge>
              </TableCell>

              <!-- Priority -->
              <TableCell>
                <div v-if="conversation.priority" class="flex items-center gap-1.5 whitespace-nowrap">
                  <PriorityMarker :priority="conversation.priority" />
                  <span>{{ conversation.priority }}</span>
                </div>
              </TableCell>

              <!-- Last activity -->
              <TableCell class="text-muted-foreground whitespace-nowrap tabular-nums">
                {{ relativeLastMessageTime(conversation) }}
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </table>

      <ConversationListLoadMore />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Inbox, MessageCircleWarning } from 'lucide-vue-next'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Badge } from '@shared-ui/components/ui/badge'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { useConversationStore } from '@/stores/conversation'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useConversationRoute } from '@/composables/useConversationRoute'
import PriorityMarker from '@/features/conversation/PriorityMarker.vue'
import EmptyList from '@/features/conversation/list/ConversationEmptyList.vue'
import ConversationListToolbar from '@/features/conversation/list/ConversationListToolbar.vue'
import ConversationListLoadMore from '@/features/conversation/list/ConversationListLoadMore.vue'

const conversationStore = useConversationStore()
const { canBulkAct } = useBulkActionPermissions()
const { conversationRouteFor } = useConversationRoute()
const router = useRouter()
const { t } = useI18n()

// Relative times re-render once a minute, same as the chat list items.
let timer = null
const now = ref(new Date())
onMounted(() => {
  timer = setInterval(() => {
    now.value = new Date()
  }, 60000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
const isLoading = computed(() => conversationStore.conversations.loading)
const showEmpty = computed(
  () =>
    !hasConversations.value &&
    !hasErrored.value &&
    !isLoading.value &&
    conversationStore.conversations.initialized
)

const isUnread = (conversation) => conversation.unread_message_count > 0
const isCurrent = (conversation) => conversation.uuid === conversationStore.current?.uuid
const isSelected = (conversation) => conversationStore.isSelected(conversation.uuid)

const relativeLastMessageTime = (conversation) =>
  conversation.last_message_at ? getRelativeTime(conversation.last_message_at, now.value) : ''

const openConversation = (conversation) => {
  router.push(conversationRouteFor(conversation))
}

const handleCheckboxClick = (conversation, event) => {
  conversationStore.toggleSelect(conversation.uuid, event.shiftKey)
}
</script>
