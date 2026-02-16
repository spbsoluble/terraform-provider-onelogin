package app

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

var (
	_ datasource.DataSource              = &appsDataSource{}
	_ datasource.DataSourceWithConfigure = &appsDataSource{}
)

func NewAppsDataSource() datasource.DataSource {
	return &appsDataSource{}
}

type appsDataSource struct {
	client *client.Client
}

type appsDataSourceModel struct {
	NameFilter  types.String       `tfsdk:"name_filter"`
	ConnectorID types.Int64        `tfsdk:"connector_id"`
	Apps        []appListItemModel `tfsdk:"apps"`
}

type appListItemModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ConnectorID types.Int64  `tfsdk:"connector_id"`
	Description types.String `tfsdk:"description"`
	Visible     types.Bool   `tfsdk:"visible"`
	AuthMethod  types.Int64  `tfsdk:"auth_method"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *appsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apps"
}

func (d *appsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to list OneLogin Apps, with optional filtering.",
		Attributes: map[string]schema.Attribute{
			"name_filter": schema.StringAttribute{
				Optional:    true,
				Description: "Filter apps by name (partial match).",
			},
			"connector_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Filter apps by connector ID.",
			},
			"apps": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of apps.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"connector_id": schema.Int64Attribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"visible": schema.BoolAttribute{
							Computed: true,
						},
						"auth_method": schema.Int64Attribute{
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"updated_at": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *appsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *appsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config appsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build query
	var query *models.AppQuery
	if !config.NameFilter.IsNull() || !config.ConnectorID.IsNull() {
		query = &models.AppQuery{}
		if !config.NameFilter.IsNull() {
			name := config.NameFilter.ValueString()
			query.Name = &name
		}
		if !config.ConnectorID.IsNull() {
			connID := int(config.ConnectorID.ValueInt64())
			query.ConnectorID = &connID
		}
	}

	result, err := d.client.SDK.GetApps(query)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Apps", "Could not list apps: "+err.Error())
		return
	}

	apps, err := client.UnmarshalApps(result)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing Apps Response", err.Error())
		return
	}

	var items []appListItemModel
	for _, a := range apps {
		item := appListItemModel{
			ID:          common.Int32PtrToInt64(a.ID),
			Name:        common.StringPtrToString(a.Name),
			ConnectorID: common.Int32PtrToInt64(a.ConnectorID),
			Description: common.StringPtrToString(a.Description),
			Visible:     common.BoolPtrToBool(a.Visible),
			AuthMethod:  common.IntPtrToInt64(a.AuthMethod),
			CreatedAt:   common.StringPtrToString(a.CreatedAt),
			UpdatedAt:   common.StringPtrToString(a.UpdatedAt),
		}
		items = append(items, item)
	}

	config.Apps = items
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
