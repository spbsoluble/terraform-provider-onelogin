package role

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
)

var (
	_ datasource.DataSource              = &rolesDataSource{}
	_ datasource.DataSourceWithConfigure = &rolesDataSource{}
)

func NewRolesDataSource() datasource.DataSource {
	return &rolesDataSource{}
}

type rolesDataSource struct {
	client *client.Client
}

type rolesDataSourceModel struct {
	NameFilter types.String        `tfsdk:"name_filter"`
	Roles      []RoleResourceModel `tfsdk:"roles"`
}

func (d *rolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *rolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to list OneLogin Roles, with optional name filtering.",
		Attributes: map[string]schema.Attribute{
			"name_filter": schema.StringAttribute{
				Optional:    true,
				Description: "A string to filter roles by name (case-insensitive substring match).",
			},
			"roles": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of roles.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"users": schema.SetAttribute{
							Computed:    true,
							ElementType: types.Int64Type,
						},
						"apps": schema.SetAttribute{
							Computed:    true,
							ElementType: types.Int64Type,
						},
						"admins": schema.SetAttribute{
							Computed:    true,
							ElementType: types.Int64Type,
						},
					},
				},
			},
		},
	}
}

func (d *rolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *rolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config rolesDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.SDK.GetRolesWithContext(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Roles", "Could not list roles: "+err.Error())
		return
	}

	roles, err := client.UnmarshalRoles(result)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Roles Response", err.Error())
		return
	}

	nameFilter := ""
	if !config.NameFilter.IsNull() && !config.NameFilter.IsUnknown() {
		nameFilter = strings.ToLower(config.NameFilter.ValueString())
	}

	var stateRoles []RoleResourceModel
	for _, r := range roles {
		if nameFilter != "" {
			if r.Name == nil || !strings.Contains(strings.ToLower(*r.Name), nameFilter) {
				continue
			}
		}
		var model RoleResourceModel
		diags = model.FromSDKRole(ctx, &r)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateRoles = append(stateRoles, model)
	}

	config.Roles = stateRoles
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
