package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccAppResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_app.test", "id"),
					resource.TestCheckResourceAttr("onelogin_app.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_app.test", "connector_id", "108419"),
				),
			},
			// ImportState
			{
				ResourceName:            "onelogin_app.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configuration"},
			},
		},
	})
}

func TestAccAppResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-app-upd")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_app.test", "name", rName),
				),
			},
			// Update name
			{
				Config: testAccAppResourceConfig_basic(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_app.test", "name", rName+"-updated"),
				),
			},
		},
	})
}

// --- Config helpers ---

func testAccAppResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "onelogin_app" "test" {
  name         = %[1]q
  connector_id = 108419
}
`, name)
}
