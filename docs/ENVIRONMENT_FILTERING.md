# Environment-Based Variables in SwarmCD

This document explains how to use environment-specific variables in SwarmCD.

## Overview

SwarmCD supports **environment-specific variables** that allow you to define different configuration values for each environment (dev, staging, prod, etc.) directly in your stack's Git repository

## How It Works

### 1. Environment Detection

SwarmCD detects the current environment by reading a label from the Docker Swarm manager node. By default, it looks for the label `swarmcd.environment`, but this can be customized in `config.yaml`.

**Setting the environment label on your Swarm manager node:**

```bash
# For a development environment
docker node update --label-add swarmcd.environment=dev <node-name>

# For a staging environment
docker node update --label-add swarmcd.environment=staging <node-name>

# For a production environment
docker node update --label-add swarmcd.environment=prod <node-name>
```

**Check current labels:**
```bash
docker node inspect <node-name> --format '{{ .Spec.Labels }}'
```

### 2. Creating the Environments File

In your **stack's Git repository**, create an `environments.yaml` file (or any name you prefer):

```yaml
# environments.yaml in your stack repository
environments:
  dev:
    DB_HOST: dev-database.local
    API_URL: https://dev-api.example.com
    LOG_LEVEL: debug
  staging:
    DB_HOST: staging-database.local
    API_URL: https://staging-api.example.com
    LOG_LEVEL: info
  prod:
    DB_HOST: prod-database.local
    API_URL: https://api.example.com
    LOG_LEVEL: warn
```

### 3. Configuring the Stack

In your **SwarmCD `stacks.yaml`** file, reference the environments file:

```yaml
my-app:
  repo: my-repo
  branch: main
  compose_file: docker-compose.yaml
  # Reference the environments file in the repository
  environments_file: environments.yaml
```

### 4. Using Variables in Docker Compose Files

In your `docker-compose.yaml` files, you can reference these variables using `${VARIABLE_NAME}` or `$VARIABLE_NAME` syntax:

```yaml
services:
  app:
    image: myapp:latest
    environment:
      DATABASE_HOST: ${DB_HOST}
      API_ENDPOINT: ${API_URL}
      LOG_LEVEL: ${LOG_LEVEL}
    deploy:
      replicas: 3
```

When SwarmCD processes this compose file:
- In the **dev** environment, it will replace `${DB_HOST}` with `dev-database.local`
- In the **staging** environment, it will replace `${DB_HOST}` with `staging-database.local`

## Configuration Reference

### config.yaml

```yaml
# Optional: customize the label name used to detect the environment
environment_label: swarmcd.environment  # default value
```

### stacks.yaml

```yaml
stack-name:
  repo: repo-name
  branch: main
  compose_file: path/to/docker-compose.yaml

  # Optional: path to environments file in the repository
  # This file contains environment-specific variables
  # If omitted, no environment variables will be replaced
  environments_file: path/to/environments.yaml
```

### environments.yaml (in your stack repository)

```yaml
environments:
  dev:
    VAR_NAME: dev-value
    ANOTHER_VAR: dev-value-2
  staging:
    VAR_NAME: staging-value
    ANOTHER_VAR: staging-value-2
  prod:
    VAR_NAME: prod-value
    ANOTHER_VAR: prod-value-2
```

## Deployment Behavior

### Stack Filtering Logic

The deployment decision is based on two factors: whether the stack has an `environments_file` configured and whether the node has an environment label.

| Node Label | Stack has `environments_file` | Behavior |
|------------|------------------------------|----------|
| ❌ No label | ❌ No file | ✅ **DEPLOY** (no filtering) |
| ❌ No label | ✅ Has file | ❌ **SKIP** (stack is environment-filtered) |
| ✅ Has label | ❌ No file | ✅ **DEPLOY** (no filtering) |
| ✅ Has label | ✅ Has file | ✅/❌ **DEPLOY if environment exists in file** |

**In summary:**
- **Stacks WITHOUT `environments_file`**: Always deploy (no environment filtering)
- **Stacks WITH `environments_file` but node WITHOUT label**: Never deploy (filtered stack requires environment)
- **Stacks WITH `environments_file` and node WITH label**: Deploy only if the current environment is defined in the file

### Variable Loading

- **If `environments_file` is not specified**: Stack always deploys, no variables loaded
- **If `environments_file` is specified but no environment label on node**: Stack is **skipped** (not deployed)
- **If `environments_file` is specified but the file doesn't exist in repo**: Stack deploys without environment variables (graceful handling)
- **If the current environment is not defined in the environments file**: Stack is **skipped** (not deployed in this environment)

**Example logs:**

```
# Stack without environments_file - always deploys
INFO detected environment from node label environment=prod label=swarmcd.environment
DEBUG loading environment variables... stack=my-app
DEBUG reading stack file... stack=my-app

# Stack with environments_file and matching environment - deploys
INFO detected environment from node label environment=prod label=swarmcd.environment
DEBUG loading environment variables... stack=my-app
DEBUG loaded environment variables stack=my-app environment=prod vars_count=8
DEBUG reading stack file... stack=my-app

# Stack with environments_file but no environment label - skipped
WARN environment label not found on manager node, all stacks will be deployed label=swarmcd.environment
INFO skipping stack with environment filtering when no environment is set on node stack=my-app environments_file=environments.yaml

# Stack with environments_file but current environment not in file - skipped
INFO detected environment from node label environment=prod label=swarmcd.environment
INFO skipping stack not configured for current environment stack=dev-tools environment=prod available_environments=[dev staging]
```

### Variable Replacement

- Variables are replaced **after** template rendering (if using `values_file`)
- Variables are replaced **before** SOPS decryption and config/secret rotation
- Variables in the format `${VAR_NAME}` or `$VAR_NAME` will be replaced
- Only variables defined in `env_vars[current_environment]` will be replaced
- If no variables are defined for the current environment, no replacement occurs
- Undefined variables remain unchanged in the compose file

## Examples

### Example 1: Basic Stack with Environment Variables

**Repository structure:**
```
my-stack/
├── docker-compose.yaml
└── environments.yaml
```

**stacks.yaml:**
```yaml
my-app:
  repo: my-stack-repo
  branch: main
  compose_file: docker-compose.yaml
  environments_file: environments.yaml
```

**environments.yaml (in repository):**
```yaml
environments:
  dev:
    DB_HOST: dev-db.local
    API_URL: https://dev-api.example.com
  prod:
    DB_HOST: prod-db.local
    API_URL: https://api.example.com
```

**docker-compose.yaml (in repository):**
```yaml
services:
  app:
    image: myapp:latest
    environment:
      DATABASE_HOST: ${DB_HOST}
      API_ENDPOINT: ${API_URL}
```

### Example 2: Different Database Configurations per Environment

**environments.yaml (in repository):**
```yaml
environments:
  dev:
    DB_HOST: postgres-dev.internal
    DB_NAME: api_dev
    DB_USER: dev_user
    REDIS_HOST: redis-dev.internal
    REDIS_PORT: "6379"
  staging:
    DB_HOST: postgres-staging.internal
    DB_NAME: api_staging
    DB_USER: staging_user
    REDIS_HOST: redis-staging.internal
    REDIS_PORT: "6379"
  prod:
    DB_HOST: postgres-prod.internal
    DB_NAME: api_production
    DB_USER: prod_user
    REDIS_HOST: redis-prod.internal
    REDIS_PORT: "6379"
```

**docker-compose.yaml (in repository):**
```yaml
services:
  api:
    image: mycompany/api:latest
    environment:
      DATABASE_URL: postgresql://${DB_USER}:password@${DB_HOST}:5432/${DB_NAME}
      REDIS_URL: redis://${REDIS_HOST}:${REDIS_PORT}
```

### Example 3: Environment-Specific Scaling and Resources

**environments.yaml (in repository):**
```yaml
environments:
  dev:
    REPLICAS: "1"
    MEMORY_LIMIT: "512M"
    CPU_LIMIT: "0.5"
  staging:
    REPLICAS: "2"
    MEMORY_LIMIT: "1G"
    CPU_LIMIT: "1.0"
  prod:
    REPLICAS: "5"
    MEMORY_LIMIT: "2G"
    CPU_LIMIT: "2.0"
```

**docker-compose.yaml (in repository):**
```yaml
services:
  web:
    image: mycompany/web:latest
    deploy:
      replicas: ${REPLICAS}
      resources:
        limits:
          memory: ${MEMORY_LIMIT}
          cpus: ${CPU_LIMIT}
```

## Troubleshooting

### Stack not deploying

**Check the environment label:**
```bash
docker node inspect self --format '{{ .Spec.Labels.swarmcd\.environment }}'
```

**Check SwarmCD logs:**
```bash
docker service logs swarm-cd
```

Look for messages like:
- `detected environment from node label`
- `skipping stack not configured for current environment`

### Variables not being replaced

1. Ensure the variable is defined in `env_vars` for the current environment
2. Check that the variable syntax in your compose file is correct (`${VAR_NAME}`)
3. Review SwarmCD logs for any errors during variable replacement

### Environment label not found

If you see:
```
WARN environment label not found on manager node, all stacks will be deployed
```

This means the manager node doesn't have the environment label. Add it using:
```bash
docker node update --label-add swarmcd.environment=<your-env> $(docker node ls --filter role=manager -q)
```

### Environments file not found

If the environments file doesn't exist in the repository, you'll see:
```
DEBUG environments file not found, skipping environment variables stack=my-app file=environments.yaml
```

This is not an error - the stack will deploy without environment-specific variables.

## Best Practices

1. **Always set the environment label** on manager nodes to ensure correct variable replacement
2. **Keep the environments file in Git** alongside your compose files for version control
3. **Use descriptive environment names**: `dev`, `staging`, `prod` are common, but use what makes sense for your workflow
4. **Test environment-specific configs** by deploying to dev/staging before production
5. **Keep sensitive values in SOPS-encrypted files**, not in plaintext environment variables
6. **Document your environment-specific variables** in your repository's README
7. **Use the same label name** across all your Swarm clusters for consistency
8. **Commit the environments.yaml file** to your stack repository so changes are tracked with the code

## Migration Guide

### Migrating Existing Configurations

If you have existing stacks and want to add environment-specific variables:

1. **Add environment labels to your Swarm nodes:**
   ```bash
   docker node update --label-add swarmcd.environment=prod <node-name>
   ```

2. **Create an `environments.yaml` file in your stack repository:**
   ```yaml
   environments:
     dev:
       API_URL: https://dev-api.example.com
       DB_HOST: dev-db.local
     prod:
       API_URL: https://api.example.com
       DB_HOST: prod-db.local
   ```

3. **Update your stacks.yaml** to reference the environments file:
   ```yaml
   existing-stack:
     # ... existing configuration ...
     environments_file: environments.yaml  # Add this
   ```

4. **Update your compose files** to use the variables:
   ```yaml
   services:
     app:
       environment:
         API_ENDPOINT: ${API_URL}  # Replace hardcoded values
         DATABASE_HOST: ${DB_HOST}
   ```

5. **Commit the environments.yaml file** to your repository

6. **Test in a non-production environment first** before rolling out to production

### Backward Compatibility

- If you don't add the `environments_file` field, stacks will deploy without environment variables (existing behavior)
- If the environment label is not found, stacks will deploy but without environment-specific variables
- If the environments file doesn't exist, stacks will deploy without errors
- The new fields are optional and don't break existing configurations
