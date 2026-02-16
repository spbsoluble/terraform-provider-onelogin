# Generic app resource — use when your connector type doesn't have a
# dedicated resource (onelogin_oidc_app, onelogin_saml_app).
# Configuration and SSO are stored as raw JSON strings.
resource "onelogin_app" "example" {
  name         = "My Custom App"
  connector_id = 108419

  # Optional: pass configuration as a JSON-encoded string
  configuration = jsonencode({
    redirect_uri = "https://example.com/callback"
    login_url    = "https://example.com/login"
  })
}

output "app_id" {
  value = onelogin_app.example.id
}
