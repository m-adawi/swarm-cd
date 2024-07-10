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
stacks:
  - name: hello-world
    repo: swarm-cd-example
    branch: main
    compose_file: compose.yaml
```

And finally, we deploy SwarmCD to the cluster 
using the following docker-compose file: 

```yaml
# docker-compose.yaml
version: '3.7'
services:
  swarm-cd:
    image: ghcr.io/m-adawi/swarm-cd:v0.0.1
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

You can find the files used in this example in [example/](example/) directory



# Documentation
See [docs](docs/).

