package cpanel

import (
	"bytes"
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
		Status   int                    `json:"status"`
		Errors   []string               `json:"errors"`
		Messages []string               `json:"messages"`
		Data     map[string]interface{} `json:"data"`
	} `json:"result"`
}

func (c *Client) AddTXTRecord(zone, name, value string, ttl int) error {
	params := url.Values{}
	params.Set("domain", zone)
	params.Set("name", name)
	params.Set("txtdata", value)
	params.Set("ttl", fmt.Sprintf("%d", ttl))

	return c.callUAPI("DNS", "add_zone_record", params)
}

func (c *Client) DeleteTXTRecord(zone, name, value string) error {
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
			line, ok := record["line"].(float64)
			if !ok {
				continue
			}
			
			params := url.Values{}
			params.Set("domain", zone)
			params.Set("line", fmt.Sprintf("%.0f", line))
			
			return c.callUAPI("DNS", "remove_zone_record", params)
		}
	}

	return nil
}

func (c *Client) fetchZoneRecords(zone, name, recordType string) ([]map[string]interface{}, error) {
	params := url.Values{}
	params.Set("domain", zone)
	params.Set("name", name)
	params.Set("type", recordType)

	resp, err := c.callUAPIWithResponse("DNS", "fetch_zone_records", params)
	if err != nil {
		return nil, err
	}

	data, ok := resp.Result.Data["data"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	records := make([]map[string]interface{}, 0, len(data))
	for _, item := range data {
		if record, ok := item.(map[string]interface{}); ok {
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
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Result.Status != 1 {
		errMsg := "unknown error"
		if len(apiResp.Result.Errors) > 0 {
			errMsg = strings.Join(apiResp.Result.Errors, "; ")
		}
		return nil, fmt.Errorf("cPanel API error: %s", errMsg)
	}

	return &apiResp, nil
}
