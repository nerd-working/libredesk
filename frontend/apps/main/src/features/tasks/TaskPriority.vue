<template>
  <span class="inline-flex items-center gap-1 text-xs font-medium" :class="TASK_PRIORITY_CLASSES[priority]">
    <SignalHigh v-if="priority === 'high'" class="h-3.5 w-3.5" />
    <SignalMedium v-else-if="priority === 'medium'" class="h-3.5 w-3.5" />
    <SignalLow v-else-if="priority === 'low'" class="h-3.5 w-3.5" />
    <AlertTriangle v-else class="h-3.5 w-3.5" />
    <span v-if="!iconOnly">{{ t(`tasks.priority.${priority}`) }}</span>
  </span>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { AlertTriangle, SignalHigh, SignalLow, SignalMedium } from 'lucide-vue-next'
import { TASK_PRIORITY_CLASSES } from '@/constants/tasks.js'

defineProps({
  priority: { type: String, default: 'medium' },
  iconOnly: { type: Boolean, default: false }
})
const { t } = useI18n()
</script>
