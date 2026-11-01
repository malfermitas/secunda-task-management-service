package app

import (
	"context"
	"time"

	"secunda-task-management-service/internal/shared/errs"
	taskdomain "secunda-task-management-service/internal/task/domain"
	teamdomain "secunda-task-management-service/internal/team/domain"
)

const (
	defaultTaskLimit = 20
	maxTaskLimit     = 100
)

type service struct {
	tasks taskdomain.Repository
	teams teamdomain.Repository
	cache Cache
}

var _ Service = (*service)(nil)

// NewService creates the task application service.
func NewService(tasks taskdomain.Repository, teams teamdomain.Repository, cache Cache) Service {
	return &service{tasks: tasks, teams: teams, cache: cache}
}

func (s *service) Create(ctx context.Context, actorID int64, in CreateInput) (*taskdomain.Task, error) {
	if _, err := s.membership(ctx, in.TeamID, actorID); err != nil {
		return nil, err
	}
	if assigneeID, ok := in.AssigneeID.TryUnwrap(); ok {
		if err := s.validateAssignee(ctx, in.TeamID, assigneeID); err != nil {
			return nil, err
		}
	}

	task, err := taskdomain.NewTask(in.TeamID, in.Title, in.Description, actorID, in.AssigneeID, time.Now())
	if err != nil {
		return nil, err
	}
	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	s.invalidate(ctx, task.TeamID)
	return task, nil
}

func (s *service) List(ctx context.Context, actorID int64, in ListInput) (*TaskList, error) {
	if _, err := s.membership(ctx, in.TeamID, actorID); err != nil {
		return nil, err
	}
	if status, ok := in.Status.TryUnwrap(); ok && !status.Valid() {
		return nil, taskdomain.ErrInvalidStatus(string(status))
	}

	if in.Limit <= 0 {
		in.Limit = defaultTaskLimit
	}
	if in.Limit > maxTaskLimit {
		in.Limit = maxTaskLimit
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

	filter := taskdomain.Filter{
		TeamID:     in.TeamID,
		Status:     in.Status,
		AssigneeID: in.AssigneeID,
		Limit:      in.Limit,
		Offset:     in.Offset,
	}

	if s.cache != nil {
		if cached, ok := s.cache.GetTasks(ctx, in.TeamID); ok && cached != nil {
			return cached, nil
		}
	}

	page, err := s.tasks.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	total := page.Total
	result := &TaskList{
		Items:  page.Items,
		Total:  shared.Some(page.Total),
		Limit:  in.Limit,
		Offset: in.Offset,
	}
	if result.Items == nil {
		result.Items = []taskdomain.Task{}
	}

	if s.cache != nil {
		s.cache.SetTasks(ctx, in.TeamID, result)
	}
	return result, nil
}

func (s *service) Update(ctx context.Context, actorID, taskID int64, in UpdateInput) (*taskdomain.Task, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, taskdomain.ErrTaskNotFound(taskID)
	}

	membership, err := s.membership(ctx, task.TeamID, actorID)
	if err != nil {
		return nil, err
	}

	update := taskdomain.TaskUpdate{
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		AssigneeID:  in.AssigneeID,
	}
	if err := authorizeTaskUpdate(membership.Role, actorID, task, update); err != nil {
		return nil, err
	}

	if assigneeID, ok := update.AssigneeID.TryUnwrap(); ok {
		if err := s.validateAssignee(ctx, task.TeamID, assigneeID); err != nil {
			return nil, err
		}
	}
	if version, ok := in.Version.TryUnwrap(); ok && version != task.Version {
		return nil, taskdomain.ErrVersionConflict()
	}

	expectedVersion := task.Version
	changes, err := task.ApplyUpdate(update, time.Now())
	if err != nil {
		return nil, err
	}

	if err := s.tasks.Save(ctx, task, expectedVersion, actorID, changes); err != nil {
		return nil, err
	}

	s.invalidate(ctx, task.TeamID)
	return task, nil
}

func (s *service) History(ctx context.Context, actorID, taskID int64) ([]taskdomain.HistoryEntry, error) {
	if _, err := s.taskForActor(ctx, actorID, taskID); err != nil {
		return nil, err
	}

	entries, err := s.tasks.History(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []taskdomain.HistoryEntry{}
	}
	return entries, nil
}

func (s *service) AddComment(ctx context.Context, actorID, taskID int64, content string) (*taskdomain.Comment, error) {
	if _, err := s.taskForActor(ctx, actorID, taskID); err != nil {
		return nil, err
	}

	comment, err := taskdomain.NewComment(taskID, actorID, content, time.Now())
	if err != nil {
		return nil, err
	}
	if err := s.tasks.AddComment(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *service) ListComments(ctx context.Context, actorID, taskID int64) ([]taskdomain.Comment, error) {
	if _, err := s.taskForActor(ctx, actorID, taskID); err != nil {
		return nil, err
	}

	comments, err := s.tasks.Comments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if comments == nil {
		comments = []taskdomain.Comment{}
	}
	return comments, nil
}

func (s *service) membership(ctx context.Context, teamID, userID int64) (*teamdomain.Member, error) {
	membership, err := s.teams.Membership(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, teamdomain.ErrNotMember(teamID)
	}
	return membership, nil
}

// taskForActor loads the task and verifies it exists and the actor is a team member.
func (s *service) taskForActor(ctx context.Context, actorID, taskID int64) (*taskdomain.Task, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, taskdomain.ErrTaskNotFound(taskID)
	}
	if _, err := s.membership(ctx, task.TeamID, actorID); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *service) validateAssignee(ctx context.Context, teamID, userID int64) error {
	if userID <= 0 {
		return errs.Validation("assignee id must be positive")
	}

	membership, err := s.teams.Membership(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if membership == nil {
		return taskdomain.ErrAssigneeNotMember(userID, teamID)
	}
	return nil
}

func (s *service) invalidate(ctx context.Context, teamID int64) {
	if s.cache != nil {
		s.cache.InvalidateTeam(ctx, teamID)
	}
}

func authorizeTaskUpdate(
	role teamdomain.Role,
	actorID int64,
	task *taskdomain.Task,
	update taskdomain.TaskUpdate,
) error {
	switch {
	case role.CanEditAnyTask():
		return nil
	case task.CreatedBy == actorID:
		return nil
	case task.AssigneeID.IsSomeAnd(func(assigneeID int64) bool { return assigneeID == actorID }):
		if update.Title.IsSome() || update.Description.IsSome() || update.AssigneeID.IsSome() {
			return errs.Forbidden("assignees can only change task status")
		}
		return nil
	default:
		return errs.Forbidden("only the task creator, assignee, team owner or admin can update this task")
	}
}
