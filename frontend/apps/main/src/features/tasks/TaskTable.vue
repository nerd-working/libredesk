<template>
  <!-- Plain <table> so the sticky header sticks to the page's scroll container. -->
  <table class="w-full table-fixed caption-bottom text-sm">
    <TableHeader class="sticky top-0 z-10 bg-background">
      <TableRow class="hover:bg-transparent">
        <TableHead>{{ t('tasks.fields.title') }}</TableHead>
        <TableHead v-if="showProject" class="w-44">{{ t('tasks.fields.project') }}</TableHead>
        <TableHead class="w-40">{{ t('globals.terms.status', 1) }}</TableHead>
        <TableHead class="w-28">{{ t('globals.terms.priority', 1) }}</TableHead>
        <TableHead class="w-48">{{ t('tasks.fields.assignee') }}</TableHead>
        <TableHead class="w-32">{{ t('tasks.fields.dueDate') }}</TableHead>
      </TableRow>
    </TableHeader>

    <TableBody>
      <template v-if="loading">
        <TableRow v-for="i in 8" :key="`skeleton-${i}`" class="hover:bg-transparent">
          <TableCell><Skeleton class="h-3.5" :style="{ width: `${40 + ((i * 17) % 40)}%` }" /></TableCell>
          <TableCell v-if="showProject"><Skeleton class="h-3.5 w-24" /></TableCell>
          <TableCell><Skeleton class="h-5 w-20 rounded-md" /></TableCell>
          <TableCell><Skeleton class="h-3.5 w-14" /></TableCell>
          <TableCell><Skeleton class="h-3.5 w-28" /></TableCell>
          <TableCell><Skeleton class="h-3.5 w-16" /></TableCell>
        </TableRow>
      </template>

      <template v-else>
        <template v-for="group in groupedRows" :key="group.key">
          <TableRow v-if="group.label" class="bg-muted/40 hover:bg-muted/40">
            <TableCell :colspan="showProject ? 6 : 5" class="py-1.5 text-xs font-semibold text-muted-foreground">
              {{ group.label }}
              <span class="ml-1 font-normal">({{ group.tasks.length }})</span>
            </TableCell>
          </TableRow>
          <TaskTableRow
            v-for="task in group.tasks"
            :key="task.id"
            :task="task"
            :show-project="showProject"
            :selected-id="selectedId"
            :refresh-key="refreshKey"
            @open="emit('open', $event)"
          />
        </template>
      </template>
    </TableBody>
  </table>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import TaskTableRow from './TaskTableRow.vue'
import { groupByDueBucket } from './taskUtils.js'

const props = defineProps({
  tasks: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  showProject: { type: Boolean, default: true },
  // Group rows under Overdue / Today / Upcoming / No due date / Done headings.
  groupByDue: { type: Boolean, default: false },
  selectedId: { type: Number, default: null },
  refreshKey: { type: Number, default: 0 }
})
const emit = defineEmits(['open'])
const { t } = useI18n()

const groupedRows = computed(() => {
  if (!props.groupByDue) return [{ key: 'all', label: '', tasks: props.tasks }]
  return groupByDueBucket(props.tasks).map((g) => ({ ...g, label: t(`tasks.due.${g.key}`) }))
})
</script>
