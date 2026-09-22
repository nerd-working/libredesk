<template>
  <span v-if="name" class="inline-flex min-w-0 items-center gap-2">
    <Avatar class="h-6 w-6 shrink-0">
      <AvatarImage :src="task.assignee_avatar_url || ''" :alt="name" />
      <AvatarFallback class="text-[10px]">{{ initials }}</AvatarFallback>
    </Avatar>
    <span v-if="!avatarOnly" class="truncate text-sm">{{ name }}</span>
  </span>
  <span v-else-if="!avatarOnly" class="text-sm text-muted-foreground">{{ t('tasks.unassigned') }}</span>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { assigneeName } from './taskUtils.js'

const props = defineProps({
  task: { type: Object, required: true },
  avatarOnly: { type: Boolean, default: false }
})
const { t } = useI18n()

const name = computed(() => assigneeName(props.task))
const initials = computed(() =>
  name.value
    .split(' ')
    .map((p) => p.charAt(0))
    .slice(0, 2)
    .join('')
    .toUpperCase()
)
</script>
