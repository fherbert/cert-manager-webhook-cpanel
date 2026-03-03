package solver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/fherbert/cert-manager-webhook-cpanel/internal/cpanel"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type CPanelDNSProviderSolver struct {
	client *kubernetes.Clientset
}

type CPanelDNSProviderConfig struct {
	Endpoint          string          `json:"endpoint"`
	Username          string          `json:"username"`
	APITokenSecretRef SecretReference `json:"apiTokenSecretRef"`
	Zone              string          `json:"zone"`
	TTL               *int            `json:"ttl,omitempty"`
}

type SecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

func (c *CPanelDNSProviderSolver) Name() string {
	return "cpanel"
}

func (c *CPanelDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.V(2).Infof("Present challenge for domain %s", ch.ResolvedFQDN)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	token, err := c.getAPIToken(cfg, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("failed to get API token: %w", err)
	}

	cpanelClient := cpanel.NewClient(cpanel.Config{
		Endpoint: cfg.Endpoint,
		Username: cfg.Username,
		Token:    token,
	})

	zone, recordName := c.extractZoneAndRecord(ch.ResolvedFQDN, cfg)

	ttl := 300
	if cfg.TTL != nil && *cfg.TTL > 0 {
		ttl = *cfg.TTL
	}

	klog.V(2).Infof("Adding TXT record: zone=%s, name=%s, value=%s, ttl=%d", zone, recordName, ch.Key, ttl)

	if err := cpanelClient.AddTXTRecord(zone, recordName, ch.Key, ttl); err != nil {
		return fmt.Errorf("failed to add TXT record: %w", err)
	}

	klog.V(2).Infof("Successfully added TXT record for %s", ch.ResolvedFQDN)
	return nil
}

func (c *CPanelDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.V(2).Infof("CleanUp challenge for domain %s", ch.ResolvedFQDN)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	token, err := c.getAPIToken(cfg, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("failed to get API token: %w", err)
	}

	cpanelClient := cpanel.NewClient(cpanel.Config{
		Endpoint: cfg.Endpoint,
		Username: cfg.Username,
		Token:    token,
	})

	zone, recordName := c.extractZoneAndRecord(ch.ResolvedFQDN, cfg)

	klog.V(2).Infof("Deleting TXT record: zone=%s, name=%s, value=%s", zone, recordName, ch.Key)

	if err := cpanelClient.DeleteTXTRecord(zone, recordName, ch.Key); err != nil {
		return fmt.Errorf("failed to delete TXT record: %w", err)
	}

	klog.V(2).Infof("Successfully deleted TXT record for %s", ch.ResolvedFQDN)
	return nil
}

func (c *CPanelDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	klog.V(2).Info("Initializing cPanel DNS provider solver")

	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	c.client = cl
	return nil
}

func (c *CPanelDNSProviderSolver) getAPIToken(cfg *CPanelDNSProviderConfig, namespace string) (string, error) {
	secret, err := c.client.CoreV1().Secrets(namespace).Get(
		context.Background(),
		cfg.APITokenSecretRef.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, cfg.APITokenSecretRef.Name, err)
	}

	token, ok := secret.Data[cfg.APITokenSecretRef.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s/%s", cfg.APITokenSecretRef.Key, namespace, cfg.APITokenSecretRef.Name)
	}

	return string(token), nil
}

func (c *CPanelDNSProviderSolver) extractZoneAndRecord(fqdn string, cfg *CPanelDNSProviderConfig) (string, string) {
	fqdn = strings.TrimSuffix(fqdn, ".")
	return cfg.Zone, fqdn
}

func loadConfig(cfgJSON *apiextensionsv1.JSON) (*CPanelDNSProviderConfig, error) {
	cfg := &CPanelDNSProviderConfig{}

	if cfgJSON == nil {
		return nil, fmt.Errorf("config is required")
	}

	if err := json.Unmarshal(cfgJSON.Raw, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if cfg.Zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	if cfg.APITokenSecretRef.Name == "" || cfg.APITokenSecretRef.Key == "" {
		return nil, fmt.Errorf("apiTokenSecretRef.name and apiTokenSecretRef.key are required")
	}

	return cfg, nil
}
