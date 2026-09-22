<!-- Content of the tasks panel; the <Sidebar> wrapper lives in components/sidebar/Sidebar.vue
     with the other sections so it picks up the .sidebar-secondary styles. -->
<template>
  <SidebarHeader>
    <SidebarMenu>
      <SidebarMenuItem>
        <div class="px-1">
          <span class="font-semibold text-xl">{{ t('tasks.title') }}</span>
        </div>
      </SidebarMenuItem>
    </SidebarMenu>
  </SidebarHeader>
  <SidebarContent>
    <MobileDrawerNav />
    <SidebarGroup>
      <SidebarMenu>
        <SidebarMenuItem v-for="item in scopeItems" :key="item.name">
          <SidebarMenuButton :isActive="route.name === item.name" asChild>
            <router-link :to="{ name: item.name }">
              <component :is="item.icon" />
              <span>{{ item.label }}</span>
            </router-link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroup>

    <SidebarGroup>
      <SidebarGroupLabel>{{ t('tasks.projects') }}</SidebarGroupLabel>
      <SidebarGroupAction
        v-if="canManage"
        :title="t('tasks.newProject')"
        @click="openProjectDialog(null)"
      >
        <Plus />
        <span class="sr-only">{{ t('tasks.newProject') }}</span>
      </SidebarGroupAction>
      <SidebarMenu>
        <SidebarMenuItem v-for="project in taskMeta.projects" :key="project.id">
          <SidebarMenuButton
            :isActive="route.name === 'tasks-project' && Number(route.params.projectId) === project.id"
            asChild
          >
            <router-link :to="{ name: 'tasks-project', params: { projectId: project.id } }">
              <span
                class="h-2.5 w-2.5 shrink-0 rounded-full border border-border"
                :style="{ backgroundColor: project.color || 'transparent' }"
              />
              <span class="flex-1 truncate">{{ project.name }}</span>
            </router-link>
          </SidebarMenuButton>
          <SidebarMenuBadge v-if="project.open_count" :class="{ 'mr-6': canManage }">
            {{ project.open_count }}
          </SidebarMenuBadge>
          <DropdownMenu v-if="canManage">
            <DropdownMenuTrigger asChild>
              <SidebarMenuAction showOnHover>
                <EllipsisVertical />
              </SidebarMenuAction>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="right" align="start">
              <DropdownMenuItem @click="openProjectDialog(project)">
                {{ t('globals.messages.edit') }}
              </DropdownMenuItem>
              <DropdownMenuItem @click="archiveProject(project)">
                {{ t('tasks.archiveProject') }}
              </DropdownMenuItem>
              <DropdownMenuItem
                class="text-destructive focus:text-destructive"
                @click="confirmDelete(project)"
              >
                {{ t('globals.messages.delete') }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
        <SidebarMenuItem v-if="!taskMeta.projects.length">
          <p class="px-2 py-1 text-xs text-muted-foreground">{{ t('tasks.noProjects') }}</p>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroup>
  </SidebarContent>
  <MobileDrawerFooter />

  <TaskProjectDialog v-model:open="projectDialogOpen" :project="editingProject" @saved="onProjectSaved" />

  <AlertDialog v-model:open="deleteOpen">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>{{ t('tasks.deleteProjectConfirmation') }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="deleteProject">
          {{ t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AlarmClock, EllipsisVertical, List, Plus, User, UserX } from 'lucide-vue-next'
import {
  SidebarContent,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem
} from '@shared-ui/components/ui/sidebar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
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
import MobileDrawerNav from '@main/components/sidebar/MobileDrawerNav.vue'
import MobileDrawerFooter from '@main/components/sidebar/MobileDrawerFooter.vue'
import TaskProjectDialog from './TaskProjectDialog.vue'
import { useUserStore } from '@main/stores/user'
import { useTaskMetaStore } from '@main/stores/taskMeta'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import api from '@main/api'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const userStore = useUserStore()
const taskMeta = useTaskMetaStore()

const canManage = computed(() => userStore.can('tasks:manage'))
const projectDialogOpen = ref(false)
const editingProject = ref(null)
// The dialog closes before its action runs, so the project is kept apart from the open state.
const deleteOpen = ref(false)
const projectToDelete = ref(null)

const scopeItems = computed(() => [
  { name: 'tasks-mine', icon: User, label: t('tasks.myTasks') },
  { name: 'tasks-all', icon: List, label: t('tasks.allTasks') },
  { name: 'tasks-overdue', icon: AlarmClock, label: t('tasks.overdue') },
  { name: 'tasks-unassigned', icon: UserX, label: t('tasks.unassigned') }
])

onMounted(() => taskMeta.fetchProjects())

const openProjectDialog = (project) => {
  editingProject.value = project
  projectDialogOpen.value = true
}

const onProjectSaved = (project) => {
  taskMeta.fetchProjects(true)
  if (!editingProject.value) {
    router.push({ name: 'tasks-project', params: { projectId: project.id } })
  }
}

const showError = (error) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })

const leaveIfOpen = (project) => {
  if (route.name === 'tasks-project' && Number(route.params.projectId) === project.id) {
    router.push({ name: 'tasks-all' })
  }
}

const archiveProject = async (project) => {
  try {
    await api.updateTaskProject(project.id, { ...project, archived: true })
    await taskMeta.fetchProjects(true)
    leaveIfOpen(project)
  } catch (error) {
    showError(error)
  }
}

const confirmDelete = (project) => {
  projectToDelete.value = project
  deleteOpen.value = true
}

const deleteProject = async () => {
  const project = projectToDelete.value
  deleteOpen.value = false
  if (!project) return
  try {
    await api.deleteTaskProject(project.id)
    await taskMeta.fetchProjects(true)
    leaveIfOpen(project)
  } catch (error) {
    showError(error)
  }
}
</script>
