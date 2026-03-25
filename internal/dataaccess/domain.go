package dataaccess

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func ListDomains(ctx context.Context, token string) (*openapiclientv1.ListSaaSPortalCustomDomainsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.SaasPortalApiAPI.SaasPortalApiListSaaSPortalCustomDomains(ctxWithToken).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleV1Error(err)
	}
	return resp, nil
}

func DeleteDomain(ctx context.Context, token, environmentType string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	r, err := apiClient.SaasPortalApiAPI.SaasPortalApiDeleteSaaSPortalCustomDomain(ctxWithToken, environmentType).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleV1Error(err)
	}
	return nil
}

func CreateDomain(ctx context.Context, token, name, description, environmentType, customDomain string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	r, err := apiClient.SaasPortalApiAPI.SaasPortalApiCreateSaaSPortalCustomDomain(ctxWithToken).
		CreateSaaSPortalCustomDomainRequest2(openapiclientv1.CreateSaaSPortalCustomDomainRequest2{
			Name:            name,
			Description:     description,
			EnvironmentType: environmentType,
			CustomDomain:    customDomain,
		}).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleV1Error(err)
	}
	return nil
}

const (
	DomainNotVerifiedWarningMsgTemplate = `
WARNING! Domain %s is not verified. Need to verify ownership before use.

Please create the following DNS section for your domain and add two CNAME records as follows.

- Type: CNAME
  Name: @
  Target: %s

- Type: CNAME
  Name: www
  Target: %s
`

	NextStepVerifyDomainMsgTemplate = `
Next step:

Verify domain ownership.

Please create the following DNS section for your domain and add two CNAME records as follows.

- Type: CNAME
  Name: @
  Target: %s

- Type: CNAME
  Name: www
  Target: %s
`
)

func PrintNextStepVerifyDomainMsg(clusterEndpoint string) {
	fmt.Println(fmt.Sprintf(NextStepVerifyDomainMsgTemplate, clusterEndpoint, clusterEndpoint))
}

func PrintDomainNotVerifiedWarningMsg(domain, clusterEndpoint string) {
	utils.PrintWarning(fmt.Sprintf(DomainNotVerifiedWarningMsgTemplate, domain, clusterEndpoint, clusterEndpoint))
}

func AskVerifyDomainIfAny(ctx context.Context) {
	token, err := config.GetToken()
	if err != nil {
		utils.PrintError(err)
		return
	}

	// List all domains
	listRes, err := ListDomains(ctx, token)
	if err != nil {
		utils.PrintError(err)
		return
	}

	// Warn if any domains are not verified
	for _, domain := range listRes.CustomDomains {
		if domain.Status == "PENDING" {
			PrintDomainNotVerifiedWarningMsg(domain.CustomDomain, domain.ClusterEndpoint)
		}
	}
}
