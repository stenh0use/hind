# Hashistack in Docker (hind)
This repository takes inspiration from the Kubernetes in Docker project. First iteration is very rough, consider this a WIP.

Implemented from the stack:
- nomad
- consul

Bits and pieces adapted from:

https://github.com/kubernetes-sigs/kind<br>
https://github.com/multani/docker-nomad<br>
https://github.com/hashicorp/docker-consul<br>

# To run Hind
To build the docker containers
```
make build
```
To run the stack
```
make up
```
To tear down the stack
```
make down
```
To connect to the web ui for nomad/consul
```
# nomad
open http://localhost:4646/ui
# consul
open http://localhost:8500/ui
```
To run an example job
```
nomad run jobs/example.hcl
```

# Enabling cilium
The following setup and information is based off of the cosmonic [blog post](https://cosmonic.com/blog/engineering/netreap-a-practical-guide-to-running-cilium-in-nomad). It hasn't been fully tested to ensure everything is working as expected.

To enable cilium, run `make up` with the environment variable set. If you're already up and running you'll need to run `make down` first to recreate the nomad client.
```
CILIUM_ENABLED=1 make up
```
Once the stack is up and running, you can deploy the netreap service
```
# Check cilium health
docker exec hind.nomad.client.1 cilium status
```
It might take a few minutes for Cilium to come up as healthy. When the last line says `Cluster health:          1/1 reachable` and the remaining helthchecks are passing you should be good to move on (approx 2-5min).

Once you've confirmed cilium agent is healthy you'll need to restart the nomad service.
```
docker exec hind.nomad.client.1 systemctl restart nomad
```
You can now run the netreap job.
```
nomad run cilium/netreap.hcl
```
Apply a policy
```
consul kv put netreap.io/policy @cilium/policy-allow-all.json
```
Run a job using cilium
```
nomad run jobs/example-cilium.hcl
```
Exec into the job
```
nomad exec -i \
    -t $(curl localhost:4646/v1/job/example_cilium/allocations 2>/dev/null \
    | jq -r '.[0].ID') \
    bash
# install curl
apt update && apt install curl -y
curl google.com --head
```
Apply deny policy
```
consul kv put netreap.io/policy @cilium/policy-blocked-egress.json
```
Test curl again, and now see that the connection is blocked.
```
nomad exec -i \
    -t $(curl localhost:4646/v1/job/example_cilium/allocations 2>/dev/null \
    | jq -r '.[0].ID') \
    curl google.com -v
```

# Requirements
This project has only been tested with MacOS and colima
- [colima](https://github.com/abiosoft/colima)
- [docker-cli](https://docs.docker.com/engine/install/binaries/#install-client-binaries-on-macos)
- [buildx](https://github.com/abiosoft/colima/discussions/273)
- [nomad](https://developer.hashicorp.com/nomad/downloads) (used for cli commands)
- [consul](https://developer.hashicorp.com/consul/downloads) (used for cli commands)

# Current limitations
There is no client persistence when running up and down.

# TODO
- enable client scaling
- add comments for where code is copied
- add sysctls for nomad client and consul??
- add other driver installs? optional java?
