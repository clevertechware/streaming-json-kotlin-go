package export

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS task_plan_movements (
    task_plan_id text NOT NULL,
    movement_id text NOT NULL,
    code         text NOT NULL,
    label        text NOT NULL,
    PRIMARY KEY (task_plan_id, movement_id)
)`

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open tasks database: %w", err)
	}
	if _, err = pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("create task_plan_movements: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

// Reset removes the rows of a previous generation, which may have stopped on a truncated stream.
func (s *Store) Reset(ctx context.Context, taskPlanID string) error {
	if _, err := s.pool.Exec(ctx, "DELETE FROM task_plan_movements WHERE task_plan_id = $1", taskPlanID); err != nil {
		return fmt.Errorf("reset task plan %s: %w", taskPlanID, err)
	}
	return nil
}

func (s *Store) Count(ctx context.Context, taskPlanID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, "SELECT count(*) FROM task_plan_movements WHERE task_plan_id = $1", taskPlanID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count task plan %s: %w", taskPlanID, err)
	}
	return count, nil
}

func (s *Store) InsertFunc(taskPlanID string) InsertFunc {
	return func(ctx context.Context, batch []Movement) error {
		query, args := insertStatement(taskPlanID, batch)
		if _, err := s.pool.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert batch of %d movements: %w", len(batch), err)
		}
		return nil
	}
}

func insertStatement(taskPlanID string, batch []Movement) (string, []any) {
	var query strings.Builder
	query.WriteString("INSERT INTO task_plan_movements (task_plan_id, movement_id, code, label) VALUES ")
	args := make([]any, 0, len(batch)*4)
	for i, movement := range batch {
		if i > 0 {
			query.WriteByte(',')
		}
		n := i * 4
		fmt.Fprintf(&query, "($%d,$%d,$%d,$%d)", n+1, n+2, n+3, n+4)
		args = append(args, taskPlanID, movement.ID, movement.Code, movement.Label)
	}
	return query.String(), args
}
