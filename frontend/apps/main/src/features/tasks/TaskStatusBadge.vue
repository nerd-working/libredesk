<template>
  <span
    class="inline-flex max-w-full items-center gap-1.5 rounded-md border border-border px-2 py-0.5 text-xs font-medium"
  >
    <span class="h-2 w-2 shrink-0 rounded-full" :style="{ backgroundColor: dotColor }" />
    <span class="truncate">{{ name }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, default: '' },
  color: { type: String, default: '' },
  category: { type: String, default: 'todo' }
})

const CATEGORY_COLORS = { todo: '#94a3b8', in_progress: '#3b82f6', done: '#22c55e' }
const dotColor = computed(() => props.color || CATEGORY_COLORS[props.category] || CATEGORY_COLORS.todo)
</script>
