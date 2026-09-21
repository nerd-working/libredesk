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

      <TransitionGroup
        enter-active-class="transition-all duration-300 ease-in-out"
        enter-from-class="opacity-0 transform translate-y-4"
        enter-to-class="opacity-100 transform translate-y-0"
        leave-active-class="transition-all duration-300 ease-in-out"
        leave-from-class="opacity-100 transform translate-y-0"
        leave-to-class="opacity-0 transform translate-y-4"
      >
        <div
          v-if="!hasErrored && !conversationStore.conversations.loading"
          key="list"
          class="divide-y divide-border"
          :class="{ 'border-b border-border': hasConversations }"
        >
          <ConversationListItem
            v-for="conversation in conversationStore.conversationsList"
            :key="conversation.uuid"
            :conversation="conversation"
            :currentConversation="conversationStore.current"
            :contactFullName="conversationStore.getContactFullName(conversation.uuid)"
            class="transition-colors duration-200"
          />
        </div>

        <div v-if="conversationStore.conversations.loading" key="loading">
          <ConversationListItemSkeleton v-for="i in 12" :key="i" :index="i - 1" />
        </div>
      </TransitionGroup>

      <ConversationListLoadMore />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Inbox, MessageCircleWarning } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import EmptyList from '@/features/conversation/list/ConversationEmptyList.vue'
import ConversationListToolbar from '@/features/conversation/list/ConversationListToolbar.vue'
import ConversationListLoadMore from '@/features/conversation/list/ConversationListLoadMore.vue'
import ConversationListItem from '@/features/conversation/list/ConversationListItem.vue'
import ConversationListItemSkeleton from '@/features/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
const { t } = useI18n()

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
const showEmpty = computed(
  () =>
    !hasConversations.value &&
    !hasErrored.value &&
    !conversationStore.conversations.loading &&
    conversationStore.conversations.initialized
)
</script>
