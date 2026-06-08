package routing

import (
	"testing"

	"github.com/gocourier/internal/domain/notification"
)

func TestNotificationSubject(t *testing.T) {
	subject := NotificationSubject("notifications", notification.ChannelEmail, notification.PriorityNormal)
	if subject != "notifications.email.normal" {
		t.Fatalf("unexpected subject: %s", subject)
	}
}

func TestParseNotificationSubjectRoundTrip(t *testing.T) {
	prefix := "notifications"
	ch := notification.ChannelSMS
	pri := notification.PriorityHigh
	subject := NotificationSubject(prefix, ch, pri)

	gotCh, gotPri, err := ParseNotificationSubject(prefix, subject)
	if err != nil {
		t.Fatal(err)
	}
	if gotCh != ch || gotPri != pri {
		t.Fatalf("round trip failed: %s %s", gotCh, gotPri)
	}
}

func TestParseNotificationSubjectInvalid(t *testing.T) {
	_, _, err := ParseNotificationSubject("notifications", "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDLQSubject(t *testing.T) {
	if DLQSubject(notification.ChannelWebhook) != "dlq.webhook" {
		t.Fatal("unexpected dlq subject")
	}
}

func TestAllNotificationSubjectsCount(t *testing.T) {
	subjects := AllNotificationSubjects("notifications")
	if len(subjects) != 12 {
		t.Fatalf("expected 12 subjects, got %d", len(subjects))
	}
}

func TestAllDLQSubjects(t *testing.T) {
	subjects := AllDLQSubjects()
	if len(subjects) != 4 {
		t.Fatalf("expected 4 dlq subjects, got %d", len(subjects))
	}
	if subjects[0] != "dlq.email" {
		t.Fatalf("unexpected subject: %s", subjects[0])
	}
}
