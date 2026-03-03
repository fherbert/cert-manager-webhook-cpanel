# cert-manager Webhook for cPanel DNS

A cert-manager ACME DNS01 webhook solver for cPanel DNS.

## Features

- DNS-01 challenge solver for Let's Encrypt certificates
- Works with cPanel/WHM UAPI
- Supports cPanel API token authentication
- Runs behind CGNAT (no inbound connectivity required)

## Prerequisites

- Kubernetes cluster with cert-manager v1.14+ installed
- cPanel/WHM account with API access
- cPanel API token with DNS zone management permissions

## Installation

### 1. Create cPanel API Token

In cPanel:
1. Go to **Security** → **Manage API Tokens**
2. Create a new token with DNS zone permissions
3. Copy the token (you'll need it for the Kubernetes secret)

### 2. Deploy the Webhook

```bash
# Clone the repository
git clone https://github.com/fherbert/cert-manager-webhook-cpanel
cd cert-manager-webhook-cpanel

# Build and push the Docker image (adjust registry as needed)
task docker-build
# Or with custom registry:
# REGISTRY=your-registry.com task docker-push

# Deploy to Kubernetes
kubectl apply -f deploy/manifests/
```

### 3. Create API Token Secret

```bash
kubectl create secret generic cpanel-api-token \
  --from-literal=token='YOUR_CPANEL_API_TOKEN' \
  -n cert-manager
```

### 4. Create ClusterIssuer

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-cpanel
spec:
  acme:
    email: you@example.com
    server: https://acme-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: letsencrypt-cpanel-account-key
    solvers:
    - dns01:
        webhook:
          groupName: acme.cpanel.webhook
          solverName: cpanel
          config:
            endpoint: "https://cpanel.example.com:2083"
            username: "cpanel_username"
            apiTokenSecretRef:
              name: cpanel-api-token
              key: token
            ttl: 300
```

For staging (testing):

```yaml
server: https://acme-staging-v02.api.letsencrypt.org/directory
```

### 5. Request a Certificate

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-tls
  namespace: default
spec:
  secretName: example-tls
  issuerRef:
    name: letsencrypt-cpanel
    kind: ClusterIssuer
  dnsNames:
  - example.com
  - "*.example.com"
```

## Configuration

### Webhook Config Options

| Field | Required | Description |
|-------|----------|-------------|
| `endpoint` | Yes | cPanel endpoint URL (e.g., `https://cpanel.example.com:2083`) |
| `username` | Yes | cPanel username |
| `apiTokenSecretRef.name` | Yes | Kubernetes secret name containing API token |
| `apiTokenSecretRef.key` | Yes | Key in secret containing the token |
| `ttl` | No | DNS record TTL in seconds (default: 300) |

## Development

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- Task (taskfile.dev)

### Build

```bash
# Download dependencies
task deps

# Build binary
task build

# Run tests
task test

# Build Docker image
task docker-build
```

### Testing

```bash
# Run unit tests
task test

# Run with coverage
task test-coverage
```

## How It Works

1. cert-manager creates a DNS-01 challenge for your domain
2. cert-manager calls the webhook's `Present()` method
3. The webhook uses cPanel UAPI to create a TXT record: `_acme-challenge.yourdomain.com`
4. Let's Encrypt validates the TXT record via public DNS
5. cert-manager retrieves the certificate
6. cert-manager calls the webhook's `CleanUp()` method
7. The webhook deletes the TXT record from cPanel

The webhook **only** manages DNS records. cert-manager handles all certificate operations.

## Troubleshooting

### Check webhook logs

```bash
kubectl logs -n cert-manager deployment/cert-manager-webhook-cpanel
```

### Check certificate status

```bash
kubectl describe certificate example-tls
kubectl describe certificaterequest -n default
```

### Common issues

**"failed to get API token"**
- Ensure the secret exists in the same namespace as the Certificate
- Verify secret name and key match your ClusterIssuer config

**"cPanel API error"**
- Check cPanel endpoint URL is correct
- Verify API token has DNS zone permissions
- Ensure cPanel username is correct

**"DNS propagation timeout"**
- Check that your domain's NS records point to cPanel DNS servers
- Verify the TXT record was created in cPanel DNS zone editor

## License

MIT

## Contributing

Contributions welcome! Please follow conventional commits standard.
