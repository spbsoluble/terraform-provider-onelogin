package app

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// paramSpec is a compact description of a parameters list element for tests.
type paramSpec struct {
	key string
	// id < 0 means "unknown", id == 0 means "null", id > 0 means that value.
	id int64
}

const (
	paramIDUnknown int64 = -1
	paramIDNull    int64 = 0
)

func buildParamsList(t *testing.T, specs []paramSpec) types.List {
	t.Helper()
	elems := make([]attr.Value, 0, len(specs))
	for _, s := range specs {
		var id types.Int64
		switch {
		case s.id == paramIDUnknown:
			id = types.Int64Unknown()
		case s.id == paramIDNull:
			id = types.Int64Null()
		default:
			id = types.Int64Value(s.id)
		}
		obj, d := types.ObjectValue(ParameterAttrTypes(), map[string]attr.Value{
			"param_key_name":             types.StringValue(s.key),
			"param_id":                   id,
			"label":                      types.StringValue(s.key),
			"user_attribute_mappings":    types.StringNull(),
			"user_attribute_macros":      types.StringNull(),
			"attributes_transformations": types.StringNull(),
			"default_values":             types.StringNull(),
			"values":                     types.StringNull(),
			"skip_if_blank":              types.BoolValue(false),
			"provisioned_entitlements":   types.BoolValue(false),
			"include_in_saml_assertion":  types.BoolValue(true),
		})
		if d.HasError() {
			t.Fatalf("build object for %q: %v", s.key, d)
		}
		elems = append(elems, obj)
	}
	lv, d := types.ListValue(types.ObjectType{AttrTypes: ParameterAttrTypes()}, elems)
	if d.HasError() {
		t.Fatalf("build list: %v", d)
	}
	return lv
}

func runParamModifier(t *testing.T, plan, state types.List) []ParameterModel {
	t.Helper()
	ctx := context.Background()
	resp := &planmodifier.ListResponse{PlanValue: plan}
	UseStateForUnknownParametersByKey().PlanModifyList(ctx, planmodifier.ListRequest{
		PlanValue:  plan,
		StateValue: state,
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("modifier diagnostics: %v", resp.Diagnostics)
	}
	var got []ParameterModel
	if d := resp.PlanValue.ElementsAs(ctx, &got, false); d.HasError() {
		t.Fatalf("decode result: %v", d)
	}
	return got
}

func idOf(t *testing.T, models []ParameterModel, key string) types.Int64 {
	t.Helper()
	for _, m := range models {
		if m.ParamKeyName.ValueString() == key {
			return m.ParamID
		}
	}
	t.Fatalf("key %q not found in result", key)
	return types.Int64Null()
}

// TestParamModifier_CarriesIDByKeyWhenSetChanges is the regression test for the
// "Provider produced inconsistent result after apply" bug on
// velo_ops_jenkins_prod_2 / _non_prd_2. A new parameter is inserted in
// alphabetical order between two existing parameters. The computed param_id must
// stay bound to its OWN key — not be carried over by list index, which is what
// the old per-element int64planmodifier.UseStateForUnknown did.
func TestParamModifier_CarriesIDByKeyWhenSetChanges(t *testing.T) {
	// Prior state: two params, sorted by key.
	state := buildParamsList(t, []paramSpec{
		{key: "aaa_existing", id: 243476},
		{key: "zzz_existing", id: 883788},
	})
	// Plan (regenerated config): a new "mmm_new" param inserted; all param_ids
	// unknown because the config never sets them.
	plan := buildParamsList(t, []paramSpec{
		{key: "aaa_existing", id: paramIDUnknown},
		{key: "mmm_new", id: paramIDUnknown},
		{key: "zzz_existing", id: paramIDUnknown},
	})

	got := runParamModifier(t, plan, state)

	if len(got) != 3 {
		t.Fatalf("expected 3 params, got %d", len(got))
	}
	if v := idOf(t, got, "aaa_existing"); v.ValueInt64() != 243476 {
		t.Errorf("aaa_existing param_id = %v, want 243476", v)
	}
	if v := idOf(t, got, "zzz_existing"); v.ValueInt64() != 883788 {
		t.Errorf("zzz_existing param_id = %v, want 883788", v)
	}
	// The brand-new key has no prior state — it must stay unknown for apply.
	if v := idOf(t, got, "mmm_new"); !v.IsUnknown() {
		t.Errorf("mmm_new param_id = %v, want unknown (new key, no prior state)", v)
	}
}

// TestParamModifier_RemovedKeyDoesNotShiftIDs guards the reverse case: removing a
// param that sorts first must not shift the later params' ids onto the wrong keys.
func TestParamModifier_RemovedKeyDoesNotShiftIDs(t *testing.T) {
	state := buildParamsList(t, []paramSpec{
		{key: "aaa", id: 100},
		{key: "bbb", id: 200},
		{key: "ccc", id: 300},
	})
	// "aaa" removed from config; remaining ids unknown.
	plan := buildParamsList(t, []paramSpec{
		{key: "bbb", id: paramIDUnknown},
		{key: "ccc", id: paramIDUnknown},
	})

	got := runParamModifier(t, plan, state)

	if len(got) != 2 {
		t.Fatalf("expected 2 params, got %d", len(got))
	}
	if v := idOf(t, got, "bbb"); v.ValueInt64() != 200 {
		t.Errorf("bbb param_id = %v, want 200", v)
	}
	if v := idOf(t, got, "ccc"); v.ValueInt64() != 300 {
		t.Errorf("ccc param_id = %v, want 300", v)
	}
}

// TestParamModifier_SteadyState confirms a no-op update still carries every id.
func TestParamModifier_SteadyState(t *testing.T) {
	state := buildParamsList(t, []paramSpec{
		{key: "aaa", id: 100},
		{key: "bbb", id: 200},
	})
	plan := buildParamsList(t, []paramSpec{
		{key: "aaa", id: paramIDUnknown},
		{key: "bbb", id: paramIDUnknown},
	})

	got := runParamModifier(t, plan, state)

	if v := idOf(t, got, "aaa"); v.ValueInt64() != 100 {
		t.Errorf("aaa param_id = %v, want 100", v)
	}
	if v := idOf(t, got, "bbb"); v.ValueInt64() != 200 {
		t.Errorf("bbb param_id = %v, want 200", v)
	}
}

// TestParamModifier_PreservesConfigOrder confirms the modifier never reorders the
// planned list: param_key_name is Required, so the plan must keep config order.
func TestParamModifier_PreservesConfigOrder(t *testing.T) {
	state := buildParamsList(t, []paramSpec{
		{key: "aaa", id: 100},
		{key: "bbb", id: 200},
	})
	// Intentionally unsorted config order (bbb before aaa).
	plan := buildParamsList(t, []paramSpec{
		{key: "bbb", id: paramIDUnknown},
		{key: "aaa", id: paramIDUnknown},
	})

	got := runParamModifier(t, plan, state)

	if got[0].ParamKeyName.ValueString() != "bbb" || got[1].ParamKeyName.ValueString() != "aaa" {
		t.Fatalf("modifier reordered the list: got %q, %q",
			got[0].ParamKeyName.ValueString(), got[1].ParamKeyName.ValueString())
	}
	if got[0].ParamID.ValueInt64() != 200 || got[1].ParamID.ValueInt64() != 100 {
		t.Errorf("ids not bound by key after preserving order: %v, %v", got[0].ParamID, got[1].ParamID)
	}
}

// TestParamModifier_CreateLeavesUnknown confirms that with no prior state (create)
// the param_ids are left unknown for apply to populate.
func TestParamModifier_CreateLeavesUnknown(t *testing.T) {
	plan := buildParamsList(t, []paramSpec{
		{key: "aaa", id: paramIDUnknown},
	})
	got := runParamModifier(t, plan, types.ListNull(types.ObjectType{AttrTypes: ParameterAttrTypes()}))

	if v := idOf(t, got, "aaa"); !v.IsUnknown() {
		t.Errorf("aaa param_id = %v, want unknown on create", v)
	}
}
