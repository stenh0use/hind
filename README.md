# Hashistack in Docker (hind)
This repository takes inspiration from the Kubernetes in Docker project. It's intended as a quick and easy playground for Nomad for testing as well as enabling multi-node integration and failure scenarios. It is a WIP and current functionality is very barebones.

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
To scale out the clients
> note: to scale in you need to docker stop/rm the containers or run `make down` to reset the stack
```
NOMAD_CLIENT_COUNT=n make up  # where n is the desired client count
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
docker exec hind.nomad.client.01 cilium status
```
It might take a few minutes for Cilium to come up as healthy. When the last line says `Cluster health:          1/1 reachable` and the remaining helthchecks are passing you should be good to move on (approx 2-5min).

Once you've confirmed cilium agent is healthy you'll need to restart the nomad service.
```
docker exec hind.nomad.client.01 systemctl restart nomad
```
## Deploying Netreap
You can now run the netreap job.
```
nomad run cilium/netreap.hcl
```
Apply a policy
```
consul kv put netreap.io/policy @cilium/policy-allow-all.json
```
## Running jobs with cilium
Run an example job using cilium and test different network policies
```
nomad run jobs/example-cilium.hcl
```
Test curl against google and see that we can connect.
```
nomad exec -i \
    -t $(curl localhost:4646/v1/job/example_cilium/allocations 2>/dev/null \
    | jq -r '.[0].ID') \
    curl google.com -v
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
## Deploying the hubble relay
The [hubble relay](https://docs.cilium.io/en/stable/internals/hubble/#hubble-relay) job is configured to run as a service job, it will let you interact with hubble using the cli.

To deploy the relay
```
nomad run cilium/hubble-relay.hcl
```

Checking the deployment health (assumes the job is deployed to the first node)
```
docker exec hind.nomad.client.01 hubble status
Healthcheck (via localhost:4245): Ok
Current/Max Flows: 485/4,095 (11.84%)
Flows/s: 2.45
Connected Nodes: 1/1
```

# Requirements
This project has mainly been tested using MacOS, colima 0.6.x, and requires cgroupsv2 enabled on the docker host.
- [colima](https://github.com/abiosoft/colima)
- [docker-cli](https://docs.docker.com/engine/install/binaries/#install-client-binaries-on-macos)
- [buildx](https://github.com/abiosoft/colima/discussions/273)
- [nomad](https://developer.hashicorp.com/nomad/downloads) (used for cli commands)
- [consul](https://developer.hashicorp.com/consul/downloads) (used for cli commands)

## Docker info
System and docker information confirmed working
1. <details><summary>Colima, 0.6.7</summary></details>
2. <details><summary>Debian GNU/Linux 12 (bookworm) - click for details</summary>

        Client: Docker Engine - Community
         Version:    24.0.7
         Context:    default
         Debug Mode: false
         Plugins:
          buildx: Docker Buildx (Docker Inc.)
            Version:  v0.11.2
            Path:     /Users/user/.docker/cli-plugins/docker-buildx

        Server:
         Containers: 4
          Running: 4
          Paused: 0
          Stopped: 0
         Images: 11
         Server Version: 25.0.0
         Storage Driver: overlay2
          Backing Filesystem: extfs
          Supports d_type: true
          Using metacopy: false
          Native Overlay Diff: true
          userxattr: false
         Logging Driver: json-file
         Cgroup Driver: systemd
         Cgroup Version: 2
         Plugins:
          Volume: local
          Network: bridge host ipvlan macvlan null overlay
          Log: awslogs fluentd gcplogs gelf journald json-file local splunk syslog
         Swarm: inactive
         Runtimes: io.containerd.runc.v2 runc
         Default Runtime: runc
         Init Binary: docker-init
         containerd version: a1496014c916f9e62104b33d1bb5bd03b0858e59
         runc version: v1.1.11-0-g4bccb38
         init version: de40ad0
         Security Options:
          apparmor
          seccomp
           Profile: builtin
          cgroupns
         Kernel Version: 6.1.0-17-amd64
         Operating System: Debian GNU/Linux 12 (bookworm)
         OSType: linux
         Architecture: x86_64
         CPUs: 2
         Total Memory: 1.921GiB
         Name: debian
         ID: 3eb5fae9-6504-4117-a93d-c099568e79c2
         Docker Root Dir: /var/lib/docker
         Debug Mode: false
         Experimental: false
         Insecure Registries:
          127.0.0.0/8
         Live Restore Enabled: false
</details>


3. <details><summary>Rocky Linux 9.3 (Blue Onyx) - click for details</summary>

        Client: Docker Engine - Community
         Version:    24.0.7
         Context:    default
         Debug Mode: false
         Plugins:
          buildx: Docker Buildx (Docker Inc.)
            Version:  v0.11.2
            Path:     /Users/user/.docker/cli-plugins/docker-buildx

        Server:
         Containers: 0
          Running: 0
          Paused: 0
          Stopped: 0
         Images: 0
         Server Version: 24.0.7
         Storage Driver: overlay2
          Backing Filesystem: xfs
          Supports d_type: true
          Using metacopy: false
          Native Overlay Diff: true
          userxattr: false
         Logging Driver: json-file
         Cgroup Driver: systemd
         Cgroup Version: 2
         Plugins:
          Volume: local
          Network: bridge host ipvlan macvlan null overlay
          Log: awslogs fluentd gcplogs gelf journald json-file local logentries splunk syslog
         Swarm: inactive
         Runtimes: io.containerd.runc.v2 runc
         Default Runtime: runc
         Init Binary: docker-init
         containerd version: a1496014c916f9e62104b33d1bb5bd03b0858e59
         runc version: v1.1.11-0-g4bccb38
         init version: de40ad0
         Security Options:
          seccomp
           Profile: builtin
          selinux
          cgroupns
         Kernel Version: 5.14.0-362.13.1.el9_3.x86_64
         Operating System: Rocky Linux 9.3 (Blue Onyx)
         OSType: linux
         Architecture: x86_64
         CPUs: 2
         Total Memory: 1.722GiB
         Name: default
         ID: 84a6e94a-7b3e-42ff-9370-1d0820695a55
         Docker Root Dir: /var/lib/docker
         Debug Mode: false
         Experimental: false
         Insecure Registries:
          127.0.0.0/8
         Live Restore Enabled: false
</details>

# Current limitations
There is no client persistence when running up and down.

# TODO
- install bpftool?
- improve cluster management tooling
- add ingress load balancer
