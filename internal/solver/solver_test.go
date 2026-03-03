package solver

import (
	"encoding/json"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "valid config",
			config: `{
				"endpoint": "https://cpanel.example.com:2083",
				"username": "testuser",
				"apiTokenSecretRef": {
					"name": "cpanel-token",
					"key": "token"
				}
			}`,
			wantErr: false,
		},
		{
			name: "missing endpoint",
			config: `{
				"username": "testuser",
				"apiTokenSecretRef": {
					"name": "cpanel-token",
					"key": "token"
				}
			}`,
			wantErr: true,
		},
		{
			name: "missing username",
			config: `{
				"endpoint": "https://cpanel.example.com:2083",
				"apiTokenSecretRef": {
					"name": "cpanel-token",
					"key": "token"
				}
			}`,
			wantErr: true,
		},
		{
			name: "missing apiTokenSecretRef",
			config: `{
				"endpoint": "https://cpanel.example.com:2083",
				"username": "testuser"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if err := json.Unmarshal([]byte(tt.config), &raw); err != nil {
				t.Fatalf("failed to unmarshal test config: %v", err)
			}

			cfgJSON := &apiextensionsv1.JSON{Raw: raw}
			_, err := loadConfig(cfgJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractZoneAndRecord(t *testing.T) {
	solver := &CPanelDNSProviderSolver{}

	tests := []struct {
		fqdn       string
		wantZone   string
		wantRecord string
	}{
		{
			fqdn:       "_acme-challenge.example.com.",
			wantZone:   "example.com",
			wantRecord: "_acme-challenge.example.com",
		},
		{
			fqdn:       "_acme-challenge.sub.example.com.",
			wantZone:   "example.com",
			wantRecord: "_acme-challenge.sub.example.com",
		},
		{
			fqdn:       "_acme-challenge.example.co.uk.",
			wantZone:   "co.uk",
			wantRecord: "_acme-challenge.example.co.uk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.fqdn, func(t *testing.T) {
			zone, record := solver.extractZoneAndRecord(tt.fqdn)

			if zone != tt.wantZone {
				t.Errorf("extractZoneAndRecord() zone = %v, want %v", zone, tt.wantZone)
			}
			if record != tt.wantRecord {
				t.Errorf("extractZoneAndRecord() record = %v, want %v", record, tt.wantRecord)
			}
		})
	}
}

func TestSolverName(t *testing.T) {
	solver := &CPanelDNSProviderSolver{}
	if solver.Name() != "cpanel" {
		t.Errorf("Name() = %v, want %v", solver.Name(), "cpanel")
	}
}
