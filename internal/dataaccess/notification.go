package dataaccess

import (
	"context"
	"net/http"
	"time"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func GetNotificationChannelEventHistory(ctx context.Context, token, channelID string, startTime, endTime *time.Time) (res *openapiclientfleet.ChannelEventHistoryResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	client := getFleetClient()
	
	request := client.NotificationsApiAPI.NotificationsApiNotificationChannelEventHistory(ctxWithToken, channelID)
	
	if startTime != nil {
		request = request.StartTime(*startTime)
	}
	if endTime != nil {
		request = request.EndTime(*endTime)
	}
	
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	
	res, r, err = request.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	
	return
}

func ReplayNotificationEvent(ctx context.Context, token, eventID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	client := getFleetClient()
	
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	
	r, err = client.NotificationsApiAPI.NotificationsApiReplayEvent(ctxWithToken, eventID).Execute()
	if err != nil {
		return handleFleetError(err)
	}
	
	return nil
}

func ListNotificationChannels(ctx context.Context, token string) (res *openapiclientfleet.ListNotificationChannelsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	client := getFleetClient()
	
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	
	res, r, err = client.NotificationsApiAPI.NotificationsApiListNotificationChannels(ctxWithToken).Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	
	return
}

func GetNotificationChannel(ctx context.Context, token, channelID string) (res *openapiclientfleet.Channel, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	client := getFleetClient()
	
	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	
	res, r, err = client.NotificationsApiAPI.NotificationsApiDescribeNotificationChannel(ctxWithToken, channelID).Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	
	return
}