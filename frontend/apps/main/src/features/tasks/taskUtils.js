// Due dates come from the API as 'YYYY-MM-DD' in no particular timezone, so
// they are compared as plain local dates, never through Date parsing in UTC.
export const localDateString = (date = new Date()) => {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

export const isDone = (task) => task?.status_category === 'done'

export const isOverdue = (task, today = localDateString()) =>
  !!task?.due_date && !isDone(task) && task.due_date < today

export const DUE_BUCKETS = ['overdue', 'today', 'upcoming', 'noDueDate', 'done']

// dueBucket places a task in one of the "My tasks" groups.
export const dueBucket = (task, today = localDateString()) => {
  if (isDone(task)) return 'done'
  if (!task.due_date) return 'noDueDate'
  if (task.due_date < today) return 'overdue'
  if (task.due_date === today) return 'today'
  return 'upcoming'
}

// groupByDueBucket returns the non-empty groups in DUE_BUCKETS order.
export const groupByDueBucket = (tasks, today = localDateString()) => {
  const groups = Object.fromEntries(DUE_BUCKETS.map((b) => [b, []]))
  for (const task of tasks) groups[dueBucket(task, today)].push(task)
  return DUE_BUCKETS.filter((b) => groups[b].length).map((b) => ({ key: b, tasks: groups[b] }))
}

// positionBetween picks a kanban position for a card dropped between two
// neighbours, so only the moved card needs saving.
export const positionBetween = (before, after) => {
  if (before == null && after == null) return 1
  if (before == null) return after - 1
  if (after == null) return before + 1
  return (before + after) / 2
}

// toTaskPayload turns a task as returned by the API into the full body the
// update endpoint expects.
export const toTaskPayload = (task, overrides = {}) => ({
  title: task.title,
  description: task.description || '',
  project_id: task.project_id ?? null,
  status_id: task.status_id,
  priority: task.priority,
  assigned_user_id: task.assigned_user_id ?? null,
  conversation_uuid: task.conversation_uuid || '',
  due_date: task.due_date || null,
  ...overrides
})

export const assigneeName = (task) =>
  [task.assignee_first_name, task.assignee_last_name].filter(Boolean).join(' ')
