package provider

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/app"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/client"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/role"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/user_mapping"

	ol "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin"
)

var _ provider.Provider = &oneloginProvider{}

// New returns a factory function for creating the provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &oneloginProvider{
			version: version,
		}
	}
}

type oneloginProviderModel struct {
	ApiUrl       types.String `tfsdk:"api_url"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Timeout      types.Int64  `tfsdk:"timeout"`
}

type oneloginProvider struct {
	version string
}

func (p *oneloginProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = common.ProviderName
	resp.Version = p.version
}

func (p *oneloginProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The OneLogin provider allows you to manage OneLogin resources.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "OneLogin API URL (e.g. https://api.us.onelogin.com). Can also be set via ONELOGIN_API_URL.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "OneLogin API client ID. Can also be set via ONELOGIN_CLIENT_ID.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OneLogin API client secret. Can also be set via ONELOGIN_CLIENT_SECRET.",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "Timeout in seconds for API operations. Defaults to 180.",
			},
		},
	}
}

func (p *oneloginProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config oneloginProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve values from config or environment
	apiURL := resolveStringValue(config.ApiUrl, common.EnvAPIURL)
	clientID := resolveStringValue(config.ClientId, common.EnvClientID)
	clientSecret := resolveStringValue(config.ClientSecret, common.EnvClientSecret)
	timeout := resolveTimeoutValue(config.Timeout)

	// Validate required fields
	if apiURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing API URL",
			"The OneLogin API URL must be set in the provider configuration or via the "+common.EnvAPIURL+" environment variable.",
		)
	}
	if clientID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Missing Client ID",
			"The OneLogin client ID must be set in the provider configuration or via the "+common.EnvClientID+" environment variable.",
		)
	}
	if clientSecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Missing Client Secret",
			"The OneLogin client secret must be set in the provider configuration or via the "+common.EnvClientSecret+" environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Set environment variables for the SDK (SDK reads from env)
	os.Setenv(common.EnvAPIURL, apiURL)
	os.Setenv(common.EnvClientID, clientID)
	os.Setenv(common.EnvClientSecret, clientSecret)
	os.Setenv(common.EnvTimeout, strconv.FormatInt(timeout, 10))
	os.Setenv(common.EnvClientTimeout, strconv.FormatInt(timeout, 10))

	// Extract subdomain from API URL for SDK compatibility
	subdomain := extractSubdomain(apiURL)
	if subdomain == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Invalid API URL",
			"Could not extract subdomain from API URL. Expected format: https://api.us.onelogin.com or https://api.eu.onelogin.com",
		)
		return
	}
	os.Setenv(common.EnvSubdomain, subdomain)

	// Initialize the SDK
	sdk, err := ol.NewOneloginSDK()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create OneLogin SDK Client",
			"An unexpected error occurred when creating the OneLogin SDK client: "+err.Error(),
		)
		return
	}

	c := &client.Client{
		SDK:    sdk,
		APIURL: apiURL,
	}

	resp.DataSourceData = c
	resp.ResourceData = c

	tflog.Info(ctx, "Configured OneLogin client", map[string]any{"api_url": apiURL})
}

func (p *oneloginProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		role.NewRoleResource,
		user_mapping.NewUserMappingResource,
		app.NewAppResource,
		app.NewOIDCAppResource,
		app.NewSAMLAppResource,
	}
}

func (p *oneloginProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		role.NewRoleDataSource,
		// role.NewRolesDataSource, // Disabled: listing all roles causes API timeout

		user_mapping.NewUserMappingDataSource,
		user_mapping.NewUserMappingsDataSource,
		app.NewAppDataSource,
		app.NewAppsDataSource,
	}
}

// resolveStringValue returns the config value if set, otherwise falls back to the env var.
func resolveStringValue(v types.String, envVar string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	return os.Getenv(envVar)
}

// resolveTimeoutValue returns the config timeout or the env var timeout, defaulting to DefaultTimeout.
func resolveTimeoutValue(v types.Int64) int64 {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueInt64()
	}
	if envVal := os.Getenv(common.EnvTimeout); envVal != "" {
		if parsed, err := strconv.ParseInt(envVal, 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return common.DefaultTimeout
}

// extractSubdomain extracts a subdomain value for the SDK from an API URL.
// The SDK constructs URLs as https://{subdomain}.onelogin.com/...
// So for api.us.onelogin.com, the subdomain must be "api.us".
func extractSubdomain(apiURL string) string {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(apiURL, "https://"), "http://")
	// Remove trailing slashes
	trimmed = strings.TrimRight(trimmed, "/")
	// Expected format: {something}.onelogin.com
	// We need the part before ".onelogin.com"
	suffix := ".onelogin.com"
	if idx := strings.Index(trimmed, suffix); idx > 0 {
		return trimmed[:idx]
	}
	return ""
}
