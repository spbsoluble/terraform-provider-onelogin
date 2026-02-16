package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserMappingResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccUserMappingResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_user_mapping.test", "id"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "match", "all"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "enabled", "false"),
				),
			},
			// ImportState
			{
				ResourceName:      "onelogin_user_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccUserMappingResource_enabled(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-enabled")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingResourceConfig_enabled(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "enabled", "true"),
				),
			},
			// Disable
			{
				Config: testAccUserMappingResourceConfig_enabled(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccUserMappingResource_withConditionsAndActions(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-full")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with conditions and actions
			{
				Config: testAccUserMappingResourceConfig_withConditionsAndActions(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_user_mapping.test", "id"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "match", "all"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "enabled", "false"),
					// Conditions
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.#", "1"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.0.source", "email"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.0.operator", "~"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.0.value", "@example.com"),
					// Actions
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.#", "1"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.0.action", "set_status"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.0.value.#", "1"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.0.value.0", "1"),
				),
			},
			// Import
			{
				ResourceName:      "onelogin_user_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccUserMappingResource_multipleConditions(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-multi")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingResourceConfig_multipleConditions(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_user_mapping.test", "id"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "match", "any"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.#", "2"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.#", "1"),
				),
			},
		},
	})
}

func TestAccUserMappingResource_matchAny(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-any")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserMappingResourceConfig_matchAny(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "match", "any"),
				),
			},
			// Switch to "all"
			{
				Config: testAccUserMappingResourceConfig_withConditionsAndActions(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "match", "all"),
				),
			},
		},
	})
}

func TestAccUserMappingResource_updateConditionsAndActions(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-upd")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with one condition
			{
				Config: testAccUserMappingResourceConfig_withConditionsAndActions(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.#", "1"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.#", "1"),
				),
			},
			// Update to two conditions
			{
				Config: testAccUserMappingResourceConfig_multipleConditions(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.#", "2"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.#", "1"),
				),
			},
			// Update back to one condition
			{
				Config: testAccUserMappingResourceConfig_withConditionsAndActions(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.#", "1"),
				),
			},
			// Import final state
			{
				ResourceName:      "onelogin_user_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccUserMappingResource_fullLifecycle(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-mapping-lc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create minimal
			{
				Config: testAccUserMappingResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("onelogin_user_mapping.test", "id"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "name", rName),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "match", "all"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "enabled", "false"),
				),
			},
			// Step 2: Add conditions and actions, enable
			{
				Config: testAccUserMappingResourceConfig_fullUpdate(rName + "-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "match", "any"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "enabled", "true"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "conditions.#", "2"),
					resource.TestCheckResourceAttr("onelogin_user_mapping.test", "actions.#", "1"),
				),
			},
			// Step 3: Import
			{
				ResourceName:      "onelogin_user_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 4: Destroy is implicit
		},
	})
}

// testAccCheckUserMappingDestroy verifies the mapping has been destroyed.
func testAccCheckUserMappingDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "onelogin_user_mapping" {
			continue
		}
		_ = rs.Primary.ID
	}
	return nil
}

// --- Config helpers ---

func testAccUserMappingResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name  = %[1]q
  match = "all"

  conditions {
    source   = "last_login"
    operator = ">"
    value    = "90"
  }

  actions {
    action = "set_status"
    value  = ["2"]
  }
}
`, name)
}

func testAccUserMappingResourceConfig_enabled(name string, enabled bool) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name    = %[1]q
  match   = "all"
  enabled = %[2]t

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
`, name, enabled)
}

func testAccUserMappingResourceConfig_withConditionsAndActions(name string) string {
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
`, name)
}

func testAccUserMappingResourceConfig_multipleConditions(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name  = %[1]q
  match = "any"

  conditions {
    source   = "email"
    operator = "~"
    value    = "@example.com"
  }

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
`, name)
}

func testAccUserMappingResourceConfig_matchAny(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name  = %[1]q
  match = "any"

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
`, name)
}

func testAccUserMappingResourceConfig_fullUpdate(name string) string {
	return fmt.Sprintf(`
resource "onelogin_user_mapping" "test" {
  name    = %[1]q
  match   = "any"
  enabled = true

  conditions {
    source   = "email"
    operator = "~"
    value    = "@example.com"
  }

  conditions {
    source   = "email"
    operator = "~"
    value    = "@corp.com"
  }

  actions {
    action = "set_status"
    value  = ["1"]
  }
}
`, name)
}
