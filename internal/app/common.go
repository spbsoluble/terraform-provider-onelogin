package app

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"
)

// ParameterModel represents an app parameter in Terraform state.
type ParameterModel struct {
	ParamKeyName              types.String `tfsdk:"param_key_name"`
	ParamID                   types.Int64  `tfsdk:"param_id"`
	Label                     types.String `tfsdk:"label"`
	UserAttributeMappings     types.String `tfsdk:"user_attribute_mappings"`
	UserAttributeMacros       types.String `tfsdk:"user_attribute_macros"`
	AttributesTransformations types.String `tfsdk:"attributes_transformations"`
	DefaultValues             types.String `tfsdk:"default_values"`
	Values                    types.String `tfsdk:"values"`
	SkipIfBlank               types.Bool   `tfsdk:"skip_if_blank"`
	ProvisionedEntitlements   types.Bool   `tfsdk:"provisioned_entitlements"`
	IncludeInSamlAssertion    types.Bool   `tfsdk:"include_in_saml_assertion"`
}

// ProvisioningModel represents provisioning settings.
type ProvisioningModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

// BaseAppModel contains fields shared across all app resource types.
type BaseAppModel struct {
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
	RoleIDs            types.Set    `tfsdk:"role_ids"`
	Provisioning       types.Object `tfsdk:"provisioning"`
	Parameters         types.List   `tfsdk:"parameters"`
}

// BaseAppSchemaAttributes returns the shared schema attributes for all app types.
func BaseAppSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "The unique identifier of the app.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "The name of the app.",
		},
		"connector_id": schema.Int64Attribute{
			Required:    true,
			Description: "The connector ID for the app type.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"description": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "A description of the app.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"notes": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Notes about the app.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"visible": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "Whether the app is visible to users. Defaults to true.",
		},
		"allow_assumed_signin": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Whether assumed sign-in is allowed. Defaults to false.",
		},
		"brand_id": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "The brand ID for the app.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"icon_url": schema.StringAttribute{
			Computed:    true,
			Description: "The URL of the app's icon.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"auth_method": schema.Int64Attribute{
			Computed:    true,
			Description: "The authentication method.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"policy_id": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "The security policy ID.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"tab_id": schema.Int64Attribute{
			Computed:    true,
			Description: "The tab ID.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"created_at": schema.StringAttribute{
			Computed:    true,
			Description: "When the app was created.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"role_ids": schema.SetAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.Int64Type,
			Description: "Set of role IDs that can access this app.",
			PlanModifiers: []planmodifier.Set{
				setplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

// ProvisioningBlock returns the provisioning nested block schema.
func ProvisioningBlock() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Provisioning settings for the app.",
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether provisioning is enabled.",
			},
		},
	}
}

// ParametersBlock returns the parameters nested block schema.
func ParametersBlock() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Application parameters.",
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"param_key_name": schema.StringAttribute{
					Required:    true,
					Description: "The parameter key name.",
				},
				"param_id": schema.Int64Attribute{
					Computed:    true,
					Description: "The parameter ID assigned by OneLogin.",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"label": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The display label for the parameter.",
				},
				"user_attribute_mappings": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "User attribute to map to this parameter.",
				},
				"user_attribute_macros": schema.StringAttribute{
					Optional:    true,
					Description: "Macro expression for user attribute.",
				},
				"attributes_transformations": schema.StringAttribute{
					Optional:    true,
					Description: "Attribute transformation rules.",
				},
				"default_values": schema.StringAttribute{
					Optional:    true,
					Description: "Default value for the parameter.",
				},
				"values": schema.StringAttribute{
					Optional:    true,
					Description: "The parameter value.",
				},
				"skip_if_blank": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Skip this parameter if value is blank.",
				},
				"provisioned_entitlements": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Whether this parameter uses provisioned entitlements.",
				},
				"include_in_saml_assertion": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Whether to include this parameter in the SAML assertion.",
				},
			},
		},
	}
}

// ProvisioningAttrTypes returns the attr types for provisioning object.
func ProvisioningAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
	}
}

// ParameterAttrTypes returns the attr types for a parameter object.
func ParameterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"param_key_name":             types.StringType,
		"param_id":                   types.Int64Type,
		"label":                      types.StringType,
		"user_attribute_mappings":    types.StringType,
		"user_attribute_macros":      types.StringType,
		"attributes_transformations": types.StringType,
		"default_values":             types.StringType,
		"values":                     types.StringType,
		"skip_if_blank":              types.BoolType,
		"provisioned_entitlements":   types.BoolType,
		"include_in_saml_assertion":  types.BoolType,
	}
}

// BaseAppToSDK converts the base app model fields to an SDK App struct.
func BaseAppToSDK(ctx context.Context, m *BaseAppModel) (*models.App, diag.Diagnostics) {
	var diags diag.Diagnostics

	app := &models.App{
		Name:               common.StringToStringPtr(m.Name),
		ConnectorID:        common.Int64ToInt32Ptr(m.ConnectorID),
		Description:        common.StringToStringPtr(m.Description),
		Notes:              common.StringToStringPtr(m.Notes),
		Visible:            common.BoolToBoolPtr(m.Visible),
		AllowAssumedSignin: common.BoolToBoolPtr(m.AllowAssumedSignin),
		BrandID:            common.Int64ToIntPtr(m.BrandID),
		PolicyID:           common.Int64ToIntPtr(m.PolicyID),
	}

	// RoleIDs
	roleIDs, d := common.SetToIntSlice(ctx, m.RoleIDs)
	diags.Append(d...)
	app.RoleIDs = roleIDs

	// Provisioning
	if !m.Provisioning.IsNull() && !m.Provisioning.IsUnknown() {
		var prov ProvisioningModel
		d := m.Provisioning.As(ctx, &prov, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			app.Provisioning = &models.Provisioning{
				Enabled: prov.Enabled.ValueBool(),
			}
		}
	}

	// Parameters
	params, d := parametersToSDK(ctx, m.Parameters)
	diags.Append(d...)
	if params != nil {
		app.Parameters = params
	}

	return app, diags
}

// BaseAppFromSDK populates the base app model fields from an SDK App struct.
func BaseAppFromSDK(ctx context.Context, m *BaseAppModel, app *models.App) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = common.Int32PtrToInt64(app.ID)
	m.Name = common.StringPtrToString(app.Name)
	m.ConnectorID = common.Int32PtrToInt64(app.ConnectorID)
	m.Description = common.StringPtrToString(app.Description)
	m.Notes = common.NormalizeAppNotes(app.Notes)
	m.Visible = common.BoolPtrToBool(app.Visible)
	m.AllowAssumedSignin = common.BoolPtrToBool(app.AllowAssumedSignin)
	m.BrandID = common.IntPtrToInt64(app.BrandID)
	m.IconURL = common.StringPtrToString(app.IconURL)
	m.AuthMethod = common.IntPtrToInt64(app.AuthMethod)
	m.PolicyID = common.IntPtrToInt64(app.PolicyID)
	m.TabID = common.IntPtrToInt64(app.TabID)
	m.CreatedAt = common.StringPtrToString(app.CreatedAt)

	// RoleIDs
	roleIDs, d := common.IntSliceToSet(ctx, app.RoleIDs)
	diags.Append(d...)
	m.RoleIDs = roleIDs

	// Provisioning
	if app.Provisioning != nil {
		provObj, d := types.ObjectValue(ProvisioningAttrTypes(), map[string]attr.Value{
			"enabled": types.BoolValue(app.Provisioning.Enabled),
		})
		diags.Append(d...)
		m.Provisioning = provObj
	} else {
		m.Provisioning = types.ObjectNull(ProvisioningAttrTypes())
	}

	// Parameters
	params, d := parametersFromSDK(ctx, app.Parameters)
	diags.Append(d...)
	m.Parameters = params

	return diags
}

func parametersToSDK(ctx context.Context, params types.List) (*map[string]models.Parameter, diag.Diagnostics) {
	if params.IsNull() || params.IsUnknown() {
		return nil, nil
	}

	var paramModels []ParameterModel
	diags := params.ElementsAs(ctx, &paramModels, false)
	if diags.HasError() {
		return nil, diags
	}

	result := make(map[string]models.Parameter, len(paramModels))
	for _, pm := range paramModels {
		key := pm.ParamKeyName.ValueString()
		p := models.Parameter{
			Label:                   pm.Label.ValueString(),
			ProvisionedEntitlements: pm.ProvisionedEntitlements.ValueBool(),
			SkipIfBlank:             pm.SkipIfBlank.ValueBool(),
			IncludeInSamlAssertion:  pm.IncludeInSamlAssertion.ValueBool(),
		}
		if !pm.ParamID.IsNull() && !pm.ParamID.IsUnknown() {
			p.ID = int(pm.ParamID.ValueInt64())
		}
		if !pm.UserAttributeMappings.IsNull() && !pm.UserAttributeMappings.IsUnknown() {
			p.UserAttributeMappings = pm.UserAttributeMappings.ValueString()
		}
		if !pm.UserAttributeMacros.IsNull() && !pm.UserAttributeMacros.IsUnknown() {
			p.UserAttributeMacros = pm.UserAttributeMacros.ValueString()
		}
		if !pm.AttributesTransformations.IsNull() && !pm.AttributesTransformations.IsUnknown() {
			p.AttributesTransformations = pm.AttributesTransformations.ValueString()
		}
		if !pm.DefaultValues.IsNull() && !pm.DefaultValues.IsUnknown() {
			p.DefaultValues = pm.DefaultValues.ValueString()
		}
		if !pm.Values.IsNull() && !pm.Values.IsUnknown() {
			p.Values = pm.Values.ValueString()
		}
		result[key] = p
	}

	return &result, nil
}

func parametersFromSDK(ctx context.Context, params *map[string]models.Parameter) (types.List, diag.Diagnostics) {
	if params == nil || len(*params) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: ParameterAttrTypes()}, []attr.Value{}), nil
	}

	var diags diag.Diagnostics

	// Sort parameter keys alphabetically so the list is always in a
	// deterministic order regardless of how the API returns them.
	keys := make([]string, 0, len(*params))
	for key := range *params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	elems := make([]attr.Value, 0, len(*params))

	for _, key := range keys {
		p := (*params)[key]
		obj, d := types.ObjectValue(ParameterAttrTypes(), map[string]attr.Value{
			"param_key_name":             types.StringValue(key),
			"param_id":                   types.Int64Value(int64(p.ID)),
			"label":                      types.StringValue(p.Label),
			"user_attribute_mappings":    common.InterfaceToStringOrEmpty(p.UserAttributeMappings),
			"user_attribute_macros":      common.InterfaceToString(p.UserAttributeMacros),
			"attributes_transformations": common.InterfaceToString(p.AttributesTransformations),
			"default_values":             common.InterfaceToString(p.DefaultValues),
			"values":                     common.InterfaceToString(p.Values),
			"skip_if_blank":              types.BoolValue(p.SkipIfBlank),
			"provisioned_entitlements":   types.BoolValue(p.ProvisionedEntitlements),
			"include_in_saml_assertion":  types.BoolValue(p.IncludeInSamlAssertion),
		})
		diags.Append(d...)
		elems = append(elems, obj)
	}

	result, d := types.ListValue(types.ObjectType{AttrTypes: ParameterAttrTypes()}, elems)
	diags.Append(d...)
	return result, diags
}

// FilterParametersByKnownKeys filters API-returned parameters to only include those
// whose param_key_name matches a key from the reference set. This prevents server-added
// default parameters from causing plan inconsistencies. If refParams is null/unknown,
// the full apiParams set is returned unchanged.
func FilterParametersByKnownKeys(ctx context.Context, refParams types.List, apiParams types.List) (types.List, diag.Diagnostics) {
	if refParams.IsNull() || refParams.IsUnknown() {
		return apiParams, nil
	}

	var diags diag.Diagnostics

	// Extract known keys from the reference set
	var refModels []ParameterModel
	d := refParams.ElementsAs(ctx, &refModels, false)
	diags.Append(d...)
	if diags.HasError() {
		return apiParams, diags
	}
	// Build a map from param_key_name -> API param for quick lookup.
	var apiModels []ParameterModel
	d = apiParams.ElementsAs(ctx, &apiModels, false)
	diags.Append(d...)
	if diags.HasError() {
		return apiParams, diags
	}
	apiByKey := make(map[string]ParameterModel, len(apiModels))
	for _, pm := range apiModels {
		apiByKey[pm.ParamKeyName.ValueString()] = pm
	}

	// Collect the set of known keys from the reference (config/prev-state) in
	// alphabetical order so the output list is always deterministically sorted.
	knownKeys := make([]string, 0, len(refModels))
	for _, pm := range refModels {
		knownKeys = append(knownKeys, pm.ParamKeyName.ValueString())
	}
	sort.Strings(knownKeys)

	// Build the filtered list in sorted order, using API values where available.
	filtered := make([]attr.Value, 0, len(knownKeys))
	for _, key := range knownKeys {
		pm, ok := apiByKey[key]
		if !ok {
			continue
		}
		obj, d := types.ObjectValue(ParameterAttrTypes(), map[string]attr.Value{
			"param_key_name":             pm.ParamKeyName,
			"param_id":                   pm.ParamID,
			"label":                      pm.Label,
			"user_attribute_mappings":    pm.UserAttributeMappings,
			"user_attribute_macros":      pm.UserAttributeMacros,
			"attributes_transformations": pm.AttributesTransformations,
			"default_values":             pm.DefaultValues,
			"values":                     pm.Values,
			"skip_if_blank":              pm.SkipIfBlank,
			"provisioned_entitlements":   pm.ProvisionedEntitlements,
			"include_in_saml_assertion":  pm.IncludeInSamlAssertion,
		})
		diags.Append(d...)
		filtered = append(filtered, obj)
	}

	if len(filtered) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: ParameterAttrTypes()}, []attr.Value{}), diags
	}

	result, d := types.ListValue(types.ObjectType{AttrTypes: ParameterAttrTypes()}, filtered)
	diags.Append(d...)
	return result, diags
}

// StringOrNull returns a types.String from a string value, returning null for empty strings.
func StringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// IntToInt64 converts an int to types.Int64 (0 → null).
func IntToInt64(v int) types.Int64 {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(v))
}

// useStateWhenConfigNullInt64 preserves the prior state value for an Int64 attribute
// when the config value is null (user didn't specify it). This handles the case where
// UseStateForUnknown doesn't trigger because the plan value is null rather than unknown
// (which can happen for inner attributes of a SingleNestedAttribute when the user
// provides the outer object but omits the inner attribute).
type useStateWhenConfigNullInt64 struct{}

func UseStateWhenConfigNullInt64() planmodifier.Int64 {
	return useStateWhenConfigNullInt64{}
}

func (m useStateWhenConfigNullInt64) Description(_ context.Context) string {
	return "Use state value when config is null"
}

func (m useStateWhenConfigNullInt64) MarkdownDescription(_ context.Context) string {
	return "Use state value when config is null"
}

func (m useStateWhenConfigNullInt64) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Only activate when: user didn't set this attribute AND we have a prior state value
	if req.ConfigValue.IsNull() && !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}

// useStateWhenConfigNullString preserves the prior state value for a String attribute
// when the config value is null.
type useStateWhenConfigNullString struct{}

func UseStateWhenConfigNullString() planmodifier.String {
	return useStateWhenConfigNullString{}
}

func (m useStateWhenConfigNullString) Description(_ context.Context) string {
	return "Use state value when config is null"
}

func (m useStateWhenConfigNullString) MarkdownDescription(_ context.Context) string {
	return "Use state value when config is null"
}

func (m useStateWhenConfigNullString) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() && !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}
