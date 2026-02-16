package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/spbsoluble/terraform-provider-onelogin/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"onelogin": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("ONELOGIN_CLIENT_ID"); v == "" {
		t.Fatal("ONELOGIN_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("ONELOGIN_CLIENT_SECRET"); v == "" {
		t.Fatal("ONELOGIN_CLIENT_SECRET must be set for acceptance tests")
	}
	if v := os.Getenv("ONELOGIN_API_URL"); v == "" {
		t.Fatal("ONELOGIN_API_URL must be set for acceptance tests")
	}
}
