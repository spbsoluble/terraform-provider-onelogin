data "onelogin_user_mapping" "example" {
  id = 12345
}

output "mapping_name" {
  value = data.onelogin_user_mapping.example.name
}
