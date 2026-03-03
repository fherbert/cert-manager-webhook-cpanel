# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial implementation of cert-manager webhook for cPanel DNS
- DNS-01 ACME challenge solver
- cPanel UAPI client for DNS record management
- Support for API token authentication
- Kubernetes manifests for deployment
- Comprehensive test suite
- Documentation and examples
- Taskfile for build automation
- Docker support with multi-stage builds

### Security
- Read-only root filesystem in container
- Non-root user execution
- Minimal distroless base image
- Secret-based API token storage
