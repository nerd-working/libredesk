-- name: get-tasks
-- $1 assigned user, $2 unassigned only, $3 project, $4 without project, $5 status,
-- $6 status category, $7 open only, $8 overdue only, $9 conversation, $10 parent task,
-- $11 top-level only, $12 priority, $13 search, $14 limit, $15 offset.
SELECT
    t.id, t.created_at, t.updated_at, t.uuid, t.project_id, t.parent_task_id, t.title, t.description,
    t.status_id, t.priority, t.assigned_user_id, t.conversation_id,
    to_char(t.due_date, 'YYYY-MM-DD') AS due_date, t.completed_at, t.position, t.created_by,
    p.name AS project_name, p.color AS project_color,
    s.name AS status_name, s.category AS status_category, s.color AS status_color,
    u.first_name AS assignee_first_name, u.last_name AS assignee_last_name, u.avatar_url AS assignee_avatar_url,
    c.uuid::text AS conversation_uuid, c.reference_number AS conversation_reference_number, c.subject AS conversation_subject,
    COALESCE(st.total, 0) AS subtask_count, COALESCE(st.done, 0) AS subtask_done_count,
    COUNT(*) OVER() AS total
FROM tasks t
JOIN task_statuses s ON s.id = t.status_id
LEFT JOIN task_projects p ON p.id = t.project_id
LEFT JOIN users u ON u.id = t.assigned_user_id
LEFT JOIN conversations c ON c.id = t.conversation_id
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS total, COUNT(*) FILTER (WHERE ss.category = 'done') AS done
    FROM tasks sub JOIN task_statuses ss ON ss.id = sub.status_id
    WHERE sub.parent_task_id = t.id
) st ON true
WHERE
    ($1 = 0 OR t.assigned_user_id = $1)
    AND (NOT $2 OR t.assigned_user_id IS NULL)
    AND ($3 = 0 OR t.project_id = $3)
    AND (NOT $4 OR t.project_id IS NULL)
    AND ($5 = 0 OR t.status_id = $5)
    AND ($6 = '' OR s.category::text = $6)
    AND (NOT $7 OR s.category <> 'done')
    AND (NOT $8 OR (t.due_date < CURRENT_DATE AND s.category <> 'done'))
    AND ($9 = 0 OR t.conversation_id = $9)
    AND ($10 = 0 OR t.parent_task_id = $10)
    AND (NOT $11 OR t.parent_task_id IS NULL)
    AND ($12 = '' OR t.priority::text = $12)
    AND ($13 = '' OR t.title ILIKE '%' || $13 || '%')
ORDER BY
    (s.category = 'done'), t.due_date ASC NULLS LAST, t.position, t.id
LIMIT NULLIF($14, 0) OFFSET $15;

-- name: get-task
SELECT
    t.id, t.created_at, t.updated_at, t.uuid, t.project_id, t.parent_task_id, t.title, t.description,
    t.status_id, t.priority, t.assigned_user_id, t.conversation_id,
    to_char(t.due_date, 'YYYY-MM-DD') AS due_date, t.completed_at, t.position, t.created_by,
    p.name AS project_name, p.color AS project_color,
    s.name AS status_name, s.category AS status_category, s.color AS status_color,
    u.first_name AS assignee_first_name, u.last_name AS assignee_last_name, u.avatar_url AS assignee_avatar_url,
    c.uuid::text AS conversation_uuid, c.reference_number AS conversation_reference_number, c.subject AS conversation_subject,
    (SELECT COUNT(*) FROM tasks sub WHERE sub.parent_task_id = t.id) AS subtask_count,
    (SELECT COUNT(*) FROM tasks sub JOIN task_statuses ss ON ss.id = sub.status_id
        WHERE sub.parent_task_id = t.id AND ss.category = 'done') AS subtask_done_count,
    0 AS total
FROM tasks t
JOIN task_statuses s ON s.id = t.status_id
LEFT JOIN task_projects p ON p.id = t.project_id
LEFT JOIN users u ON u.id = t.assigned_user_id
LEFT JOIN conversations c ON c.id = t.conversation_id
WHERE t.id = $1;

-- name: get-task-depth
-- Number of levels from the task up to its root, the task itself counting as one.
WITH RECURSIVE ancestors AS (
    SELECT id, parent_task_id, 1 AS depth FROM tasks WHERE id = $1
    UNION ALL
    SELECT t.id, t.parent_task_id, a.depth + 1
    FROM tasks t JOIN ancestors a ON t.id = a.parent_task_id
    WHERE a.depth < 50
)
SELECT COALESCE(MAX(depth), 0) FROM ancestors;

-- name: insert-task
-- A missing status falls back to the default one. Tasks go to the end of their kanban column.
WITH st AS (
    SELECT id, category FROM task_statuses
    WHERE id = COALESCE(NULLIF($5, 0), (SELECT id FROM task_statuses WHERE is_default LIMIT 1))
)
INSERT INTO tasks (title, description, project_id, parent_task_id, status_id, priority,
    assigned_user_id, conversation_id, due_date, completed_at, position, created_by)
SELECT $1::text, $2::text, $3::int, $4::bigint, st.id, $6::task_priority, $7::bigint, $8::bigint, $9::date,
    CASE WHEN st.category = 'done' THEN NOW() ELSE NULL END,
    COALESCE((SELECT MAX(position) FROM tasks WHERE status_id = st.id), 0) + 1,
    $10::bigint
FROM st
RETURNING id;

-- name: update-task
WITH st AS (SELECT id, category FROM task_statuses WHERE id = $6)
UPDATE tasks SET
    title = $2,
    description = $3,
    project_id = $4,
    conversation_id = $5,
    status_id = st.id,
    priority = $7::task_priority,
    assigned_user_id = $8,
    due_date = $9::date,
    completed_at = CASE WHEN st.category = 'done' THEN COALESCE(tasks.completed_at, NOW()) ELSE NULL END,
    updated_at = NOW()
FROM st
WHERE tasks.id = $1;

-- name: move-task
WITH st AS (SELECT id, category FROM task_statuses WHERE id = $2)
UPDATE tasks SET
    status_id = st.id,
    position = $3,
    completed_at = CASE WHEN st.category = 'done' THEN COALESCE(tasks.completed_at, NOW()) ELSE NULL END,
    updated_at = NOW()
FROM st
WHERE tasks.id = $1;

-- name: set-subtasks-project
-- Subtasks always live in their root task's project.
WITH RECURSIVE descendants AS (
    SELECT id FROM tasks WHERE parent_task_id = $1
    UNION ALL
    SELECT t.id FROM tasks t JOIN descendants d ON t.parent_task_id = d.id
)
UPDATE tasks SET project_id = $2, updated_at = NOW() WHERE id IN (SELECT id FROM descendants);

-- name: delete-task
DELETE FROM tasks WHERE id = $1;

-- name: get-projects
SELECT
    p.id, p.created_at, p.updated_at, p.name, p.description, p.color, p.archived_at, p.created_by,
    (SELECT COUNT(*) FROM tasks t JOIN task_statuses s ON s.id = t.status_id
        WHERE t.project_id = p.id AND s.category <> 'done') AS open_count
FROM task_projects p
WHERE ($1 OR p.archived_at IS NULL)
ORDER BY p.name;

-- name: get-project
SELECT id, created_at, updated_at, name, description, color, archived_at, created_by, 0 AS open_count
FROM task_projects WHERE id = $1;

-- name: insert-project
INSERT INTO task_projects (name, description, color, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, updated_at, name, description, color, archived_at, created_by, 0 AS open_count;

-- name: update-project
UPDATE task_projects SET
    name = $2,
    description = $3,
    color = $4,
    archived_at = CASE WHEN $5 THEN COALESCE(archived_at, NOW()) ELSE NULL END,
    updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, name, description, color, archived_at, created_by, 0 AS open_count;

-- name: delete-project
DELETE FROM task_projects WHERE id = $1;

-- name: get-statuses
SELECT id, created_at, updated_at, name, category, color, position, is_default
FROM task_statuses
ORDER BY position, id;

-- name: get-status
SELECT id, created_at, updated_at, name, category, color, position, is_default
FROM task_statuses WHERE id = $1;

-- name: insert-status
INSERT INTO task_statuses (name, category, color, position)
VALUES ($1, $2::task_status_category, $3, $4)
RETURNING id, created_at, updated_at, name, category, color, position, is_default;

-- name: update-status
UPDATE task_statuses SET
    name = $2,
    category = $3::task_status_category,
    color = $4,
    position = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, name, category, color, position, is_default;

-- name: set-default-status
-- Swaps the default in one statement, so there's always exactly one.
UPDATE task_statuses SET is_default = (id = $1) WHERE is_default OR id = $1;

-- name: delete-status
DELETE FROM task_statuses WHERE id = $1;

-- name: get-comments
SELECT
    c.id, c.created_at, c.updated_at, c.task_id, c.user_id, c.content,
    u.first_name AS author_first_name, u.last_name AS author_last_name, u.avatar_url AS author_avatar_url
FROM task_comments c
LEFT JOIN users u ON u.id = c.user_id
WHERE c.task_id = $1
ORDER BY c.created_at, c.id;

-- name: insert-comment
INSERT INTO task_comments (task_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING id;

-- name: get-comment
SELECT
    c.id, c.created_at, c.updated_at, c.task_id, c.user_id, c.content,
    u.first_name AS author_first_name, u.last_name AS author_last_name, u.avatar_url AS author_avatar_url
FROM task_comments c
LEFT JOIN users u ON u.id = c.user_id
WHERE c.id = $1;

-- name: delete-comment
DELETE FROM task_comments WHERE id = $1 AND task_id = $2;

-- name: get-activities
SELECT
    a.id, a.created_at, a.task_id, a.actor_id, a.activity_type, a.meta,
    u.first_name AS actor_first_name, u.last_name AS actor_last_name
FROM task_activities a
LEFT JOIN users u ON u.id = a.actor_id
WHERE a.task_id = $1
ORDER BY a.created_at DESC, a.id DESC;

-- name: insert-activity
INSERT INTO task_activities (task_id, actor_id, activity_type, meta)
VALUES ($1, $2, $3, $4);
