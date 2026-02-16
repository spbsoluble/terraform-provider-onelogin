package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
)

// oidcSSOData represents the full OIDC SSO response from the API.
type oidcSSOData struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func extractOIDCSSO(sso interface{}) (*oidcSSOData, error) {
	if sso == nil {
		return nil, nil
	}
	b, err := json.Marshal(sso)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OIDC SSO: %w", err)
	}
	var data oidcSSOData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OIDC SSO: %w", err)
	}
	return &data, nil
}

func oidcConfigAsOptions() basetypes.ObjectAsOptions {
	return basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}
}

// OIDCConfigurationModel represents OIDC-specific configuration.
type OIDCConfigurationModel struct {
	RedirectURIs                  types.Set    `tfsdk:"redirect_uris"`
	LoginURL                      types.String `tfsdk:"login_url"`
	OidcApplicationType           types.String `tfsdk:"oidc_application_type"`
	TokenEndpointAuthMethod       types.String `tfsdk:"token_endpoint_auth_method"`
	AccessTokenExpirationMinutes  types.Int64  `tfsdk:"access_token_expiration_minutes"`
	RefreshTokenExpirationMinutes types.Int64  `tfsdk:"refresh_token_expiration_minutes"`
}

// OIDCSSOModel represents OIDC-specific SSO settings (read-only).
type OIDCSSOModel struct {
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// OIDCAppResourceModel describes the OIDC app resource data model.
type OIDCAppResourceModel struct {
	BaseAppModel
	Configuration types.Object `tfsdk:"configuration"`
	SSO           types.Object `tfsdk:"sso"`
}

// OIDCConfigAttrTypes returns the attribute types for the OIDC configuration object.
func OIDCConfigAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"redirect_uris":                    types.SetType{ElemType: types.StringType},
		"login_url":                        types.StringType,
		"oidc_application_type":            types.StringType,
		"token_endpoint_auth_method":       types.StringType,
		"access_token_expiration_minutes":  types.Int64Type,
		"refresh_token_expiration_minutes": types.Int64Type,
	}
}

// oidcAppTypeToInt converts a string OIDC application type to its integer value.
// Accepts: "Web" (0), "Native" (1), or numeric strings "0", "1".
func oidcAppTypeToInt(s string) (int, error) {
	switch s {
	case "Web", "web", "0":
		return 0, nil
	case "Native", "native", "1":
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid oidc_application_type %q: must be \"Web\", \"Native\", \"0\", or \"1\"", s)
	}
}

// oidcAppTypeFromInt converts an integer OIDC application type to its string name.
func oidcAppTypeFromInt(v int) string {
	switch v {
	case 0:
		return "Web"
	case 1:
		return "Native"
	default:
		return fmt.Sprintf("%d", v)
	}
}

// tokenAuthMethodToInt converts a string token endpoint auth method to its integer value.
// Accepts: "BASIC" (0), "POST" (1), "PKCE"/"None" (2), or numeric strings.
func tokenAuthMethodToInt(s string) (int, error) {
	switch s {
	case "BASIC", "Basic", "basic", "0":
		return 0, nil
	case "POST", "Post", "post", "1":
		return 1, nil
	case "PKCE", "pkce", "None", "none", "2":
		return 2, nil
	default:
		return 0, fmt.Errorf("invalid token_endpoint_auth_method %q: must be \"BASIC\", \"POST\", \"PKCE\", \"None\", \"0\", \"1\", or \"2\"", s)
	}
}

// tokenAuthMethodFromInt converts an integer token endpoint auth method to its string name.
func tokenAuthMethodFromInt(v int) string {
	switch v {
	case 0:
		return "BASIC"
	case 1:
		return "POST"
	case 2:
		return "PKCE"
	default:
		return fmt.Sprintf("%d", v)
	}
}

// OIDCSSOAttrTypes returns the attribute types for the OIDC SSO object.
func OIDCSSOAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"client_id":     types.StringType,
		"client_secret": types.StringType,
	}
}

// preserveOIDCSSO preserves SSO credentials from a previous state when the API
// returns empty values (client_id and client_secret are only returned on create).
func preserveOIDCSSO(ctx context.Context, prevSSO, newSSO types.Object) types.Object {
	if prevSSO.IsNull() || prevSSO.IsUnknown() {
		return newSSO
	}
	if newSSO.IsNull() || newSSO.IsUnknown() {
		return prevSSO
	}

	var prev, cur OIDCSSOModel
	prevSSO.As(ctx, &prev, basetypes.ObjectAsOptions{})
	newSSO.As(ctx, &cur, basetypes.ObjectAsOptions{})

	// Preserve client_id from state if API returned empty
	clientID := cur.ClientID
	if clientID.IsNull() || clientID.ValueString() == "" {
		clientID = prev.ClientID
	}

	// Preserve client_secret from state if API returned empty
	clientSecret := cur.ClientSecret
	if clientSecret.IsNull() || clientSecret.ValueString() == "" {
		clientSecret = prev.ClientSecret
	}

	obj, _ := types.ObjectValue(OIDCSSOAttrTypes(), map[string]attr.Value{
		"client_id":     clientID,
		"client_secret": clientSecret,
	})
	return obj
}

// ToSDKApp converts the OIDC Terraform model to the SDK App struct.
func (m *OIDCAppResourceModel) ToSDKApp(ctx context.Context) (*models.App, diag.Diagnostics) {
	app, diags := BaseAppToSDK(ctx, &m.BaseAppModel)
	if diags.HasError() {
		return nil, diags
	}

	// Configuration - use a map to only send user-specified fields
	if !m.Configuration.IsNull() && !m.Configuration.IsUnknown() {
		var cfg OIDCConfigurationModel
		d := m.Configuration.As(ctx, &cfg, oidcConfigAsOptions())
		diags.Append(d...)
		if !diags.HasError() {
			configMap := map[string]interface{}{}

			// redirect_uris → send as JSON array under "redirect_uri" key
			if !cfg.RedirectURIs.IsNull() && !cfg.RedirectURIs.IsUnknown() {
				var uris []string
				d := cfg.RedirectURIs.ElementsAs(ctx, &uris, false)
				diags.Append(d...)
				// Deduplicate
				seen := make(map[string]bool, len(uris))
				unique := make([]string, 0, len(uris))
				for _, u := range uris {
					if !seen[u] {
						seen[u] = true
						unique = append(unique, u)
					}
				}
				configMap["redirect_uri"] = unique
			}

			if !cfg.LoginURL.IsNull() && !cfg.LoginURL.IsUnknown() {
				configMap["login_url"] = cfg.LoginURL.ValueString()
			}
			if !cfg.OidcApplicationType.IsNull() && !cfg.OidcApplicationType.IsUnknown() {
				v, err := oidcAppTypeToInt(cfg.OidcApplicationType.ValueString())
				if err != nil {
					diags.AddError("Invalid OIDC Application Type", err.Error())
				} else {
					configMap["oidc_application_type"] = v
				}
			}
			if !cfg.TokenEndpointAuthMethod.IsNull() && !cfg.TokenEndpointAuthMethod.IsUnknown() {
				v, err := tokenAuthMethodToInt(cfg.TokenEndpointAuthMethod.ValueString())
				if err != nil {
					diags.AddError("Invalid Token Endpoint Auth Method", err.Error())
				} else {
					configMap["token_endpoint_auth_method"] = v
				}
			}
			if !cfg.AccessTokenExpirationMinutes.IsNull() && !cfg.AccessTokenExpirationMinutes.IsUnknown() {
				configMap["access_token_expiration_minutes"] = int(cfg.AccessTokenExpirationMinutes.ValueInt64())
			}
			if !cfg.RefreshTokenExpirationMinutes.IsNull() && !cfg.RefreshTokenExpirationMinutes.IsUnknown() {
				configMap["refresh_token_expiration_minutes"] = int(cfg.RefreshTokenExpirationMinutes.ValueInt64())
			}
			app.Configuration = configMap
		}
	}

	return app, diags
}

// FromSDKApp populates the OIDC Terraform model from an SDK App struct.
func (m *OIDCAppResourceModel) FromSDKApp(ctx context.Context, app *models.App) diag.Diagnostics {
	diags := BaseAppFromSDK(ctx, &m.BaseAppModel, app)
	if diags.HasError() {
		return diags
	}

	// Configuration
	if app.Configuration != nil {
		cfg, err := client.ExtractAppConfigOIDC(app.Configuration)
		if err != nil {
			diags.AddError("Error Extracting OIDC Configuration", err.Error())
			return diags
		}
		if cfg != nil {
			// Build redirect_uris set from the polymorphic API response
			var redirectURIsVal attr.Value
			if len(cfg.RedirectURIs) > 0 {
				// Deduplicate
				seen := make(map[string]bool, len(cfg.RedirectURIs))
				unique := make([]attr.Value, 0, len(cfg.RedirectURIs))
				for _, u := range cfg.RedirectURIs {
					if u != "" && !seen[u] {
						seen[u] = true
						unique = append(unique, types.StringValue(u))
					}
				}
				if len(unique) > 0 {
					setVal, d := types.SetValue(types.StringType, unique)
					diags.Append(d...)
					redirectURIsVal = setVal
				} else {
					redirectURIsVal = types.SetNull(types.StringType)
				}
			} else {
				redirectURIsVal = types.SetNull(types.StringType)
			}

			configObj, d := types.ObjectValue(OIDCConfigAttrTypes(), map[string]attr.Value{
				"redirect_uris":                    redirectURIsVal,
				"login_url":                        StringOrNull(cfg.LoginURL),
				"oidc_application_type":            types.StringValue(oidcAppTypeFromInt(cfg.OidcApplicationType)),
				"token_endpoint_auth_method":       types.StringValue(tokenAuthMethodFromInt(cfg.TokenEndpointAuthMethod)),
				"access_token_expiration_minutes":  types.Int64Value(int64(cfg.AccessTokenExpirationMinutes)),
				"refresh_token_expiration_minutes": types.Int64Value(int64(cfg.RefreshTokenExpirationMinutes)),
			})
			diags.Append(d...)
			m.Configuration = configObj
		} else {
			m.Configuration = types.ObjectNull(OIDCConfigAttrTypes())
		}
	} else {
		m.Configuration = types.ObjectNull(OIDCConfigAttrTypes())
	}

	// SSO - extract both client_id and client_secret from the API response
	if app.SSO != nil {
		ssoData, err := extractOIDCSSO(app.SSO)
		if err != nil {
			diags.AddError("Error Extracting OIDC SSO", err.Error())
			return diags
		}
		if ssoData != nil {
			ssoObj, d := types.ObjectValue(OIDCSSOAttrTypes(), map[string]attr.Value{
				"client_id":     StringOrNull(ssoData.ClientID),
				"client_secret": StringOrNull(ssoData.ClientSecret),
			})
			diags.Append(d...)
			m.SSO = ssoObj
		} else {
			m.SSO = types.ObjectNull(OIDCSSOAttrTypes())
		}
	} else {
		m.SSO = types.ObjectNull(OIDCSSOAttrTypes())
	}

	return diags
}
