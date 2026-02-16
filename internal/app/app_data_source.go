package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
)

var (
	_ datasource.DataSource              = &appDataSource{}
	_ datasource.DataSourceWithConfigure = &appDataSource{}
)

func NewAppDataSource() datasource.DataSource {
	return &appDataSource{}
}

type appDataSource struct {
	client *client.Client
}

// appDataSourceModel is used for reading a single app by ID.
type appDataSourceModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	ConnectorID        types.Int64  `tfsdk:"connector_id"`
	Description        types.String `tfsdk:"description"`
	Notes              types.String `tfsdk:"notes"`
	Visible            types.Bool   `tfsdk:"visible"`
	AllowAssumedSignin types.Bool   `tfsdk:"allow_assumed_signin"`
	BrandID            types.Int64  `tfsdk:"brand_id"`
	IconURL            types.String `tfsdk:"icon_url"`
	AuthMethod         types.Int64  `tfsdk:"auth_method"`
	PolicyID           types.Int64  `tfsdk:"policy_id"`
	TabID              types.Int64  `tfsdk:"tab_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	Configuration      types.String `tfsdk:"configuration"`
	SSO                types.String `tfsdk:"sso"`
}

func (d *appDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *appDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Use this data source to read a OneLogin App by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "The app ID to look up.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the app.",
			},
			"connector_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The connector ID for the app type.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A description of the app.",
			},
			"notes": schema.StringAttribute{
				Computed:    true,
				Description: "Notes about the app.",
			},
			"visible": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the app is visible to users.",
			},
			"allow_assumed_signin": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether assumed sign-in is allowed.",
			},
			"brand_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The brand ID for the app.",
			},
			"icon_url": schema.StringAttribute{
				Computed:    true,
				Description: "The URL of the app's icon.",
			},
			"auth_method": schema.Int64Attribute{
				Computed:    true,
				Description: "The authentication method.",
			},
			"policy_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The security policy ID.",
			},
			"tab_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The tab ID.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the app was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the app was last updated.",
			},
			"configuration": schema.StringAttribute{
				Computed:    true,
				Description: "App configuration as a JSON string.",
			},
			"sso": schema.StringAttribute{
				Computed:    true,
				Description: "SSO settings as a JSON string.",
			},
		},
	}
}

func (d *appDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *appDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config appDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := int(config.ID.ValueInt64())
	tflog.Debug(ctx, "Reading app data source", map[string]any{"id": id})

	result, err := d.client.SDK.GetAppByID(id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading App", fmt.Sprintf("Could not read app ID %d: %s", id, err.Error()))
		return
	}

	app, err := client.UnmarshalApp(result)
	if err != nil {
		resp.Diagnostics.AddError("Error Parsing App Response", err.Error())
		return
	}
	if app == nil {
		resp.Diagnostics.AddError("App Not Found", fmt.Sprintf("App ID %d not found", id))
		return
	}

	config.ID = types.Int64Value(int64(*app.ID))
	if app.Name != nil {
		config.Name = types.StringValue(*app.Name)
	}
	if app.ConnectorID != nil {
		config.ConnectorID = types.Int64Value(int64(*app.ConnectorID))
	}
	if app.Description != nil {
		config.Description = types.StringValue(*app.Description)
	} else {
		config.Description = types.StringNull()
	}
	if app.Notes != nil {
		config.Notes = types.StringValue(*app.Notes)
	} else {
		config.Notes = types.StringNull()
	}
	if app.Visible != nil {
		config.Visible = types.BoolValue(*app.Visible)
	}
	if app.AllowAssumedSignin != nil {
		config.AllowAssumedSignin = types.BoolValue(*app.AllowAssumedSignin)
	}
	if app.BrandID != nil {
		config.BrandID = types.Int64Value(int64(*app.BrandID))
	} else {
		config.BrandID = types.Int64Null()
	}
	if app.IconURL != nil {
		config.IconURL = types.StringValue(*app.IconURL)
	} else {
		config.IconURL = types.StringNull()
	}
	if app.AuthMethod != nil {
		config.AuthMethod = types.Int64Value(int64(*app.AuthMethod))
	} else {
		config.AuthMethod = types.Int64Null()
	}
	if app.PolicyID != nil {
		config.PolicyID = types.Int64Value(int64(*app.PolicyID))
	} else {
		config.PolicyID = types.Int64Null()
	}
	if app.TabID != nil {
		config.TabID = types.Int64Value(int64(*app.TabID))
	} else {
		config.TabID = types.Int64Null()
	}
	if app.CreatedAt != nil {
		config.CreatedAt = types.StringValue(*app.CreatedAt)
	} else {
		config.CreatedAt = types.StringNull()
	}
	if app.UpdatedAt != nil {
		config.UpdatedAt = types.StringValue(*app.UpdatedAt)
	} else {
		config.UpdatedAt = types.StringNull()
	}

	// Configuration: serialize to JSON
	if app.Configuration != nil {
		b, err := json.Marshal(app.Configuration)
		if err == nil {
			config.Configuration = types.StringValue(string(b))
		} else {
			config.Configuration = types.StringNull()
		}
	} else {
		config.Configuration = types.StringNull()
	}

	// SSO: serialize to JSON
	if app.SSO != nil {
		b, err := json.Marshal(app.SSO)
		if err == nil {
			config.SSO = types.StringValue(string(b))
		} else {
			config.SSO = types.StringNull()
		}
	} else {
		config.SSO = types.StringNull()
	}

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
