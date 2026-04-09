package task

import (
	"time"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceParity        RecurrenceType = "parity"
)

type Task struct {
	ID                 int64           `json:"id"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	Status             Status          `json:"status"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	RecurrenceType     *RecurrenceType `json:"recurrence_type,omitempty"`
	RecurrenceInterval *int            `json:"recurrence_interval,omitempty"`
	RecurrenceDates    []string        `json:"recurrence_dates,omitempty"` // для specific_dates
	RecurrenceEndDate  *time.Time      `json:"recurrence_end_date,omitempty"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}
