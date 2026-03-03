# GitHub Container Registry Setup

This project uses GitHub Container Registry (GHCR) to host Docker images automatically via GitHub Actions.

## Automatic Image Building

Images are automatically built and pushed to GHCR when:

1. **On every push to `main`**: Creates `edge` and `main-<sha>` tagged images
2. **On every tag `v*`**: Creates versioned releases (e.g., `v1.0.0`, `1.0`, `1`, `latest`)

## Making the Package Public

By default, GHCR packages are private. To make the image publicly accessible:

1. Go to your GitHub repository: `https://github.com/fherbert/cert-manager-webhook-cpanel`
2. Click on **Packages** (right sidebar)
3. Click on the `cert-manager-webhook-cpanel` package
4. Click **Package settings** (gear icon)
5. Scroll to **Danger Zone**
6. Click **Change visibility**
7. Select **Public**
8. Confirm by typing the package name

## Using the Images

### Latest (from main branch)
```yaml
image: ghcr.io/fherbert/cert-manager-webhook-cpanel:latest
```

### Specific version (from tags)
```yaml
image: ghcr.io/fherbert/cert-manager-webhook-cpanel:v1.0.0
```

### Edge (latest commit on main)
```yaml
image: ghcr.io/fherbert/cert-manager-webhook-cpanel:edge
```

## No Secrets Required

GitHub Actions automatically has access to push to GHCR using the built-in `GITHUB_TOKEN`. No additional secrets or configuration needed!

## Supported Platforms

All images are built for:
- `linux/amd64` (x86_64)
- `linux/arm64` (ARM64/Apple Silicon)

## Image Layers

The images use:
- Multi-stage builds for minimal size
- Distroless base image for security
- Non-root user execution
- Read-only root filesystem
