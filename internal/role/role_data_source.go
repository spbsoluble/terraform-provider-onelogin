package role

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
)

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
		Description: "Use this data source to get information about a specific OneLogin Role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "The ID of the role.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the role.",
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
}
