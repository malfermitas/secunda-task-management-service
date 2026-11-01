package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	taskdomain "secunda-task-management-service/internal/task/domain"
)

const taskColumns = `id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version`

// TaskRepository is the MySQL adapter for the task aggregate.
type TaskRepository struct {
	db *sqlx.DB
}

var _ taskdomain.Repository = (*TaskRepository)(nil)

// NewTaskRepository creates a task repository.
func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

type taskRow struct {
	ID          int64          `db:"id"`
	TeamID      int64          `db:"team_id"`
	Title       string         `db:"title"`
	Description sql.NullString `db:"description"`
	Status      string         `db:"status"`
	CreatedBy   int64          `db:"created_by"`
	AssigneeID  sql.NullInt64  `db:"assignee_id"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	ClosedAt    sql.NullTime   `db:"closed_at"`
	Version     uint64         `db:"version"`
}

func (row taskRow) toDomain() *taskdomain.Task {
	task := &taskdomain.Task{
		ID:        row.ID,
		TeamID:    row.TeamID,
		Title:     row.Title,
		Status:    taskdomain.Status(row.Status),
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Version:   uint(row.Version),
	}
	if row.Description.Valid {
		description := row.Description.String
		task.Description = &description
	}
	if row.AssigneeID.Valid {
		assigneeID := row.AssigneeID.Int64
		task.AssigneeID = &assigneeID
	}
	if row.ClosedAt.Valid {
		closedAt := row.ClosedAt.Time
		task.ClosedAt = &closedAt
	}
	return task
}

func (r *TaskRepository) Create(ctx context.Context, task *taskdomain.Task) error {
	return withTx(ctx, r.db, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(
			ctx,
			`INSERT INTO tasks
			 (team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			task.TeamID,
			task.Title,
			nullableString(task.Description),
			string(task.Status),
			task.CreatedBy,
			nullableInt64(task.AssigneeID),
			task.CreatedAt,
			task.UpdatedAt,
			nullableTime(task.ClosedAt),
			task.Version,
		)
		if err != nil {
			return err
		}

		taskID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		task.ID = taskID

		payload, err := json.Marshal(initialChanges(task))
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO task_history (task_id, changed_by, changes, created_at) VALUES (?, ?, ?, ?)`,
			task.ID,
			task.CreatedBy,
			payload,
			task.CreatedAt,
		)
		return err
	})
}

func (r *TaskRepository) GetByID(ctx context.Context, taskID int64) (*taskdomain.Task, error) {
	var row taskRow
	if err := r.db.GetContext(
		ctx,
		&row,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`,
		taskID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	task := row.toDomain()
	return task, nil
}

func (r *TaskRepository) List(ctx context.Context, filter taskdomain.Filter) (*taskdomain.Page, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	conditions := []string{"team_id = ?"}
	args := []any{filter.TeamID}

	if status, ok := filter.Status.TryUnwrap(); ok {
		conditions = append(conditions, "status = ?")
		args = append(args, string(status))
	}
	if filter.AssigneeID != nil {
		conditions = append(conditions, "assignee_id = ?")
		args = append(args, *filter.AssigneeID)
	}

	where := strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM tasks WHERE `+where, args...); err != nil {
		return nil, err
	}

	listArgs := make([]any, 0, len(args)+2)
	listArgs = append(listArgs, args...)
	listArgs = append(listArgs, filter.Limit, filter.Offset)

	rows := make([]taskRow, 0)
	if err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT `+taskColumns+` FROM tasks WHERE `+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`,
		listArgs...,
	); err != nil {
		return nil, err
	}

	tasks := make([]taskdomain.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, *row.toDomain())
	}

	return &taskdomain.Page{Items: tasks, Total: total}, nil
}

func (r *TaskRepository) Save(
	ctx context.Context,
	task *taskdomain.Task,
	expectedVersion uint,
	changedBy int64,
	changes map[string]any,
) error {
	return withTx(ctx, r.db, func(tx *sqlx.Tx) error {
		var currentVersion uint64
		if err := tx.GetContext(
			ctx,
			&currentVersion,
			`SELECT version FROM tasks WHERE id = ? FOR UPDATE`,
			task.ID,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return taskdomain.ErrTaskNotFound(task.ID)
			}
			return err
		}
		if uint(currentVersion) != expectedVersion {
			return taskdomain.ErrVersionConflict()
		}

		result, err := tx.ExecContext(
			ctx,
			`UPDATE tasks
			 SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?, updated_at = ?, version = ?
			 WHERE id = ? AND version = ?`,
			task.Title,
			nullableString(task.Description),
			string(task.Status),
			nullableInt64(task.AssigneeID),
			nullableTime(task.ClosedAt),
			task.UpdatedAt,
			task.Version,
			task.ID,
			expectedVersion,
		)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return taskdomain.ErrVersionConflict()
		}

		payload, err := json.Marshal(changes)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO task_history (task_id, changed_by, changes, created_at) VALUES (?, ?, ?, ?)`,
			task.ID,
			changedBy,
			payload,
			task.UpdatedAt,
		)
		return err
	})
}

type historyRow struct {
	ID        int64     `db:"id"`
	TaskID    int64     `db:"task_id"`
	ChangedBy int64     `db:"changed_by"`
	Changes   []byte    `db:"changes"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *TaskRepository) History(ctx context.Context, taskID int64) ([]taskdomain.HistoryEntry, error) {
	rows := make([]historyRow, 0)
	if err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT id, task_id, changed_by, changes, created_at
		 FROM task_history
		 WHERE task_id = ?
		 ORDER BY created_at ASC, id ASC`,
		taskID,
	); err != nil {
		return nil, err
	}

	entries := make([]taskdomain.HistoryEntry, 0, len(rows))
	for _, row := range rows {
		changes := make(map[string]any)
		if err := json.Unmarshal(row.Changes, &changes); err != nil {
			return nil, err
		}
		entries = append(entries, taskdomain.HistoryEntry{
			ID:        row.ID,
			TaskID:    row.TaskID,
			ChangedBy: row.ChangedBy,
			Changes:   changes,
			CreatedAt: row.CreatedAt,
		})
	}
	return entries, nil
}

func (r *TaskRepository) AddComment(ctx context.Context, comment *taskdomain.Comment) error {
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO task_comments (task_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`,
		comment.TaskID,
		comment.UserID,
		comment.Content,
		comment.CreatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	comment.ID = id
	return nil
}

type commentRow struct {
	ID        int64     `db:"id"`
	TaskID    int64     `db:"task_id"`
	UserID    int64     `db:"user_id"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
}

func (r *TaskRepository) Comments(ctx context.Context, taskID int64) ([]taskdomain.Comment, error) {
	rows := make([]commentRow, 0)
	if err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT id, task_id, user_id, content, created_at
		 FROM task_comments
		 WHERE task_id = ?
		 ORDER BY created_at ASC, id ASC`,
		taskID,
	); err != nil {
		return nil, err
	}

	comments := make([]taskdomain.Comment, 0, len(rows))
	for _, row := range rows {
		comments = append(comments, taskdomain.Comment{
			ID:        row.ID,
			TaskID:    row.TaskID,
			UserID:    row.UserID,
			Content:   row.Content,
			CreatedAt: row.CreatedAt,
		})
	}
	return comments, nil
}

func initialChanges(task *taskdomain.Task) map[string]any {
	return map[string]any{
		"title":       fieldChange(nil, task.Title),
		"description": fieldChange(nil, nullableString(task.Description)),
		"status":      fieldChange(nil, string(task.Status)),
		"assignee_id": fieldChange(nil, nullableInt64(task.AssigneeID)),
	}
}

func fieldChange(oldValue, newValue any) map[string]any {
	return map[string]any{"old": oldValue, "new": newValue}
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
