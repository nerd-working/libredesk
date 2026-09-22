<template>
  <div class="flex h-full gap-3 overflow-x-auto p-4">
    <section
      v-for="status in statuses"
      :key="status.id"
      class="flex w-72 shrink-0 flex-col rounded-lg border border-border bg-muted/30"
      :aria-label="status.name"
    >
      <header class="flex items-center justify-between px-3 py-2">
        <TaskStatusBadge :name="status.name" :color="status.color" :category="status.category" />
        <span class="text-xs text-muted-foreground">{{ columns[status.id]?.length || 0 }}</span>
      </header>

      <Draggable
        v-model="columns[status.id]"
        item-key="id"
        group="tasks"
        class="flex min-h-24 flex-1 flex-col gap-2 overflow-y-auto px-2 pb-2"
        ghost-class="opacity-40"
        :disabled="!canEdit"
        :force-fallback="true"
        :fallback-tolerance="3"
        @change="(e) => onChange(status, e)"
      >
        <template #item="{ element }">
          <article
            tabindex="0"
            class="cursor-pointer space-y-2 rounded-md border border-border bg-background p-3 shadow-sm transition-colors hover:border-primary/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            :class="{ 'border-primary': selectedId === element.id }"
            @click="emit('open', element)"
            @keydown.enter="emit('open', element)"
          >
            <p
              class="text-sm font-medium leading-snug"
              :class="{ 'text-muted-foreground line-through': isDone(element) }"
            >
              {{ element.title }}
            </p>
            <div v-if="element.project_name && showProject" class="flex items-center gap-1.5 text-xs text-muted-foreground">
              <span
                class="h-2 w-2 rounded-full border border-border"
                :style="{ backgroundColor: element.project_color || 'transparent' }"
              />
              <span class="truncate">{{ element.project_name }}</span>
            </div>
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <TaskPriority :priority="element.priority" icon-only />
                <TaskDueDate :task="element" />
                <span v-if="element.subtask_count" class="inline-flex items-center gap-1 text-xs text-muted-foreground">
                  <ListChecks class="h-3.5 w-3.5" />
                  {{ element.subtask_done_count }}/{{ element.subtask_count }}
                </span>
                <MessageSquare v-if="element.conversation_uuid" class="h-3.5 w-3.5 text-muted-foreground" />
              </div>
              <TaskAssignee :task="element" avatar-only />
            </div>
          </article>
        </template>
      </Draggable>
    </section>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import Draggable from 'vuedraggable'
import { ListChecks, MessageSquare } from 'lucide-vue-next'
import TaskStatusBadge from './TaskStatusBadge.vue'
import TaskPriority from './TaskPriority.vue'
import TaskDueDate from './TaskDueDate.vue'
import TaskAssignee from './TaskAssignee.vue'
import { isDone, positionBetween } from './taskUtils.js'

const props = defineProps({
  tasks: { type: Array, default: () => [] },
  statuses: { type: Array, default: () => [] },
  showProject: { type: Boolean, default: true },
  canEdit: { type: Boolean, default: false },
  selectedId: { type: Number, default: null }
})
// move: { task, statusId, position }, handled by the parent so it can save and roll back.
const emit = defineEmits(['open', 'move'])

const columns = ref({})

watch(
  () => [props.tasks, props.statuses],
  () => {
    const next = Object.fromEntries(props.statuses.map((s) => [s.id, []]))
    for (const task of props.tasks) next[task.status_id]?.push(task)
    for (const list of Object.values(next)) list.sort((a, b) => a.position - b.position || a.id - b.id)
    columns.value = next
  },
  { immediate: true, deep: false }
)

const onChange = (status, event) => {
  const change = event.added || event.moved
  if (!change) return
  const list = columns.value[status.id]
  const index = change.newIndex
  const position = positionBetween(list[index - 1]?.position, list[index + 1]?.position)
  const task = change.element
  emit('move', { task, statusId: status.id, position })
}
</script>
