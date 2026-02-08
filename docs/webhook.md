# Webhook

SwarmCD provides a webhook endpoint that allows you to trigger stack updates on-demand, instead of waiting for the polling interval. This is useful for CI/CD pipelines where you want to deploy immediately after pushing changes.

## Configuration

The webhook requires authentication using a secret key. You can configure the key using one of the following methods (in priority order):

### 1. Environment Variable (Recommended for simple setups)

```bash
WEBHOOK_KEY=your-secret-key
```

### 2. File Path (Recommended for Docker Swarm secrets)

In your `config.yaml`:

```yaml
webhook_key_file: /run/secrets/webhook_key
```

Then mount your Docker secret:

```yaml
services:
  swarm-cd:
    image: swarm-cd
    secrets:
      - webhook_key

secrets:
  webhook_key:
    external: true
```

### 3. Config File (Not recommended for production)

In your `config.yaml`:

```yaml
webhook_key: your-secret-key
```

## Usage

### Update All Stacks

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Authorization: Bearer your-secret-key"
```

### Update a Specific Stack

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Authorization: Bearer your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"stack": "my-stack-name"}'
```

## Response

### Success

```json
{"message": "all stacks update triggered"}
```

or for a specific stack:

```json
{"message": "stack update triggered", "stack": "my-stack-name"}
```

### Errors

**401 Unauthorized** - Missing or invalid webhook key:

```json
{"error": "missing Authorization header"}
```

```json
{"error": "invalid webhook key"}
```

```json
{"error": "webhook not configured"}
```

**404 Not Found** - Stack not found:

```json
{"error": "stack my-stack-name not found"}
```

## CI/CD Integration Examples

### GitHub Actions

```yaml
- name: Trigger SwarmCD deployment
  run: |
    curl -X POST ${{ secrets.SWARMCD_URL }}/webhook \
      -H "Authorization: Bearer ${{ secrets.SWARMCD_WEBHOOK_KEY }}" \
      -H "Content-Type: application/json" \
      -d '{"stack": "my-app"}'
```

### GitLab CI

```yaml
deploy:
  script:
    - |
      curl -X POST ${SWARMCD_URL}/webhook \
        -H "Authorization: Bearer ${SWARMCD_WEBHOOK_KEY}" \
        -H "Content-Type: application/json" \
        -d '{"stack": "my-app"}'
```

## Security Considerations

- Always use HTTPS in production to protect the webhook key in transit
- Use Docker secrets or environment variables instead of storing the key in config files
- Use a strong, randomly generated key (e.g., `openssl rand -hex 32`)
- Consider restricting network access to the webhook endpoint