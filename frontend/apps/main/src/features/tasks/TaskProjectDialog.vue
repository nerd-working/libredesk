<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[440px]">
      <DialogHeader>
        <DialogTitle>{{ project ? t('tasks.editProject') : t('tasks.newProject') }}</DialogTitle>
        <DialogDescription>{{ t('tasks.projectDescription') }}</DialogDescription>
      </DialogHeader>
      <form class="space-y-4" @submit.prevent="onSubmit">
        <FormField v-slot="{ componentField }" name="name">
          <FormItem>
            <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="description">
          <FormItem>
            <FormLabel>{{ t('globals.terms.description') }}</FormLabel>
            <FormControl>
              <Textarea rows="3" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ value, handleChange }" name="color">
          <FormItem>
            <FormLabel>{{ t('tasks.color') }}</FormLabel>
            <FormControl>
              <ColorSwatches :model-value="value" @update:model-value="handleChange" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <DialogFooter>
          <Button type="submit" :isLoading="saving" :disabled="saving">
            {{ project ? t('globals.messages.save') : t('globals.messages.create') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Button } from '@shared-ui/components/ui/button'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import ColorSwatches from './ColorSwatches.vue'
import { createProjectFormSchema } from './formSchema.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import api from '@main/api'

const props = defineProps({
  open: { type: Boolean, default: false },
  project: { type: Object, default: null }
})
const emit = defineEmits(['update:open', 'saved'])

const { t } = useI18n()
const emitter = useEmitter()
const saving = ref(false)

const form = useForm({ validationSchema: toTypedSchema(createProjectFormSchema(t)) })

watch(
  () => props.open,
  (open) => {
    if (!open) return
    form.resetForm({
      values: {
        name: props.project?.name || '',
        description: props.project?.description || '',
        color: props.project?.color || ''
      }
    })
  }
)

const onSubmit = form.handleSubmit(async (values) => {
  saving.value = true
  try {
    const resp = props.project
      ? await api.updateTaskProject(props.project.id, { ...values, archived: false })
      : await api.createTaskProject(values)
    emit('saved', resp.data.data)
    emit('update:open', false)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    saving.value = false
  }
})
</script>
