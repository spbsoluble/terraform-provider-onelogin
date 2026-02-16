# List all apps
data "onelogin_apps" "all" {}

# Filter apps by connector ID (e.g., OIDC apps only)
data "onelogin_apps" "oidc_apps" {
  connector_id = 108419
}

output "oidc_app_count" {
  value = length(data.onelogin_apps.oidc_apps.apps)
}
