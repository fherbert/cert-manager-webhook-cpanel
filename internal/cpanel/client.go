package cpanel

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	endpoint string
	username string
	token    string
	client   *http.Client
}

type Config struct {
	Endpoint string
	Username string
	Token    string
}

func NewClient(cfg Config) *Client {
	return &Client{
		endpoint: strings.TrimSuffix(cfg.Endpoint, "/"),
		username: cfg.Username,
		token:    cfg.Token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type uapiResponse struct {
	Result struct {
		Status   int             `json:"status"`
		Errors   []string        `json:"errors"`
		Messages []string        `json:"messages"`
		Data     json.RawMessage `json:"data"`
	} `json:"result"`
}

func (c *Client) AddTXTRecord(zone, name, value string, ttl int) error {
	serial, err := c.getZoneSerial(zone)
	if err != nil {
		return fmt.Errorf("failed to get zone serial: %w", err)
	}

	record := map[string]interface{}{
		"dname":       name,
		"ttl":         ttl,
		"record_type": "TXT",
		"data":        []string{value},
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	params := url.Values{}
	params.Set("zone", zone)
	params.Set("serial", fmt.Sprintf("%d", serial))
	params.Set("add", string(recordJSON))

	return c.callUAPI("DNS", "mass_edit_zone", params)
}

func (c *Client) DeleteTXTRecord(zone, name, value string) error {
	serial, err := c.getZoneSerial(zone)
	if err != nil {
		return fmt.Errorf("failed to get zone serial: %w", err)
	}

	records, err := c.fetchZoneRecords(zone, name, "TXT")
	if err != nil {
		return err
	}

	for _, record := range records {
		txtData, ok := record["txtdata"].(string)
		if !ok {
			continue
		}

		if strings.TrimSpace(txtData) == strings.TrimSpace(value) {
			lineIndex, ok := record["line_index"].(float64)
			if !ok {
				continue
			}

			params := url.Values{}
			params.Set("zone", zone)
			params.Set("serial", fmt.Sprintf("%d", serial))
			params.Set("remove", fmt.Sprintf("%.0f", lineIndex))

			return c.callUAPI("DNS", "mass_edit_zone", params)
		}
	}

	return nil
}

func (c *Client) getZoneSerial(zone string) (int, error) {
	params := url.Values{}
	params.Set("zone", zone)

	resp, err := c.callUAPIWithResponse("DNS", "parse_zone", params)
	if err != nil {
		return 0, err
	}

	// parse_zone returns data as an array of zone records
	// Find the SOA record which contains the serial in data_b64[2]
	var parsed []interface{}
	if err := json.Unmarshal(resp.Result.Data, &parsed); err != nil {
		return 0, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	for _, item := range parsed {
		record, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		recType, _ := record["record_type"].(string)
		if recType == "SOA" {
			// SOA record data is: [primary-ns, admin-email, serial, refresh, retry, expire, minimum]
			data, ok := record["data_b64"].([]interface{})
			if !ok || len(data) < 3 {
				continue
			}

			// data[2] is the serial number (base64 encoded)
			serialB64, ok := data[2].(string)
			if !ok {
				continue
			}

			// Decode base64
			serialBytes, err := base64.StdEncoding.DecodeString(serialB64)
			if err != nil {
				continue
			}

			// Parse as integer
			serial := 0
			fmt.Sscanf(string(serialBytes), "%d", &serial)
			return serial, nil
		}
	}

	return 0, fmt.Errorf("SOA record not found in zone")
}

func (c *Client) fetchZoneRecords(zone, name, recordType string) ([]map[string]interface{}, error) {
	params := url.Values{}
	params.Set("zone", zone)

	resp, err := c.callUAPIWithResponse("DNS", "parse_zone", params)
	if err != nil {
		return nil, err
	}

	var parsed []interface{}
	if err := json.Unmarshal(resp.Result.Data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	records := make([]map[string]interface{}, 0)
	for _, item := range parsed {
		record, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		recType, _ := record["record_type"].(string)
		recName, _ := record["dname"].(string)

		if recType == recordType && (name == "" || strings.HasPrefix(recName, name)) {
			if data, ok := record["data"].([]interface{}); ok && len(data) > 0 {
				if txtData, ok := data[0].(string); ok {
					record["txtdata"] = txtData
				}
			}
			records = append(records, record)
		}
	}

	return records, nil
}

func (c *Client) callUAPI(module, function string, params url.Values) error {
	_, err := c.callUAPIWithResponse(module, function, params)
	return err
}

func (c *Client) callUAPIWithResponse(module, function string, params url.Values) (*uapiResponse, error) {
	apiURL := fmt.Sprintf("%s/execute/%s/%s", c.endpoint, module, function)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("cpanel %s:%s", c.username, c.token))
	req.URL.RawQuery = params.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp uapiResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	if apiResp.Result.Status != 1 {
		errMsg := "unknown error"
		if len(apiResp.Result.Errors) > 0 {
			errMsg = strings.Join(apiResp.Result.Errors, "; ")
		}
		return nil, fmt.Errorf("cPanel API error: %s, status: %d, messages: %v, response body: %s",
			errMsg, apiResp.Result.Status, apiResp.Result.Messages, string(body))
	}

	return &apiResp, nil
}
