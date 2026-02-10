# Terraform Provider for Porkbun

This is an unofficial Terraform provider for interacting with the Porkbun API. It allows you to manage your domains' configuration via Infrastructure as Code.

## Supported Resources

- **DNS Records**: Manage A, AAAA, CNAME, TXT, MX, NS, and other record types.
- **Nameservers**: Set the authoritative nameservers for your domain.
- **Glue Records**: Manage nameserver IP addresses (host records) at the registry level.

## Configuration

To use this provider, you need an API Key and Secret Key from your Porkbun account settings.

```hcl
terraform {
  required_providers {
    porkbun = {
      source = "kyswtn/porkbun"
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
```

## Disclaimer

This project is not affiliated with or endorsed by Porkbun. Use it at your own risk.
