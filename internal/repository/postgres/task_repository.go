package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "github.com/astralxx/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at,
		                   recurrence_type, recurrence_interval, recurrence_dates, recurrence_end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, status, created_at, updated_at,
		          recurrence_type, recurrence_interval, recurrence_dates, recurrence_end_date
	`

	var datesJSON []byte
	if task.RecurrenceDates != nil {
		var err error
		datesJSON, err = json.Marshal(task.RecurrenceDates)
		if err != nil {
			return nil, fmt.Errorf("marshal recurrence_dates: %w", err)
		}
	}

	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt,
		task.RecurrenceType, task.RecurrenceInterval, datesJSON, task.RecurrenceEndDate,
	)
	return scanTask(row)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at,
		       recurrence_type, recurrence_interval, recurrence_dates, recurrence_end_date
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}
	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
		    description = $2,
		    status = $3,
		    updated_at = $4
		WHERE id = $5
		RETURNING id, title, description, status, created_at, updated_at,
		          recurrence_type, recurrence_interval, recurrence_dates, recurrence_end_date
	`

	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status, task.UpdatedAt,
		task.ID,
	)

	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at,
		       recurrence_type, recurrence_interval, recurrence_dates, recurrence_end_date
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task                  taskdomain.Task
		status                string
		recurrenceTypePtr     *string
		recurrenceIntervalPtr *int
		recurrenceDatesJSON   []byte
		recurrenceEndPtr      *time.Time
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&recurrenceTypePtr,
		&recurrenceIntervalPtr,
		&recurrenceDatesJSON,
		&recurrenceEndPtr,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	if recurrenceTypePtr != nil {
		rt := taskdomain.RecurrenceType(*recurrenceTypePtr)
		task.RecurrenceType = &rt
	}
	task.RecurrenceInterval = recurrenceIntervalPtr
	if len(recurrenceDatesJSON) > 0 {
		var dates []string
		if err := json.Unmarshal(recurrenceDatesJSON, &dates); err == nil {
			task.RecurrenceDates = dates
		}
	}
	task.RecurrenceEndDate = recurrenceEndPtr

	return &task, nil
}

func (r *Repository) CreateMany(ctx context.Context, tasks []*taskdomain.Task) ([]*taskdomain.Task, error) {
	if len(tasks) == 0 {
		return []*taskdomain.Task{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at,
		                   recurrence_type, recurrence_interval, recurrence_dates, recurrence_end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, status, created_at, updated_at,
		          recurrence_type, recurrence_interval, recurrence_dates, recurrence_end_date
	`

	created := make([]*taskdomain.Task, 0, len(tasks))
	for _, task := range tasks {
		var datesJSON []byte
		if task.RecurrenceDates != nil {
			var err error
			datesJSON, err = json.Marshal(task.RecurrenceDates)
			if err != nil {
				return nil, fmt.Errorf("marshal recurrence_dates: %w", err)
			}
		}
		row := tx.QueryRow(ctx, query,
			task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt,
			task.RecurrenceType, task.RecurrenceInterval, datesJSON, task.RecurrenceEndDate,
		)
		createdTask, err := scanTask(row)
		if err != nil {
			return nil, err
		}
		created = append(created, createdTask)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}
