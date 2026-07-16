package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	porkbun "github.com/vitvio/terraform-provider-porkbun/internal/client"
)

func TestDNSSECRecordResource(t *testing.T) {
	providerConfig, mockbunServer := getProviderConfigWithMockServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test create and read.
			{
				Config: providerConfig + `
					resource "porkbun_dnssec_record" "test" {
						domain      = "example.com"
						key_tag     = "5878"
						algorithm   = "13"
						digest_type = "2"
						digest      = "2f9b8d20eb9ee356fbd4b10bcf5ea18b975beb9f3bfcfd67c8bd9021d2e7b06d"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "id", "example.com:5878"),
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "domain", "example.com"),
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "key_tag", "5878"),
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "algorithm", "13"),
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "digest_type", "2"),
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "digest", "2f9b8d20eb9ee356fbd4b10bcf5ea18b975beb9f3bfcfd67c8bd9021d2e7b06d"),
				),
			},
			// Test that changing an attribute forces replacement, as Porkbun
			// has no DNSSEC update endpoint.
			{
				Config: providerConfig + `
					resource "porkbun_dnssec_record" "test" {
						domain      = "example.com"
						key_tag     = "34505"
						algorithm   = "13"
						digest_type = "2"
						digest      = "9e32c968c1efd12dab52ce7dcb7f7fc2f8b58bea17b90d0b3d0a3e0afca7cbef"
					}
				`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("porkbun_dnssec_record.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "id", "example.com:34505"),
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "key_tag", "34505"),
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "digest", "9e32c968c1efd12dab52ce7dcb7f7fc2f8b58bea17b90d0b3d0a3e0afca7cbef"),
				),
			},
			// Test import.
			{
				ResourceName:      "porkbun_dnssec_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test replacement with create_before_destroy while the key tag
			// stays the same: the deferred Delete must not remove the record
			// the replacement just created.
			{
				Config: providerConfig + `
					resource "porkbun_dnssec_record" "test" {
						domain      = "example.com"
						key_tag     = "34505"
						algorithm   = "13"
						digest_type = "2"
						digest      = "c7f4f3422e5b4d75c95bbcfbe6ffa06cbdcae4ba0e5ccdfb0917f3d0938ba1e4"

						lifecycle {
							create_before_destroy = true
						}
					}
				`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("porkbun_dnssec_record.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("porkbun_dnssec_record.test", "digest", "c7f4f3422e5b4d75c95bbcfbe6ffa06cbdcae4ba0e5ccdfb0917f3d0938ba1e4"),
				),
			},
			// Test that a server-side hex-case normalization of the digest
			// does not produce drift (which would force replacement).
			{
				PreConfig: func() {
					mockbunServer.SetDNSSECRecords("example.com", map[string]porkbun.DNSSECRecord{
						"34505": {
							KeyTag:     "34505",
							Alg:        "13",
							DigestType: "2",
							Digest:     "C7F4F3422E5B4D75C95BBCFBE6FFA06CBDCAE4BA0E5CCDFB0917F3D0938BA1E4",
						},
					})
				},
				RefreshState: true,
			},
			// Test out-of-band deletion: the refresh must drop the record
			// from state and plan a re-create.
			{
				PreConfig: func() {
					mockbunServer.SetDNSSECRecords("example.com", map[string]porkbun.DNSSECRecord{})
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
