# SwarmCD

A declarative GitOps and Continuous Deployment tool for Docker Swarm.

Inspired by [ArgoCD](https://argo-cd.readthedocs.io/en/stable/).

Features:
- [**GitOps:**](#basic-usage) Host your Swarm stacks, configs, and secrets in a Git repo and SwarmCD will automatically deploy new changes
- [**Encrypted Secrets:**](#encrypt-secrets-using-sops) Encrypt your secrets using [SOPS](https://github.com/getsops/sops) and configure SwarmCD to decrypt them before deployment
- [**Automatic Rotation of Configs and Secrets:**](#automatically-rotate-configs-and-secrets) You don't need to rename your configs or secrets every time you change them
- [**Templating:**](#templating) Define stack templates and fill them with different values for different environments, similar to [helm](https://helm.sh/)



## Basic Usage
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


## Encrypt Secrets Using SOPS
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
For example, if you used [age](https://github.com/FiloSottile/age)
to encrypt them, you have to mount the age key file to SwarmCD
and set the environment variable SOPS `SOPS_AGE_KEY_FILE`
to the path of the key file. See the following docker-compose example

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

This way, SwarmCD will decrypt the files each time before it updates
the stack.


## Automatically Rotate Configs and Secrets

In the official docker swarm docs, the recommended method to rotate a config or a secret is by creating a new object with a new name, deleting the old one and making the docker service use the new one, as in [this](https://docs.docker.com/engine/swarm/configs/#example-rotate-a-config) and [this](https://docs.docker.com/engine/swarm/secrets/#example-rotate-a-secret). SwarmCD does this automatically by renaming the config or secret object by appending a short hash to it. This hash is calculated using md5 and its value will be different for different file contents. You can disable this behavior be setting `auto_rotate` property in [config.yaml](docs/config.yaml) file to `false`, which means you would have to manually rename the object in stack defenition every time you change it.




## Templating



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
    image: ghcr.io/m-adawi/swarm-cd:1.1.0
    depends_on:
      - socket_proxy
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

## Documentation

See [docs](https://github.com/m-adawi/swarm-cd/blob/main/docs).
