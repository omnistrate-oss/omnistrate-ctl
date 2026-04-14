package testutils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

const (
	defaultUpgradeTimeoutMinutes      = "15"
	defaultUpgradeStartTimeoutMinutes = "5"
)

func WaitForInstanceUpgradeToComplete(ctx context.Context, upgradePathID, instanceID, tierVersion string) error {
	if err := WaitForUpgradePathToReachTerminalState(ctx, upgradePathID); err != nil {
		return err
	}

	return WaitForInstanceToReachStatusAndVersion(ctx, instanceID, instance.InstanceStatusRunning, tierVersion)
}

func WaitForUpgradePathToReachTerminalState(ctx context.Context, upgradePathID string) error {
	token, err := config.GetToken()
	if err != nil {
		return err
	}

	serviceID, productTierID, err := getUpgradePathIdentifiers(ctx, token, upgradePathID)
	if err != nil {
		return err
	}

	timeout := time.Duration(config.GetEnvAsInteger("OMNISTRATECTL_UPGRADE_TIMEOUT_MINUTES", defaultUpgradeTimeoutMinutes)) * time.Minute
	stalledStartTimeout := time.Duration(config.GetEnvAsInteger("OMNISTRATECTL_UPGRADE_START_TIMEOUT_MINUTES", defaultUpgradeStartTimeoutMinutes)) * time.Minute
	startedDeadline := time.Now().Add(stalledStartTimeout)

	b := &backoff.ExponentialBackOff{
		InitialInterval:     30 * time.Second,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         180 * time.Second,
		MaxElapsedTime:      timeout,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	ticker := backoff.NewTicker(b)

	for range ticker.C {
		upgradePath, err := dataaccess.DescribeUpgradePath(ctx, token, serviceID, productTierID, upgradePathID)
		if err != nil {
			return err
		}

		switch upgradePath.Status {
		case model.Complete.String():
			ticker.Stop()
			return nil
		case model.Failed.String(), model.Cancelled.String(), model.Skipped.String():
			ticker.Stop()
			if upgradePath.StatusMessage != nil && *upgradePath.StatusMessage != "" {
				return fmt.Errorf("upgrade path %s ended with status %s: %s", upgradePathID, upgradePath.Status, *upgradePath.StatusMessage)
			}
			return fmt.Errorf("upgrade path %s ended with status %s", upgradePathID, upgradePath.Status)
		}

		instanceUpgrades, err := dataaccess.ListEligibleInstancesPerUpgrade(ctx, token, serviceID, productTierID, upgradePathID)
		if err != nil {
			return err
		}

		started := false
		for _, instanceUpgrade := range instanceUpgrades {
			if instanceUpgrade.UpgradeStartTime != nil || instanceUpgrade.UpgradeEndTime != nil {
				started = true
				break
			}
			if instanceUpgrade.Status == model.InProgress.String() || instanceUpgrade.Status == model.Complete.String() {
				started = true
				break
			}
		}

		if !started && time.Now().After(startedDeadline) {
			ticker.Stop()
			return fmt.Errorf(
				"upgrade path %s did not start any instance workflows within %s (status=%s pending=%d in_progress=%d completed=%d failed=%d)",
				upgradePathID,
				stalledStartTimeout,
				upgradePath.Status,
				upgradePath.PendingCount,
				upgradePath.InProgressCount,
				upgradePath.CompletedCount,
				upgradePath.FailedCount,
			)
		}

		log.Printf(
			"Current upgrade path status: %s, pending: %d, in progress: %d, completed: %d, failed: %d",
			upgradePath.Status,
			upgradePath.PendingCount,
			upgradePath.InProgressCount,
			upgradePath.CompletedCount,
			upgradePath.FailedCount,
		)
	}

	return fmt.Errorf("upgrade path %s did not reach a terminal state within the timeout period of %s", upgradePathID, timeout)
}

func getUpgradePathIdentifiers(ctx context.Context, token, upgradePathID string) (serviceID, productTierID string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("upgradepath:%s", upgradePathID))
	if err != nil {
		return "", "", err
	}

	for _, upgradePath := range searchRes.UpgradePathResults {
		if upgradePath.Id == upgradePathID {
			return upgradePath.ServiceId, upgradePath.ProductTierID, nil
		}
	}

	return "", "", fmt.Errorf("%s not found", upgradePathID)
}
