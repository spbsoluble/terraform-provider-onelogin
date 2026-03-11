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
)

// roleNameQuery implements mod.Queryable and passes ?name=<Name> to the roles API.
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
			"users": schema.SetAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "A set of user IDs assigned to this role.",
			},
			"apps": schema.SetAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "A set of app IDs accessible by this role.",
			},
			"admins": schema.SetAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "A set of user IDs who administer this role.",
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

	if !config.ID.IsNull() && !config.ID.IsUnknown() {
		// ID-based lookup (original path)
		id := int(config.ID.ValueInt64())
		result, err := d.client.SDK.GetRoleByIDWithContext(ctx, id, nil)
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
			resp.Diagnostics.AddError("Role Not Found", fmt.Sprintf("Role with ID %d not found.", id))
			return
		}

		var state RoleResourceModel
		diags = state.FromSDKRole(ctx, role)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		return
	}

	// Name-based lookup: use the ?name= query filter to fetch only matching roles.
	// This avoids fetching all pages of roles (accounts may have hundreds).
	name := config.Name.ValueString()
	result, err := d.client.SDK.GetRolesWithContext(ctx, &roleNameQuery{Name: name})
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Roles", "Could not list roles: "+err.Error())
		return
	}

	roles, err := client.UnmarshalRoles(result)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Roles Response", err.Error())
		return
	}

	for _, r := range roles {
		if r.Name != nil && *r.Name == name {
			var state RoleResourceModel
			diags = state.FromSDKRole(ctx, &r)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			diags = resp.State.Set(ctx, &state)
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Role Not Found",
		fmt.Sprintf("No role with name %q was found.", name),
	)
}
