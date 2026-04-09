package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "github.com/astralxx/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) ([]*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()

	if normalized.RecurrenceType != nil {
		tasks, err := GenerateRecurringTasks(normalized, now)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		createdTasks, err := s.repo.CreateMany(ctx, tasks)
		if err != nil {
			return nil, err
		}
		return createdTasks, nil
	}

	task := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	created, err := s.repo.Create(ctx, task)
	if err != nil {
		return nil, err
	}
	return []*taskdomain.Task{created}, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}
	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}
	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}
	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.RecurrenceType != nil {
		switch *input.RecurrenceType {
		case taskdomain.RecurrenceDaily:
			if input.RecurrenceInterval == nil || *input.RecurrenceInterval < 1 {
				return CreateInput{}, fmt.Errorf("%w: recurrence_interval must be >=1 for daily", ErrInvalidInput)
			}
		case taskdomain.RecurrenceMonthly:
			if input.RecurrenceInterval == nil || *input.RecurrenceInterval < 1 || *input.RecurrenceInterval > 30 {
				return CreateInput{}, fmt.Errorf("%w: recurrence_interval must be between 1 and 30 for monthly", ErrInvalidInput)
			}
		case taskdomain.RecurrenceParity:
			if input.RecurrenceInterval == nil || (*input.RecurrenceInterval != 0 && *input.RecurrenceInterval != 1) {
				return CreateInput{}, fmt.Errorf("%w: recurrence_interval must be 0 (even) or 1 (odd) for parity", ErrInvalidInput)
			}
		case taskdomain.RecurrenceSpecificDates:
			if len(input.RecurrenceDates) == 0 {
				return CreateInput{}, fmt.Errorf("%w: recurrence_dates required for specific_dates", ErrInvalidInput)
			}
		default:
			return CreateInput{}, fmt.Errorf("%w: unknown recurrence_type", ErrInvalidInput)
		}
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	return input, nil
}

// GenerateRecurringTasks создаёт список задач на основе шаблона периодичности
func GenerateRecurringTasks(input CreateInput, now time.Time) ([]*taskdomain.Task, error) {
	var tasks []*taskdomain.Task

	// Определяем конечную дату: если не указана, генерируем на 1 год вперёд
	endDate := input.RecurrenceEndDate
	if endDate == nil {
		oneYear := now.AddDate(1, 0, 0)
		endDate = &oneYear
	}

	switch *input.RecurrenceType {
	case taskdomain.RecurrenceDaily:
		interval := *input.RecurrenceInterval
		for d := now; !d.After(*endDate); d = d.AddDate(0, 0, interval) {
			tasks = append(tasks, newTaskFromInput(input, d))
		}

	case taskdomain.RecurrenceMonthly:
		dayOfMonth := *input.RecurrenceInterval
		// Начинаем с текущего месяца, но не раньше today
		year, month, _ := now.Date()
		current := time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC)
		if current.Before(now) {
			current = current.AddDate(0, 1, 0)
		}
		for !current.After(*endDate) {
			tasks = append(tasks, newTaskFromInput(input, current))
			current = current.AddDate(0, 1, 0)
		}

	case taskdomain.RecurrenceParity:
		// 0 = чётные дни месяца, 1 = нечётные
		parity := *input.RecurrenceInterval
		for d := now; !d.After(*endDate); d = d.AddDate(0, 0, 1) {
			if (d.Day()%2 == 0 && parity == 0) || (d.Day()%2 == 1 && parity == 1) {
				tasks = append(tasks, newTaskFromInput(input, d))
			}
		}

	case taskdomain.RecurrenceSpecificDates:
		for _, dateStr := range input.RecurrenceDates {
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return nil, fmt.Errorf("invalid date format: %s", dateStr)
			}
			date = date.UTC()
			if date.Before(now) || date.After(*endDate) {
				continue
			}
			tasks = append(tasks, newTaskFromInput(input, date))
		}
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks generated for the given recurrence rules")
	}
	return tasks, nil
}

func newTaskFromInput(input CreateInput, targetDate time.Time) *taskdomain.Task {
	return &taskdomain.Task{
		Title:              input.Title,
		Description:        input.Description,
		Status:             input.Status,
		CreatedAt:          targetDate,
		UpdatedAt:          targetDate,
		RecurrenceType:     input.RecurrenceType,
		RecurrenceInterval: input.RecurrenceInterval,
		RecurrenceDates:    input.RecurrenceDates,
		RecurrenceEndDate:  input.RecurrenceEndDate,
	}
}
