<template>
  <div class="flex flex-wrap gap-2" role="radiogroup">
    <button
      v-for="color in TASK_COLORS"
      :key="color || 'none'"
      type="button"
      role="radio"
      :aria-checked="modelValue === color"
      :aria-label="color || t('globals.terms.none')"
      class="h-6 w-6 rounded-full border border-border ring-offset-background transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      :class="{ 'ring-2 ring-ring ring-offset-2': modelValue === color }"
      :style="{ backgroundColor: color || 'transparent' }"
      @click="emit('update:modelValue', color)"
    />
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'

defineProps({ modelValue: { type: String, default: '' } })
const emit = defineEmits(['update:modelValue'])
const { t } = useI18n()

const TASK_COLORS = ['', '#64748b', '#ef4444', '#f97316', '#eab308', '#22c55e', '#06b6d4', '#3b82f6', '#8b5cf6', '#ec4899']
</script>
