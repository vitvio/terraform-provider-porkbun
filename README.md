# Terraform Provider for Porkbun

This is an unofficial Terraform provider for interacting with the Porkbun API. It allows you to manage your domains' configuration via Infrastructure as Code.

## Supported Resources

- **DNS Records**: Manage A, AAAA, CNAME, TXT, MX, NS, and other record types.
- **Nameservers**: Set the authoritative nameservers for your domain.
- **Glue Records**: Manage nameserver IP addresses (host records) at the registry level.
- **DNSSEC Records**: Manage DS records at the registry level.

## Configuration

To use this provider, you need an API Key and Secret Key from your Porkbun account settings.

```hcl
terraform {
  required_providers {
    porkbun = {
      source = "vitvio/porkbun"
    }
  }
}

provider "porkbun" {
  api_key    = "YOUR_API_KEY"
  secret_key = "YOUR_SECRET_KEY"
}
```

## Example Usage

```hcl
resource "porkbun_dns_record" "www" {
  domain  = "example.com"
  name    = "www"
  type    = "A"
  content = "192.0.2.1"
}

resource "porkbun_glue_record" "ns1" {
  domain    = "example.com"
  subdomain = "ns1"
  ips       = ["198.51.100.1"]
}

resource "porkbun_dnssec_record" "example" {
  domain      = "example.com"
  key_tag     = "5878"
  algorithm   = "13"
  digest_type = "2"
  digest      = "2f9b8d20eb9ee356fbd4b10bcf5ea18b975beb9f3bfcfd67c8bd9021d2e7b06d"
}
```

## Disclaimer

This project is not affiliated with or endorsed by Porkbun. Use it at your own risk.
