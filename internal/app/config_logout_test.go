package app

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestOIDCToSDKApp_SendsPostLogoutRedirectURI is the send-direction regression
// test for the logout_url gap: the OIDC model must forward post_logout_redirect_uri
// to the API on create/update.
func TestOIDCToSDKApp_SendsPostLogoutRedirectURI(t *testing.T) {
	ctx := context.Background()
	cfgObj, d := types.ObjectValue(OIDCConfigAttrTypes(), map[string]attr.Value{
		"redirect_uris":                    types.SetNull(types.StringType),
		"login_url":                        types.StringNull(),
		"post_logout_redirect_uri":         types.StringValue("https://app.example.com/logout"),
		"oidc_application_type":            types.StringNull(),
		"token_endpoint_auth_method":       types.StringNull(),
		"access_token_expiration_minutes":  types.Int64Null(),
		"refresh_token_expiration_minutes": types.Int64Null(),
	})
	if d.HasError() {
		t.Fatalf("build config object: %v", d)
	}
	m := &OIDCAppResourceModel{
		BaseAppModel: BaseAppModel{
			Name:         types.StringValue("oidc-test"),
			ConnectorID:  types.Int64Value(108419),
			RoleIDs:      types.SetNull(types.Int64Type),
			Provisioning: types.ObjectNull(ProvisioningAttrTypes()),
			Parameters:   types.ListNull(types.ObjectType{AttrTypes: ParameterAttrTypes()}),
		},
		Configuration: cfgObj,
		SSO:           types.ObjectNull(OIDCSSOAttrTypes()),
	}

	sdkApp, diags := m.ToSDKApp(ctx)
	if diags.HasError() {
		t.Fatalf("ToSDKApp: %v", diags)
	}
	cm, ok := sdkApp.Configuration.(map[string]interface{})
	if !ok {
		t.Fatalf("Configuration is %T, want map", sdkApp.Configuration)
	}
	if got := cm["post_logout_redirect_uri"]; got != "https://app.example.com/logout" {
		t.Errorf("post_logout_redirect_uri = %v, want https://app.example.com/logout", got)
	}
}

// TestSAMLToSDKApp_SendsLogoutURLAndRelayState is the send-direction regression
// test for SAML logout_url and the relaystate key.
func TestSAMLToSDKApp_SendsLogoutURLAndRelayState(t *testing.T) {
	ctx := context.Background()
	cfgObj, d := types.ObjectValue(SAMLConfigAttrTypes(), map[string]attr.Value{
		"signature_algorithm": types.StringNull(),
		"certificate_id":      types.Int64Null(),
		"provider_arn":        types.StringNull(),
		"acs":                 types.StringNull(),
		"audience":            types.StringValue("https://app.example.com"),
		"recipient":           types.StringNull(),
		"relaystate":          types.StringValue("https://app.example.com/relay"),
		"logout_url":          types.StringValue("https://app.example.com/slo"),
		"subdomain":           types.StringNull(),
	})
	if d.HasError() {
		t.Fatalf("build config object: %v", d)
	}
	m := &SAMLAppResourceModel{
		BaseAppModel: BaseAppModel{
			Name:         types.StringValue("saml-test"),
			ConnectorID:  types.Int64Value(110016),
			RoleIDs:      types.SetNull(types.Int64Type),
			Provisioning: types.ObjectNull(ProvisioningAttrTypes()),
			Parameters:   types.ListNull(types.ObjectType{AttrTypes: ParameterAttrTypes()}),
		},
		Configuration: cfgObj,
		SSO:           types.ObjectNull(SAMLSSOAttrTypes()),
	}

	sdkApp, diags := m.ToSDKApp(ctx)
	if diags.HasError() {
		t.Fatalf("ToSDKApp: %v", diags)
	}
	cm, ok := sdkApp.Configuration.(map[string]interface{})
	if !ok {
		t.Fatalf("Configuration is %T, want map", sdkApp.Configuration)
	}
	if got := cm["logout_url"]; got != "https://app.example.com/slo" {
		t.Errorf("logout_url = %v, want https://app.example.com/slo", got)
	}
	if got := cm["relaystate"]; got != "https://app.example.com/relay" {
		t.Errorf("relaystate = %v, want https://app.example.com/relay", got)
	}
}
