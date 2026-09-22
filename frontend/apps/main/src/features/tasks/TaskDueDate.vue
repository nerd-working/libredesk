<template>
  <span
    v-if="task.due_date"
    class="inline-flex items-center gap-1 text-xs"
    :class="overdue ? 'font-medium text-red-600 dark:text-red-400' : 'text-muted-foreground'"
  >
    <CalendarDays class="h-3.5 w-3.5" />
    {{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CalendarDays } from 'lucide-vue-next'
import { isOverdue, localDateString } from './taskUtils.js'

const props = defineProps({ task: { type: Object, required: true } })
const { t, locale } = useI18n()

const overdue = computed(() => isOverdue(props.task))
const label = computed(() => {
  const due = props.task.due_date
  if (due === localDateString()) return t('tasks.due.today')
  const [y, m, d] = due.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString(locale.value, {
    day: 'numeric',
    month: 'short',
    year: y === new Date().getFullYear() ? undefined : 'numeric'
  })
})
</script>
