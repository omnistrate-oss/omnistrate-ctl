package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/pkg/errors"
)

func WaitForInstanceToReachStatus(ctx context.Context, instanceID string, status instance.InstanceStatusType) error {
	timeout := time.Duration(config.GetEnvAsInteger("OMNISTRATECTL_INSTANCE_DEPLOYMENT_TIMEOUT_MINUTES", "15")) * time.Minute
	b := &backoff.ExponentialBackOff{
		InitialInterval:     10 * time.Second,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         10 * time.Second,
		MaxElapsedTime:      timeout,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	ticker := backoff.NewTicker(b)

	for range ticker.C {
		cmd.RootCmd.SetArgs([]string{"instance", "describe", instanceID})
		err := cmd.RootCmd.ExecuteContext(ctx)
		if err != nil {
			return err
		}
		currentStatus := instance.InstanceStatus

		if currentStatus == status {
			ticker.Stop()
			return nil
		}

		if currentStatus == instance.InstanceStatusFailed {
			ticker.Stop()
			return errors.New("instance deployment failed")
		}

		if currentStatus == instance.InstanceStatusCancelled {
			ticker.Stop()
			return errors.New("instance deployment cancelled")
		}
	}

	return fmt.Errorf("instance did not reach the expected status %s within the timeout period of %s", status, timeout)
}
