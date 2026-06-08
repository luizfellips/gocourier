package notification

import (
	"fmt"
	"strings"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

func ParsePriority(s string) (Priority, error) {
	p := Priority(strings.ToLower(strings.TrimSpace(s)))
	if p == "" {
		return PriorityNormal, nil
	}
	switch p {
	case PriorityLow, PriorityNormal, PriorityHigh:
		return p, nil
	default:
		return "", fmt.Errorf("invalid priority: %s", s)
	}
}
