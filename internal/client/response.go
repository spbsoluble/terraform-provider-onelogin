package client

import (
	"encoding/json"
	"fmt"

	models "github.com/onelogin/onelogin-go-sdk/v4/pkg/onelogin/models"
)

// UnmarshalRole converts an SDK interface{} response to a *models.Role.
// The "users" field is stripped before marshaling — roles can have tens of thousands
// of members (e.g. 38K+ for POC business tiers) which causes minutes-long JSON processing.
// Role membership is managed via OneLogin mappings, not Terraform.
func UnmarshalRole(data interface{}) (*models.Role, error) {
	if data == nil {
		return nil, nil
	}
	if m, ok := data.(map[string]interface{}); ok {
		delete(m, "users")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role response: %w", err)
	}
	var role models.Role
	if err := json.Unmarshal(b, &role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role: %w", err)
	}
	return &role, nil
}

// UnmarshalRoles converts an SDK interface{} response to a slice of models.Role.
func UnmarshalRoles(data interface{}) ([]models.Role, error) {
	if data == nil {
		return nil, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal roles response: %w", err)
	}
	var roles []models.Role
	if err := json.Unmarshal(b, &roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
	}
	return roles, nil
}

// UnmarshalApp converts an SDK interface{} response to a *models.App.
func UnmarshalApp(data interface{}) (*models.App, error) {
	if data == nil {
		return nil, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal app response: %w", err)
	}
	var app models.App
	if err := json.Unmarshal(b, &app); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app: %w", err)
	}
	return &app, nil
}

// UnmarshalApps converts an SDK interface{} response to a slice of models.App.
func UnmarshalApps(data interface{}) ([]models.App, error) {
	if data == nil {
		return nil, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal apps response: %w", err)
	}
	var apps []models.App
	if err := json.Unmarshal(b, &apps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal apps: %w", err)
	}
	return apps, nil
}

// UnmarshalUser converts an SDK interface{} response to a *models.User.
func UnmarshalUser(data interface{}) (*models.User, error) {
	if data == nil {
		return nil, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user response: %w", err)
	}
	var user models.User
	if err := json.Unmarshal(b, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}
	return &user, nil
}

// UnmarshalUsers converts an SDK interface{} response to a slice of models.User.
func UnmarshalUsers(data interface{}) ([]models.User, error) {
	if data == nil {
		return nil, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal users response: %w", err)
	}
	var users []models.User
	if err := json.Unmarshal(b, &users); err != nil {
		return nil, fmt.Errorf("failed to unmarshal users: %w", err)
	}
	return users, nil
}

// OIDCConfig represents the OIDC configuration with polymorphic redirect_uri handling.
// The API can return redirect_uri as a string or as a JSON array of strings.
type OIDCConfig struct {
	RedirectURIs                  []string        `json:"-"`
	RedirectURIRaw                json.RawMessage `json:"redirect_uri"`
	LoginURL                      string          `json:"login_url"`
	OidcApplicationType           int             `json:"oidc_application_type"`
	TokenEndpointAuthMethod       int             `json:"token_endpoint_auth_method"`
	AccessTokenExpirationMinutes  int             `json:"access_token_expiration_minutes"`
	RefreshTokenExpirationMinutes int             `json:"refresh_token_expiration_minutes"`
}

// ExtractAppConfigOIDC extracts OIDC configuration from the App's interface{} Configuration field.
// Handles the polymorphic redirect_uri field (can be string or []string from the API).
func ExtractAppConfigOIDC(config interface{}) (*OIDCConfig, error) {
	if config == nil {
		return nil, nil
	}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OIDC config: %w", err)
	}
	var cfg OIDCConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OIDC config: %w", err)
	}

	// Parse the polymorphic redirect_uri field
	if cfg.RedirectURIRaw != nil && string(cfg.RedirectURIRaw) != "null" {
		// Try as array first
		var uris []string
		if err := json.Unmarshal(cfg.RedirectURIRaw, &uris); err == nil {
			cfg.RedirectURIs = uris
		} else {
			// Fall back to string (may be newline or comma separated)
			var s string
			if err := json.Unmarshal(cfg.RedirectURIRaw, &s); err == nil && s != "" {
				cfg.RedirectURIs = []string{s}
			}
		}
	}

	return &cfg, nil
}

// SAMLConfig represents the full SAML configuration including fields beyond the SDK's ConfigurationSAML.
// The SDK struct only has provider_arn, signature_algorithm, and certificate_id, but the API
// supports additional fields for Advanced SAML (110016) and catalog SAML connectors.
type SAMLConfig struct {
	ProviderArn        interface{} `json:"provider_arn"`
	SignatureAlgorithm string      `json:"signature_algorithm"`
	CertificateID      int         `json:"certificate_id"`
	ACS                string      `json:"consumer_url"`
	Consumer           string      `json:"consumer"` // Some connectors (e.g., AWS SSO 130413) use "consumer" instead of "consumer_url"
	Audience           string      `json:"audience"`
	Recipient          string      `json:"recipient"`
	RelayState         string      `json:"relaystate"`
	Subdomain          string      `json:"subdomain"`
}

// ExtractAppConfigSAML extracts SAML configuration from the App's interface{} Configuration field.
func ExtractAppConfigSAML(config interface{}) (*SAMLConfig, error) {
	if config == nil {
		return nil, nil
	}
	b, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SAML config: %w", err)
	}
	var cfg SAMLConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SAML config: %w", err)
	}
	// Some connectors return "consumer" instead of "consumer_url" — normalize to ACS.
	if cfg.ACS == "" && cfg.Consumer != "" {
		cfg.ACS = cfg.Consumer
	}
	return &cfg, nil
}

// ExtractSSOOpenId extracts OpenID SSO data from the App's interface{} SSO field.
func ExtractSSOOpenId(sso interface{}) (*models.SSOOpenId, error) {
	if sso == nil {
		return nil, nil
	}
	b, err := json.Marshal(sso)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenID SSO: %w", err)
	}
	var s models.SSOOpenId
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenID SSO: %w", err)
	}
	return &s, nil
}

// ExtractSSOSAML extracts SAML SSO data from the App's interface{} SSO field.
func ExtractSSOSAML(sso interface{}) (*models.SSOSAML, error) {
	if sso == nil {
		return nil, nil
	}
	b, err := json.Marshal(sso)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SAML SSO: %w", err)
	}
	var s models.SSOSAML
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SAML SSO: %w", err)
	}
	return &s, nil
}
