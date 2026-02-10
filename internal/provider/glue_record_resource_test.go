package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestGlueRecordResource(t *testing.T) {
	providerConfig, _ := getProviderConfigWithMockServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test create and read.
			{
				Config: providerConfig + `
					resource "porkbun_glue_record" "test" {
						domain    = "example.com"
						subdomain = "ns1"
						ips       = ["192.168.1.1", "2001:db8::1"]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_glue_record.test", "domain", "example.com"),
					resource.TestCheckResourceAttr("porkbun_glue_record.test", "subdomain", "ns1"),
					resource.TestCheckResourceAttr("porkbun_glue_record.test", "ips.#", "2"),
					resource.TestCheckResourceAttr("porkbun_glue_record.test", "ips.0", "192.168.1.1"),
				),
			},
			// Test update.
			{
				Config: providerConfig + `
					resource "porkbun_glue_record" "test" {
						domain    = "example.com"
						subdomain = "ns1"
						ips       = ["192.168.1.2"]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_glue_record.test", "ips.#", "1"),
					resource.TestCheckResourceAttr("porkbun_glue_record.test", "ips.0", "192.168.1.2"),
				),
			},
			// Test import.
			{
				ResourceName:      "porkbun_glue_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
