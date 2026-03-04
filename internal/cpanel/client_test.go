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
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/execute/DNS/parse_zone" {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Serial 2024030401 in base64 is "MjAyNDAzMDQwMQ=="
			if callCount == 1 {
				// First call: check for existing records - return empty (no existing TXT records)
				w.Write([]byte(`{"status":1,"errors":null,"messages":null,"warnings":null,"metadata":{},"data":[{"record_type":"SOA","data_b64":["bnMxLmV4YW1wbGUuY29tLg==","YWRtaW5AZXhhbXBsZS5jb20u","MjAyNDAzMDQwMQ==","3600","1800","1209600","86400"]}]}`))
			} else {
				// Second call: get serial for adding record
				w.Write([]byte(`{"status":1,"errors":null,"messages":null,"warnings":null,"metadata":{},"data":[{"record_type":"SOA","data_b64":["bnMxLmV4YW1wbGUuY29tLg==","YWRtaW5AZXhhbXBsZS5jb20u","MjAyNDAzMDQwMQ==","3600","1800","1209600","86400"]}]}`))
			}
			return
		}

		if r.URL.Path == "/execute/DNS/mass_edit_zone" {
			auth := r.Header.Get("Authorization")
			if auth != "cpanel testuser:testtoken" {
				t.Errorf("unexpected auth header: %s", auth)
			}

			query := r.URL.Query()
			if query.Get("zone") != "example.com" {
				t.Errorf("unexpected zone: %s", query.Get("zone"))
			}
			if query.Get("serial") != "2024030401" {
				t.Errorf("unexpected serial: %s", query.Get("serial"))
			}
			if query.Get("add") == "" {
				t.Error("expected add parameter")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":1,"errors":null,"messages":null,"warnings":null,"metadata":{}}`))
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

	err := client.AddTXTRecord("example.com", "_acme-challenge.example.com", "test-token", 300)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAddTXTRecordIdempotent(t *testing.T) {
	massEditCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/execute/DNS/parse_zone" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Return existing TXT record with the same value
			w.Write([]byte(`{"status":1,"errors":null,"messages":null,"warnings":null,"metadata":{},"data":[{"record_type":"SOA","data_b64":["bnMxLmV4YW1wbGUuY29tLg==","YWRtaW5AZXhhbXBsZS5jb20u","MjAyNDAzMDQwMQ==","3600","1800","1209600","86400"]},{"line_index":5,"dname":"_acme-challenge.example.com","record_type":"TXT","txtdata":"test-token","data":["test-token"]}]}`))
			return
		}

		if r.URL.Path == "/execute/DNS/mass_edit_zone" {
			massEditCalled = true
			t.Error("mass_edit_zone should not be called when record already exists")
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

	// Add the same record twice - second call should be idempotent
	err := client.AddTXTRecord("example.com", "_acme-challenge.example.com", "test-token", 300)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if massEditCalled {
		t.Error("mass_edit_zone was called when it shouldn't have been")
	}
}

func TestDeleteTXTRecord(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/execute/DNS/parse_zone" {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if callCount == 1 {
				// First call for getZoneSerial - return SOA record
				w.Write([]byte(`{"status":1,"errors":null,"messages":null,"warnings":null,"metadata":{},"data":[{"record_type":"SOA","data_b64":["bnMxLmV4YW1wbGUuY29tLg==","YWRtaW5AZXhhbXBsZS5jb20u","MjAyNDAzMDQwMQ==","3600","1800","1209600","86400"]}]}`))
			} else {
				// Second call for fetchZoneRecords - return SOA and TXT records
				w.Write([]byte(`{"status":1,"errors":null,"messages":null,"warnings":null,"metadata":{},"data":[{"record_type":"SOA","data_b64":["bnMxLmV4YW1wbGUuY29tLg==","YWRtaW5AZXhhbXBsZS5jb20u","MjAyNDAzMDQwMQ==","3600","1800","1209600","86400"]},{"line_index":5,"dname":"_acme-challenge.example.com","record_type":"TXT","data":["test-token"]}]}`))
			}
			return
		}

		if r.URL.Path == "/execute/DNS/mass_edit_zone" {
			query := r.URL.Query()
			if query.Get("zone") != "example.com" {
				t.Errorf("unexpected zone: %s", query.Get("zone"))
			}
			if query.Get("serial") != "2024030401" {
				t.Errorf("unexpected serial: %s", query.Get("serial"))
			}
			if query.Get("remove") != "5" {
				t.Errorf("unexpected remove: %s", query.Get("remove"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":1,"errors":null,"messages":null,"warnings":null,"metadata":{}}`))
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
