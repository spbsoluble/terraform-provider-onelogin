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

// TestAccSAMLAppResource_parameterSetChange is the end-to-end regression test for
// the "Provider produced inconsistent result after apply" bug seen on
// velo_ops_jenkins_prod_2 / _non_prd_2 (pipeline job 4182990). Inserting a
// parameter that sorts in the middle of the existing set used to shift computed
// param_id values onto the wrong elements (index-based UseStateForUnknown) and
// fail the apply. Each step here runs plan+apply and the test framework also
// asserts an empty plan afterwards, so any param_id drift fails the test.
func TestAccSAMLAppResource_parameterSetChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-saml-param")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with two params (sorted by key, as the generator emits).
			{
				Config: testAccSAMLAppResourceConfig_params2(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "parameters.#", "2"),
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "parameters.0.param_id"),
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "parameters.1.param_id"),
				),
			},
			// Insert a param that sorts between the two existing ones.
			{
				Config: testAccSAMLAppResourceConfig_params3(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "parameters.#", "3"),
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "parameters.0.param_id"),
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "parameters.1.param_id"),
					resource.TestCheckResourceAttrSet("onelogin_saml_app.test", "parameters.2.param_id"),
				),
			},
			// Remove the middle param again — exercises the shrink path too.
			{
				Config: testAccSAMLAppResourceConfig_params2(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_saml_app.test", "parameters.#", "2"),
				),
			},
		},
	})
}

// --- Config helpers ---

func testAccSAMLAppResourceConfig_params2(name string) string {
	return fmt.Sprintf(`
resource "onelogin_saml_app" "test" {
  name         = %[1]q
  connector_id = 110016

  configuration {
    signature_algorithm = "SHA-256"
  }

  parameters = [
    {
      param_key_name            = "aaa_dept"
      include_in_saml_assertion = true
    },
    {
      param_key_name            = "zzz_team"
      include_in_saml_assertion = true
    },
  ]
}
`, name)
}

func testAccSAMLAppResourceConfig_params3(name string) string {
	return fmt.Sprintf(`
resource "onelogin_saml_app" "test" {
  name         = %[1]q
  connector_id = 110016

  configuration {
    signature_algorithm = "SHA-256"
  }

  parameters = [
    {
      param_key_name            = "aaa_dept"
      include_in_saml_assertion = true
    },
    {
      param_key_name            = "mmm_role"
      include_in_saml_assertion = true
    },
    {
      param_key_name            = "zzz_team"
      include_in_saml_assertion = true
    },
  ]
}
`, name)
}

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
