import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import api from '@main/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'

// Task statuses and projects, shared by the tasks pages, the sidebar and the
// conversation panel.
export const useTaskMetaStore = defineStore('taskMeta', () => {
  const statuses = ref([])
  const projects = ref([])
  const statusesLoaded = ref(false)
  const projectsLoaded = ref(false)
  const emitter = useEmitter()

  const showError = (error) =>
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })

  const fetchStatuses = async (force = false) => {
    if (statusesLoaded.value && !force) return
    try {
      const resp = await api.getTaskStatuses()
      statuses.value = resp.data.data
      statusesLoaded.value = true
    } catch (error) {
      showError(error)
    }
  }

  const fetchProjects = async (force = false) => {
    if (projectsLoaded.value && !force) return
    try {
      const resp = await api.getTaskProjects()
      projects.value = resp.data.data
      projectsLoaded.value = true
    } catch (error) {
      showError(error)
    }
  }

  const defaultStatus = computed(
    () => statuses.value.find((s) => s.is_default) || statuses.value[0] || null
  )
  const statusById = (id) => statuses.value.find((s) => s.id === id)
  const projectById = (id) => projects.value.find((p) => p.id === id)

  const statusOptions = computed(() =>
    statuses.value.map((s) => ({ label: s.name, value: String(s.id) }))
  )
  const projectOptions = computed(() =>
    projects.value.map((p) => ({ label: p.name, value: String(p.id) }))
  )

  return {
    statuses,
    projects,
    fetchStatuses,
    fetchProjects,
    defaultStatus,
    statusById,
    projectById,
    statusOptions,
    projectOptions
  }
})
