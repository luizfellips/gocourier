package notification

import (
	"fmt"
	"strings"
)

type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelSMS     Channel = "sms"
	ChannelPush    Channel = "push"
	ChannelWebhook Channel = "webhook"
)

func ParseChannel(s string) (Channel, error) {
	ch := Channel(strings.ToLower(strings.TrimSpace(s)))
	switch ch {
	case ChannelEmail, ChannelSMS, ChannelPush, ChannelWebhook:
		return ch, nil
	default:
		return "", fmt.Errorf("invalid channel: %s", s)
	}
}

func AllChannels() []Channel {
	return []Channel{ChannelEmail, ChannelSMS, ChannelPush, ChannelWebhook}
}
