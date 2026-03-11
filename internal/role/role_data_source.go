package role

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	schemavalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

// roleNameQuery implements models.Queryable and passes ?name=<Name> to the roles API.
type roleNameQuery struct {
	Name string `json:"name,omitempty"`
}

func (q *roleNameQuery) GetKeyValidators() map[string]func(interface{}) bool {
	return map[string]func(interface{}) bool{
		"name": func(v interface{}) bool { _, ok := v.(string); return ok },
	}
}

var (
	_ datasource.DataSource              = &roleDataSource{}
	_ datasource.DataSourceWithConfigure = &roleDataSource{}
)

func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	client *client.Client
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to look up a OneLogin Role by ID or by name. Exactly one of 'id' or 'name' must be specified.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the role. Exactly one of 'id' or 'name' must be set.",
				Validators: []schemavalidator.Int64{
					int64validator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the role. Exactly one of 'id' or 'name' must be set.",
				Validators: []schemavalidator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("id")),
				},
			},
			"apps": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of app names accessible by this role.",
			},
			"admins": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of email addresses for users who administer this role.",
			},
		},
	}
}

func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config RoleResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var matchedID int

	if !config.ID.IsNull() && !config.ID.IsUnknown() {
		// Lookup by ID.
		matchedID = int(config.ID.ValueInt64())
	} else {
		// Lookup by name using ?name= filter to avoid full pagination.
		name := config.Name.ValueString()
		if name == "" {
			resp.Diagnostics.AddError("Missing Argument", "Either id or name must be specified.")
			return
		}
		result, err := d.client.SDK.GetRolesWithContext(ctx, &roleNameQuery{Name: name})
		if err != nil {
			resp.Diagnostics.AddError("Error Querying Role", fmt.Sprintf("Could not query role %q: %s", name, err.Error()))
			return
		}
		roles, err := client.UnmarshalRoles(result)
		if err != nil {
			resp.Diagnostics.AddError("Error Parsing Role Response", err.Error())
			return
		}
		var found bool
		for i := range roles {
			if roles[i].Name != nil && *roles[i].Name == name && roles[i].ID != nil {
				matchedID = int(*roles[i].ID)
				found = true
				break
			}
		}
		if !found {
			resp.Diagnostics.AddError("Role Not Found", fmt.Sprintf("No role named %q found.", name))
			return
		}
	}

	// Fetch the full role by ID to get apps and admins.
	roleResult, err := d.client.SDK.GetRoleByIDWithContext(ctx, matchedID, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Role", fmt.Sprintf("Could not read role ID %d: %s", matchedID, err.Error()))
		return
	}
	role, err := client.UnmarshalRole(roleResult)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Role Response", err.Error())
		return
	}
	if role == nil {
		resp.Diagnostics.AddError("Role Not Found", fmt.Sprintf("Role ID %d not found.", matchedID))
		return
	}

	var state RoleResourceModel
	diags = state.FromSDKRole(ctx, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reuse roleResource helpers for app/admin name resolution.
	// Note: role detail endpoint does NOT include apps/admins; use dedicated endpoints.
	rr := &roleResource{client: d.client}

	fetchedAppIDs, err := rr.fetchRoleAppIDs(ctx, matchedID)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Apps", err.Error())
		return
	}
	appNames, err := rr.resolveAppIDsToNames(ctx, fetchedAppIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving App IDs", err.Error())
		return
	}
	state.Apps, diags = common.StringSliceToSet(ctx, appNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fetchedAdminIDs, err := rr.fetchRoleAdminIDs(ctx, matchedID)
	if err != nil {
		resp.Diagnostics.AddError("Error Fetching Role Admins", err.Error())
		return
	}
	adminEmails, err := rr.resolveAdminIDsToEmails(ctx, fetchedAdminIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error Resolving Admin IDs", err.Error())
		return
	}
	state.Admins, diags = common.StringSliceToSet(ctx, adminEmails)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
