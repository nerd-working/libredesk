<template>
  <div class="flex h-full min-h-0 flex-col">
    <!-- Header -->
    <div class="flex h-12 items-center gap-3 px-2">
      <SidebarTrigger class="cursor-pointer" />
      <span
        v-if="project"
        class="h-3 w-3 shrink-0 rounded-full border border-border"
        :style="{ backgroundColor: project.color || 'transparent' }"
      />
      <h1 class="truncate text-xl font-semibold">{{ title }}</h1>
      <span v-if="!loading" class="text-sm text-muted-foreground">{{ total }}</span>

      <div class="ml-auto flex items-center gap-2 pr-2">
        <div class="flex rounded-md border border-border p-0.5" role="group" :aria-label="t('tasks.layout')">
          <Button
            variant="ghost"
            size="sm"
            class="h-7 px-2"
            :class="{ 'bg-muted': layout === TASK_LAYOUT.TABLE }"
            :aria-pressed="layout === TASK_LAYOUT.TABLE"
            :title="t('tasks.tableView')"
            @click="layout = TASK_LAYOUT.TABLE"
          >
            <TableProperties class="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="h-7 px-2"
            :class="{ 'bg-muted': layout === TASK_LAYOUT.KANBAN }"
            :aria-pressed="layout === TASK_LAYOUT.KANBAN"
            :title="t('tasks.kanbanView')"
            @click="layout = TASK_LAYOUT.KANBAN"
          >
            <Columns3 class="h-4 w-4" />
          </Button>
        </div>
        <Button v-if="canWrite" size="sm" @click="createOpen = true">
          <Plus class="mr-1 h-4 w-4" />
          {{ t('tasks.newTask') }}
        </Button>
      </div>
    </div>
    <Separator />

    <!-- Filters -->
    <div class="flex flex-wrap items-center gap-2 px-4 py-2">
      <div class="relative w-56">
        <Search class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input v-model="search" :placeholder="t('tasks.searchPlaceholder')" class="h-8 pl-8" />
      </div>
      <div v-if="showAssigneeFilter" class="w-48">
        <SelectAgentCombobox
          v-model="filterAssignee"
          exclude-ai-assistants
          :prepend-items="[{ value: '', label: t('tasks.anyAssignee') }]"
          :placeholder="t('tasks.anyAssignee')"
        />
      </div>
      <Select v-model="filterStatus">
        <SelectTrigger class="h-8 w-40"><SelectValue :placeholder="t('tasks.anyStatus')" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="open">{{ t('tasks.openTasks') }}</SelectItem>
          <SelectItem value="any">{{ t('tasks.anyStatus') }}</SelectItem>
          <SelectItem v-for="s in taskMeta.statuses" :key="s.id" :value="String(s.id)">{{ s.name }}</SelectItem>
        </SelectContent>
      </Select>
      <Select v-model="filterPriority">
        <SelectTrigger class="h-8 w-36"><SelectValue :placeholder="t('tasks.anyPriority')" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="any">{{ t('tasks.anyPriority') }}</SelectItem>
          <SelectItem v-for="p in TASK_PRIORITIES" :key="p" :value="p">{{ t(`tasks.priority.${p}`) }}</SelectItem>
        </SelectContent>
      </Select>
      <Button v-if="hasFilters" variant="ghost" size="sm" class="h-8" @click="clearFilters">
        {{ t('tasks.clearFilters') }}
      </Button>
    </div>
    <Separator />

    <!-- Content -->
    <div class="min-h-0 flex-1 overflow-auto">
      <div v-if="!loading && !tasks.length" class="flex flex-col items-center gap-2 px-4 py-16 text-center">
        <ListTodo class="h-10 w-10 text-muted-foreground" />
        <p class="font-medium">{{ t('tasks.empty') }}</p>
        <p class="text-sm text-muted-foreground">{{ t('tasks.emptyHint') }}</p>
      </div>

      <TaskKanban
        v-else-if="layout === TASK_LAYOUT.KANBAN && !loading"
        :tasks="tasks"
        :statuses="kanbanStatuses"
        :show-project="scope !== 'project'"
        :can-edit="canWrite"
        :selected-id="selectedTaskId"
        @open="openTask"
        @move="onMove"
      />

      <template v-else>
        <TaskTable
          :tasks="tasks"
          :loading="loading"
          :show-project="scope !== 'project'"
          :group-by-due="scope === 'mine'"
          :selected-id="selectedTaskId"
          :refresh-key="refreshKey"
          @open="openTask"
        />
        <div v-if="tasks.length < total" class="flex justify-center p-4">
          <Button variant="outline" size="sm" :disabled="loadingMore" @click="loadMore">
            {{ t('tasks.loadMore') }}
          </Button>
        </div>
      </template>
    </div>

    <TaskDetailSheet
      :task-id="selectedTaskId"
      @close="closeTask"
      @open="openTaskId"
      @changed="onTaskChanged"
    />
    <TaskCreateDialog
      v-model:open="createOpen"
      :project-id="scope === 'project' ? projectId : null"
      :assign-to-me="scope === 'mine'"
      @created="onCreated"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useStorage, watchDebounced } from '@vueuse/core'
import { Columns3, ListTodo, Plus, Search, TableProperties } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Separator } from '@shared-ui/components/ui/separator'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import TaskTable from '@main/features/tasks/TaskTable.vue'
import TaskKanban from '@main/features/tasks/TaskKanban.vue'
import TaskDetailSheet from '@main/features/tasks/TaskDetailSheet.vue'
import TaskCreateDialog from '@main/features/tasks/TaskCreateDialog.vue'
import { TASK_LAYOUT, TASK_PRIORITIES } from '@main/constants/tasks.js'
import { useTaskMetaStore } from '@main/stores/taskMeta'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import api from '@main/api'

const props = defineProps({
  scope: { type: String, required: true },
  projectId: { type: Number, default: null }
})

const PAGE_SIZE = 100

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const taskMeta = useTaskMetaStore()
const userStore = useUserStore()

const layout = useStorage('tasksLayout', TASK_LAYOUT.TABLE)
const tasks = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const loadingMore = ref(false)
const createOpen = ref(false)
const refreshKey = ref(0)

const search = ref('')
const filterAssignee = ref('')
const filterStatus = ref('open')
const filterPriority = ref('any')

const canWrite = computed(() => userStore.can('tasks:write'))
const project = computed(() => (props.scope === 'project' ? taskMeta.projectById(props.projectId) : null))
const showAssigneeFilter = computed(() => !['mine', 'unassigned'].includes(props.scope))
const hasFilters = computed(
  () => !!search.value || !!filterAssignee.value || filterStatus.value !== 'open' || filterPriority.value !== 'any'
)

const title = computed(() => {
  if (project.value) return project.value.name
  return t(route.meta.titleKey || 'tasks.title')
})

// With the "open" filter the done columns stay on the board as drop targets,
// they just start empty.
const kanbanStatuses = computed(() => {
  if (filterStatus.value === 'open' || filterStatus.value === 'any') return taskMeta.statuses
  return taskMeta.statuses.filter((s) => String(s.id) === filterStatus.value)
})

const selectedTaskId = computed(() => {
  const id = Number(route.query.task)
  return Number.isInteger(id) && id > 0 ? id : null
})

const params = () => {
  const p = { page: page.value, page_size: PAGE_SIZE }
  if (props.scope === 'mine') p.assignee = 'me'
  if (props.scope === 'unassigned') p.assignee = 'none'
  if (props.scope === 'overdue') p.overdue = 1
  if (props.scope === 'project') p.project_id = props.projectId
  if (showAssigneeFilter.value && filterAssignee.value && filterAssignee.value !== 'none') {
    p.assignee = filterAssignee.value
  }
  if (filterStatus.value === 'open') p.open = 1
  else if (filterStatus.value !== 'any') p.status_id = filterStatus.value
  if (filterPriority.value !== 'any') p.priority = filterPriority.value
  if (search.value.trim()) p.q = search.value.trim()
  return p
}

const showError = (error) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })

// Responses can come back out of order when filters change quickly; only the
// latest request is applied.
let requestSeq = 0
const fetchTasks = async ({ silent = false } = {}) => {
  const seq = ++requestSeq
  page.value = 1
  if (!silent) loading.value = true
  try {
    const resp = await api.getTasks(params())
    if (seq !== requestSeq) return
    tasks.value = resp.data.data.results
    total.value = resp.data.data.total
    refreshKey.value++
  } catch (error) {
    if (seq === requestSeq) showError(error)
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

const loadMore = async () => {
  loadingMore.value = true
  page.value++
  try {
    const resp = await api.getTasks(params())
    tasks.value = [...tasks.value, ...resp.data.data.results]
    total.value = resp.data.data.total
  } catch (error) {
    page.value--
    showError(error)
  } finally {
    loadingMore.value = false
  }
}

const clearFilters = () => {
  search.value = ''
  filterAssignee.value = ''
  filterStatus.value = 'open'
  filterPriority.value = 'any'
}

onMounted(() => {
  taskMeta.fetchStatuses()
  taskMeta.fetchProjects()
})

watch(() => [props.scope, props.projectId], () => fetchTasks(), { immediate: true })
watch([filterAssignee, filterStatus, filterPriority], () => fetchTasks())
watchDebounced(search, () => fetchTasks(), { debounce: 300 })

const openTaskId = (id) => router.replace({ query: { ...route.query, task: id } })
const openTask = (task) => openTaskId(task.id)
const closeTask = () => {
  const query = { ...route.query }
  delete query.task
  router.replace({ query })
}

// Refresh quietly so the list doesn't flash while the sheet stays open, and
// keep sidebar counts current.
const onTaskChanged = () => {
  fetchTasks({ silent: true })
  taskMeta.fetchProjects(true)
}

const onCreated = (task) => {
  onTaskChanged()
  openTask(task)
}

const onMove = async ({ task, statusId, position }) => {
  const previous = { status_id: task.status_id, position: task.position }
  Object.assign(task, { status_id: statusId, position })
  try {
    const resp = await api.moveTask(task.id, { status_id: statusId, position })
    Object.assign(task, resp.data.data)
    taskMeta.fetchProjects(true)
  } catch (error) {
    Object.assign(task, previous)
    tasks.value = [...tasks.value]
    showError(error)
  }
}
</script>
