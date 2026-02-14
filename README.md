# SwarmCD

A declarative GitOps and Continuous Deployment tool for Docker Swarm.

Inspired by [ArgoCD](https://argo-cd.readthedocs.io/en/stable/).

![SwarmCD UI](assets/ui.png)

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration Reference](#configuration-reference)
  - [config.yaml](#configyaml)
  - [repos.yaml](#reposyaml)
  - [stacks.yaml](#stacksyaml)
- [Features](#features)
  - [SOPS Secrets Management](#sops-secrets-management)
  - [Go Template Support](#go-template-support)
  - [Auto-Rotate Configs and Secrets](#auto-rotate-configs-and-secrets)
  - [Webhook Integration](#webhook-integration)
- [Advanced Configuration](#advanced-configuration)
  - [Remote Docker Socket](#remote-docker-socket)
  - [Private Container Registries](#private-container-registries)
- [Web UI](#web-ui)
- [Quick Reference](#quick-reference)

---

## Overview

SwarmCD bridges the gap between GitOps practices and Docker Swarm deployments. Define your infrastructure as code in Git repositories, and SwarmCD will:

- **Poll** your repositories at configurable intervals
- **Detect** changes in your stack definitions
- **Deploy** updates automatically to your Docker Swarm cluster
- **Decrypt** SOPS-encrypted secrets before deployment
- **Report** status through a built-in Web UI

---

## Quick Start

In this example, we use SwarmCD to deploy the stack in the repo
[swarm-cd-example](https://github.com/m-adawi/swarm-cd-example) to a Docker Swarm cluster.

**1. Add the repo to `repos.yaml`:**

```yaml
# repos.yaml
swarm-cd-example:
  url: "https://github.com/m-adawi/swarm-cd-example.git"
```

**2. Define the stack in `stacks.yaml`:**

```yaml
# stacks.yaml
nginx:
  repo: swarm-cd-example
  branch: main
  compose_file: nginx/compose.yaml
```

**3. Deploy SwarmCD to the cluster:**

```yaml
# docker-compose.yaml
version: '3.7'
services:
  swarm-cd:
    image: ghcr.io/m-adawi/swarm-cd:latest
    deploy:
      placement:
        constraints:
          - node.role == manager
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./repos.yaml:/app/repos.yaml:ro
      - ./stacks.yaml:/app/stacks.yaml:ro
```

**4. Run on a swarm manager node:**

```bash
docker stack deploy --compose-file docker-compose.yaml swarm-cd
```

SwarmCD will now periodically check the stack repo for changes, pull them, and update the stack automatically.

---

## Configuration Reference

SwarmCD uses three main configuration files. You can either use separate files or consolidate everything into `config.yaml`.

### config.yaml

The main configuration file for SwarmCD behavior and global settings.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `update_interval` | integer | `120` | Polling interval in seconds — how often SwarmCD checks repositories for changes |
| `repos_path` | string | `repos/` | Local filesystem path where repositories are cloned |
| `address` | string | `0.0.0.0:8080` | Address and port for the Web UI |
| `auto_rotate` | boolean | `false` | Automatically rotate Docker configs and secrets when they change (adds content hash to names) |
| `sops_secrets_discovery` | boolean | `false` | Globally enable automatic detection of SOPS-encrypted files |
| `repos` | object | — | Inline repository definitions (alternative to `repos.yaml`) |
| `stacks` | object | — | Inline stack definitions (alternative to `stacks.yaml`) |
| `webhook_key` | string | — | Secret key for webhook authentication (not recommended for production) |
| `webhook_key_file` | string | — | Path to file containing webhook key (recommended for Docker secrets) |

**Example:**

```yaml
update_interval: 300
repos_path: /data/repos/
address: 0.0.0.0:8080
auto_rotate: true
sops_secrets_discovery: true
webhook_key_file: /run/secrets/webhook_key
```

> **Note:** The webhook key can also be set via the `WEBHOOK_KEY` environment variable. Priority order: `WEBHOOK_KEY` env var > `webhook_key_file` > `webhook_key`

---

### repos.yaml

Defines the Git repositories containing your stack definitions.

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `url` | string | **Yes** | Git repository URL |
| `username` | string | No | Username for authentication (required if using password) |
| `password` | string | No | Password or personal access token |
| `password_file` | string | No | Path to file containing password (recommended for Docker secrets) |

**Example:**

```yaml
# Public repository — no authentication needed
my-public-repo:
  url: "https://github.com/myorg/public-stacks.git"

# Private repository with inline credentials
my-private-repo:
  url: "https://github.com/myorg/private-stacks.git"
  username: deploy-user
  password: ghp_xxxxxxxxxxxxxxxxxxxx

# Private repository with file-based credentials (recommended)
production-repo:
  url: "https://github.com/myorg/production.git"
  username: deploy-user
  password_file: /run/secrets/github_token
```

> **Security Tip:** Always use `password_file` with Docker secrets in production environments rather than inline passwords.

---

### stacks.yaml

Defines the Docker Swarm stacks that SwarmCD manages.

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `repo` | string | **Yes** | Name of the repository (must match a key in `repos.yaml`) |
| `branch` | string | **Yes** | Git branch to track |
| `compose_file` | string | **Yes** | Path to the Docker Compose file within the repository |
| `values_file` | string | No | Path to values file for Go template rendering |
| `sops_files` | list | No | List of SOPS-encrypted file paths to decrypt before deployment |
| `sops_secrets_discovery` | boolean | No | Enable automatic SOPS file detection for this stack |

**Example:**

```yaml
# Basic stack
nginx:
  repo: my-public-repo
  branch: main
  compose_file: nginx/compose.yaml

# Stack with Go template values
webapp:
  repo: production-repo
  branch: main
  compose_file: webapp/compose.yaml
  values_file: webapp/values-prod.yaml

# Stack with explicit SOPS-encrypted secrets
webapp-ssl:
  repo: production-repo
  branch: main
  compose_file: webapp-ssl/compose.yaml
  sops_files:
    - webapp-ssl/secrets/tls.crt
    - webapp-ssl/secrets/tls.key

# Stack with automatic SOPS discovery
microservice:
  repo: production-repo
  branch: main
  compose_file: microservice/compose.yaml
  sops_secrets_discovery: true
```

---

## Features

### SOPS Secrets Management

SwarmCD integrates with [SOPS](https://github.com/getsops/sops) to decrypt encrypted files before deployment. This allows you to safely store secrets in Git.

**Supported backends:**
- **age** — Set `SOPS_AGE_KEY_FILE` to the path of your age key file
- **GPG** — Set `SOPS_GPG_PRIVATE_KEY_FILE` to the path of your GPG private key, or `SOPS_GPG_PRIVATE_KEY` with the key content directly

**Two approaches for specifying secrets:**

1. **Explicit list** — Use `sops_files` in your stack definition
2. **Automatic discovery** — Enable `sops_secrets_discovery` to automatically detect and decrypt SOPS files

**Example with age encryption:**

```yaml
version: '3.7'
services:
  swarm-cd:
    image: ghcr.io/m-adawi/swarm-cd:latest
    deploy:
      placement:
        constraints:
          - node.role == manager
    secrets:
      - source: age
        target: /secrets/age.key
    environment:
      - SOPS_AGE_KEY_FILE=/secrets/age.key
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./repos.yaml:/app/repos.yaml:ro
      - ./stacks.yaml:/app/stacks.yaml:ro
secrets:
  age:
    file: age.key
```

> **Note:** When `sops_secrets_discovery` is enabled globally in `config.yaml`, it takes precedence over individual stack settings. When enabled at the stack level, it ignores `sops_files`.

---

### Go Template Support

Compose files can be treated as Go templates when a `values_file` is specified. This enables dynamic configuration based on environment-specific values.

---

### Auto-Rotate Configs and Secrets

When `auto_rotate: true` is set, SwarmCD automatically appends a content hash to Docker config and secret names. This ensures services are restarted when configuration changes, as Docker Swarm doesn't natively support config/secret updates.

---

### Webhook Integration

Trigger immediate stack updates instead of waiting for the polling interval. Useful for CI/CD integration.

See [docs/webhook.md](docs/webhook.md) for detailed webhook configuration and usage.

---

## Advanced Configuration

### Remote Docker Socket

You can use the `DOCKER_HOST` environment variable to point SwarmCD to a remote Docker socket, be it in the same swarm or a different host.

**Example with docker-socket-proxy:**

```yaml
version: '3.7'

services:
  socket_proxy:
    image: tecnativa/docker-socket-proxy:0.2.0
    deploy:
      placement:
        constraints: 
          - node.role == manager
    volumes: 
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      TZ: Europe/Rome
      INFO: 1
      SERVICES: 1
      NETWORKS: 1
      SECRETS: 1
      CONFIGS: 1
      POST: 1

  swarm-cd:
    image: ghcr.io/m-adawi/swarm-cd:latest
    environment:
      DOCKER_HOST: tcp://socket_proxy:2375
    configs:
      - source: stacks
        target: /app/stacks.yaml
        mode: 0400
      - source: repos
        target: /app/repos.yaml
        mode: 0400

configs:
  stacks:
    file: ./stacks.yaml
  repos:
    file: ./repos.yaml
```

---

### Private Container Registries

You can pass authentication to private container registries via the `~/.docker/config.json` file.

**1. Encode your credentials with base64:**

```shell
printf 'username:password' | base64
```

**2. Create the docker config file:**

```json
{
    "auths": {
        "my.registry.example": {
            "auth": "(base64 output here)"
        }
    }
}
```

**3. Mount as a Docker secret:**

```yaml
version: '3.7'
services:
  swarm-cd:
    image: ghcr.io/m-adawi/swarm-cd:latest
    deploy:
      placement:
        constraints:
          - node.role == manager
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./repos.yaml:/app/repos.yaml:ro
      - ./stacks.yaml:/app/stacks.yaml:ro
    secrets:
      - source: docker-config
        target: /root/.docker/config.json
secrets:
  docker-config:
    file: docker-config.json
```

> **Note:** If running SwarmCD as a user other than root, modify the docker config mount path to match.

---

## Web UI

SwarmCD includes a built-in web dashboard accessible at the configured `address` (default: `http://localhost:8080`).

**Features:**
- View all managed stacks at a glance
- See current Git revision for each stack
- Monitor stack health and error states
- Search and filter stacks
- Dark mode support

---

## Quick Reference

| File | Purpose |
|------|---------|
| `config.yaml` | Global settings, intervals, feature flags |
| `repos.yaml` | Git repository definitions and credentials |
| `stacks.yaml` | Stack definitions mapping to compose files |

| Environment Variable | Purpose |
|---------------------|---------|
| `DOCKER_HOST` | Connect to a remote Docker socket |
| `WEBHOOK_KEY` | Webhook authentication key |
| `SOPS_AGE_KEY_FILE` | Path to age key for SOPS decryption |
| `SOPS_GPG_PRIVATE_KEY_FILE` | Path to GPG key for SOPS decryption |
| `SOPS_GPG_PRIVATE_KEY` | GPG private key content for SOPS |

---

## Documentation

Additional documentation:
- [docs/webhook.md](docs/webhook.md) — Webhook configuration and CI/CD integration
- [docs/config.yaml](docs/config.yaml) — Annotated config file reference
- [docs/repos.yaml](docs/repos.yaml) — Annotated repos file reference
- [docs/stacks.yaml](docs/stacks.yaml) — Annotated stacks file reference