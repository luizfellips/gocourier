package providers

import (
	"github.com/gocourier/internal/adapters/providers/mock"
	"github.com/gocourier/internal/ports"
)

// Default returns the configured channel providers for this deployment.
func Default() []ports.ChannelProvider {
	return mock.All()
}
