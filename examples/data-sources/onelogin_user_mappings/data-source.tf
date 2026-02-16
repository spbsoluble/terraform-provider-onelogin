data "onelogin_user_mappings" "enabled" {
  enabled = true
}

output "enabled_mappings" {
  value = data.onelogin_user_mappings.enabled.mappings
}
