package ai

import (
	"context"

	"github.com/viphase/sparkle/internal/domain"
)

// Provider is the minimal AI completion boundary.
type Provider interface {
	Complete(ctx context.Context, req domain.CompletionRequest) (domain.CompletionResponse, error)
	// Ping sends a minimal request to verify the provider is reachable and the
	// API key is valid. Returns nil on success.
	Ping(ctx context.Context) error
}
