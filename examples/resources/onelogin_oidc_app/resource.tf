# OIDC app with typed configuration and multiple redirect URIs
resource "onelogin_oidc_app" "example" {
  name         = "My OIDC App"
  connector_id = 108419

  configuration {
    redirect_uris = [
      "https://app.example.com/callback",
      "https://staging.example.com/callback",
      "http://localhost:3000/callback",
    ]
    login_url                       = "https://app.example.com/login"
    oidc_application_type           = "Web"
    token_endpoint_auth_method      = "BASIC"
    access_token_expiration_minutes = 60
  }

  provisioning = {
    enabled = true
  }

  # Custom parameters
  parameters = [{
    param_key_name          = "email"
    label                   = "Email"
    user_attribute_mappings = "email"
  }]
}

# SSO credentials are computed by the API and stored in state
output "client_id" {
  value     = onelogin_oidc_app.example.sso.client_id
  sensitive = true
}

output "client_secret" {
  value     = onelogin_oidc_app.example.sso.client_secret
  sensitive = true
}
