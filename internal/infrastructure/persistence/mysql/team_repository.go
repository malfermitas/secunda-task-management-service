package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	teamdomain "secunda-task-management-service/internal/team/domain"
)

// TeamRepository is the MySQL adapter for the team aggregate.
type TeamRepository struct {
	db *sqlx.DB
}

var _ teamdomain.Repository = (*TeamRepository)(nil)

// NewTeamRepository creates a team repository.
func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

type teamRow struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedBy int64     `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

type memberRow struct {
	TeamID int64  `db:"team_id"`
	UserID int64  `db:"user_id"`
	Role   string `db:"role"`
}

func (r *TeamRepository) Create(ctx context.Context, team *teamdomain.Team, ownerRole teamdomain.Role) error {
	return withTx(ctx, r.db, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(
			ctx,
			`INSERT INTO teams (name, created_by, created_at) VALUES (?, ?, ?)`,
			team.Name,
			team.CreatedBy,
			team.CreatedAt,
		)
		if err != nil {
			return err
		}

		teamID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
			teamID,
			team.CreatedBy,
			string(ownerRole),
		); err != nil {
			return err
		}

		team.ID = teamID
		return nil
	})
}

func (r *TeamRepository) ListByUser(ctx context.Context, userID int64) ([]teamdomain.Team, error) {
	rows := make([]teamRow, 0)
	if err := r.db.SelectContext(
		ctx,
		&rows,
		`SELECT t.id, t.name, t.created_by, t.created_at
		 FROM teams t
		 INNER JOIN team_members m ON m.team_id = t.id
		 WHERE m.user_id = ?
		 ORDER BY t.created_at DESC, t.id DESC`,
		userID,
	); err != nil {
		return nil, err
	}

	teams := make([]teamdomain.Team, 0, len(rows))
	for _, row := range rows {
		teams = append(teams, teamdomain.Team{
			ID:        row.ID,
			Name:      row.Name,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt,
		})
	}
	return teams, nil
}

func (r *TeamRepository) Membership(ctx context.Context, teamID, userID int64) (*teamdomain.Member, error) {
	var row memberRow
	if err := r.db.GetContext(
		ctx,
		&row,
		`SELECT team_id, user_id, role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID,
		userID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &teamdomain.Member{
		TeamID: row.TeamID,
		UserID: row.UserID,
		Role:   teamdomain.Role(row.Role),
	}, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, member *teamdomain.Member) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		member.TeamID,
		member.UserID,
		string(member.Role),
	)
	if isDuplicateKey(err) {
		return teamdomain.ErrAlreadyMember(member.TeamID, member.UserID)
	}
	return err
}
