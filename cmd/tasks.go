package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	tmodels "github.com/abhinavxd/libredesk/internal/task/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

// taskListResponse is a page of tasks with the total count across pages.
type taskListResponse struct {
	Results []tmodels.Task `json:"results"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
}

// pathID reads a positive integer path param.
func pathID(r *fastglue.Request, name string) (int, bool) {
	raw, _ := r.RequestCtx.UserValue(name).(string)
	id, err := strconv.Atoi(raw)
	return id, err == nil && id > 0
}

func sendBadID(r *fastglue.Request) error {
	var app = r.Context.(*App)
	return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
}

// handleGetTasks lists tasks. Filters come from the query string; `assignee`
// takes a user ID, `me` or `none`.
func handleGetTasks(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		args  = r.RequestCtx.QueryArgs()
		f     tmodels.Filter
	)
	switch assignee := string(args.Peek("assignee")); assignee {
	case "":
	case "me":
		f.AssignedUserID = auser.ID
	case "none":
		f.Unassigned = true
	default:
		f.AssignedUserID, _ = strconv.Atoi(assignee)
	}
	if project := string(args.Peek("project_id")); project == "none" {
		f.NoProject = true
	} else {
		f.ProjectID, _ = strconv.Atoi(project)
	}
	f.StatusID, _ = strconv.Atoi(string(args.Peek("status_id")))
	f.ParentTaskID, _ = strconv.Atoi(string(args.Peek("parent_task_id")))
	f.Category = string(args.Peek("category"))
	f.Priority = string(args.Peek("priority"))
	f.Query = string(args.Peek("q"))
	f.OpenOnly = args.GetBool("open")
	f.Overdue = args.GetBool("overdue")

	page, pageSize := getPagination(r)
	if args.Has("all") {
		page, pageSize = 1, 0
	}
	tasks, total, err := app.task.GetAll(f, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(taskListResponse{Results: tasks, Total: total, Page: page, PerPage: pageSize})
}

// handleGetTask returns a task along with its direct subtasks.
func handleGetTask(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	t, err := app.task.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	subtasks, _, err := app.task.GetAll(tmodels.Filter{ParentTaskID: id}, 1, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct {
		tmodels.Task
		Subtasks []tmodels.Task `json:"subtasks"`
	}{t, subtasks})
}

// decodeTaskInput reads a task from the request body and resolves the linked
// conversation. Linking a new conversation requires access to it; keeping the
// current link (current is nil on create) doesn't.
func decodeTaskInput(r *fastglue.Request, current *tmodels.Task) (tmodels.TaskInput, error) {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		in    tmodels.TaskInput
	)
	if err := r.Decode(&in, "json"); err != nil {
		return in, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil)
	}
	if in.ConversationUUID == "" {
		return in, nil
	}
	if current != nil && current.ConversationUUID.String == in.ConversationUUID {
		in.ConversationID = current.Conversation
		return in, nil
	}
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return in, err
	}
	conv, err := enforceConversationAccess(app, in.ConversationUUID, user)
	if err != nil {
		return in, err
	}
	in.ConversationID = null.IntFrom(conv.ID)
	return in, nil
}

// handleCreateTask creates a task or a subtask.
func handleCreateTask(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	in, err := decodeTaskInput(r, nil)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	t, err := app.task.Create(in, auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

// handleUpdateTask updates all the fields of a task.
func handleUpdateTask(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	current, err := app.task.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	in, err := decodeTaskInput(r, &current)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	t, err := app.task.Update(id, in, auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

// handleMoveTask moves a task to a status column and a position within it.
func handleMoveTask(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   struct {
			StatusID int     `json:"status_id"`
			Position float64 `json:"position"`
		}
	)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	t, err := app.task.Move(id, req.StatusID, req.Position, auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

// handleDeleteTask deletes a task and its subtasks.
func handleDeleteTask(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	app.lo.Info("deleting task", "task_id", id, "actor_id", auser.ID)
	if err := app.task.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetTaskActivities returns a task's change history.
func handleGetTaskActivities(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	out, err := app.task.GetActivities(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleGetTaskComments returns a task's comments.
func handleGetTaskComments(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	out, err := app.task.GetComments(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleCreateTaskComment adds a comment to a task.
func handleCreateTaskComment(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   struct {
			Content string `json:"content"`
		}
	)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	c, err := app.task.AddComment(id, auser.ID, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

// handleDeleteTaskComment deletes a comment. Only its author or an admin can.
func handleDeleteTaskComment(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, ok := pathID(r, "id")
	commentID, ok2 := pathID(r, "comment_id")
	if !ok || !ok2 {
		return sendBadID(r)
	}
	agent, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !agent.HasAdminRole() {
		c, err := app.task.GetComment(commentID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if !c.UserID.Valid || c.UserID.Int != auser.ID {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("tasks.canOnlyDeleteOwnComment"), nil, envelope.PermissionError)
		}
	}
	if err := app.task.DeleteComment(id, commentID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetConversationTasks lists every task linked to a conversation.
func handleGetConversationTasks(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	tasks, _, err := app.task.GetAll(tmodels.Filter{ConversationID: conv.ID}, 1, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tasks)
}

// handleGetTaskProjects returns projects; archived ones only with `archived=true`.
func handleGetTaskProjects(r *fastglue.Request) error {
	var app = r.Context.(*App)
	out, err := app.task.GetProjects(r.RequestCtx.QueryArgs().GetBool("archived"))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

type taskProjectRequest struct {
	tmodels.Project
	Archived bool `json:"archived"`
}

// handleCreateTaskProject creates a project.
func handleCreateTaskProject(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   taskProjectRequest
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	p, err := app.task.CreateProject(req.Project, auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(p)
}

// handleUpdateTaskProject updates or archives a project.
func handleUpdateTaskProject(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req taskProjectRequest
	)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	p, err := app.task.UpdateProject(id, req.Project, req.Archived)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(p)
}

// handleDeleteTaskProject deletes a project; its tasks remain without one.
func handleDeleteTaskProject(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	if err := app.task.DeleteProject(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetTaskStatuses returns statuses in board order.
func handleGetTaskStatuses(r *fastglue.Request) error {
	var app = r.Context.(*App)
	out, err := app.task.GetStatuses()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleCreateTaskStatus creates a status.
func handleCreateTaskStatus(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req tmodels.Status
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	s, err := app.task.CreateStatus(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(s)
}

// handleUpdateTaskStatus updates a status.
func handleUpdateTaskStatus(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req tmodels.Status
	)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	s, err := app.task.UpdateStatus(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(s)
}

// handleDeleteTaskStatus deletes a status that no task uses.
func handleDeleteTaskStatus(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, ok := pathID(r, "id")
	if !ok {
		return sendBadID(r)
	}
	if err := app.task.DeleteStatus(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
