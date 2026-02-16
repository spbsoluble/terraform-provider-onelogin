package app

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
)

// usesConsumerKey returns true for connector IDs that use "consumer" instead of "consumer_url"
// as the ACS configuration key (e.g., AWS SSO connector 130413).
func usesConsumerKey(connectorID int64) bool {
	switch connectorID {
	case 130413: // AWS SSO
		return true
	default:
		return false
	}
}

// SAMLConfigurationModel represents SAML-specific configuration.
type SAMLConfigurationModel struct {
	SignatureAlgorithm types.String `tfsdk:"signature_algorithm"`
	CertificateID      types.Int64  `tfsdk:"certificate_id"`
	ProviderArn        types.String `tfsdk:"provider_arn"`
	ACS                types.String `tfsdk:"acs"`
	Audience           types.String `tfsdk:"audience"`
	Recipient          types.String `tfsdk:"recipient"`
	RelayState         types.String `tfsdk:"relaystate"`
	Subdomain          types.String `tfsdk:"subdomain"`
}

// SAMLSSOCertificateModel represents a SAML certificate.
type SAMLSSOCertificateModel struct {
	ID    types.Int64  `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// SAMLAppResourceModel describes the SAML app resource data model.
type SAMLAppResourceModel struct {
	BaseAppModel
	Configuration types.Object `tfsdk:"configuration"`
	SSO           types.Object `tfsdk:"sso"`
}

// SAMLConfigAttrTypes returns the attribute types for the SAML configuration object.
func SAMLConfigAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"signature_algorithm": types.StringType,
		"certificate_id":      types.Int64Type,
		"provider_arn":        types.StringType,
		"acs":                 types.StringType,
		"audience":            types.StringType,
		"recipient":           types.StringType,
		"relaystate":          types.StringType,
		"subdomain":           types.StringType,
	}
}

// SAMLSSOCertAttrTypes returns the attribute types for the SAML SSO certificate.
func SAMLSSOCertAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":    types.Int64Type,
		"name":  types.StringType,
		"value": types.StringType,
	}
}

// SAMLSSOAttrTypes returns the attribute types for the SAML SSO object.
func SAMLSSOAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"metadata_url": types.StringType,
		"acs_url":      types.StringType,
		"sls_url":      types.StringType,
		"issuer":       types.StringType,
		"certificate":  types.ObjectType{AttrTypes: SAMLSSOCertAttrTypes()},
	}
}

// ToSDKApp converts the SAML Terraform model to the SDK App struct.
func (m *SAMLAppResourceModel) ToSDKApp(ctx context.Context) (*models.App, diag.Diagnostics) {
	app, diags := BaseAppToSDK(ctx, &m.BaseAppModel)
	if diags.HasError() {
		return nil, diags
	}

	// Configuration - use a map to avoid sending unwanted null fields
	// The SDK's ConfigurationSAML struct doesn't use omitempty, which causes
	// the API to reject null fields like provider_arn for non-AWS connectors.
	if !m.Configuration.IsNull() && !m.Configuration.IsUnknown() {
		var cfg SAMLConfigurationModel
		d := m.Configuration.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true})
		diags.Append(d...)
		if !diags.HasError() {
			configMap := map[string]interface{}{}
			if !cfg.SignatureAlgorithm.IsNull() && !cfg.SignatureAlgorithm.IsUnknown() {
				configMap["signature_algorithm"] = cfg.SignatureAlgorithm.ValueString()
			}
			if !cfg.CertificateID.IsNull() && !cfg.CertificateID.IsUnknown() && cfg.CertificateID.ValueInt64() > 0 {
				configMap["certificate_id"] = int(cfg.CertificateID.ValueInt64())
			}
			if !cfg.ProviderArn.IsNull() && !cfg.ProviderArn.IsUnknown() {
				configMap["provider_arn"] = cfg.ProviderArn.ValueString()
			}
			if !cfg.ACS.IsNull() && !cfg.ACS.IsUnknown() {
				// Some connectors (e.g., 130413 AWS SSO) use "consumer" instead of "consumer_url".
				// Detect based on connector_id from the base model.
				acsKey := "consumer_url"
				if !m.BaseAppModel.ConnectorID.IsNull() && !m.BaseAppModel.ConnectorID.IsUnknown() {
					connID := m.BaseAppModel.ConnectorID.ValueInt64()
					if usesConsumerKey(connID) {
						acsKey = "consumer"
					}
				}
				configMap[acsKey] = cfg.ACS.ValueString()
			}
			if !cfg.Audience.IsNull() && !cfg.Audience.IsUnknown() {
				configMap["audience"] = cfg.Audience.ValueString()
			}
			if !cfg.Recipient.IsNull() && !cfg.Recipient.IsUnknown() {
				configMap["recipient"] = cfg.Recipient.ValueString()
			}
			if !cfg.RelayState.IsNull() && !cfg.RelayState.IsUnknown() {
				configMap["relaystate"] = cfg.RelayState.ValueString()
			}
			if !cfg.Subdomain.IsNull() && !cfg.Subdomain.IsUnknown() {
				configMap["subdomain"] = cfg.Subdomain.ValueString()
			}
			app.Configuration = configMap
		}
	}

	return app, diags
}

// FromSDKApp populates the SAML Terraform model from an SDK App struct.
func (m *SAMLAppResourceModel) FromSDKApp(ctx context.Context, app *models.App) diag.Diagnostics {
	diags := BaseAppFromSDK(ctx, &m.BaseAppModel, app)
	if diags.HasError() {
		return diags
	}

	// Configuration
	if app.Configuration != nil {
		cfg, err := client.ExtractAppConfigSAML(app.Configuration)
		if err != nil {
			diags.AddError("Error Extracting SAML Configuration", err.Error())
			return diags
		}
		if cfg != nil {
			// ProviderArn can be interface{} in the SDK
			providerArn := types.StringNull()
			if cfg.ProviderArn != nil {
				switch v := cfg.ProviderArn.(type) {
				case string:
					if v != "" {
						providerArn = types.StringValue(v)
					}
				}
			}

			configObj, d := types.ObjectValue(SAMLConfigAttrTypes(), map[string]attr.Value{
				"signature_algorithm": StringOrNull(cfg.SignatureAlgorithm),
				"certificate_id":      IntToInt64(cfg.CertificateID),
				"provider_arn":        providerArn,
				"acs":                 StringOrNull(cfg.ACS),
				"audience":            StringOrNull(cfg.Audience),
				"recipient":           StringOrNull(cfg.Recipient),
				"relaystate":          StringOrNull(cfg.RelayState),
				"subdomain":           StringOrNull(cfg.Subdomain),
			})
			diags.Append(d...)
			m.Configuration = configObj
		} else {
			m.Configuration = types.ObjectNull(SAMLConfigAttrTypes())
		}
	} else {
		m.Configuration = types.ObjectNull(SAMLConfigAttrTypes())
	}

	// SSO
	if app.SSO != nil {
		sso, err := client.ExtractSSOSAML(app.SSO)
		if err != nil {
			diags.AddError("Error Extracting SAML SSO", err.Error())
			return diags
		}
		if sso != nil {
			certObj, d := types.ObjectValue(SAMLSSOCertAttrTypes(), map[string]attr.Value{
				"id":    types.Int64Value(int64(sso.Certificate.ID)),
				"name":  StringOrNull(sso.Certificate.Name),
				"value": StringOrNull(sso.Certificate.Value),
			})
			diags.Append(d...)

			ssoObj, d := types.ObjectValue(SAMLSSOAttrTypes(), map[string]attr.Value{
				"metadata_url": StringOrNull(sso.MetadataURL),
				"acs_url":      StringOrNull(sso.AcsURL),
				"sls_url":      StringOrNull(sso.SlsURL),
				"issuer":       StringOrNull(sso.Issuer),
				"certificate":  certObj,
			})
			diags.Append(d...)
			m.SSO = ssoObj
		} else {
			m.SSO = types.ObjectNull(SAMLSSOAttrTypes())
		}
	} else {
		m.SSO = types.ObjectNull(SAMLSSOAttrTypes())
	}

	return diags
}
