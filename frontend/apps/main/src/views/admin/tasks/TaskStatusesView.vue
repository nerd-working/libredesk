<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <div class="mb-5 flex justify-end">
          <Button @click="openDialog(null)">{{ t('tasks.newStatus') }}</Button>
        </div>
        <DataTable :columns="columns" :data="taskMeta.statuses" :loading="isLoading" />
      </LoadingOverlay>
    </template>
    <template #help>
      <p>{{ t('tasks.statusesHelp') }}</p>
    </template>
  </AdminSplitLayout>

  <Dialog v-model:open="dialogOpen">
    <DialogContent class="sm:max-w-[440px]">
      <DialogHeader>
        <DialogTitle>{{ editing ? t('tasks.editStatus') : t('tasks.newStatus') }}</DialogTitle>
        <DialogDescription>{{ t('tasks.statusCategoryHelp') }}</DialogDescription>
      </DialogHeader>
      <TaskStatusForm @submit.prevent="onSubmit">
        <template #footer>
          <DialogFooter class="mt-6">
            <Button type="submit" :isLoading="saving" :disabled="saving">
              {{ editing ? t('globals.messages.save') : t('globals.messages.create') }}
            </Button>
          </DialogFooter>
        </template>
      </TaskStatusForm>
    </DialogContent>
  </Dialog>

  <AlertDialog v-model:open="deleteOpen">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>{{ t('tasks.deleteStatusConfirmation') }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="onDelete">
          {{ t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Pencil, Trash } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import TaskStatusForm from '@/features/admin/task-statuses/TaskStatusForm.vue'
import TaskStatusBadge from '@/features/tasks/TaskStatusBadge.vue'
import { createStatusFormSchema } from '@/features/tasks/formSchema.js'
import { useTaskMetaStore } from '@/stores/taskMeta'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const taskMeta = useTaskMetaStore()
const isLoading = ref(false)
const saving = ref(false)
const dialogOpen = ref(false)
const editing = ref(null)
// The dialog closes before its action runs, so the item is kept apart from the open state.
const deleteOpen = ref(false)
const toDelete = ref(null)

const form = useForm({ validationSchema: toTypedSchema(createStatusFormSchema(t)) })

const showError = (error) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })

const load = async () => {
  isLoading.value = true
  await taskMeta.fetchStatuses(true)
  isLoading.value = false
}
onMounted(load)

const iconButton = (icon, label, onClick, extra = '') =>
  h(
    Button,
    { variant: 'ghost', size: 'icon', class: `h-8 w-8 ${extra}`, title: label, 'aria-label': label, onClick },
    () => h(icon, { class: 'h-4 w-4' })
  )

const columns = [
  {
    accessorKey: 'name',
    header: () => t('globals.terms.name'),
    cell: ({ row }) =>
      h('div', { class: 'flex justify-center' }, [
        h(TaskStatusBadge, {
          name: row.original.name,
          color: row.original.color,
          category: row.original.category
        })
      ])
  },
  {
    accessorKey: 'category',
    header: () => t('globals.terms.category'),
    cell: ({ row }) => t(`tasks.category.${row.original.category}`)
  },
  {
    accessorKey: 'position',
    header: () => t('tasks.position')
  },
  {
    accessorKey: 'is_default',
    enableGlobalFilter: false,
    header: () => t('tasks.defaultStatus'),
    cell: ({ row }) => (row.original.is_default ? '✓' : '')
  },
  {
    id: 'actions',
    enableSorting: false,
    cell: ({ row }) =>
      h('div', { class: 'flex justify-end gap-1' }, [
        iconButton(Pencil, t('globals.messages.edit'), () => openDialog(row.original)),
        row.original.is_default
          ? null
          : iconButton(Trash, t('globals.messages.delete'), () => confirmDelete(row.original), 'text-destructive')
      ])
  }
]

const openDialog = (status) => {
  editing.value = status
  form.resetForm({
    values: status
      ? { name: status.name, category: status.category, color: status.color, position: status.position, is_default: status.is_default }
      : { name: '', category: 'todo', color: '', position: (taskMeta.statuses.at(-1)?.position || 0) + 1, is_default: false }
  })
  dialogOpen.value = true
}

const onSubmit = form.handleSubmit(async (values) => {
  saving.value = true
  try {
    if (editing.value) await api.updateTaskStatus(editing.value.id, values)
    else await api.createTaskStatus(values)
    dialogOpen.value = false
    await load()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.savedSuccessfully') })
  } catch (error) {
    showError(error)
  } finally {
    saving.value = false
  }
})

const confirmDelete = (status) => {
  toDelete.value = status
  deleteOpen.value = true
}

const onDelete = async () => {
  const status = toDelete.value
  deleteOpen.value = false
  if (!status) return
  try {
    await api.deleteTaskStatus(status.id)
    await load()
  } catch (error) {
    showError(error)
  }
}
</script>
