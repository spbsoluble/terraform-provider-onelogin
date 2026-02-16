package user_mapping

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
)

// UserMappingResourceModel describes the resource data model.
type UserMappingResourceModel struct {
	ID         types.Int64      `tfsdk:"id"`
	Name       types.String     `tfsdk:"name"`
	Match      types.String     `tfsdk:"match"`
	Enabled    types.Bool       `tfsdk:"enabled"`
	Position   types.Int64      `tfsdk:"position"`
	Conditions []ConditionModel `tfsdk:"conditions"`
	Actions    []ActionModel    `tfsdk:"actions"`
}

// ConditionModel represents a single condition in a user mapping.
type ConditionModel struct {
	Source   types.String `tfsdk:"source"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

// ActionModel represents a single action in a user mapping.
type ActionModel struct {
	Action types.String   `tfsdk:"action"`
	Value  []types.String `tfsdk:"value"`
}

// ToSDKUserMapping converts the Terraform model to the SDK UserMapping struct.
func (m *UserMappingResourceModel) ToSDKUserMapping(_ context.Context) (*models.UserMapping, diag.Diagnostics) {
	mapping := &models.UserMapping{
		Name:    common.StringToStringPtr(m.Name),
		Match:   common.StringToStringPtr(m.Match),
		Enabled: common.BoolToBoolPtr(m.Enabled),
	}

	if !m.ID.IsNull() && !m.ID.IsUnknown() {
		id := int32(m.ID.ValueInt64())
		mapping.ID = &id
	}

	if !m.Position.IsNull() && !m.Position.IsUnknown() {
		pos := int32(m.Position.ValueInt64())
		mapping.Position = &pos
	}

	// Convert conditions
	if m.Conditions != nil {
		conditions := make([]models.UserMappingConditions, len(m.Conditions))
		for i, c := range m.Conditions {
			conditions[i] = models.UserMappingConditions{
				Source:   common.StringToStringPtr(c.Source),
				Operator: common.StringToStringPtr(c.Operator),
				Value:    common.StringToStringPtr(c.Value),
			}
		}
		mapping.Conditions = conditions
	}

	// Convert actions
	if m.Actions != nil {
		actions := make([]models.UserMappingActions, len(m.Actions))
		for i, a := range m.Actions {
			values := make([]string, len(a.Value))
			for j, v := range a.Value {
				values[j] = v.ValueString()
			}
			actions[i] = models.UserMappingActions{
				Action: common.StringToStringPtr(a.Action),
				Value:  values,
			}
		}
		mapping.Actions = actions
	}

	return mapping, nil
}

// FromSDKUserMapping populates the Terraform model from an SDK UserMapping struct.
func (m *UserMappingResourceModel) FromSDKUserMapping(_ context.Context, mapping *models.UserMapping) diag.Diagnostics {
	m.ID = common.Int32PtrToInt64(mapping.ID)
	m.Name = common.StringPtrToString(mapping.Name)
	m.Match = common.StringPtrToString(mapping.Match)
	m.Enabled = common.BoolPtrToBool(mapping.Enabled)
	m.Position = common.Int32PtrToInt64(mapping.Position)

	// Convert conditions
	if mapping.Conditions != nil {
		conditions := make([]ConditionModel, len(mapping.Conditions))
		for i, c := range mapping.Conditions {
			conditions[i] = ConditionModel{
				Source:   common.StringPtrToString(c.Source),
				Operator: common.StringPtrToString(c.Operator),
				Value:    common.StringPtrToString(c.Value),
			}
		}
		m.Conditions = conditions
	} else {
		m.Conditions = []ConditionModel{}
	}

	// Convert actions
	if mapping.Actions != nil {
		actions := make([]ActionModel, len(mapping.Actions))
		for i, a := range mapping.Actions {
			values := make([]types.String, len(a.Value))
			for j, v := range a.Value {
				values[j] = types.StringValue(v)
			}
			actions[i] = ActionModel{
				Action: common.StringPtrToString(a.Action),
				Value:  values,
			}
		}
		m.Actions = actions
	} else {
		m.Actions = []ActionModel{}
	}

	return nil
}
