package notification

import "testing"

func TestParseChannel(t *testing.T) {
	ch, err := ParseChannel(" EMAIL ")
	if err != nil || ch != ChannelEmail {
		t.Fatalf("unexpected: %v %v", ch, err)
	}
	if _, err := ParseChannel("fax"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAllChannels(t *testing.T) {
	if len(AllChannels()) != 4 {
		t.Fatal("expected 4 channels")
	}
}

func TestParsePriority(t *testing.T) {
	p, err := ParsePriority(" HIGH ")
	if err != nil || p != PriorityHigh {
		t.Fatalf("unexpected: %v %v", p, err)
	}
	if _, err := ParsePriority("urgent"); err == nil {
		t.Fatal("expected error")
	}
}
