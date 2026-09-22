package models

import (
	"encoding/json"
	"time"

	"github.com/volatiletech/null/v9"
)

// Status categories give meaning to admin-defined statuses.
const (
	CategoryTodo       = "todo"
	CategoryInProgress = "in_progress"
	CategoryDone       = "done"
)

// Priorities a task can have.
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// Activity types recorded in a task's history.
const (
	ActivityCreated         = "created"
	ActivityTitleChanged    = "title_changed"
	ActivityStatusChanged   = "status_changed"
	ActivityPriorityChanged = "priority_changed"
	ActivityAssigned        = "assigned"
	ActivityDueDateChanged  = "due_date_changed"
	ActivityProjectChanged  = "project_changed"
	ActivityCommented       = "commented"
)

type Project struct {
	ID          int       `db:"id" json:"id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Color       string    `db:"color" json:"color"`
	ArchivedAt  null.Time `db:"archived_at" json:"archived_at"`
	CreatedBy   null.Int  `db:"created_by" json:"created_by"`
	OpenCount   int       `db:"open_count" json:"open_count"`
}

type Status struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Name      string    `db:"name" json:"name"`
	Category  string    `db:"category" json:"category"`
	Color     string    `db:"color" json:"color"`
	Position  int       `db:"position" json:"position"`
	IsDefault bool      `db:"is_default" json:"is_default"`
}

type Task struct {
	ID           int         `db:"id" json:"id"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at" json:"updated_at"`
	UUID         string      `db:"uuid" json:"uuid"`
	ProjectID    null.Int    `db:"project_id" json:"project_id"`
	ParentTaskID null.Int    `db:"parent_task_id" json:"parent_task_id"`
	Title        string      `db:"title" json:"title"`
	Description  string      `db:"description" json:"description"`
	StatusID     int         `db:"status_id" json:"status_id"`
	Priority     string      `db:"priority" json:"priority"`
	AssignedUser null.Int    `db:"assigned_user_id" json:"assigned_user_id"`
	Conversation null.Int    `db:"conversation_id" json:"conversation_id"`
	DueDate      null.String `db:"due_date" json:"due_date"`
	CompletedAt  null.Time   `db:"completed_at" json:"completed_at"`
	Position     float64     `db:"position" json:"position"`
	CreatedBy    null.Int    `db:"created_by" json:"created_by"`

	// Joined fields.
	ProjectName                 null.String `db:"project_name" json:"project_name"`
	ProjectColor                null.String `db:"project_color" json:"project_color"`
	StatusName                  string      `db:"status_name" json:"status_name"`
	StatusCategory              string      `db:"status_category" json:"status_category"`
	StatusColor                 string      `db:"status_color" json:"status_color"`
	AssigneeFirstName           null.String `db:"assignee_first_name" json:"assignee_first_name"`
	AssigneeLastName            null.String `db:"assignee_last_name" json:"assignee_last_name"`
	AssigneeAvatarURL           null.String `db:"assignee_avatar_url" json:"assignee_avatar_url"`
	ConversationUUID            null.String `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationReferenceNumber null.String `db:"conversation_reference_number" json:"conversation_reference_number"`
	ConversationSubject         null.String `db:"conversation_subject" json:"conversation_subject"`
	SubtaskCount                int         `db:"subtask_count" json:"subtask_count"`
	SubtaskDoneCount            int         `db:"subtask_done_count" json:"subtask_done_count"`
	Total                       int         `db:"total" json:"-"`
}

// TaskInput is what callers send to create or update a task. On update every
// field is written, so clients send the full task.
type TaskInput struct {
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	ProjectID        null.Int    `json:"project_id"`
	ParentTaskID     null.Int    `json:"parent_task_id"`
	StatusID         int         `json:"status_id"`
	Priority         string      `json:"priority"`
	AssignedUserID   null.Int    `json:"assigned_user_id"`
	ConversationUUID string      `json:"conversation_uuid"`
	DueDate          null.String `json:"due_date"`

	// Resolved by the handler from ConversationUUID.
	ConversationID null.Int `json:"-"`
}

// Filter narrows down task listings. Zero values mean "no filter".
type Filter struct {
	AssignedUserID int
	Unassigned     bool
	ProjectID      int
	NoProject      bool
	StatusID       int
	Category       string
	OpenOnly       bool
	Overdue        bool
	ConversationID int
	// ParentTaskID lists the subtasks of a task; when 0 only top-level tasks are listed.
	ParentTaskID int
	Priority     string
	Query        string
}

type Comment struct {
	ID              int         `db:"id" json:"id"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time   `db:"updated_at" json:"updated_at"`
	TaskID          int         `db:"task_id" json:"task_id"`
	UserID          null.Int    `db:"user_id" json:"user_id"`
	Content         string      `db:"content" json:"content"`
	AuthorFirstName null.String `db:"author_first_name" json:"author_first_name"`
	AuthorLastName  null.String `db:"author_last_name" json:"author_last_name"`
	AuthorAvatarURL null.String `db:"author_avatar_url" json:"author_avatar_url"`
}

type Activity struct {
	ID             int             `db:"id" json:"id"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	TaskID         int             `db:"task_id" json:"task_id"`
	ActorID        null.Int        `db:"actor_id" json:"actor_id"`
	ActivityType   string          `db:"activity_type" json:"activity_type"`
	Meta           json.RawMessage `db:"meta" json:"meta"`
	ActorFirstName null.String     `db:"actor_first_name" json:"actor_first_name"`
	ActorLastName  null.String     `db:"actor_last_name" json:"actor_last_name"`
}
