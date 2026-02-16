package common

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Int32PtrToInt64 converts *int32 to types.Int64.
func Int32PtrToInt64(v *int32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// Int64ToInt32Ptr converts types.Int64 to *int32.
func Int64ToInt32Ptr(v types.Int64) *int32 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := int32(v.ValueInt64())
	return &i
}

// StringPtrToString converts *string to types.String.
func StringPtrToString(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// StringToStringPtr converts types.String to *string.
func StringToStringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// BoolPtrToBool converts *bool to types.Bool.
func BoolPtrToBool(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

// BoolToBoolPtr converts types.Bool to *bool.
func BoolToBoolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// ParseImportID converts a string import ID to int64 for use as a resource ID.
func ParseImportID(id string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid ID %q: must be a numeric value", id)
	}
	return parsed, nil
}

// SetToInt32Slice converts a types.Set of Int64 to []int32.
func SetToInt32Slice(ctx context.Context, s types.Set) ([]int32, diag.Diagnostics) {
	if s.IsNull() || s.IsUnknown() {
		return nil, nil
	}
	var vals []types.Int64
	diags := s.ElementsAs(ctx, &vals, false)
	if diags.HasError() {
		return nil, diags
	}
	result := make([]int32, len(vals))
	for i, v := range vals {
		result[i] = int32(v.ValueInt64())
	}
	return result, nil
}

// Int32SliceToSet converts []int32 to a types.Set of Int64.
func Int32SliceToSet(ctx context.Context, vals []int32) (types.Set, diag.Diagnostics) {
	if vals == nil {
		return types.SetValueMust(types.Int64Type, []attr.Value{}), nil
	}
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.Int64Value(int64(v))
	}
	return types.SetValue(types.Int64Type, elems)
}

// Int32SliceToIntSlice converts []int32 to []int for SDK methods that expect []int.
func Int32SliceToIntSlice(vals []int32) []int {
	result := make([]int, len(vals))
	for i, v := range vals {
		result[i] = int(v)
	}
	return result
}

// IntPtrToInt64 converts *int to types.Int64.
func IntPtrToInt64(v *int) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// Int64ToIntPtr converts types.Int64 to *int.
func Int64ToIntPtr(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := int(v.ValueInt64())
	return &i
}

// IntSliceToSet converts *[]int to a types.Set of Int64.
func IntSliceToSet(ctx context.Context, vals *[]int) (types.Set, diag.Diagnostics) {
	if vals == nil || len(*vals) == 0 {
		return types.SetValueMust(types.Int64Type, []attr.Value{}), nil
	}
	elems := make([]attr.Value, len(*vals))
	for i, v := range *vals {
		elems[i] = types.Int64Value(int64(v))
	}
	return types.SetValue(types.Int64Type, elems)
}

// SetToIntSlice converts a types.Set of Int64 to *[]int.
func SetToIntSlice(ctx context.Context, s types.Set) (*[]int, diag.Diagnostics) {
	if s.IsNull() || s.IsUnknown() {
		return nil, nil
	}
	var vals []types.Int64
	diags := s.ElementsAs(ctx, &vals, false)
	if diags.HasError() {
		return nil, diags
	}
	result := make([]int, len(vals))
	for i, v := range vals {
		result[i] = int(v.ValueInt64())
	}
	return &result, nil
}

// InterfaceToString converts an interface{} value to types.String.
func InterfaceToString(v interface{}) types.String {
	if v == nil {
		return types.StringNull()
	}
	switch val := v.(type) {
	case string:
		return types.StringValue(val)
	default:
		return types.StringValue(fmt.Sprintf("%v", val))
	}
}
