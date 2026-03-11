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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
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
			"apps": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of app names accessible by this role.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"admins": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of email addresses for users who administer this role.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
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

	// Resolve admin emails → IDs before creating.
	adminEmails, d := common.SetToStringSlice(ctx, plan.Admins)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	adminIDs, err := r.resolveAdminEmailsToIDs(ctx, adminEmails)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Admin Emails", err.Error())
		return
	}

	sdkRole := plan.ToSDKRole()
	sdkRole.Apps = appIDs
	sdkRole.Admins = adminIDs

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

	// Fetch apps and admins via dedicated endpoints (not included in role detail response).
	fetchedAppIDs, err := r.fetchRoleAppIDs(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Apps", err.Error())
		return
	}
	resolvedNames, err := r.resolveAppIDsToNames(ctx, fetchedAppIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App IDs to Names", err.Error())
		return
	}
	state.Apps, d = common.StringSliceToSet(ctx, resolvedNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	fetchedAdminIDs, err := r.fetchRoleAdminIDs(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Admins", err.Error())
		return
	}
	resolvedAdminEmails, err := r.resolveAdminIDsToEmails(ctx, fetchedAdminIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Admin IDs to Emails", err.Error())
		return
	}
	state.Admins, d = common.StringSliceToSet(ctx, resolvedAdminEmails)
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
	tflog.Debug(ctx, "Read: fetching role from API", map[string]any{"id": id, "name": state.Name.ValueString()})

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

	// Fetch apps and admins via dedicated endpoints (not included in role detail response).
	fetchedAppIDs, err := r.fetchRoleAppIDs(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Apps", err.Error())
		return
	}
	tflog.Debug(ctx, "Read: fetched role app IDs", map[string]any{"id": id, "app_count": len(fetchedAppIDs)})

	appNames, err := r.resolveAppIDsToNames(ctx, fetchedAppIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App IDs to Names", err.Error())
		return
	}
	tflog.Debug(ctx, "Read: app names resolved", map[string]any{"id": id, "app_count": len(appNames)})

	var d diag.Diagnostics
	state.Apps, d = common.StringSliceToSet(ctx, appNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	fetchedAdminIDs, err := r.fetchRoleAdminIDs(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Admins", err.Error())
		return
	}
	adminEmails, err := r.resolveAdminIDsToEmails(ctx, fetchedAdminIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Admin IDs to Emails", err.Error())
		return
	}
	state.Admins, d = common.StringSliceToSet(ctx, adminEmails)
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

	// Step 3: Differential admin sync (resolve emails → IDs first)
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

	// Fetch apps and admins via dedicated endpoints (not included in role detail response).
	fetchedAppIDs, err := r.fetchRoleAppIDs(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Apps", err.Error())
		return
	}
	resolvedNames, err := r.resolveAppIDsToNames(ctx, fetchedAppIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App IDs to Names", err.Error())
		return
	}
	newState.Apps, d = common.StringSliceToSet(ctx, resolvedNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	fetchedAdminIDs, err := r.fetchRoleAdminIDs(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Admins", err.Error())
		return
	}
	adminEmails, err := r.resolveAdminIDsToEmails(ctx, fetchedAdminIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Admin IDs to Emails", err.Error())
		return
	}
	newState.Admins, d = common.StringSliceToSet(ctx, adminEmails)
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

// fetchRoleAppIDs calls GET /api/2/roles/{id}/apps to get the app IDs for a role.
// The role detail endpoint (GET /api/2/roles/{id}) does NOT include apps or admins.
func (r *roleResource) fetchRoleAppIDs(ctx context.Context, roleID int) ([]int32, error) {
	result, err := r.client.SDK.GetRoleApps(roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch apps for role %d: %w", roleID, err)
	}
	if result == nil {
		return nil, nil
	}
	apps, err := client.UnmarshalApps(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse apps for role %d: %w", roleID, err)
	}
	ids := make([]int32, 0, len(apps))
	for i := range apps {
		if apps[i].ID != nil {
			ids = append(ids, *apps[i].ID)
		}
	}
	return ids, nil
}

// fetchRoleAdminIDs calls GET /api/2/roles/{id}/admins to get the admin user IDs for a role.
// The role detail endpoint (GET /api/2/roles/{id}) does NOT include apps or admins.
func (r *roleResource) fetchRoleAdminIDs(ctx context.Context, roleID int) ([]int32, error) {
	result, err := r.client.SDK.GetRoleAdmins(roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch admins for role %d: %w", roleID, err)
	}
	if result == nil {
		return nil, nil
	}
	users, err := client.UnmarshalUsers(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse admins for role %d: %w", roleID, err)
	}
	ids := make([]int32, 0, len(users))
	for i := range users {
		if users[i].ID != 0 {
			ids = append(ids, users[i].ID)
		}
	}
	return ids, nil
}

// resolveAppIDsToNames converts a slice of OneLogin app IDs to their display names.
// Results are cached on the client for the lifetime of the Terraform operation so
// that repeated reads of different roles sharing the same apps only hit the API once.
// Duplicate IDs are skipped. When two distinct IDs resolve to the same name, the ID
// is appended in parentheses (e.g. "My App (4094617)") to keep Set elements unique.
func (r *roleResource) resolveAppIDsToNames(ctx context.Context, ids []int32) ([]string, error) {
	// Warm the cache once per Terraform operation (no-op on subsequent calls).
	if err := r.client.PreloadAppCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to preload app cache: %w", err)
	}

	seenIDs := make(map[int32]struct{}, len(ids))
	seenNames := make(map[string]struct{}, len(ids))
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, dup := seenIDs[id]; dup {
			continue
		}
		seenIDs[id] = struct{}{}

		var appName string
		if cached, ok := r.client.CachedAppName(id); ok {
			tflog.Debug(ctx, "resolveAppIDsToNames: cache hit", map[string]any{"app_id": id, "name": cached})
			appName = cached
		} else {
			tflog.Debug(ctx, "resolveAppIDsToNames: cache miss, fetching from API", map[string]any{"app_id": id})
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
			appName = *app.Name
			tflog.Debug(ctx, "resolveAppIDsToNames: fetched and cached", map[string]any{"app_id": id, "name": appName})
			r.client.SetCachedAppName(id, appName)
		}

		label := appName
		if _, nameDup := seenNames[label]; nameDup {
			label = fmt.Sprintf("%s (%d)", appName, id)
		}
		seenNames[label] = struct{}{}
		names = append(names, label)
	}
	return names, nil
}

// resolveAppNamesToIDs converts a slice of OneLogin app display names to their IDs.
// Each name uses GET /api/2/apps?name=<name> and collects all exact matches.
// Returns an error if more than one app has the same name (ambiguous — include IDs
// so the operator can identify and remove the duplicate in OneLogin).
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
		var matches []int32
		for i := range apps {
			if apps[i].Name != nil && *apps[i].Name == name && apps[i].ID != nil {
				matches = append(matches, *apps[i].ID)
			}
		}
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("app %q not found in OneLogin", name)
		case 1:
			ids = append(ids, matches[0])
		default:
			return nil, fmt.Errorf("app %q is ambiguous: %d apps share this name (IDs: %v) — delete the duplicate in OneLogin before applying", name, len(matches), matches)
		}
	}
	return ids, nil
}

// syncAdmins resolves email addresses to user IDs, then performs differential add/remove.
func (r *roleResource) syncAdmins(ctx context.Context, roleID int, oldState, newPlan RoleResourceModel, resp *resource.UpdateResponse) {
	oldEmails, d := common.SetToStringSlice(ctx, oldState.Admins)
	resp.Diagnostics.Append(d...)
	newEmails, d := common.SetToStringSlice(ctx, newPlan.Admins)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldIDs, err := r.resolveAdminEmailsToIDs(ctx, oldEmails)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Old Admin Emails", err.Error())
		return
	}
	newIDs, err := r.resolveAdminEmailsToIDs(ctx, newEmails)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving New Admin Emails", err.Error())
		return
	}

	toAdd, toRemove := diffInt32Slices(oldIDs, newIDs)

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

// resolveAdminIDsToEmails converts a slice of user IDs to their email addresses.
func (r *roleResource) resolveAdminIDsToEmails(ctx context.Context, ids []int32) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	emails := make([]string, 0, len(ids))
	for _, id := range ids {
		result, err := r.client.SDK.GetUserByIDWithContext(ctx, int(id), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user ID %d: %w", id, err)
		}
		user, err := client.UnmarshalUser(result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user ID %d: %w", id, err)
		}
		if user == nil || user.Email == "" {
			return nil, fmt.Errorf("user ID %d not found or has no email", id)
		}
		emails = append(emails, user.Email)
	}
	return emails, nil
}

// resolveAdminEmailsToIDs converts a slice of email addresses to user IDs.
func (r *roleResource) resolveAdminEmailsToIDs(ctx context.Context, emails []string) ([]int32, error) {
	ids := make([]int32, 0, len(emails))
	for _, email := range emails {
		e := email
		result, err := r.client.SDK.GetUsersWithContext(ctx, &models.UserQuery{Email: &e})
		if err != nil {
			return nil, fmt.Errorf("failed to query user %q: %w", email, err)
		}
		users, err := client.UnmarshalUsers(result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse users response for %q: %w", email, err)
		}
		var found bool
		for i := range users {
			if users[i].Email == email && users[i].ID != 0 {
				ids = append(ids, users[i].ID)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("user with email %q not found in OneLogin", email)
		}
	}
	return ids, nil
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
