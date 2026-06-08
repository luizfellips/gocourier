package notification

type Status string

const (
	StatusPending    Status = "pending"
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusRetrying   Status = "retrying"
	StatusSucceeded  Status = "succeeded"
	StatusFailed     Status = "failed"
	StatusDLQ        Status = "dlq"
	StatusCancelled  Status = "cancelled"
	StatusRejected   Status = "rejected"
)

func (s Status) IsTerminal() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusDLQ, StatusCancelled, StatusRejected:
		return true
	default:
		return false
	}
}

func (s Status) CanDispatch() bool {
	switch s {
	case StatusQueued, StatusRetrying, StatusProcessing:
		return true
	default:
		return false
	}
}
