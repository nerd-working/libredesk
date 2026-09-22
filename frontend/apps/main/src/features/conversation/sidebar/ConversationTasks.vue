<template>
  <div class="space-y-2">
    <p v-if="!loading && !tasks.length" class="py-2 text-center text-sm text-muted-foreground">
      {{ t('tasks.noLinkedTasks') }}
    </p>
    <ul v-else class="space-y-1">
      <li v-for="task in tasks" :key="task.id">
        <router-link
          :to="{ name: 'tasks-all', query: { task: task.id } }"
          class="block space-y-1 rounded-md p-2 hover:bg-muted"
        >
          <span
            class="sidebar-value block truncate font-medium"
            :class="{ 'text-muted-foreground line-through': isDone(task) }"
          >
            {{ task.title }}
          </span>
          <span class="flex flex-wrap items-center gap-2">
            <TaskStatusBadge :name="task.status_name" :color="task.status_color" :category="task.status_category" />
            <TaskDueDate :task="task" />
            <TaskAssignee :task="task" avatar-only />
          </span>
        </router-link>
      </li>
    </ul>
    <Button
      v-if="userStore.can('tasks:write')"
      variant="ghost"
      size="sm"
      class="h-7 w-full gap-1 text-xs"
      @click="createOpen = true"
    >
      <Plus class="h-3.5 w-3.5" />
      {{ t('tasks.createFromConversation') }}
    </Button>
    <TaskCreateDialog
      v-if="conversation"
      v-model:open="createOpen"
      :conversation="conversation"
      assign-to-me
      @created="load"
    />
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useConversationStore } from '@/stores/conversation'
import { useUserStore } from '@/stores/user'
import TaskStatusBadge from '@/features/tasks/TaskStatusBadge.vue'
import TaskDueDate from '@/features/tasks/TaskDueDate.vue'
import TaskAssignee from '@/features/tasks/TaskAssignee.vue'
import TaskCreateDialog from '@/features/tasks/TaskCreateDialog.vue'
import { isDone } from '@/features/tasks/taskUtils.js'
import api from '@/api'

const { t } = useI18n()
const conversationStore = useConversationStore()
const userStore = useUserStore()
const tasks = ref([])
const loading = ref(false)
const createOpen = ref(false)

const conversation = computed(() => {
  const c = conversationStore.current
  return c?.uuid ? { uuid: c.uuid, reference_number: c.reference_number } : null
})

const load = async () => {
  const uuid = conversation.value?.uuid
  if (!uuid) return
  loading.value = true
  try {
    const resp = await api.getConversationTasks(uuid)
    if (conversation.value?.uuid === uuid) tasks.value = resp.data.data
  } catch {
    tasks.value = []
  } finally {
    loading.value = false
  }
}

watch(() => conversation.value?.uuid, load, { immediate: true })
</script>
