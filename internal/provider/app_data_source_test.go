package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-app-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.onelogin_app.test", "id",
						"onelogin_oidc_app.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.onelogin_app.test", "name",
						"onelogin_oidc_app.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.onelogin_app.test", "connector_id",
						"onelogin_oidc_app.test", "connector_id",
					),
				),
			},
		},
	})
}

func TestAccAppsDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-apps-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppsDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.onelogin_apps.test", "apps.#"),
				),
			},
		},
	})
}

func TestAccAppsDataSource_filterByConnectorID(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-apps-ds-conn")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppsDataSourceConfig_filterConnector(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.onelogin_apps.oidc", "apps.#"),
				),
			},
		},
	})
}

// --- Config helpers ---

func testAccAppDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419
}

data "onelogin_app" "test" {
  id = onelogin_oidc_app.test.id
}
`, name)
}

func testAccAppsDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419
}

data "onelogin_apps" "test" {
  depends_on = [onelogin_oidc_app.test]
}
`, name)
}

func testAccAppsDataSourceConfig_filterConnector(name string) string {
	return fmt.Sprintf(`
resource "onelogin_oidc_app" "test" {
  name         = %[1]q
  connector_id = 108419
}

data "onelogin_apps" "oidc" {
  connector_id = 108419
  depends_on   = [onelogin_oidc_app.test]
}
`, name)
}
