<template>
  <!-- Header -->
  <div class="flex items-center space-x-4 px-2 h-12 border-b shrink-0">
    <SidebarTrigger class="cursor-pointer" />
    <span class="text-xl font-semibold">{{ title }}</span>
  </div>

  <!-- Bulk Action Toolbar (when items selected) -->
  <ConversationBulkActionToolbar v-if="hasSelection && canBulkAct" />

  <!-- Filters (hidden when bulk selecting) -->
  <div v-else class="p-2 flex justify-between items-center">
    <!-- Status dropdown-menu, hidden when a view is selected as views are pre-filtered -->
    <DropdownMenu v-if="!route.params.viewID">
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" class="w-30">
          <div>
            <span class="mr-1">{{ conversationStore.conversations.total }}</span>
            <span>{{ conversationStore.getListStatus }}</span>
          </div>
          <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          v-for="status in conversationStore.statusOptions"
          :key="status.value"
          @click="handleStatusChange(status)"
        >
          {{ status.label }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <div v-else>
      <Button variant="ghost" class="w-30">
        <span>{{ conversationStore.conversations.total }}</span>
      </Button>
    </div>

    <div class="flex items-center gap-1">
      <!-- Sort dropdown-menu -->
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            {{ conversationStore.getListSortField }}
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="handleSortChange('oldest')">
            {{ $t('conversation.sort.oldestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('newest')">
            {{ $t('conversation.sort.newestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_first')">
            {{ $t('conversation.sort.startedFirst') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_last')">
            {{ $t('conversation.sort.startedLast') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('waiting_longest')">
            {{ $t('conversation.sort.waitingLongest') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('next_sla_target')">
            {{ $t('conversation.sort.nextSLATarget') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('priority_first')">
            {{ $t('conversation.sort.priorityFirst') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <!-- Chat / table layout toggle. The table doesn't fit on phones, so it's desktop only. -->
      <Tooltip v-if="!isMobile">
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            :aria-label="layoutToggleLabel"
            @click="toggleLayout"
          >
            <component :is="isTableLayout ? LayoutList : Table2" class="w-4 h-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ layoutToggleLabel }}</TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronDown, LayoutList, Table2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useIsMobile } from '@shared-ui/composables'
import { useConversationStore } from '@/stores/conversation'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useInboxLayout } from '@/composables/useInboxLayout'
import ConversationBulkActionToolbar from '@/features/conversation/list/ConversationBulkActionToolbar.vue'

const conversationStore = useConversationStore()
const { canBulkAct } = useBulkActionPermissions()
const { isTableLayout, toggleLayout } = useInboxLayout()
const isMobile = useIsMobile()
const route = useRoute()
const { t } = useI18n()

const hasSelection = computed(() => conversationStore.selectedCount > 0)

const title = computed(() => {
  const typeKey = route.meta?.typeKey?.(route)
  if (typeKey) {
    return t(typeKey)
  }
  const key = route.meta?.titleKey
  if (!key) return ''
  return t(key, route.meta?.titleCount || 1)
})

const layoutToggleLabel = computed(() =>
  isTableLayout.value
    ? t('conversation.layout.switchToChat')
    : t('conversation.layout.switchToTable')
)

const handleStatusChange = (status) => {
  conversationStore.setListStatus(status.label)
}

const handleSortChange = (order) => {
  conversationStore.setListSortField(order)
}
</script>
