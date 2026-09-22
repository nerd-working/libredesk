<template>
  <div>
    <div class="flex items-center gap-1.5">
      <p class="sidebar-label">{{ $t('globals.terms.subject', 1) }}</p>
      <button
        v-if="canEdit && !isEditing"
        type="button"
        class="text-muted-foreground hover:text-foreground"
        :aria-label="t('conversation.editSubject')"
        :title="t('conversation.editSubject')"
        @click="startEditing"
      >
        <Pencil class="size-3" />
      </button>
    </div>

    <!-- Reading -->
    <p v-if="!isEditing" class="sidebar-value break-all">
      {{ subject || '-' }}
    </p>

    <!-- Editing -->
    <div v-else class="space-y-2">
      <Textarea
        ref="inputRef"
        v-model="draft"
        rows="2"
        :maxlength="MAX_LENGTH"
        :disabled="saving"
        :aria-label="t('globals.terms.subject', 1)"
        class="text-sm"
        @keydown.enter.exact.prevent="save"
        @keydown.esc.prevent="cancel"
      />
      <div class="flex items-center gap-2">
        <Button size="sm" :disabled="saving || !isDirty" @click="save">
          {{ t('globals.messages.save') }}
        </Button>
        <Button size="sm" variant="ghost" :disabled="saving" @click="cancel">
          {{ t('globals.messages.cancel') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Pencil } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { useConversationStore } from '@/stores/conversation'
import { useUserStore } from '@/stores/user'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { permissions as perms } from '@/constants/permissions.js'

// Mirrors maxConversationSubjectLen in cmd/conversation.go.
const MAX_LENGTH = 1000

const conversationStore = useConversationStore()
const userStore = useUserStore()
const emitter = useEmitter()
const { t } = useI18n()

const isEditing = ref(false)
const saving = ref(false)
const draft = ref('')
const inputRef = ref(null)

const subject = computed(() => conversationStore.current?.subject || '')
const canEdit = computed(() => userStore.can(perms.CONVERSATIONS_WRITE))
const isDirty = computed(() => draft.value.trim() !== subject.value.trim())

const startEditing = async () => {
  draft.value = subject.value
  isEditing.value = true
  await nextTick()
  // Textarea forwards attrs to the underlying element, so reach for it either way.
  const el = inputRef.value?.$el ?? inputRef.value
  el?.focus?.()
}

const cancel = () => {
  isEditing.value = false
  draft.value = ''
}

const save = async () => {
  const next = draft.value.trim()
  if (!next || saving.value) return
  if (!isDirty.value) {
    cancel()
    return
  }
  saving.value = true
  try {
    await conversationStore.updateSubject(next)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    cancel()
  } catch {
    // The store already surfaced the error and rolled the subject back.
  } finally {
    saving.value = false
  }
}
</script>
