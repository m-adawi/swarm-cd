# SwarmCD

A declarative GitOps and Continuous Deployment tool for Docker Swarm. 

Inspired by [ArgoCD](https://argo-cd.readthedocs.io/en/stable/). 


# Usage
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


# Manage Encrypted Secrets Using SOPS
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


# Documentation
See [docs](https://github.com/m-adawi/swarm-cd/blob/main/docs).

