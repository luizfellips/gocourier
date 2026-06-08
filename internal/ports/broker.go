package ports

import "context"

type MessageHandler func(ctx context.Context, subject string, data []byte, headers map[string]string) error

type MessageBroker interface {
	Publish(ctx context.Context, subject string, data []byte, headers map[string]string) error
	Subscribe(ctx context.Context, stream string, consumer string, handler MessageHandler) error
	EnsureStreams(ctx context.Context) error
	Close() error
}
