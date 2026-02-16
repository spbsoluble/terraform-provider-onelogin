package app

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

var (
	_ resource.Resource                = &oidcAppResource{}
	_ resource.ResourceWithConfigure   = &oidcAppResource{}
	_ resource.ResourceWithImportState = &oidcAppResource{}
)

func NewOIDCAppResource() resource.Resource {
	return &oidcAppResource{}
}

type oidcAppResource struct {
	client *client.Client
}

func (r *oidcAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_app"
}

func (r *oidcAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := BaseAppSchemaAttributes()
	attrs["provisioning"] = ProvisioningBlock()
	attrs["parameters"] = ParametersBlock()

	blocks := map[string]schema.Block{
		"configuration": schema.SingleNestedBlock{
			Description: "OIDC-specific configuration settings.",
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier.UseStateForUnknown(),
			},
			Attributes: map[string]schema.Attribute{
				"redirect_uris": schema.SetAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "Set of redirect URIs for the OIDC app. Order does not matter; duplicates are automatically removed.",
					PlanModifiers: []planmodifier.Set{
						setplanmodifier.UseStateForUnknown(),
					},
				},
				"login_url": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The login URL.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"oidc_application_type": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "OIDC application type. Accepts: \"Web\" (default), \"Native\".",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"token_endpoint_auth_method": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Token endpoint auth method. Accepts: \"BASIC\", \"POST\", \"PKCE\" (None/PKCE).",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"access_token_expiration_minutes": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "Access token expiration in minutes.",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"refresh_token_expiration_minutes": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "Refresh token expiration in minutes.",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
			},
		},
	}

	attrs["sso"] = schema.SingleNestedAttribute{
		Computed:    true,
		Description: "OIDC SSO settings (read-only, populated by the API).",
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The OIDC client ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The OIDC client secret.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a OneLogin OIDC App.",
		Attributes:  attrs,
		Blocks:      blocks,
	}
}

func (r *oidcAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *oidcAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OIDCAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkApp, diags := plan.ToSDKApp(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating OIDC app", map[string]any{"name": plan.Name.ValueString()})

	createResult, err := r.client.SDK.CreateApp(*sdkApp)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating OIDC App", "Could not create app: "+err.Error())
		return
	}

	createdApp, err := client.UnmarshalApp(createResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing App Response", err.Error())
		return
	}
	if createdApp == nil || createdApp.ID == nil {
		resp.Diagnostics.AddError("Error Creating OIDC App", "API returned nil response or missing ID")
		return
	}

	// Read back the full app
	id := int(*createdApp.ID)
	readResult, err := r.client.SDK.GetAppByID(id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading App After Create", fmt.Sprintf("Could not read app ID %d: %s", id, err.Error()))
		return
	}

	app, err := client.UnmarshalApp(readResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing App Response", err.Error())
		return
	}
	if app == nil {
		resp.Diagnostics.AddError("Error Reading App After Create", "App not found after creation")
		return
	}

	var state OIDCAppResourceModel
	diags = state.FromSDKApp(ctx, app)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve SSO credentials from the create response (GET doesn't return client_secret)
	if createdApp.SSO != nil {
		ssoData, ssoErr := extractOIDCSSO(createdApp.SSO)
		if ssoErr == nil && ssoData != nil {
			ssoObj, d := types.ObjectValue(OIDCSSOAttrTypes(), map[string]attr.Value{
				"client_id":     StringOrNull(ssoData.ClientID),
				"client_secret": StringOrNull(ssoData.ClientSecret),
			})
			resp.Diagnostics.Append(d...)
			state.SSO = ssoObj
		}
	}

	// Filter parameters to only include user-specified ones (API may add defaults)
	filtered, d := FilterParametersByKnownKeys(ctx, plan.Parameters, state.Parameters)
	resp.Diagnostics.Append(d...)
	state.Parameters = filtered

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *oidcAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OIDCAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save values from state before overwriting with API response
	prevParams := state.Parameters
	prevSSO := state.SSO

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Reading OIDC app", map[string]any{"id": id})

	result, err := r.client.SDK.GetAppByID(id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading OIDC App", fmt.Sprintf("Could not read app ID %d: %s", id, err.Error()))
		return
	}

	app, err := client.UnmarshalApp(result)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing App Response", err.Error())
		return
	}
	if app == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = state.FromSDKApp(ctx, app)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Filter parameters to only include those already in state
	filtered, d := FilterParametersByKnownKeys(ctx, prevParams, state.Parameters)
	resp.Diagnostics.Append(d...)
	state.Parameters = filtered

	// Preserve SSO credentials from state (API only returns them on create)
	state.SSO = preserveOIDCSSO(ctx, prevSSO, state.SSO)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *oidcAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OIDCAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OIDCAppResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Updating OIDC app", map[string]any{"id": id})

	sdkApp, diags := plan.ToSDKApp(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.SDK.UpdateApp(id, *sdkApp)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating OIDC App", fmt.Sprintf("Could not update app ID %d: %s", id, err.Error()))
		return
	}

	// Read back the full app
	readResult, err := r.client.SDK.GetAppByID(id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading App After Update", fmt.Sprintf("Could not read app ID %d: %s", id, err.Error()))
		return
	}

	app, err := client.UnmarshalApp(readResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing App Response", err.Error())
		return
	}

	var newState OIDCAppResourceModel
	diags = newState.FromSDKApp(ctx, app)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Filter parameters to only include user-specified ones
	filtered, d := FilterParametersByKnownKeys(ctx, plan.Parameters, newState.Parameters)
	resp.Diagnostics.Append(d...)
	newState.Parameters = filtered

	// Preserve SSO credentials from state (API only returns them on create)
	newState.SSO = preserveOIDCSSO(ctx, state.SSO, newState.SSO)

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
}

func (r *oidcAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OIDCAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Deleting OIDC app", map[string]any{"id": id})

	_, err := r.client.SDK.DeleteApp(id)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting OIDC App", fmt.Sprintf("Could not delete app ID %d: %s", id, err.Error()))
	}
}

func (r *oidcAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := common.ParseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
