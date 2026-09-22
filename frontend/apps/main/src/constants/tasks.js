export const TASK_PRIORITIES = ['low', 'medium', 'high', 'urgent']

export const TASK_PRIORITY_CLASSES = {
  low: 'text-muted-foreground',
  medium: 'text-blue-600 dark:text-blue-400',
  high: 'text-orange-600 dark:text-orange-400',
  urgent: 'text-red-600 dark:text-red-400'
}

export const TASK_STATUS_CATEGORIES = ['todo', 'in_progress', 'done']

// Mirrors maxDepth in internal/task: a root task plus two levels of subtasks.
export const TASK_MAX_DEPTH = 3

export const TASK_LAYOUT = { TABLE: 'table', KANBAN: 'kanban' }

// Sidebar scopes, each one a route under /tasks.
export const TASK_SCOPES = {
  MINE: 'mine',
  ALL: 'all',
  OVERDUE: 'overdue',
  UNASSIGNED: 'unassigned',
  PROJECT: 'project'
}
