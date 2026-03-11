package role

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
)


// RoleResourceModel describes the resource data model.
// Users are intentionally absent — role membership is managed via OneLogin mappings, not
// Terraform. See docs/resources/role.md for details. Admins are tracked as email addresses
// since they are typically a small, stable list of administrators.
type RoleResourceModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Apps   types.Set    `tfsdk:"apps"`
	Admins types.Set    `tfsdk:"admins"`
}

// ToSDKRole converts the Terraform model to the SDK Role struct.
// Note: apps and admins are handled separately at the resource level.
func (m *RoleResourceModel) ToSDKRole() *models.Role {
	role := &models.Role{
		Name: common.StringToStringPtr(m.Name),
	}
	if !m.ID.IsNull() && !m.ID.IsUnknown() {
		id := int32(m.ID.ValueInt64())
		role.ID = &id
	}
	return role
}

// FromSDKRole populates the Terraform model from an SDK Role struct.
// Note: apps and admins are NOT populated here — the resource resolves IDs to human-readable
// names before storing in state.
func (m *RoleResourceModel) FromSDKRole(_ context.Context, role *models.Role) diag.Diagnostics {
	m.ID = common.Int32PtrToInt64(role.ID)
	m.Name = common.StringPtrToString(role.Name)
	return nil
}
