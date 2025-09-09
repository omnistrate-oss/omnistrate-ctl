package dataaccess

import (
	"context"
	"net/http"
	"time"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

type ListAuditEventsOptions struct {
	NextPageToken    string
	PageSize         *int64
	ServiceID        string
	EnvironmentType  string
	EventSourceTypes []string
	InstanceID       string
	ProductTierID    string
	StartDate        *time.Time
	EndDate          *time.Time
}

func ListAuditEvents(ctx context.Context, token string, opts *ListAuditEventsOptions) (resp *openapiclientfleet.FleetAuditEventsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.AuditEventsApiAPI.AuditEventsApiAuditEvents(ctxWithToken)

	if opts != nil {
		if opts.NextPageToken != "" {
			req = req.NextPageToken(opts.NextPageToken)
		}
		if opts.PageSize != nil {
			req = req.PageSize(*opts.PageSize)
		}
		if opts.ServiceID != "" {
			req = req.ServiceID(opts.ServiceID)
		}
		if opts.EnvironmentType != "" {
			req = req.EnvironmentType(opts.EnvironmentType)
		}
		if len(opts.EventSourceTypes) > 0 {
			req = req.EventSourceTypes(opts.EventSourceTypes)
		}
		if opts.InstanceID != "" {
			req = req.InstanceID(opts.InstanceID)
		}
		if opts.ProductTierID != "" {
			req = req.ProductTierID(opts.ProductTierID)
		}
		if opts.StartDate != nil {
			req = req.StartDate(*opts.StartDate)
		}
		if opts.EndDate != nil {
			req = req.EndDate(*opts.EndDate)
		}
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}