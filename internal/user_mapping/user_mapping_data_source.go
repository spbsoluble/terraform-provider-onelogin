package user_mapping

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
)

var (
	_ datasource.DataSource              = &userMappingDataSource{}
	_ datasource.DataSourceWithConfigure = &userMappingDataSource{}
)

func NewUserMappingDataSource() datasource.DataSource {
	return &userMappingDataSource{}
}

type userMappingDataSource struct {
	client *client.Client
}

func (d *userMappingDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_mapping"
}

func (d *userMappingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to get information about a specific OneLogin User Mapping.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "The ID of the user mapping.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the user mapping.",
			},
			"match": schema.StringAttribute{
				Computed:    true,
				Description: "Indicates how conditions are matched.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the user mapping is enabled.",
			},
			"position": schema.Int64Attribute{
				Computed:    true,
				Description: "The position in evaluation order.",
			},
		},
		Blocks: map[string]schema.Block{
			"conditions": schema.ListNestedBlock{
				Description: "The conditions for this mapping.",
				NestedObject: schema.NestedBlockObject{
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
			"actions": schema.ListNestedBlock{
				Description: "The actions for this mapping.",
				NestedObject: schema.NestedBlockObject{
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
	}
}

func (d *userMappingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userMappingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserMappingResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int32(config.ID.ValueInt64())
	result, err := d.client.SDK.GetUserMapping(id)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User Mapping", fmt.Sprintf("Could not read user mapping ID %d: %s", id, err.Error()))
		return
	}
	if result == nil {
		resp.Diagnostics.AddError("User Mapping Not Found", fmt.Sprintf("User mapping with ID %d not found.", id))
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
