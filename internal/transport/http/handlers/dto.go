package handlers

import (
	"time"

	taskdomain "github.com/astralxx/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title              string                     `json:"title"`
	Description        string                     `json:"description"`
	Status             taskdomain.Status          `json:"status"`
	RecurrenceType     *taskdomain.RecurrenceType `json:"recurrence_type,omitempty"`
	RecurrenceInterval *int                       `json:"recurrence_interval,omitempty"`
	RecurrenceDates    []string                   `json:"recurrence_dates,omitempty"`
	RecurrenceEndDate  *time.Time                 `json:"recurrence_end_date,omitempty"`
}

type taskDTO struct {
	ID                 int64                      `json:"id"`
	Title              string                     `json:"title"`
	Description        string                     `json:"description"`
	Status             taskdomain.Status          `json:"status"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
	RecurrenceType     *taskdomain.RecurrenceType `json:"recurrence_type,omitempty"`
	RecurrenceInterval *int                       `json:"recurrence_interval,omitempty"`
	RecurrenceDates    []string                   `json:"recurrence_dates,omitempty"`
	RecurrenceEndDate  *time.Time                 `json:"recurrence_end_date,omitempty"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:                 task.ID,
		Title:              task.Title,
		Description:        task.Description,
		Status:             task.Status,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
		RecurrenceType:     task.RecurrenceType,
		RecurrenceInterval: task.RecurrenceInterval,
		RecurrenceDates:    task.RecurrenceDates,
		RecurrenceEndDate:  task.RecurrenceEndDate,
	}
}
