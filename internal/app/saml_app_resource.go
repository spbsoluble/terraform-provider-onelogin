package app

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

var (
	_ resource.Resource                = &samlAppResource{}
	_ resource.ResourceWithConfigure   = &samlAppResource{}
	_ resource.ResourceWithImportState = &samlAppResource{}
)

func NewSAMLAppResource() resource.Resource {
	return &samlAppResource{}
}

type samlAppResource struct {
	client *client.Client
}

func (r *samlAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_app"
}

func (r *samlAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := BaseAppSchemaAttributes()
	attrs["provisioning"] = ProvisioningBlock()
	attrs["parameters"] = ParametersBlock()

	blocks := map[string]schema.Block{
		"configuration": schema.SingleNestedBlock{
			Description: "SAML-specific configuration settings.",
			PlanModifiers: []planmodifier.Object{
				objectplanmodifier.UseStateForUnknown(),
			},
			Attributes: map[string]schema.Attribute{
				"signature_algorithm": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The SAML signature algorithm (e.g., \"SHA-1\", \"SHA-256\", \"SHA-384\", \"SHA-512\").",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullString(),
					},
				},
				"certificate_id": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "The certificate ID to use for SAML signing.",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullInt64(),
					},
				},
				"provider_arn": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The AWS provider ARN (for AWS SAML apps).",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullString(),
					},
				},
				"acs": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The Assertion Consumer Service (ACS) URL where the SAML assertion is sent.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullString(),
					},
				},
				"audience": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The audience restriction / SP Entity ID.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullString(),
					},
				},
				"recipient": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The recipient URL for the SAML assertion.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullString(),
					},
				},
				"relaystate": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The relay state URL.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullString(),
					},
				},
				"subdomain": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The subdomain for catalog SAML connectors (e.g., LogicMonitor).",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						UseStateWhenConfigNullString(),
					},
				},
			},
		},
	}

	attrs["sso"] = schema.SingleNestedAttribute{
		Computed:    true,
		Description: "SAML SSO settings (read-only, populated by the API).",
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"metadata_url": schema.StringAttribute{
				Computed:    true,
				Description: "The SAML metadata URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"acs_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Assertion Consumer Service URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sls_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Single Logout Service URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"issuer": schema.StringAttribute{
				Computed:    true,
				Description: "The SAML issuer URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The SAML signing certificate.",
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Computed:    true,
						Description: "The certificate ID.",
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "The certificate name.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"value": schema.StringAttribute{
						Computed:    true,
						Sensitive:   true,
						Description: "The certificate value (PEM-encoded).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a OneLogin SAML App.",
		Attributes:  attrs,
		Blocks:      blocks,
	}
}

func (r *samlAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *samlAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SAMLAppResourceModel
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

	tflog.Debug(ctx, "Creating SAML app", map[string]any{"name": plan.Name.ValueString()})

	createResult, err := r.client.SDK.CreateApp(*sdkApp)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating SAML App", "Could not create app: "+err.Error())
		return
	}

	createdApp, err := client.UnmarshalApp(createResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing App Response", err.Error())
		return
	}
	if createdApp == nil || createdApp.ID == nil {
		resp.Diagnostics.AddError("Error Creating SAML App", "API returned nil response or missing ID")
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

	var state SAMLAppResourceModel
	diags = state.FromSDKApp(ctx, app)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Filter parameters to only include user-specified ones (API may add defaults)
	filtered, d := FilterParametersByKnownKeys(ctx, plan.Parameters, state.Parameters)
	resp.Diagnostics.Append(d...)
	state.Parameters = filtered

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *samlAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SAMLAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	prevParams := state.Parameters

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Reading SAML app", map[string]any{"id": id})

	result, err := r.client.SDK.GetAppByID(id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading SAML App", fmt.Sprintf("Could not read app ID %d: %s", id, err.Error()))
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

	filtered, d := FilterParametersByKnownKeys(ctx, prevParams, state.Parameters)
	resp.Diagnostics.Append(d...)
	state.Parameters = filtered

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *samlAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SAMLAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SAMLAppResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Updating SAML app", map[string]any{"id": id})

	sdkApp, diags := plan.ToSDKApp(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.SDK.UpdateApp(id, *sdkApp)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating SAML App", fmt.Sprintf("Could not update app ID %d: %s", id, err.Error()))
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

	var newState SAMLAppResourceModel
	diags = newState.FromSDKApp(ctx, app)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filtered, d := FilterParametersByKnownKeys(ctx, plan.Parameters, newState.Parameters)
	resp.Diagnostics.Append(d...)
	newState.Parameters = filtered

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
}

func (r *samlAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SAMLAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Deleting SAML app", map[string]any{"id": id})

	_, err := r.client.SDK.DeleteApp(id)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting SAML App", fmt.Sprintf("Could not delete app ID %d: %s", id, err.Error()))
	}
}

func (r *samlAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := common.ParseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
