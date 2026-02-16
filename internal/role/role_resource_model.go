package role

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/common"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
)

// RoleResourceModel describes the resource data model.
type RoleResourceModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Users  types.Set    `tfsdk:"users"`
	Apps   types.Set    `tfsdk:"apps"`
	Admins types.Set    `tfsdk:"admins"`
}

// ToSDKRole converts the Terraform model to the SDK Role struct.
func (m *RoleResourceModel) ToSDKRole(ctx context.Context) (*models.Role, diag.Diagnostics) {
	var diags diag.Diagnostics

	role := &models.Role{
		Name: common.StringToStringPtr(m.Name),
	}

	if !m.ID.IsNull() && !m.ID.IsUnknown() {
		id := int32(m.ID.ValueInt64())
		role.ID = &id
	}

	users, d := common.SetToInt32Slice(ctx, m.Users)
	diags.Append(d...)
	role.Users = users

	apps, d := common.SetToInt32Slice(ctx, m.Apps)
	diags.Append(d...)
	role.Apps = apps

	admins, d := common.SetToInt32Slice(ctx, m.Admins)
	diags.Append(d...)
	role.Admins = admins

	return role, diags
}

// FromSDKRole populates the Terraform model from an SDK Role struct.
func (m *RoleResourceModel) FromSDKRole(ctx context.Context, role *models.Role) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = common.Int32PtrToInt64(role.ID)
	m.Name = common.StringPtrToString(role.Name)

	users, d := common.Int32SliceToSet(ctx, role.Users)
	diags.Append(d...)
	m.Users = users

	apps, d := common.Int32SliceToSet(ctx, role.Apps)
	diags.Append(d...)
	m.Apps = apps

	admins, d := common.Int32SliceToSet(ctx, role.Admins)
	diags.Append(d...)
	m.Admins = admins

	return diags
}
