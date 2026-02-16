package user_mapping

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	frameworkvalidators "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
	utl "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/utilities"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

var (
	_ resource.Resource                = &userMappingResource{}
	_ resource.ResourceWithConfigure   = &userMappingResource{}
	_ resource.ResourceWithImportState = &userMappingResource{}
)

func NewUserMappingResource() resource.Resource {
	return &userMappingResource{}
}

type userMappingResource struct {
	client *client.Client
}

func (r *userMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_mapping"
}

func (r *userMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a OneLogin User Mapping rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "The unique identifier of the user mapping.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the user mapping.",
			},
			"match": schema.StringAttribute{
				Required:    true,
				Description: "Indicates how conditions are matched. Valid values: \"all\", \"any\".",
				Validators: []validator.String{
					frameworkvalidators.OneOf("all", "any"),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the user mapping is enabled. Defaults to false.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The position of the user mapping in the evaluation order.",
			},
		},
		Blocks: map[string]schema.Block{
			"conditions": schema.ListNestedBlock{
				Description: "A list of conditions that must be met for the mapping to apply.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Required:    true,
							Description: "The source field to evaluate (e.g., \"email\").",
						},
						"operator": schema.StringAttribute{
							Required:    true,
							Description: "The comparison operator (e.g., \"contains\", \"=\").",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value to compare against.",
						},
					},
				},
			},
			"actions": schema.ListNestedBlock{
				Description: "A list of actions to perform when conditions are met.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Required:    true,
							Description: "The action type (e.g., \"set_role\").",
						},
						"value": schema.ListAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "The values for the action.",
						},
					},
				},
			},
		},
	}
}

func (r *userMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserMappingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdkMapping, diags := plan.ToSDKUserMapping(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating user mapping", map[string]any{"name": plan.Name.ValueString()})

	created, err := r.client.SDK.CreateUserMapping(*sdkMapping)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating User Mapping", "Could not create user mapping: "+err.Error())
		return
	}
	if created == nil || created.ID == nil {
		resp.Diagnostics.AddError("Error Creating User Mapping", "API returned nil response or missing ID")
		return
	}

	// Read back the full mapping (Create may return only the ID)
	result, err := r.client.SDK.GetUserMapping(*created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User Mapping After Create", fmt.Sprintf("Could not read user mapping ID %d: %s", *created.ID, err.Error()))
		return
	}
	if result == nil {
		resp.Diagnostics.AddError("Error Reading User Mapping After Create", "User mapping not found after creation")
		return
	}

	var state UserMappingResourceModel
	diags = state.FromSDKUserMapping(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *userMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserMappingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int32(state.ID.ValueInt64())
	tflog.Debug(ctx, "Reading user mapping", map[string]any{"id": id})

	result, err := r.client.SDK.GetUserMapping(id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User Mapping", fmt.Sprintf("Could not read user mapping ID %d: %s", id, err.Error()))
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = state.FromSDKUserMapping(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// userMappingUpdatePayload is a custom struct for the PUT request body.
// The OneLogin API requires: no "id" field in the body, and "position" must always be present.
// When disabling a mapping, position must be null (not a numeric value).
// The SDK model uses omitempty on both id and position, so we use this custom struct.
type userMappingUpdatePayload struct {
	Name       *string                        `json:"name,omitempty"`
	Match      *string                        `json:"match,omitempty"`
	Enabled    *bool                          `json:"enabled,omitempty"`
	Position   *int32                         `json:"position"` // No omitempty - API always requires this field
	Conditions []models.UserMappingConditions `json:"conditions"`
	Actions    []models.UserMappingActions    `json:"actions"`
}

func (r *userMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserMappingResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserMappingResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int32(state.ID.ValueInt64())
	tflog.Debug(ctx, "Updating user mapping", map[string]any{"id": id})

	sdkMapping, diags := plan.ToSDKUserMapping(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build payload without id field and with position always present.
	// API quirk: position must be null (not a numeric value) when disabling.
	var position *int32
	enabled := sdkMapping.Enabled != nil && *sdkMapping.Enabled
	if enabled {
		position = sdkMapping.Position // May be nil (serialized as null), which is fine
	}
	// When disabled: position stays nil → serialized as "position": null

	payload := userMappingUpdatePayload{
		Name:       sdkMapping.Name,
		Match:      sdkMapping.Match,
		Enabled:    sdkMapping.Enabled,
		Position:   position,
		Conditions: sdkMapping.Conditions,
		Actions:    sdkMapping.Actions,
	}

	p, err := utl.BuildAPIPath("api/2/mappings", id)
	if err != nil {
		resp.Diagnostics.AddError("Error Building API Path", err.Error())
		return
	}

	httpResp, err := r.client.SDK.Client.Put(&p, payload)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating User Mapping", fmt.Sprintf("Could not update user mapping ID %d: %s", id, err.Error()))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("Error Updating User Mapping",
			fmt.Sprintf("API returned status %d for mapping ID %d: %s", httpResp.StatusCode, id, string(body)))
		return
	}

	// Parse response - API may return just {id: XYZ}
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Update Response", err.Error())
		return
	}

	var responseObj struct {
		ID int32 `json:"id"`
	}
	_ = json.Unmarshal(body, &responseObj)

	// Read back the full mapping
	var readID int32
	if responseObj.ID > 0 {
		readID = responseObj.ID
	} else {
		readID = id
	}

	result, err := r.client.SDK.GetUserMapping(readID)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User Mapping After Update", fmt.Sprintf("Could not read user mapping ID %d: %s", readID, err.Error()))
		return
	}

	var newState UserMappingResourceModel
	diags = newState.FromSDKUserMapping(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
}

func (r *userMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserMappingResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int32(state.ID.ValueInt64())
	tflog.Debug(ctx, "Deleting user mapping", map[string]any{"id": id})

	err := r.client.SDK.DeleteUserMapping(id)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting User Mapping", fmt.Sprintf("Could not delete user mapping ID %d: %s", id, err.Error()))
	}
}

func (r *userMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := common.ParseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
