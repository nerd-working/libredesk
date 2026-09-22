<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[560px]">
      <DialogHeader>
        <DialogTitle>{{ parentTask ? t('tasks.newSubtask') : t('tasks.newTask') }}</DialogTitle>
        <DialogDescription v-if="parentTask">
          {{ t('tasks.subtaskOf', { title: parentTask.title }) }}
        </DialogDescription>
        <DialogDescription v-else-if="conversation">
          {{ t('tasks.linkedTo', { ref: conversation.reference_number }) }}
        </DialogDescription>
      </DialogHeader>

      <form class="space-y-4" @submit.prevent="onSubmit">
        <FormField v-slot="{ componentField }" name="title">
          <FormItem>
            <FormLabel>{{ t('tasks.fields.title') }}</FormLabel>
            <FormControl>
              <Input type="text" autofocus v-bind="componentField" />
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

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField v-if="!parentTask" v-slot="{ value, handleChange }" name="project_id">
            <FormItem>
              <FormLabel>{{ t('tasks.fields.project') }}</FormLabel>
              <FormControl>
                <SelectComboBox
                  :model-value="value"
                  :items="projectItems"
                  :placeholder="t('tasks.noProject')"
                  @update:model-value="handleChange"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ value, handleChange }" name="status_id">
            <FormItem>
              <FormLabel>{{ t('globals.terms.status', 1) }}</FormLabel>
              <FormControl>
                <SelectComboBox
                  :model-value="value"
                  :items="taskMeta.statusOptions"
                  :placeholder="t('globals.terms.status', 1)"
                  @update:model-value="handleChange"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="priority">
            <FormItem>
              <FormLabel>{{ t('globals.terms.priority', 1) }}</FormLabel>
              <Select v-bind="componentField">
                <FormControl>
                  <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem v-for="p in TASK_PRIORITIES" :key="p" :value="p">
                    {{ t(`tasks.priority.${p}`) }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ value, handleChange }" name="assigned_user_id">
            <FormItem>
              <FormLabel>{{ t('tasks.fields.assignee') }}</FormLabel>
              <FormControl>
                <SelectAgentCombobox
                  :model-value="value"
                  include-none
                  current-user-first
                  exclude-ai-assistants
                  @update:model-value="handleChange"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="due_date">
            <FormItem>
              <FormLabel>{{ t('tasks.fields.dueDate') }}</FormLabel>
              <FormControl>
                <Input type="date" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <DialogFooter>
          <Button type="submit" :isLoading="saving" :disabled="saving">
            {{ t('globals.messages.create') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import { createTaskFormSchema } from './formSchema.js'
import { TASK_PRIORITIES } from '@/constants/tasks.js'
import { useTaskMetaStore } from '@main/stores/taskMeta'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import api from '@main/api'

const props = defineProps({
  open: { type: Boolean, default: false },
  // Prefills: a project, a parent task (subtask) or a conversation ({ uuid, reference_number }).
  projectId: { type: Number, default: null },
  parentTask: { type: Object, default: null },
  conversation: { type: Object, default: null },
  assignToMe: { type: Boolean, default: false }
})
const emit = defineEmits(['update:open', 'created'])

const { t } = useI18n()
const emitter = useEmitter()
const taskMeta = useTaskMetaStore()
const userStore = useUserStore()
const saving = ref(false)

const form = useForm({ validationSchema: toTypedSchema(createTaskFormSchema(t)) })

const projectItems = computed(() => [
  { value: 'none', label: t('tasks.noProject') },
  ...taskMeta.projectOptions
])

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    await Promise.all([taskMeta.fetchStatuses(), taskMeta.fetchProjects()])
    form.resetForm({
      values: {
        title: '',
        description: '',
        project_id: props.projectId ? String(props.projectId) : 'none',
        status_id: taskMeta.defaultStatus ? String(taskMeta.defaultStatus.id) : undefined,
        priority: 'medium',
        assigned_user_id: props.assignToMe ? String(userStore.userID) : 'none',
        due_date: ''
      }
    })
  },
  { immediate: true }
)

const onSubmit = form.handleSubmit(async (values) => {
  saving.value = true
  try {
    const resp = await api.createTask({
      ...values,
      status_id: values.status_id || 0,
      parent_task_id: props.parentTask?.id ?? null,
      conversation_uuid: props.conversation?.uuid || ''
    })
    emit('created', resp.data.data)
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
