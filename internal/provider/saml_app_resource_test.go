package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSAMLAppResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-saml")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSAMLAppResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "id"),
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "connector_id", "110016"),
				),
			},
			// ImportState
			{
				ResourceName:      "onelogin_saml_app.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSAMLAppResource_withConfiguration(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-saml-cfg")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSAMLAppResourceConfig_withConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "id"),
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "configuration.signature_algorithm", "SHA-256"),
				),
			},
		},
	})
}

func TestAccSAMLAppResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-saml-upd")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSAMLAppResourceConfig_withConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "configuration.signature_algorithm", "SHA-256"),
				),
			},
			// Update to SHA-512
			{
				Config: testAccSAMLAppResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "configuration.signature_algorithm", "SHA-512"),
				),
			},
		},
	})
}

func TestAccSAMLAppResource_fullLifecycle(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-saml-lc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccSAMLAppResourceConfig_withConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "id"),
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "configuration.signature_algorithm", "SHA-256"),
				),
			},
			// Update
			{
				Config: testAccSAMLAppResourceConfig_updated(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "configuration.signature_algorithm", "SHA-512"),
				),
			},
			// Import
			{
				ResourceName:      "onelogin_saml_app.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// --- Config helpers ---

func testAccSAMLAppResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "onelogin_saml_app" "test" {
  name         = %[1]q
  connector_id = 110016

  configuration {
    signature_algorithm = "SHA-256"
  }
}
`, name)
}

func testAccSAMLAppResourceConfig_withConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_saml_app" "test" {
  name         = %[1]q
  connector_id = 110016

  configuration {
    signature_algorithm = "SHA-256"
  }
}
`, name)
}

func testAccSAMLAppResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "onelogin_saml_app" "test" {
  name         = %[1]q
  connector_id = 110016

  configuration {
    signature_algorithm = "SHA-512"
  }
}
`, name)
}
