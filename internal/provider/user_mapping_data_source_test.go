package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserMappingDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.onelogin_user_mapping.test", "id",
						"onelogin_user_mapping.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.onelogin_user_mapping.test", "name",
						"onelogin_user_mapping.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.onelogin_user_mapping.test", "match",
						"onelogin_user_mapping.test", "match",
					),
					resource.TestCheckResourceAttrPair(
						"data.onelogin_user_mapping.test", "enabled",
						"onelogin_user_mapping.test", "enabled",
					),
				),
			},
		},
	})
}

func TestAccUserMappingDataSource_withConditionsAndActions(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-ds-full")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingDataSourceConfig_full(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.onelogin_user_mapping.test", "id",
						"onelogin_user_mapping.test", "id",
					),
					resource.TestCheckResourceAttr("data.onelogin_user_mapping.test", "conditions.#", "1"),
					resource.TestCheckResourceAttr("data.onelogin_user_mapping.test", "conditions.0.source", "email"),
					resource.TestCheckResourceAttr("data.onelogin_user_mapping.test", "conditions.0.operator", "~"),
					resource.TestCheckResourceAttr("data.onelogin_user_mapping.test", "conditions.0.value", "@example.com"),
					resource.TestCheckResourceAttr("data.onelogin_user_mapping.test", "actions.#", "1"),
					resource.TestCheckResourceAttr("data.onelogin_user_mapping.test", "actions.0.action", "set_status"),
				),
			},
		},
	})
}

func TestAccUserMappingsDataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mappings-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingsDataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.onelogin_user_mappings.test", "mappings.#"),
				),
			},
		},
	})
}

func TestAccUserMappingsDataSource_filterEnabled(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mappings-ds-en")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingsDataSourceConfig_filterEnabled(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.onelogin_user_mappings.enabled", "mappings.#"),
				),
			},
		},
	})
}

func TestAccUserMappingsDataSource_filterHasCondition(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mappings-ds-cond")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingsDataSourceConfig_filterHasCondition(rName),
				// has_condition filter uses format "source:value"; results may be empty
				// depending on API indexing timing, so just verify no error
			},
		},
	})
}

// --- Config helpers ---

func testAccUserMappingDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name  = %[1]q
  match = "all"

  conditions {
    source   = "email"
    operator = "~"
    value    = "@test.com"
  }

  actions {
    action = "set_status"
    value  = ["1"]
  }
}

data "onelogin_user_mapping" "test" {
  id = onelogin_user_mapping.test.id
}
`, name)
}

func testAccUserMappingDataSourceConfig_full(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name  = %[1]q
  match = "all"

  conditions {
    source   = "email"
    operator = "~"
    value    = "@example.com"
  }

  actions {
    action = "set_status"
    value  = ["1"]
  }
}

data "onelogin_user_mapping" "test" {
  id = onelogin_user_mapping.test.id
}
`, name)
}

func testAccUserMappingsDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name  = %[1]q
  match = "all"

  conditions {
    source   = "email"
    operator = "~"
    value    = "@test.com"
  }

  actions {
    action = "set_status"
    value  = ["1"]
  }
}

data "onelogin_user_mappings" "test" {
  depends_on = [onelogin_user_mapping.test]
}
`, name)
}

func testAccUserMappingsDataSourceConfig_filterEnabled(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name    = %[1]q
  match   = "all"
  enabled = true

  conditions {
    source   = "email"
    operator = "~"
    value    = "@test.com"
  }

  actions {
    action = "set_status"
    value  = ["1"]
  }
}

data "onelogin_user_mappings" "enabled" {
  enabled    = true
  depends_on = [onelogin_user_mapping.test]
}
`, name)
}

func testAccUserMappingsDataSourceConfig_filterHasCondition(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name  = %[1]q
  match = "all"

  conditions {
    source   = "email"
    operator = "~"
    value    = "@test.com"
  }

  actions {
    action = "set_status"
    value  = ["1"]
  }
}

data "onelogin_user_mappings" "filtered" {
  has_condition = "email:@test.com"
  depends_on    = [onelogin_user_mapping.test]
}
`, name)
}
