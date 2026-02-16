package app

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

var (
	_ resource.Resource                = &appResource{}
	_ resource.ResourceWithConfigure   = &appResource{}
	_ resource.ResourceWithImportState = &appResource{}
)

func NewAppResource() resource.Resource {
	return &appResource{}
}

type appResource struct {
	client *client.Client
}

func (r *appResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *appResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := BaseAppSchemaAttributes()
	attrs["provisioning"] = ProvisioningBlock()
	attrs["parameters"] = ParametersBlock()
	attrs["configuration"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "App configuration as a JSON string. Use jsonencode() to set.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["sso"] = schema.StringAttribute{
		Computed:    true,
		Description: "SSO settings as a JSON string (read-only).",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a generic OneLogin App.",
		Attributes:  attrs,
	}
}

func (r *appResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *appResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GenericAppResourceModel
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

	tflog.Debug(ctx, "Creating app", map[string]any{"name": plan.Name.ValueString()})

	createResult, err := r.client.SDK.CreateApp(*sdkApp)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating App", "Could not create app: "+err.Error())
		return
	}

	createdApp, err := client.UnmarshalApp(createResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing App Response", err.Error())
		return
	}
	if createdApp == nil || createdApp.ID == nil {
		resp.Diagnostics.AddError("Error Creating App", "API returned nil response or missing ID")
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

	var state GenericAppResourceModel
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

func (r *appResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GenericAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	prevParams := state.Parameters

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Reading app", map[string]any{"id": id})

	result, err := r.client.SDK.GetAppByID(id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading App", fmt.Sprintf("Could not read app ID %d: %s", id, err.Error()))
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

func (r *appResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GenericAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GenericAppResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Updating app", map[string]any{"id": id})

	sdkApp, diags := plan.ToSDKApp(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Don't send configuration back if unchanged — the API GET response
	// may include read-only fields that the API rejects on PUT.
	if plan.Configuration.Equal(state.Configuration) {
		sdkApp.Configuration = nil
	}

	_, err := r.client.SDK.UpdateApp(id, *sdkApp)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating App", fmt.Sprintf("Could not update app ID %d: %s", id, err.Error()))
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

	var newState GenericAppResourceModel
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

func (r *appResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GenericAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Deleting app", map[string]any{"id": id})

	_, err := r.client.SDK.DeleteApp(id)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting App", fmt.Sprintf("Could not delete app ID %d: %s", id, err.Error()))
	}
}

func (r *appResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := common.ParseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
