package role

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	client *client.Client
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a OneLogin Role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "The unique identifier of the role.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the role.",
			},
			"users": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "A set of user IDs assigned to this role.",
			},
			"apps": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of app names accessible by this role.",
			},
			"admins": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "A set of user IDs who administer this role.",
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve app names → IDs before sending to the API.
	appNames, d := common.SetToStringSlice(ctx, plan.Apps)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	appIDs, err := r.resolveAppNamesToIDs(ctx, appNames)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App Names", err.Error())
		return
	}

	sdkRole, diags := plan.ToSDKRole(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	sdkRole.Apps = appIDs

	tflog.Debug(ctx, "Creating role", map[string]any{"name": plan.Name.ValueString()})

	createResult, err := r.client.SDK.CreateRoleWithContext(ctx, sdkRole)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Role", "Could not create role: "+err.Error())
		return
	}

	// Parse the create response to get the ID
	createdRole, err := client.UnmarshalRole(createResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Role Response", err.Error())
		return
	}
	if createdRole == nil || createdRole.ID == nil {
		resp.Diagnostics.AddError("Error Creating Role", "API returned nil response or missing ID")
		return
	}

	// Read back the full role to get all fields (create may return partial data)
	id := int(*createdRole.ID)
	readResult, err := r.client.SDK.GetRoleByIDWithContext(ctx, id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Role After Create", fmt.Sprintf("Could not read role ID %d: %s", id, err.Error()))
		return
	}

	role, err := client.UnmarshalRole(readResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Role Response", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Error Reading Role After Create", "Role not found after creation")
		return
	}

	var state RoleResourceModel
	diags = state.FromSDKRole(ctx, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store app names (not IDs) in state.
	resolvedNames, err := r.resolveAppIDsToNames(ctx, role.Apps)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App IDs to Names", err.Error())
		return
	}
	state.Apps, d = common.StringSliceToSet(ctx, resolvedNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Reading role", map[string]any{"id": id})

	result, err := r.client.SDK.GetRoleByIDWithContext(ctx, id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Role", fmt.Sprintf("Could not read role ID %d: %s", id, err.Error()))
		return
	}

	role, err := client.UnmarshalRole(result)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Role Response", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = state.FromSDKRole(ctx, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve app IDs → names for human-readable diffs.
	appNames, err := r.resolveAppIDsToNames(ctx, role.Apps)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App IDs to Names", err.Error())
		return
	}
	var d diag.Diagnostics
	state.Apps, d = common.StringSliceToSet(ctx, appNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RoleResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Updating role", map[string]any{"id": id})

	// Step 1: Update name via the role PUT endpoint.
	// Only send name — users/admins are synced differentially, apps via UpdateRoleApps.
	_, err := r.client.SDK.UpdateRoleWithContext(ctx, id, &models.Role{
		Name: common.StringToStringPtr(plan.Name),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Role", fmt.Sprintf("Could not update role ID %d: %s", id, err.Error()))
		return
	}

	// Step 2: Resolve app names → IDs and sync via dedicated endpoint.
	planAppNames, d := common.SetToStringSlice(ctx, plan.Apps)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planAppNames != nil {
		planAppIDs, err := r.resolveAppNamesToIDs(ctx, planAppNames)
		if err != nil {
			resp.Diagnostics.AddError("Error Resolving App Names", err.Error())
			return
		}
		_, err = r.client.SDK.UpdateRoleApps(id, common.Int32SliceToIntSlice(planAppIDs))
		if err != nil {
			resp.Diagnostics.AddError("Error Updating Role Apps", err.Error())
			return
		}
	}

	// Step 3: Differential user sync (UpdateRole does NOT handle user removal)
	r.syncUsers(ctx, id, state, plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 4: Differential admin sync
	r.syncAdmins(ctx, id, state, plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 4: Read back the final state
	result, err := r.client.SDK.GetRoleByIDWithContext(ctx, id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Role After Update", err.Error())
		return
	}

	role, err := client.UnmarshalRole(result)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Role Response", err.Error())
		return
	}

	var newState RoleResourceModel
	diags = newState.FromSDKRole(ctx, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve app IDs → names for state.
	resolvedNames, err := r.resolveAppIDsToNames(ctx, role.Apps)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App IDs to Names", err.Error())
		return
	}
	newState.Apps, d = common.StringSliceToSet(ctx, resolvedNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(state.ID.ValueInt64())
	tflog.Debug(ctx, "Deleting role", map[string]any{"id": id})

	_, err := r.client.SDK.DeleteRoleWithContext(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Role", fmt.Sprintf("Could not delete role ID %d: %s", id, err.Error()))
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := common.ParseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// resolveAppIDsToNames converts a slice of OneLogin app IDs to their display names.
// Each ID requires one API call (GET /api/2/apps/{id}).
// Duplicate IDs are skipped. When two distinct IDs resolve to the same name, the ID
// is appended in parentheses (e.g. "My App (4094617)") to keep Set elements unique
// and make the diff readable.
func (r *roleResource) resolveAppIDsToNames(ctx context.Context, ids []int32) ([]string, error) {
	seenIDs := make(map[int32]struct{}, len(ids))
	seenNames := make(map[string]struct{}, len(ids))
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, dup := seenIDs[id]; dup {
			continue
		}
		seenIDs[id] = struct{}{}
		result, err := r.client.SDK.GetAppByID(int(id), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch app ID %d: %w", id, err)
		}
		app, err := client.UnmarshalApp(result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse app ID %d: %w", id, err)
		}
		if app == nil || app.Name == nil {
			return nil, fmt.Errorf("app ID %d not found or has no name", id)
		}
		label := *app.Name
		if _, nameDup := seenNames[label]; nameDup {
			label = fmt.Sprintf("%s (%d)", *app.Name, id)
		}
		seenNames[label] = struct{}{}
		names = append(names, label)
	}
	return names, nil
}

// resolveAppNamesToIDs converts a slice of OneLogin app display names to their IDs.
// Each name uses GET /api/2/apps?name=<name> and picks the first exact match.
func (r *roleResource) resolveAppNamesToIDs(ctx context.Context, names []string) ([]int32, error) {
	ids := make([]int32, 0, len(names))
	for _, name := range names {
		n := name
		result, err := r.client.SDK.GetApps(&models.AppQuery{Name: &n})
		if err != nil {
			return nil, fmt.Errorf("failed to query app %q: %w", name, err)
		}
		apps, err := client.UnmarshalApps(result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse apps response for %q: %w", name, err)
		}
		var matched *int32
		for i := range apps {
			if apps[i].Name != nil && *apps[i].Name == name && apps[i].ID != nil {
				matched = apps[i].ID
				break
			}
		}
		if matched == nil {
			return nil, fmt.Errorf("app %q not found in OneLogin", name)
		}
		ids = append(ids, *matched)
	}
	return ids, nil
}

// syncUsers performs differential add/remove of users on a role.
func (r *roleResource) syncUsers(ctx context.Context, roleID int, oldState, newPlan RoleResourceModel, resp *resource.UpdateResponse) {
	oldUsers, d := common.SetToInt32Slice(ctx, oldState.Users)
	resp.Diagnostics.Append(d...)
	newUsers, d := common.SetToInt32Slice(ctx, newPlan.Users)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	toAdd, toRemove := diffInt32Slices(oldUsers, newUsers)

	if len(toAdd) > 0 {
		tflog.Debug(ctx, "Adding users to role", map[string]any{"role_id": roleID, "users": toAdd})
		_, err := r.client.SDK.AddRoleUsers(roleID, common.Int32SliceToIntSlice(toAdd))
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Role Users", err.Error())
			return
		}
	}

	if len(toRemove) > 0 {
		tflog.Debug(ctx, "Removing users from role", map[string]any{"role_id": roleID, "users": toRemove})
		_, err := r.client.SDK.DeleteRoleUsers(roleID, common.Int32SliceToIntSlice(toRemove))
		if err != nil {
			resp.Diagnostics.AddError("Error Removing Role Users", err.Error())
			return
		}
	}
}

// syncAdmins performs differential add/remove of admins on a role.
func (r *roleResource) syncAdmins(ctx context.Context, roleID int, oldState, newPlan RoleResourceModel, resp *resource.UpdateResponse) {
	oldAdmins, d := common.SetToInt32Slice(ctx, oldState.Admins)
	resp.Diagnostics.Append(d...)
	newAdmins, d := common.SetToInt32Slice(ctx, newPlan.Admins)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	toAdd, toRemove := diffInt32Slices(oldAdmins, newAdmins)

	if len(toAdd) > 0 {
		tflog.Debug(ctx, "Adding admins to role", map[string]any{"role_id": roleID, "admins": toAdd})
		_, err := r.client.SDK.AddRoleAdmins(roleID, common.Int32SliceToIntSlice(toAdd))
		if err != nil {
			resp.Diagnostics.AddError("Error Adding Role Admins", err.Error())
			return
		}
	}

	if len(toRemove) > 0 {
		tflog.Debug(ctx, "Removing admins from role", map[string]any{"role_id": roleID, "admins": toRemove})
		_, err := r.client.SDK.DeleteRoleAdmins(roleID, common.Int32SliceToIntSlice(toRemove))
		if err != nil {
			resp.Diagnostics.AddError("Error Removing Role Admins", err.Error())
			return
		}
	}
}

// diffInt32Slices returns (toAdd, toRemove) comparing old vs new slices.
func diffInt32Slices(old, new []int32) (toAdd, toRemove []int32) {
	oldSet := make(map[int32]struct{}, len(old))
	for _, v := range old {
		oldSet[v] = struct{}{}
	}
	newSet := make(map[int32]struct{}, len(new))
	for _, v := range new {
		newSet[v] = struct{}{}
	}

	for _, v := range new {
		if _, exists := oldSet[v]; !exists {
			toAdd = append(toAdd, v)
		}
	}
	for _, v := range old {
		if _, exists := newSet[v]; !exists {
			toRemove = append(toRemove, v)
		}
	}
	return
}
