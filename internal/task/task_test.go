package task

import (
	"io"
	"testing"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/task/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func newManager(t *testing.T, name string) (*Manager, int) {
	t.Helper()
	db := testutil.NewDB(t, name)
	lo := logf.New(logf.Opts{Writer: io.Discard})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating manager: %v", err)
	}
	var agentID int
	if err := db.Get(&agentID, `INSERT INTO users (type, email, first_name, last_name) VALUES ('agent', 'tasks@example.com', 'Ana', 'Lima') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	return m, agentID
}

func statusByCategory(t *testing.T, m *Manager, category string) models.Status {
	t.Helper()
	statuses, err := m.GetStatuses()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range statuses {
		if s.Category == category {
			return s
		}
	}
	t.Fatalf("no %s status seeded", category)
	return models.Status{}
}

func errType(err error) string {
	if e, ok := err.(envelope.Error); ok {
		return e.ErrorType
	}
	return ""
}

func TestCreateUsesDefaultStatusAndRecordsHistory(t *testing.T) {
	m, agentID := newManager(t, "task_create")

	task, err := m.Create(models.TaskInput{Title: "  Call the customer  ", AssignedUserID: null.IntFrom(agentID)}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Title != "Call the customer" || task.Priority != models.PriorityMedium {
		t.Fatalf("unexpected task: %+v", task)
	}
	if task.StatusCategory != models.CategoryTodo || task.CompletedAt.Valid {
		t.Fatalf("new task should use the default open status, got %q", task.StatusCategory)
	}
	if task.AssigneeFirstName.String != "Ana" {
		t.Fatalf("assignee not joined: %+v", task)
	}

	acts, err := m.GetActivities(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(acts) != 1 || acts[0].ActivityType != models.ActivityCreated {
		t.Fatalf("want one created activity, got %+v", acts)
	}

	if _, err := m.Create(models.TaskInput{Title: " "}, agentID); errType(err) != envelope.InputError {
		t.Fatalf("empty title should be an input error, got %v", err)
	}
	if _, err := m.Create(models.TaskInput{Title: "x", Priority: "critical"}, agentID); errType(err) != envelope.InputError {
		t.Fatalf("unknown priority should be an input error, got %v", err)
	}
}

func TestDoneStatusSetsAndClearsCompletedAt(t *testing.T) {
	m, agentID := newManager(t, "task_done")
	done := statusByCategory(t, m, models.CategoryDone)
	todo := statusByCategory(t, m, models.CategoryTodo)

	task, err := m.Create(models.TaskInput{Title: "Ship it"}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	task, err = m.Move(task.ID, done.ID, 5, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if !task.CompletedAt.Valid || task.Position != 5 {
		t.Fatalf("moving to done should set completed_at and position: %+v", task)
	}

	in := models.TaskInput{Title: task.Title, StatusID: todo.ID, Priority: models.PriorityHigh}
	task, err = m.Update(task.ID, in, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if task.CompletedAt.Valid {
		t.Fatal("reopening should clear completed_at")
	}

	acts, err := m.GetActivities(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	var statusChanges, priorityChanges int
	for _, a := range acts {
		switch a.ActivityType {
		case models.ActivityStatusChanged:
			statusChanges++
		case models.ActivityPriorityChanged:
			priorityChanges++
		}
	}
	if statusChanges != 2 || priorityChanges != 1 {
		t.Fatalf("want 2 status and 1 priority changes, got %d and %d", statusChanges, priorityChanges)
	}

	if _, err := m.Move(task.ID, 999999, 1, agentID); errType(err) != envelope.InputError {
		t.Fatalf("unknown status should be an input error, got %v", err)
	}
}

func TestSubtasksInheritProjectAndStopAtMaxDepth(t *testing.T) {
	m, agentID := newManager(t, "task_subtasks")
	p1, err := m.CreateProject(models.Project{Name: "Onboarding"}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := m.CreateProject(models.Project{Name: "Billing"}, agentID)
	if err != nil {
		t.Fatal(err)
	}

	root, err := m.Create(models.TaskInput{Title: "Root", ProjectID: null.IntFrom(p1.ID)}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	// The subtask asks for another project but follows its parent.
	child, err := m.Create(models.TaskInput{Title: "Child", ParentTaskID: null.IntFrom(root.ID), ProjectID: null.IntFrom(p2.ID)}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if child.ProjectID.Int != p1.ID {
		t.Fatalf("subtask should inherit the parent's project, got %v", child.ProjectID)
	}
	grandchild, err := m.Create(models.TaskInput{Title: "Grandchild", ParentTaskID: null.IntFrom(child.ID)}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(models.TaskInput{Title: "Too deep", ParentTaskID: null.IntFrom(grandchild.ID)}, agentID); errType(err) != envelope.InputError {
		t.Fatalf("a 4th level should be rejected, got %v", err)
	}

	// Moving the root to another project moves the whole tree.
	if _, err := m.Update(root.ID, models.TaskInput{Title: "Root", ProjectID: null.IntFrom(p2.ID)}, agentID); err != nil {
		t.Fatal(err)
	}
	grandchild, err = m.Get(grandchild.ID)
	if err != nil {
		t.Fatal(err)
	}
	if grandchild.ProjectID.Int != p2.ID {
		t.Fatalf("descendants should follow the root's project, got %v", grandchild.ProjectID)
	}

	// Project listings only show top-level tasks, with subtask counts.
	tasks, total, err := m.GetAll(models.Filter{ProjectID: p2.ID}, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || tasks[0].ID != root.ID || tasks[0].SubtaskCount != 1 {
		t.Fatalf("want only the root with one subtask, got %d tasks: %+v", total, tasks)
	}

	// Deleting the root deletes its subtasks.
	if err := m.Delete(root.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(grandchild.ID); errType(err) != envelope.NotFoundError {
		t.Fatalf("subtasks should be deleted with their root, got %v", err)
	}
}

func TestAssigneeListingIncludesSubtasks(t *testing.T) {
	m, agentID := newManager(t, "task_assignee")
	root, err := m.Create(models.TaskInput{Title: "Root"}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(models.TaskInput{Title: "Mine", ParentTaskID: null.IntFrom(root.ID), AssignedUserID: null.IntFrom(agentID)}, agentID); err != nil {
		t.Fatal(err)
	}
	tasks, _, err := m.GetAll(models.Filter{AssignedUserID: agentID}, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Mine" {
		t.Fatalf("my tasks should include my subtasks, got %+v", tasks)
	}
}

func TestStatusRules(t *testing.T) {
	m, agentID := newManager(t, "task_statuses")
	todo := statusByCategory(t, m, models.CategoryTodo)

	if err := m.DeleteStatus(todo.ID); errType(err) != envelope.ConflictError {
		t.Fatalf("default status must not be deletable, got %v", err)
	}

	review, err := m.CreateStatus(models.Status{Name: "Review", Category: models.CategoryInProgress, Position: 4, IsDefault: true})
	if err != nil {
		t.Fatal(err)
	}
	if !review.IsDefault {
		t.Fatal("new status should become the default")
	}
	statuses, err := m.GetStatuses()
	if err != nil {
		t.Fatal(err)
	}
	defaults := 0
	for _, s := range statuses {
		if s.IsDefault {
			defaults++
		}
	}
	if defaults != 1 {
		t.Fatalf("want exactly one default status, got %d", defaults)
	}

	if _, err := m.Create(models.TaskInput{Title: "In review"}, agentID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.UpdateStatus(review.ID, models.Status{Name: "Review", Category: models.CategoryInProgress}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.setDefaultStatus(todo.ID); err != nil {
		t.Fatal(err)
	}
	if err := m.DeleteStatus(review.ID); errType(err) != envelope.ConflictError {
		t.Fatalf("status in use must not be deletable, got %v", err)
	}

	if _, err := m.CreateStatus(models.Status{Name: "Review", Category: models.CategoryTodo}); errType(err) != envelope.ConflictError {
		t.Fatalf("duplicate status name should conflict, got %v", err)
	}
}

func TestComments(t *testing.T) {
	m, agentID := newManager(t, "task_comments")
	task, err := m.Create(models.TaskInput{Title: "Discuss"}, agentID)
	if err != nil {
		t.Fatal(err)
	}
	c, err := m.AddComment(task.ID, agentID, "  Looks good  ")
	if err != nil {
		t.Fatal(err)
	}
	if c.Content != "Looks good" || c.AuthorFirstName.String != "Ana" {
		t.Fatalf("unexpected comment: %+v", c)
	}
	if _, err := m.AddComment(task.ID, agentID, " "); errType(err) != envelope.InputError {
		t.Fatalf("empty comment should be an input error, got %v", err)
	}
	if err := m.DeleteComment(task.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	comments, err := m.GetComments(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 0 {
		t.Fatalf("comment should be deleted, got %+v", comments)
	}
}
