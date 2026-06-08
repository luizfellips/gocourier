package notification

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Attempt struct {
	ID               string          `json:"id"`
	AttemptNumber    int             `json:"attempt_number"`
	StartedAt        time.Time       `json:"started_at"`
	FinishedAt       *time.Time      `json:"finished_at,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	ProviderResponse json.RawMessage `json:"provider_response,omitempty"`
	Success          bool            `json:"success"`
}

type Delivery struct {
	ID             string
	IdempotencyKey string
	TenantID       string
	Channel        Channel
	Priority       Priority
	Recipient      json.RawMessage
	Template       json.RawMessage
	Payload        json.RawMessage
	Status         Status
	Attempts       []Attempt
	ScheduledAt    *time.Time
	CorrelationID  string
	CausationID    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RetryCount     int
	LastError      string
}

func NewDeliveryFromRequest(req IngestRequest, now time.Time) (*Delivery, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	ch, _ := ParseChannel(req.Channel)
	pri, _ := ParsePriority(req.Priority)

	status := StatusPending
	if req.ScheduledAt != nil && req.ScheduledAt.After(now) {
		status = StatusPending
	}

	return &Delivery{
		ID:             uuid.NewString(),
		IdempotencyKey: req.IdempotencyKey,
		TenantID:       req.Metadata.TenantID,
		Channel:        ch,
		Priority:       pri,
		Recipient:      req.Recipient,
		Template:       req.Template,
		Payload:        req.Payload,
		Status:         status,
		ScheduledAt:    req.ScheduledAt,
		CorrelationID:  req.Metadata.CorrelationID,
		CausationID:    req.Metadata.CausationID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (d *Delivery) Queue(now time.Time) error {
	if d.Status != StatusPending {
		return fmt.Errorf("cannot queue from status %s", d.Status)
	}
	d.Status = StatusQueued
	d.UpdatedAt = now
	return nil
}

func (d *Delivery) StartProcessing(now time.Time) error {
	switch d.Status {
	case StatusQueued, StatusRetrying:
		d.Status = StatusProcessing
		d.UpdatedAt = now
		return nil
	case StatusSucceeded:
		return fmt.Errorf("already succeeded")
	default:
		return fmt.Errorf("cannot process from status %s", d.Status)
	}
}

func (d *Delivery) RecordAttempt(attempt Attempt) {
	d.Attempts = append(d.Attempts, attempt)
	d.UpdatedAt = time.Now()
}

func (d *Delivery) MarkSucceeded(attempt Attempt, now time.Time) {
	attempt.Success = true
	attempt.FinishedAt = &now
	d.RecordAttempt(attempt)
	d.Status = StatusSucceeded
	d.UpdatedAt = now
	d.LastError = ""
}

func (d *Delivery) MarkRetrying(attempt Attempt, errMsg string, now time.Time) {
	attempt.Success = false
	attempt.ErrorMessage = errMsg
	attempt.FinishedAt = &now
	d.RecordAttempt(attempt)
	d.Status = StatusRetrying
	d.RetryCount++
	d.LastError = errMsg
	d.UpdatedAt = now
}

func (d *Delivery) MarkDLQ(attempt Attempt, errMsg string, now time.Time) {
	if attempt.ID != "" {
		attempt.Success = false
		attempt.ErrorMessage = errMsg
		attempt.FinishedAt = &now
		d.RecordAttempt(attempt)
	}
	d.Status = StatusDLQ
	d.LastError = errMsg
	d.UpdatedAt = now
}

func (d *Delivery) Replay(now time.Time) error {
	if d.Status != StatusDLQ && d.Status != StatusFailed {
		return fmt.Errorf("cannot replay from status %s", d.Status)
	}
	d.Status = StatusQueued
	d.UpdatedAt = now
	return nil
}

func (d *Delivery) ShouldSchedule(now time.Time) bool {
	return d.ScheduledAt != nil && d.ScheduledAt.After(now)
}

func (d *Delivery) PayloadJSON() (json.RawMessage, error) {
	if len(d.Payload) > 0 {
		return d.Payload, nil
	}
	body := map[string]json.RawMessage{
		"recipient": d.Recipient,
		"template":  d.Template,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
