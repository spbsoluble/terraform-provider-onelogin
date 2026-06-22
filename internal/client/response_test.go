package client

import "testing"

// TestExtractAppConfigOIDC_PostLogoutRedirectURI is the regression test for the
// logout_url gap: OneLogin OIDC apps expose the logout target as
// post_logout_redirect_uri, which the provider previously did not parse.
func TestExtractAppConfigOIDC_PostLogoutRedirectURI(t *testing.T) {
	config := map[string]interface{}{
		"redirect_uri":             "https://app.example.com/callback",
		"post_logout_redirect_uri": "https://app.example.com/logout",
		"login_url":                "https://app.example.com/login",
	}
	cfg, err := ExtractAppConfigOIDC(config)
	if err != nil {
		t.Fatalf("ExtractAppConfigOIDC: %v", err)
	}
	if cfg.PostLogoutRedirectURI != "https://app.example.com/logout" {
		t.Errorf("PostLogoutRedirectURI = %q, want https://app.example.com/logout", cfg.PostLogoutRedirectURI)
	}
}

// TestExtractAppConfigSAML_LogoutURL guards SAML logout_url parsing.
func TestExtractAppConfigSAML_LogoutURL(t *testing.T) {
	config := map[string]interface{}{
		"consumer_url": "https://app.example.com/acs",
		"audience":     "https://app.example.com",
		"logout_url":   "https://app.example.com/slo",
		"relaystate":   "https://app.example.com/relay",
	}
	cfg, err := ExtractAppConfigSAML(config)
	if err != nil {
		t.Fatalf("ExtractAppConfigSAML: %v", err)
	}
	if cfg.LogoutURL != "https://app.example.com/slo" {
		t.Errorf("LogoutURL = %q, want https://app.example.com/slo", cfg.LogoutURL)
	}
	if cfg.RelayState != "https://app.example.com/relay" {
		t.Errorf("RelayState = %q, want https://app.example.com/relay", cfg.RelayState)
	}
}
