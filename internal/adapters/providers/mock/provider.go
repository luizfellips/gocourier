package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/google/uuid"
)

type Provider struct {
	channel notification.Channel
	mu      sync.Mutex
	sent    []string
}

func New(channel notification.Channel) *Provider {
	return &Provider{channel: channel}
}

func (p *Provider) Channel() notification.Channel {
	return p.channel
}

func (p *Provider) Send(ctx context.Context, d *notification.Delivery) (ports.ProviderResult, error) {
	_ = ctx
	recipient := string(d.Recipient)

	switch {
	case strings.Contains(recipient, "fail-permanent"):
		return ports.ProviderResult{}, fmt.Errorf("%w: invalid recipient", apperrors.ErrPermanent)
	case strings.Contains(recipient, "fail-transient"):
		return ports.ProviderResult{}, fmt.Errorf("%w: provider unavailable", apperrors.ErrTransient)
	case strings.Contains(recipient, "fail-circuit"):
		return ports.ProviderResult{}, fmt.Errorf("%w: circuit test failure", apperrors.ErrTransient)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, d.ID)

	resp, _ := json.Marshal(map[string]string{
		"mock":    "true",
		"channel": string(p.channel),
	})
	return ports.ProviderResult{
		ProviderMessageID: uuid.NewString(),
		Response:          resp,
	}, nil
}

func (p *Provider) SentCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sent)
}

func All() []ports.ChannelProvider {
	return []ports.ChannelProvider{
		New(notification.ChannelEmail),
		New(notification.ChannelSMS),
		New(notification.ChannelPush),
		New(notification.ChannelWebhook),
	}
}
