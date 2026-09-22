<template>
  <Sheet :open="taskId != null" @update:open="(v) => !v && emit('close')">
    <SheetContent class="flex w-full flex-col gap-0 overflow-y-auto p-0 sm:max-w-xl">
      <div v-if="loading && !task" class="space-y-3 p-6">
        <Skeleton class="h-6 w-2/3" />
        <Skeleton class="h-4 w-1/2" />
        <Skeleton class="h-24 w-full" />
      </div>

      <template v-else-if="task">
        <SheetHeader class="space-y-2 border-b border-border p-6 pr-12 text-left">
          <button
            v-if="task.parent_task_id"
            type="button"
            class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
            @click="emit('open', task.parent_task_id)"
          >
            <CornerLeftUp class="h-3.5 w-3.5" />
            {{ t('tasks.backToParent') }}
          </button>
          <SheetTitle class="sr-only">{{ task.title }}</SheetTitle>
          <SheetDescription class="sr-only">{{ t('tasks.detailDescription') }}</SheetDescription>
          <Textarea
            v-model="draftTitle"
            rows="1"
            class="min-h-0 resize-none border-transparent px-1 text-lg font-semibold shadow-none hover:border-border focus-visible:border-border"
            :disabled="!canWrite"
            :aria-label="t('tasks.fields.title')"
            @keydown.enter.prevent="$event.target.blur()"
            @blur="saveTitle"
          />
        </SheetHeader>

        <div class="space-y-6 p-6">
          <!-- Properties -->
          <dl class="grid grid-cols-[8rem_1fr] items-center gap-x-4 gap-y-3 text-sm">
            <dt class="text-muted-foreground">{{ t('globals.terms.status', 1) }}</dt>
            <dd>
              <SelectComboBox
                :model-value="String(task.status_id)"
                :items="taskMeta.statusOptions"
                :disabled="!canWrite"
                @update:model-value="(v) => save({ status_id: Number(v) })"
              />
            </dd>

            <dt class="text-muted-foreground">{{ t('globals.terms.priority', 1) }}</dt>
            <dd>
              <Select
                :model-value="task.priority"
                :disabled="!canWrite"
                @update:model-value="(v) => save({ priority: v })"
              >
                <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="p in TASK_PRIORITIES" :key="p" :value="p">
                    <TaskPriority :priority="p" />
                  </SelectItem>
                </SelectContent>
              </Select>
            </dd>

            <dt class="text-muted-foreground">{{ t('tasks.fields.assignee') }}</dt>
            <dd>
              <SelectAgentCombobox
                :model-value="task.assigned_user_id ? String(task.assigned_user_id) : 'none'"
                include-none
                current-user-first
                exclude-ai-assistants
                :disabled="!canWrite"
                @update:model-value="(v) => save({ assigned_user_id: v && v !== 'none' ? Number(v) : null })"
              />
            </dd>

            <dt class="text-muted-foreground">{{ t('tasks.fields.project') }}</dt>
            <dd>
              <SelectComboBox
                v-if="!task.parent_task_id"
                :model-value="task.project_id ? String(task.project_id) : 'none'"
                :items="projectItems"
                :disabled="!canWrite"
                @update:model-value="(v) => save({ project_id: v && v !== 'none' ? Number(v) : null })"
              />
              <span v-else class="text-muted-foreground">{{ task.project_name || t('tasks.noProject') }}</span>
            </dd>

            <dt class="text-muted-foreground">{{ t('tasks.fields.dueDate') }}</dt>
            <dd class="flex items-center gap-2">
              <Input
                type="date"
                class="w-44"
                :model-value="task.due_date || ''"
                :disabled="!canWrite"
                @change="(e) => save({ due_date: e.target.value || null })"
              />
              <TaskDueDate :task="task" />
            </dd>

            <template v-if="task.conversation_uuid">
              <dt class="text-muted-foreground">{{ t('globals.terms.conversation', 1) }}</dt>
              <dd class="flex min-w-0 items-center gap-2">
                <router-link
                  :to="{ name: 'inbox-conversation', params: { type: 'assigned', uuid: task.conversation_uuid } }"
                  class="truncate text-primary hover:underline"
                >
                  #{{ task.conversation_reference_number }} {{ task.conversation_subject }}
                </router-link>
                <Button
                  v-if="canWrite"
                  variant="ghost"
                  size="icon"
                  class="h-6 w-6 shrink-0"
                  :title="t('tasks.unlinkConversation')"
                  @click="save({ conversation_uuid: '' })"
                >
                  <X class="h-3.5 w-3.5" />
                </Button>
              </dd>
            </template>
          </dl>

          <!-- Description -->
          <section class="space-y-2">
            <h3 class="text-sm font-medium">{{ t('globals.terms.description') }}</h3>
            <Textarea
              v-model="draftDescription"
              rows="4"
              :placeholder="t('tasks.descriptionPlaceholder')"
              :disabled="!canWrite"
              @blur="saveDescription"
            />
          </section>

          <!-- Subtasks -->
          <section class="space-y-2">
            <h3 class="flex items-center gap-2 text-sm font-medium">
              {{ t('tasks.subtasks') }}
              <span v-if="subtasks.length" class="text-xs font-normal text-muted-foreground">
                {{ doneSubtasks }}/{{ subtasks.length }}
              </span>
            </h3>
            <ul class="space-y-1">
              <li
                v-for="sub in subtasks"
                :key="sub.id"
                class="group flex items-center gap-2 rounded-md px-1 py-1 hover:bg-muted/50"
              >
                <Checkbox
                  :checked="isDone(sub)"
                  :disabled="!canWrite || !doneStatus"
                  :aria-label="t('tasks.markDone')"
                  @update:checked="toggleSubtask(sub)"
                />
                <button
                  type="button"
                  class="flex-1 truncate text-left text-sm"
                  :class="isDone(sub) ? 'text-muted-foreground line-through' : 'hover:underline'"
                  @click="emit('open', sub.id)"
                >
                  {{ sub.title }}
                </button>
                <span v-if="sub.subtask_count" class="text-xs text-muted-foreground">
                  {{ sub.subtask_done_count }}/{{ sub.subtask_count }}
                </span>
                <TaskAssignee :task="sub" avatar-only />
              </li>
            </ul>
            <form v-if="canWrite && canNest" class="flex gap-2" @submit.prevent="addSubtask">
              <Input v-model="newSubtask" :placeholder="t('tasks.addSubtask')" class="h-8" />
              <Button type="submit" size="sm" variant="secondary" :disabled="!newSubtask.trim()">
                <Plus class="h-4 w-4" />
              </Button>
            </form>
          </section>

          <!-- Comments and history -->
          <Tabs v-model="activeTab">
            <TabsList>
              <TabsTrigger value="comments">
                {{ t('tasks.comments') }}
                <span v-if="comments.length" class="ml-1 text-xs text-muted-foreground">({{ comments.length }})</span>
              </TabsTrigger>
              <TabsTrigger value="history">{{ t('tasks.history') }}</TabsTrigger>
            </TabsList>

            <TabsContent value="comments" class="space-y-4">
              <p v-if="!comments.length" class="text-sm text-muted-foreground">{{ t('tasks.noComments') }}</p>
              <article v-for="c in comments" :key="c.id" class="group flex gap-3">
                <Avatar class="h-7 w-7 shrink-0">
                  <AvatarImage :src="c.author_avatar_url || ''" />
                  <AvatarFallback class="text-[10px]">{{ (c.author_first_name || '?').charAt(0) }}</AvatarFallback>
                </Avatar>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2 text-xs">
                    <span class="font-medium">{{ [c.author_first_name, c.author_last_name].filter(Boolean).join(' ') }}</span>
                    <span class="text-muted-foreground" :title="formatFullTimestamp(new Date(c.created_at))">
                      {{ getRelativeTime(new Date(c.created_at)) }}
                    </span>
                    <button
                      v-if="c.user_id === userStore.userID || userStore.can('tasks:manage')"
                      type="button"
                      class="ml-auto hidden text-muted-foreground hover:text-destructive group-hover:block"
                      :aria-label="t('globals.messages.delete')"
                      @click="deleteComment(c)"
                    >
                      <Trash2 class="h-3.5 w-3.5" />
                    </button>
                  </div>
                  <p class="whitespace-pre-wrap break-words text-sm">{{ c.content }}</p>
                </div>
              </article>
              <form v-if="canWrite" class="space-y-2" @submit.prevent="addComment">
                <Textarea
                  v-model="newComment"
                  rows="2"
                  :placeholder="t('tasks.commentPlaceholder')"
                  @keydown.ctrl.enter.prevent="addComment"
                  @keydown.meta.enter.prevent="addComment"
                />
                <div class="flex justify-end">
                  <Button type="submit" size="sm" :disabled="!newComment.trim() || sendingComment">
                    {{ t('tasks.comment') }}
                  </Button>
                </div>
              </form>
            </TabsContent>

            <TabsContent value="history">
              <ol class="space-y-2 text-sm">
                <li v-for="a in activities" :key="a.id" class="flex gap-2">
                  <History class="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <div class="min-w-0">
                    <span>{{ describeActivity(a) }}</span>
                    <span class="ml-2 text-xs text-muted-foreground" :title="formatFullTimestamp(new Date(a.created_at))">
                      {{ getRelativeTime(new Date(a.created_at)) }}
                    </span>
                  </div>
                </li>
              </ol>
            </TabsContent>
          </Tabs>
        </div>

        <footer v-if="canDelete" class="mt-auto flex justify-end border-t border-border p-4">
          <Button variant="ghost" class="text-destructive hover:text-destructive" @click="confirmDelete = true">
            <Trash2 class="mr-2 h-4 w-4" />
            {{ t('tasks.deleteTask') }}
          </Button>
        </footer>
      </template>
    </SheetContent>
  </Sheet>

  <AlertDialog :open="confirmDelete" @update:open="confirmDelete = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ task?.subtask_count ? t('tasks.deleteWithSubtasksConfirmation') : t('tasks.deleteConfirmation') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="deleteTask">
          {{ t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CornerLeftUp, History, Plus, Trash2, X } from 'lucide-vue-next'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle
} from '@shared-ui/components/ui/sheet'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Input } from '@shared-ui/components/ui/input'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { formatFullTimestamp, getRelativeTime } from '@shared-ui/utils/datetime.js'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import TaskPriority from './TaskPriority.vue'
import TaskDueDate from './TaskDueDate.vue'
import TaskAssignee from './TaskAssignee.vue'
import { isDone, toTaskPayload } from './taskUtils.js'
import { TASK_MAX_DEPTH, TASK_PRIORITIES } from '@/constants/tasks.js'
import { useTaskMetaStore } from '@main/stores/taskMeta'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import api from '@main/api'

const props = defineProps({
  taskId: { type: Number, default: null }
})
// open: another task id to show (parent or subtask). changed: the task was saved or deleted.
const emit = defineEmits(['close', 'open', 'changed'])

const { t } = useI18n()
const emitter = useEmitter()
const taskMeta = useTaskMetaStore()
const userStore = useUserStore()

const task = ref(null)
const subtasks = ref([])
const comments = ref([])
const activities = ref([])
const depth = ref(1)
const loading = ref(false)
const draftTitle = ref('')
const draftDescription = ref('')
const newSubtask = ref('')
const newComment = ref('')
const sendingComment = ref(false)
const confirmDelete = ref(false)
const activeTab = ref('comments')

const canWrite = computed(() => userStore.can('tasks:write'))
const canDelete = computed(() => userStore.can('tasks:delete'))
const canNest = computed(() => depth.value < TASK_MAX_DEPTH)
const doneStatus = computed(() => taskMeta.statuses.find((s) => s.category === 'done'))
const doneSubtasks = computed(() => subtasks.value.filter(isDone).length)
const projectItems = computed(() => [
  { value: 'none', label: t('tasks.noProject') },
  ...taskMeta.projectOptions
])

const showError = (error) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })

const setTask = (data) => {
  task.value = data
  draftTitle.value = data.title
  draftDescription.value = data.description || ''
}

// depthOf walks up the parents to know whether subtasks can be added here.
const depthOf = async (data) => {
  let d = 1
  let parentID = data.parent_task_id
  while (parentID && d < TASK_MAX_DEPTH) {
    d++
    const resp = await api.getTask(parentID)
    parentID = resp.data.data.parent_task_id
  }
  return parentID ? d + 1 : d
}

const load = async (id) => {
  loading.value = true
  try {
    const [taskResp, commentsResp, activitiesResp] = await Promise.all([
      api.getTask(id),
      api.getTaskComments(id),
      api.getTaskActivities(id),
      taskMeta.fetchStatuses(),
      taskMeta.fetchProjects()
    ])
    const data = taskResp.data.data
    setTask(data)
    subtasks.value = data.subtasks || []
    comments.value = commentsResp.data.data
    activities.value = activitiesResp.data.data
    depth.value = await depthOf(data)
  } catch (error) {
    showError(error)
    emit('close')
  } finally {
    loading.value = false
  }
}

watch(
  () => props.taskId,
  (id) => {
    task.value = null
    newSubtask.value = ''
    newComment.value = ''
    if (id != null) load(id)
  },
  { immediate: true }
)

const refreshSide = async () => {
  const resp = await api.getTaskActivities(task.value.id)
  activities.value = resp.data.data
}

const save = async (overrides) => {
  if (!task.value) return
  try {
    const resp = await api.updateTask(task.value.id, toTaskPayload(task.value, overrides))
    setTask({ ...resp.data.data })
    emit('changed', task.value)
    refreshSide()
  } catch (error) {
    showError(error)
    setTask({ ...task.value })
  }
}

const saveTitle = () => {
  const title = draftTitle.value.trim()
  if (!title) {
    draftTitle.value = task.value.title
    return
  }
  if (title !== task.value.title) save({ title })
}

const saveDescription = () => {
  if (draftDescription.value !== (task.value.description || '')) {
    save({ description: draftDescription.value })
  }
}

const reloadSubtasks = async () => {
  const resp = await api.getTask(task.value.id)
  subtasks.value = resp.data.data.subtasks || []
  setTask({ ...task.value, ...resp.data.data })
}

const toggleSubtask = async (sub) => {
  const target = isDone(sub) ? taskMeta.defaultStatus : doneStatus.value
  if (!target) return
  try {
    await api.moveTask(sub.id, { status_id: target.id, position: sub.position })
    await reloadSubtasks()
    emit('changed', task.value)
  } catch (error) {
    showError(error)
  }
}

const addSubtask = async () => {
  const title = newSubtask.value.trim()
  if (!title) return
  try {
    await api.createTask({
      title,
      parent_task_id: task.value.id,
      status_id: taskMeta.defaultStatus?.id || 0,
      priority: 'medium'
    })
    newSubtask.value = ''
    await reloadSubtasks()
    emit('changed', task.value)
  } catch (error) {
    showError(error)
  }
}

const addComment = async () => {
  const content = newComment.value.trim()
  if (!content || sendingComment.value) return
  sendingComment.value = true
  try {
    const resp = await api.createTaskComment(task.value.id, { content })
    comments.value.push(resp.data.data)
    newComment.value = ''
    refreshSide()
  } catch (error) {
    showError(error)
  } finally {
    sendingComment.value = false
  }
}

const deleteComment = async (c) => {
  try {
    await api.deleteTaskComment(task.value.id, c.id)
    comments.value = comments.value.filter((x) => x.id !== c.id)
  } catch (error) {
    showError(error)
  }
}

const deleteTask = async () => {
  confirmDelete.value = false
  try {
    const deleted = task.value
    await api.deleteTask(deleted.id)
    emit('changed', { ...deleted, deleted: true })
    if (deleted.parent_task_id) emit('open', deleted.parent_task_id)
    else emit('close')
  } catch (error) {
    showError(error)
  }
}

const describeActivity = (a) => {
  const actor = [a.actor_first_name, a.actor_last_name].filter(Boolean).join(' ') || t('tasks.someone')
  const meta = a.meta || {}
  const blank = t('globals.terms.none')
  let from = meta.from || blank
  let to = meta.to || blank
  if (a.activity_type === 'priority_changed') {
    from = meta.from ? t(`tasks.priority.${meta.from}`) : blank
    to = meta.to ? t(`tasks.priority.${meta.to}`) : blank
  }
  return t(`tasks.activity.${a.activity_type}`, { actor, from, to })
}
</script>
