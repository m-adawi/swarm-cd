# SwarmCD

A declarative GitOps and Continuous Deployment tool for Docker Swarm.

Inspired by [ArgoCD](https://argo-cd.readthedocs.io/en/stable/).

![SwarmCD UI](assets/ui.png)

## Usage

In this example, we use SwarmCD to deploy the stack in the repo
[swarm-cd-example](https://github.com/m-adawi/swarm-cd-example) to a docker swarm cluster.

First we add the repo to the file `repos.yaml`

```yaml
# repos.yaml
swarm-cd-example:
  url: "https://github.com/m-adawi/swarm-cd-example.git"
```

Then we define the stack in `stacks.yaml`

```yaml
# stacks.yaml
nginx:
  repo: swarm-cd-example
  branch: main
  compose_file: nginx/compose.yaml
```

And finally, we deploy SwarmCD to the cluster
using the following docker-compose file:

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

Run this on a swarm manager node:

```bash
docker stack deploy --compose-file docker-compose.yaml swarm-cd
```

This will start SwarmCD, it will periodically check the stack repo
for new changes, pulling them and updating the stack.

## Manage Encrypted Secrets Using SOPS

You can use [sops](https://github.com/getsops/sops) to encrypt secrets in git repos and
have SwarmCD decrypt them before deploying or updating your stacks.

The stack `nginx-ssl` in the
[example repo](https://github.com/m-adawi/swarm-cd-example)
has two secret files under `nginx-ssl/secrets/` directory.
You can configure SwarmCD files to decrypt them by
setting the property`sops_files` in a stack defenition.

```yaml
# stacks.yaml
nginx-ssl:
    repo: swarm-cd-example
    branch: main
    compose_file: nginx-ssl/compose.yaml
    sops_files: 
      - nginx-ssl/secrets/www.example.com.crt
      - nginx-ssl/secrets/www.example.com.key
```

Then you need to set the SOPS environment variables that are required
to decrypt the files.
Depending on the backend you used for sops encryption, the configuration
can be a little different:
- If you used [age](https://github.com/FiloSottile/age)
to encrypt, you have to mount the age key file to SwarmCD
and set the environment variable SOPS `SOPS_AGE_KEY_FILE`
to the path of the key file.
- If you used gpg, you have to mount the file containing your gpg private
key in the container, and set the environment variable
`SOPS_GPG_PRIVATE_KEY_FILE` to the path of the gpg private key file.
It is also possible to directly provide the gpg key in the `SOPS_GPG_PRIVATE_KEY`
environment variable.

See the following docker-compose example.

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
        target: /secrets/age.key # or /secrets/private.gpg
    environment:
      - SOPS_AGE_KEY_FILE=/secrets/age.key
      # or
      - SOPS_GPG_PRIVATE_KEY_FILE=/secrets/private.gpg
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./repos.yaml:/app/repos.yaml:ro
      - ./stacks.yaml:/app/stacks.yaml:ro
secrets:
  age:
    file: age.key
```

This way, SwarmCD will decrypt the files each time before it updates
the stack.

### Automatic SOPS secrets detection

Instead of specifying the paths of every single secrets you need to decrypt,
you can use the `sops_secrets_discovery: true` option:

- in the `config.yaml` file to enable it globally
- in the `stacks.yaml` file for the individual stacks.

Please note that:

- if the global setting is set to `true`, it ignores individual stacks overrides.
- if the stack-level setting is set to `true`, it ignores the `sops_files` setting altogether.

## External Value Resolvers

Swarm-CD supports resolving environment variable values from external systems before deploying Docker stacks. This feature allows you to dynamically inject configuration values from HashiCorp Vault or Consul KV into your compose files without hardcoding sensitive or environment-specific data.

The resolver system works by recognizing special reference formats in environment variable values. When Swarm-CD processes a stack, it automatically resolves these references and substitutes them with actual values from the configured external systems.

### Supported Resolvers

- **HashiCorp Vault** (KV v2) - for resolving values from Vault secrets
- **HashiCorp Consul** (KV store) - for resolving values from Consul KV

### How It Works

When Swarm-CD processes a stack:

1. It reads the compose file from git.
2. Scans environment variables for resolver references (e.g., `vault:...` or `consul:...`).
3. Calls the appropriate external system to fetch the value.
4. Substitutes the resolved value into the environment variable.
5. Deploys the stack with resolved values.

Environment variables may be declared either as a map:

```yaml
environment:
  INTEGRATION_ID: "vault:kv/data/dev#INTEGRATION_ID"
```

or as a list:

```yaml
environment:
  - INTEGRATION_ID=vault:kv/data/dev#INTEGRATION_ID
```

Both forms are supported.

> Note: Environment variables in Docker are always strings. Even if a value in the external system is stored as a number or boolean, it will be passed to the container as a string, and the application is expected to parse it to the desired type if needed.

---

### Vault Integration

Swarm-CD can resolve values from HashiCorp Vault (KV v2) and substitute them into environment variables.

#### Configuration

Vault client is configured in `config.yaml`:

```yaml
vault_address: "https://vault.example.com:8200"
vault_token: ""                    # optional, can be provided via VAULT_TOKEN env var
vault_namespace: ""                 # optional, for Vault Enterprise namespaces
vault_token_renew_interval: 1       # token renewal interval in days (default: 1)
```

- **`vault_address`**: URL of your Vault server.
- **`vault_token`**: Read token for Swarm-CD. It is recommended to provide it via the `VAULT_TOKEN` environment variable instead of committing it into git.
- **`vault_namespace`**: Optional Vault namespace. Leave empty if you do not use namespaces. If namespaces are enabled and your API paths look like `/v1/ns1/kv/data/dev`, set `vault_namespace: "ns1"` and keep `kv/data/dev` in the references (do not include the namespace in the path itself).
- **`vault_token_renew_interval`**: Token renewal interval in days. Swarm-CD will automatically renew the Vault token before it expires. Default is 1 day. The token will be renewed periodically in the background, and if a token expiration error is detected during a read operation, Swarm-CD will attempt to renew the token and retry the operation.

Token resolution order:

1. `vault_token` from `config.yaml` (if non-empty)
2. `VAULT_TOKEN` environment variable

#### Token Renewal

Swarm-CD automatically handles Vault token renewal to prevent expiration issues:

- Tokens are renewed periodically in the background based on the configured `vault_token_renew_interval`
- If a token expiration error is detected during a read operation, Swarm-CD will automatically attempt to renew the token and retry the operation
- This ensures continuous operation even with tokens that have limited TTL

#### Referencing Vault Values

Swarm-CD recognizes values starting with `vault:` prefix:

```text
vault:<path>#<key>
```

Where:

- `<path>` is the logical Vault path that is used with `GET /v1/<path>`,
- `<key>` is the field name inside the secret data.

For Vault KV v2, data is usually nested under the `data` field, and Swarm-CD takes this into account automatically.

**Example:**

In Vault UI you see a value available via API path:

```text
/v1/kv/data/dev
```

and inside it there is a key:

```text
INTEGRATION_ID = "91ba31a7-ec20-467e-b324-b21ce7e90429"
```

In your compose file you can write:

```yaml
services:
  app:
    environment:
      INTEGRATION_ID: "vault:kv/data/dev#INTEGRATION_ID"
```

---

### Consul Integration

Swarm-CD can resolve values from HashiCorp Consul KV store and substitute them into environment variables.

#### Configuration

Consul client is configured in `config.yaml`:

```yaml
consul_address: "https://consul.example.com:8500"
consul_token: ""          # optional, can be provided via CONSUL_TOKEN env var
```

- **`consul_address`**: URL of your Consul server.
- **`consul_token`**: Optional Consul ACL token. It is recommended to provide it via the `CONSUL_TOKEN` environment variable instead of committing it into git.

Token resolution order:

1. `consul_token` from `config.yaml` (if non-empty)
2. `CONSUL_TOKEN` environment variable

#### Referencing Consul Values

Swarm-CD recognizes values starting with `consul:` prefix. You can reference either the entire value or a specific field from a JSON value:

**Getting the Entire Value**

To get the complete value stored at a Consul key:

```text
consul:<key>
```

**Example:**

If Consul KV contains:
```
Key: variables/dev/INTEGRATION_ID
Value: "91ba31a7-ec20-467e-b324-b21ce7e90429"
```

In your compose file:

```yaml
services:
  app:
    environment:
      INTEGRATION_ID: "consul:variables/dev/INTEGRATION_ID"
```

**Getting a Field from JSON**

To extract a specific field from a JSON value stored in Consul:

```text
consul:<key>#<field>
```

Where:

- `<key>` is the Consul KV key path,
- `<field>` is the JSON field name to extract.

**Example:**

If Consul KV contains:
```
Key: variables/dev/integration
Value: {"INTEGRATION_ID": "91ba31a7-ec20-467e-b324-b21ce7e90429", "INTEGRATION_URI": "https://example.integration.com/v1"}
```

In your compose file you can extract individual fields:

```yaml
services:
  app:
    environment:
      INTEGRATION_ID: "consul:variables/dev/integration#INTEGRATION_ID"
      INTEGRATION_URI: "consul:variables/dev/integration#INTEGRATION_URI"
```

When Swarm-CD processes this:

1. It reads the compose file from git.
2. Finds values starting with `consul:`.
3. Calls Consul KV API to fetch the value at the specified key.
4. If `#field` is specified, parses the value as JSON and extracts the field.
5. Substitutes the resolved value into the environment variable before deploying the stack.

> Note: If you specify a field with `#field` but the value at the Consul key is not valid JSON, Swarm-CD will return an error. For non-JSON values, use the format without `#field` to get the raw string value.

## Connect SwarmCD to a remote docker socket

You can use the `DOCKER_HOST` environment variable to point SwarmCD to a remote docker socket,
be it in the same swarm or a different host.

In the following example `docker-socket-proxy` talks directly to the host socket proxy,
and SwarmCD connects to it:

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

## Give SwarmCD access to private registries

You can pass the authentication to private container registries via the `~/.docker/config.json` file.

First, encode your credentials with base64 (here we use `printf` to avoid the trailing newline):

```shell
printf 'username:password' | base64
```

Then create the docker config file like this:

```json
// docker-config.json
{
    "auths": {
        "my.registry.example": {
            "auth": "(base64 output here)"
        }
    }
}
```

Lastly, add the config file as secret and mount it to `/root/.docker/config.json`:

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
    secrets:
      - source: docker-config
        target: /root/.docker/config.json
secrets:
  docker-config:
    file: docker-config.json
```
Note: if running swarmcd as a user other than root, modify the docker config mount path to match.

## Documentation

See [docs](https://github.com/m-adawi/swarm-cd/blob/main/docs).
