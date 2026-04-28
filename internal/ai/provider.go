package ai

import (
	"context"

	"github.com/viphase/sparkle/internal/domain"
)

// Provider is the minimal AI completion boundary. Real providers land behind
// this interface in M6; M5 uses MockProvider only.
type Provider interface {
	Complete(ctx context.Context, req domain.CompletionRequest) (domain.CompletionResponse, error)
}
