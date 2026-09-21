<template>
  <div
    v-if="!hasErrored && (conversationStore.conversations.hasMore || hasConversations)"
    class="flex justify-center items-center p-5"
  >
    <Button
      v-if="conversationStore.conversations.hasMore"
      variant="outline"
      @click="conversationStore.fetchNextConversations"
      :disabled="conversationStore.conversations.fetching"
      class="max-md:h-11 transition-all duration-200 ease-in-out transform hover:scale-105"
    >
      <Loader2 v-if="conversationStore.conversations.fetching" class="mr-2 h-4 w-4 animate-spin" />
      {{ conversationStore.conversations.fetching ? t('globals.terms.loading') : t('globals.terms.loadMore') }}
    </Button>
    <p
      class="text-sm text-muted-foreground"
      v-else-if="conversationStore.conversationsList.length > 10"
    >
      {{ $t('conversation.allLoaded') }}
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useConversationStore } from '@/stores/conversation'

const conversationStore = useConversationStore()
const { t } = useI18n()

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
</script>
