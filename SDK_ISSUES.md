# OneLogin Go SDK v4 - Recommended Fixes

Issues discovered while building the Terraform Provider using `onelogin-go-sdk/v4` v4.7.0.

## 1. `CreateUserMapping` Does Not Handle Partial Responses

**File:** `pkg/onelogin/user_mappings.go` - `CreateUserMapping()`

**Problem:** The `POST /api/2/mappings` endpoint returns only `{"id": 12345}`, not the full mapping object. `CreateUserMapping` unmarshals this into a `UserMapping` struct where all fields except `ID` are nil. This causes Terraform (and likely other consumers) to see null values for name, match, enabled, conditions, and actions.

**Comparison:** `UpdateUserMapping` already handles this case correctly (lines 78-103) by checking if the response is just an `{id: XYZ}` object and re-fetching the full object via `GetUserMapping`.

**Fix:** Add the same ID-only response handling from `UpdateUserMapping` to `CreateUserMapping`:

```go
func (sdk *OneloginSDK) CreateUserMapping(mapping mod.UserMapping) (*mod.UserMapping, error) {
    p, err := utl.BuildAPIPath(UserMappingsPath)
    if err != nil {
        return nil, err
    }
    resp, err := sdk.Client.Post(&p, mapping)
    if err != nil {
        return nil, err
    }

    // Handle partial response (API may return just {id: XYZ})
    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
        var responseObj struct {
            ID int32 `json:"id"`
        }
        body, err := io.ReadAll(resp.Body)
        resp.Body.Close()
        if err != nil {
            return nil, err
        }
        err = json.Unmarshal(body, &responseObj)
        if err == nil && responseObj.ID > 0 {
            return sdk.GetUserMapping(responseObj.ID)
        }
        resp = &http.Response{
            StatusCode: resp.StatusCode,
            Body:       io.NopCloser(bytes.NewBuffer(body)),
        }
    }

    var newMapping mod.UserMapping
    err = utl.CheckHTTPResponseAndUnmarshal(resp, &newMapping)
    return &newMapping, err
}
```

---

## 2. `UserMapping.Position` Uses `omitempty` But API Requires It on PUT

**File:** `pkg/onelogin/models/user_mapping.go`

**Problem:** The `Position` field has `json:"position,omitempty"`. When `Position` is nil (common after creation), `json.Marshal` omits the field entirely. However, the `PUT /api/2/mappings/{id}` endpoint **requires** the `position` field to be present (returns 422 "Required field is missing" if omitted). The field must be sent as `null` when the value is unknown or when the mapping is disabled.

**Fix:** Remove `omitempty` from the Position tag:

```go
type UserMapping struct {
    ID         *int32                  `json:"id,omitempty"`
    Name       *string                 `json:"name,omitempty"`
    Match      *string                 `json:"match,omitempty"`
    Enabled    *bool                   `json:"enabled,omitempty"`
    Position   *int32                  `json:"position"`      // Removed omitempty
    Conditions []UserMappingConditions `json:"conditions"`
    Actions    []UserMappingActions    `json:"actions"`
}
```

---

## 3. `UserMapping.ID` Uses `omitempty` But API Rejects It on PUT

**File:** `pkg/onelogin/models/user_mapping.go`

**Problem:** The `ID` field has `json:"id,omitempty"`. When callers populate the ID (common when building an update payload from existing state), `json.Marshal` includes it. However, the `PUT /api/2/mappings/{id}` endpoint **rejects** the `id` field in the request body (returns 422 "Field is not allowed"). The ID should only be in the URL path, not the body.

**Current workaround:** Consumers must avoid setting the `ID` field when building update payloads, or use a custom struct without `ID`.

**Recommended fix:** `UpdateUserMapping` should strip the ID from the payload before sending:

```go
func (sdk *OneloginSDK) UpdateUserMapping(mappingID int32, mapping mod.UserMapping) (*mod.UserMapping, error) {
    mapping.ID = nil // API rejects id in body
    // ... rest of method
}
```

---

## 4. API Position Rules Are Contradictory (Documentation Issue)

**Endpoint:** `PUT /api/2/mappings/{id}`

**Problem:** The API has contradictory validation rules for the `position` field:
- `position` is **required** on every PUT (422 if omitted)
- `position` must be `null` when `enabled=false` (422 "Position cannot be set while the mapping is not enabled" if a numeric value is sent for a disabled mapping)
- `position` can be `null` or a numeric value when `enabled=true`

This means:
- Disabling: must send `"position": null`
- Enabling: must send `"position": null` or `"position": <int>`
- Always: must include the position field

This is not documented anywhere and was discovered through trial and error. The SDK should either document this behavior or handle it internally in `UpdateUserMapping`.

---

## 5. `RoleQuery` Does Not Implement `Queryable` Interface

**Problem:** There is no `RoleQuery` type that implements the `Queryable` interface. The `GetRoles` and `GetRolesWithContext` methods accept `mod.Queryable` but there's no way to pass query parameters (like `limit` or pagination) for roles. This means every call to `GetRoles` fetches ALL roles, which can be extremely slow on accounts with many roles.

**Fix:** Add a `RoleQuery` type similar to `UserMappingsQuery`:

```go
type RoleQuery struct {
    Limit  string `json:"limit,omitempty"`
    Page   string `json:"page,omitempty"`
    Cursor string `json:"cursor,omitempty"`
}

func (q *RoleQuery) GetKeyValidators() map[string]func(interface{}) bool {
    return map[string]func(interface{}) bool{
        "limit":  validateString,
        "page":   validateString,
        "cursor": validateString,
    }
}
```

---

## 6. Role Create API Returns Partial Data (Existing Pattern)

**Problem:** The `POST /api/1/roles` endpoint returns only the created role object with partial data (sometimes just the ID). This is already handled by consumers reading back the full role after creation, but the SDK could handle this internally like `UpdateUserMapping` does.

---

## 7. Role Update API Rejects Empty Arrays for Users/Admins

**Endpoint:** `PUT /api/1/roles/{id}`

**Problem:** Sending `"users": []` or `"admins": []` in the PUT body causes a 400 error. The Role model's custom `MarshalJSON` always includes these fields (even when empty) because they don't use `omitempty`. The API expects user/admin management to be done via the dedicated endpoints (`POST/DELETE /api/1/roles/{id}/users`, `POST/DELETE /api/1/roles/{id}/admins`).

**Workaround:** Consumers must send only the `name` field in the PUT body, and use the dedicated user/admin endpoints for membership changes.

**Fix:** Either:
1. Document that `UpdateRole` should only be used for name updates
2. Have the SDK's `UpdateRole` strip users/admins from the payload
3. Change the Role model's `MarshalJSON` to use `omitempty` behavior for empty slices

---

## 8. `ConfigurationSAML` Struct Lacks `omitempty`, Causes API 400 Errors

**File:** `pkg/onelogin/models/app.go` - `ConfigurationSAML`

**Problem:** The `ConfigurationSAML` struct does not use `omitempty` on its JSON tags. When serialized, all fields are included—even null/zero ones. The API rejects unknown or null configuration fields for certain connector types (e.g., sending `"provider_arn": null` for a non-AWS SAML connector returns HTTP 400 "Unknown parameter on configuration section: provider_arn").

**Workaround:** Use `map[string]interface{}` instead of the typed struct, and only include fields the user explicitly specified.

**Fix:** Add `omitempty` to optional configuration fields:

```go
type ConfigurationSAML struct {
    SignatureAlgorithm string      `json:"signature_algorithm"`
    CertificateID      int         `json:"certificate_id,omitempty"`
    ProviderArn        interface{} `json:"provider_arn,omitempty"`
}
```

---

## 9. `SSOOpenId` Struct Missing `ClientSecret` Field

**File:** `pkg/onelogin/models/app.go` - `SSOOpenId`

**Problem:** The `SSOOpenId` struct only has `ClientID`. However, the API response for OIDC apps includes both `client_id` and `client_secret` in the SSO object. The missing `client_secret` field means consumers cannot capture this value from the SDK's typed struct.

**Workaround:** Use JSON re-marshaling of the raw `App.SSO` interface{} to a custom struct that includes both fields.

**Fix:** Add the `ClientSecret` field:

```go
type SSOOpenId struct {
    ClientID     string `json:"client_id"`
    ClientSecret string `json:"client_secret"`
}
```

---

## 10. App `SSO` and `Configuration` Fields Use `interface{}` Instead of Typed Structs

**File:** `pkg/onelogin/models/app.go` - `App`

**Problem:** The `App` struct declares `SSO` and `Configuration` as `interface{}`. While this provides flexibility, it means consumers must perform JSON re-marshaling (marshal to bytes, then unmarshal to the typed struct) to access typed fields. This is error-prone and loses compile-time type safety.

**Impact:** Every consumer of the SDK must implement their own extraction helpers. The SDK provides typed structs (`SSOOpenId`, `SSOSAML`, `ConfigurationSAML`, `ConfigurationOpenId`) but no way to automatically populate them from the raw `interface{}` values.

**Fix:** Either:
1. Add typed accessor methods on `App` (e.g., `app.GetSSOOpenId()`, `app.GetConfigSAML()`)
2. Or use concrete types with a discriminator (based on `AuthMethod` or `ConnectorID`)

---

## 11. OIDC SSO Credentials Only Returned on Create

**Endpoint:** `POST /api/2/apps` (OIDC connector)

**Problem:** The `client_id` and `client_secret` SSO fields are only returned in the response to the initial app creation. Subsequent `GET /api/2/apps/{id}` calls return the SSO object but with empty `client_secret`. This is a security design choice by the API but is not documented in the SDK.

**Impact:** Consumers must capture and persist `client_id` and `client_secret` from the create response. The Terraform provider handles this by marking these as `Sensitive` and using `UseStateForUnknown()` plan modifiers to preserve the values across reads.

---

## Summary

| # | Severity | Issue | Affects |
|---|----------|-------|---------|
| 1 | High | CreateUserMapping returns partial data | All consumers |
| 2 | High | Position omitempty causes 422 on update | All consumers |
| 3 | Medium | ID in PUT body causes 422 | Update operations |
| 4 | Low | Position validation rules undocumented | API documentation |
| 5 | Medium | No RoleQuery type for pagination | Large accounts |
| 6 | Low | CreateRole returns partial data | All consumers |
| 7 | Medium | Empty arrays in role update cause 400 | Role updates |
| 8 | Medium | ConfigurationSAML lacks omitempty | SAML app create/update |
| 9 | Medium | SSOOpenId missing client_secret field | OIDC apps |
| 10 | Low | App SSO/Configuration use interface{} | All app consumers |
| 11 | Low | OIDC SSO credentials only on create | OIDC app management |
