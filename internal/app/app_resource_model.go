package app

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
)

// GenericAppResourceModel describes the generic app resource data model.
type GenericAppResourceModel struct {
	BaseAppModel
	Configuration types.String `tfsdk:"configuration"`
	SSO           types.String `tfsdk:"sso"`
}

// ToSDKApp converts the Terraform model to the SDK App struct.
func (m *GenericAppResourceModel) ToSDKApp(ctx context.Context) (*models.App, diag.Diagnostics) {
	app, diags := BaseAppToSDK(ctx, &m.BaseAppModel)
	if diags.HasError() {
		return nil, diags
	}

	// Configuration: parse JSON string into interface{}
	if !m.Configuration.IsNull() && !m.Configuration.IsUnknown() {
		var config interface{}
		if err := json.Unmarshal([]byte(m.Configuration.ValueString()), &config); err != nil {
			diags.AddError("Invalid Configuration JSON", "Could not parse configuration: "+err.Error())
			return nil, diags
		}
		app.Configuration = config
	}

	return app, diags
}

// FromSDKApp populates the Terraform model from an SDK App struct.
func (m *GenericAppResourceModel) FromSDKApp(ctx context.Context, app *models.App) diag.Diagnostics {
	diags := BaseAppFromSDK(ctx, &m.BaseAppModel, app)
	if diags.HasError() {
		return diags
	}

	// Configuration: serialize to JSON string
	if app.Configuration != nil {
		b, err := json.Marshal(app.Configuration)
		if err != nil {
			diags.AddError("Error Serializing Configuration", err.Error())
			return diags
		}
		m.Configuration = types.StringValue(string(b))
	} else {
		m.Configuration = types.StringNull()
	}

	// SSO: serialize to JSON string (read-only)
	if app.SSO != nil {
		b, err := json.Marshal(app.SSO)
		if err != nil {
			diags.AddError("Error Serializing SSO", err.Error())
			return diags
		}
		m.SSO = types.StringValue(string(b))
	} else {
		m.SSO = types.StringNull()
	}

	return diags
}
