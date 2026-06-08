package routing

import (
	"fmt"
	"strings"

	"github.com/gocourier/internal/domain/notification"
)

func NotificationSubject(prefix string, ch notification.Channel, pri notification.Priority) string {
	return fmt.Sprintf("%s.%s.%s", prefix, ch, pri)
}

func DLQSubject(ch notification.Channel) string {
	return fmt.Sprintf("dlq.%s", ch)
}

func ParseNotificationSubject(prefix, subject string) (notification.Channel, notification.Priority, error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid subject: %s", subject)
	}
	if parts[0] != prefix {
		return "", "", fmt.Errorf("unexpected stream prefix in subject: %s", subject)
	}
	ch, err := notification.ParseChannel(parts[1])
	if err != nil {
		return "", "", err
	}
	pri, err := notification.ParsePriority(parts[2])
	if err != nil {
		return "", "", err
	}
	return ch, pri, nil
}

func AllNotificationSubjects(prefix string) []string {
	var subjects []string
	for _, ch := range notification.AllChannels() {
		for _, pri := range []notification.Priority{notification.PriorityLow, notification.PriorityNormal, notification.PriorityHigh} {
			subjects = append(subjects, NotificationSubject(prefix, ch, pri))
		}
	}
	return subjects
}

func AllDLQSubjects() []string {
	var subjects []string
	for _, ch := range notification.AllChannels() {
		subjects = append(subjects, DLQSubject(ch))
	}
	return subjects
}
