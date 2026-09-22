// Package task handles tasks: projects, configurable statuses, tasks with
// subtasks, their comments and change history.
package task

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/task/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS

	priorities = []string{models.PriorityLow, models.PriorityMedium, models.PriorityHigh, models.PriorityUrgent}
	categories = []string{models.CategoryTodo, models.CategoryInProgress, models.CategoryDone}
)

// maxDepth is how many levels a task tree can have, the root task counting as one.
const maxDepth = 3

type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetTasks           *sqlx.Stmt `query:"get-tasks"`
	GetTask            *sqlx.Stmt `query:"get-task"`
	GetTaskDepth       *sqlx.Stmt `query:"get-task-depth"`
	InsertTask         *sqlx.Stmt `query:"insert-task"`
	UpdateTask         *sqlx.Stmt `query:"update-task"`
	MoveTask           *sqlx.Stmt `query:"move-task"`
	SetSubtasksProject *sqlx.Stmt `query:"set-subtasks-project"`
	DeleteTask         *sqlx.Stmt `query:"delete-task"`

	GetProjects   *sqlx.Stmt `query:"get-projects"`
	GetProject    *sqlx.Stmt `query:"get-project"`
	InsertProject *sqlx.Stmt `query:"insert-project"`
	UpdateProject *sqlx.Stmt `query:"update-project"`
	DeleteProject *sqlx.Stmt `query:"delete-project"`

	GetStatuses      *sqlx.Stmt `query:"get-statuses"`
	GetStatus        *sqlx.Stmt `query:"get-status"`
	InsertStatus     *sqlx.Stmt `query:"insert-status"`
	UpdateStatus     *sqlx.Stmt `query:"update-status"`
	SetDefaultStatus *sqlx.Stmt `query:"set-default-status"`
	DeleteStatus     *sqlx.Stmt `query:"delete-status"`

	GetComments   *sqlx.Stmt `query:"get-comments"`
	GetComment    *sqlx.Stmt `query:"get-comment"`
	InsertComment *sqlx.Stmt `query:"insert-comment"`
	DeleteComment *sqlx.Stmt `query:"delete-comment"`

	GetActivities  *sqlx.Stmt `query:"get-activities"`
	InsertActivity *sqlx.Stmt `query:"insert-activity"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

func (m *Manager) internalError() error {
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}

func (m *Manager) notFound() error {
	return envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
}

func (m *Manager) inputError(key string) error {
	return envelope.NewError(envelope.InputError, m.i18n.T(key), nil)
}

// GetAll returns the tasks matching the filter along with the total count.
func (m *Manager) GetAll(f models.Filter, page, pageSize int) ([]models.Task, int, error) {
	var (
		tasks = make([]models.Task, 0)
		// Subtasks show up at top level only when the listing is about a
		// person or a conversation, not in a project or "all tasks" listing.
		topLevelOnly = f.ParentTaskID == 0 && f.AssignedUserID == 0 && f.ConversationID == 0
	)
	if err := m.q.GetTasks.Select(&tasks,
		f.AssignedUserID, f.Unassigned, f.ProjectID, f.NoProject, f.StatusID,
		f.Category, f.OpenOnly, f.Overdue, f.ConversationID, f.ParentTaskID,
		topLevelOnly, f.Priority, strings.TrimSpace(f.Query),
		pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		m.lo.Error("error fetching tasks", "error", err)
		return nil, 0, m.internalError()
	}
	total := 0
	if len(tasks) > 0 {
		total = tasks[0].Total
	}
	return tasks, total, nil
}

// Get returns a task by ID.
func (m *Manager) Get(id int) (models.Task, error) {
	var t models.Task
	if err := m.q.GetTask.Get(&t, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, m.notFound()
		}
		m.lo.Error("error fetching task", "id", id, "error", err)
		return t, m.internalError()
	}
	return t, nil
}

func (m *Manager) validate(in *models.TaskInput) error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", "`title`"), nil)
	}
	if len(in.Title) > 500 {
		return m.inputError("tasks.titleTooLong")
	}
	if in.Priority == "" {
		in.Priority = models.PriorityMedium
	}
	if !slices.Contains(priorities, in.Priority) {
		return m.inputError("tasks.invalidPriority")
	}
	if in.DueDate.Valid && strings.TrimSpace(in.DueDate.String) == "" {
		in.DueDate = null.String{}
	}
	return nil
}

// Create creates a task and records it in the task's history.
func (m *Manager) Create(in models.TaskInput, actorID int) (models.Task, error) {
	if err := m.validate(&in); err != nil {
		return models.Task{}, err
	}

	// Subtasks inherit their parent's project and can't go deeper than maxDepth.
	if in.ParentTaskID.Valid {
		parent, err := m.Get(in.ParentTaskID.Int)
		if err != nil {
			return models.Task{}, err
		}
		var depth int
		if err := m.q.GetTaskDepth.Get(&depth, parent.ID); err != nil {
			m.lo.Error("error fetching task depth", "error", err)
			return models.Task{}, m.internalError()
		}
		if depth >= maxDepth {
			return models.Task{}, m.inputError("tasks.maxDepthReached")
		}
		in.ProjectID = parent.ProjectID
	}

	var id int
	if err := m.q.InsertTask.Get(&id, in.Title, in.Description, in.ProjectID, in.ParentTaskID,
		in.StatusID, in.Priority, in.AssignedUserID, in.ConversationID, in.DueDate, actorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Task{}, m.inputError("tasks.invalidStatus")
		}
		if dbutil.IsForeignKeyError(err) {
			return models.Task{}, m.inputError("tasks.invalidReference")
		}
		m.lo.Error("error inserting task", "error", err)
		return models.Task{}, m.internalError()
	}
	m.recordActivity(id, actorID, models.ActivityCreated, nil)
	return m.Get(id)
}

// Update overwrites a task's fields and records what changed. The parent of a
// task is fixed at creation.
func (m *Manager) Update(id int, in models.TaskInput, actorID int) (models.Task, error) {
	if err := m.validate(&in); err != nil {
		return models.Task{}, err
	}
	prev, err := m.Get(id)
	if err != nil {
		return prev, err
	}
	if in.StatusID == 0 {
		in.StatusID = prev.StatusID
	}
	// A subtask's project follows its parent.
	if prev.ParentTaskID.Valid {
		in.ProjectID = prev.ProjectID
	}

	res, err := m.q.UpdateTask.Exec(id, in.Title, in.Description, in.ProjectID, in.ConversationID,
		in.StatusID, in.Priority, in.AssignedUserID, in.DueDate)
	if err != nil {
		if dbutil.IsForeignKeyError(err) {
			return prev, m.inputError("tasks.invalidReference")
		}
		m.lo.Error("error updating task", "id", id, "error", err)
		return prev, m.internalError()
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return prev, m.inputError("tasks.invalidStatus")
	}

	if !prev.ParentTaskID.Valid && prev.ProjectID != in.ProjectID {
		if _, err := m.q.SetSubtasksProject.Exec(id, in.ProjectID); err != nil {
			m.lo.Error("error moving subtasks to project", "id", id, "error", err)
		}
	}

	next, err := m.Get(id)
	if err != nil {
		return next, err
	}
	m.recordChanges(prev, next, actorID)
	return next, nil
}

// Move changes a task's status and its position within the status column.
func (m *Manager) Move(id, statusID int, position float64, actorID int) (models.Task, error) {
	prev, err := m.Get(id)
	if err != nil {
		return prev, err
	}
	res, err := m.q.MoveTask.Exec(id, statusID, position)
	if err != nil {
		m.lo.Error("error moving task", "id", id, "error", err)
		return prev, m.internalError()
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return prev, m.inputError("tasks.invalidStatus")
	}
	next, err := m.Get(id)
	if err != nil {
		return next, err
	}
	m.recordChanges(prev, next, actorID)
	return next, nil
}

// Delete deletes a task and, through the foreign key, all its subtasks.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteTask.Exec(id); err != nil {
		m.lo.Error("error deleting task", "id", id, "error", err)
		return m.internalError()
	}
	return nil
}

// recordChanges writes one history entry per tracked field that changed.
func (m *Manager) recordChanges(prev, next models.Task, actorID int) {
	if prev.Title != next.Title {
		m.recordActivity(next.ID, actorID, models.ActivityTitleChanged, map[string]any{"from": prev.Title, "to": next.Title})
	}
	if prev.StatusID != next.StatusID {
		m.recordActivity(next.ID, actorID, models.ActivityStatusChanged, map[string]any{"from": prev.StatusName, "to": next.StatusName})
	}
	if prev.Priority != next.Priority {
		m.recordActivity(next.ID, actorID, models.ActivityPriorityChanged, map[string]any{"from": prev.Priority, "to": next.Priority})
	}
	if prev.AssignedUser != next.AssignedUser {
		m.recordActivity(next.ID, actorID, models.ActivityAssigned, map[string]any{
			"from": fullName(prev.AssigneeFirstName, prev.AssigneeLastName),
			"to":   fullName(next.AssigneeFirstName, next.AssigneeLastName),
		})
	}
	if prev.DueDate != next.DueDate {
		m.recordActivity(next.ID, actorID, models.ActivityDueDateChanged, map[string]any{"from": prev.DueDate.String, "to": next.DueDate.String})
	}
	if prev.ProjectID != next.ProjectID {
		m.recordActivity(next.ID, actorID, models.ActivityProjectChanged, map[string]any{"from": prev.ProjectName.String, "to": next.ProjectName.String})
	}
}

// recordActivity writes a history entry. History is best effort and never
// fails the change it describes.
func (m *Manager) recordActivity(taskID, actorID int, activityType string, meta map[string]any) {
	if meta == nil {
		meta = map[string]any{}
	}
	b, err := json.Marshal(meta)
	if err != nil {
		m.lo.Error("error marshalling task activity", "error", err)
		return
	}
	var actor null.Int
	if actorID > 0 {
		actor = null.IntFrom(actorID)
	}
	if _, err := m.q.InsertActivity.Exec(taskID, actor, activityType, string(b)); err != nil {
		m.lo.Error("error inserting task activity", "task_id", taskID, "error", err)
	}
}

func fullName(first, last null.String) string {
	return strings.TrimSpace(first.String + " " + last.String)
}

// GetActivities returns a task's history, newest first.
func (m *Manager) GetActivities(taskID int) ([]models.Activity, error) {
	var out = make([]models.Activity, 0)
	if err := m.q.GetActivities.Select(&out, taskID); err != nil {
		m.lo.Error("error fetching task activities", "error", err)
		return nil, m.internalError()
	}
	return out, nil
}

// GetComments returns a task's comments, oldest first.
func (m *Manager) GetComments(taskID int) ([]models.Comment, error) {
	var out = make([]models.Comment, 0)
	if err := m.q.GetComments.Select(&out, taskID); err != nil {
		m.lo.Error("error fetching task comments", "error", err)
		return nil, m.internalError()
	}
	return out, nil
}

// AddComment adds a comment to a task.
func (m *Manager) AddComment(taskID, userID int, content string) (models.Comment, error) {
	var c models.Comment
	content = strings.TrimSpace(content)
	if content == "" {
		return c, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", "`content`"), nil)
	}
	if _, err := m.Get(taskID); err != nil {
		return c, err
	}
	var id int
	if err := m.q.InsertComment.Get(&id, taskID, userID, content); err != nil {
		m.lo.Error("error inserting task comment", "error", err)
		return c, m.internalError()
	}
	if err := m.q.GetComment.Get(&c, id); err != nil {
		m.lo.Error("error fetching task comment", "error", err)
		return c, m.internalError()
	}
	m.recordActivity(taskID, userID, models.ActivityCommented, nil)
	return c, nil
}

// GetComment returns a comment by ID.
func (m *Manager) GetComment(id int) (models.Comment, error) {
	var c models.Comment
	if err := m.q.GetComment.Get(&c, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c, m.notFound()
		}
		m.lo.Error("error fetching task comment", "error", err)
		return c, m.internalError()
	}
	return c, nil
}

// DeleteComment deletes a comment from a task.
func (m *Manager) DeleteComment(taskID, commentID int) error {
	if _, err := m.q.DeleteComment.Exec(commentID, taskID); err != nil {
		m.lo.Error("error deleting task comment", "error", err)
		return m.internalError()
	}
	return nil
}

// GetProjects returns projects with their open task counts.
func (m *Manager) GetProjects(includeArchived bool) ([]models.Project, error) {
	var out = make([]models.Project, 0)
	if err := m.q.GetProjects.Select(&out, includeArchived); err != nil {
		m.lo.Error("error fetching task projects", "error", err)
		return nil, m.internalError()
	}
	return out, nil
}

// GetProject returns a project by ID.
func (m *Manager) GetProject(id int) (models.Project, error) {
	var p models.Project
	if err := m.q.GetProject.Get(&p, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, m.notFound()
		}
		m.lo.Error("error fetching task project", "error", err)
		return p, m.internalError()
	}
	return p, nil
}

func (m *Manager) validateProject(p *models.Project) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if len(p.Name) > 140 || len(p.Description) > 2000 || len(p.Color) > 20 {
		return m.inputError("tasks.projectTooLong")
	}
	return nil
}

// CreateProject creates a project.
func (m *Manager) CreateProject(p models.Project, actorID int) (models.Project, error) {
	var out models.Project
	if err := m.validateProject(&p); err != nil {
		return out, err
	}
	if err := m.q.InsertProject.Get(&out, p.Name, p.Description, p.Color, actorID); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return out, envelope.NewError(envelope.ConflictError, m.i18n.T("tasks.projectAlreadyExists"), nil)
		}
		m.lo.Error("error inserting task project", "error", err)
		return out, m.internalError()
	}
	return out, nil
}

// UpdateProject updates a project. Archived projects are hidden from pickers
// but keep their tasks.
func (m *Manager) UpdateProject(id int, p models.Project, archived bool) (models.Project, error) {
	var out models.Project
	if err := m.validateProject(&p); err != nil {
		return out, err
	}
	if err := m.q.UpdateProject.Get(&out, id, p.Name, p.Description, p.Color, archived); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, m.notFound()
		}
		if dbutil.IsUniqueViolationError(err) {
			return out, envelope.NewError(envelope.ConflictError, m.i18n.T("tasks.projectAlreadyExists"), nil)
		}
		m.lo.Error("error updating task project", "error", err)
		return out, m.internalError()
	}
	return out, nil
}

// DeleteProject deletes a project. Its tasks stay, without a project.
func (m *Manager) DeleteProject(id int) error {
	if _, err := m.q.DeleteProject.Exec(id); err != nil {
		m.lo.Error("error deleting task project", "error", err)
		return m.internalError()
	}
	return nil
}

// GetStatuses returns statuses in board order.
func (m *Manager) GetStatuses() ([]models.Status, error) {
	var out = make([]models.Status, 0)
	if err := m.q.GetStatuses.Select(&out); err != nil {
		m.lo.Error("error fetching task statuses", "error", err)
		return nil, m.internalError()
	}
	return out, nil
}

func (m *Manager) validateStatus(s *models.Status) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if len(s.Name) > 140 || len(s.Color) > 20 {
		return m.inputError("tasks.projectTooLong")
	}
	if !slices.Contains(categories, s.Category) {
		return m.inputError("tasks.invalidCategory")
	}
	return nil
}

// CreateStatus creates a status.
func (m *Manager) CreateStatus(s models.Status) (models.Status, error) {
	var out models.Status
	if err := m.validateStatus(&s); err != nil {
		return out, err
	}
	if err := m.q.InsertStatus.Get(&out, s.Name, s.Category, s.Color, s.Position); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return out, envelope.NewError(envelope.ConflictError, m.i18n.T("tasks.statusAlreadyExists"), nil)
		}
		m.lo.Error("error inserting task status", "error", err)
		return out, m.internalError()
	}
	if s.IsDefault {
		return m.setDefaultStatus(out.ID)
	}
	return out, nil
}

// UpdateStatus updates a status. Existing tasks keep their completion dates
// even if the status category changes.
func (m *Manager) UpdateStatus(id int, s models.Status) (models.Status, error) {
	var out models.Status
	if err := m.validateStatus(&s); err != nil {
		return out, err
	}
	if err := m.q.UpdateStatus.Get(&out, id, s.Name, s.Category, s.Color, s.Position); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, m.notFound()
		}
		if dbutil.IsUniqueViolationError(err) {
			return out, envelope.NewError(envelope.ConflictError, m.i18n.T("tasks.statusAlreadyExists"), nil)
		}
		m.lo.Error("error updating task status", "error", err)
		return out, m.internalError()
	}
	if s.IsDefault && !out.IsDefault {
		return m.setDefaultStatus(out.ID)
	}
	return out, nil
}

func (m *Manager) setDefaultStatus(id int) (models.Status, error) {
	var out models.Status
	if _, err := m.q.SetDefaultStatus.Exec(id); err != nil {
		m.lo.Error("error setting default task status", "error", err)
		return out, m.internalError()
	}
	if err := m.q.GetStatus.Get(&out, id); err != nil {
		m.lo.Error("error fetching task status", "error", err)
		return out, m.internalError()
	}
	return out, nil
}

// DeleteStatus deletes a status that is neither the default nor in use.
func (m *Manager) DeleteStatus(id int) error {
	var s models.Status
	if err := m.q.GetStatus.Get(&s, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return m.notFound()
		}
		m.lo.Error("error fetching task status", "error", err)
		return m.internalError()
	}
	if s.IsDefault {
		return envelope.NewError(envelope.ConflictError, m.i18n.T("tasks.cannotDeleteDefaultStatus"), nil)
	}
	if _, err := m.q.DeleteStatus.Exec(id); err != nil {
		if dbutil.IsForeignKeyError(err) {
			return envelope.NewError(envelope.ConflictError, m.i18n.T("tasks.statusInUse"), nil)
		}
		m.lo.Error("error deleting task status", "error", err)
		return m.internalError()
	}
	return nil
}
