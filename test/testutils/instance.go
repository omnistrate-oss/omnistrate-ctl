package testutils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/pkg/errors"
)

const defaultInstanceDeploymentTimeoutMinutes = "15"

func WaitForInstanceToReachStatus(ctx context.Context, instanceID string, status instance.InstanceStatusType) error {
	return waitForInstanceState(ctx, instanceID, status, "")
}

func WaitForInstanceToReachStatusAndVersion(ctx context.Context, instanceID string, status instance.InstanceStatusType, tierVersion string) error {
	return waitForInstanceState(ctx, instanceID, status, tierVersion)
}

func waitForInstanceState(ctx context.Context, instanceID string, status instance.InstanceStatusType, tierVersion string) error {
	timeout := time.Duration(config.GetEnvAsInteger("OMNISTRATECTL_INSTANCE_DEPLOYMENT_TIMEOUT_MINUTES", defaultInstanceDeploymentTimeoutMinutes)) * time.Minute
	b := &backoff.ExponentialBackOff{
		InitialInterval:     60 * time.Second,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         600 * time.Second,
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
		currentTierVersion := instance.InstanceTierVersion

		if currentStatus == status && (tierVersion == "" || currentTierVersion == tierVersion) {
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
		if tierVersion != "" {
			log.Printf("Current instance status: %s, tier version: %s, waiting for status: %s, tier version: %s", currentStatus, currentTierVersion, status, tierVersion)
			continue
		}
		log.Printf("Current instance status: %s, waiting for status: %s", currentStatus, status)
	}

	if tierVersion != "" {
		return fmt.Errorf("instance did not reach the expected status %s and tier version %s within the timeout period of %s", status, tierVersion, timeout)
	}

	return fmt.Errorf("instance did not reach the expected status %s within the timeout period of %s", status, timeout)
}
