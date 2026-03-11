---
page_title: "onelogin_role Resource - terraform-provider-onelogin"
subcategory: ""
description: |-
  Manages a OneLogin Role.
---

# onelogin_role (Resource)

Manages a OneLogin Role.

> **Note on role membership:** The `users` attribute is intentionally not supported.
> Roles in this provider manage application access and administrators only.
> Role membership (which users belong to a role) must be managed through
> [OneLogin Mappings](https://developers.onelogin.com/api-docs/2/user-mappings/overview)
> using the `onelogin_user_mapping` resource. This avoids Terraform tracking tens of
> thousands of user IDs in state for high-membership roles such as POC or trial tiers.

## Example Usage

```terraform
resource "onelogin_role" "example" {
  name = "Engineering"

  apps   = ["My SAML App", "Dev Portal"]
  admins = ["alice@example.com", "bob@example.com"]
}
```

## Schema

### Required

- `name` (String) The name of the role.

### Optional

- `admins` (Set of String) A set of email addresses for users who administer this role.
- `apps` (Set of String) A set of app display names accessible by this role.

### Read-Only

- `id` (Number) The unique identifier of the role.
