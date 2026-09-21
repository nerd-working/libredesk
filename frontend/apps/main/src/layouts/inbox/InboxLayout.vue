<template>
  <ResizablePanelGroup
    v-if="!isSearchRoute && !isMobile && !isTableLayout"
    direction="horizontal"
    class="h-full w-full"
    @layout="onLayoutChange"
  >
    <!-- Conversation List Panel -->
    <ResizablePanel :default-size="panelSizes[0]" :min-size="20" :max-size="45">
      <ConversationList />
    </ResizablePanel>

    <ResizableHandle />

    <!-- Conversation Detail Panel -->
    <ResizablePanel :default-size="panelSizes[1]" :min-size="30">
      <router-view v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </ResizablePanel>
  </ResizablePanelGroup>

  <!-- Full-screen list / detail: mobile, and the table layout on desktop.
       v-show, not v-if: the list keeps its scroll position. -->
  <div v-else-if="!isSearchRoute" class="h-full w-full">
    <component :is="listComponent" v-show="isListRoute" />
    <div v-show="!isListRoute" class="h-full">
      <router-view v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useStorage } from '@vueuse/core'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import ConversationTable from '@/features/conversation/list/ConversationTable.vue'
import { useIsMobile } from '@shared-ui/composables'
import { useInboxLayout } from '@/composables/useInboxLayout'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@shared-ui/components/ui/resizable'

defineOptions({ name: 'InboxLayout' })

const route = useRoute()
const isMobile = useIsMobile()
const { isTableLayout } = useInboxLayout()
const isSearchRoute = computed(() => route.name === 'search')

// Every detail route is its list route's name plus `-conversation`.
const isListRoute = computed(() => !String(route.name).endsWith('-conversation'))

// The table doesn't fit on phones: mobile always gets the chat list.
const listComponent = computed(() =>
  isTableLayout.value && !isMobile.value ? ConversationTable : ConversationList
)

// [conversationList, conversationDetail]
const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}
</script>
