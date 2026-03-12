# Terraform Provider for OneLogin

A Terraform provider for managing [OneLogin](https://www.onelogin.com/) resources, built on the
[terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework) and the
[OneLogin Go SDK v4](https://github.com/onelogin/onelogin-go-sdk).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider)

## Provider Configuration

```hcl
terraform {
  required_providers {
    onelogin = {
      source = "spbsoluble/onelogin"
    }
  }
}

provider "onelogin" {}
```

Authentication is configured via environment variables:

| Variable | Description |
|---|---|
| `ONELOGIN_CLIENT_ID` | OAuth2 client ID |
| `ONELOGIN_CLIENT_SECRET` | OAuth2 client secret |
| `ONELOGIN_API_URL` | API base URL (e.g. `https://api.us.onelogin.com`) |

These can also be set directly in the provider block:

```hcl
provider "onelogin" {
  api_url       = "https://api.us.onelogin.com"
  client_id     = var.onelogin_client_id
  client_secret = var.onelogin_client_secret
  timeout       = 180 # API timeout in seconds (default: 180)
}
```

## Resources

### `onelogin_role`

Manages a OneLogin role, including app access and user/admin membership.

```hcl
resource "onelogin_role" "engineering" {
  name = "Engineering"

  apps   = [onelogin_oidc_app.my_app.id]
  users  = [111, 222, 333]
  admins = [111]
}
```

| Attribute | Type | Description |
|---|---|---|
| `name` | string, required | Role name |
| `apps` | set(number), optional | App IDs accessible by this role |
| `users` | set(number), optional | User IDs assigned to this role |
| `admins` | set(number), optional | User IDs who administer this role |

### `onelogin_user_mapping`

Manages a user mapping rule that automatically assigns roles based on conditions.

```hcl
resource "onelogin_user_mapping" "engineering" {
  name    = "Auto-assign Engineering Role"
  match   = "all"
  enabled = true

  conditions {
    source   = "email"
    operator = "contains"
    value    = "@engineering.example.com"
  }

  actions {
    action = "add_role"
    value  = [tostring(onelogin_role.engineering.id)]
  }
}
```

| Attribute | Type | Description |
|---|---|---|
| `name` | string, required | Mapping name |
| `match` | string, required | Condition logic: `"all"` or `"any"` |
| `enabled` | bool, optional | Whether the mapping is active |
| `position` | number, optional | Evaluation order |
| `conditions` | block, optional | Condition blocks (see below) |
| `actions` | block, optional | Action blocks (see below) |

**`conditions` block:**
- `source` (string) - Field to evaluate (e.g. `"email"`, `"member_of"`)
- `operator` (string) - Comparison operator (e.g. `"="`, `"contains"`)
- `value` (string) - Value to compare against

**`actions` block:**
- `action` (string) - Action type (e.g. `"add_role"`, `"set_status"`)
- `value` (list of strings) - Values for the action

### `onelogin_oidc_app`

Manages an OIDC application with typed configuration for redirect URIs, token settings, and auth methods.

```hcl
resource "onelogin_oidc_app" "my_app" {
  name         = "My OIDC App"
  connector_id = 108419

  configuration {
    redirect_uris = [
      "https://app.example.com/callback",
      "http://localhost:3000/callback",
    ]
    login_url                       = "https://app.example.com/login"
    oidc_application_type           = "Web"
    token_endpoint_auth_method      = "POST"
    access_token_expiration_minutes = 60
  }
}

# SSO credentials are computed by the API
output "client_id" {
  value     = onelogin_oidc_app.my_app.sso.client_id
  sensitive = true
}
```

**`configuration` block:**

| Attribute | Type | Description |
|---|---|---|
| `redirect_uris` | set(string) | OAuth2 redirect URIs |
| `login_url` | string | Login URL |
| `oidc_application_type` | string | `"Web"` (default) or `"Native"` |
| `token_endpoint_auth_method` | string | `"BASIC"`, `"POST"`, or `"PKCE"` |
| `access_token_expiration_minutes` | number | Access token TTL in minutes |
| `refresh_token_expiration_minutes` | number | Refresh token TTL in minutes |

**`sso` attribute (computed, read-only):**

| Attribute | Type | Description |
|---|---|---|
| `client_id` | string, sensitive | OIDC client ID assigned by OneLogin |
| `client_secret` | string, sensitive | OIDC client secret assigned by OneLogin |

### `onelogin_saml_app`

Manages a SAML application with configuration for ACS, audience, signature algorithm, and more.

```hcl
resource "onelogin_saml_app" "my_app" {
  name         = "My SAML App"
  connector_id = 110016

  configuration {
    signature_algorithm = "SHA-256"
    acs                 = "https://app.example.com/saml/acs"
    audience            = "https://app.example.com"
  }
}

output "metadata_url" {
  value = onelogin_saml_app.my_app.sso.metadata_url
}
```

**`configuration` block:**

| Attribute | Type | Description |
|---|---|---|
| `signature_algorithm` | string | `"SHA-1"`, `"SHA-256"`, `"SHA-384"`, or `"SHA-512"` |
| `certificate_id` | number | Certificate ID for SAML signing |
| `provider_arn` | string | AWS provider ARN (for AWS SAML apps) |
| `acs` | string | Assertion Consumer Service URL |
| `audience` | string | SP Entity ID / audience restriction |
| `recipient` | string | Recipient URL |
| `relaystate` | string | Relay state URL |
| `subdomain` | string | Subdomain for catalog SAML connectors |

**`sso` attribute (computed, read-only):**

| Attribute | Type | Description |
|---|---|---|
| `metadata_url` | string | SAML metadata URL |
| `acs_url` | string | Assertion Consumer Service URL |
| `sls_url` | string | Single Logout Service URL |
| `issuer` | string | SAML issuer URL |
| `certificate` | object | Signing certificate (`id`, `name`, `value`) |

### `onelogin_app`

A generic app resource for connector types not covered by `onelogin_oidc_app` or `onelogin_saml_app`.
Configuration and SSO are raw JSON strings.

```hcl
resource "onelogin_app" "custom" {
  name         = "My Custom App"
  connector_id = 108419

  configuration = jsonencode({
    redirect_uri = "https://example.com/callback"
    login_url    = "https://example.com/login"
  })
}
```

### Shared App Attributes

All three app resources (`onelogin_app`, `onelogin_oidc_app`, `onelogin_saml_app`) share these attributes:

| Attribute | Type | Description |
|---|---|---|
| `name` | string, required | App name |
| `connector_id` | number, required | Connector type ID (forces replacement if changed) |
| `description` | string, optional | App description |
| `notes` | string, optional | App notes |
| `visible` | bool, optional | Whether visible to users (default: `true`) |
| `allow_assumed_signin` | bool, optional | Allow assumed sign-in |
| `policy_id` | number, optional | Security policy ID |
| `role_ids` | set(number), optional | Role IDs that can access this app |
| `icon_url` | string, computed | App icon URL |
| `auth_method` | number, computed | Authentication method code |
| `created_at` | string, computed | Creation timestamp |

**`provisioning` attribute (optional):**

```hcl
provisioning = {
  enabled = true
}
```

**`parameters` attribute (optional, list):**

Parameters are a **list** — elements are matched positionally between config and state. Sort parameters by `param_key_name` to ensure consistent ordering. When using `generate-tf-applications` with API credentials, all parameters (including API-injected ones) are automatically sorted and included.

```hcl
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
    skip_if_blank             = true
  },
]
```

## Data Sources

| Data Source | Lookup | Description |
|---|---|---|
| `onelogin_role` | by `id` | Read a single role |
| `onelogin_app` | by `id` | Read a single app |
| `onelogin_apps` | `name_filter`, `connector_id` | List/filter apps |
| `onelogin_user_mapping` | by `id` | Read a single user mapping |
| `onelogin_user_mappings` | `enabled` | List user mappings |

### Example: Look up apps by name

```hcl
data "onelogin_apps" "grafana" {
  name_filter = "Grafana"
}

output "grafana_app_ids" {
  value = [for app in data.onelogin_apps.grafana.apps : app.id]
}
```

## Importing Existing Resources

All resources support import by numeric ID:

```bash
terraform import onelogin_role.example 12345
terraform import onelogin_oidc_app.example 67890
terraform import onelogin_saml_app.example 11111
terraform import onelogin_user_mapping.example 22222
terraform import onelogin_app.example 33333
```

## Local Development

### Building the Provider

```bash
git clone https://github.com/spbsoluble/terraform-provider-onelogin.git
cd terraform-provider-onelogin
make build
```

### Running Locally Without the Terraform Registry

Use Terraform's [dev_overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
to point Terraform at your locally built binary. Create or edit `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "spbsoluble/onelogin" = "/path/to/terraform-provider-onelogin"
  }
  direct {}
}
```

Replace `/path/to/terraform-provider-onelogin` with the absolute path to the directory
containing the built `terraform-provider-onelogin` binary.

With `dev_overrides` active:
- **Do not** run `terraform init` (it will warn about the override, which is expected).
- Terraform will use your local binary directly.

Then configure your credentials and run:

```bash
export ONELOGIN_CLIENT_ID="your-client-id"
export ONELOGIN_CLIENT_SECRET="your-client-secret"
export ONELOGIN_API_URL="https://api.us.onelogin.com"

terraform plan
terraform apply
```

### Running Tests

```bash
# Unit tests
make test

# Acceptance tests (requires OneLogin API credentials)
export ONELOGIN_CLIENT_ID="..."
export ONELOGIN_CLIENT_SECRET="..."
export ONELOGIN_API_URL="..."
make testacc
```

### Makefile Targets

| Target | Description |
|---|---|
| `make build` | Build the provider binary |
| `make install` | Build and install to the local plugin directory |
| `make test` | Run unit tests |
| `make testacc` | Run acceptance tests against a live OneLogin environment |
| `make lint` | Run golangci-lint |
| `make fmt` | Format Go source files |
| `make tfdocs` | Generate provider documentation with tfplugindocs |

