package provider_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOIDCAppResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-oidc")

	resource.Test(
		t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccOIDCAppResourceConfig_basic(rName),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "id"),
						resource.TestCheckResourceAttr("onelogin_oidc_app.test", "name", rName),
						resource.TestCheckResourceAttr("onelogin_oidc_app.test", "connector_id", "108419"),
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "sso.client_id"),
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "sso.client_secret"),
					),
				},
				// ImportState - SSO credentials are not returned on import (only on create)
				{
					ResourceName:            "onelogin_oidc_app.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"sso"},
				},
			},
		},
	)
}

func TestAccOIDCAppResource_withConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-oidc-cfg")

	resource.Test(
		t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccOIDCAppResourceConfig_withConfig(rName),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "id"),
						resource.TestCheckResourceAttr("onelogin_oidc_app.test", "name", rName),
						resource.TestCheckResourceAttr(
							"onelogin_oidc_app.test",
							"configuration.redirect_uris.#",
							"1",
						),
						resource.TestCheckResourceAttr(
							"onelogin_oidc_app.test",
							"configuration.login_url",
							"https://example.com/login",
						),
					),
				},
			},
		},
	)
}

func TestAccOIDCAppResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-oidc-upd")

	resource.Test(
		t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccOIDCAppResourceConfig_withConfig(rName),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							"onelogin_oidc_app.test",
							"configuration.redirect_uris.#",
							"1",
						),
					),
				},
				// Update redirect_uris
				{
					Config: testAccOIDCAppResourceConfig_updated(rName),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							"onelogin_oidc_app.test",
							"configuration.redirect_uris.#",
							"2",
						),
					),
				},
			},
		},
	)
}

func TestAccOIDCAppResource_withParameters(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-oidc-par")

	resource.Test(
		t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccOIDCAppResourceConfig_withParams(rName),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "id"),
						resource.TestCheckResourceAttr("onelogin_oidc_app.test", "name", rName),
					),
				},
			},
		},
	)
}

func TestAccOIDCAppResource_fullLifecycle(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-oidc-lc")

	resource.Test(
		t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				// Create with config
				{
					Config: testAccOIDCAppResourceConfig_withConfig(rName),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "id"),
						resource.TestCheckResourceAttr("onelogin_oidc_app.test", "name", rName),
						resource.TestCheckResourceAttr(
							"onelogin_oidc_app.test",
							"configuration.redirect_uris.#",
							"1",
						),
						// SSO credentials are populated on create
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "sso.client_id"),
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "sso.client_secret"),
					),
				},
				// Update — SSO credentials should be preserved
				{
					Config: testAccOIDCAppResourceConfig_updated(rName + "-updated"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("onelogin_oidc_app.test", "name", rName+"-updated"),
						resource.TestCheckResourceAttr(
							"onelogin_oidc_app.test",
							"configuration.redirect_uris.#",
							"2",
						),
						// SSO credentials should still be present after update
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "sso.client_id"),
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "sso.client_secret"),
					),
				},
				// Import — SSO credentials won't match (only returned on create)
				{
					ResourceName:            "onelogin_oidc_app.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"sso"},
				},
			},
		},
	)
}

func TestAccOIDCAppResource_manyRedirectURIs(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-oidc-many")
	uriCount := 55

	resource.Test(
		t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccOIDCAppResourceConfig_manyURIs(rName, uriCount),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("onelogin_oidc_app.test", "id"),
						resource.TestCheckResourceAttr("onelogin_oidc_app.test", "name", rName),
						resource.TestCheckResourceAttr(
							"onelogin_oidc_app.test",
							"configuration.redirect_uris.#",
							fmt.Sprintf("%d", uriCount),
						),
					),
				},
				// ImportState — verify all URIs come back
				{
					ResourceName:            "onelogin_oidc_app.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"sso"},
				},
			},
		},
	)
}

// --- Config helpers ---

func testAccOIDCAppResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419
}
`, name)
}

func testAccOIDCAppResourceConfig_withConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419

  configuration {
    redirect_uris = ["https://example.com/callback"]
    login_url     = "https://example.com/login"
  }
}
`, name)
}

func testAccOIDCAppResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419

  configuration {
    redirect_uris = [
      "https://example.com/callback/v2",
      "https://example.com/callback/v3",
    ]
    login_url = "https://example.com/login"
  }
}
`, name)
}

func testAccOIDCAppResourceConfig_withParams(name string) string {
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419

  configuration {
    redirect_uris = ["https://example.com/callback"]
  }

  parameters = [{
    param_key_name          = "email"
    label                   = "Email"
    user_attribute_mappings = "email"
  }]
}
`, name)
}

func testAccOIDCAppResourceConfig_manyURIs(name string, count int) string {
	uris := make([]string, count)
	for i := 0; i < count; i++ {
		uris[i] = fmt.Sprintf("    \"https://app-%03d.example.com/callback\"", i+1)
	}
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419

  configuration {
    redirect_uris = [
%s
    ]
  }
}
`, name, strings.Join(uris, ",\n"))
}
