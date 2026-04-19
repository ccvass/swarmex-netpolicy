<p align="center"><img src="https://raw.githubusercontent.com/ccvass/swarmex/main/docs/assets/logo.svg" alt="Swarmex" width="400"></p>

[![Test, Build & Deploy](https://github.com/ccvass/swarmex-netpolicy/actions/workflows/publish.yml/badge.svg)](https://github.com/ccvass/swarmex-netpolicy/actions/workflows/publish.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

# Swarmex Netpolicy

Cross-namespace network access control for Docker Swarm.

Part of [Swarmex](https://github.com/ccvass/swarmex) — enterprise-grade orchestration for Docker Swarm.

## What It Does

Controls which services can communicate across namespace boundaries. By default, namespaces are isolated — this controller selectively grants access by attaching services to allowed namespace networks.

## Labels

```yaml
deploy:
  labels:
    swarmex.namespace: "backend"              # Service's own namespace
    swarmex.netpolicy.allow: "frontend,data"  # Namespaces this service can access
```

## How It Works

1. Watches for services with `swarmex.netpolicy.allow` labels.
2. Resolves the target namespace overlay networks.
3. Attaches the service to each allowed namespace network.
4. Removes access when labels are updated or removed.

## Quick Start

```bash
docker service update \
  --label-add swarmex.namespace=backend \
  --label-add swarmex.netpolicy.allow=frontend \
  my-backend
```

## Verified

`svc-be` granted access to the `ns-frontend` network via network attachment.

## License

Apache-2.0
