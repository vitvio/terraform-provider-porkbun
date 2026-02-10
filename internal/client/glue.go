package client

import (
	"context"
)

type GlueRecord struct {
	Domain    string
	Subdomain string
	IPs       []string
}

type GlueIPs struct {
	V4 []string `json:"v4"`
	V6 []string `json:"v6"`
}

type getGlueRecordsResponse struct {
	status
	Hosts [][]any `json:"hosts"`
}

func (c *Client) CreateGlueRecord(ctx context.Context, domain, subdomain string, ips []string) error {
	url := c.baseURL.JoinPath("domain", "createGlue", domain, subdomain)

	payload := struct {
		IPs []string `json:"ips"`
	}{
		IPs: ips,
	}

	response := status{}
	err := c.do(ctx, url, payload, &response)

	if err != nil {
		return err
	}

	if response.failed() {
		return response
	}

	return nil
}

func (c *Client) UpdateGlueRecord(ctx context.Context, domain, subdomain string, ips []string) error {
	url := c.baseURL.JoinPath("domain", "updateGlue", domain, subdomain)

	payload := struct {
		IPs []string `json:"ips"`
	}{
		IPs: ips,
	}

	response := status{}
	err := c.do(ctx, url, payload, &response)

	if err != nil {
		return err
	}

	if response.failed() {
		return response
	}

	return nil
}

func (c *Client) DeleteGlueRecord(ctx context.Context, domain, subdomain string) error {
	url := c.baseURL.JoinPath("domain", "deleteGlue", domain, subdomain)

	response := status{}
	err := c.do(ctx, url, nil, &response)

	if err != nil {
		return err
	}

	if response.failed() {
		return response
	}

	return nil
}

func (c *Client) GetGlueRecords(ctx context.Context, domain string) ([]GlueRecord, error) {
	url := c.baseURL.JoinPath("domain", "getGlue", domain)

	var response getGlueRecordsResponse
	err := c.do(ctx, url, nil, &response)

	if err != nil {
		return nil, err
	}

	if response.failed() {
		return nil, response.status
	}

	var records []GlueRecord
	for _, hostRaw := range response.Hosts {
		if len(hostRaw) != 2 {
			continue
		}

		hostName, ok := hostRaw[0].(string)
		if !ok {
			continue
		}

		ipsMap, ok := hostRaw[1].(map[string]any)
		if !ok {
			continue
		}

		var ips []string
		if v4, ok := ipsMap["v4"].([]any); ok {
			for _, ip := range v4 {
				if ipStr, ok := ip.(string); ok {
					ips = append(ips, ipStr)
				}
			}
		}
		if v6, ok := ipsMap["v6"].([]any); ok {
			for _, ip := range v6 {
				if ipStr, ok := ip.(string); ok {
					ips = append(ips, ipStr)
				}
			}
		}

		records = append(records, GlueRecord{
			Domain:    domain,
			Subdomain: hostName, // This provides the full hostname, caller might need to parse
			IPs:       ips,
		})
	}

	return records, nil
}
