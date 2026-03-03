# Quick Start Guide

This guide will get you up and running with cert-manager webhook for cPanel DNS in under 10 minutes.

## Prerequisites

- Kubernetes cluster (1.21+)
- cert-manager v1.14+ installed
- cPanel account with API access
- Docker (for building the image)

## Step 1: Build and Push the Docker Image

```bash
cd cert-manager-webhook-cpanel

# Build the image
docker build -t your-registry.com/cert-manager-webhook-cpanel:v1.0.0 .

# Push to your registry
docker push your-registry.com/cert-manager-webhook-cpanel:v1.0.0
```

Update `deploy/manifests/deployment.yaml` to use your image:

```yaml
spec:
  containers:
    - name: webhook
      image: your-registry.com/cert-manager-webhook-cpanel:v1.0.0
```

## Step 2: Get Your cPanel API Token

1. Log into cPanel
2. Navigate to **Security** → **Manage API Tokens**
3. Click **Create** and give it a name (e.g., "cert-manager")
4. Copy the token immediately (you won't see it again)

## Step 3: Create the API Token Secret

```bash
kubectl create secret generic cpanel-api-token \
  --from-literal=token='YOUR_CPANEL_API_TOKEN_HERE' \
  -n cert-manager
```

## Step 4: Deploy the Webhook

```bash
kubectl apply -f deploy/manifests/
```

Verify it's running:

```bash
kubectl get pods -n cert-manager | grep cpanel
# Should show: cert-manager-webhook-cpanel-xxx   1/1   Running
```

## Step 5: Create a ClusterIssuer

Create `clusterissuer.yaml`:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-cpanel-staging
spec:
  acme:
    email: your-email@example.com
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: letsencrypt-cpanel-staging-key
    solvers:
    - dns01:
        webhook:
          groupName: acme.cpanel.webhook
          solverName: cpanel
          config:
            endpoint: "https://cpanel.yourdomain.com:2083"
            username: "your_cpanel_username"
            apiTokenSecretRef:
              name: cpanel-api-token
              key: token
```

Apply it:

```bash
kubectl apply -f clusterissuer.yaml
```

## Step 6: Request Your First Certificate

Create `certificate.yaml`:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-cert
  namespace: default
spec:
  secretName: test-cert-tls
  issuerRef:
    name: letsencrypt-cpanel-staging
    kind: ClusterIssuer
  dnsNames:
  - test.yourdomain.com
```

Apply and watch:

```bash
kubectl apply -f certificate.yaml
kubectl describe certificate test-cert
```

## Step 7: Verify Success

```bash
# Check certificate status
kubectl get certificate test-cert

# Should show:
# NAME        READY   SECRET           AGE
# test-cert   True    test-cert-tls    2m

# View the secret
kubectl get secret test-cert-tls -o yaml
```

## Step 8: Switch to Production

Once testing works, create a production issuer:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-cpanel
spec:
  acme:
    email: your-email@example.com
    server: https://acme-v02.api.letsencrypt.org/directory  # Production!
    privateKeySecretRef:
      name: letsencrypt-cpanel-key
    solvers:
    - dns01:
        webhook:
          groupName: acme.cpanel.webhook
          solverName: cpanel
          config:
            endpoint: "https://cpanel.yourdomain.com:2083"
            username: "your_cpanel_username"
            apiTokenSecretRef:
              name: cpanel-api-token
              key: token
```

## Troubleshooting

### Check webhook logs

```bash
kubectl logs -n cert-manager deployment/cert-manager-webhook-cpanel -f
```

### Check cert-manager logs

```bash
kubectl logs -n cert-manager deployment/cert-manager -f
```

### Common Issues

**Certificate stuck in "Pending"**
- Check webhook logs for errors
- Verify cPanel API token has DNS permissions
- Ensure endpoint URL is correct (including port)

**"failed to get API token"**
- Secret must be in the same namespace as the Certificate
- Verify secret name and key match your ClusterIssuer

**DNS propagation timeout**
- Check that domain's NS records point to cPanel DNS
- Verify TXT record was created in cPanel DNS Zone Editor

## Next Steps

- Set up automatic certificate renewal (happens automatically!)
- Use certificates in Ingress resources
- Create wildcard certificates with `dnsNames: ["*.yourdomain.com"]`
