package user_mapping

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
)

var (
	_ datasource.DataSource              = &userMappingsDataSource{}
	_ datasource.DataSourceWithConfigure = &userMappingsDataSource{}
)

func NewUserMappingsDataSource() datasource.DataSource {
	return &userMappingsDataSource{}
}

type userMappingsDataSource struct {
	client *client.Client
}

type userMappingsDataSourceModel struct {
	Enabled      types.Bool                 `tfsdk:"enabled"`
	HasCondition types.String               `tfsdk:"has_condition"`
	HasAction    types.String               `tfsdk:"has_action"`
	Mappings     []UserMappingResourceModel `tfsdk:"mappings"`
}

func (d *userMappingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_mappings"
}

func (d *userMappingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to list OneLogin User Mappings, with optional filtering.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Filter by enabled status.",
			},
			"has_condition": schema.StringAttribute{
				Optional:    true,
				Description: "Filter by condition in the format \"source:value\" (e.g., \"email:@test.com\").",
			},
			"has_action": schema.StringAttribute{
				Optional:    true,
				Description: "Filter by action type (e.g., \"set_role\").",
			},
			"mappings": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of user mappings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"match": schema.StringAttribute{
							Computed: true,
						},
						"enabled": schema.BoolAttribute{
							Computed: true,
						},
						"position": schema.Int64Attribute{
							Computed: true,
						},
						"conditions": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"source": schema.StringAttribute{
										Computed: true,
									},
									"operator": schema.StringAttribute{
										Computed: true,
									},
									"value": schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
						"actions": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"action": schema.StringAttribute{
										Computed: true,
									},
									"value": schema.ListAttribute{
										Computed:    true,
										ElementType: types.StringType,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *userMappingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userMappingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userMappingsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build query params
	var query *models.UserMappingsQuery
	if !config.Enabled.IsNull() || !config.HasCondition.IsNull() || !config.HasAction.IsNull() {
		query = &models.UserMappingsQuery{}
		if !config.Enabled.IsNull() {
			if config.Enabled.ValueBool() {
				query.Enabled = "true"
			} else {
				query.Enabled = "false"
			}
		}
		if !config.HasCondition.IsNull() {
			query.HasCondition = config.HasCondition.ValueString()
		}
		if !config.HasAction.IsNull() {
			query.HasAction = config.HasAction.ValueString()
		}
	}

	mappings, err := d.client.SDK.ListUserMappings(query)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User Mappings", "Could not list user mappings: "+err.Error())
		return
	}

	var stateMappings []UserMappingResourceModel
	for _, m := range mappings {
		var model UserMappingResourceModel
		mCopy := m
		diags = model.FromSDKUserMapping(ctx, &mCopy)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateMappings = append(stateMappings, model)
	}

	config.Mappings = stateMappings
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
