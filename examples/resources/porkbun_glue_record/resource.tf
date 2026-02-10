resource "porkbun_glue_record" "ns1" {
  domain    = "example.com"
  subdomain = "ns1"
  ips = [
    "192.0.2.1",
    "2001:db8::1"
  ]
}
