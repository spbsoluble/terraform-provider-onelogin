resource "onelogin_role" "example" {
  name = "Engineering"

  apps   = [12345, 67890]
  users  = [111, 222, 333]
  admins = [111]
}
