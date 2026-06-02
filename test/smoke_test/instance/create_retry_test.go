package instance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/rs/zerolog/log"
)

func executeInstanceCreateWithInventoryRetry(t *testing.T, ctx context.Context, args []string) error {
	t.Helper()

	timeout := 3 * time.Minute
	b := &backoff.ExponentialBackOff{
		InitialInterval:     10 * time.Second,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         30 * time.Second,
		MaxElapsedTime:      timeout,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	ticker := backoff.NewTicker(b)
	defer ticker.Stop()

	var lastErr error
	for range ticker.C {
		cmd.RootCmd.SetArgs(args)
		lastErr = cmd.RootCmd.ExecuteContext(ctx)
		if lastErr == nil {
			return nil
		}
		if !strings.Contains(lastErr.Error(), "target resource not found") {
			return lastErr
		}
		log.Debug().Msgf("Instance create could not find newly-built resource yet, retrying: %v", lastErr)
	}

	return lastErr
}
