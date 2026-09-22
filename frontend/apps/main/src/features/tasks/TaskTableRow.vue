<template>
  <TableRow
    tabindex="0"
    class="cursor-pointer hover:bg-muted/30 focus-visible:bg-muted/30 focus-visible:outline-none"
    :class="{ 'bg-accent': selectedId === task.id }"
    @click="emit('open', task)"
    @keydown.enter.self="emit('open', task)"
  >
    <TableCell class="max-w-0">
      <div class="flex items-center gap-1.5" :style="{ paddingLeft: `${depth * 1.25}rem` }">
        <button
          v-if="task.subtask_count"
          type="button"
          class="-ml-1 rounded p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground"
          :aria-expanded="expanded"
          :aria-label="t('tasks.toggleSubtasks')"
          @click.stop="toggle"
        >
          <ChevronRight class="h-4 w-4 transition-transform" :class="{ 'rotate-90': expanded }" />
        </button>
        <span v-else class="w-4 shrink-0" />
        <span class="truncate font-medium" :class="{ 'text-muted-foreground line-through': isDone(task) }">
          {{ task.title }}
        </span>
        <span
          v-if="task.subtask_count"
          class="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground"
        >
          {{ task.subtask_done_count }}/{{ task.subtask_count }}
        </span>
        <MessageSquare
          v-if="task.conversation_uuid"
          class="h-3.5 w-3.5 shrink-0 text-muted-foreground"
          :aria-label="t('tasks.linkedConversation')"
        />
      </div>
    </TableCell>
    <TableCell v-if="showProject" class="max-w-0">
      <span v-if="task.project_name" class="flex items-center gap-1.5 truncate text-sm">
        <span
          class="h-2 w-2 shrink-0 rounded-full border border-border"
          :style="{ backgroundColor: task.project_color || 'transparent' }"
        />
        <span class="truncate">{{ task.project_name }}</span>
      </span>
    </TableCell>
    <TableCell>
      <TaskStatusBadge :name="task.status_name" :color="task.status_color" :category="task.status_category" />
    </TableCell>
    <TableCell><TaskPriority :priority="task.priority" /></TableCell>
    <TableCell class="max-w-0"><TaskAssignee :task="task" /></TableCell>
    <TableCell><TaskDueDate :task="task" /></TableCell>
  </TableRow>

  <template v-if="expanded">
    <TableRow v-if="loading" class="hover:bg-transparent">
      <TableCell :colspan="showProject ? 6 : 5">
        <Skeleton class="h-3.5 w-48" :style="{ marginLeft: `${(depth + 1) * 1.25 + 1.5}rem` }" />
      </TableCell>
    </TableRow>
    <TaskTableRow
      v-for="sub in subtasks"
      v-else
      :key="sub.id"
      :task="sub"
      :depth="depth + 1"
      :show-project="showProject"
      :selected-id="selectedId"
      :refresh-key="refreshKey"
      @open="emit('open', $event)"
    />
  </template>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronRight, MessageSquare } from 'lucide-vue-next'
import { TableCell, TableRow } from '@shared-ui/components/ui/table'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import TaskStatusBadge from './TaskStatusBadge.vue'
import TaskPriority from './TaskPriority.vue'
import TaskAssignee from './TaskAssignee.vue'
import TaskDueDate from './TaskDueDate.vue'
import { isDone } from './taskUtils.js'
import api from '@/api'

defineOptions({ name: 'TaskTableRow' })

const props = defineProps({
  task: { type: Object, required: true },
  depth: { type: Number, default: 0 },
  showProject: { type: Boolean, default: true },
  selectedId: { type: Number, default: null },
  // Bumped by the parent after any change so expanded rows reload.
  refreshKey: { type: Number, default: 0 }
})
const emit = defineEmits(['open'])
const { t } = useI18n()

const expanded = ref(false)
const loading = ref(false)
const subtasks = ref([])

const loadSubtasks = async () => {
  loading.value = subtasks.value.length === 0
  try {
    const resp = await api.getTasks({ parent_task_id: props.task.id, all: 1 })
    subtasks.value = resp.data.data.results
  } catch {
    subtasks.value = []
  } finally {
    loading.value = false
  }
}

const toggle = () => {
  expanded.value = !expanded.value
  if (expanded.value) loadSubtasks()
}

watch(
  () => props.refreshKey,
  () => {
    if (expanded.value) loadSubtasks()
  }
)
</script>
