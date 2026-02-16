# SAML app with typed configuration
resource "onelogin_saml_app" "example" {
  name         = "My SAML App"
  connector_id = 110016

  configuration {
    signature_algorithm = "SHA-256"
    acs                 = "https://app.example.com/saml/acs"
    audience            = "https://app.example.com"
  }

  provisioning = {
    enabled = true
  }

  # Map user attributes into the SAML assertion
  parameters = [
    {
      param_key_name            = "email"
      label                     = "Email"
      user_attribute_mappings   = "email"
      include_in_saml_assertion = true
    },
    {
      param_key_name            = "firstname"
      label                     = "First Name"
      user_attribute_mappings   = "firstname"
      include_in_saml_assertion = true
    },
  ]
}

# SSO metadata is populated by the API
output "metadata_url" {
  value = onelogin_saml_app.example.sso.metadata_url
}

output "acs_url" {
  value = onelogin_saml_app.example.sso.acs_url
}
