package mysql

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/jmoiron/sqlx"

	analyticsdomain "secunda-task-management-service/internal/analytics/domain"
)

// StatsRepository is the MySQL read adapter for team analytics.
type StatsRepository struct {
	db *sqlx.DB
}

var _ analyticsdomain.Repository = (*StatsRepository)(nil)

// NewStatsRepository creates an analytics repository.
func NewStatsRepository(db *sqlx.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

const teamStatsQuery = `
WITH team_tasks AS (
    SELECT id, status, assignee_id, created_at, closed_at
    FROM tasks
    WHERE team_id = ?
),
status_counts AS (
    SELECT status, COUNT(*) AS cnt
    FROM team_tasks
    GROUP BY status
),
top_assignees AS (
    SELECT t.assignee_id, u.name, COUNT(*) AS closed_count
    FROM team_tasks t
    INNER JOIN users u ON u.id = t.assignee_id
    WHERE t.status = 'done'
      AND t.closed_at >= UTC_TIMESTAMP() - INTERVAL 30 DAY
    GROUP BY t.assignee_id, u.name
    ORDER BY closed_count DESC, t.assignee_id ASC
    LIMIT 3
),
avg_close AS (
    SELECT CAST(AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)) AS DOUBLE) AS avg_seconds
    FROM team_tasks
    WHERE closed_at IS NOT NULL
),
comments_per_task AS (
    SELECT t.id AS task_id, COUNT(c.id) AS cnt
    FROM team_tasks t
    LEFT JOIN task_comments c ON c.task_id = t.id
    GROUP BY t.id
)
SELECT JSON_OBJECT(
    'by_status', (SELECT JSON_ARRAYAGG(JSON_OBJECT('status', status, 'count', cnt)) FROM status_counts),
    'top_assignees', (SELECT JSON_ARRAYAGG(JSON_OBJECT('user_id', assignee_id, 'name', name, 'closed_count', closed_count)) FROM top_assignees),
    'avg_close_seconds', (SELECT avg_seconds FROM avg_close),
    'comments_per_task', (SELECT JSON_ARRAYAGG(JSON_OBJECT('task_id', task_id, 'count', cnt)) FROM comments_per_task)
) AS stats`

func (r *StatsRepository) TeamStats(ctx context.Context, teamID int64) (*analyticsdomain.TeamStats, error) {
	var raw []byte
	if err := r.db.GetContext(ctx, &raw, teamStatsQuery, teamID); err != nil {
		return nil, err
	}

	stats := &analyticsdomain.TeamStats{}
	if err := json.Unmarshal(raw, stats); err != nil {
		return nil, err
	}

	stats.TeamID = teamID
	if stats.ByStatus == nil {
		stats.ByStatus = []analyticsdomain.TaskStatusCount{}
	}
	if stats.TopAssignees == nil {
		stats.TopAssignees = []analyticsdomain.TopAssignee{}
	}
	if stats.CommentsPerTask == nil {
		stats.CommentsPerTask = []analyticsdomain.TaskCommentCount{}
	}

	sort.Slice(stats.ByStatus, func(i, j int) bool {
		return stats.ByStatus[i].Status < stats.ByStatus[j].Status
	})
	sort.Slice(stats.TopAssignees, func(i, j int) bool {
		if stats.TopAssignees[i].ClosedCount == stats.TopAssignees[j].ClosedCount {
			return stats.TopAssignees[i].UserID < stats.TopAssignees[j].UserID
		}
		return stats.TopAssignees[i].ClosedCount > stats.TopAssignees[j].ClosedCount
	})
	sort.Slice(stats.CommentsPerTask, func(i, j int) bool {
		return stats.CommentsPerTask[i].TaskID < stats.CommentsPerTask[j].TaskID
	})

	return stats, nil
}
