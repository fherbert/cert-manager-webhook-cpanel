package cpanel

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		Endpoint: "https://cpanel.example.com:2083",
		Username: "testuser",
		Token:    "testtoken",
	}

	client := NewClient(cfg)

	if client.endpoint != "https://cpanel.example.com:2083" {
		t.Errorf("expected endpoint %s, got %s", cfg.Endpoint, client.endpoint)
	}
	if client.username != cfg.Username {
		t.Errorf("expected username %s, got %s", cfg.Username, client.username)
	}
	if client.token != cfg.Token {
		t.Errorf("expected token %s, got %s", cfg.Token, client.token)
	}
}

func TestAddTXTRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/execute/DNS/add_zone_record" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth != "cpanel testuser:testtoken" {
			t.Errorf("unexpected auth header: %s", auth)
		}

		query := r.URL.Query()
		if query.Get("domain") != "example.com" {
			t.Errorf("unexpected domain: %s", query.Get("domain"))
		}
		if query.Get("name") != "_acme-challenge.example.com" {
			t.Errorf("unexpected name: %s", query.Get("name"))
		}
		if query.Get("txtdata") != "test-token" {
			t.Errorf("unexpected txtdata: %s", query.Get("txtdata"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":{"status":1,"errors":[],"messages":[],"data":{}}}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		Endpoint: server.URL,
		Username: "testuser",
		Token:    "testtoken",
	})

	err := client.AddTXTRecord("example.com", "_acme-challenge.example.com", "test-token", 300)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteTXTRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/execute/DNS/fetch_zone_records" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":{"status":1,"errors":[],"messages":[],"data":{"data":[{"line":1,"txtdata":"test-token","type":"TXT"}]}}}`))
			return
		}

		if r.URL.Path == "/execute/DNS/remove_zone_record" {
			query := r.URL.Query()
			if query.Get("line") != "1" {
				t.Errorf("unexpected line: %s", query.Get("line"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":{"status":1,"errors":[],"messages":[],"data":{}}}`))
			return
		}

		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := NewClient(Config{
		Endpoint: server.URL,
		Username: "testuser",
		Token:    "testtoken",
	})

	err := client.DeleteTXTRecord("example.com", "_acme-challenge.example.com", "test-token")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
