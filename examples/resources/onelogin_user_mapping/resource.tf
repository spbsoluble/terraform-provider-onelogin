resource "onelogin_user_mapping" "example" {
  name    = "Auto-assign Engineering Role"
  match   = "all"
  enabled = true

  conditions {
    source   = "email"
    operator = "contains"
    value    = "@engineering.example.com"
  }

  actions {
    action = "add_role"
    value  = [tostring(onelogin_role.engineering.id)]
  }
}
