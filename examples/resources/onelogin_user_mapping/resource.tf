data "onelogin_role" "engineering" {
  name = "Engineering"
}

resource "onelogin_user_mapping" "example" {
  name    = "Auto-assign Engineering Role"
  match   = "any"
  enabled = true

  # operator "~" means "contains" in the OneLogin API
  conditions {
    source   = "member_of"
    operator = "~"
    value    = "eng_all_staff"
  }

  actions {
    action = "set_role"
    value  = [tostring(data.onelogin_role.engineering.id)]
  }
}
