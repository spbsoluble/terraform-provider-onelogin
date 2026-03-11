# Lookup by ID
data "onelogin_role" "by_id" {
  id = 12345
}

output "role_name" {
  value = data.onelogin_role.by_id.name
}

# Lookup by name — uses the ?name= API filter; efficient for large accounts
data "onelogin_role" "by_name" {
  name = "Engineering"
}

output "role_id" {
  value = data.onelogin_role.by_name.id
}
