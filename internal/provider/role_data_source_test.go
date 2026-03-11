package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoleDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-role-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First create a role, then read it back via data source
			{
				Config: testAccRoleDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.onelogin_role.test", "id",
						"onelogin_role.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.onelogin_role.test", "name",
						"onelogin_role.test", "name",
					),
				),
			},
		},
	})
}

func TestAccRolesDataSource_basic(t *testing.T) {
	t.Skip("Disabled: listing all roles causes API timeout")
	rName := acctest.RandomWithPrefix("tf-acc-roles-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRolesDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The roles list should contain at least the role we created
					resource.TestCheckResourceAttrSet("data.onelogin_roles.test", "roles.#"),
				),
			},
		},
	})
}

func TestAccRolesDataSource_withNameFilter(t *testing.T) {
	t.Skip("Disabled: listing all roles causes API timeout")
	rName := acctest.RandomWithPrefix("tf-acc-roles-filter")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRolesDataSourceConfig_withNameFilter(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.onelogin_roles.filtered", "roles.#"),
				),
			},
		},
	})
}

func TestAccRoleDataSource_byName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-role-ds-name")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceConfig_byName(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.onelogin_role.test", "name", rName),
					resource.TestCheckResourceAttrSet("data.onelogin_role.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.onelogin_role.test", "id",
						"onelogin_role.test", "id",
					),
				),
			},
		},
	})
}

func testAccRoleDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_role" "test" {
  name = %[1]q
}

data "onelogin_role" "test" {
  id = onelogin_role.test.id
}
`, name)
}

func testAccRoleDataSourceConfig_byName(name string) string {
	return fmt.Sprintf(`
resource "onelogin_role" "test" {
  name = %[1]q
}

data "onelogin_role" "test" {
  name       = onelogin_role.test.name
  depends_on = [onelogin_role.test]
}
`, name)
}

func testAccRolesDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_role" "test" {
  name = %[1]q
}

data "onelogin_roles" "test" {
  depends_on = [onelogin_role.test]
}
`, name)
}

func testAccRolesDataSourceConfig_withNameFilter(name string) string {
	return fmt.Sprintf(`
resource "onelogin_role" "test" {
  name = %[1]q
}

data "onelogin_roles" "filtered" {
  name_filter = %[1]q
  depends_on  = [onelogin_role.test]
}
`, name)
}
