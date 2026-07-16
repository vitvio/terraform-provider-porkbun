resource "porkbun_dnssec_record" "example" {
  domain      = "example.com"
  key_tag     = "5878"
  algorithm   = "13"
  digest_type = "2"
  digest      = "2f9b8d20eb9ee356fbd4b10bcf5ea18b975beb9f3bfcfd67c8bd9021d2e7b06d"
}
