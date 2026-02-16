data "onelogin_role" "example" {
  id = 12345
}

output "role_name" {
  value = data.onelogin_role.example.name
}
