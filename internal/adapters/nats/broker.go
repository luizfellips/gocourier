package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocourier/internal/domain/routing"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/telemetry"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Broker struct {
	nc           *nats.Conn
	js           jetstream.JetStream
	streamPrefix string
	maxDeliver   int
	ackWait      time.Duration
}

type Config struct {
	URL          string
	StreamPrefix string
	MaxDeliver   int
	AckWait      time.Duration
}

func NewBroker(cfg Config) (*Broker, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.Name("gocourier"),
		nats.Timeout(10*time.Second),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	return &Broker{
		nc:           nc,
		js:           js,
		streamPrefix: cfg.StreamPrefix,
		maxDeliver:   cfg.MaxDeliver,
		ackWait:      cfg.AckWait,
	}, nil
}

func (b *Broker) EnsureStreams(ctx context.Context) error {
	notifSubjects := routing.AllNotificationSubjects(b.streamPrefix)
	_, err := b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "NOTIFICATIONS",
		Subjects:  notifSubjects,
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    7 * 24 * time.Hour,
		Discard:   jetstream.DiscardOld,
	})
	if err != nil {
		return fmt.Errorf("create notifications stream: %w", err)
	}

	dlqSubjects := routing.AllDLQSubjects()
	_, err = b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "DLQ",
		Subjects:  dlqSubjects,
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    30 * 24 * time.Hour,
	})
	if err != nil {
		return fmt.Errorf("create dlq stream: %w", err)
	}
	return nil
}

func (b *Broker) Publish(ctx context.Context, subject string, data []byte, headers map[string]string) error {
	start := time.Now()
	if headers == nil {
		headers = map[string]string{}
	}
	telemetry.InjectTrace(ctx, headers)

	msg := &nats.Msg{Subject: subject, Data: data}
	msg.Header = make(nats.Header)
	for k, v := range headers {
		msg.Header.Set(k, v)
	}
	_, err := b.js.PublishMsg(ctx, msg)
	result := "success"
	if err != nil {
		result = "error"
		if m := telemetry.MetricsGlobal(); m != nil {
			m.BrokerPublishFailed.Inc()
		}
	}
	if m := telemetry.MetricsGlobal(); m != nil {
		m.BrokerPublishTotal.WithLabelValues(subject, result).Inc()
		m.BrokerPublishDuration.WithLabelValues(subject).Observe(time.Since(start).Seconds())
	}
	return err
}

func (b *Broker) Subscribe(ctx context.Context, stream string, consumer string, handler ports.MessageHandler) error {
	var filter string
	switch stream {
	case "NOTIFICATIONS":
		filter = fmt.Sprintf("%s.>", b.streamPrefix)
	case "DLQ":
		filter = "dlq.>"
	default:
		return fmt.Errorf("unknown stream: %s", stream)
	}

	consName := sanitizeConsumerName(fmt.Sprintf("%s-%s", consumer, filter))
	cons, err := b.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:       consName,
		FilterSubject: filter,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    b.maxDeliver,
		AckWait:       b.ackWait,
	})
	if err != nil {
		return fmt.Errorf("create consumer %s: %w", consName, err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		start := time.Now()
		h := make(map[string]string)
		if msg.Headers() != nil {
			for k := range msg.Headers() {
				h[k] = msg.Headers().Get(k)
			}
		}
		msgCtx := telemetry.ExtractTrace(ctx, h)
		handlerErr := handler(msgCtx, msg.Subject(), msg.Data(), h)
		result := "success"
		if handlerErr != nil {
			result = "error"
			_ = msg.Nak()
			if m := telemetry.MetricsGlobal(); m != nil {
				m.BrokerConsumeFailed.Inc()
			}
		} else {
			_ = msg.Ack()
		}
		if m := telemetry.MetricsGlobal(); m != nil {
			m.BrokerConsumeTotal.WithLabelValues(msg.Subject(), result).Inc()
			m.BrokerConsumeDuration.WithLabelValues(msg.Subject()).Observe(time.Since(start).Seconds())
		}
	})
	if err != nil {
		return fmt.Errorf("consume %s: %w", consName, err)
	}
	go func() {
		<-ctx.Done()
		cc.Stop()
	}()
	return nil
}

func (b *Broker) Close() error {
	if b.nc != nil {
		b.nc.Close()
	}
	return nil
}

// sanitizeConsumerName produces a NATS-safe durable consumer name.
func sanitizeConsumerName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		case r == '.', r == '>', r == '*':
			b.WriteRune('-')
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "consumer"
	}
	return out
}
