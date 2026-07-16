package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// DNSSECRecord holds the DS data variant of a Porkbun DNSSEC record.
// Porkbun's API names the algorithm field `alg`, not `algorithm`.
type DNSSECRecord struct {
	KeyTag     string `json:"keyTag"`
	Alg        string `json:"alg"`
	DigestType string `json:"digestType"`
	Digest     string `json:"digest"`
}

func (c *Client) CreateDNSSECRecord(ctx context.Context, domain string, record DNSSECRecord) error {
	url := c.baseURL.JoinPath("dns", "createDnssecRecord", domain)

	response := status{}
	err := c.do(ctx, url, record, &response)

	if err != nil {
		return err
	}

	if response.failed() {
		return response
	}

	return nil
}

type getDNSSECRecordsResponse struct {
	status
	// Kept raw because an empty record set may be serialized as a JSON array
	// instead of an object keyed by key tag.
	Records json.RawMessage `json:"records"`
}

func (c *Client) GetDNSSECRecords(ctx context.Context, domain string) (map[string]DNSSECRecord, error) {
	url := c.baseURL.JoinPath("dns", "getDnssecRecords", domain)

	var response getDNSSECRecordsResponse
	err := c.do(ctx, url, nil, &response)

	if err != nil {
		return nil, err
	}

	if response.failed() {
		return nil, response.status
	}

	records := map[string]DNSSECRecord{}
	if bytes.HasPrefix(bytes.TrimSpace(response.Records), []byte("{")) {
		if err := json.Unmarshal(response.Records, &records); err != nil {
			return nil, fmt.Errorf("unmarshaling DNSSEC records failed: %w", err)
		}
	}

	return records, nil
}

func (c *Client) DeleteDNSSECRecord(ctx context.Context, domain, keyTag string) error {
	url := c.baseURL.JoinPath("dns", "deleteDnssecRecord", domain, keyTag)

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
