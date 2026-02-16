# Look up an app by ID
data "onelogin_app" "example" {
  id = 12345
}

output "app_name" {
  value = data.onelogin_app.example.name
}

output "connector_id" {
  value = data.onelogin_app.example.connector_id
}
